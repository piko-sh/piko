---
title: How to set up the Piko LSP in your editor
description: Install the Piko language server and configure VS Code, JetBrains, Zed, Neovim, Helix, Sublime Text, Claude Code, or other editors.
nav:
  sidebar:
    section: "how-to"
    subsection: "tooling"
    order: 10
---

# How to set up the Piko LSP in your editor

Piko ships a Language Server Protocol (LSP) implementation that provides completion, hover, navigation, references, diagnostics, and more for `.pk` and `.pkc` files. This guide covers installation and editor configuration. The LSP is a standalone binary called `pikopls`. Its source lives at [`cmd/pikopls/`](https://github.com/piko-sh/piko/tree/master/cmd/pikopls), and it is **not** a subcommand of the `piko` CLI. Configure your editor to launch the `pikopls` executable directly. The integration tests under [`tests/integration/lsp/`](https://github.com/piko-sh/piko/tree/master/tests/integration/lsp) exercise its capabilities.

## Install the LSP binary

The VS Code and JetBrains plugins bundle `pikopls`, and the Zed extension downloads it, so you only need a separate install for other editors or to use your own build.

Install with Go. The binary is named `pikopls`, so no renaming is needed:

```bash
go install piko.sh/piko/cmd/pikopls@latest
```

Or let the Piko CLI download a checksum-verified release into `~/.local/bin` (or `/usr/local/bin` when that is on `PATH` and writable):

```bash
piko agents install
```

Or build from source:

```bash
git clone https://github.com/piko-sh/piko
cd piko
make build-lsp
# Produces ./bin/lsp/pikopls for the current platform.
# (Run `make build-lsp-all` to cross-compile every supported target into
# ./bin/lsp/<goos>-<goarch>/pikopls.)
sudo cp bin/lsp/pikopls /usr/local/bin/pikopls
```

Check the install with `pikopls --version`.

## VS Code

The Piko VS Code extension lives under [`plugins/vscode/`](https://github.com/piko-sh/piko/tree/master/plugins/vscode).

```bash
make plugin-vscode-install
```

The extension bundles `pikopls`, registers `.pk` and `.pkc` file associations, and activates syntax colouring. Reload the VS Code window after installation. The `piko.goplsPath`, `piko.goplsDisable`, `piko.goplsMaxChildren`, and `piko.goplsMaxOverlays` settings control the gopls bridge described below.

## JetBrains plugin

The Piko IntelliJ plugin lives under [`plugins/idea/`](https://github.com/piko-sh/piko/tree/master/plugins/idea).

```bash
make plugin-idea-install
```

The plugin injects syntax colouring, registers file types, and wires up the LSP. Restart the IDE after install.

## Zed

The Piko Zed extension lives under [`plugins/zed/`](https://github.com/piko-sh/piko/tree/master/plugins/zed). Install it from the editor:

1. Open Zed.
2. Command palette, then **zed: install dev extension**.
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

Installing the dev extension clones the grammar repository into `plugins/zed/grammars/` (about 300 MB). The directory is git-ignored and safe to delete; Zed clones it again the next time you install the dev extension.

## Claude Code

The repository is a Claude Code plugin marketplace. The `piko` plugin gives Claude the Piko skill and live `pikopls` diagnostics, hover, and navigation for `.pk` and `.pkc` files. Inside Claude Code:

```text
/plugin marketplace add piko-sh/piko
/plugin install piko@piko-sh
```

To try a local checkout instead, run `/plugin marketplace add /path/to/piko`. The marketplace plugin launches `pikopls` from `PATH`, so install the binary first.

Alternatively, `piko agents install` installs the same skill and language-server configuration into `~/.claude/skills/piko/`, downloads `pikopls`, and pins its absolute path into the installed `.lsp.json`, so it works even when the binary's directory is not on the `PATH` that Claude Code inherits.

Both routes enable the gopls bridge through the `goBridge` initialisation option. `pikopls` inherits Claude Code's environment, so export `PIKO_GOPLS_PATH` before starting Claude Code if gopls is not on `PATH` or in a Go bin directory.

## Go intelligence inside `.pk` script blocks (gopls bridge)

For the embedded `<script type="application/x-go">` block, `pikopls` can delegate to **gopls** so you get full Go hover, completion, go-to-definition, signature help, rename, code actions, and diagnostics inside the block, identical to editing a real `.go` file. pikopls runs gopls as a managed child process per Go module and maps positions between the `.pk` file and an in-memory virtual Go document, so nothing is written to disk.

The bridge is **opt-in** and needs gopls (`go install golang.org/x/tools/gopls@latest`). pikopls looks for gopls in the path given by `--gopls-path` or `PIKO_GOPLS_PATH`, then on `PATH`, then in `GOBIN`, `GOPATH/bin`, `~/go/bin`, and `~/.local/bin`. Enable it per editor:

- **Zed**: enabled automatically; the extension passes `--gopls-bridge=true` and `--gopls-path` (from `gopls` on your worktree `PATH`, unless `PIKO_GOPLS_PATH` is set).
- **VS Code** and **Claude Code**: send `initializationOptions: { "goBridge": true }`.
- **Neovim, Helix, Sublime Text, generic LSP clients**: launch pikopls with `--gopls-bridge`, set `PIKO_LSP_GOPLS_BRIDGE=true`, or send `{ "goBridge": true }` as initialisation options.
- **JetBrains**: leave it **off**; IntelliJ's native Go injection handles the block, so the bridge stays disabled by default.

When gopls is not found, the bridge disables itself and all template and state features keep working.

| Flag | Environment variable | Purpose |
|---|---|---|
| `--gopls-bridge` | `PIKO_LSP_GOPLS_BRIDGE` | Turn the bridge on by default for every connection. |
| `--gopls-path=<path>` | `PIKO_GOPLS_PATH` | Use this gopls executable. |
| `--gopls-disable` | `PIKO_LSP_GOPLS_DISABLE` | Hard-disable the bridge, even when a client asks for it. |
| `--gopls-max-children=<n>` | `PIKO_LSP_GOPLS_MAX_CHILDREN` | Maximum concurrent gopls processes (default 8); raise it for large multi-module workspaces. |
| `--gopls-max-overlays=<n>` | `PIKO_LSP_GOPLS_MAX_OVERLAYS` | Maximum open Go-block documents per gopls process (default 1024). |

Flags take precedence over environment variables. Boolean variables accept `1`, `true`, `yes`, or `on`.

## Syntax highlighting outside VS Code, JetBrains, and Zed

`pikopls` does not provide semantic tokens, so colouring comes from the editor. The tree-sitter grammar at [`plugins/zed/tree-sitter-piko/`](https://github.com/piko-sh/piko/tree/master/plugins/zed/tree-sitter-piko) works in any tree-sitter editor, and the Zed queries in [`plugins/zed/languages/piko/`](https://github.com/piko-sh/piko/tree/master/plugins/zed/languages/piko) (`highlights.scm`, `injections.scm`, `indents.scm`) are a starting point for that editor's query files. Some capture names are Zed-specific and may need renaming to your editor's conventions.

## Neovim

Associate the extensions with a filetype and start the server:

```lua
vim.filetype.add({
    extension = {
        pk  = "piko",
        pkc = "piko",
    },
})

vim.api.nvim_create_autocmd("FileType", {
    pattern = { "piko" },
    callback = function(args)
        vim.lsp.start({
            name = "pikopls",
            cmd = { "pikopls", "--gopls-bridge" },
            root_dir = vim.fs.root(args.buf, { "go.mod" }),
        })
    end,
})
```

For highlighting, register the grammar with nvim-treesitter, run `:TSInstall piko`, and copy the query files into `queries/piko/` on your runtime path:

```lua
require("nvim-treesitter.parsers").get_parser_configs().piko = {
    install_info = {
        url = "https://github.com/piko-sh/piko",
        location = "plugins/zed/tree-sitter-piko",
        files = { "src/parser.c", "src/scanner.c" },
        branch = "master",
    },
    filetype = "piko",
}
```

Newer nvim-treesitter releases register custom parsers from a `User TSUpdate` autocommand instead; the `install_info` fields are the same.

## Helix

Add the language, server, and grammar to `~/.config/helix/languages.toml`:

```toml
[[language]]
name = "piko"
scope = "source.piko"
injection-regex = "^piko$"
file-types = ["pk", "pkc"]
roots = ["go.mod"]
language-servers = ["pikopls"]

[language-server.pikopls]
command = "pikopls"
args = ["--gopls-bridge"]

[[grammar]]
name = "piko"
source = { git = "https://github.com/piko-sh/piko", rev = "master", subpath = "plugins/zed/tree-sitter-piko" }
```

Then build the grammar and add queries:

```bash
hx --grammar fetch
hx --grammar build
mkdir -p ~/.config/helix/runtime/queries/piko
cp plugins/zed/languages/piko/{highlights,injections,indents}.scm ~/.config/helix/runtime/queries/piko/
```

Pin `rev` to a commit hash for reproducible builds. Run `hx --health piko` to confirm the server and grammar are found.

## Sublime Text

Install the [LSP](https://packagecontrol.io/packages/LSP) package. Sublime Text has no Piko syntax, so create `Packages/User/Piko.sublime-syntax` that extends HTML (Sublime Text 4):

```yaml
%YAML 1.2
---
name: Piko
scope: text.html.piko
file_extensions: [pk, pkc]
extends: Packages/HTML/HTML.sublime-syntax
contexts: {}
```

Then add a client in **Preferences > Package Settings > LSP > Settings**:

```json
{
  "clients": {
    "pikopls": {
      "enabled": true,
      "command": ["pikopls", "--gopls-bridge"],
      "selector": "text.html.piko"
    }
  }
}
```

## Other editors with a generic LSP client

Any editor supporting LSP 3.16 or later can talk to `pikopls` over stdio. Configure the editor to launch `pikopls` for files with extensions `.pk` and `.pkc`, using the language id `piko` for both. The server announces its capabilities during initialisation and registers its file watchers dynamically, so you do not need to pass extra flags. For debugging, `pikopls --tcp` listens on `127.0.0.1:4389` (`--host` and `--port` override it, `--max-connections` caps concurrent clients).

## Capabilities

The LSP currently provides:

| Capability | Behaviour |
|---|---|
| Completion | Directive names (`p-if`, `p-for`, etc.), state field names from the `Response` type, imported partial aliases, directive-attribute values. |
| Hover | Type information for expressions inside `{{ ... }}`, doc comments from the `Response` struct, partial-prop docs. |
| Go-to-definition, declaration, type definition | Jumps to the `Response` or `Props` struct field, to the imported partial, to actions referenced by `$form` or `action.*`, and to the types behind expressions. |
| References and document highlight | Finds every use of a field, prop, or partial and highlights occurrences in the current file. |
| Diagnostics | Template-expression type errors, missing imports, unknown directive attributes. Each diagnostic's code, such as `T141`, links to its explanation in the [diagnostics reference](../reference/diagnostics.md); editors show it as a clickable code. |
| Rename | Rename a field across struct declaration, template usage, and TS/JS script blocks. |
| Symbols | Document outline and workspace symbol search. |
| Document links and colours | Clickable links to imported files and assets, and colour swatches for CSS values. |
| Inlay hints and type hierarchy | Inline type and parameter hints, and supertypes and subtypes of a type. |
| Code actions and commands | Quick fixes for diagnostics and a command to refresh diagnostics. |
| Formatting | Whole-document, range, and on-type formatting when started with `--formatting`. |

## Troubleshooting

**LSP does not start.** Check that `pikopls` is on `PATH` and executable: run `pikopls --version`. Alternatively launch `pikopls --tcp` in a terminal to verify it starts and listens before pointing the editor at stdio mode.

**Need logs.** Start the server with `--log-file=/tmp/pikopls.log` (or `--file-logging` to write `pikopls-<pid>.log` in the system temp directory).

**Diagnostics are missing.** The LSP needs to know the project root, which is the nearest directory containing `go.mod`. If the editor picks the wrong root, set it explicitly in the editor's LSP config.

**No Go intelligence in script blocks.** Confirm the bridge is enabled for your editor and that gopls is installed; set `--gopls-path` or `PIKO_GOPLS_PATH` when the editor's `PATH` does not include it.

**Completion is slow.** The LSP caches type information per open buffer. First-open for a large project takes a moment, and subsequent edits are fast.

## See also

- [Diagnostics reference](../reference/diagnostics.md) for what each diagnostic code means and how to fix it.
- [CLI reference](../reference/cli.md) for `piko fmt` (formatter accessible from the LSP).
- [Client components reference](../reference/client-components.md) for the PKC file format the LSP understands.
- [PK file format reference](../reference/pk-file-format.md).
