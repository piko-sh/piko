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

# `.pk` breakpoints in GoLand: design decision

Status: primary design adopted (real Delve breakpoints on the `.pk` file).
Evidence below comes from a static inspection, with `javap -p -c`, of the GoLand 2025.3 jars that the build compiles against
(`goland-2025.3/plugins/go-plugin/lib/go-plugin.jar` and `goland-2025.3/lib/app.jar` in the Gradle transforms cache).

## Question

Can a GoLand Delve line breakpoint live on a `.pk` file and reach Delve with the absolute `.pk` path, given that the
generator now writes `//line /abs/path/page.pk:N` directives into `generated.go`?

## Evidence

1. **`DlvBreakpointType` only gates the gutter.**
   `com.goide.dlv.breakpoint.DlvBreakpointType` (id `DlvLineBreakpoint`, public, not `@ApiStatus.Internal`) implements
   `canPutAt` as `line >= 0 && LanguageUtil.getLanguageForPsi(project, file) == GoLanguage.INSTANCE && iterateLine(...)`.
   A `.pk` file is never Go, so the gutter never offers it there.
2. **`addLineBreakpoint` bypasses `canPutAt`.**
   `XBreakpointManagerImpl.addLineBreakpoint(type, url, line, props, temporary)` creates and registers the breakpoint with
   no `canPutAt` call and no threading assertion. In `app.jar`, `canPutAt` is only consulted by the toggle path
   (`XBreakpointUtil.getAvailableLineBreakpointTypes`, `XDebuggerUtilImpl`, `XToggleLineBreakpointActionHandler`) and by
   drag-to-move (`XBreakpointVisualRepresentation.canMoveTo`).
3. **The handler accepts any file.**
   `DlvDebugProcess$MyBreakpointHandler.registerBreakpoint` reads `getSourcePosition().getFile()/getLine()`, maps the file
   with `DlvDebugProcess.getDebuggerPath(VirtualFile)` and sends `DlvRequest$CreateBreakpoint(path, line, condition)`.
   There is no file type or language check.
4. **Path conversion works for non-Go files.**
   `getDebuggerPath` delegates to `DlvPositionConverter.toRemotePath`. The module-aware
   `com.goide.vgo.dlv.VgoDlvPositionConverter` takes the file's path relative to its `go.mod` root and suffix-matches it
   against Delve's `ListSources` result. Delve lists every file named in the DWARF line tables, which includes the
   absolute `.pk` paths from the `//line` directives. The fallback `DefaultDlvPositionConverter` returns the local path
   when the remote source set contains it exactly. Either way, Delve receives the absolute `.pk` path that the
   generator wrote.
5. **Stopped frames open the `.pk` file.**
   `DlvStackFrame.getSourcePosition` calls `DlvDebugProcess.findFile(location)`, which uses
   `DlvPositionConverter.toLocalFile(remotePath)` (module root plus `findFileByRelativePath`, or a `FilenameIndex`
   lookup), then `LocalFileSystem.findFileByPath`, then `XDebuggerUtil.createPosition(file, line)`. None of these care
   about the file type, so a frame at `/abs/pages/index.pk:69` opens `index.pk` at line 69.
6. **No extension point lets another language accept Dlv breakpoints.**
   The Go plugin exposes `com.goide.dlv.positionConverterFactory`, which converts paths but does not influence
   `canPutAt`. The bundled `go-template` plugin has no breakpoint support at all.
7. **Toggle only sees available types.**
   `XDebuggerUtilImpl.selectBreakpointVariantWithPopup` looks for existing breakpoints only among the types whose
   `canPutAt` accepted the line. A Dlv breakpoint on a `.pk` line is therefore invisible to the gutter toggle, so the
   shim has to implement "click again to remove" itself.

## Decision

Primary design:

- `PKLineBreakpointType` (`piko-line`) is a gutter shim, registered only in `piko-go.xml`. Its `canPutAt` accepts
  mappable Go script lines and template lines in `.pk` files, never JS lines and never `.pkc`. It also accepts any
  line that already holds a Delve breakpoint, so the user can always click it away.
- `PKBreakpointBridge` listens on `XBreakpointListener.TOPIC`. When a shim breakpoint appears it removes it and either
  creates a `DlvBreakpointType` breakpoint on the same file and line through
  `XBreakpointManager.addLineBreakpoint`, or, if a Delve breakpoint already exists there, removes that one (toggle off).
- `PKBreakpointMigrationActivity` converts any persisted `piko-line` breakpoints from older plugin versions into Delve
  breakpoints on project open, keeping condition, log and suspend settings. After that the only persisted state is the
  Go plugin's own breakpoint, so there is one source of truth.

The fallback (mirroring breakpoints onto `dist/**/generated.go` and mapping frames back) is not needed.

## Limitations

- Requires the generator's `PIKO_DWARF_LINE_DIRECTIVES=true` build and a binary built with `-gcflags "all=-N -l"`
  and without `-trimpath`. With directives off, Delve has no `.pk` sources and GoLand marks the breakpoint invalid
  ("executable does not contain debug information for ...").
- Gutter eligibility is a heuristic. Statements inside closures and lines inside multi-line expressions may be
  accepted by the gutter but have no code; Delve then rejects them and GoLand shows the breakpoint as invalid.
- Dragging a Delve breakpoint to another `.pk` line is refused, because drag uses the Delve type's `canPutAt`.
  Remove and re-add it instead.
- The toggle relies on the monolithic breakpoint manager. Remote development (split frontend and backend) is untested.
- A template line in a partial binds once in every page that includes the partial; Delve reports several locations.
