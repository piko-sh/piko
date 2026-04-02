# Piko IntelliJ Plugin

IntelliJ IDEA / GoLand plugin for Piko template files (.pk).

```bash
export JAVA_HOME=/usr/lib/jvm/java-21-openjdk
./gradlew clean generatePKLexer build
./gradlew clean generatePKLexer build runIde
./gradlew build
./gradlew runIde
./gradlew buildPlugin
./gradlew verifyPlugin 
./gradlew signPlugin 
./gradlew publishPlugin 

ls -al build/distributions/piko-idea-0.1.0.zip
```

## Features

### Language Injection

The plugin injects the appropriate language into each block:

| Block Type | Language Injected | Requirements |
|------------|------------------|--------------|
| `<script>` | Go | Go plugin installed |
| `<script lang="go">` | Go | Go plugin installed |
| `<script lang="js">` | TypeScript | JavaScript plugin installed |
| `<script lang="ts">` | TypeScript | JavaScript plugin installed |
| `<style>` | CSS | CSS plugin installed |
| `<template>` | Piko LSP | LSP4IJ + piko-lsp binary |

### LSP Support

The template block uses the Piko Language Server Protocol for:
- Hover information
- Go to definition
- Completions
- Diagnostics
- And more...

## Prerequisites

- IntelliJ IDEA Ultimate or GoLand 2024.3+
- Go plugin (bundled with GoLand)
- CSS plugin (bundled)
- JavaScript plugin (bundled with Ultimate)
- [LSP4IJ](https://plugins.jetbrains.com/plugin/23257-lsp4ij) plugin (for template LSP support)
- `piko-lsp` binary in PATH or project directory

## Building

### Prerequisites for Building

- JDK 21 (not just JRE - requires `javac`)
- On Fedora: `sudo dnf install java-21-openjdk-devel`
- On Ubuntu: `sudo apt install openjdk-21-jdk`

### Build Commands

```bash
# Generate the lexer
./gradlew generatePKLexer

# Build the plugin
./gradlew buildPlugin
```

The plugin will be created in `build/distributions/`.

## Development

```bash
# Run IDE with plugin for testing
./gradlew runIde
```

## Installation

1. Build the plugin or download from JetBrains Marketplace
2. In IntelliJ: Settings -> Plugins -> Install from disk
3. Install the LSP4IJ plugin from the Marketplace
4. Ensure `piko-lsp` is available in your PATH

## File Structure

```html
.pk file structure:

<script>
// Go code here - full IntelliJ Go support
</script>

<script lang="js">
// TypeScript/JavaScript code - full TS support
</script>

<style>
/* CSS code - full CSS support */
</style>

<template>
<!-- HTML template with Piko directives - LSP support -->
<div p-if="condition">
  {{ expression }}
</div>
</template>
```
