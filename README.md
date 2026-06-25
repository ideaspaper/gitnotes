# 📝 gitnotes _(git + notes + review)_

A small, fast Go CLI that turns **git notes** into reviewable, line-anchored comments on a commit — then posts them as **GitHub PR** / **GitLab MR** review comments.

Annotate code while you read it, browse your notes in a fuzzy-searchable TUI, and push them to a pull/merge request when you're done — without leaving the terminal.

## ✨ Features

- 📌 **Line, block, or general notes** — annotate a single line, a range (`file:1-17`), a whole file, or the commit itself.
- 🧷 **Code capture** — each note snapshots the source at its location, **as of the commit** (falling back to the working tree), so the context travels with the note.
- 🔍 **Interactive TUI** — `list` opens a **fuzzy-searchable** picker (powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss)) with a live preview; pick a note to see its full detail. `edit`, `remove`, and `unsubmit` open the **same picker** when you omit the index, so you never have to look up a note number.
- 🚀 **Post to PR/MR** — `submit` classifies each note against the PR/MR diff and posts in-diff notes as **line comments** (true multi-line ranges) and the rest as **general comments**, via the `gh` / `glab` CLIs.
- ✅ **Submit-once tracking** — every posted note is flagged `submitted`, so re-running `submit` skips it and never double-posts. `unsubmit` clears the flag.
- 🗂️ **Plain, portable storage** — notes are CSV rows in the standard `refs/notes/commits` ref; inspectable with plain `git`, scriptable from any editor.
- 🧭 **Any commit** — every command works on `HEAD` by default, or any commit with `-c, --commit`.

## ⚡️ Requirements

- [**git**](https://git-scm.com/) — notes are stored in `refs/notes/commits`.
- [**Go**](https://go.dev/) >= 1.26 — only to build from source (not needed for the Homebrew install).
- [**`gh`**](https://cli.github.com/) (GitHub) or [**`glab`**](https://gitlab.com/gitlab-org/cli) (GitLab) — **_(optional)_** only required for `submit`, installed and authenticated.

## 📦 Installation

Install with Homebrew (via the tap):

```sh
brew install ideaspaper/tap/gitnotes
```

Or build from source:

```sh
git clone https://github.com/ideaspaper/gitnotes.git
cd gitnotes
make install        # builds with the version stamped in, installs to $GOPATH/bin
# or just: make build   ->   ./gitnotes
```

Verify it's on your `PATH`:

```sh
gitnotes version
```

## 🚀 Usage

| Command                                                     | Description                                                    |
| ----------------------------------------------------------- | -------------------------------------------------------------- |
| `gitnotes add -f <loc> -n <note>`                           | Add a line / block / whole-file note                           |
| `gitnotes add -g -n <note>`                                 | Add a general (commit-level) note                              |
| `gitnotes list`                                             | Browse notes in a fuzzy-searchable TUI (plain text when piped) |
| `gitnotes edit [index] [-n <note>]`                         | Edit a note's text (interactive picker if index omitted)       |
| `gitnotes remove [index] \| -a`                             | Remove one note (or all with `-a`)                             |
| `gitnotes submit <number> [--github\|--gitlab] [--dry-run]` | Post notes to PR/MR `<number>`                                 |
| `gitnotes unsubmit [index] \| -a`                           | Clear a note's `submitted` flag so `submit` posts it again     |
| `gitnotes export [-o <file>]`                               | Write HEAD's notes as a Markdown review                        |
| `gitnotes version`                                          | Print the version                                              |

> Commands act on `HEAD` by default. Pass `-c, --commit <commitish>` to target another commit (e.g. `gitnotes edit -c <sha> 1 -n "…"`). This applies to `add`, `list`, `edit`, `remove`, and `unsubmit`; `submit` and `export` always operate on `HEAD`.

### 📍 Location specs

| Spec                   | Meaning                           |
| ---------------------- | --------------------------------- |
| `path/to/file.go:14`   | a single line                     |
| `path/to/file.go:1-17` | a block of lines (1 through 17)   |
| `path/to/file.go`      | the whole file (no code captured) |

### 💡 Examples

```sh
gitnotes add -f internal/cli/commands.go:42 -n "use slog here"
gitnotes add -f internal/note/entry.go:20-34 -n "this block needs a doc comment"
gitnotes add -g -n "overall LGTM, two nits inline"
gitnotes list                     # fuzzy-search, preview, and inspect notes
gitnotes edit                     # pick a note interactively, then edit it
gitnotes remove 1
gitnotes submit 42 --dry-run      # preview every payload without posting
gitnotes submit 42                # post to PR/MR #42
```

## 🗂️ Note format

Each commit's note is a CSV document with one row per entry:

```
file,startLine,endLine,code,note,submitted
```

- `pkg/config/config.go:1-17` → `pkg/config/config.go,1,17,<captured code>,<note>,`
- `pkg/config/config.go:14` → `pkg/config/config.go,14,,<captured code>,<note>,`
- a general note → `,,,,<note>,`

The `code` column holds the source captured from the file **as of the commit**; multi-line code is CSV-quoted, so it round-trips losslessly. The `submitted` column is `true` once the entry has been posted to a PR/MR. Notes live in the standard `refs/notes/commits` ref.

## 📤 How `submit` works

`submit` requires the **PR/MR number** and derives the **diff base** from it — GitHub's base branch (e.g. `origin/main`), GitLab's `diff_refs.base_sha` — so there's no `-b` flag. It then classifies each note against that diff:

- A note whose lines are **all inside the diff** becomes a **line comment**. A range (`file:1-17`) posts as a true **multi-line** comment — GitHub via `start_line`/`line`, GitLab via a `line_range` — anchored across the block.
- Any other note becomes a **general** PR/issue comment (GitLab: an MR note), rendered as its location, a blank line, then the note:

  ```
  path/to/file.go:10-14

  the note text
  ```

Each posted entry is flagged `submitted`, so re-running `submit` skips it (`• … already submitted, skipping`) and only posts new notes — a `--dry-run` never sets the flag, and `unsubmit` clears it.

`submit` auto-detects the platform from the `origin` remote (override with `--github`/`--gitlab`) and shells out to `gh` / `glab`. Use `--dry-run` to print every payload without posting.

### 📄 Markdown export

`gitnotes export [-o <file>]` writes HEAD's notes as a readable Markdown review (default `git-notes.md`) — one section per note with its location, the captured code in a fenced block, and the note text:

````markdown
# Review notes (a1b2c3d — Fix the parser)

2 note(s).

## #1 — `internal/cli/cli.go:14-20`

```go
const appName = "gitnotes"
```

needs a doc comment here

## #2 — `(general)`

overall LGTM
````

## 🤖 Claude Code skill

This repo carries a [Claude Code](https://claude.com/claude-code) skill definition at [`.claude/skills/gitnotes/SKILL.md`](.claude/skills/gitnotes/SKILL.md). Install it into your user skills folder so the agent can drive gitnotes while reviewing **any** codebase — recording findings as line-anchored notes, listing/editing them, and submitting to a PR/MR (invoke with `/gitnotes`):

```sh
mkdir -p ~/.claude/skills
cp -r .claude/skills/gitnotes ~/.claude/skills/
```

It teaches the agent the **non-interactive** command forms (explicit index, `-n`, reading notes via `list`), since an agent has no TTY and the interactive picker needs one.

## 🪪 License

See [LICENSE](LICENSE).
