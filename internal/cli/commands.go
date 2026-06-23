package cli

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"gitnotes/internal/note"
	"gitnotes/internal/review"
)

// newFlagSet builds a flag set that reports errors to a.out and never calls
// os.Exit (so Run stays testable).
func (a *app) newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.out)
	return fs
}

// parseArgs parses args allowing flags and positionals to be interleaved
// (Go's flag package stops at the first positional). It returns the positional
// arguments in order.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positionals []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, errUsage
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positionals, nil
		}
		positionals = append(positionals, rest[0])
		args = rest[1:]
	}
}

// firstPositional returns the first positional argument, or "".
func firstPositional(p []string) string {
	if len(p) == 0 {
		return ""
	}
	return p[0]
}

func (a *app) runAdd(ctx context.Context, args []string) error {
	fs := a.newFlagSet("add")
	file := fs.String("f", "", "location: file[:line] or file[:start-end]")
	text := fs.String("n", "", "note text (required)")
	general := fs.Bool("g", false, "add a general commit-level note")
	if _, err := parseArgs(fs, args); err != nil {
		return errUsage
	}

	if *general == (*file != "") {
		return fmt.Errorf("specify exactly one of -f <location> or -g")
	}
	body := strings.TrimSpace(*text)
	if body == "" {
		return fmt.Errorf("note text is required: -n <note>")
	}

	var loc note.Location
	if !*general {
		parsed, err := note.ParseSpec(*file)
		if err != nil {
			return err
		}
		loc = parsed
	}

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
}

func (a *app) runList(ctx context.Context, args []string) error {
	fs := a.newFlagSet("list")
	if err := fs.Parse(args); err != nil {
		return errUsage
	}

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
}

func (a *app) runEdit(ctx context.Context, args []string) error {
	fs := a.newFlagSet("edit")
	text := fs.String("n", "", "new note text (prompted on stdin when omitted)")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

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

	idx, err := selectIndex(firstPositional(positionals), len(entries))
	if err != nil {
		return err
	}

	newText := strings.TrimSpace(*text)
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
}

func (a *app) runRemove(ctx context.Context, args []string) error {
	fs := a.newFlagSet("remove")
	all := fs.Bool("a", false, "remove all notes from HEAD")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

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

	if *all {
		if err := a.mgr.Write(ctx, commit, nil); err != nil {
			return err
		}
		fmt.Fprintf(a.out, "Removed all notes from %s.\n", commit)
		return nil
	}

	idx, err := selectIndex(firstPositional(positionals), len(entries))
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
}

func (a *app) runExport(ctx context.Context, args []string) error {
	fs := a.newFlagSet("export")
	output := fs.String("o", "git-notes.json", "file to write the JSON payload to")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		return err
	}

	payload, err := review.NewBuilder(a.git, a.mgr).Build(ctx, firstPositional(positionals))
	if err != nil {
		return err
	}
	if err := review.Export(payload, *output); err != nil {
		return err
	}
	fmt.Fprintf(a.out, "Wrote %d comment(s) to %s.\n", len(payload.Comments), *output)
	return nil
}

func (a *app) runSubmit(ctx context.Context, args []string) error {
	fs := a.newFlagSet("submit")
	file := fs.String("f", "", "post a pre-exported JSON file instead of computing")
	github := fs.Bool("github", false, "force GitHub (default: detect from origin)")
	gitlab := fs.Bool("gitlab", false, "force GitLab")
	dryRun := fs.Bool("dry-run", false, "print payloads without posting")
	positionals, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if *github && *gitlab {
		return fmt.Errorf("pass only one of --github / --gitlab")
	}

	number := firstPositional(positionals)
	if number == "" {
		return fmt.Errorf("the PR/MR number is required: %s submit [flags] <number>", appName)
	}

	opts := review.Options{Number: number, DryRun: *dryRun}
	switch {
	case *github:
		opts.Platform = review.GitHub
	case *gitlab:
		opts.Platform = review.GitLab
	}

	sub := review.NewSubmitter(a.git, review.NewBuilder(a.git, a.mgr))
	if *file != "" {
		payload, err := review.Load(*file)
		if err != nil {
			return err
		}
		return sub.SubmitPayload(ctx, payload, opts)
	}
	return sub.Submit(ctx, opts)
}

// selectIndex resolves a user-supplied index string against n entries. An empty
// string is allowed only when there is exactly one entry (defaulting to 0).
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

// renderEntries prints the entries as an aligned table.
func (a *app) renderEntries(entries []note.Entry) {
	tw := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "#\tLOCATION\tCODE\tNOTE")
	for i, e := range entries {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", i, e.Location().Label(), preview(e.Code), preview(e.Note))
	}
	_ = tw.Flush()
}

// preview renders a one-line, tab-free cell for table display: the first line
// of s with surrounding space trimmed, internal tabs turned into spaces, and an
// ellipsis when the value spans multiple lines.
func preview(s string) string {
	head, _, multiline := strings.Cut(s, "\n")
	head = strings.ReplaceAll(head, "\t", " ")
	head = strings.TrimSpace(head)
	if multiline {
		head += " …"
	}
	return head
}

// prompt reads a single trimmed line from stdin.
func prompt(label string) (string, error) {
	fmt.Print(label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
