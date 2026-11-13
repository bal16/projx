# projx

Universal project manager for developers. Manages projects/workspaces with
aliases, tags, groups and cached git status, and launches them in your
preferred editor (zed, vscode, nvim, ...).

Designed to feed desktop shell launchers (Noctalia, DMS) and the terminal.

## Install

```sh
go install github.com/bal16/projx/cmd/pjx@latest
# or build locally
make build
```

## Quick start

```sh
# Auto-discover git repositories
pjx scan ~/projects ~/workspace --max-depth 2

# Or migrate from an existing editor (one-time)
pjx import zed
pjx import --all

# Add a single project manually
pjx add ~/dotfiles --name "Dotfiles" --tag Personal --favorite

# List (text)
pjx list
pjx list --tag Work

# List (JSON, for launchers)
pjx list --json

# List (delimited, for shell launcher scripts)
pjx list --format=delimited

# Open
pjx open my-app
pjx open my-app --new-window
pjx open my-app --editor nvim

# Refresh cached git status
pjx refresh

# Config
pjx config
pjx config set default_editor zed
pjx config set scan_roots '["~/projects","~/workspace"]'
```

## Data

- Database: `~/.local/share/projx/projects.db` (SQLite, WAL mode)
- Config: `~/.config/projx/config.json`

## Plugins

- `plugins/noctalia/` - Noctalia v5 launcher provider (trigger `/pm` or `pm`)
- `plugins/dms/` - DMS (DankMaterialShell) launcher plugin (trigger `pm`)

Both call `pjx list --format=delimited` and `pjx open`.

## License

MIT
