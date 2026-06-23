// Package gitcmd is the single boundary to the git command line. Every other
// package reaches git only through Runner, so the rest of the app never builds
// argv or parses git's output directly.
package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Sentinel errors callers branch on with errors.Is.
var (
	ErrNotInstalled = errors.New("git is not installed")
	ErrNotRepo      = errors.New("not inside a git repository")
)

// Runner executes git subcommands. The zero value is usable.
type Runner struct{}

// New returns a Runner.
func New() Runner { return Runner{} }

// run executes `git args...`, optionally feeding stdin, and returns trimmed
// stdout. On a non-zero exit it returns an error carrying git's stderr.
func (Runner) run(ctx context.Context, stdin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			msg := strings.TrimSpace(errBuf.String())
			if msg == "" {
				msg = exitErr.String()
			}
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out.String(), nil
}

// EnsureInstalled fails fast when the git binary is not on PATH.
func (Runner) EnsureInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNotInstalled
	}
	return nil
}

// EnsureRepo fails when the current directory is not within a work tree.
func (r Runner) EnsureRepo(ctx context.Context) error {
	if _, err := r.run(ctx, "", "rev-parse", "--is-inside-work-tree"); err != nil {
		return ErrNotRepo
	}
	return nil
}

// ShortHash resolves a commit-ish to its abbreviated hash.
func (r Runner) ShortHash(ctx context.Context, commitish string) (string, error) {
	out, err := r.run(ctx, "", "rev-parse", "--short", commitish)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// FullHash resolves a commit-ish to its full 40-char hash.
func (r Runner) FullHash(ctx context.Context, commitish string) (string, error) {
	out, err := r.run(ctx, "", "rev-parse", commitish)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Subject returns the one-line subject of a commit.
func (r Runner) Subject(ctx context.Context, commit string) (string, error) {
	out, err := r.run(ctx, "", "log", "-1", "--pretty=format:%s", commit)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ReadNote returns the note attached to commit, or ("", false) when there is
// none. A missing note is not an error (git exits non-zero), mirroring the
// behaviour of the nushell gn it replaces.
func (r Runner) ReadNote(ctx context.Context, commit string) (string, bool) {
	out, err := r.run(ctx, "", "notes", "show", commit)
	if err != nil {
		return "", false
	}
	return out, true
}

// WriteNote sets (force-overwrites) the note on commit from content.
func (r Runner) WriteNote(ctx context.Context, commit, content string) error {
	_, err := r.run(ctx, content, "notes", "add", "-f", "-F", "-", commit)
	return err
}

// RemoveNote deletes the note on commit.
func (r Runner) RemoveNote(ctx context.Context, commit string) error {
	_, err := r.run(ctx, "", "notes", "remove", commit)
	return err
}

// ShowFile returns the contents of file as of commit, or ("", false) when the
// file does not exist at that commit.
func (r Runner) ShowFile(ctx context.Context, commit, file string) (string, bool) {
	out, err := r.run(ctx, "", "show", fmt.Sprintf("%s:%s", commit, file))
	if err != nil {
		return "", false
	}
	return out, true
}

// DiffUnified0 returns the `git diff --unified=0 base...HEAD -- file` output
// used to determine which new-side lines a file changed.
func (r Runner) DiffUnified0(ctx context.Context, base, file string) (string, error) {
	return r.run(ctx, "", "diff", "--unified=0", base+"...HEAD", "--", file)
}

// MergeBase returns the best common ancestor of a and b.
func (r Runner) MergeBase(ctx context.Context, a, b string) (string, error) {
	out, err := r.run(ctx, "", "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CommitExists reports whether ref resolves to a commit object.
func (r Runner) CommitExists(ctx context.Context, ref string) bool {
	_, err := r.run(ctx, "", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// RemoteURL returns the configured URL of the named remote.
func (r Runner) RemoteURL(ctx context.Context, name string) (string, error) {
	out, err := r.run(ctx, "", "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
