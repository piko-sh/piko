<!--
Copyright 2026 PolitePixels Limited

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This project stands against fascism, authoritarianism, and all forms of
oppression. We built this to empower people, not to enable those who would
strip others of their rights and dignity.
-->

# CLI Reference

Use this guide when running Piko CLI commands for project creation, formatting, code generation, or development.

## Project creation

```bash
# Interactive wizard
piko new

# Creates project structure, runs go mod tidy
```

The wizard prompts for:
1. Project name
2. Destination (new folder or current directory)
3. Go module path
4. Optional features (checklist)

## Formatting

### Format files

```bash
# Format a single file
piko fmt path/to/file.pk

# Format recursively
piko fmt -r .

# Format all .pk files in a directory
piko fmt -r ./pages ./partials
```

### Check mode (CI)

```bash
# Exit code 1 if any files need formatting
piko fmt -check -r .

# List files that need formatting
piko fmt -l -r .
```

### What gets formatted

| Section | Formatter |
|---------|-----------|
| Template (HTML) | Piko formatter (2-space indent, attribute sorting, 100-char wrap) |
| Script (Go) | `go/format` (same as `gofmt`) |
| Style (CSS) | esbuild CSS parser |
| i18n (JSON) | 2-space indentation |

The formatter is idempotent - running it twice produces identical output.

### Whitespace-sensitive elements

`<pre>`, `<code>`, and `<textarea>` preserve content exactly.

### Self-closing tags

Void elements are formatted with self-closing syntax: `<img />`, `<br />`, `<hr />`.

## Code generation

```bash
# Generate all assets (pages, partials, actions, components)
go run ./cmd/generator/main.go all

# Required after adding new pages, partials, or actions
```

## Development server

```bash
# With live reloading (recommended)
air

# Without live reloading
go run ./cmd/main/main.go
```

Default address: `http://localhost:8080`

## LSP server

`pikopls` is a standalone binary, not a `piko` subcommand. Install it with
`go install piko.sh/piko/cmd/pikopls@latest` (the binary is named `pikopls`) or
`piko agents install`, which downloads a checksum-verified release and pins its
absolute path into the Claude Code plugin's `.lsp.json`.

```bash
# Stdio mode (default, for editor integration)
pikopls

# TCP mode (for debugging)
pikopls --tcp --port 4389

# File logging for debugging
pikopls --file-logging
pikopls --log-file=/tmp/pikopls.log

# Go intelligence inside <script type="application/x-go"> via a child gopls
pikopls --gopls-bridge --gopls-path="$(command -v gopls)"
```

### LSP flags

| Flag | Description | Default |
|------|-------------|---------|
| `--version` | Print the pikopls version and exit | |
| `--tcp` | Use TCP instead of stdio | `false` |
| `--host` | TCP host to bind (with `--tcp`) | `127.0.0.1` |
| `--port` | TCP port (with `--tcp`) | `4389` |
| `--max-connections` | Maximum concurrent TCP connections | `64` |
| `--formatting` | Enable document formatting | `false` |
| `--file-logging` | Write debug logs to `$TMPDIR/pikopls-<pid>.log` | `false` |
| `--log-file=<path>` | Write debug logs to this path (implies `--file-logging`) | |
| `--pprof` | Start the pprof HTTP server | `false` |
| `--pprof-port` | pprof port (with `--pprof`) | `6060` |
| `--gopls-bridge` | Delegate Go script blocks to a child gopls by default | `false` |
| `--gopls-path` | Path to gopls (otherwise PATH, then GOBIN, GOPATH/bin, ~/go/bin, ~/.local/bin) | |
| `--gopls-disable` | Hard-disable the gopls bridge, even when a client asks for it | `false` |
| `--gopls-max-children` | Maximum concurrent gopls child processes (0 uses the default) | `8` |
| `--gopls-max-overlays` | Maximum open Go-block overlays per gopls child (0 uses the default) | `1024` |

Clients can also turn the bridge on per connection with the initialisation option
`{"goBridge": true}`; the Claude Code plugin sends it.

### LSP capabilities

| Feature | Description |
|---------|-------------|
| Hover | Type info and documentation |
| Completion | Context-aware suggestions (triggered by `.`, `-`, and `"`) |
| Signature Help | Parameter hints for function calls |
| Diagnostics | Type errors, undefined variables, missing imports |
| Go to Definition / Declaration / Type Definition | Navigate to symbol, declaration, and type definitions |
| Find References | Locate all references to a symbol |
| Document Highlight | Highlight other occurrences of the symbol under the cursor |
| Document and Workspace Symbols | Outline of a file and symbol search across the workspace |
| Document Links | Clickable links to imported files and assets |
| Colours | Colour swatches and picker for CSS colour values |
| Inlay Hints | Inline type and parameter hints |
| Type Hierarchy | Supertypes and subtypes of a type |
| Go to Implementation | Jump to implementations |
| Folding Ranges | Fold template elements and blocks |
| Document Formatting | Format entire document (needs `--formatting`) |
| Quick Fixes | Auto-fix diagnostics (add coerce, add import, etc.) |
| Rename | Rename symbols across workspace |
| Linked Editing | Edit matching HTML tags simultaneously |

With the gopls bridge on, Go script blocks also get full gopls hover, completion,
navigation, rename, code actions, and diagnostics.

## Production build

```bash
go build -o app ./cmd/main
./app prod
```

## Docker

The scaffolded project includes a `Dockerfile`:

```bash
docker build -t my-piko-app .
docker run -p 8080:8080 my-piko-app
```

## Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PIKO_LSP_DRIVER` | LSP transport: `stdio` or `tcp` | `stdio` |
| `PIKO_LSP_TCP_ADDR` | TCP address for LSP | `127.0.0.1:4389` |
| `PIKO_LSP_GOPLS_BRIDGE` | Enable the gopls bridge by default (`1`, `true`, `yes`, `on`) | off |
| `PIKO_GOPLS_PATH` | Path to the gopls executable (same as `--gopls-path`) | discovered |
| `PIKO_LSP_GOPLS_DISABLE` | Hard-disable the gopls bridge (`1`, `true`, `yes`, `on`) | off |
| `PIKO_LSP_GOPLS_MAX_CHILDREN` | Maximum concurrent gopls child processes | `8` |
| `PIKO_LSP_GOPLS_MAX_OVERLAYS` | Maximum open Go-block overlays per gopls child | `1024` |

Command-line flags take precedence over these variables. pikopls inherits the
environment of the editor or Claude Code session that starts it, so export
`PIKO_GOPLS_PATH` before launching the editor when gopls is somewhere unusual.

## LLM mistake checklist

- Forgetting to run `go run ./cmd/generator/main.go all` after adding pages/partials
- Using `piko fmt` without `-r` flag (subdirectories are skipped)
- Forgetting to run `go mod tidy` after changing dependencies

## Related

- `references/project-structure.md` - directory layout and config files
