---
name: gitnotes
description: "Use when reviewing code and recording findings as line-anchored comments on a commit, or posting a review to a GitHub PR / GitLab MR. gitnotes stores review comments as git notes (CSV in refs/notes/commits), anchored to a file:line or file:start-end, capturing the code at that location, and can submit in-diff notes as PR/MR line comments and the rest as general comments. Trigger when the user asks to review a diff/PR/MR and leave inline comments, annotate specific lines, draft review comments, or push review comments to a pull/merge request."
trigger: /gitnotes
---

# /gitnotes

Drive the `gitnotes` CLI to record code-review findings as line-anchored git notes on a commit, then optionally post them to a GitHub PR or GitLab MR.

Notes are stored as CSV rows in `refs/notes/commits`, one per finding: `file,startLine,endLine,code,note,submitted`. Each note snapshots the source at its location (as of the commit), and `submit` posts in-diff notes as line comments and the rest as general comments — flagging each as `submitted` so re-running never double-posts.

## ⚠️ Agent rules (read first)

You run without a TTY, so the interactive picker / prompts will hang or error. Always use the **non-interactive** forms:

- **Always pass an explicit index** to `edit`, `remove`, `unsubmit` — the **1-based** number shown in `list`'s `#` column (never omit it — omitting opens an interactive picker that needs a terminal and errors with "interactive selection needs a terminal").
- **Always pass `-n "<text>"`** to `add` and `edit` (omitting `-n` makes `edit` wait on stdin).
- `list` is safe — it auto-detects the non-TTY and prints plain aligned text.
- To read notes programmatically, prefer `gitnotes list` (plain text: `# / LOCATION / CODE / NOTE / SUBMITTED`) or `gitnotes export -o <file>` then read the JSON.
- Verify availability first: `gitnotes version` (and that you're inside a git repo). `submit` additionally needs `gh` (GitHub) or `glab` (GitLab) installed and authenticated.

## Setup check

```sh
gitnotes version            # confirm the binary is on PATH
git rev-parse --is-inside-work-tree   # confirm a repo
```

If `gitnotes` is missing, tell the user to install it (`brew install ideaspaper/tap/gitnotes`, or build from source) — do not fall back to hand-editing `git notes`.

## Review workflow

1. **Read the change.** Inspect the diff / files under review (e.g. `git diff <base>...HEAD`, or read the files).
2. **Record each finding as a note**, anchored to the exact line(s):
   ```sh
   gitnotes add -f path/to/file.go:42 -n "use slog here"
   gitnotes add -f path/to/file.go:20-34 -n "extract this block; it duplicates X"
   gitnotes add -g -n "overall LGTM, two nits inline"   # general / commit-level
   ```
3. **Review what you recorded:**
   ```sh
   gitnotes list
   ```
4. **Fix mistakes by index** (from the `#` column in `list`):
   ```sh
   gitnotes edit 1 -n "actually prefer log/slog"
   gitnotes remove 1
   gitnotes remove -a            # clear all notes on the commit
   ```
5. **Submit to the PR/MR** (preview first):
   ```sh
   gitnotes submit 42 --dry-run  # print every payload, post nothing
   gitnotes submit 42            # post to PR/MR #42 (auto-detects GitHub/GitLab from origin)
   ```
   Already-submitted notes are skipped automatically. Use `--github` / `--gitlab` to override platform detection.
6. **Re-open a note for another round** with `gitnotes unsubmit <index>` (or `-a`), which clears the `submitted` flag so the next `submit` posts it again.

## Command reference (non-interactive subset)

| Command                                                     | Purpose                                                                                   |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `gitnotes add -f <file>:<line>          -n "<note>"`        | Add a single-line note                                                                    |
| `gitnotes add -f <file>:<start>-<end>   -n "<note>"`        | Add a block/range note                                                                    |
| `gitnotes add -f <file>                 -n "<note>"`        | Whole-file note (no code captured)                                                        |
| `gitnotes add -g                        -n "<note>"`        | General commit-level note                                                                 |
| `gitnotes list`                                             | List notes (plain text when non-TTY)                                                      |
| `gitnotes edit <index> -n "<note>"`                         | Replace a note's text                                                                     |
| `gitnotes remove <index>` / `gitnotes remove -a`            | Remove one note / all                                                                     |
| `gitnotes unsubmit <index>` / `gitnotes unsubmit -a`        | Clear the `submitted` flag                                                                |
| `gitnotes export [base] -o <file>`                          | Write the review payload as JSON (machine-readable)                                       |
| `gitnotes export --format md -o <file>`                     | Write HEAD's notes as a Markdown review (fenced code) — good for sharing a review summary |
| `gitnotes submit <number> [--github\|--gitlab] [--dry-run]` | Post notes to PR/MR `<number>`                                                            |
| `gitnotes version`                                          | Print the version                                                                         |

### Location specs

- `path/to/file.go:14` — a single line
- `path/to/file.go:1-17` — a block of lines (1 through 17, inclusive)
- `path/to/file.go` — the whole file (no code captured)

Paths are **relative to the repo root**.

## Targeting a commit other than HEAD

`add`, `list`, `edit`, `remove`, and `unsubmit` act on `HEAD` by default; pass `-c, --commit <commitish>` to target another commit:

```sh
gitnotes edit -c <sha> 1 -n "…"
```

`submit` and `export` always operate on `HEAD` (the `-c` flag does not affect them; `export`'s positional `[base]` is the diff base, not the commit whose notes are read).

## How `submit` decides line vs general

`submit` takes the PR/MR **number** and derives the diff base from it (GitHub's base branch, GitLab's `diff_refs.base_sha`). A note whose lines are **all inside that diff** posts as a **line comment** (ranges post as true multi-line comments); anything else posts as a **general** comment rendered as `location` + blank line + note text.

## Notes

- `gitnotes` is HEAD/commit-centric, not PR-centric, for storage — the notes live on the commit and are portable via `git notes`.
- The `submitted` flag (`✓`/`✗` in `list`) makes `submit` idempotent: run it as many times as you like; only new notes are posted.
- Don't hand-edit `refs/notes/commits` CSV directly — let `gitnotes` read/write it so the format stays consistent.
