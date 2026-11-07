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

# Piko VS Code Extension: Quick Setup Guide

A runbook for building, installing and checking the extension locally.

## Prerequisites

```bash
go version      # must satisfy the go directive in the repository's go.mod
code --version  # VS Code 1.107 or later
npm --version
```

Install gopls for Go intelligence in script blocks:

```bash
go install golang.org/x/tools/gopls@latest
gopls version
```

## Build and install

From the repository root:

```bash
make plugin-vscode-install
```

This stops any `pikopls` started from `plugins/vscode/bin/`, builds `pikopls` for all six platforms, clears and refills `plugins/vscode/bin/`, compiles the extension with esbuild, packages a `.vsix` and installs it with `code --install-extension`. Reload the window afterwards (**Developer: Reload Window**).

The six bundled platforms are:

- `linux-amd64`, `linux-arm64`
- `darwin-amd64`, `darwin-arm64`
- `windows-amd64`, `windows-arm64`

Each lives in `plugins/vscode/bin/<os>-<arch>/pikopls` (`pikopls.exe` on Windows). The extension maps VS Code's platform names onto these (`win32` becomes `windows`, `x64` becomes `amd64`).

The packaged `.vsix` is roughly 120 MB, almost all of it the six binaries.

To build without installing, use `make plugin-vscode-build`; to remove build output, `make plugin-vscode-clean`.

## Verify

1. Open a `.pk` or `.pkc` file.
2. Run **Piko: Show Output**. You should see `Starting LSP over stdio: ...` followed by `Piko LSP client started`. There is no status bar item.
3. Hover a template expression such as `{{ state.Title }}`; you should get type information.
4. Type `p-` inside a tag; you should get directive completions.
5. Hover a Go identifier inside `<script type="application/x-go">`; with gopls installed you should get Go hover text.

To see exactly which server binary is running, check the path in the output channel and run it with `--version`.

## Troubleshooting

### Server logs

Server logs are off by default. Enable them with:

```json
{
  "piko.enableFileLogging": true,
  "piko.trace.server": "verbose"
}
```

Then run **Piko: Open LSP Logs**. The log is `pikopls.log` in the extension's log directory (the path is shown if the file does not exist yet). Protocol traces go to the "Piko Language Server" output channel.

### Binary not found

```bash
ls -lh plugins/vscode/bin/*/pikopls*
make plugin-vscode-build
```

### gopls not working

```bash
which gopls
```

If gopls is installed somewhere VS Code cannot see (for example `~/go/bin` missing from the `PATH` VS Code was launched with), set `piko.goplsPath`.

## Development workflow

```bash
cd plugins/vscode
npm install
npm run watch
```

Press `F5` to launch an Extension Development Host with the `test-workspace` folder. The watch task uses the esbuild problem matcher, which comes from the recommended `connor4312.esbuild-problem-matchers` extension.

### Use a locally built server

```bash
go build -o /tmp/pikopls ./cmd/pikopls
```

```json
{
  "piko.lspPath": "/tmp/pikopls"
}
```

Then run **Piko: Restart Language Server**. `piko.lspPath` also accepts `~/...`, `${workspaceFolder}/...` or a bare `pikopls` found on `PATH`.

### Debug over TCP

Set `"piko.transport": "tcp"` to start pikopls on a free local port (one connection only), or start pikopls yourself with `--tcp --port=4389` and set `"piko.useExternalServer": true`.

## Block support at a glance

| Block | Served by |
| --- | --- |
| `<template>` | pikopls |
| Go `<script>` | pikopls, which delegates to gopls |
| TypeScript `<script>` | VS Code's TypeScript service through a virtual document |
| `<style>`, `<i18n>`, JavaScript `<script>` | Syntax highlighting and comment toggling only |

## Support

- Issue tracker: https://github.com/piko-sh/piko/issues
- See [README.md](README.md) for settings and features, and [RELEASE.md](RELEASE.md) for packaging.
