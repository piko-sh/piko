# Counter Analysis

## IsIdle() Four-Part Check

The dispatcher's `IsIdle()` returns true when ALL of:
1. No pending tasks (`pendingTasks == 0`)
2. Dispatched <= completed + failed + retried (`dispatched <= completed + failed + retried`)
3. No delayed tasks (`DelayedPublisher.PendingCount() == 0`)
4. No in-flight tasks (`InFlightCount() == 0`)

## Counter Flow for Always-Failing Executor (maxRetries=2)

1. `Dispatch()`: dispatched=1, pending=1
2. Attempt 1 fails -> `ScheduleTaskRetry`: retried=1, `DelayedPublisher` schedules delayed re-dispatch
3. `DelayedPublisher` fires -> calls `d.Dispatch()` again: dispatched=2, pending=1
4. Attempt 2 fails -> `MarkTaskFailed`: failed=1
5. **Final: dispatched=2, completed=0, failed=1, retried=1** -> 2 <= (0+1+1)=2 -> **IsIdle=true**

## Counter Flow for Successful Executor (single profile)

1. `Dispatch()`: dispatched=1, pending=1
2. Attempt 1 succeeds -> `HandleTaskSuccess`: completed=1
3. **Final: dispatched=1, completed=1, failed=0, retried=0** -> 1 <= (1+0+0)=1 -> **IsIdle=true**

## Two-Phase Idle Detection (Build Runner)

**Phase 1 (Pipeline Flush):**
- Polls: `bridge.ArtefactEventsProcessed() >= registry.ArtefactEventsPublished()`
- 50ms poll interval
- Short-circuits if published == 0

**Phase 2 (Dispatcher Idle):**
- Polls: `dispatcher.IsIdle()`
- 50ms poll interval
- 5-minute timeout

## Key Locations

- IsIdle: `internal/orchestrator/orchestrator_adapters/driven_taskdispatcher_watermill.go`
- Counter increments: `internal/orchestrator/orchestrator_domain/task_processing_core.go`
- DelayedPublisher: `internal/orchestrator/orchestrator_domain/delayed_publisher.go`
- Build runner phases: `internal/lifecycle/lifecycle_adapters/driver_build_runner.go:324-414`
