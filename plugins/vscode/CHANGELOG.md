# Changelog

All notable changes to the Piko VS Code extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2025-01-XX

### Added

#### Language Support
- Initial release of Piko language support for VS Code
- Full LSP integration for `<template>` blocks with Piko template DSL
- Embedded language support for `<script>` blocks via gopls (Go language server)
- Embedded language support for `<style>` blocks via built-in CSS language server
- Embedded language support for `<i18n>` blocks via built-in JSON language server

#### Features
- Syntax highlighting for all block types in `.pk` files
- Hover information for template variables and directives
- Code completion for template expressions and directives
- Go to definition for template symbols
- Document symbols (outline view)
- Document formatting (full and range)
- On-type formatting with auto-indent
- Folding ranges for block structures
- Linked editing ranges
- Code actions and quick fixes
- Rename refactoring support
- Signature help for template functions

#### Configuration
- `piko.lspPath` - Custom path to piko-lsp binary
- `piko.trace.server` - LSP communication tracing
- `piko.enableTemplateSupport` - Enable/disable template support

#### Commands
- `Piko: Restart Language Server` - Restart the LSP server
- `Piko: Show Output` - Show LSP output channel
- `Piko: Open LSP Logs` - Open LSP log files

#### Developer Experience
- Comprehensive README with usage examples
- Debug configurations for extension development
- Build scripts for cross-platform LSP binaries
- Middleware architecture for clean separation of language servers

### Technical Details

#### Architecture
- TextMate grammar with embedded language scopes
- Request filtering middleware to route by block type
- Regex-based block detection (future: WASM-based sfcparser)
- Cross-platform LSP binary bundling (Linux, macOS, Windows)

#### Dependencies
- vscode-languageclient ^9.0.1
- TypeScript ^5.3.0
- Requires gopls for Go support

### Known Limitations
- Block detection uses regex (may have edge cases with nested structures)
- Requires gopls to be installed separately for `<script>` block support
- First activation may take 1-2 seconds (LSP bootstrap)

[unreleased]: https://piko.sh/piko/compare/v0.1.0...HEAD
[0.1.0]: https://piko.sh/piko/releases/tag/v0.1.0
