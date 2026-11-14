# Development

## Requirements

**Core (Go):**
- Go 1.26+
- `gofmt` (bundled with Go)
- `golangci-lint` v2.x
- `goimports` (optional, falls back to gofmt)

**QML (DMS plugin):**
- `qmlformat`, `qmllint` — Qt 6 declarative dev tools
  - Arch: `pacman -S qt6-declarative`
  - Debian/Ubuntu: `apt install qt6-declarative-dev-tools`

**Luau (Noctalia plugin):**
- `stylua` v2.x — `cargo install stylua`

Install all tooling:
```sh
make install-tools
```

## Notes

- The scanner does not follow symlinks by default (avoids cycles).
- Go 1.26+ is required (uses modernc.org/sqlite pure-Go driver).

## Quality gates

```sh
make quality        # fmt-check + lint + vet + test
make fmt            # format Go + Luau
make fmt-check      # check formatting (Go, Luau, QML)
make lint           # lint Go + QML + Luau
make vet            # go vet
make staticcheck    # static analysis (if installed)
make test           # run tests
```

CI runs the same gates on every push/PR (see `.github/workflows/ci.yml`).

## Project layout

```
projx/
├── cmd/pjx/              # CLI entry point
├── internal/              # Go packages
├── plugins/
│   ├── project-manager/   # Noctalia v5 plugin (Luau)
│   └── dms/               # DMS plugin (QML)
├── scripts/               # install-tools.sh
├── .github/workflows/     # CI
├── Makefile
└── README.md
```

## Shell completion

```sh
pjx completion bash > /etc/bash_completion.d/pjx
pjx completion zsh  > ~/.zfunc/_pjx
pjx completion fish > ~/.config/fish/completions/pjx.fish
```

See [docs/USAGE.md#shell-completion](docs/USAGE.md#shell-completion) for
session-only setup and notes (e.g. zsh `fpath`/`compinit`).
