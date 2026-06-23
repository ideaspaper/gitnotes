package review

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"gitnotes/internal/gitcmd"
)

// Platform is a code-review host.
type Platform string

const (
	GitHub Platform = "github"
	GitLab Platform = "gitlab"
)

// Options controls a Submit run.
type Options struct {
	Number   string   // PR/MR number — required
	Platform Platform // empty = detect from the origin remote
	DryRun   bool
}

// Submitter posts a commit's notes to a PR/MR via the gh / glab CLIs. It owns a
// Builder so it can derive the diff base from the PR/MR itself.
type Submitter struct {
	git     gitcmd.Runner
	builder *Builder
}

// NewSubmitter returns a Submitter.
func NewSubmitter(g gitcmd.Runner, b *Builder) *Submitter {
	return &Submitter{git: g, builder: b}
}

// Submit computes HEAD's notes and posts them to the PR/MR named by opts.Number.
// The diff base is taken from the PR/MR (GitHub's base branch, GitLab's
// diff_refs.base_sha), so in-diff classification matches the actual review.
func (s *Submitter) Submit(ctx context.Context, opts Options) error {
	platform, err := s.platform(ctx, opts)
	if err != nil {
		return err
	}
	switch platform {
	case GitHub:
		return s.submitGitHub(ctx, opts)
	case GitLab:
		return s.submitGitLab(ctx, opts)
	default:
		return fmt.Errorf("unknown platform %q", platform)
	}
}

// SubmitPayload posts a pre-exported payload as-is (its in_diff flags and base
// SHAs are trusted), without recomputing against the PR/MR base.
func (s *Submitter) SubmitPayload(ctx context.Context, p Payload, opts Options) error {
	platform, err := s.platform(ctx, opts)
	if err != nil {
		return err
	}
	if len(p.Comments) == 0 {
		fmt.Println("No notes to submit.")
		return nil
	}
	switch platform {
	case GitHub:
		return s.postGitHub(ctx, p, opts.Number, p.Commit, opts.DryRun)
	case GitLab:
		refs := diffRefs{BaseSHA: p.BaseSHA, StartSHA: p.BaseSHA, HeadSHA: p.Commit}
		return s.postGitLab(ctx, p, opts.Number, refs, opts.DryRun)
	default:
		return fmt.Errorf("unknown platform %q", platform)
	}
}

// platform resolves the target platform from opts or the origin remote.
func (s *Submitter) platform(ctx context.Context, opts Options) (Platform, error) {
	if opts.Platform != "" {
		return opts.Platform, nil
	}
	return s.detectPlatform(ctx)
}

func (s *Submitter) submitGitHub(ctx context.Context, opts Options) error {
	view, viewErr := ghPRView(ctx, opts.Number)
	if viewErr != nil && !opts.DryRun {
		return viewErr
	}

	number, baseBranch, commitID := opts.Number, "", ""
	if viewErr == nil {
		number, baseBranch, commitID = view.Number.String(), view.BaseRefName, view.HeadRefOID
	}

	base := s.resolveBaseRef(ctx, baseBranch)
	p, err := s.builder.Build(ctx, base)
	if err != nil {
		return err
	}
	if commitID == "" {
		commitID = p.Commit
	}
	return s.postGitHub(ctx, p, number, commitID, opts.DryRun)
}

func (s *Submitter) submitGitLab(ctx context.Context, opts Options) error {
	refs, refErr := glabDiffRefs(ctx, opts.Number)
	if refErr != nil && !opts.DryRun {
		return refErr
	}

	base := "HEAD^"
	if refErr == nil {
		base = refs.BaseSHA
	}
	p, err := s.builder.Build(ctx, base)
	if err != nil {
		return err
	}
	if refErr != nil { // dry-run fallback when glab is unavailable
		refs = diffRefs{BaseSHA: p.BaseSHA, StartSHA: p.BaseSHA, HeadSHA: p.Commit}
	}
	return s.postGitLab(ctx, p, opts.Number, refs, opts.DryRun)
}

