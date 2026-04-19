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

Piko ships a Language Server Protocol (LSP) implementation that provides completion, hover, go-to-definition, diagnostics, and formatting for `.pk` and `.pkc` files. This guide covers installation and editor configuration. The LSP source lives at [`plugins/lsp/`](https://github.com/piko-sh/piko/tree/master/plugins/lsp). The integration tests under [`tests/integration/lsp/`](https://github.com/piko-sh/piko/tree/master/tests/integration/lsp) exercise its capabilities.

## Install the LSP binary

Build from source:

```bash
git clone https://github.com/piko-sh/piko
cd piko
make build-lsp
# Produces ./bin/lsp
```

Copy or symlink the resulting binary somewhere on `PATH`:

```bash
sudo cp bin/lsp /usr/local/bin/piko-lsp
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

## Neovim (with a generic LSP client)

Configure via `lspconfig` or a bare `vim.lsp.start`:

```lua
vim.api.nvim_create_autocmd("FileType", {
    pattern = { "piko", "pkc" },
    callback = function(args)
        vim.lsp.start({
            name = "piko-lsp",
            cmd = { "piko-lsp" },
            root_dir = vim.fs.root(args.buf, { "go.mod", "config.json" }),
            filetypes = { "piko", "pkc" },
        })
    end,
})
```

Associate the extensions with a filetype:

```lua
vim.filetype.add({
    extension = {
        pk  = "piko",
        pkc = "pkc",
    },
})
```

Optionally, install the `piko.vim` syntax colouring plugin for colour support if you find one. Otherwise, fall back to HTML syntax on the template portion.

## Vim / other editors with generic LSP

Any editor supporting the LSP 3.16+ protocol can talk to `piko-lsp` over stdio. Configure the editor to launch `piko-lsp` for files with extensions `.pk` and `.pkc`. The server announces its capabilities during initialisation, and you do not need to pass extra flags.

## Capabilities

The LSP currently provides:

| Capability | Behaviour |
|---|---|
| Completion | Directive names (`p-if`, `p-for`, etc.), state field names from the `Response` type, imported partial aliases, directive-attribute values. |
| Hover | Type information for expressions inside `{{ ... }}`, doc comments from the `Response` struct, partial-prop docs. |
| Go-to-definition | Jumps to the `Response` or `Props` struct field, to the imported partial, to actions referenced by `$form` or `action.*`. |
| Diagnostics | Template-expression type errors, missing imports, unknown directive attributes. |
| Formatting | `piko fmt`-equivalent formatting of the template block. |
| Rename | Rename a field across struct declaration, template usage, and TS/JS script blocks. |

## Troubleshooting

**LSP does not start.** Check that `piko-lsp` is on `PATH` and executable. Run `piko-lsp --version` manually to confirm.

**Diagnostics are missing.** The LSP needs to know the project root, which is the nearest directory with `go.mod` or `config.json`. If the editor picks the wrong root, set it explicitly in the editor's LSP config.

**Completion is slow.** The LSP caches type information per open buffer. First-open for a large project takes a moment, and subsequent edits are fast.

## See also

- [CLI reference](../reference/cli.md) for `piko fmt` (formatter accessible from the LSP).
- [Client components reference](../reference/client-components.md) for the PKC file format the LSP understands.
- [PK file format reference](../reference/pk-file-format.md).
