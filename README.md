# gitnotes

A small Go CLI for managing **git notes** as reviewable, CSV-backed comments on
`HEAD` — and posting them as GitHub PR / GitLab MR review comments.

Each commit's note is a CSV document with one row per entry:

```
file,startLine,endLine,code,note
```

- `pkg/config/config.go:1-17` → `pkg/config/config.go,1,17,<captured code>,<note>`
- `pkg/config/config.go:14` → `pkg/config/config.go,14,,<captured code>,<note>`
- a general note → `,,,,<note>`

The `code` column holds the source captured from the file **as of the commit**
(falling back to the working tree). Multi-line code is CSV-quoted, so it
round-trips losslessly. Notes live in the standard `refs/notes/commits` ref.

## Install

Homebrew (via the tap):

```sh
brew install ideaspaper/tap/gitnotes
```

Or build from source (Go 1.26+):

```sh
git clone https://github.com/ideaspaper/gitnotes.git
cd gitnotes
make install        # builds with version stamped in, installs to $GOPATH/bin
# or just: make build   ->   ./gitnotes
```

Posting to a PR/MR additionally needs the [`gh`](https://cli.github.com/)
(GitHub) or [`glab`](https://gitlab.com/gitlab-org/cli) (GitLab) CLI, installed
and authenticated.

A `gn` shell alias mirrors the original nushell tool:

```sh
alias gn=gitnotes
```

## Usage

```
gitnotes add -f <file[:line]|file[:start-end]> -n <note>                Add a line / block note
gitnotes add -g -n <note>                                               Add a general note
gitnotes list                                                           List HEAD's notes
gitnotes edit [index] [-n <note>]                                       Edit a note's text
gitnotes remove [index] | -a                                            Remove one note (or all)
gitnotes export [base] [-o <file>]                                      Write the review payload as JSON
gitnotes submit <number> [--github|--gitlab] [-f <file>] [--dry-run]    Post notes to PR/MR <number>
gitnotes version                                                        Print the version
```

### Location specs

| spec                   | meaning                           |
| ---------------------- | --------------------------------- |
| `path/to/file.go:14`   | a single line                     |
| `path/to/file.go:1-17` | a block of lines (1 through 17)   |
| `path/to/file.go`      | the whole file (no code captured) |

### Examples

```sh
gitnotes add -f internal/cli/commands.go:42 -n "use slog here"
gitnotes add -f internal/note/entry.go:20-34 -n "this block needs a doc comment"
gitnotes add -g -n "overall LGTM, two nits inline"
gitnotes list
gitnotes edit 1 -n "actually prefer log/slog"
gitnotes remove 0
gitnotes submit 42 --dry-run      # preview without posting
gitnotes submit 42                # post to PR/MR #42
```

## How `submit` works

`submit` requires the **PR/MR number** and derives the **diff base** from it —
GitHub's base branch (e.g. `origin/main`), GitLab's `diff_refs.base_sha` — so
there is no `-b` flag. It then classifies each note against that diff:

- A note whose lines are **all inside the diff** becomes a **line comment**. A
  range (`file:1-17`) posts as a true **multi-line** comment — GitHub via
  `start_line`/`line`, GitLab via a `line_range` — anchored across the block.
- Any other note becomes a **general** PR/issue comment (GitLab: an MR note),
  rendered as its location, a blank line, then the note:

  ```
  path/to/file.go:10-14

  the note text
  ```

`submit` posts directly — no file needed — auto-detecting the platform from the
`origin` remote (override with `--github`/`--gitlab`). It shells out to the
[`gh`](https://cli.github.com/) (GitHub) or [`glab`](https://gitlab.com/gitlab-org/cli)
(GitLab) CLI, so those must be installed and authenticated. Use `--dry-run` to
print every payload without posting, or `-f <file>` to post a pre-`export`ed
JSON. `export` keeps an optional `base` argument (default `HEAD^`) for producing
a standalone payload.

## Layout

```
main.go                  entrypoint (signal handling)
internal/
  gitcmd/                the single boundary to the git CLI
  note/                  Entry model, location parsing, CSV codec, CRUD manager
  review/                payload builder + export + gh/glab submit
  cli/                   stdlib-flag subcommand dispatch (transport layer)
```
