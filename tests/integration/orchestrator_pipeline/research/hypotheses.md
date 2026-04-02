# Hang Bug Hypotheses

## Status of Each Hypothesis

### H1: Counter Mismatch in IsIdle() [RULED OUT]
- The IsIdle() counter (dispatched <= completed + failed + retried) was fixed in prior sessions
- All integration tests pass with correct counter tracking
- Both mock clock and real clock tests verify this

### H2: Retry Timer Prevents Idle Detection [RULED OUT]
- Tests with real clock and exponential backoff (10s delay) pass correctly
- DelayedPublisher.PendingCount() is correctly checked in IsIdle()

### H3: Real Executor Triggers Different Behaviour [PARTIALLY EXPLORED]
- TestLifecycle_RealCompiler_InvalidPKC_FullRoundTrip: compilation SUCCEEDS on invalid PKC
- The `const v = 1; const v = 2;` error does NOT cause a capability failure
- The error in the real system may be "storage backend '' not configured" not a compilation error
- This changes the diagnosis: the hang may be on the SUCCESS path

### H4: Secondary Events from AddVariant [CONFIRMED WORKING IN ISOLATION]
- TestLifecycle_RealCompiler_ValidPKC_SuccessRoundTrip confirms:
  Published=2, Processed=2 for single artefact with one profile
- AddVariant correctly publishes secondary EventArtefactUpdated
- Bridge correctly processes it and increments counter

### H5: Lifecycle Observer Bug [ACTIVE INVESTIGATION]
The user suggests "maybe the issue isnt with the system but its with the observer".
- The build runner's `waitUntilIdle()` uses the same counters as our tests
- But the real system may have timing differences we haven't captured
- Key area: what happens when Phase 1 completes but Phase 2 never does?

### H6: Profile Chain Cascading [UNTESTED - HIGH PRIORITY]
**This is the most likely unexplored cause.**
- Real `.pkc` files get 4+ chained profiles: compile -> minify -> gzip -> brotli
- Each successful profile triggers AddVariant -> new event -> bridge may dispatch next profile
- Our tests only assign ONE profile per artefact
- The cascading chain could:
  - Cause counter imbalance if downstream tasks are dispatched from secondary events
  - Create a scenario where flush completes but dispatcher never idles
  - Trigger deduplication issues with same-artefact multiple-profile dispatch
  - Have timing issues between secondary event processing and new task dispatch

### H7: Bridge Processes Secondary Event and Dispatches New Tasks [UNTESTED]
When AddVariant fires EventArtefactUpdated:
- The bridge's `handleEvent()` receives it
- Does it call `processArtefactEvent()` which calls `dispatchProfileTask()` again?
- If the artefact now has NEW desired profiles whose dependencies are satisfied,
  new tasks could be dispatched from the secondary event handler
- This creates a cascading chain that our single-profile tests never exercise

### H8: Singleflight Deduplication with Event Type [UNTESTED]
The bridge uses singleflight keyed on artefactID + eventType:
- EventArtefactCreated and EventArtefactUpdated are different event types
- But within the same type, concurrent events for the same artefact are deduplicated
- Could rapid AddVariant events cause a singleflight to skip processing?

## RESOLVED: Root Cause Identified

### Finding: The "Hang" is Exponential Backoff During Minification Retry

Running the real `generator-output` command revealed:

1. **compile-component SUCCEEDS** on the invalid PKC (const v=1; const v=2;)
2. compiled_js variant is created with the error still in the JS
3. minified profile is dispatched (depends on compiled_js which is READY)
4. **minify-js FAILS** - tdewolff minifier rejects the duplicate declaration
5. minified is retried with exponential backoff: 10s, 100s, ...
6. The build runner waits ~100s for the second retry to fire
7. After attempt 3 fails, the build completes with "1 failed task"

**The "hang" is actually a ~100s+ wait for the exponential retry backoff.**

### Evidence from Real Build Logs

```
tasksDispatched=11 tasksCompleted=9 tasksFailed=0  (stuck here ~100s)
```

The error is in the STORAGE layer during minification:
```
capability=minify-js profile=minified
error="storage provider put failed: identifier v has already been declared"
```

