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

# Piko IntelliJ Plugin

IntelliJ IDEA / GoLand plugin for Piko single-file components: `.pk` pages and partials, and `.pkc` client components.

## Features

### Language Injection

The plugin injects the appropriate language into each block:

| Block Type | Language Injected | Requirements |
|------------|------------------|--------------|
| `<script>` | Go | Go plugin installed |
| `<script type="application/x-go">` | Go | Go plugin installed |
| `<script lang="go">` | Go | Go plugin installed |
| `<script lang="js">` / `lang="ts"` | TypeScript | JavaScript plugin installed |
| `<script lang="...">` (anything else) | None (plain text) | |
| `<style>` | CSS | CSS plugin installed |
| `<i18n>` | JSON | JSON support (bundled) |
| `<template>` | Piko LSP | LSP4IJ |

Go script blocks resolve `.pk` component imports (`"@/partials/card.pk"` or the full module path) to the imported component's script block. TypeScript blocks see the `piko` namespace and the project's server actions: the plugin indexes `dist/ts/piko-ide.d.ts` and `dist/ts/piko-actions.d.ts` when they exist, and falls back to bundled copies before the first build.

### Template Support

Without the language server the plugin still provides syntax highlighting (including nested `<template>` elements and `{{ }}` inside attribute values), folding, brace and quote matching, the structure view, breadcrumbs, Emmet, live templates and file templates for pages, partials and `.pkc` components. Find Usages and rename work from script functions and style classes to their uses in the template.

### LSP Support

The template block uses the Piko language server (`pikopls`) for hover, go to definition, completions, diagnostics, references, document links, colours, inlay hints and more. When the Go plugin is not installed, the server's gopls bridge provides Go intelligence for script blocks instead.

The plugin bundles `pikopls` for Linux, macOS and Windows on amd64 and arm64, and extracts it to the IDE system directory on first use. It looks for the binary in this order:

1. The custom path in Settings > Languages & Frameworks > Piko
2. The bundled binary
3. `GOBIN`, then `GOPATH/bin` (default `~/go/bin`)
4. `PATH`

Paths inside the project are never searched, so opening a repository cannot make the IDE run a binary that the repository supplies.

The server runs over stdio by default. TCP transport and connecting to an external server are available under the Advanced settings for debugging. File logging is off by default; when enabled, Tools > Piko > Open LSP Logs opens the log, which is written under the IDE log directory.

## Prerequisites

- IntelliJ IDEA Ultimate or GoLand 2025.3+
- Go plugin (bundled with GoLand)
- CSS plugin (bundled)
- JavaScript plugin (bundled with Ultimate)
- [LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) plugin (for template LSP support)

## Building

### Prerequisites for Building

- JDK 21 (not just JRE - requires `javac`)
- On Fedora: `sudo dnf install java-21-openjdk-devel`
- On Ubuntu: `sudo apt install openjdk-21-jdk`

### Build Commands

From the repository root, `make plugin-idea-build` builds the `pikopls` binaries, copies them into the plugin and packages it. Inside `plugins/idea`:

```bash
# Build the plugin (the lexer is generated automatically)
./gradlew buildPlugin -x buildSearchableOptions

# Run the tests
./gradlew test

# Check the plugin descriptors
./gradlew verifyPluginStructure

# Check compatibility against the recommended IDE versions (downloads several IDEs)
./gradlew verifyPlugin
```

The plugin will be created in `build/distributions/`. Local builds take their version from `pluginVersion` in `gradle.properties`; release builds pass `-Pversion=<version>`.

## Development

```bash
# Run IDE with plugin for testing
./gradlew runIde
```

## Installation

1. Build the plugin or download from JetBrains Marketplace
2. In IntelliJ: Settings -> Plugins -> Install from disk
3. Install the LSP4IJ plugin from the Marketplace

## File Structure

```html
.pk file structure:

<script type="application/x-go">
// Go code here - full IntelliJ Go support
</script>

<script lang="ts">
// TypeScript/JavaScript code - full TS support
</script>

<style>
/* CSS code - full CSS support */
</style>

<i18n lang="json">
{ "en": { "greeting": "Hello" } }
</i18n>

<template>
<!-- HTML template with Piko directives - LSP support -->
<div p-if="condition">
  {{ expression }}
</div>
</template>
```
