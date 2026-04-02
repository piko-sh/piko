# Profile Chain Cascade Analysis

## Real .pkc Profile Chain

For a `.pkc` file, `GetProfilesForFile` returns 4 profiles:

1. `compiled_js` - compile-component, depends on "source", Priority NEED
2. `minified` - minify-js, depends on "compiled_js", Priority WANT
3. `gzip` - compress-gzip, depends on "minified", Priority WANT
4. `br` - compress-brotli, depends on "minified", Priority WANT

Source: `internal/lifecycle/lifecycle_domain/build_profiles.go:150-174`

## Cascading Dispatch Flow

### Step 1: Initial Event (UpsertArtefact -> EventArtefactCreated)
- Bridge evaluates profiles: only "source" variant exists and is READY
- `compiled_js` depends on "source" -> dependency met -> DISPATCH
- `minified` depends on "compiled_js" -> not ready -> SKIP
- `gzip` depends on "minified" -> not ready -> SKIP
- `br` depends on "minified" -> not ready -> SKIP
- **Dispatched: 1** (compiled_js)
- **Published: 1, Processed: 1**

### Step 2: compiled_js Completes -> AddVariant -> EventArtefactUpdated
- Bridge re-evaluates: "source" READY, "compiled_js" READY
- `compiled_js` -> already READY -> SKIP
- `minified` depends on "compiled_js" -> READY -> DISPATCH
- `gzip` depends on "minified" -> not ready -> SKIP
- `br` depends on "minified" -> not ready -> SKIP
- **Dispatched: 2** (compiled_js + minified)
- **Published: 2, Processed: 2**

### Step 3: minified Completes -> AddVariant -> EventArtefactUpdated
- Bridge re-evaluates: "source", "compiled_js", "minified" all READY
- `minified` -> already READY -> SKIP
- `gzip` depends on "minified" -> READY -> DISPATCH
- `br` depends on "minified" -> READY -> DISPATCH
- **Dispatched: 4** (all four)
- **Published: 3, Processed: 3**

### Step 4: gzip + br Complete -> 2x AddVariant -> 2x EventArtefactUpdated
- Bridge processes each: all variants READY, no new dispatches
- **Dispatched: 4, Completed: 4**
- **Published: 5, Processed: 5**

## Potential Race Condition

Between a task completing (HandleTaskSuccess -> completed++) and the bridge
dispatching the next task in the chain, there's a brief window where:

- dispatched == completed (the just-completed task balances)
- InFlightCount == 0 (worker has finished)
- pending == 0, delayed == 0
- **IsIdle() returns TRUE** even though more work should be dispatched

The EventArtefactUpdated event is published INSIDE executor.Execute(), BEFORE
HandleTaskSuccess is called. But event delivery is asynchronous - the bridge
may not have processed it yet when the build runner polls IsIdle().

### Phase 1 Impact
Phase 1 (flush) can pass EARLY at Step 1: processed=1 >= published=1.
After that, Phase 1 never runs again. New AddVariant events increment published
but Phase 1 has already completed.

### Phase 2 Impact
Phase 2 (IsIdle) may see true in the gap between task completion and
bridge dispatching the downstream task. This would cause PREMATURE completion
(not a hang).

### But the User Reports a HANG
If the issue is premature completion, the build would finish with missing
profiles. If it's a hang, something else is going on:

1. **Exponential backoff exceeds timeout**: With maxRetries=3 and
   10^attempt second backoff (10s, 100s, 1000s), a task that fails on
   attempt 2 schedules retry 3 for ~1000s. The 5-minute timeout fires
   before the retry, causing a timeout error (which looks like a hang).

2. **Phase 1 never completes**: If AddVariant publishes events that
   the bridge never processes (e.g., subscription issues, event bus
   problems), published > processed forever.

3. **InFlightCount never reaches 0**: If a worker goroutine is stuck
   (e.g., executor hangs), InFlightCount > 0 forever.

## What Our Tests Don't Cover

- **Profile chain cascading**: All tests use single profiles
- **Multiple AddVariant events per artefact**: Tests don't exercise the
  scenario where one artefact generates 4+ AddVariant events
- **Bridge re-evaluation on EventArtefactUpdated**: Tests don't verify
  that the bridge correctly dispatches downstream profiles
- **Race between IsIdle and cascading dispatch**: Tests don't check if
  IsIdle briefly returns true between cascading steps

## Key Files

- Profile chain: `internal/lifecycle/lifecycle_domain/build_profiles.go`
- Bridge dispatch: `internal/orchestrator/orchestrator_adapters/driving_artefact_workflow_bridge.go:572-726`
- evaluateAndDispatchProfile: line 637-668
- findMissingDependencies: line 614-623
