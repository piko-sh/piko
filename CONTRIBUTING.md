# Contributing to Piko

## Requirements

Go 1.27+. Node.js 22+ if you are touching frontend or plugins. `make help` lists all the targets.

`make check` and `make test-all` need CGO tooling and the libvips and FFmpeg
development libraries. The full test suite also needs Java 21 for the IntelliJ plugin.

---

## Quick start

```bash
git clone https://piko.sh/piko.git
cd piko
go mod download
make test
```

If you are touching FlatBuffers, Protocol Buffers, or quicktemplate schemas, also run `make tools-download` and `make generate-all`.

---

## Code organisation

Piko often splits packages under `internal/` into three directories, called hexagons:

```text
internal/cache/
├── cache_domain/      # Business logic, port interfaces
├── cache_adapters/    # Concrete implementations (Redis, Otter, etc.)
└── cache_dto/         # Data transfer objects
```

Most public-facing packages follow this pattern because it allows extensibility. Domain defines the ports, adapters implement them, DTOs cross boundaries. Packages that could support extensibility should follow this pattern.

The `wdk/` directory is the public API surface. It re-exports internal types via aliases so users get a stable interface without importing `internal/` directly.

Dependency injection happens in `internal/bootstrap/`, using functional options (`WithEventBus(...)`, `WithCacheService(...)`, etc.). The options system exposes these at the root, or through a facade in wdk.

Entry points are in `cmd/piko` (CLI), `cmd/pikopls` (language server), and `cmd/wasm` (WebAssembly).

---

## Things to know

### Safe/unsafe build tags

Performance-sensitive code can use paired files. The default `*_unsafe.go`
implementation uses `unsafe`. The `*_safe.go` implementation is selected by `-tags safe`.

For string/bytes conversions, use the `internal/mem` package (`mem.String()`, `mem.Bytes()`).
Its zero-allocation implementation uses `//go:build !safe && !(js && wasm)`. Its
allocating implementation uses `//go:build safe || (js && wasm)`.

Tests must pass in both modes.

### Code generation

Piko uses FlatBuffers, Protocol Buffers, and quicktemplate. Schemas live in `*_schema/` directories. Run `make generate-all` after pulling changes that touch them.

### Other bits

The `hack/` directory has scripts for building, testing, linting, and code generation. Most Make targets delegate to those scripts.

Framework packages keep OTEL metrics and traces in `otel.go`. Put the licence header and package comment in `doc.go`.

Piko ships a custom seven-level logger. TRACE and INTERNAL are framework-only and stay invisible to end users unless explicitly enabled. DEBUG and INFO target users directly, and the NOTICE, WARN, and ERROR levels serve both framework and user code.

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
golangci-lint run ./internal/yourpkg/...   # Lint one package
make lint-go                               # Lint every workspace module, untagged (what CI runs)
make lint-go-all                           # Also lint every build tag and platform variant
```

Configuration is in `.golangci.yml`. The limits that tend to come up are listed below.

| Rule | Limit |
|------|-------|
| Function length | 60 lines, 90 statements |
| Cyclomatic complexity | 20 |
| Cognitive complexity | 15 |
| Control nesting | 3 levels |
| Function arguments | 7 |
| Line length | 200 characters |

Suppress with `//nolint:linter // reason` when you have a good reason (for example `//nolint:revive // cyclomatic: dispatch table`).

---

## Testing

### Linux prerequisites

Some integration test packages create `fsnotify` watchers that recursively watch directory trees. When `go test ./...` runs packages in parallel, the test suite can exhaust the default kernel inotify limits, causing `too many open files` errors. Check your current limits and raise them if needed:

```bash
# Check current values
cat /proc/sys/fs/inotify/max_user_instances   # default 128 - needs raising
cat /proc/sys/fs/inotify/max_user_watches     # default 8192-65536 - needs raising
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
make test-sum                          # Short tests with coverage and gotestsum output
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

If in doubt, run `make check` for vet, lint, and short Go tests.

### Running fuzz checks

Campaigns need Go 1.27, Linux cgroup v2 with access to a suitable cgroup or user
systemd scope, and `taskset` from util-linux. The launcher builds with `GOPROXY=off`,
so download the Go dependencies before running a campaign from the repository root.

```sh
./hack/fuzz/fuzz.sh doctor -target pages -mode compiled -browser off -n 4
./hack/fuzz/fuzz.sh campaign -target pages -mode compiled -browser off -seed 7 -n 4
```

Run `./hack/fuzz/fuzz.sh help` to list targets, or
`./hack/fuzz/fuzz.sh replay /path/to/finding` to reproduce a saved failure. For
interpreted campaigns, follow the [local Pipit setup](#unreleased-pipit-fixes).

### Extending the fuzz toolkit

Add fixed-seed generation and saved-case replay tests for new inputs. Give independent
assertions stable identifiers and verify that a deliberately wrong result fails the
check. Exercise application targets in each supported server and browser mode.

The fuzz module is part of the repository workspace. Run its tests from the
repository root.

```sh
go test -p 1 ./tests/fuzz/...
```

### Unreleased Pipit fixes

Piko builds against a released Pipit. Tests that require an unreleased fix call
`pipitpending.Skip` with the fix they need. Check out Pipit at `../pipit` and export
`PIKO_PIPIT_LOCAL=1` to run interpreted fuzz campaigns with the local workspace.
The launcher selects that workspace unless `GOWORK` already names an absolute path.
To run the Go tests directly, pass the workspace explicitly.

```sh
export PIKO_PIPIT_LOCAL=1
GOWORK="$(./hack/fuzz/fuzz.sh workspace --local-pipit)" go test -p 1 ./tests/fuzz/...
```

When a new Pipit release ships, update the module requirements, remove the skip
calls for fixes in the release, and rerun the affected tests without
`PIKO_PIPIT_LOCAL`.
