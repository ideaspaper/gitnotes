package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"gitnotes/internal/note"
	"gitnotes/internal/review"
)

func (a *app) newAddCmd() *cobra.Command {
	var (
		file    string
		text    string
		general bool
	)
	cmd := &cobra.Command{
		Use:   "add (-f <file[:line]|file[:start-end]> | -g) -n <note>",
		Short: "Add a line / block note, or a general commit-level note",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if general == (file != "") {
				return fmt.Errorf("specify exactly one of -f <location> or -g")
			}
			body := strings.TrimSpace(text)
			if body == "" {
				return fmt.Errorf("note text is required: -n <note>")
			}

			var loc note.Location
			if !general {
				parsed, err := note.ParseSpec(file)
				if err != nil {
					return err
				}
				loc = parsed
			}

			ctx := cmd.Context()
			commit, err := a.head(ctx)
			if err != nil {
				return err
			}
			entries, err := a.mgr.Add(ctx, commit, loc, body)
			if err != nil {
				return err
			}
			fmt.Fprintf(a.out, "Added %s note to %s (%d total).\n", loc.Label(), commit, len(entries))
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "location: file[:line] or file[:start-end]")
	cmd.Flags().StringVarP(&text, "note", "n", "", "note text (required)")
	cmd.Flags().BoolVarP(&general, "general", "g", false, "add a general commit-level note")
	return cmd
}

func (a *app) newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List HEAD's notes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			commit, err := a.head(ctx)
			if err != nil {
				return err
			}
			entries, err := a.mgr.Read(ctx, commit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintf(a.out, "No git notes on %s.\n", commit)
				return nil
			}
			subject, _ := a.git.Subject(ctx, commit)
			fmt.Fprintf(a.out, "%s  %s\n", commit, subject)
			a.renderEntries(entries)
			return nil
		},
	}
}

func (a *app) newEditCmd() *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "edit [index]",
		Short: "Edit a note's text",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			commit, err := a.head(ctx)
			if err != nil {
				return err
			}
			entries, err := a.mgr.Read(ctx, commit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintf(a.out, "No git notes on %s.\n", commit)
				return nil
			}

			idx, err := selectIndex(firstArg(args), len(entries))
			if err != nil {
				return err
			}

			newText := strings.TrimSpace(text)
			if newText == "" {
				fmt.Fprintf(a.out, "Current: %s\n", entries[idx].Note)
				newText, err = prompt("New note (blank = keep): ")
				if err != nil {
					return err
				}
				if newText == "" {
					fmt.Fprintln(a.out, "Unchanged.")
					return nil
				}
			}

			entries[idx].Note = newText
			if err := a.mgr.Write(ctx, commit, entries); err != nil {
				return err
			}
			fmt.Fprintf(a.out, "Updated note #%d on %s.\n", idx, commit)
			return nil
		},
	}
	cmd.Flags().StringVarP(&text, "note", "n", "", "new note text (prompted on stdin when omitted)")
	return cmd
}

func (a *app) newRemoveCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "remove [index]",
		Short: "Remove one note (or all with -a)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			commit, err := a.head(ctx)
			if err != nil {
				return err
			}
			entries, err := a.mgr.Read(ctx, commit)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintf(a.out, "No git notes on %s.\n", commit)
				return nil
			}

			if all {
				if err := a.mgr.Write(ctx, commit, nil); err != nil {
					return err
				}
				fmt.Fprintf(a.out, "Removed all notes from %s.\n", commit)
				return nil
			}

			idx, err := selectIndex(firstArg(args), len(entries))
			if err != nil {
				return err
			}
			remaining := append(entries[:idx:idx], entries[idx+1:]...)
			if err := a.mgr.Write(ctx, commit, remaining); err != nil {
				return err
			}
			if len(remaining) == 0 {
				fmt.Fprintf(a.out, "Removed last note; cleared notes on %s.\n", commit)
			} else {
				fmt.Fprintf(a.out, "Removed note #%d from %s.\n", idx, commit)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&all, "all", "a", false, "remove all notes from HEAD")
	return cmd
}

func (a *app) newExportCmd() *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "export [base]",
		Short: "Write the review payload as JSON",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			payload, err := review.NewBuilder(a.git, a.mgr).Build(ctx, firstArg(args))
			if err != nil {
				return err
			}
			if err := review.Export(payload, output); err != nil {
				return err
			}
			fmt.Fprintf(a.out, "Wrote %d comment(s) to %s.\n", len(payload.Comments), output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "git-notes.json", "file to write the JSON payload to")
	return cmd
}

func (a *app) newSubmitCmd() *cobra.Command {
	var (
		file   string
		github bool
		gitlab bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "submit <number>",
		Short: "Post notes to PR/MR <number>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if github && gitlab {
				return fmt.Errorf("pass only one of --github / --gitlab")
			}

			opts := review.Options{Number: args[0], DryRun: dryRun}
			switch {
			case github:
				opts.Platform = review.GitHub
			case gitlab:
				opts.Platform = review.GitLab
			}

			ctx := cmd.Context()
			sub := review.NewSubmitter(a.git, a.mgr, review.NewBuilder(a.git, a.mgr))
			if file != "" {
				payload, err := review.Load(file)
				if err != nil {
					return err
				}
				return sub.SubmitPayload(ctx, payload, opts)
			}
			return sub.Submit(ctx, opts)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "post a pre-exported JSON file instead of computing")
	cmd.Flags().BoolVar(&github, "github", false, "force GitHub (default: detect from origin)")
	cmd.Flags().BoolVar(&gitlab, "gitlab", false, "force GitLab")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print payloads without posting")
	return cmd
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func selectIndex(arg string, n int) (int, error) {
	if arg == "" {
		if n == 1 {
			return 0, nil
		}
		return 0, fmt.Errorf("there are %d notes; specify which by index (0-%d)", n, n-1)
	}
	idx, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid index %q", arg)
	}
	if idx < 0 || idx >= n {
		return 0, fmt.Errorf("index %d out of range (0-%d)", idx, n-1)
	}
	return idx, nil
}

var (
	headerStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	cellStyle      = lipgloss.NewStyle().Padding(0, 1)
	submittedStyle = lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("2"))
)

func (a *app) renderEntries(entries []note.Entry) {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("240"))).
		Headers("#", "LOCATION", "CODE", "NOTE", "SUBMITTED").
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case col == 4:
				return submittedStyle
			default:
				return cellStyle
			}
		})
	for i, e := range entries {
		t.Row(strconv.Itoa(i), e.Location().Label(), preview(e.Code), preview(e.Note), submittedMark(e.Submitted))
	}
	fmt.Fprintln(a.out, t.Render())
}

func submittedMark(submitted bool) string {
	if submitted {
		return "✓"
	}
	return ""
}

func preview(s string) string {
	head, _, multiline := strings.Cut(s, "\n")
	head = strings.ReplaceAll(head, "\t", " ")
	head = strings.TrimSpace(head)
	if multiline {
		head += " …"
	}
	return head
}

func prompt(label string) (string, error) {
	fmt.Print(label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
