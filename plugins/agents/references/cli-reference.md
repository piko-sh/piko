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
go run ./cmd/main
```

Default address: `http://localhost:8080`

## LSP server

```bash
# Stdio mode (default, for editor integration)
go run ./cmd/lsp

# With formatting enabled
go run ./cmd/lsp --formatting

# TCP mode (for debugging)
go run ./cmd/lsp --tcp --port 4389

# File logging for debugging
go run ./cmd/lsp --file-logging
```

### LSP capabilities

| Feature | Description |
|---------|-------------|
| Hover | Type info and documentation |
| Completion | Context-aware suggestions (triggered by `.`, `-`, and `"`) |
| Diagnostics | Type errors, undefined variables, missing imports |
| Go to Definition | Navigate to symbol definitions |
| Find References | Locate all references |
| Document Formatting | Format entire document |
| Quick Fixes | Auto-fix diagnostics (add coerce, add import, etc.) |
| Rename | Rename symbols across workspace |
| Linked Editing | Edit matching HTML tags simultaneously |

## Production build

```bash
go build -o app ./cmd/main
./app
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

## LLM mistake checklist

- Forgetting to run `go run ./cmd/generator/main.go all` after adding pages/partials
- Using `piko fmt` without `-r` flag (only formats a single file)
- Forgetting to run `go mod tidy` after changing dependencies

## Related

- `references/project-structure.md` - directory layout and config files
