# Piko Makefile
#
# This Makefile provides a unified interface for common development tasks.
# Run `make help` to see available targets.

.DEFAULT_GOAL := help

# Directories
HACK_DIR := ./hack
BIN_DIR := ./bin

# Go settings
GO ?= go
GOFLAGS ?=

##@ General

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Build

.PHONY: build-cli
build-cli: ## Build CLI binary for current platform
	@$(HACK_DIR)/cli/build.sh current $(BIN_DIR)/piko

.PHONY: build-cli-all
build-cli-all: ## Build CLI binaries for all platforms
	@$(HACK_DIR)/cli/build.sh all $(BIN_DIR)/cli

.PHONY: build-lsp
build-lsp: ## Build LSP binary for current platform
	@$(HACK_DIR)/lsp/build.sh current $(BIN_DIR)/lsp

.PHONY: build-lsp-all
build-lsp-all: ## Build LSP binaries for all platforms
	@$(HACK_DIR)/lsp/build.sh all $(BIN_DIR)/lsp

.PHONY: build-wasm
build-wasm: ## Build, optimise, and compress WASM binary
	@$(HACK_DIR)/wasm/build.sh build

.PHONY: build-wasm-clean
build-wasm-clean: ## Clean WASM build artefacts
	@$(HACK_DIR)/wasm/build.sh clean

.PHONY: install-cli
install-cli: ## Build and install CLI binary to PATH
	@$(HACK_DIR)/cli/install.sh

##@ Test

.PHONY: test
test: ## Run Go unit tests (quick, quiet output)
	@$(HACK_DIR)/test/test.sh quick

.PHONY: test-sum
test-sum: ## Run Go unit tests with gotestsum (better output)
	@$(HACK_DIR)/test/test.sh sum

.PHONY: test-go
test-go: ## Run all Go tests
	@$(HACK_DIR)/test/test.sh go

.PHONY: test-integration
test-integration: ## Run all Go tests including integration
	@$(HACK_DIR)/test/test.sh integration

.PHONY: test-frontend
test-frontend: ## Run frontend/core vitest tests
	@$(HACK_DIR)/test/test.sh frontend

.PHONY: test-vscode
test-vscode: ## Run VSCode plugin vitest tests
	@$(HACK_DIR)/test/test.sh vscode

.PHONY: test-idea
test-idea: ## Run IntelliJ plugin gradle tests
	@$(HACK_DIR)/test/test.sh idea

.PHONY: test-all
test-all: ## Run ALL tests - the "is everything OK" check
	@$(HACK_DIR)/test/test.sh all

.PHONY: test-coverage-total
test-coverage-total: ## Show whole-project coverage percentage (quick)
	@$(HACK_DIR)/test/test.sh coverage-total

.PHONY: test-coverage-total-full
test-coverage-total-full: ## Show coverage with integration tests (slow)
	@$(HACK_DIR)/test/test.sh coverage-total-full

.PHONY: test-coverage-internal-total
test-coverage-internal-total: ## Show internal/ packages coverage percentage (quick)
	@$(HACK_DIR)/test/test.sh coverage-internal-total

.PHONY: test-update-badges
test-update-badges: ## Update README.md coverage badges
	@$(HACK_DIR)/test/test.sh update-badges

.PHONY: test-coverage
test-coverage: ## Generate coverage for a package (PKG=internal/ast)
	@$(HACK_DIR)/test/coverage.sh $(PKG)

.PHONY: test-coverage-report
test-coverage-report: ## Generate full coverage report
	@$(HACK_DIR)/test/coverage-report.sh

.PHONY: test-profile
test-profile: ## Profile a package (PKG=internal/ast)
	@$(HACK_DIR)/test/profile.sh $(PKG)

##@ Lint

.PHONY: lint
lint: lint-go lint-scripts lint-licences ## Run all linters

.PHONY: lint-go
lint-go: ## Lint all Go code (all workspace modules)
	@$(HACK_DIR)/lint/go.sh

.PHONY: lint-go-all
lint-go-all: ## Lint all Go code including build-constrained modules
	@$(HACK_DIR)/lint/go.sh --all

.PHONY: lint-scripts
lint-scripts: ## Lint bash scripts with shellcheck
	@$(HACK_DIR)/lint/scripts.sh

