# Piko Language Support for VS Code

Official VS Code extension providing language support for Piko template files (`.pk`).

## Features

### Multi-Language Support with Embedded Language Servers

This extension provides full language support for all blocks in `.pk` files:

- **`<template>` blocks**: Piko template DSL with custom directives (`p-if`, `p-for`, etc.)
  - Hover information
  - Code completion
  - Go to definition
  - Document symbols
  - Formatting
  - Syntax highlighting

- **`<script type="application/x-go">` blocks**: Full Go language support via gopls
  - Go IntelliSense, diagnostics, and all features from the official Go extension
  - Automatically handled by the gopls language server

- **`<style>` blocks**: Full CSS language support
  - CSS IntelliSense and validation
  - Handled by VS Code's built-in CSS language server

- **`<i18n>` blocks**: JSON language support
  - JSON schema validation
  - Handled by VS Code's built-in JSON language server

### Architecture: Language Injection Pattern

This extension follows the same architecture as the IntelliJ plugin:
- Each block type is handled by its specialized language server
- No reimplementation of parsers - we use existing, established language servers
- The Piko LSP focuses exclusively on template block features
- TextMate grammar scopes trigger the appropriate language server for each block

## Prerequisites

### Required

1. **VS Code** version 1.75.0 or higher
2. **Go** toolchain (for `<script>` block support)
   - **CRITICAL**: Your Go version must match or exceed your project's `go.mod` requirement
   - The Piko LSP uses Go to analyse template code and build type information
   - Check your version: `go version`
   - If you see errors like "go.mod requires go >= X.Y.Z", upgrade Go or set `GOTOOLCHAIN=auto`
3. **gopls** (Go language server)
   ```bash
   go install golang.org/x/tools/gopls@latest
   ```

### Recommended

- **Go extension** for VS Code (provides additional Go features)
  - Install: `ms-vscode.go`

## Installation

### From `.vsix` Package (Local Development)

1. Build the extension:
   ```bash
   cd plugins/vscode
   npm install
   npm run compile
   npm run package
   ```

2. Install the generated `.vsix` file:
   ```bash
   code --install-extension piko-0.1.0.vsix
   ```

### From VS Code Marketplace (Future)

Search for "Piko" in the Extensions view (`Ctrl+Shift+X` / `Cmd+Shift+X`)

## Configuration

### Settings

Access settings via `File > Preferences > Settings` (or `Code > Preferences > Settings` on macOS), then search for "Piko":

- **`piko.lspPath`** (string, default: `""`)
  - Custom path to the `piko-lsp` binary
  - Leave empty to use the bundled binary
  - Example: `/usr/local/bin/piko-lsp`

- **`piko.trace.server`** (enum, default: `"off"`)
  - Trace LSP communication for debugging
  - Options: `"off"`, `"messages"`, `"verbose"`
  - View traces in the "Piko Language Server" output channel

- **`piko.enableTemplateSupport`** (boolean, default: `true`)
  - Enable/disable Piko LSP for `<template>` blocks
  - Disable if experiencing issues

### Example Configuration

Add to your `settings.json`:

```json
{
  "piko.lspPath": "",
  "piko.trace.server": "off",
  "piko.enableTemplateSupport": true
}
```

## Usage

### Opening a Piko File

1. Open any `.pk` file
2. The extension activates automatically
3. Check the "Piko Language Server" output channel for status

### Testing Different Block Types

**Template Block**:
```html
<template>
  <div class="container">
    <h1>{{ title }}</h1>
    <button p-if="showButton">Click me</button>
  </div>
</template>
```
- Hover over `title` to see information from Piko LSP
- Type `p-` to see directive completions

**Script Block**:
```html
<script type="application/x-go">
package main

type Props struct {
    Title string
}

func Render(props Props) any {
    return map[string]any{
        "title": props.Title,
    }
}
</script>
```
- Hover over `Props` to see Go type information from gopls
- Press `F12` on `Render` to go to definition (handled by gopls)

**Style Block**:
```html
<style>
.container {
    padding: 20px;
    background-color: #f0f0f0;
}
</style>
```
- Get CSS completions (e.g., type `col` to see color properties)
- Hover over color values to see color picker

