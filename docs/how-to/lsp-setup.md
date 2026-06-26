---
title: How to set up the Piko LSP in your editor
description: Install the Piko language server and configure VS Code, JetBrains, Neovim, or other editors.
nav:
  sidebar:
    section: "how-to"
    subsection: "tooling"
    order: 10
---

# How to set up the Piko LSP in your editor

Piko ships a Language Server Protocol (LSP) implementation that provides completion, hover, go-to-definition, and diagnostics for `.pk` and `.pkc` files. This guide covers installation and editor configuration. The LSP is a standalone binary called `pikopls`. Its source lives at [`cmd/pikopls/`](https://github.com/piko-sh/piko/tree/master/cmd/pikopls), it builds with `make build-lsp`, and it is **not** a subcommand of the `piko` CLI. Configure your editor to launch the `pikopls` executable directly. The integration tests under [`tests/integration/lsp/`](https://github.com/piko-sh/piko/tree/master/tests/integration/lsp) exercise its capabilities.

## Install the LSP binary

Build from source:

```bash
git clone https://github.com/piko-sh/piko
cd piko
make build-lsp
# Produces ./bin/lsp/pikopls for the current platform.
# (Run `make build-lsp-all` to cross-compile every supported target into
# ./bin/lsp/<goos>-<goarch>/pikopls.)
```

Copy or symlink the resulting binary somewhere on `PATH`:

```bash
sudo cp bin/lsp/pikopls /usr/local/bin/pikopls
```

## VS Code

The Piko VS Code extension lives under [`plugins/vscode/`](https://github.com/piko-sh/piko/tree/master/plugins/vscode).

```bash
make plugin-vscode-install
```

The extension installs the LSP, registers `.pk` and `.pkc` file associations, and activates syntax colouring. Reload the VS Code window after installation.

## JetBrains plugin

The Piko IntelliJ plugin lives under [`plugins/idea/`](https://github.com/piko-sh/piko/tree/master/plugins/idea).

```bash
make plugin-idea-install
```

The plugin injects syntax colouring, registers file types, and wires up the LSP. Restart the IDE after install.

## Zed

The Piko Zed extension lives under [`plugins/zed/`](https://github.com/piko-sh/piko/tree/master/plugins/zed).

Install `pikopls` first so it is on `PATH` under that exact name. `make build-lsp` produces `./bin/lsp/pikopls`; copy it onto `PATH` (`cp bin/lsp/pikopls /usr/local/bin/`). You can also `go install piko.sh/piko/cmd/pikopls@latest`, but Go names the binary after the package's last path segment (`lsp`), so rename it: `mv "$(go env GOBIN)/lsp" "$(go env GOBIN)/pikopls"` (falling back to `"$(go env GOPATH)/bin"` when `GOBIN` is empty). Then install the extension from the editor:

1. Open Zed.
2. Command palette → **zed: install dev extension**.
3. Select the `plugins/zed/` directory.

Zed resolves the language server in order: an explicit `lsp.pikopls.binary.path` setting, then `pikopls` on `PATH`, then a download of the matching `pikopls` asset from [GitHub releases](https://github.com/piko-sh/piko/releases). To point at a custom build, add to your Zed `settings.json`:

```jsonc
{
  "lsp": {
    "pikopls": {
      "binary": { "path": "/absolute/path/to/pikopls" }
    }
  }
}
```

## Go intelligence inside `.pk` script blocks (gopls bridge)

For the embedded `<script type="application/x-go">` block, `pikopls` can delegate to **gopls** so you get full Go hover, completion, go-to-definition, signature help, rename, code actions, and diagnostics inside the block, identical to editing a real `.go` file. pikopls runs gopls as a managed child process per Go module and maps positions between the `.pk` file and an in-memory virtual Go document, so nothing is written to disk.

This **bridge is opt-in** and requires `gopls` on `PATH` (`go install golang.org/x/tools/gopls@latest`). Enable it per editor:

- **Zed** - enabled automatically; the extension passes `--gopls-bridge=true` and `--gopls-path` (from `gopls` on your worktree PATH).
- **Neovim / Helix / generic LSP** - launch pikopls with `--gopls-bridge=true`, or set `PIKO_LSP_GOPLS_BRIDGE=true` in the environment. Optionally pass `--gopls-path=/path/to/gopls`.
- **VS Code** - sends `initializationOptions: { "goBridge": true }`.
- **JetBrains** - leave it **off**; IntelliJ's native Go injection handles the block, so the bridge stays disabled by default.

When `gopls` is not found, the bridge disables itself and all template/state features keep working; the Go block stays tree-sitter highlighted.

## Neovim (with a generic LSP client)

Configure via `lspconfig` or a bare `vim.lsp.start`:

```lua
vim.api.nvim_create_autocmd("FileType", {
    pattern = { "piko" },
    callback = function(args)
        vim.lsp.start({
            name = "pikopls",
            cmd = { "pikopls" },
            root_dir = vim.fs.root(args.buf, { "go.mod" }),
            filetypes = { "piko" },
        })
    end,
})
```

Associate the extensions with a filetype:

```lua
vim.filetype.add({
    extension = {
        pk  = "piko",
        pkc = "piko",
    },
})
```

Piko does not currently ship a Vim or Neovim syntax-colouring plugin. The LSP supplies semantic tokens through the standard protocol, which most modern Neovim setups render directly. For older clients without semantic-token support, fall back to HTML syntax on the template portion.

## Vim / other editors with generic LSP

Any editor supporting the LSP 3.16+ protocol can talk to `pikopls` over stdio. Configure the editor to launch `pikopls` for files with extensions `.pk` and `.pkc`. The server announces its capabilities during initialisation, and you do not need to pass extra flags.

## Capabilities

The LSP currently provides:

| Capability | Behaviour |
|---|---|
| Completion | Directive names (`p-if`, `p-for`, etc.), state field names from the `Response` type, imported partial aliases, directive-attribute values. |
| Hover | Type information for expressions inside `{{ ... }}`, doc comments from the `Response` struct, partial-prop docs. |
| Go-to-definition | Jumps to the `Response` or `Props` struct field, to the imported partial, to actions referenced by `$form` or `action.*`. |
| Diagnostics | Template-expression type errors, missing imports, unknown directive attributes. |
| Rename | Rename a field across struct declaration, template usage, and TS/JS script blocks. |

## Troubleshooting

**LSP does not start.** Check that `pikopls` is on `PATH` and executable. Run `which pikopls` and confirm the binary file is executable. Alternatively launch `pikopls --tcp` in a terminal (the default port is `4389`, override it with `--port 9999`) to verify it starts and listens before pointing the editor at stdio mode.

**Diagnostics are missing.** The LSP needs to know the project root, which is the nearest directory containing `go.mod`. If the editor picks the wrong root, set it explicitly in the editor's LSP config.

**Completion is slow.** The LSP caches type information per open buffer. First-open for a large project takes a moment, and subsequent edits are fast.

## See also

- [CLI reference](../reference/cli.md) for `piko fmt` (formatter accessible from the LSP).
- [Client components reference](../reference/client-components.md) for the PKC file format the LSP understands.
- [PK file format reference](../reference/pk-file-format.md).
