BINARY := pjx
PKG    := github.com/bal16/projx/cmd/pjx

# Install paths (overridable)
PREFIX            ?= $(HOME)/.local
BINDIR            ?= $(PREFIX)/bin
DMS_PLUGINS_DIR   ?= $(HOME)/.config/dms/plugins
DANK_PLUGINS_DIR  ?= $(HOME)/.config/DankMaterialShell/plugins

DMS_LAUNCHER_ID   := projxLauncher
DMS_WIDGET_ID     := projxCurrentProject

# Tool paths (overridable)
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo $(HOME)/go/bin/golangci-lint)
STATICCHECK   ?= $(shell command -v staticcheck 2>/dev/null || echo $(HOME)/go/bin/staticcheck)
STYLUA        ?= $(shell command -v stylua 2>/dev/null || echo $(HOME)/.local/share/nvim/mason/bin/stylua)
QMLFORMAT     ?= $(shell command -v qmlformat 2>/dev/null)
QMMLINT       ?= $(shell command -v qmllint 2>/dev/null)
GOIMPORTS     ?= $(shell command -v goimports 2>/dev/null || echo $(HOME)/go/bin/goimports)

.PHONY: build test vet fmt fmt-check lint quality install-tools clean \
	install install-bin install-dms install-noctalia \
	update update-bin update-dms \
	uninstall uninstall-bin uninstall-dms uninstall-noctalia \
	install-dms-to uninstall-dms-from

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

# --- Install / Update / Uninstall ---
# Binary
install-bin: build
	install -Dm755 bin/$(BINARY) "$(BINDIR)/$(BINARY)"
	@echo "Installed $(BINARY) to $(BINDIR)/$(BINARY)"

update-bin: install-bin

uninstall-bin:
	rm -f "$(BINDIR)/$(BINARY)"
	@echo "Removed $(BINDIR)/$(BINARY)"

# DMS plugins (launcher + bar widget). Installs to DMS_PLUGINS_DIR and, if present,
# also to DANK_PLUGINS_DIR (some setups keep both in sync).
install-dms-to:
	@dir="$(DIR)"; \
	if [ -z "$$dir" ]; then echo "DIR is required"; exit 1; fi; \
	mkdir -p "$$dir"; \
	rm -rf "$$dir/$(DMS_LAUNCHER_ID)" "$$dir/$(DMS_WIDGET_ID)"; \
	cp -r plugins/dms "$$dir/$(DMS_LAUNCHER_ID)"; \
	cp -r plugins/dms-widget "$$dir/$(DMS_WIDGET_ID)"; \
	echo "Installed DMS plugins to $$dir"

uninstall-dms-from:
	@dir="$(DIR)"; \
	if [ -z "$$dir" ]; then echo "DIR is required"; exit 1; fi; \
	rm -rf "$$dir/$(DMS_LAUNCHER_ID)" "$$dir/$(DMS_WIDGET_ID)"; \
	echo "Removed DMS plugins from $$dir"

install-dms:
	@$(MAKE) install-dms-to DIR="$(DMS_PLUGINS_DIR)"
	@if [ -d "$(DANK_PLUGINS_DIR)" ]; then $(MAKE) install-dms-to DIR="$(DANK_PLUGINS_DIR)"; fi
	@echo "Restart DMS to load changes (e.g. dms restart)"

update-dms: install-dms

uninstall-dms:
	@$(MAKE) uninstall-dms-from DIR="$(DMS_PLUGINS_DIR)"
	@if [ -d "$(DANK_PLUGINS_DIR)" ]; then $(MAKE) uninstall-dms-from DIR="$(DANK_PLUGINS_DIR)"; fi
	@echo "Restart DMS to apply removal (e.g. dms restart)"

# Noctalia plugin: the repo's plugins/ directory can be used directly as a path source.
install-noctalia:
	@if ! command -v noctalia >/dev/null 2>&1; then \
		echo "noctalia CLI not found; register manually:"; \
		echo "  noctalia msg plugins source add dev path $(CURDIR)/plugins"; \
		echo "  noctalia msg plugins enable projx/project-manager"; \
		exit 0; \
	fi
	@noctalia msg plugins source add dev path "$(CURDIR)/plugins" 2>/dev/null || true
	@noctalia msg plugins enable projx/project-manager 2>/dev/null || true
	@echo "Noctalia plugin registered from $(CURDIR)/plugins (use /pjx in launcher)"

uninstall-noctalia:
	@if command -v noctalia >/dev/null 2>&1; then \
		noctalia msg plugins disable projx/project-manager 2>/dev/null || true; \
		echo "Disabled projx/project-manager (remove its source manually if desired)"; \
	else \
		echo "noctalia CLI not found; disable/remove the plugin manually"; \
	fi

# Convenience targets
install: install-bin install-dms
	@echo "Done. Run 'make install-noctalia' to register the Noctalia plugin."

update: update-bin update-dms
	@echo "Done. Noctalia uses the repo source directly; pull/rebuild updates it."

uninstall: uninstall-bin uninstall-dms uninstall-noctalia

clean:
	rm -rf bin/