## Commands

Access commands via the Command Palette (`Ctrl+Shift+P` / `Cmd+Shift+P`):

- **`Piko: Restart Language Server`**
  - Restart the Piko LSP server
  - Useful if the server becomes unresponsive

- **`Piko: Show Output`**
  - Open the "Piko Language Server" output channel
  - View LSP communication logs and status

- **`Piko: Open LSP Logs`**
  - Open the LSP server log file at `/tmp/piko-lsp-main.log`
  - Useful for debugging server-side issues

## Troubleshooting

### Extension Not Activating

1. Check the "Piko Language Server" output channel
2. Verify the file extension is `.pk`
3. Ensure `piko.enableTemplateSupport` is `true`

### LSP Binary Not Found

**Error**: `Bundled piko-lsp binary not found`

**Solution**:
1. Build the LSP binaries:
   ```bash
   cd /path/to/piko
   make build-lsp-all
   ```
2. Rebuild the extension:
   ```bash
   cd plugins/vscode
   npm run compile
   ```

### Go Features Not Working in `<script>` Blocks

**Error: "go.mod requires go >= X.Y.Z"**

Your installed Go version is too old for your project. Fix:

1. **Check Go version:**
   ```bash
   go version
   # Compare with your go.mod requirement
   ```

2. **Option A: Enable auto-toolchain (recommended):**
   ```bash
   export GOTOOLCHAIN=auto
   # Restart VS Code
   ```

3. **Option B: Upgrade Go:**
   - Download from https://go.dev/dl/
   - Install and verify: `go version`

**Other Issues:**

1. Verify `gopls` is installed:
   ```bash
   gopls version
   ```
2. Install if missing:
   ```bash
   go install golang.org/x/tools/gopls@latest
   ```
3. Ensure the Go extension is installed in VS Code

### CSS/JSON Features Not Working

- These are built into VS Code and should work automatically
- Try reloading the window: `Developer: Reload Window`

### Server Unresponsive

1. Run: `Piko: Restart Language Server` command
2. Check `/tmp/piko-lsp-main.log` for errors
3. Increase trace level: Set `piko.trace.server` to `"verbose"`

## Development

### Building from Source

```bash
# Clone the repository
git clone https://piko.sh/piko
cd piko

# Build LSP binaries
make build-lsp-all

# Build the extension
cd plugins/vscode
npm install
npm run compile
```

### Testing

Press `F5` in VS Code to launch the Extension Development Host with the extension loaded.

### Packaging

```bash
npm run package
```

This creates `piko-0.1.0.vsix` which can be installed locally or published to the marketplace.

## Architecture

### Request Flow

```text
.pk file opened
    ↓
TextMate grammar assigns scopes:
  - <script> content → source.go
  - <style> content → source.css
  - <i18n> content → source.json
  - <template> content → text.html.piko
    ↓
VS Code routes requests based on scope:
  - source.go → gopls
  - source.css → Built-in CSS LS
  - source.json → Built-in JSON LS
  - text.html.piko → Piko LSP (with middleware filter)
    ↓
Middleware in extension.ts:
  - Checks cursor position
  - Only forwards template block requests to Piko LSP
  - Returns null for other blocks (handled by their respective servers)
```

### Components

1. **TextMate Grammar** (`syntaxes/piko.tmLanguage.json`)
   - Defines syntax highlighting
   - Uses `contentName` to assign language scopes

2. **Block Detector** (`src/blockDetector.ts`)
   - Regex-based block position detection
   - Determines which block the cursor is in

3. **Middleware** (`src/middleware.ts`)
   - Filters LSP requests by block type
   - Only forwards template block requests to Piko LSP

4. **Extension Activation** (`src/extension.ts`)
   - Discovers and launches the Piko LSP binary
   - Configures the LSP client with middleware

## Contributing

Contributions are welcome! Please see the main Piko repository for contribution guidelines.

## License

MIT License - See LICENSE file in the Piko repository.

## Links

- [Piko Repository](https://piko.sh/piko)
- [Issue Tracker](https://piko.sh/piko/issues)
- [Documentation](https://piko.politepixels.io)
