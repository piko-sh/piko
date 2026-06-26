# Piko for Zed

Zed editor support for Piko single-file components (`.pk` and `.pkc`):

- **Syntax highlighting** via a Tree-sitter grammar (`tree-sitter-piko`), with
  embedded **Go**, **JavaScript/TypeScript**, **CSS** and **JSON** highlighted
  by their own grammars through injections.
- **Language intelligence** — completion, hover, go-to-definition, diagnostics,
  rename, code actions, document symbols, folding and (optional) formatting —
  provided by the `pikopls` language server, the same server used by the
  VS Code and JetBrains plugins.

## Install

1. Install the language server so it is on your `PATH`:

   ```bash
   make build-lsp && cp bin/lsp/pikopls /usr/local/bin/   # binary is named pikopls
   # or: go install piko.sh/piko/cmd/pikopls@latest
   #     then rename, since Go installs it as 'lsp':
   #     mv "$(go env GOBIN)/lsp" "$(go env GOBIN)/pikopls"
   ```

   If `pikopls` is not found, the extension downloads the matching prebuilt
   binary from [GitHub releases](https://github.com/piko-sh/piko/releases).

2. Install the extension in Zed:
   - Command palette → **zed: install dev extension** → select this directory.

## Configuration

Point the extension at a specific binary or pass arguments via Zed
`settings.json`:

```jsonc
{
  "lsp": {
    "pikopls": {
      "binary": {
        "path": "/absolute/path/to/pikopls",
        "arguments": ["--formatting"]
      }
    }
  }
}
```

`pikopls` defaults to stdio, which is how Zed drives it — no transport flags
are required.

## Layout

| Path                    | Purpose                                             |
| ----------------------- | --------------------------------------------------- |
| `extension.toml`        | Extension manifest (language, grammar, LSP)         |
| `src/piko.rs`           | Rust → WASM module that locates and launches the LSP |
| `languages/piko/`       | `config.toml` and Tree-sitter `*.scm` queries       |
| `tree-sitter-piko/`     | The Piko Tree-sitter grammar (generated parser)     |

See [`SETUP.md`](./SETUP.md) for development and publishing.
