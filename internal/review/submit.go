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
	"gitnotes/internal/note"
)

type Platform string

const (
	GitHub Platform = "github"
	GitLab Platform = "gitlab"
)

type Options struct {
	Number   string
	Platform Platform
	DryRun   bool
}

type Submitter struct {
	git     gitcmd.Runner
	mgr     *note.Manager
	builder *Builder
}

func NewSubmitter(g gitcmd.Runner, m *note.Manager, b *Builder) *Submitter {
	return &Submitter{git: g, mgr: m, builder: b}
}

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
	posted := s.postGitHub(ctx, p, number, commitID, opts.DryRun)
	s.flagSubmitted(ctx, p.Commit, posted, opts.DryRun)
	return nil
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
	if refErr != nil {
		refs = diffRefs{BaseSHA: p.BaseSHA, StartSHA: p.BaseSHA, HeadSHA: p.Commit}
	}
	posted := s.postGitLab(ctx, p, opts.Number, refs, opts.DryRun)
	s.flagSubmitted(ctx, p.Commit, posted, opts.DryRun)
	return nil
}

func (s *Submitter) flagSubmitted(ctx context.Context, commit string, posted []int, dryRun bool) {
	if dryRun || len(posted) == 0 {
		return
	}
	entries, err := s.mgr.Read(ctx, commit)
	if err != nil {
		fmt.Printf("  ! posted, but could not read notes to mark them submitted: %v\n", err)
		return
	}
	changed := false
	for _, idx := range posted {
		if idx >= 0 && idx < len(entries) && !entries[idx].Submitted {
			entries[idx].Submitted = true
			changed = true
		}
	}
	if !changed {
		return
	}
	if err := s.mgr.Write(ctx, commit, entries); err != nil {
		fmt.Printf("  ! posted, but could not mark notes submitted: %v\n", err)
	}
}

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

func (s *Submitter) postGitHub(ctx context.Context, p Payload, number, commitID string, dryRun bool) []int {
	if len(p.Comments) == 0 {
		fmt.Println("No notes to submit.")
		return nil
	}
	dryTag := ""
	if dryRun {
		dryTag = "  (dry-run)"
	}
	fmt.Printf("GitHub PR #%s%s\n", number, dryTag)

	var posted []int
	attempted := 0
	for i, c := range p.Comments {
		if c.Submitted {
			fmt.Printf("  • %s (already submitted, skipping)\n", commentLabel(c))
			continue
		}
		attempted++
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
			if apiPost(ctx, "gh", fmt.Sprintf("repos/{owner}/{repo}/pulls/%s/comments", number), body, "line "+lineLabel(c), dryRun) {
				posted = append(posted, i)
			}
			continue
		}
		if apiPost(ctx, "gh", fmt.Sprintf("repos/{owner}/{repo}/issues/%s/comments", number),
			map[string]any{"body": generalBody(c)}, generalLabel(c), dryRun) {
			posted = append(posted, i)
		}
	}
	if attempted == 0 {
		fmt.Println("  All notes were already submitted; nothing to post.")
	}
	return posted
}

func (s *Submitter) postGitLab(ctx context.Context, p Payload, iid string, refs diffRefs, dryRun bool) []int {
	if len(p.Comments) == 0 {
		fmt.Println("No notes to submit.")
		return nil
	}
	dryTag := ""
	if dryRun {
		dryTag = "  (dry-run)"
	}
	fmt.Printf("GitLab MR !%s%s\n", iid, dryTag)

	var posted []int
	attempted := 0
	for i, c := range p.Comments {
		if c.Submitted {
			fmt.Printf("  • %s (already submitted, skipping)\n", commentLabel(c))
			continue
		}
		attempted++
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
			if c.StartLine > 0 && c.StartLine < c.Line {
				position["line_range"] = map[string]any{
					"start": map[string]any{"line_code": gitlabLineCode(c.Path, c.StartOldLine, c.StartLine), "type": "new"},
					"end":   map[string]any{"line_code": gitlabLineCode(c.Path, c.EndOldLine, c.Line), "type": "new"},
				}
			}
			if apiPost(ctx, "glab", fmt.Sprintf("projects/:id/merge_requests/%s/discussions", iid),
				map[string]any{"body": c.Body, "position": position}, "line "+lineLabel(c), dryRun) {
				posted = append(posted, i)
			}
			continue
		}
		if apiPost(ctx, "glab", fmt.Sprintf("projects/:id/merge_requests/%s/notes", iid),
			map[string]any{"body": generalBody(c)}, generalLabel(c), dryRun) {
			posted = append(posted, i)
		}
	}
	if attempted == 0 {
		fmt.Println("  All notes were already submitted; nothing to post.")
	}
	return posted
}

func commentLabel(c Comment) string {
	if c.Type == "line" && c.InDiff {
		return "line " + lineLabel(c)
	}
	return generalLabel(c)
}

func gitlabLineCode(path string, oldLine, newLine int) string {
	sum := sha1.Sum([]byte(path))
	return fmt.Sprintf("%s_%d_%d", hex.EncodeToString(sum[:]), oldLine, newLine)
}

func lineLabel(c Comment) string {
	if c.StartLine > 0 && c.StartLine < c.Line {
		return fmt.Sprintf("%s:%d-%d", c.Path, c.StartLine, c.Line)
	}
	return fmt.Sprintf("%s:%d", c.Path, c.Line)
}

func generalLabel(c Comment) string {
	if c.Type == "line" {
		return fmt.Sprintf("general (%s outside diff)", lineLabel(c))
	}
	return "general"
}

func generalBody(c Comment) string {
	if c.Type == "line" {
		return fmt.Sprintf("%s\n\n%s", lineLabel(c), c.Body)
	}
	return c.Body
}

func apiPost(ctx context.Context, tool, endpoint string, payload map[string]any, label string, dryRun bool) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("  ✗ %s: encoding payload: %v\n", label, err)
		return false
	}
	if dryRun {
		fmt.Printf("  [dry-run] %s  →  POST %s\n", label, endpoint)
		fmt.Printf("            %s\n", data)
		return true
	}

	if _, err := runTool(ctx, tool, string(data), "api", "--method", "POST", endpoint, "--input", "-", "-H", "Content-Type: application/json"); err != nil {
		fmt.Printf("  ✗ %s: %v\n", label, err)
		return false
	}
	fmt.Printf("  ✓ %s\n", label)
	return true
}

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
