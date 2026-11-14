# Contributing to projx

Thanks for your interest. projx is a Go CLI with Noctalia/DMS plugins.

## Development setup

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Quick version:

```sh
git clone <repo>
cd project-manager-ext
make install-tools
make build
make quality
```

## Workflow

- Branch from `main`: `git checkout -b feat/my-change`
- Make small, atomic commits following [Conventional Commits](https://www.conventionalcommits.org/):
  `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`, `test:`, `style:`, `ci:`.
- Run `make quality` before pushing (fmt-check + lint + vet).
- Open a PR against `main`.

## Code style

- Go: `gofmt` + `goimports` (local prefix `github.com/bal16/projx`). Enforced by golangci-lint.
- QML: `qmllint` must pass.
- Luau: `stylua` must pass (`stylua --check`).
- No new warnings; keep `golangci-lint` at 0 issues.

## Tests

- Add tests for new logic (repository, scanner, importers, helpers).
- Use temp DB via `XDG_DATA_HOME` env (see `internal/db/repository_test.go`).
- Use `PROJX_ZED_DB` / `PROJX_VSCODE_DB` env overrides for importer tests.

## Plugin changes

- Noctalia: Luau scripts must pass `stylua --check`.
- DMS: QML must pass `qmllint`.
- Follow the real DMS contract (`getItems`/`executeItem`, `{name, comment, action, icon, categories}`).

## Commit / PR checklist

- [ ] `make quality` passes
- [ ] `go test ./...` passes
- [ ] README updated if user-facing change
- [ ] Conventional commit messages
