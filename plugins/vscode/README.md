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

# Piko Language Support for VS Code

Official VS Code extension for Piko single-file components: pages and partials (`.pk`) and client components (`.pkc`).

## Features

The extension bundles the `pikopls` language server for Linux, macOS and Windows (x64 and arm64) and starts it automatically when you open a `.pk` or `.pkc` file.

### `<template>` blocks (pikopls)

- Hover, completion, go to definition, find references and document highlights for template expressions and directives (`p-if`, `p-for`, `:attr`, `@event`, ...)
- Diagnostics, code actions, rename and linked editing
- Document symbols (outline and breadcrumbs) and folding
- Optional formatting (`piko.enableFormatting`)
- Syntax highlighting, including nested `<template p-for>` elements, attributed templates (`<template name="...">`, `<template public>`) and `<piko:*>` elements

### Go `<script>` blocks (pikopls with gopls)

Go script blocks are recognised by `type="application/x-go"` or `type="application/go"`, or by `lang="go"`, `lang="golang"`, `lang="application/x-go"` or `lang="application/go"`, in any attribute order.

pikopls forwards Go blocks to a `gopls` child process (the gopls bridge), so you get Go completion, hover, definition and diagnostics inside the `.pk` file. Install gopls with:

```bash
go install golang.org/x/tools/gopls@latest
```

pikopls looks for gopls on `PATH` and in the Go bin directories. Use `piko.goplsPath` to point at a specific binary, or `piko.goplsDisable` to turn the bridge off.

### TypeScript `<script>` blocks

Scripts with `lang="ts"`, `lang="typescript"` or `type="application/typescript"` (the usual form in `.pkc` components) are served by VS Code's built-in TypeScript support through a virtual document. It includes the Piko client types (`dist/ts/piko-ide.d.ts` and `dist/ts/piko-actions.d.ts`) and a per-file `pk` (or `pkc`) declaration built from the template's `p-ref` attributes. Hover, completion, signature help and go to definition are available.

If the type files are missing, the extension writes fallback copies into `dist/ts/` of the project, but only when that project is inside an open workspace folder. They are replaced when `piko run` generates the real files.

### `<style>` and `<i18n>` blocks

These blocks get syntax highlighting (CSS, and JSON or YAML for `<i18n lang="yaml">`), bracket matching and comment toggling in the block's own syntax. They do not get CSS or JSON language-server features such as completion or validation.

### Comment toggling

`Ctrl+/` and `Ctrl+Shift+/` (`Cmd` on macOS) use `<!-- -->` in template markup and `/* */` inside template expressions. In script, style and i18n blocks they use the embedded language's comment syntax.

### New file command

**Piko: New Piko File** creates a page (`.pk`), a partial (`.pk`) or a client component (`.pkc`) from a template. It uses the folder of the active file, falls back to the workspace root, and asks before overwriting an existing file.

## Requirements

- VS Code 1.107 or later
- A Go toolchain that satisfies your project's `go.mod` (`go version`). If you see "go.mod requires go >= X.Y.Z", upgrade Go or set `GOTOOLCHAIN=auto`.
- gopls, for Go intelligence in script blocks

The extension finds `go` from `piko.goBinPath`, then the Go extension's settings (`go.alternateTools`, `go.goroot`, `GOROOT`), then `PATH`, then common install locations (`/usr/local/go/bin`, Homebrew, `~/sdk/go*`, newest first).

## Settings