// resolveBaseRef turns a PR base branch name into a local diff base, preferring
// the remote-tracking ref. Falls back to HEAD^ when nothing resolves.
func (s *Submitter) resolveBaseRef(ctx context.Context, branch string) string {
	if branch != "" {
		for _, ref := range []string{"origin/" + branch, branch} {
			if s.git.CommitExists(ctx, ref) {
				return ref
			}
		}
	}
	return "HEAD^"
}

// detectPlatform sniffs the origin remote host.
func (s *Submitter) detectPlatform(ctx context.Context) (Platform, error) {
	url, err := s.git.RemoteURL(ctx, "origin")
	if err != nil {
		return "", errors.New("could not read `origin`; pass --github or --gitlab")
	}
	host := strings.ToLower(url)
	switch {
	case strings.Contains(host, "github"):
		return GitHub, nil
	case strings.Contains(host, "gitlab"):
		return GitLab, nil
	default:
		return "", errors.New("could not detect the platform from `origin`; pass --github or --gitlab")
	}
}

// --- posting ------------------------------------------------------------------

func (s *Submitter) postGitHub(ctx context.Context, p Payload, number, commitID string, dryRun bool) error {
	if len(p.Comments) == 0 {
		fmt.Println("No notes to submit.")
		return nil
	}
	dryTag := ""
	if dryRun {
		dryTag = "  (dry-run)"
	}
	fmt.Printf("GitHub PR #%s%s\n", number, dryTag)

	for _, c := range p.Comments {
		if c.Type == "line" && c.InDiff {
			body := map[string]any{
				"body":      c.Body,
				"commit_id": commitID,
				"path":      c.Path,
				"line":      c.Line,
				"side":      "RIGHT",
			}
			if c.StartLine > 0 {
				body["start_line"] = c.StartLine
				body["start_side"] = "RIGHT"
			}
			apiPost(ctx, "gh", fmt.Sprintf("repos/{owner}/{repo}/pulls/%s/comments", number), body, "line "+lineLabel(c), dryRun)
			continue
		}
		apiPost(ctx, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%s/comments", number),
			map[string]any{"body": generalBody(c)}, generalLabel(c), dryRun)
	}
	return nil
}

func (s *Submitter) postGitLab(ctx context.Context, p Payload, iid string, refs diffRefs, dryRun bool) error {
	if len(p.Comments) == 0 {
		fmt.Println("No notes to submit.")
		return nil
	}
	dryTag := ""
	if dryRun {
		dryTag = "  (dry-run)"
	}
	fmt.Printf("GitLab MR !%s%s\n", iid, dryTag)

	for _, c := range p.Comments {
		if c.Type == "line" && c.InDiff {
			position := map[string]any{
				"position_type": "text",
				"base_sha":      refs.BaseSHA,
				"start_sha":     refs.StartSHA,
				"head_sha":      refs.HeadSHA,
				"new_path":      c.Path,
				"old_path":      c.Path,
				"new_line":      c.Line,
			}
			if c.StartLine > 0 && c.StartLine < c.Line { // multi-line range
				position["line_range"] = map[string]any{
					"start": map[string]any{"line_code": gitlabLineCode(c.Path, c.StartOldLine, c.StartLine), "type": "new"},
					"end":   map[string]any{"line_code": gitlabLineCode(c.Path, c.EndOldLine, c.Line), "type": "new"},
				}
			}
			apiPost(ctx, "glab", fmt.Sprintf("projects/:id/merge_requests/%s/discussions", iid),
				map[string]any{"body": c.Body, "position": position}, "line "+lineLabel(c), dryRun)
			continue
		}
		apiPost(ctx, "glab", fmt.Sprintf("projects/:id/merge_requests/%s/notes", iid),
			map[string]any{"body": generalBody(c)}, generalLabel(c), dryRun)
	}
	return nil
}

