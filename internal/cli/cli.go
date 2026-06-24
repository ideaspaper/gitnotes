package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"gitnotes/internal/gitcmd"
	"gitnotes/internal/note"
)

const appName = "gitnotes"

var Version = "dev"

type app struct {
	git gitcmd.Runner
	mgr *note.Manager
	out io.Writer
}

func Run(ctx context.Context, out io.Writer, args []string) error {
	a := &app{out: out}
	root := a.newRootCmd()
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs(args)
	return root.ExecuteContext(ctx)
}

func (a *app) newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           appName,
		Short:         "Manage git notes as reviewable, CSV-backed comments on HEAD",
		Long:          longDescription(),
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,

		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			switch cmd.Name() {
			case "help", "version", appName:
				return nil
			}
			return a.ensureRepo(cmd.Context())
		},
	}
	root.SetVersionTemplate(fmt.Sprintf("%s {{.Version}}\n", appName))
	root.AddCommand(
		a.newAddCmd(),
		a.newListCmd(),
		a.newEditCmd(),
		a.newRemoveCmd(),
		a.newExportCmd(),
		a.newSubmitCmd(),
		a.newVersionCmd(),
	)
	return root
}

func (a *app) ensureRepo(ctx context.Context) error {
	g := gitcmd.New()
	if err := g.EnsureInstalled(); err != nil {
		return err
	}
	if err := g.EnsureRepo(ctx); err != nil {
		return err
	}
	a.git = g
	a.mgr = note.NewManager(g)
	return nil
}

func (a *app) head(ctx context.Context) (string, error) {
	return a.git.ShortHash(ctx, "HEAD")
}

func (a *app) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(a.out, "%s %s\n", appName, Version)
			return nil
		},
	}
}

func longDescription() string {
	return `gitnotes manages git notes as reviewable, CSV-backed comments on HEAD and
posts them as GitHub PR / GitLab MR review comments.

Location specs:
  path/to/file.go:14      a single line
  path/to/file.go:1-17    a block of lines (1 through 17)
  path/to/file.go         the whole file (no code captured)

Notes are stored as CSV (file,startLine,endLine,code,note,submitted) in
refs/notes/commits. submit takes the PR/MR number and derives the diff base
from it (GitHub's base branch, GitLab's diff_refs); in-diff notes become line
comments, the rest general comments, and posted entries are flagged submitted
so re-running submit never posts them twice. It needs the gh (GitHub) or glab
(GitLab) CLI.`
}
