# VSCode Extension Release Guide

## Quick Release

The fastest way to build and install the extension:

```bash
# From the project root
make plugin-vscode-install
```

Then reload VSCode (Ctrl+Shift+P → "Developer: Reload Window")

## Makefile Targets

The build process is driven by Makefile targets that wrap `hack/plugin/vscode.sh`.

### Commands

```bash
# Build everything (LSP binaries + TypeScript + VSIX package)
make plugin-vscode-build

# Build and install in VSCode
make plugin-vscode-install

# Clean build artefacts
make plugin-vscode-clean
```

### What the Build Does

1. **Kills running LSP processes**
   - Finds and terminates any `piko-lsp` processes
   - Prevents VSCode from using cached old binaries

2. **Builds LSP binaries** for all platforms:
   - Linux AMD64
   - macOS AMD64 (Intel)
   - macOS ARM64 (Apple Silicon)
   - Windows AMD64

3. **Compiles TypeScript**
   - Runs esbuild in production mode
   - Bundles extension code

4. **Packages extension**
   - Creates `.vsix` file with `vsce package`
   - Includes all binaries and resources

5. **Installs in VSCode** (with `plugin-vscode-install`)
   - Uninstalls old version
   - Installs new `.vsix`
   - Prompts to reload window

## Manual Build Process

If you need to build manually:

### 1. Build LSP Binary (Linux example)

```bash
cd ../../  # Go to piko root
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-s -w" \
  -o plugins/vscode/bin/linux-amd64/piko-lsp \
  ./cmd/lsp
```

### 2. Compile TypeScript

```bash
cd plugins/vscode
npm install
npm run compile:production
```

### 3. Package Extension

```bash
npm run package
```

This creates `piko-<version>.vsix`

### 4. Install in VSCode

```bash
# Kill old LSP processes
pkill -f piko-lsp

# Uninstall old version
code --uninstall-extension politepixels.piko

# Install new version
code --install-extension piko-*.vsix --force
```

### 5. Reload VSCode

Press `Ctrl+Shift+P` and select "Developer: Reload Window"

## Platform-Specific Builds

### Building for macOS (from Linux)

```bash
# macOS Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
  -o plugins/vscode/bin/darwin-amd64/piko-lsp ./cmd/lsp

# macOS Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
  -o plugins/vscode/bin/darwin-arm64/piko-lsp ./cmd/lsp
```

### Building for Windows (from Linux)

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
  -o plugins/vscode/bin/windows-amd64/piko-lsp.exe ./cmd/lsp
```

## Troubleshooting

### "Please restart VS Code before reinstalling"

This happens when VSCode still has the old extension loaded. Solutions:

1. **Best:** Use the Makefile target which kills processes first
   ```bash
   make plugin-vscode-install
   ```

2. **Alternative:** Fully quit and restart VSCode
   ```bash
   pkill -f code
   # Wait a few seconds
   code .
   ```

### Extension not using new binary

1. Kill all LSP processes:
   ```bash
   pkill -f piko-lsp
   ```

2. Reload VSCode window:
   - Ctrl+Shift+P → "Developer: Reload Window"

3. Check the binary timestamp:
   ```bash
   ls -lh ~/.vscode/extensions/politepixels.piko-*/bin/linux-amd64/piko-lsp
   ```

### Verify LSP is working

Check the logs:
```bash
# Main LSP log
tail -f /tmp/piko-lsp-main.log

# LSP communication log
tail -f /tmp/piko-lsp.log
```

Look for:
- No "context canceled" errors on every file open
- "Analysis completed successfully" messages
- `piko/analysisComplete` notifications

### Clean rebuild

If something goes wrong:

```bash
# Clean everything
make plugin-vscode-clean

# Remove VSCode extension
rm -rf ~/.vscode/extensions/politepixels.piko-*

# Rebuild and install
make plugin-vscode-install
```

## Release Checklist

Before releasing a new version:

- [ ] Update version in `package.json`
- [ ] Test LSP fixes are working
- [ ] Run `make plugin-vscode-build` to build all platforms
- [ ] Test on Linux
- [ ] Test on macOS (if possible)
- [ ] Test on Windows (if possible)
- [ ] Update CHANGELOG.md
- [ ] Commit and tag release
- [ ] Publish to marketplace: `npm run publish`

## CI/CD Notes

The Makefile targets work in CI/CD pipelines:

```yaml
# Example GitHub Actions
- name: Build Extension
  run: make plugin-vscode-build

- name: Upload VSIX
  uses: actions/upload-artefact@v2
  with:
    name: piko-extension
    path: plugins/vscode/piko-*.vsix
```

## Directory Structure

```text
plugins/vscode/
├── bin/                    # LSP binaries (gitignored)
│   ├── linux-amd64/
│   │   └── piko-lsp
│   ├── darwin-amd64/
│   │   └── piko-lsp
│   ├── darwin-arm64/
│   │   └── piko-lsp
│   └── windows-amd64/
│       └── piko-lsp.exe
├── out/                    # Compiled TypeScript (gitignored)
├── src/                    # TypeScript source
├── package.json
└── piko-*.vsix            # Packaged extension (gitignored)
```

## Development Workflow

### During LSP Development

1. Make changes to LSP code in `cmd/lsp/` or `internal/lsp/`
2. Run `make plugin-vscode-install` to rebuild and install
3. Reload VSCode window
4. Test changes

### During Extension Development

1. Make changes to TypeScript in `src/`
2. Run `npm run watch` for auto-compilation
3. Press F5 in VSCode to launch Extension Development Host
4. Test changes
5. When satisfied, run `make plugin-vscode-install`

## Publishing to Marketplace

```bash
# Make sure you're logged in
vsce login politepixels

# Build all platforms
make plugin-vscode-build

# Publish
npm run publish
```

Or publish manually:
```bash
vsce publish
```

## Version Management

Update version before release:

```bash
# Patch version (0.1.0 → 0.1.1)
npm version patch

# Minor version (0.1.0 → 0.2.0)
npm version minor

# Major version (0.1.0 → 1.0.0)
npm version major
```

This automatically updates `package.json` and creates a git tag.
