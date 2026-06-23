// Package cli wires the gitnotes subcommands to the note and review packages.
// It is the transport layer: it parses flags, calls one operation, and writes
// the result. No business logic lives here.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"gitnotes/internal/gitcmd"
	"gitnotes/internal/note"
)

const appName = "gitnotes"

// Version is the build version, overridden at release time via -ldflags
// "-X gitnotes/internal/cli.Version=...".
var Version = "dev"

// app holds the dependencies shared by every subcommand.
type app struct {
	git gitcmd.Runner
	mgr *note.Manager
	out io.Writer
}

// Run dispatches args[0] to a subcommand. All commands operate on HEAD.
func Run(ctx context.Context, out io.Writer, args []string) error {
	if len(args) == 0 {
		printUsage(out)
		return nil
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		printUsage(out)
		return nil
	case "version", "-v", "--version":
		fmt.Fprintf(out, "%s %s\n", appName, Version)
		return nil
	}

	g := gitcmd.New()
	if err := g.EnsureInstalled(); err != nil {
		return err
	}
	if err := g.EnsureRepo(ctx); err != nil {
		return err
	}
	a := &app{git: g, mgr: note.NewManager(g), out: out}

	switch cmd {
	case "add":
		return a.runAdd(ctx, rest)
	case "list":
		return a.runList(ctx, rest)
	case "edit":
		return a.runEdit(ctx, rest)
	case "remove":
		return a.runRemove(ctx, rest)
	case "export":
		return a.runExport(ctx, rest)
	case "submit":
		return a.runSubmit(ctx, rest)
	default:
		return fmt.Errorf("unknown command %q (run `%s help`)", cmd, appName)
	}
}

// head resolves HEAD to a short hash, the commit every command acts on.
func (a *app) head(ctx context.Context) (string, error) {
	return a.git.ShortHash(ctx, "HEAD")
}

// errUsage signals a flag-parsing failure already reported by the flag set.
var errUsage = errors.New("usage")

func printUsage(out io.Writer) {
	fmt.Fprintf(out, `%s — manage git notes as reviewable, CSV-backed comments on HEAD

Usage:
  %s add -f <file[:line]|file[:start-end]> -n <note>                Add a line / block note
  %s add -g -n <note>                                               Add a general note
  %s list                                                           List HEAD's notes
  %s edit [index] [-n <note>]                                       Edit a note's text
  %s remove [index] | -a                                            Remove one note (or all)
  %s export [base] [-o <file>]                                      Write the review payload as JSON
  %s submit <number> [--github|--gitlab] [-f <file>] [--dry-run]    Post notes to PR/MR <number>

Location specs:
  path/to/file.go:14      a single line
  path/to/file.go:1-17    a block of lines (1 through 17)
  path/to/file.go         the whole file (no code captured)

Notes are stored as CSV (file,startLine,endLine,code,note) in refs/notes/commits.
submit takes the PR/MR number and derives the diff base from it (GitHub's base
branch, GitLab's diff_refs); in-diff notes become line comments, the rest
general comments. It needs the gh (GitHub) or glab (GitLab) CLI.
`, appName, appName, appName, appName, appName, appName, appName, appName)
}