// gitlabLineCode builds a GitLab diff line code: the SHA-1 of the file path,
// then the old-side and new-side line numbers, joined by underscores.
func gitlabLineCode(path string, oldLine, newLine int) string {
	sum := sha1.Sum([]byte(path))
	return fmt.Sprintf("%s_%d_%d", hex.EncodeToString(sum[:]), oldLine, newLine)
}

// lineLabel renders a line comment's location ("path:line" or "path:start-end").
func lineLabel(c Comment) string {
	if c.StartLine > 0 && c.StartLine < c.Line {
		return fmt.Sprintf("%s:%d-%d", c.Path, c.StartLine, c.Line)
	}
	return fmt.Sprintf("%s:%d", c.Path, c.Line)
}

// generalLabel describes a comment for the progress line.
func generalLabel(c Comment) string {
	if c.Type == "line" {
		return fmt.Sprintf("general (%s outside diff)", lineLabel(c))
	}
	return "general"
}

// generalBody is the body posted for a general comment. A line note that fell
// outside the diff is rendered as its location, a blank line, then the note:
//
//	path/to/file.go:10-14
//
//	the note text
func generalBody(c Comment) string {
	if c.Type == "line" {
		return fmt.Sprintf("%s\n\n%s", lineLabel(c), c.Body)
	}
	return c.Body
}

// apiPost POSTs payload to endpoint through the gh/glab `api` command, or prints
// it under dry-run.
func apiPost(ctx context.Context, tool, endpoint string, payload map[string]any, label string, dryRun bool) {
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("  ✗ %s: encoding payload: %v\n", label, err)
		return
	}
	if dryRun {
		fmt.Printf("  [dry-run] %s  →  POST %s\n", label, endpoint)
		fmt.Printf("            %s\n", data)
		return
	}
	// glab does not set a JSON content type for --input bodies (GitLab then
	// answers 415); gh already does, but the explicit header is harmless there.
	if _, err := runTool(ctx, tool, string(data), "api", "--method", "POST", endpoint, "--input", "-", "-H", "Content-Type: application/json"); err != nil {
		fmt.Printf("  ✗ %s: %v\n", label, err)
		return
	}
	fmt.Printf("  ✓ %s\n", label)
}

// --- gh / glab helpers --------------------------------------------------------

type jsonNumber int

func (n jsonNumber) String() string { return fmt.Sprintf("%d", int(n)) }

type ghView struct {
	Number      jsonNumber `json:"number"`
	HeadRefOID  string     `json:"headRefOid"`
	BaseRefName string     `json:"baseRefName"`
}

func ghPRView(ctx context.Context, number string) (ghView, error) {
	out, err := runTool(ctx, "gh", "", "pr", "view", number, "--json", "number,headRefOid,baseRefName")
	if err != nil {
		return ghView{}, fmt.Errorf("resolving GitHub PR #%s (check `gh auth status`): %w", number, err)
	}
	var v ghView
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return ghView{}, fmt.Errorf("parsing `gh pr view` output: %w", err)
	}
	return v, nil
}

type diffRefs struct {
	BaseSHA  string `json:"base_sha"`
	StartSHA string `json:"start_sha"`
	HeadSHA  string `json:"head_sha"`
}

func glabDiffRefs(ctx context.Context, iid string) (diffRefs, error) {
	out, err := runTool(ctx, "glab", "", "api", fmt.Sprintf("projects/:id/merge_requests/%s", iid))
	if err != nil {
		return diffRefs{}, fmt.Errorf("fetching MR !%s diff refs (check `glab auth status`): %w", iid, err)
	}
	var v struct {
		DiffRefs diffRefs `json:"diff_refs"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return diffRefs{}, fmt.Errorf("parsing MR diff refs: %w", err)
	}
	return v.DiffRefs, nil
}

// runTool executes an external CLI (gh/glab), optionally feeding stdin.
func runTool(ctx context.Context, tool, stdin string, args ...string) (string, error) {
	if _, err := exec.LookPath(tool); err != nil {
		return "", fmt.Errorf("%s is not installed", tool)
	}
	cmd := exec.CommandContext(ctx, tool, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return out.String(), nil
}
