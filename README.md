# projx

**projx** is a universal project manager for developers. It keeps all your
codebases (repos and workspaces) in one place — with aliases, tags, groups,
favorites, and cached git status — and opens any of them in your editor of
choice (Zed, VS Code, Neovim, ...) with a single command.

Built to feed desktop shell launchers (Noctalia, DMS) and the terminal.

## Why

- **One list for every project**, regardless of which editor you use.
- **Launch any project in any editor** — new window or current window.
- **Auto-discover** git repositories; **migrate** from VS Code / Zed in one command.
- **Fast fuzzy TUI picker** with a preview pane.
- **Plugins for Noctalia and DMS** so you can switch projects from your launcher.

## Features

- Aliases, tags, groups, favorites
- Auto-scan directories for projects (by markers like `.git`, `Cargo.toml`, `package.json`)
- Import from Zed, VS Code, Cursor, VSCodium, Windsurf, and the Project Manager extension
- Cached git branch + dirty file count
- Workspace sets (open several repos together)
- Editor profiles (VS Code profiles) per project
- `doctor` / `prune` for missing or renamed paths
- Export / import JSON backups
- Shell completion (bash, zsh, fish)

## Quick start

```sh
# Install
go install github.com/bal16/projx/cmd/pjx@latest

# Auto-discover git repositories
pjx scan ~/projects ~/workspace --max-depth 2

# Or migrate from an existing editor
pjx import zed        # or: pjx import --all

# Add a single project manually
pjx add ~/dotfiles --name "Dotfiles" --tag Personal --favorite

# List / filter
pjx list
pjx list --tag Work

# Open
pjx open my-app
pjx open my-app --new-window

# Interactive picker
pjx tui
```

## Plugins

- **Noctalia v5** — `plugins/project-manager/` (launcher trigger `pjx`)
- **DMS (DankMaterialShell)** — `plugins/dms/` (launcher trigger `pjx`)

Both call `pjx list --format=delimited` and `pjx open`. See
[docs/PLUGINS.md](docs/PLUGINS.md) for installation.

## Data locations

- Database: `~/.local/share/projx/projects.db` (SQLite, WAL)
- Config: `~/.config/projx/config.json`
- Cache: `~/.cache/projx/` (reserved)

For dev/CI isolation use `PROJX_HOME` or `PROJX_*_DIR`. See
[docs/CONFIGURATION.md](docs/CONFIGURATION.md).

## Documentation

- [Getting started](docs/GETTING_STARTED.md)
- [Usage & command reference](docs/USAGE.md)
- [Configuration & data locations](docs/CONFIGURATION.md)
- [Plugins](docs/PLUGINS.md)
- [Development](docs/DEVELOPMENT.md)
- [Contributing](CONTRIBUTING.md)

## License

MIT