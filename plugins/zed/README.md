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

# Piko for Zed

Zed editor support for Piko single-file components (`.pk` and `.pkc`):

- **Syntax highlighting** via a Tree-sitter grammar (`tree-sitter-piko`), with
  embedded **Go**, **JavaScript/TypeScript**, **CSS/SCSS** and **JSON/YAML**
  highlighted by their own grammars through injections. Template
  interpolations (`{{ }}`) and directive values use JavaScript highlighting.
- **Outline, indentation and bracket matching** for the top-level blocks and
  for `piko:partial` / `piko:slot` elements.
- **Language intelligence**: completion, hover, go-to-definition, diagnostics,
  rename, code actions, document symbols, folding and (optional) formatting,
  provided by the `pikopls` language server, the same server used by the
  VS Code and JetBrains plugins.

## Install

1. Install the language server so it is on your `PATH`:

   ```bash
   make build-lsp && cp bin/lsp/pikopls /usr/local/bin/
   # or: go install piko.sh/piko/cmd/pikopls@latest
   ```

   If `pikopls` is not found, the extension downloads the matching prebuilt
   binary from [GitHub releases](https://github.com/piko-sh/piko/releases),
   preferring the latest stable release and falling back to a pre-release.
   When the release lookup fails (for example offline), a previously
   downloaded `pikopls` is reused.

2. Install the extension in Zed:
   - Command palette, then **zed: install dev extension**, then select this directory.

## Configuration

Point the extension at a specific binary, pass arguments or set environment
variables via Zed `settings.json`:

```jsonc
{
  "lsp": {
    "pikopls": {
      "binary": {
        "path": "/absolute/path/to/pikopls",
        "arguments": ["--formatting"],
        "env": { "PIKO_GOPLS_PATH": "/absolute/path/to/gopls" }
      }
    }
  }
}
```

Variables in `binary.env` are merged over the shell environment Zed captures
for the worktree. When `PIKO_GOPLS_PATH` is set there, the extension does not
pass its own `--gopls-path`, so the variable takes effect.

`pikopls` defaults to stdio, which is how Zed drives it, so no transport flags
are required.

## Go intelligence (gopls bridge)

Go code in `<script type="application/x-go">` blocks gets completion, hover,
definitions and diagnostics from `gopls`, which `pikopls` runs as a child
process. The extension enables this by passing `--gopls-bridge=true` and, when
`gopls` is on the worktree `PATH`, `--gopls-path=<path>`. Zed often does not
see `$HOME/go/bin`, so install gopls somewhere on `PATH` or set the path
yourself:

```bash
go install golang.org/x/tools/gopls@latest
```

To control the bridge, add the flags to `binary.arguments` (one or two leading
dashes both work):

| Argument                 | Effect                                                    |
| ------------------------ | --------------------------------------------------------- |
| `--gopls-bridge=false`   | Turn the bridge off; the extension adds nothing else.     |
| `--gopls-disable`        | Hard-disable the bridge, whatever the client requests.    |
| `--gopls-path=/path`     | Use this gopls instead of the one found on `PATH`.        |
| `--gopls-max-children=N` | Cap concurrent gopls processes (large workspaces).        |
| `--gopls-max-overlays=N` | Cap open Go-block overlays per gopls process.             |

The matching environment variables (`PIKO_LSP_GOPLS_DISABLE`,
`PIKO_GOPLS_PATH`, `PIKO_LSP_GOPLS_MAX_CHILDREN`, `PIKO_LSP_GOPLS_MAX_OVERLAYS`)
can be set in `binary.env` instead.

Without gopls, `pikopls` still provides all template, directive and Piko
intelligence; only the Go-specific features are unavailable.

## Layout

| Path                    | Purpose                                             |
| ----------------------- | --------------------------------------------------- |
| `extension.toml`        | Extension manifest (language, grammar, LSP)         |
| `src/piko.rs`           | Rust to WASM module that locates and launches the LSP |
| `languages/piko/`       | `config.toml` and Tree-sitter `*.scm` queries       |
| `tree-sitter-piko/`     | The Piko Tree-sitter grammar (generated parser)     |

## Grammar development

The grammar lives in `tree-sitter-piko/`. After editing `grammar.js` or
`src/scanner.c`, regenerate and test it with tree-sitter CLI 0.24.7 so the
generated files stay reproducible:

```bash
cd tree-sitter-piko
npx tree-sitter-cli@0.24.7 generate
npx tree-sitter-cli@0.24.7 test
```

Zed fetches the grammar from the git commit pinned in `extension.toml`, so
commit the grammar change and then re-pin it with
`make plugin-zed-sync-commit` before reinstalling the dev extension. The
`grammars/` directory is Zed's local clone of that commit and is safe to
delete.
