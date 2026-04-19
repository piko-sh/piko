---
title: Server actions and forms
description: Build a validated contact form, then extend it with a streaming progress action.
nav:
  sidebar:
    section: "tutorials"
    subsection: "getting-started"
    order: 30
---

# Server actions and forms

In this tutorial we will build a contact form with validation, typed responses, and client-side feedback. We then extend the pattern with a long-running action that streams progress back to the browser. The finished project mirrors scenarios [002](../showcase/002-contact-form.md) and [010](../showcase/010-progress-tracker.md).

<p align="center">
  <img src="../diagrams/tutorial-03-preview.svg"
       alt="Preview of the finished contact form: name, email with an inline validation error, and message fields; a Send button beside a progress bar showing 62 percent during a streaming action."
       width="500"/>
</p>

You should have completed [Adding interactivity](02-adding-interactivity.md) first.

## Step 1: Create the action file

Actions live under `actions/`. The directory name maps to the URL segment, and the struct name maps to the action method. [about the action protocol](../explanation/about-the-action-protocol.md) explains why actions use typed RPC instead of free-form HTTP handlers. [server actions reference](../reference/server-actions.md) documents the full API. Create `actions/contact/submit.go`:

> **Note:** Three things make an action discoverable: it lives under `actions/<package>/`, it embeds `piko.ActionMetadata`, and the struct name ends in `Action`. The generator reads those three signals to mount `actions/contact/submit.go`'s `SubmitAction` at `/actions/contact.Submit`; missing any one means the action silently does not register.

```go
package contact

import (
    "piko.sh/piko"
)

type SubmitInput struct {
    Name    string `json:"name"    validate:"required,min=2"`
    Email   string `json:"email"   validate:"required,email"`
    Message string `json:"message" validate:"required,min=10"`
}

type SubmitResponse struct {
    Ticket string `json:"ticket"`
}

type SubmitAction struct {
    piko.ActionMetadata
}

func (a *SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
    // Imagine this persists to a database and returns a ticket ID.
    return SubmitResponse{Ticket: "T-12345"}, nil
}
```

Run `piko generate` to regenerate the dispatch code:

```bash
piko generate
```

## Step 2: Create the form page

Create `pages/contact.pk`:

```piko
<template>
    <piko:partial is="layout" :server.page_title="'Contact us'">
        <h1>Contact us</h1>

        <form id="contact-form" p-on:submit.prevent="handleSubmit($event, $form)" class="contact-form">
            <label>
                Your name
                <input type="text" name="name" required minlength="2" />
            </label>

            <label>
                Email
                <input type="email" name="email" required />
            </label>

            <label>
                Message
                <textarea name="message" required minlength="10"></textarea>
            </label>

            <button type="submit">Send</button>
        </form>
    </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "Contact us"}, nil
}
</script>

<script lang="ts">
async function handleSubmit(event: SubmitEvent, form: FormDataHandle): Promise<void> {
    try {
        const data = await action.contact.Submit(form).call();
        console.log("Submitted. Ticket:", data.ticket);
    } catch (err) {
        console.error("Submit failed", err);
    }
}
</script>
```

Visit `http://localhost:8080/contact`, fill in the fields, and submit. Open the browser's console: `Submitted. Ticket: T-12345` logs. Piko handled the POST, CSRF token, JSON parsing, and validation tags in one round-trip. See [server actions reference](../reference/server-actions.md#call-signature) for the exact contract between `Call` and the client runtime.

## Step 3: Add field-level validation

Update the action to reject invalid emails and report them against the `email` field:

```go
package contact

import (
    "net/mail"

    "piko.sh/piko"
)

type SubmitInput struct {
    Name    string `json:"name"    validate:"required,min=2"`
    Email   string `json:"email"   validate:"required,email"`
    Message string `json:"message" validate:"required,min=10"`
}

type SubmitResponse struct {
    Ticket string `json:"ticket"`
}

type SubmitAction struct {
    piko.ActionMetadata
}

func (a *SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
    // The `validate` tags on SubmitInput have already run.
    // This block is for rules the tags cannot express:
    if _, err := mail.ParseAddress(input.Email); err != nil {
        return SubmitResponse{}, piko.ValidationField("email", "Enter a valid email address.")
    }

    ticket := generateTicket()

    a.Response().AddHelper("showToast", "Message sent. We will reply shortly.", "success")

    return SubmitResponse{Ticket: ticket}, nil
}

func generateTicket() string {
    // Call your ticketing system here.
    return "T-12345"
}
```

