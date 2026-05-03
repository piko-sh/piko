---
title: How to run background tasks
description: Patterns for one-shot startup work, periodic jobs, and post-request follow-ups using lifecycle components and the clock facade.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 110
---

# How to run background tasks

Combine the lifecycle and clock facades with ordinary goroutines for one-shot startup work, periodic jobs, and post-request follow-ups. No public dispatcher facade ships yet. See [lifecycle API reference](../reference/lifecycle-api.md) and [clock API reference](../reference/clock-api.md) for the surface.

## Run a one-shot task at startup

Put the work inside a `LifecycleComponent.OnStart`. Piko calls it once during managed startup, before serving traffic, and a non-nil return aborts boot.

```go
package components

import (
    "context"
    "piko.sh/piko"
)

type WarmCacheComponent struct{ store CacheStore }

func (c *WarmCacheComponent) Name() string { return "warm-cache" }

func (c *WarmCacheComponent) OnStart(ctx context.Context) error {
    return c.store.Prefetch(ctx, popularKeys())
}

func (c *WarmCacheComponent) OnStop(ctx context.Context) error { return nil }
```

Register it on the server at bootstrap.

```go
ssr := piko.New()
ssr.RegisterLifecycle(&components.WarmCacheComponent{store: store})
```

See [how to register a lifecycle component](lifecycle.md) for the full interface, the start timeout, and health probes.

## Run a periodic task

Spawn a goroutine in `OnStart` that ranges over a `clock.Ticker`. Cancel it from `OnStop` so shutdown is clean. Always take the clock from the facade instead of calling `time.NewTicker` directly, so tests can advance virtual time.

```go
package components

import (
    "context"
    "time"

    "piko.sh/piko/wdk/clock"
)

type NightlyReport struct {
    clock  clock.Clock
    cancel context.CancelFunc
    done   chan struct{}
}

func (c *NightlyReport) Name() string { return "nightly-report" }

func (c *NightlyReport) OnStart(ctx context.Context) error {
    runCtx, cancel := context.WithCancelCause(context.Background())
    c.cancel = func() { cancel(context.Canceled) }
    c.done = make(chan struct{})

    ticker := c.clock.NewTicker(1 * time.Hour)
    go func() {
        defer close(c.done)
        defer ticker.Stop()
        for {
            select {
            case <-runCtx.Done():
                return
            case <-ticker.C():
                c.runOnce(runCtx)
            }
        }
    }()
    return nil
}

func (c *NightlyReport) OnStop(ctx context.Context) error {
    c.cancel()
    select {
    case <-c.done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (c *NightlyReport) runOnce(ctx context.Context) { /* report */ }
```

For "every Tuesday at 03:00", check `c.clock.Now()` inside `runOnce` and skip when the wall clock does not match. There is no cron parser in `wdk/`. For coarser external scheduling, point a system cron at an authenticated action endpoint.

## Defer work after a request returns

Some follow-up work is best-effort, such as analytics, audit logs, or cache priming. When the action's response can ship immediately, spawn a goroutine with a fresh context that survives the request. Do not pass the request context. Piko cancels it once it writes the response.

```go
package main

import (
    "context"
    "errors"
    "time"

    "piko.sh/piko"
    "piko.sh/piko/wdk/logger"
)

type RegisterInput struct {
    Email  string `json:"email"`
    UserID string `json:"userId"`
}

type RegisterResponse struct {
    OK bool `json:"ok"`
}

type RegisterAction struct {
    piko.ActionMetadata
    users     UserStore
    analytics AnalyticsRecorder
    log       logger.Logger
}

func (a *RegisterAction) Call(input RegisterInput) (RegisterResponse, error) {
    ctx := a.Ctx()
    if err := a.users.Create(ctx, input); err != nil {
        return RegisterResponse{}, err
    }

    bgCtx, cancel := context.WithTimeoutCause(
        context.WithoutCancel(ctx),
        30*time.Second,
        errors.New("register: post-response work timed out"),
    )
    go func() {
        defer cancel()
        runCtx, log := logger.From(bgCtx, a.log)
        if err := a.analytics.Record(runCtx, "user_registered", input.UserID); err != nil {
            log.Warn("post-response analytics failed", logger.Error(err))
        }
    }()

    return RegisterResponse{OK: true}, nil
}
```

Use `context.WithoutCancel(ctx)` to retain request values (request ID, locale) without inheriting the cancellation. Always add a timeout cause. Capture the context returned by `logger.From(ctx, log)`.

This pattern has no retry and dies with the process. For durable work (email batches, uploads, flaky third-party APIs), drive a lifecycle-managed worker that polls a table or queue you own.

## See also

- [How to register a lifecycle component](lifecycle.md) for `OnStart` and `OnStop` mechanics.
- [Lifecycle API reference](../reference/lifecycle-api.md) for the interfaces.
- [Clock API reference](../reference/clock-api.md) for `Clock`, `Ticker`, and the mock used in tests.
- [How to notifications](notifications.md) for sending out-of-band messages from a background worker.
- [How to email templates](email-templates.md) for rendering the messages those workers send.
