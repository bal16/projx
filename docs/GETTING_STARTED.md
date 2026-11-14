# Getting started

## Install

```sh
go install github.com/bal16/projx/cmd/pjx@latest
# or build locally
make build
```

Requirements: Go 1.27+. See [docs/DEVELOPMENT.md](DEVELOPMENT.md) for tooling.

## First steps

```sh
# 1. Auto-discover git repositories
pjx scan ~/projects ~/workspace --max-depth 2

# 2. Or migrate from an existing editor (one-time)
pjx import zed
pjx import --all

# 3. Add a single project manually
pjx add ~/dotfiles --name "Dotfiles" --tag Personal --favorite

# 4. List
pjx list
pjx list --tag Work

# 5. Open
pjx open my-app
pjx open my-app --new-window

# 6. Interactive picker
pjx tui
```

## Next

- Full command reference: [docs/USAGE.md](USAGE.md)
- Configuration & data locations: [docs/CONFIGURATION.md](CONFIGURATION.md)
- Plugins: [docs/PLUGINS.md](PLUGINS.md)