.PHONY: lint-licences
lint-licences: ## Check dependency licences for Apache-2.0 compatibility
	@$(HACK_DIR)/lint/licences.sh

##@ Go

.PHONY: go-mod-tidy
go-mod-tidy: ## Tidy all go.mod files
	@$(HACK_DIR)/go/mod-tidy.sh

.PHONY: go-mod-upgrade
go-mod-upgrade: ## Upgrade all direct dependencies across all go.mod files
	@$(HACK_DIR)/go/mod-upgrade.sh

.PHONY: go-mod-upgrade-dry
go-mod-upgrade-dry: ## Report available dependency upgrades without applying them
	@$(HACK_DIR)/go/mod-upgrade.sh --dry-run

.PHONY: go-mod-major
go-mod-major: ## List available major version upgrades across all go.mod files
	@$(HACK_DIR)/go/mod-major.sh

.PHONY: go-mod-verify
go-mod-verify: ## Verify go.mod files are tidy
	@$(HACK_DIR)/go/mod-verify.sh

.PHONY: go-packages-list
go-packages-list: ## List all Go packages (excludes generated/test-only)
	@$(HACK_DIR)/go/packages.sh

##@ Verify

.PHONY: verify-all
verify-all: ## Run all verification checks
	@$(HACK_DIR)/verify/all.sh

##@ Analysis

.PHONY: cloc
CLOC_OPTS := --vcs=git --quiet --exclude-dir=testdata,examples,ast_schema_gen,collection_schema_gen,generator_schema_gen,i18n_schema_gen,inspector_schema_gen,registry_schema_gen,search_schema_gen

cloc: ## Count lines of code
	@echo "=== Source ==="
	@cloc $(CLOC_OPTS) --fmt=2 --not-match-f='(_test\.go|\.spec\.\w+|\.test\.\w+)$$' .
	@echo ""
	@echo "=== Tests ==="
	@cloc $(CLOC_OPTS) --fmt=2 --match-f='(_test\.go|\.spec\.\w+|\.test\.\w+)$$' .
	@echo ""
	@echo "=== Source % ==="
	@cloc $(CLOC_OPTS) --not-match-f='(_test\.go|\.spec\.\w+|\.test\.\w+)$$' --percent .
	@echo ""
	@echo "=== Tests % ==="
	@cloc $(CLOC_OPTS) --match-f='(_test\.go|\.spec\.\w+|\.test\.\w+)$$' --percent .

##@ Tools

.PHONY: tools-download
tools-download: ## Download all required tools (flatc, protoc)
	@$(HACK_DIR)/tools/download.sh

.PHONY: tools-flatc
tools-flatc: ## Download flatc (FlatBuffers compiler)
	@$(HACK_DIR)/tools/flatc.sh

.PHONY: tools-protoc
tools-protoc: ## Download protoc (Protocol Buffers compiler)
	@$(HACK_DIR)/tools/protoc.sh

##@ Generate

.PHONY: generate-all
generate-all: ## Run all code generators (dal, flatc, protoc, qt, interp, asmgen)
	@$(HACK_DIR)/generate/all.sh

.PHONY: generate-dal
generate-dal: ## Generate Go code from SQL using generate_dal
	@$(HACK_DIR)/generate/dal.sh

.PHONY: generate-asmgen
generate-asmgen: ## Generate Plan 9 assembly files for interp and vectormaths
	@$(HACK_DIR)/generate/asmgen.sh

.PHONY: generate-asmgen-validate
generate-asmgen-validate: ## Validate generated assembly files are up to date
	@$(HACK_DIR)/generate/asmgen.sh --validate

.PHONY: generate-flatc
generate-flatc: ## Generate Go code from FlatBuffers schemas
	@$(HACK_DIR)/generate/flatc.sh

.PHONY: generate-protoc
generate-protoc: ## Generate Go code from Protocol Buffers
	@$(HACK_DIR)/generate/protoc.sh

.PHONY: generate-qt
generate-qt: ## Generate Go code from quicktemplate files
	@$(HACK_DIR)/generate/qt.sh

.PHONY: generate-interp-symbols
generate-interp-symbols: ## Generate bytecode interpreter stdlib symbol tables
	@$(HACK_DIR)/generate/interp_symbols.sh