| Setting | Default | Description |
| --- | --- | --- |
| `piko.lspPath` | `""` | Custom `pikopls` binary. Empty uses the bundled binary. Supports `~`, `${userHome}` and `${workspaceFolder}`; a bare name such as `pikopls` is looked up on `PATH`; relative paths resolve against the first workspace folder. |
| `piko.transport` | `"stdio"` | `stdio`, or `tcp` to start pikopls on a free local port limited to one connection (debugging only). |
| `piko.trace.server` | `"off"` | Trace LSP messages (`off`, `messages`, `verbose`) to the "Piko Language Server" output channel. |
| `piko.enableTemplateSupport` | `true` | Start pikopls. When off, the extension still provides highlighting, comment toggling, outline and folding. |
| `piko.enableFormatting` | `false` | Enable document formatting from pikopls. |
| `piko.enableFileLogging` | `false` | Write pikopls debug logs to `pikopls.log` in the extension's log directory. Open it with **Piko: Open LSP Logs**. |
| `piko.enablePprof` | `false` | Start the pprof server at `http://localhost:<port>/_piko/debug/pprof/`. |
| `piko.pprofPort` | `6060` | Port for the pprof server, from 1 to 65535. |
| `piko.useExternalServer` | `false` | Connect to an already-running pikopls over TCP instead of starting one. |
| `piko.externalServerPort` | `4389` | Port for `piko.useExternalServer`, from 1 to 65535. |
| `piko.goBinPath` | `""` | Directory containing the `go` binary. Empty means auto-detect. |
| `piko.detectGoFromExtension` | `true` | Use the Go extension's `go.alternateTools` and `go.goroot` settings. |
| `piko.searchGlobalGoLocations` | `true` | When `go` is not on `PATH`, search common install locations. |
| `piko.goplsPath` | `""` | gopls executable for the gopls bridge. Empty lets pikopls find it. |
| `piko.goplsDisable` | `false` | Turn the gopls bridge off. |
| `piko.goplsMaxChildren` | `0` | Maximum concurrent gopls child processes, up to 256 (`0` uses the server default). |
| `piko.goplsMaxOverlays` | `0` | Maximum open Go-block overlays per gopls child, up to 65536 (`0` uses the server default). |

Changing any of these restarts the language server.

### Workspace trust

In untrusted workspaces the extension runs in limited mode: workspace-level values of `piko.lspPath`, `piko.goBinPath`, `piko.goplsPath`, `piko.useExternalServer`, `piko.externalServerPort` and `piko.detectGoFromExtension` are ignored, so a cloned repository cannot choose which executables run. Virtual workspaces are not supported.

## Commands

- **Piko: New Piko File**
- **Piko: Restart Language Server** (works even if the server failed to start)
- **Piko: Show Output**
- **Piko: Open LSP Logs** (needs `piko.enableFileLogging`)
- **Piko: Toggle Line Comment** and **Piko: Toggle Block Comment** (bound to the usual comment keys in Piko files)

## Troubleshooting

- **The server did not start.** Open **Piko: Show Output**. A missing bundled binary means the extension was packaged without `bin/<os>-<arch>/pikopls`; a bad `piko.lspPath` is reported with the expanded path.
- **Go features are missing in script blocks.** Check `gopls version`, set `piko.goplsPath` if gopls is not on `PATH`, and make sure `piko.goplsDisable` is off.
- **Something looks stale.** Run **Piko: Restart Language Server**. For server-side detail, enable `piko.enableFileLogging` and `piko.trace.server`.
- **Check which server is running.** `pikopls --version` prints the version of a given binary.

## Development

```bash
cd plugins/vscode
npm install
npm run compile          # development build into out/
npm run lint             # eslint, tsc --noEmit and duplicate detection
npm test                 # vitest unit tests
npm run test:coverage    # unit tests with coverage thresholds
npm run test:grammar     # TextMate grammar fixtures in test/grammar
```

Press `F5` to launch an Extension Development Host. `make plugin-vscode-build` from the repository root builds the six platform binaries and packages the `.vsix`; see [RELEASE.md](RELEASE.md) and [SETUP.md](SETUP.md).

### How requests are routed

```text
.pk / .pkc document
  template block           -> pikopls
  Go script block          -> pikopls -> gopls (bridge)
  TypeScript script block  -> virtual document -> VS Code TypeScript
  style / i18n / JS blocks -> highlighting only (not sent to pikopls)
```

Block boundaries come from `src/sfcBlocks.ts`, which mirrors the server's parser (`internal/sfcparser`), so the extension and pikopls always agree on where each block starts and ends.

## Licence

Apache-2.0. See [LICENSE](LICENSE).

## Links

- [Piko repository](https://github.com/piko-sh/piko)
- [Issue tracker](https://github.com/piko-sh/piko/issues)
- [Website](https://piko.sh)
