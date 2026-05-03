---
title: Safedisk API
description: Sandbox factory and sandboxed filesystem operations for path-validated I/O.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 280
---

# Safedisk API

The `piko.sh/piko/wdk/safedisk` package provides a `Factory` that creates `Sandbox` instances rooted at validated absolute paths. A sandbox uses Go 1.24's `os.Root` (Linux: `openat2(..., RESOLVE_BENEATH)`) to confine reads and writes to a directory. Path traversal, symlink escape, and time-of-check-to-time-of-use (TOCTOU) attempts fail at the syscall layer instead of in user code. Use it for any code path that accepts untrusted file paths (uploads, browser-test fixtures, project-relative reads).

## Factory

```go
type Factory interface {
    Create(purpose, path string, mode Mode) (Sandbox, error)
    MustCreate(purpose, path string, mode Mode) Sandbox
    IsPathAllowed(path string) bool
    AllowedPaths() []string
}
```

| Method | Purpose |
|---|---|
| `Create(purpose, path, mode)` | Create a sandbox at `path`. Returns an error if `path` is not within the factory's `AllowedPaths`. `purpose` is descriptive text used in error messages and logs. |
| `MustCreate(purpose, path, mode)` | Same as `Create` but panics on error. Use only at startup when the caller cannot recover from failure. |
| `IsPathAllowed(path)` | Reports whether `Create` would accept a path without actually creating a sandbox. |
| `AllowedPaths()` | Returns the configured allowlist (empty when the factory allows all paths). |

### Constructors

```go
func NewFactory(config FactoryConfig) (Factory, error)
func NewCLIFactory(cwd string) (Factory, error)
func Create(purpose, path string, mode Mode) (Sandbox, error)
```

| Function | Purpose |
|---|---|
| `NewFactory(config)` | Build a factory from `FactoryConfig`. |
| `NewCLIFactory(cwd)` | Convenience constructor for CLI tools. Allows the working directory and validates absolute paths. |
| `Create(...)` | Top-level convenience that uses the global factory. Initialise it via `NewFactory` (or `NewCLIFactory`) at startup. |

### `FactoryConfig`

```go
type FactoryConfig struct {
    CWD          string   // Current working directory; always allowed. Auto-detected when empty.
    AllowedPaths []string // Absolute paths that may be sandboxed. Empty list allows all paths.
    Enabled      bool     // false means no-op sandbox (no real isolation, useful for tests).
}
```

## Modes

```go
type Mode uint8
```

`Mode` distinguishes read-only from read-write sandboxes. `Mode.String()` returns the human-readable form. The canonical check on a sandbox is `IsReadOnly()`.

## Sandbox interface

`Sandbox` exposes a sandboxed `os`-shaped surface. The sandbox root resolves every relative path. The kernel rejects absolute paths and paths that escape (`..`).

### Read

| Method | Purpose |
|---|---|
| `Open(name) (FileHandle, error)` | Open a file for reading. |
| `ReadFile(name) ([]byte, error)` | Read the entire file. |
| `ReadFileLimit(name, maxBytes) ([]byte, int64, error)` | Stat-then-read with a size cap. Returns `(data, statSize, error)`. Refuses to allocate when the file exceeds `maxBytes` so a malformed or attacker-influenced file cannot dominate memory. |
| `Stat(name) (fs.FileInfo, error)` | Follows symlinks. |
| `Lstat(name) (fs.FileInfo, error)` | Does NOT follow symlinks. |
| `ReadDir(name) ([]fs.DirEntry, error)` | Directory entries. |
| `WalkDir(root, walkFunc) error` | Recursive walk under `root`. |

### Write

All write methods return an error when the caller opens the sandbox in read-only mode.

| Method | Purpose |
|---|---|
| `Create(name) (FileHandle, error)` | Create or truncate a file for writing. |
| `OpenFile(name, flag, perm) (FileHandle, error)` | Most flexible open (mirrors `os.OpenFile`). |
| `WriteFile(name, data, perm) error` | Create or truncate, write, close. |
| `WriteFileAtomic(name, data, perm) error` | Write to a temp file in the same directory, then rename - survives a crash mid-write. |
| `Mkdir(name, perm) error` | Create one directory. |
| `MkdirAll(path, perm) error` | Create the full path tree. |
| `Remove(name) error` | Delete a file or empty directory. |
| `RemoveAll(path) error` | Delete recursively. |
| `Rename(oldpath, newpath) error` | Both paths must be inside the same sandbox. |
| `Chmod(name, mode) error` | Change permissions. |
| `CreateTemp(dir, pattern) (FileHandle, error)` | Like `os.CreateTemp` but inside the sandbox. |
| `MkdirTemp(dir, pattern) (string, error)` | Like `os.MkdirTemp` but inside the sandbox. |

### Inspect

| Method | Purpose |
|---|---|
| `Root() string` | Absolute path of the sandbox root. |
| `Mode() Mode` | The configured `Mode`. |
| `IsReadOnly() bool` | Convenience check. |
| `Close() error` | Release the underlying `os.Root` handle. |
| `RelPath(path) string` | Convert an absolute path (or one prefixed with the sandbox folder name) to a sandbox-relative path. |

## `FileHandle`

A `FileHandle` exposes the standard `io.Reader`/`io.Writer`/`io.Closer` surface plus stat, sync, truncate, and seek. The concrete implementation is `*File`, returned by `Open`, `Create`, `OpenFile`, and `CreateTemp`.

| Method | Purpose |
|---|---|
| `Name() string` | Sandbox-relative name. |
| `AbsolutePath() string` | Absolute path on disk. |
| `Read`, `ReadAt`, `Write`, `WriteAt`, `WriteString`, `Seek` | Standard byte-level operations. |
| `Sync() error` | Flush to disk. |
| `Truncate(size) error` | Truncate to length. |
| `Stat() (fs.FileInfo, error)` | File info. |
| `Chmod(mode) error` | Change permissions. |
| `ReadDir(n) ([]fs.DirEntry, error)` | Directory entries (when the handle is a directory). |
| `Fd() uintptr` | Underlying file descriptor. |
| `Close() error` | Release the handle. |

## Errors

The package exposes typed sentinel errors for the common rejection cases (empty path, path escapes the sandbox root, file exceeds `ReadFileLimit`, sandbox closed). Use `errors.Is` to match.

## Mock

`safedisk` ships a mock implementation suitable for tests that do not want to touch disk. See `mock.go` for constructors. The contract matches the `Factory` interface so test code is identical.

## See also

- Source: [`wdk/safedisk/`](https://github.com/piko-sh/piko/tree/master/wdk/safedisk).
- [Hexagonal architecture explanation](../explanation/about-the-hexagonal-architecture.md) for why Piko exposes filesystem access as a port.
