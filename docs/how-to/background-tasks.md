---
title: How to run background tasks
description: Queue, retry, and observe background work using the orchestrator service.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 110
---

# How to run background tasks

Piko ships an orchestrator service that queues, runs, retries, and observes background work. Use it for anything an HTTP request should not wait on. Typical examples include sending an email batch, processing an upload, calling an external API with retry, or running a nightly report. The integration tests under [`tests/integration/orchestrator_pipeline`](https://github.com/piko-sh/piko/tree/master/tests/integration/orchestrator_pipeline) exercise the service.

## When to reach for it

Use the orchestrator when:

- The work is slow enough that a user should not wait (seconds or minutes).
- The work must run even if the caller aborts the original request.
- The work should retry on transient failure.
- You want a single place to observe queued, running, and failed tasks (via `piko get tasks` or the TUI).

Use a simple goroutine when the work is fast and best-effort (for example, fire-and-forget analytics logging).

## Register the orchestrator

Piko wires the orchestrator by default. Override the implementation if you need a custom queue backend:

```go
ssr := piko.New(
    piko.WithOrchestratorService(myCustomOrchestrator),
)
```

The default implementation uses an in-process queue with at-least-once delivery and exponential-backoff retry. For production, wire a durable queue as shown below.

## Declare a task type

A task is a Go struct that implements the orchestrator's task interface. The struct carries the task's inputs, and its `Run` method performs the work.

```go
package tasks

import (
    "context"
    "fmt"

    "piko.sh/piko"
)

type SendWelcomeEmail struct {
    UserID    int64
    UserEmail string
}

func (t SendWelcomeEmail) Run(ctx context.Context) error {
    svc := piko.GetEmailService()
    return svc.Send(ctx, buildWelcomeMessage(t.UserID, t.UserEmail))
}

func (t SendWelcomeEmail) Name() string {
    return "send-welcome-email"
}
```

Register the task type at startup so the orchestrator knows how to decode it from persistent storage:

```go
ssr.RegisterTask(tasks.SendWelcomeEmail{})
```

## Enqueue from an action

```go
func (a *SignupAction) Call(input SignupInput) (SignupResponse, error) {
    user, err := createUser(a.Ctx(), input)
    if err != nil {
        return SignupResponse{}, piko.NewError("signup failed", err)
    }

    piko.GetOrchestrator().Enqueue(a.Ctx(), tasks.SendWelcomeEmail{
        UserID:    user.ID,
        UserEmail: user.Email,
    })

    return SignupResponse{UserID: user.ID}, nil
}
```

`Enqueue` returns as soon as the queue accepts the task, and the action response goes out immediately. The orchestrator's worker pool picks up the task on its own schedule.

## Retry and backoff

Return an error from `Run` to signal a retry:

```go
func (t SendWelcomeEmail) Run(ctx context.Context) error {
    if err := sendEmail(ctx, t); err != nil {
        return fmt.Errorf("send email: %w", err)   // retries with backoff
    }
    return nil
}
```

The orchestrator applies exponential backoff up to a configurable maximum attempt count. Once the task exhausts the budget, it lands in the dead-letter queue.

## Dead-letter queue

The orchestrator parks tasks that exceed their retry budget in a dead-letter queue (DLQ). Inspect with `piko get dlq` or the TUI. You can re-enqueue a DLQ'd task once you resolve the underlying problem.

## Schedule work

For periodic work, implement a scheduled task:

```go
type CleanupExpiredSessions struct{}

func (t CleanupExpiredSessions) Run(ctx context.Context) error {
    return sessions.DeleteExpired(ctx)
}

func (t CleanupExpiredSessions) Schedule() string {
    return "0 */15 * * * *"  // Every 15 minutes.
}
```

The orchestrator enqueues scheduled tasks on the declared cron expression.

## Observe tasks

Pike's CLI exposes the orchestrator over the gRPC monitoring endpoint:

```bash
piko get tasks                 # Recent tasks.
piko get tasks -n 50           # Last 50.
piko watch tasks --interval 2s # Live stream.
piko describe task <id>        # Full detail for one task.
piko get dlq                   # Dead-letter queue.
```

See the [CLI reference](../reference/cli.md) for all monitoring commands.

## Durable queue backends

For production, register a durable queue (Postgres-backed, Redis Streams, SQS) so enqueued tasks survive process restarts:

```go
ssr := piko.New(
    piko.WithOrchestratorService(queue.NewPostgresOrchestrator(db)),
)
```

Exact adapter implementations vary. Check the `adapters/orchestrator` package for what ships.

## See also

- [CLI reference](../reference/cli.md) for the task-inspection commands.
- [How to notifications](notifications.md) for reliable notification delivery.
- [How to email templates](email-templates.md).
- Integration tests: [`tests/integration/orchestrator_pipeline`](https://github.com/piko-sh/piko/tree/master/tests/integration/orchestrator_pipeline).