### Counter Analysis (Real Build)

- Total dispatched: 12 (9 normal + compiled_js + 3x minified dispatches)
- Total completed: 10 (9 normal + compiled_js)
- Total retried: 2
- Total failed: 1 (minified after 3 attempts)
- gzip + br: never dispatched (blocked by minified not being READY)
- After final retry: 12 <= (10+1+2) = 13 → IsIdle = TRUE

### Root Causes

1. **compile-component doesn't validate JS output** - passes through invalid code
2. **Exponential backoff is 10^attempt seconds** - attempt 2 waits ~100s
3. **The failure is deterministic** - retrying won't fix a syntax error
4. **No "fail fast" for deterministic errors** - all retries are attempted

### User's Observation Was Correct

"maybe the issue isnt with the system but its with the observer of the system"
- The system works correctly (counters balance, tasks complete/fail properly).
The issue is that the exponential backoff makes the build APPEAR to hang for
minutes. With maxRetries=3, the total wait could be 10+100 = ~110s before the
build can report the failure.

## Key Insight

The invalid PKC (`const v = 1; const v = 2;`) does NOT cause a compilation error.
The capability SUCCEEDS. The error surfaces at the MINIFICATION step when
tdewolff tries to parse the compiled JS. This means:
- The real production "hang" is on the FAILURE path of the DOWNSTREAM profile
- The cascading profile chain works correctly
- The exponential backoff is the perceived "hang"

## Confirmed: Build Does Complete (Not Truly Infinite)

Ran the real `generator-output` command twice with clean slate on 007_todo_app.
Both runs completed after ~110s of retry backoff:

### Timeline (Run 2, 2026-02-27)
```
06:56:33 - Attempt 1 fails (minify-js, "identifier v has already been declared")
06:56:33 - Retry 1 scheduled: +10.4s (backoff = 10^1)
06:56:43 - Attempt 2 fails (same deterministic error)
06:56:43 - Retry 2 scheduled: +1m40.5s (backoff = 10^2 = 100s)
06:58:24 - Attempt 3 fails → MarkTaskFailed (3 >= maxRetries=3)
06:58:24 - Build completes: "1 failed task out of 12 dispatched"
```

### WAL State After Build
```json
"pp-todo-list.pkc": {
  "desiredProfiles": ["compiled_js", "minified", "gzip", "br"],
  "actualVariants": ["source", "compiled_js"]  // minified never succeeded
}
```

### The Real Problem: No Fatal Error Classification

The system has **no mechanism for executors to signal a permanent/non-retryable error**.
`HandleTaskFailure` only checks `attempt < maxRetries` - the error content is never
inspected. This means:

- **Deterministic errors** (parse failures, syntax errors, invalid input) get the
  same exponential backoff treatment as **transient errors** (network timeouts,
  resource exhaustion)
- A provably impossible task wastes ~110s on pointless retries
- During this time the build appears completely hung with no progress
- The counters and idle detection work correctly - the issue is purely wasted time

### Design Gap in Task Processing

`task_processing_core.go:340-358`:
```go
func (c *TaskProcessingCore) HandleTaskFailure(...) {
    // Only checks attempt count, never inspects the error
    if !shouldRetryTask(task.Attempt, task.Config.MaxRetries, c.Config.DefaultMaxRetries) {
        c.MarkTaskFailed(...)
        return
    }
    c.ScheduleTaskRetry(...)  // Always retries if under max
}
```

There is no `NonRetryableError`, no `FatalError` sentinel, and no error classification
in the entire orchestrator domain. The `TaskExecutor` interface returns a plain `error`
with no way to signal retryability.

### Phase 1 vs Phase 2 Timeout Gap

- **Phase 1** (`waitForPipelineFlush`): NO timeout - polls `processed >= published` forever
- **Phase 2** (`waitForDispatcherIdle`): 5-minute timeout via `checkBuildTimeout`
- In the current scenario, Phase 1 completes quickly, Phase 2 waits for retries
- But if Phase 1 ever stalled (e.g., lost event), the build truly WOULD hang forever
  because Phase 2's timeout would never get a chance to fire
