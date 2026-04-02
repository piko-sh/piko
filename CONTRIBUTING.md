# Contributing to Piko

## Requirements

Go 1.26+. Node.js 22+ if you're touching frontend or plugins. `make help` lists all the targets.

---

## Quick start

```bash
git clone https://piko.sh/piko.git
cd piko
go mod tidy
make test
```

If you're touching FlatBuffers, Protocol Buffers, or quicktemplate schemas, also run `make tools-download` and `make generate-all`.

---

## Code organisation

Packages under `internal/` are often split into three directories, these are called hexagons:

```text
internal/cache/
├── cache_domain/      # Business logic, port interfaces
├── cache_adapters/    # Concrete implementations (Redis, Otter, etc.)
└── cache_dto/         # Data transfer objects
```

Quite a lot of packages follow this pattern, it allows extensibility. Domain defines the ports, adapters implement them, DTOs cross boundaries. Packages which could support extensibility should follow this pattern.

The `wdk/` directory is the public API surface. It re-exports internal types via aliases so users get a stable interface without importing `internal/` directly.

Dependency injection happens in `internal/bootstrap/`, using functional options (`WithEventBus(...)`, `WithCacheService(...)`, etc.). These are exposed through the options system, at the root, or in a facade in wdk.

Entry points are in `cmd/`: `cli/` (wizard, formatter), `lsp/` (language server), `wasm/` (WebAssembly target).

---

## Things to know

### Safe/unsafe build tags

Performance-sensitive code has paired files: `*_unsafe.go` (the default, zero-alloc via `unsafe`) and `*_safe.go` (opt-in with `-tags safe`). The build tags are `//go:build !safe && !(js && wasm)` and `//go:build safe || (js && wasm)`.

For string/bytes conversions, use the `internal/mem` package (`mem.String()`, `mem.Bytes()`). 

Tests must pass in both modes.

### Code generation

We use FlatBuffers, Protocol Buffers, and quicktemplate. Schemas live in `*_schema/` directories. Run `make generate-all` after pulling changes that touch them.

### Other bits

The `hack/` directory has scripts for building, testing, linting, and code generation. All Make targets are implemented there.

Each package defines its OTEL metrics and traces in a dedicated `otel.go` file. Logger and meter are package-level vars in that file. Every package also has a `doc.go` with the licence header and a package comment.

We have a custom 7-level logger: TRACE and INTERNAL are framework-only (invisible to end users unless explicitly enabled), DEBUG and INFO are user-facing, and NOTICE/WARN/ERROR are shared.

---

## Conventions

- Minimise depth of nesting, prefer guard clauses.
- Prefer verbose code, no complicated boolean expressions.
- Piko is exclusively written in British English.
- Always use field initialisation for structs (`Foo{bar: 1}`, not `Foo{1}`).
- Table-driven tests, with field initialisation for all test cases.
- No `goto`.

---

## Licence header

Every source code file needs this header.

```go
// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.
```

The anti-fascism statement is mandatory and non-negotiable.

---

## Linting

```bash
make lint-go PKG=./internal/yourpkg   # Lint one package
make lint-go-all                      # Lint everything
```

Configuration is in `.golangci.yml` and `revive.toml`. The limits that tend to come up:

| Rule | Limit |
|------|-------|
| Function length | 80 lines, 60 statements |
| Cyclomatic complexity | 20 |
| Cognitive complexity | 15 |
| Control nesting | 3 levels |
| Function arguments | 7 |
| Line length | 200 characters |

Suppress with `//nolint:linter // reason` when you have a good reason (e.g. `//nolint:revive // cyclomatic: dispatch table`).

---

## Testing

### Linux prerequisites

Several integration test packages create `fsnotify` watchers that recursively watch directory trees. When `go test ./...` runs many packages in parallel the default kernel inotify limits can be exhausted, causing `too many open files` errors. Check your current limits and raise them if needed:

```bash
# Check current values
cat /proc/sys/fs/inotify/max_user_instances   # default 128 - needs raising
cat /proc/sys/fs/inotify/max_user_watches     # default 8192–65536 - needs raising
cat /proc/sys/fs/inotify/max_queued_events    # default 16384 - usually fine

# Temporary (resets on reboot)
sudo sysctl fs.inotify.max_user_instances=1024
sudo sysctl fs.inotify.max_user_watches=524288

# Permanent
cat <<'EOF' | sudo tee /etc/sysctl.d/90-inotify.conf
fs.inotify.max_user_instances = 1024
fs.inotify.max_user_watches = 524288
EOF
sudo sysctl --system
```

### Running tests

```bash
make test                              # Quick unit tests (-short, skips integration)
make test-sum                          # Same thing, gotestsum output
go test -tags safe -short piko.sh/piko/...        # Safe build mode (test both!)
make test-all                          # Full suite, run this before submitting a PR
```

The default `make test` (or `go test -race -short ./...`) is the recommended way to run tests during development. It skips stress tests and integration suites, keeping runs fast.

When running without `-short`, some packages (particularly the annotator test suites) spawn heavyweight Go toolchain subprocesses per testcase. With Go's default parallelism (`-p = GOMAXPROCS`), this can consume a large amount of memory. Use `-p 4` (or lower) to throttle concurrent package execution:

```bash
go test piko.sh/piko/... -race -count=1 -p 4 -timeout=60m
```

Integration tests require the `integration` build tag and are heavy. Run them one package at a time:

```bash
go test -tags integration -v ./tests/integration/storage/...
```

Frontend and plugin tests: `make test-frontend`, `make test-vscode`, `make test-idea`.

If in doubt run `make check`, its a good once over of the go code.