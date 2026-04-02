# Piko VS Code Extension - Quick Setup Guide

Quick runbook for setting up and testing the Piko LSP extension locally.

## Prerequisites

Ensure you have the following installed:

```bash
# Check Go version (requires 1.26+)
go version

# Check VS Code is available
code --version

# Check npm is installed
npm --version
```

## Setup Steps

### 1. Install gopls (Go Language Server)

Required for Go support in `<script>` blocks:

```bash
go install golang.org/x/tools/gopls@latest

# Verify installation
gopls version
```

### 2. Build LSP Binaries

From the project root:

```bash
make build-lsp-all
```

This creates binaries in `bin/lsp/` for:
- darwin-amd64 (macOS Intel)
- darwin-arm64 (macOS Apple Silicon)
- linux-amd64 (Linux)
- windows-amd64 (Windows)

To build only for your current platform:

```bash
make build-lsp
```

### 3. Build and Package Extension

```bash
cd plugins/vscode

# Install dependencies (first time only)
npm install

# Build and package
npm run package
```

This creates `piko-0.1.0.vsix` (approximately 45MB).

**Bundling:** The extension uses esbuild to bundle all dependencies into a single optimised file:
- Reduces from 336 files to 13 files
- Tree-shakes unused code
- Minifies for production
- Improves load performance
- See `esbuild.js` for configuration

### 4. Install Extension in VS Code

```bash
code --install-extension piko-0.1.0.vsix
```

Alternatively:
- Open VS Code
- Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
- Run: `Extensions: Install from VSIX...`
- Select `piko-0.1.0.vsix`

### 5. Reload VS Code

```bash
# Reload window to activate extension
# In VS Code: Ctrl+Shift+P > "Developer: Reload Window"
```

## Verification

### Check Extension is Active

1. Open a `.pk` file
2. Check status bar for "Piko" indicator
3. Open Output panel: `View > Output`
4. Select "Piko Language Server" from dropdown
5. Should see: "Piko LSP server started successfully"

### Test Features

**Hover in `<template>` block:**
```html
<template>
  <h1>{{ title }}</h1>
</template>
```
Hover over `title` - should show type information.

**Completion in `<template>` block:**
Type `p-` - should see directive completions (`p-if`, `p-for`, etc.).

**Go support in `<script>` block:**
```html
<script type="application/x-go">
package main

type Props struct {
    Title string
}
</script>
```
Hover over `string` - should show Go type information from gopls.

## Troubleshooting

### Extension Not Activating

```bash
# Check LSP logs
cat /tmp/piko-lsp-main.log

# In VS Code, enable verbose logging
# Settings > Piko: Trace Server > "verbose"
```

### LSP Binary Not Found

```bash
# Verify binaries exist
ls -lh bin/lsp/*/piko-lsp*

# Rebuild if missing
make build-lsp-all
```

### gopls Not Working

```bash
# Verify gopls is in PATH
which gopls

# Install Go extension for VS Code
# Extension ID: golang.go
code --install-extension golang.go
```

### Rebuild After Changes

```bash
# After modifying LSP code
make build-lsp-all

# After modifying extension code
cd plugins/vscode
npm run compile
npm run package
code --install-extension piko-0.1.0.vsix
```

## Development Workflow

### Test Without Installing

Press `F5` in VS Code while in the `plugins/vscode` directory to launch Extension Development Host with live reloading.

### Development with Watch Mode

```bash
cd plugins/vscode

# Run esbuild in watch mode for live rebuilding
npm run watch
```

This automatically rebuilds the extension when you modify TypeScript files. Press `F5` to test changes in Extension Development Host.

### Hot Reload LSP Binary

During development, use a custom LSP path:

```json
// settings.json
{
  "piko.lspPath": "/path/to/piko/cmd/lsp/piko-lsp"
}
```

Build the binary separately:
```bash
go build -o /tmp/piko-lsp ./cmd/lsp
```

Then update the setting to `/tmp/piko-lsp` and restart the language server.

## Configuration

### Recommended Settings

```json
{
  "piko.lspPath": "",
  "piko.trace.server": "off",
  "piko.enableTemplateSupport": true,
  "[piko]": {
    "editor.formatOnSave": true,
    "editor.defaultFormatter": "politepixels.piko"
  }
}
```

### Enable Debug Logging

```json
{
  "piko.trace.server": "verbose"
}
```

Then check:
- VS Code Output: "Piko Language Server"
- Log file: `/tmp/piko-lsp-main.log`

## Quick Commands

```bash
# Full rebuild and install (recommended)
make plugin-vscode-install

# Or manually:
make build-lsp-all && \
  cd plugins/vscode && \
  npm run package && \
  code --install-extension piko-0.1.0.vsix

# Restart LSP server in VS Code
# Ctrl+Shift+P > "Piko: Restart Language Server"

# View LSP logs
tail -f /tmp/piko-lsp-main.log
```

## Multi-Language Support

The extension automatically provides language support for all block types:

| Block Type | Language Server | Auto-Installed |
|------------|----------------|----------------|
| `<template>` | Piko LSP | Yes (bundled) |
| `<script type="application/x-go">` | gopls | No (manual install) |
| `<style>` | VS Code built-in CSS LS | Yes |
| `<i18n>` | VS Code built-in JSON LS | Yes |

**Note:** Only gopls requires manual installation. CSS and JSON support is built into VS Code.

## CI/CD Notes

For automated builds:

```bash
# Build LSP for all platforms
make build-lsp-all

# Package extension
cd plugins/vscode
npm ci
npm run package

# Artefact: piko-0.1.0.vsix
```

## Support

- Issue Tracker: https://piko.sh/piko/issues
- Documentation: See `README.md` in this directory
