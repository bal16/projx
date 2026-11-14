BINARY := pjx
PKG    := github.com/bal16/projx/cmd/pjx

# Tool paths (overridable)
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)
STATICCHECK   ?= $(shell command -v staticcheck 2>/dev/null || echo $(HOME)/go/bin/staticcheck)
STYLUA        ?= $(shell command -v stylua 2>/dev/null || echo $(HOME)/.local/share/nvim/mason/bin/stylua)
QMLFORMAT     ?= $(shell command -v qmlformat 2>/dev/null)
QMMLINT       ?= $(shell command -v qmllint 2>/dev/null)
GOIMPORTS     ?= $(shell command -v goimports 2>/dev/null || echo $(HOME)/go/bin/goimports)

.PHONY: build test vet fmt fmt-check lint quality install-tools clean

# --- Build ---
build:
	go build -o bin/$(BINARY) ./cmd/pjx

test:
	go test ./...

vet:
	go vet ./...

# --- Formatting ---
fmt: fmt-go fmt-luau
fmt-go:
	gofmt -w .
	@if command -v $(GOIMPORTS) >/dev/null 2>&1; then $(GOIMPORTS) -w .; else echo "goimports not found; skipping"; fi
fmt-qml:
	@if command -v $(QMLFORMAT) >/dev/null 2>&1; then \
		for f in plugins/dms/*.qml; do $(QMLFORMAT) -w $$f; done; \
	else echo "qmlformat not found; skipping"; fi
fmt-luau:
	@if [ -x "$(STYLUA)" ]; then $(STYLUA) plugins/project-manager/; else echo "stylua not found; skipping"; fi

fmt-check: fmt-go-check fmt-luau-check fmt-qml-check
fmt-go-check:
	@test -z "$$(gofmt -l .)" || { echo "gofmt issues:"; gofmt -l .; exit 1; }
fmt-luau-check:
	@if [ -x "$(STYLUA)" ]; then $(STYLUA) --check plugins/project-manager/; else echo "stylua not found; skipping"; fi
fmt-qml-check:
	@if command -v $(QMMLINT) >/dev/null 2>&1; then $(QMMLINT) plugins/dms/*.qml || echo "qmllint warnings (DMS components may be unresolved locally)"; else echo "qmllint not found; skipping"; fi

# --- Linting ---
lint: lint-go lint-qml lint-luau
lint-go:
	$(GOLANGCI_LINT) run ./...
lint-qml:
	@if command -v $(QMMLINT) >/dev/null 2>&1; then $(QMMLINT) plugins/dms/*.qml || echo "qmllint warnings (DMS components may be unresolved locally)"; else echo "qmllint not found; skipping"; fi
lint-luau:
	@if [ -x "$(STYLUA)" ]; then $(STYLUA) --check plugins/project-manager/; else echo "stylua not found; skipping"; fi

# --- Static analysis ---
staticcheck:
	@if command -v $(STATICCHECK) >/dev/null 2>&1; then $(STATICCHECK) ./...; else echo "staticcheck not found; skipping"; fi

# --- All quality gates ---
quality: fmt-check lint vet test
	@echo "All quality checks passed."

# --- Tooling install ---
install-tools:
	bash scripts/install-tools.sh

clean:
	rm -rf bin/