.PHONY: generate-interp-piko-symbols
generate-interp-piko-symbols: ## Generate bytecode interpreter piko runtime symbol tables
	@$(HACK_DIR)/generate/interp_piko_symbols.sh

##@ Import

.PHONY: import-esbuild-update
import-esbuild-update: ## Update vendored esbuild
	@$(HACK_DIR)/import/esbuild-update.sh

##@ Plugins - VSCode

.PHONY: plugin-vscode-build
plugin-vscode-build: ## Build and package VSCode extension
	@$(HACK_DIR)/plugin/vscode.sh build

.PHONY: plugin-vscode-install
plugin-vscode-install: ## Build, package, and install in VSCode
	@$(HACK_DIR)/plugin/vscode.sh install

.PHONY: plugin-vscode-clean
plugin-vscode-clean: ## Clean VSCode extension build artefacts
	@$(HACK_DIR)/plugin/vscode.sh clean

##@ Plugins - IntelliJ

.PHONY: plugin-idea-build
plugin-idea-build: ## Build and package IntelliJ plugin
	@$(HACK_DIR)/plugin/idea.sh build

.PHONY: plugin-idea-install
plugin-idea-install: ## Build, package, and install in IDE
	@$(HACK_DIR)/plugin/idea.sh install

.PHONY: plugin-idea-run
plugin-idea-run: ## Run sandbox IDE with IntelliJ plugin
	@$(HACK_DIR)/plugin/idea.sh run

.PHONY: plugin-idea-clean
plugin-idea-clean: ## Clean IntelliJ plugin build artefacts
	@$(HACK_DIR)/plugin/idea.sh clean

##@ Pre-submit

.PHONY: check
check: ## Run local validation (vet, lint, tests) before pushing
	$(GO) vet -tags "vips integration ffmpeg" piko.sh/piko/...
	@$(MAKE) lint-go-all
	$(GO) test -race -count=1 -shuffle=on piko.sh/piko/... -short

##@ Development

.PHONY: lsp-debug
lsp-debug: ## Start LSP with Delve debugger (PROJECT=path)
	@$(HACK_DIR)/lsp/debug.sh $(PROJECT)

.PHONY: e2e-browser-run
e2e-browser-run: ## Run E2E browser tests interactively
	@$(HACK_DIR)/e2e/browser-run.sh

##@ VM

.PHONY: vm-windows-up
vm-windows-up: ## Start and provision Windows VM
	@$(HACK_DIR)/vm/windows.sh up

.PHONY: vm-windows-halt
vm-windows-halt: ## Stop Windows VM gracefully
	@$(HACK_DIR)/vm/windows.sh halt

.PHONY: vm-windows-destroy
vm-windows-destroy: ## Destroy Windows VM and all state
	@$(HACK_DIR)/vm/windows.sh destroy

.PHONY: vm-windows-status
vm-windows-status: ## Show Windows VM status
	@$(HACK_DIR)/vm/windows.sh status

.PHONY: vm-windows-ssh
vm-windows-ssh: ## Open SSH session to Windows VM
	@$(HACK_DIR)/vm/windows.sh ssh

.PHONY: vm-windows-test
vm-windows-test: ## Cross-compile and run Go tests on Windows VM
	@$(HACK_DIR)/vm/windows.sh test

.PHONY: vm-windows-outlook
vm-windows-outlook: ## Render emails in Outlook and capture screenshots
	@$(HACK_DIR)/vm/windows.sh outlook

##@ Supply Chain

.PHONY: sbom-cli
sbom-cli: build-cli ## Generate SBOM for CLI binary (current platform)
	@$(HACK_DIR)/release/sbom.sh $(BIN_DIR)/piko

.PHONY: sbom-lsp
sbom-lsp: build-lsp ## Generate SBOM for LSP binary (current platform)
	@$(HACK_DIR)/release/sbom.sh $(BIN_DIR)/lsp

##@ Clean

.PHONY: clean
clean: ## Clean build artefacts
	rm -rf $(BIN_DIR)
	rm -f COVERAGE_REPORT.md
	rm -f /tmp/piko-coverage-total.out /tmp/piko-coverage-total.deduped.out /tmp/coverage.out
	rm -f /tmp/piko-lsp-debug
	rm -f /tmp/piko-lsp-*.log /tmp/piko-lsp-*.log.gz
	rm -rf /tmp/chromedp-runner*
