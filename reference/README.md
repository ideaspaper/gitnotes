# projector

A powerful command-line project manager inspired by the [VS Code Project Manager](https://marketplace.visualstudio.com/items?itemName=alefragnani.project-manager) extension. Easily access your projects, organize them with tags, and auto-detect Git, SVN, and Mercurial repositories.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [add](#add)
  - [list](#list)
  - [open](#open)
  - [remove](#remove)
  - [edit](#edit)
  - [scan](#scan)
  - [select](#select)
  - [tags](#tags)
  - [clear-cache](#clear-cache)
  - [completion](#completion)
- [Configuration](#configuration)
- [Projects File](#projects-file)
- [Global Flags](#global-flags)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Features

- Save any folder as a favorite project
- Auto-detect Git, SVN, Mercurial repositories
- Auto-detect VS Code workspaces
- Organize projects with custom tags
- Open projects in your preferred editor (VS Code, Cursor, Vim, etc.)
- Interactive project selection
- Colored output with customizable display
- Cache detected projects for fast access
- Shell completion for bash, zsh, fish, and PowerShell
- Cross-platform support (macOS, Linux, Windows)

## Installation

### Using Homebrew

```bash
brew tap ideaspaper/tap
brew install --cask projector-cli
```

### Using `go install`

```bash
go install github.com/ideaspaper/projector@latest
```

This will install the binary as `projector` in your `$GOPATH/bin` directory. Make sure `$GOPATH/bin` is in your `PATH`.

### From Source

```bash
# Clone the repository
git clone https://github.com/ideaspaper/projector.git
cd projector

# Build the binary
make build

# Install to $GOPATH/bin
make install

# Or move to PATH manually
sudo mv projector /usr/local/bin/
```

### Dependencies

Requires Go 1.24.4 or later.

```bash
go mod download
```

## Quick Start

### Add Your First Project

```bash
# Add current directory as a project
projector add

# Add a specific directory
projector add ~/projects/myapp

# Add with a custom name
projector add ~/projects/myapp --name "My Application"

# Add with tags
projector add --name "Work API" --tag Work --tag Backend
```

### List Projects

```bash
# List all projects
projector list

# List with paths
projector list --path

# List grouped by type
projector list --grouped

# Filter by tag
projector list --tag Work
```

### Open a Project

```bash
# Open by name
projector open myproject

# Open in new window
projector open myproject --new-window

# Open with a specific editor
projector open myproject --editor vim

# Interactive selection
projector open
```

### Scan for Repositories

```bash
# Scan for Git repositories
projector scan --git ~/projects

# Scan for all repository types
projector scan --all ~/code

# Scan with custom depth
projector scan --git --depth 5 ~/projects
```

## Commands

### add

Add a folder as a project to your favorites.

```bash
projector add [path] [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--name` | `-n` | Project name (defaults to folder name) |
| `--tag` | `-t` | Tags for the project (can be repeated) |
| `--enabled` | | Whether the project is enabled (default: true) |

**Examples:**

```bash
# Add current directory
projector add

# Add with custom name
projector add ~/projects/api --name "Backend API"

# Add with multiple tags
projector add --name "Frontend" --tag Work --tag React --tag TypeScript
```

### list

List all saved and detected projects.

```bash
projector list [flags]
```

**Aliases:** `ls`

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--tag` | `-t` | Filter projects by tag |
| `--path` | `-p` | Show project paths |
| `--grouped` | `-g` | Group projects by type |
| `--all` | `-a` | Include disabled projects |
| `--favorites` | | Show only favorites |
| `--git` | | Show only Git repositories |
| `--svn` | | Show only SVN repositories |
| `--mercurial` | | Show only Mercurial repositories |
| `--vscode` | | Show only VS Code workspaces |
| `--any` | | Show only any-folder projects |

**Examples:**

```bash
# List all projects
projector list

# List with full paths
projector list --path

# List grouped by type (Favorites, Git, SVN, etc.)
projector list --grouped

# Filter by tag
projector list --tag Work

# Show only Git repositories
projector list --git
```

### open

Open a project in your configured editor.

```bash
projector open [project-name] [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--new-window` | `-n` | Open in a new window |
| `--editor` | `-e` | Editor to use (overrides config) |
| `--tag` | `-t` | Filter projects by tag |
| `--grouped` | `-g` | Group projects by type (overrides config) |
| `--favorites` | | Show only favorites |
| `--git` | | Show only Git repositories |
| `--svn` | | Show only SVN repositories |
| `--mercurial` | | Show only Mercurial repositories |
| `--vscode` | | Show only VS Code workspaces |
| `--any` | | Show only any-folder projects |

**Supported Editors:**

- `code` / `vscode` - Visual Studio Code
- `cursor` - Cursor
- `subl` / `sublime` - Sublime Text
- `atom` - Atom
- `vim` / `nvim` - Vim / Neovim
- `emacs` - Emacs
- `idea` / `intellij` - IntelliJ IDEA
- `webstorm` - WebStorm
- `goland` - GoLand
- `pycharm` - PyCharm
- `open` - macOS default handler
- `xdg-open` - Linux default handler
- `explorer` - Windows Explorer

**Examples:**

```bash
# Open by name (fuzzy matching)
projector open myproject

# Open in a new window
projector open myproject -n

# Open with Vim
projector open myproject --editor vim

# Interactive selection (no argument)
projector open

# Filter interactive selection by tag
projector open --tag Work

# Interactive selection with flat list (no grouping)
projector open --grouped=false

# Interactive selection with grouping (overrides config)
projector open -g

# Open only from Git repositories
projector open --git

# Open only from favorites
projector open --favorites
```

**Interactive Selection:**

When no project name is provided, an interactive menu is displayed:

```
Select a project to open:

Favorites
  [1] my-app [Work] - /Users/you/projects/my-app
  [2] blog - /Users/you/projects/blog

Git Repositories
  [3] api - /Users/you/work/api
  [4] frontend [React] - /Users/you/work/frontend

Enter project number (or 'q' to quit):
```

Projects are grouped by type (if `groupList` is enabled in config), showing tags and truncated paths. Enter the number to open that project.

### remove

Remove a project from favorites.

```bash
projector remove <project-name>
```

**Aliases:** `rm`, `delete`

**Examples:**

```bash
projector remove myproject
projector rm old-project
```

### edit

Edit a project's properties.

```bash
projector edit <project-name> [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--name` | New project name |
| `--path` | New project path |
| `--enabled` | Enable/disable project (true/false) |
| `--add-tag` | Add a tag to the project (can be repeated) |
| `--remove-tag` | Remove a tag from the project (can be repeated) |

**Examples:**

```bash
# Rename a project
projector edit myproject --name "New Name"

# Update path
projector edit myproject --path ~/new/location

# Disable a project
projector edit myproject --enabled=false

# Add tags
projector edit myproject --add-tag Work --add-tag Important

# Remove a tag
projector edit myproject --remove-tag Old

# Add and remove tags in one command
projector edit myproject --add-tag Backend --remove-tag Frontend
```

### scan

Scan directories for repositories and workspaces.

```bash
projector scan [paths...] [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--git` | | Scan for Git repositories |
| `--svn` | | Scan for SVN repositories |
| `--mercurial` | | Scan for Mercurial repositories |
| `--vscode` | | Scan for VS Code workspaces |
| `--any` | | Scan for any folder |
| `--all` | `-a` | Scan for all types |
| `--depth` | `-d` | Maximum scan depth (0 = use config) |

**Examples:**

```bash
# Scan for Git repos in a directory
projector scan --git ~/projects

# Scan for all types in configured base folders
projector scan --all

# Scan multiple directories
projector scan --git ~/work ~/personal

# Limit scan depth
projector scan --git --depth 3 ~/code
```

### select

Select a project and output its path to stdout.

```bash
projector select [project-name] [flags]
```

If no project name is provided, an interactive selection is shown.
This is useful for scripting and shell integration.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--tag` | `-t` | Filter projects by tag |
| `--grouped` | `-g` | Group projects by type (overrides config) |
| `--favorites` | | Show only favorites |
| `--git` | | Show only Git repositories |
| `--svn` | | Show only SVN repositories |
| `--mercurial` | | Show only Mercurial repositories |
| `--vscode` | | Show only VS Code workspaces |
| `--any` | | Show only any-folder projects |

**Examples:**

```bash
# Interactive selection
projector select

# Select by name
projector select myproject

# Select by partial name match
projector select my

# Filter interactive selection by tag
projector select --tag Work

# Interactive selection with flat list (no grouping)
projector select --grouped=false

# Interactive selection with grouping (overrides config)
projector select -g

# Select only from Git repositories
projector select --git

# Select only from favorites
projector select --favorites
```

**Interactive Selection:**

Similar to `open`, when no project name is provided, an interactive menu is displayed with 1-based numbering. The selected project's path is output to stdout, making it ideal for shell scripting.

**Shell Function for cd:**

Add this to your `.bashrc` or `.zshrc` to create a `pjcd` command that selects a project and changes to its directory:

```bash
pjcd() {
  local dir
  dir=$(projector select "$@")
  [ -n "$dir" ] && [ -d "$dir" ] && cd "$dir"
}
```

Usage:

```bash
# Interactive selection
pjcd

# Direct selection by name
pjcd myproject
```

### tags

List all unique tags currently in use by projects.

```bash
projector tags
```

**Examples:**

```bash
# List all tags in use
projector tags
```

**Output:**

```
Tags in use:
  - Backend
  - Frontend
  - Go
  - Personal
  - Work
```

### clear-cache

Clear the cached auto-detected projects.

```bash
projector clear-cache
```

**Aliases:** `cc`

This command removes the cache file (`~/.projector/cache.json`) that stores auto-detected repositories (Git, SVN, Mercurial, VS Code workspaces, and any-folder projects).

**Note:** This does not affect your saved favorites in `projects.json`. After clearing the cache, run `projector scan` to re-detect projects.

**Examples:**

```bash
# Clear the project cache
projector clear-cache

# Using the alias
projector cc

# Clear cache and re-scan for Git repositories
projector clear-cache && projector scan --git ~/projects
```

### completion

Generate shell completion scripts.

```bash
projector completion [bash|zsh|fish|powershell]
```

## Configuration

Configuration is stored in `~/.projector/config.json`:

```json
{
  "sortList": "Name",
  "groupList": true,
  "showColors": true,
  "checkInvalidPathsBeforeListing": true,
  "showParentFolderInfoOnDuplicates": false,
  "filterOnFullPath": false,
  "removeCurrentProjectFromList": true,
  "cacheProjectsBetweenSessions": true,
  "ignoreProjectsWithinProjects": false,
  "supportSymlinksOnBaseFolders": false,
  "editor": "code",
  "openInNewWindow": false,
  "gitBaseFolders": ["~/projects", "~/work"],
  "gitIgnoredFolders": [
    "node_modules",
    "out",
    "typings",
    "test",
    ".haxelib",
    "vendor"
  ],
  "gitMaxDepthRecursion": 4,
  "svnBaseFolders": [],
  "svnIgnoredFolders": ["node_modules", "out", "typings", "test"],
  "svnMaxDepthRecursion": 4,
  "hgBaseFolders": [],
  "hgIgnoredFolders": ["node_modules", "out", "typings", "test", ".haxelib"],
  "hgMaxDepthRecursion": 4,
  "vscodeBaseFolders": [],
  "vscodeIgnoredFolders": ["node_modules", "out", "typings", "test"],
  "vscodeMaxDepthRecursion": 4,
  "anyBaseFolders": [],
  "anyIgnoredFolders": ["node_modules", "out", "typings", "test"],
  "anyMaxDepthRecursion": 4,
  "projectsLocation": ""
}
```

### Configuration Options

| Option                             | Description                                                               | Default                 |
| ---------------------------------- | ------------------------------------------------------------------------- | ----------------------- |
| `sortList`                         | Sort order: `Name`, `Path`, `Saved`                                       | `Name`                  |
| `groupList`                        | Group projects by type in list (can be overridden with `--grouped` flag)  | `true`                  |
| `showColors`                       | Enable colored output                                                     | `true`                  |
| `checkInvalidPathsBeforeListing`   | Check if paths exist                                                      | `true`                  |
| `showParentFolderInfoOnDuplicates` | Append the parent folder when duplicate project names are displayed       | `false`                 |
| `filterOnFullPath`                 | Allow `open` and `select` to match against project paths as well as names | `false`                 |
| `removeCurrentProjectFromList`     | Hide the current working directory from project lists and selectors       | `true`                  |
| `editor`                           | Default editor command                                                    | `code`                  |
| `openInNewWindow`                  | Always open in new window                                                 | `false`                 |
| `gitBaseFolders`                   | Folders to scan for Git repos                                             | `[]`                    |
| `gitIgnoredFolders`                | Folders to skip when scanning Git                                         | `["node_modules", ...]` |
| `gitMaxDepthRecursion`             | Max depth for Git scanning                                                | `4`                     |
| `cacheProjectsBetweenSessions`     | Cache detected projects                                                   | `true`                  |
| `ignoreProjectsWithinProjects`     | Skip nested projects                                                      | `false`                 |
| `supportSymlinksOnBaseFolders`     | Follow symlinks                                                           | `false`                 |
| `projectsLocation`                 | Custom location for projects.json                                         | `""`                    |

## Projects File

Saved projects are stored in `~/.projector/projects.json`:

```json
[
  {
    "name": "My App",
    "rootPath": "~/projects/myapp",
    "tags": ["Work", "Go"],
    "enabled": true
  },
  {
    "name": "Website",
    "rootPath": "~/projects/website",
    "tags": ["Personal", "React"],
    "enabled": true
  }
]
```

You can use `~` or `$home` in paths - they will be expanded automatically.

## Global Flags

| Flag         | Short | Description            |
| ------------ | ----- | ---------------------- |
| `--no-color` |       | Disable colored output |
| `--verbose`  | `-v`  | Verbose output         |
| `--version`  |       | Show version           |
| `--help`     | `-h`  | Show help              |

## Examples

### Complete Workflow

```bash
# Add some projects
projector add ~/projects/backend --name "Backend API" --tag Work --tag Go
projector add ~/projects/frontend --name "Frontend App" --tag Work --tag React
projector add ~/personal/blog --name "My Blog" --tag Personal

# Configure base folders for scanning
# Edit ~/.projector/config.json and add:
# "gitBaseFolders": ["~/projects", "~/work"]

# Scan for Git repositories
projector scan --git

# List all projects grouped
projector list --grouped --path

# Open a project
projector open backend

# Filter by tag
projector list --tag Work

# Remove a project
projector remove "My Blog"
```

### Script Integration

```bash
#!/bin/bash
# Open project in tmux session

# Non-interactive: pass project name directly
PROJECT=$(projector select myproject 2>/dev/null)

# Or interactive: let user choose
# PROJECT=$(projector select 2>/dev/null)

if [ -n "$PROJECT" ]; then
    SESSION_NAME=$(basename "$PROJECT")
    tmux new-session -d -s "$SESSION_NAME" -c "$PROJECT"
    tmux attach-session -t "$SESSION_NAME"
fi
```

### FZF Integration

```bash
# Add to your .bashrc or .zshrc
proj() {
    local project
    project=$(projector list --path | fzf --height 40% --reverse)
    if [ -n "$project" ]; then
        # Extract path (second line)
        path=$(echo "$project" | tail -1 | xargs)
        cd "$path" || return
    fi
}
```

### Alfred/Raycast Integration

Create a script that outputs project paths for launcher integration:

```bash
#!/bin/bash
projector list --path | while IFS= read -r line; do
    if [[ $line == "  "* ]]; then
        echo "${line:2}"  # Path line (indented)
    fi
done
```

## Troubleshooting

### Project Not Found

```
Error: project 'myproject' not found
```

1. Check project name with `projector list`
2. Names are case-insensitive but must match
3. Use partial matching: `projector open my` will match `myproject`

### Path Does Not Exist

```
Error: project path does not exist
```

The project's directory has been moved or deleted. Update it:

```bash
projector edit myproject --path ~/new/location
```

Or remove and re-add:

```bash
projector remove myproject
projector add ~/new/location --name myproject
```

### Editor Not Opening

1. Verify the editor is installed and in PATH
2. Check config: `cat ~/.projector/config.json | grep editor`
3. Override with flag: `projector open myproject --editor code`

### Scan Not Finding Projects

1. Ensure base folders are configured in `config.json`
2. Check scan depth (default is 4)
3. Verify folders aren't in the ignored list
4. Try scanning a specific path: `projector scan --git ~/projects`

## Development

### Building

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Lint
make lint
```

### Project Structure

```
projector/
├── cmd/                    # Command implementations
│   ├── root.go            # Base command
│   ├── add.go             # Add command
│   ├── list.go            # List and scan commands
│   ├── open.go            # Open command
│   ├── select.go          # Select command
│   ├── manage.go          # Remove, edit, tag commands
│   └── completion.go      # Shell completions
├── pkg/
│   ├── config/            # Configuration
│   ├── models/            # Data structures
│   ├── output/            # Formatted output
│   ├── paths/             # Path utilities
│   ├── scanner/           # Repository detection
│   └── storage/           # JSON persistence
├── main.go
├── go.mod
├── Makefile
└── README.md
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Acknowledgments

- Inspired by [VS Code Project Manager](https://marketplace.visualstudio.com/items?itemName=alefragnani.project-manager)
- Built with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper)
