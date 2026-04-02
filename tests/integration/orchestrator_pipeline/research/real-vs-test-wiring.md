# Real System vs Test Harness Wiring

## Real System (Bootstrap)

### Initialisation Order

1. `initialiseOrchestratorEarly()` is called BEFORE generation starts
2. `GetOrchestratorService()` creates:
   - Task store, event bus, dispatcher, bridge
   - `bridge.StartListening()` blocks until subscriptions established
   - Orchestrator goroutine started
3. Generation runs (annotations, code emission, artefact writing)
4. `buildStaticAssets()` calls `RunBuild()` which:
   - Walks directories and calls `UpsertArtefact()` for each file
   - Calls `waitUntilIdle()` with two-phase detection

### Key Files

- Bootstrap entry: `internal/bootstrap/build.go:111-152`
- Early init: `internal/bootstrap/build.go:163-171`
- Container wiring: `internal/bootstrap/container_orchestrator.go:143-298`
- Build service: `internal/lifecycle/lifecycle_adapters/driver_build_runner.go`

### Bridge Context

- Created with `c.GetAppContext()` (application-level context)
- Separate from the build context passed to `RunBuild()`
- Bridge goroutine (`go wait()`) runs until app context is cancelled

## Test Harness

### Initialisation Order

1. Create event bus (GoChannel provider)
2. Create dispatcher with config
3. Create bridge with `NewArtefactWorkflowBridge()`
4. `bridge.StartListening()` blocks until subscriptions established
5. Start dispatcher in goroutine
6. Tests seed artefacts via `seedArtefact()` or `seedArtefactWithContent()`

### Differences

| Aspect | Real System | Test Harness |
|--------|------------|--------------|
| Event bus | NATS or GoChannel via container | GoChannel directly |
| Bridge context | App context (long-lived) | Test context (cancelled on cleanup) |
| Profile assignment | `GetProfilesForFile()` - chains! | `makeProfile()` - single profile |
| Storage backend | Real disk/S3 backends | None configured |
| Multiple profiles | compile -> minify -> gzip -> brotli | Just one at a time |

## CRITICAL DIFFERENCE: Profile Chains

The real system assigns MULTIPLE CHAINED profiles per file via `GetProfilesForFile()`.
A `.pkc` file gets:
- compiled (compile-component) -> produces JS variant
- minified (minify) -> depends on compiled variant
- gzip compressed -> depends on minified variant
- brotli compressed -> depends on minified variant

Each successful task triggers AddVariant -> EventArtefactUpdated -> bridge processes
-> potentially dispatches NEXT profile in chain.

Our tests only assign ONE profile per artefact. The chain behaviour is NEVER tested.
