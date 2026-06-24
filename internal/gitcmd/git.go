package gitcmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrNotInstalled = errors.New("git is not installed")
	ErrNotRepo      = errors.New("not inside a git repository")
)

type Runner struct{}

func New() Runner { return Runner{} }

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

func (Runner) EnsureInstalled() error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNotInstalled
	}
	return nil
}

func (r Runner) EnsureRepo(ctx context.Context) error {
	if _, err := r.run(ctx, "", "rev-parse", "--is-inside-work-tree"); err != nil {
		return ErrNotRepo
	}
	return nil
}

func (r Runner) ShortHash(ctx context.Context, commitish string) (string, error) {
	out, err := r.run(ctx, "", "rev-parse", "--short", commitish)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) FullHash(ctx context.Context, commitish string) (string, error) {
	out, err := r.run(ctx, "", "rev-parse", commitish)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) Subject(ctx context.Context, commit string) (string, error) {
	out, err := r.run(ctx, "", "log", "-1", "--pretty=format:%s", commit)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) ReadNote(ctx context.Context, commit string) (string, bool) {
	out, err := r.run(ctx, "", "notes", "show", commit)
	if err != nil {
		return "", false
	}
	return out, true
}

func (r Runner) WriteNote(ctx context.Context, commit, content string) error {
	_, err := r.run(ctx, content, "notes", "add", "-f", "-F", "-", commit)
	return err
}

func (r Runner) RemoveNote(ctx context.Context, commit string) error {
	_, err := r.run(ctx, "", "notes", "remove", commit)
	return err
}

func (r Runner) ShowFile(ctx context.Context, commit, file string) (string, bool) {
	out, err := r.run(ctx, "", "show", fmt.Sprintf("%s:%s", commit, file))
	if err != nil {
		return "", false
	}
	return out, true
}

func (r Runner) DiffUnified0(ctx context.Context, base, file string) (string, error) {
	return r.run(ctx, "", "diff", "--unified=0", base+"...HEAD", "--", file)
}

func (r Runner) MergeBase(ctx context.Context, a, b string) (string, error) {
	out, err := r.run(ctx, "", "merge-base", a, b)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r Runner) CommitExists(ctx context.Context, ref string) bool {
	_, err := r.run(ctx, "", "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

func (r Runner) RemoteURL(ctx context.Context, name string) (string, error) {
	out, err := r.run(ctx, "", "remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