Enable the toast module at bootstrap so the queued toast renders:

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleToasts),
)
```

Run the server and submit with a malformed email (`not-an-email`). A red message appears under the email input. Submit with valid data and a green toast slides in reading "Message sent. We will reply shortly." See [errors reference](../reference/errors.md) for the full list of validation and action-error constructors.

## Step 4: Handle unexpected errors

Validation errors are for user-correctable input. Internal errors (database down, email provider rejecting) must stay hidden from the user, while the server records them in full. Use `piko.NewError` or `piko.Errorf`:

```go
ticket, err := ticketing.Create(a.Ctx(), name, email, message)
if err != nil {
    return SubmitResponse{}, piko.Errorf(
        "Sorry, we could not submit your message. Try again shortly.",
        "creating ticket for %s: %w", email, err,
    )
}
```

The user sees the first string. The internal logs record the formatted second string with the wrapped cause. In development mode, Piko surfaces the internal detail to the browser too.

## Step 5: Add a streaming action

A streaming action implements an optional `StreamProgress` method alongside `Call`. For the full signature, cancellation, timeouts, and reconnection see [how to streaming with SSE](../how-to/actions/streaming-with-sse.md). Create `actions/task/process.go`:

```go
package task

import (
    "fmt"
    "time"

    "piko.sh/piko"
)

type ProcessInput struct {
    JobID string `json:"jobID" validate:"required"`
}

type ProcessResponse struct {
    JobID string `json:"job_id"`
    Total int    `json:"total"`
}

type ProcessAction struct {
    piko.ActionMetadata
    Input ProcessInput
}

func (a *ProcessAction) Call(input ProcessInput) (ProcessResponse, error) {
    a.Input = input
    return ProcessResponse{JobID: input.JobID, Total: 10}, nil
}

func (a *ProcessAction) StreamProgress(stream *piko.SSEStream) error {
    total := 10

    for i := 1; i <= total; i++ {
        if err := stream.Send("progress", map[string]any{
            "done":  i,
            "total": total,
            "label": fmt.Sprintf("Processing step %d of %d", i, total),
        }); err != nil {
            return err
        }

        select {
        case <-a.Ctx().Done():
            return a.Ctx().Err()
        case <-time.After(400 * time.Millisecond):
        }
    }

    return stream.SendComplete(ProcessResponse{JobID: a.Input.JobID, Total: total})
}
```

Run `piko generate` again to pick up the new action.

## Step 6: Consume the stream on the client

Create `components/pp-progress.pkc`:

```pkc
<template name="pp-progress">
    <div class="progress">
        <button p-if="state.status === 'idle'" p-on:click="start">
            Start
        </button>

        <div p-if="state.status === 'running' || state.status === 'done'">
            <progress :value="state.done" :max="state.total"></progress>
            <p>{{ state.label }}</p>
        </div>

        <p p-if="state.status === 'done'">Complete! Job: {{ state.jobId }}</p>
        <p p-if="state.status === 'error'" class="error">{{ state.error }}</p>
    </div>
</template>

<script lang="ts">
    const state = {
        status: 'idle' as 'idle' | 'running' | 'done' | 'error',
        done: 0 as number,
        total: 0 as number,
        label: '' as string,
        jobId: '' as string,
        error: '' as string,
    };

    async function start() {
        state.status = 'running';
        state.done = 0;
        state.label = 'Starting...';

        try {
            const result = await action.task.Process({ jobID: crypto.randomUUID() })
                .withOnProgress((data: unknown, eventType: string) => {
                    const event = data as { done: number; total: number; label: string };
                    state.done = event.done;
                    state.total = event.total;
                    state.label = event.label;
                })
                .call();

            state.status = 'done';
            state.jobId = result.job_id;
            state.total = result.total;
            state.done = result.total;
        } catch (err) {
            state.status = 'error';
            state.error = (err as Error).message;
        }
    }
</script>

<style>
    progress { width: 100%; height: 1rem; }
    .error { color: #b91c1c; }
</style>
```

## Step 7: Drop the component on a page

Create `pages/progress.pk`:

```piko
<template>
    <piko:partial is="layout" :server.page_title="'Progress demo'">
        <h1>Progress demo</h1>
        <p>Click Start to begin a long-running task.</p>
        <pp-progress />
    </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "Progress demo"}, nil
}
</script>
```

Visit `http://localhost:8080/progress` and click Start. The progress bar advances in ten steps of roughly 400 ms each. The label underneath updates with every step. When the run finishes, "Complete. Job: &lt;uuid&gt;" replaces the running label. [how to streaming with SSE](../how-to/actions/streaming-with-sse.md) documents the `withOnProgress(...).call()` builder.

## Next steps

- [Shipping a real site](04-shipping-a-real-site.md): assemble the pieces into a project you can deploy.
- [Server actions reference](../reference/server-actions.md) for the full API.
- [How to forms](../how-to/actions/forms.md) and [how to streaming with SSE](../how-to/actions/streaming-with-sse.md).
- [Scenario 002](../showcase/002-contact-form.md) and [Scenario 010](../showcase/010-progress-tracker.md) for the runnable source.
