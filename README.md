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

# Interactive TUI picker (fuzzy search, preview pane)
pjx tui

# Open
pjx open my-app
pjx open my-app --new-window
pjx open my-app --editor nvim

# Switch editor / profile for a project (persistent)
pjx set my-app --editor code --profile "Work Profile"
pjx set my-app --editor zed          # clear profile by omitting --profile... actually use --profile ""
pjx set my-app --editor code --profile ""   # clear profile

# One-off override (does not change stored project)
pjx open my-app --editor vscode --profile "Personal"

# Set default profile per editor
pjx config set editor_profiles '{"code":"Default VSCode Profile"}'

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
- Cache: `~/.cache/projx/` (reserved for future remote metadata cache)

### Isolating dev vs production

Production uses the default XDG paths above. For development/CI, isolate with env vars so test data never pollutes your real DB/config:

```sh
# All-in-one isolation (recommended for dev)
export PROJX_HOME=/tmp/projx-dev
pjx scan ~/scratchpad

# Or granular overrides
export PROJX_DATA_DIR=/tmp/projx-data
export PROJX_CONFIG_DIR=/tmp/projx-config
export PROJX_CACHE_DIR=/tmp/projx-cache
```

Resolution order:
- Data: `PROJX_DATA_DIR` > `PROJX_HOME/data` > `XDG_DATA_HOME/projx` > `~/.local/share/projx`
- Config: `PROJX_CONFIG_DIR` > `PROJX_HOME/config` > `XDG_CONFIG_HOME/projx` > `~/.config/projx`
- Cache: `PROJX_CACHE_DIR` > `PROJX_HOME/cache` > `XDG_CACHE_HOME/projx` > `~/.cache/projx`

## Plugins

- `plugins/noctalia/` - Noctalia v5 launcher provider (trigger `pjx`)
- `plugins/dms/` - DMS (DankMaterialShell) launcher plugin (trigger `pjx`)

Both call `pjx list --format=delimited` and `pjx open`.

### Installing the Noctalia plugin

```sh
noctalia msg plugins source add dev path /path/to/projx/plugins/noctalia
noctalia msg plugins enable projx/project-manager
```

Then open the Noctalia launcher and type `/pjx`.

### Installing the DMS plugin

Copy the plugin directory into DMS plugins folder:

```sh
cp -r /path/to/projx/plugins/dms ~/.config/dms/plugins/projxLauncher
dms restart
```

Then open the DMS launcher and type `pjx`.

There is also a bar widget plugin (`plugins/dms-widget/`) showing the current
project name in the bar:

```sh
cp -r /path/to/projx/plugins/dms-widget ~/.config/dms/plugins/projxCurrentProject
dms restart
```

### Shell completion

pjx ships cobra-generated completion scripts:

```sh
pjx completion bash > /etc/bash_completion.d/pjx
pjx completion zsh > ~/.zfunc/_pjx
pjx completion fish > ~/.config/fish/completions/pjx.fish
```

## Development

### Requirements

**Core (Go):**

- Go 1.27+
- `gofmt` (bundled with Go)
- `golangci-lint` v2.x — `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
- `goimports` — `go install golang.org/x/tools/cmd/goimports@latest` (optional, falls back to gofmt)

**QML (DMS plugin):**

- `qmlformat` and `qmllint` — part of Qt 6 declarative dev tools
  - Arch: `pacman -S qt6-declarative`
  - Debian/Ubuntu: `apt install qt6-declarative-dev-tools`

**Notes:**
- The scanner does not follow symlinks by default (avoids cycles). Pass explicit roots if needed.
- Go 1.27+ is required (uses modernc.org/sqlite pure-Go driver).
**Luau (Noctalia plugin):**

- `stylua` v2.x — `cargo install stylua` or download from [releases](https://github.com/JohnnyMorganz/StyLua/releases)
  - Already available if you use nvim mason: `~/.local/share/nvim/mason/bin/stylua`

### Install all tooling

```sh
make install-tools
```

This installs the Go tools (offline-friendly from the module cache) and prints
instructions for any missing non-Go tools.

### Quality gates

```sh
make quality        # fmt-check + lint + vet
make fmt            # format Go + Luau
make fmt-check      # check formatting (Go, Luau, QML)
make lint           # lint Go + QML + Luau
make vet            # go vet
make staticcheck    # static analysis (if installed)
make test           # run tests
```

CI runs the same gates on every push and pull request (see `.github/workflows/ci.yml`).

### Project layout

```
projx/
├── cmd/pjx/              # CLI entry point
├── internal/              # Go packages (config, db, model, discovery, importer, gitinfo, cli, tui)
├── plugins/
│   ├── noctalia/          # Noctalia v5 plugin (Luau)
│   └── dms/               # DMS plugin (QML)
├── scripts/               # install-tools.sh
├── .github/workflows/     # CI
├── Makefile
└── README.md
```

## License

MIT
