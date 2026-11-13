#!/usr/bin/env bash
# Install development tooling for projx.
# Go-based tools are installed via `go install` (offline-friendly from module cache).
# Non-Go tools (qmlformat, qmllint, stylua) are checked and install instructions printed.

set -euo pipefail

GOBIN="${GOBIN:-$(go env GOPATH)/bin}"
export GOFLAGS="${GOFLAGS:--mod=mod}"
export GOPROXY="${GOPROXY:-off}"

echo "==> Installing Go tools into $GOBIN"
go install golang.org/x/tools/cmd/goimports@v0.45.0 || echo "  (goimports not available offline; using gofmt only)"
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || echo "  (golangci-lint already installed or unavailable)"
go install honnef.co/go/tools/cmd/staticcheck@latest 2>/dev/null || echo "  (staticcheck already installed or unavailable)"

echo ""
echo "==> Checking non-Go tools"

check() {
  local name=$1; local cmd=$2; local hint=$3
  if command -v "$cmd" >/dev/null 2>&1; then
    echo "  [ok] $name: $(command -v $cmd)"
  else
    echo "  [missing] $name"
    echo "    install: $hint"
  fi
}

check "qmlformat" "qmlformat" "pacman -S qt6-declarative  (or apt install qt6-declarative-dev-tools)"
check "qmllint"   "qmllint"   "pacman -S qt6-declarative  (or apt install qt6-declarative-dev-tools)"
check "stylua"    "stylua"    "cargo install stylua  OR  download from https://github.com/JohnnyMorganz/StyLua/releases"

# stylua via nvim mason fallback
if ! command -v stylua >/dev/null 2>&1 && [ -x "$HOME/.local/share/nvim/mason/bin/stylua" ]; then
  echo "  [ok] stylua (via nvim mason): $HOME/.local/share/nvim/mason/bin/stylua"
fi

echo ""
echo "Done. Add $GOBIN to your PATH if not already."
