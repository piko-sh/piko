---
title: Safedisk API
description: Sandboxed filesystem with kernel-level path traversal protection.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 280
---

# Safedisk API

The safedisk package scopes filesystem operations to a kernel-enforced sandbox using Go 1.24's `os.Root` (which on Linux uses `openat2(..., RESOLVE_BENEATH)`). Any path traversal, symlink escape, or TOCTOU attempt fails at the syscall layer instead of in user code. Browser tests, user file uploads, and any other untrusted-path code path should use safedisk. Source of truth: [`wdk/safedisk/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/safedisk/facade.go).

## Constructors

```go
func NewFactory(config FactoryConfig) (Factory, error)
```

`FactoryConfig` fields: `Enabled` (disables enforcement for testing), `ReadOnly` (forbids writes).

## Factory

```go
type Factory interface {
    CreateSandbox(path string, mode Mode) (Sandbox, error)
    AllowPath(path string)
    AllowPaths(paths ...string)
    DisallowPath(path string)
    DisallowPaths(paths ...string)
    IsPathAllowed(path string) bool
}
```

`AllowPath` / `DisallowPath` form an allow-list that governs which sandbox root paths callers may open. The factory implicitly allows the working directory.

## Modes

| Constant | Meaning |
|---|---|
| `ModeReadOnly` | Forbids writes at the sandbox level. |
| `ModeReadWrite` | Writes allowed. |

## Sandbox

```go
type Sandbox interface {
    Open(name string) (FileHandle, error)
    ReadFile(name string) ([]byte, error)
    Stat(name string) (fs.FileInfo, error)
    CreateTemp(dir, pattern string) (FileHandle, error)
    WriteFile(name string, data []byte, perm fs.FileMode) error
    Rename(oldname, newname string) error
    Remove(name string) error
    Mkdir(name string, perm fs.FileMode) error
    MkdirAll(name string, perm fs.FileMode) error
    RemoveAll(name string) error
    Readlink(name string) (string, error)
    Symlink(oldname, newname string) error
    ReadDir(name string) ([]fs.DirEntry, error)
    Walk(root string, fn fs.WalkDirFunc) error
    Chmod(name string, mode fs.FileMode) error
    Chown(name string, uid, gid int) error
    Chtimes(name string, atime, mtime time.Time) error
}
```

## `FileHandle`

Returned by `Open` and `CreateTemp`. Implements a superset of `*os.File`:

```go
type FileHandle interface {
    Name() string
    AbsolutePath() string
    Read(p []byte) (int, error)
    ReadAt(p []byte, off int64) (int, error)
    Write(p []byte) (int, error)
    WriteAt(p []byte, off int64) (int, error)
    WriteString(s string) (int, error)
    Seek(offset int64, whence int) (int64, error)
    Sync() error
    Truncate(size int64) error
    Stat() (fs.FileInfo, error)
    Chmod(mode fs.FileMode) error
    Close() error
    ReadDir(n int) ([]fs.DirEntry, error)
    Fd() uintptr
}
```

## Safety guarantees

| Guarantee | Enforcement |
|---|---|
| Traversal | `../` escapes fail at `openat2` syscall level. |
| Symlinks | Escapes fail with `ENOTDIR`. |
| TOCTOU | The sandbox holds a directory file descriptor, and all subsequent paths resolve relative to it. |
| Atomic writes | `CreateTemp` + `Rename` within the same sandbox produces atomic file replacement. |
| Read-only mode | The factory rejects writes at its boundary, before any syscall runs. |

## See also

- [Browser testing harness](browser-testing.md) uses safedisk for test output.
