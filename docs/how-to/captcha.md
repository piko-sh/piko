---
title: How to protect an action with a captcha
description: Wire a captcha provider, integrate it with a server action, and swap between Turnstile, reCAPTCHA v3, hCaptcha, and the HMAC dev backend.
nav:
  sidebar:
    section: "how-to"
    subsection: "security"
    order: 30
---

# How to protect an action with a captcha

This guide walks through adding captcha protection to a Piko server action. For the API surface see [captcha API reference](../reference/captcha-api.md). For the rationale behind the four-provider, single-verify-call design see [about captcha](../explanation/about-captcha.md).

## Register a Turnstile provider at bootstrap

Turnstile is the default recommendation. It runs without user friction, avoids a Google dependency, and costs nothing up to the free tier. Wire it in `main.go`:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/captcha/captcha_provider_turnstile"
)

func main() {
    provider, err := captcha_provider_turnstile.NewProvider(captcha_provider_turnstile.Config{
        SiteKey:   os.Getenv("TURNSTILE_SITE_KEY"),
        SecretKey: os.Getenv("TURNSTILE_SECRET_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    ssr := piko.New(
        piko.WithCaptchaProvider("turnstile", provider),
        piko.WithDefaultCaptchaProvider("turnstile"),
    )
    // ... ssr.Run(piko.RunModeProd)
}
```

The site key is public and embeds in the client widget. The secret key stays server-side and Piko forwards it to the verify endpoint.

## Call Verify from the action

The server action reads the token field from the input struct and passes it to `captcha.Verify`:

```go
package contact

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/captcha"
)

type SubmitInput struct {
    Name         string `json:"name"         validate:"required"`
    Email        string `json:"email"        validate:"required,email"`
    Message      string `json:"message"      validate:"required"`
    CaptchaToken string `json:"captchaToken" validate:"required"`
}

type SubmitResponse struct {
    Ticket string `json:"ticket"`
}

type SubmitAction struct {
    piko.ActionMetadata
}

func (a *SubmitAction) Call(input SubmitInput) (SubmitResponse, error) {
    if err := captcha.Verify(a.Ctx(), input.CaptchaToken, a.ClientIP(), "contact_submit"); err != nil {
        return SubmitResponse{}, piko.ValidationField("captchaToken", "Captcha verification failed. Please try again.")
    }

    // ... existing business logic
    return SubmitResponse{Ticket: "T-12345"}, nil
}
```

`captcha.Verify` returns one of the typed errors such as `ErrTokenMissing`, `ErrTokenExpired`, or `ErrScoreBelowThreshold`. Map the error to a user-facing field-level validation message with `piko.ValidationField`.

The third argument (`"contact_submit"` above) is the action label. Score-based providers (reCAPTCHA v3, Turnstile) use it to detect mismatched or replayed tokens.

## Embed the widget on the form

Turnstile ships a small JavaScript widget. Load it once in a layout partial or directly in the page:

```piko
<template>
  <script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>

  <form id="contact-form" p-on:submit.prevent="handleSubmit($event, $form)">
    <input type="text" name="name" required />
    <input type="email" name="email" required />
    <textarea name="message" required></textarea>

    <div
      class="cf-turnstile"
      :data-sitekey="state.SiteKey"
      data-action="contact_submit"
    ></div>

    <button type="submit">Send</button>
  </form>
</template>

<script type="application/x-go">
package main

import (
    "os"

    "piko.sh/piko"
)

type Response struct {
    SiteKey string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{SiteKey: os.Getenv("TURNSTILE_SITE_KEY")}, piko.Metadata{Title: "Contact"}, nil
}
</script>

<script lang="ts">
async function handleSubmit(event: SubmitEvent, form: FormDataHandle): Promise<void> {
    const token = form.get("cf-turnstile-response");
    if (!token) {
        return;
    }
    form.set("captchaToken", token);
    await action.contact.Submit(form).call();
}
</script>
```

The widget puts its generated token into a hidden input named `cf-turnstile-response`. The client script copies that value into the `captchaToken` field expected by the action.

## Swap in reCAPTCHA v3

The service interface is provider-agnostic. To switch, replace the provider registration:

```go
import "piko.sh/piko/wdk/captcha/captcha_provider_recaptcha_v3"

recaptcha, err := captcha_provider_recaptcha_v3.NewProvider(captcha_provider_recaptcha_v3.Config{
    SiteKey:   os.Getenv("RECAPTCHA_SITE_KEY"),
    SecretKey: os.Getenv("RECAPTCHA_SECRET_KEY"),
})
// ...
piko.WithCaptchaProvider("recaptcha", recaptcha),
piko.WithDefaultCaptchaProvider("recaptcha"),
```

The widget script URL changes (`https://www.google.com/recaptcha/api.js`), but the action code stays identical because `captcha.Verify` is the same. reCAPTCHA v3 returns a numeric confidence score, and Piko normalises it onto the same `0.0-1.0` scale used elsewhere.

hCaptcha (`captcha_provider_hcaptcha`) follows the same shape.

## Use the HMAC challenge for local development

Running a third-party captcha during development is friction. The built-in HMAC challenge is a deterministic, no-network provider useful for tests and local servers:

```go
import "piko.sh/piko/wdk/captcha/captcha_provider_hmac_challenge"

dev, err := captcha_provider_hmac_challenge.NewProvider(captcha_provider_hmac_challenge.Config{
    Secret: []byte(os.Getenv("DEV_CAPTCHA_SECRET")), // at least 16 bytes
    TTL:    5 * time.Minute,
})
// ...
piko.WithCaptchaProvider("hmac", dev),
piko.WithDefaultCaptchaProvider("hmac"),
```

The HMAC provider is not safe for production. Never expose it on a public endpoint.

## Tune the score threshold

The default threshold is `0.5`. To override, build a service with `WithDefaultScoreThreshold` and register it. See [captcha API reference](../reference/captcha-api.md) for the option list.

## Troubleshoot verification failures

`captcha.Verify` returns one of `ErrTokenMissing`, `ErrTokenExpired`, `ErrScoreBelowThreshold`, or `ErrProviderUnavailable`. See [captcha API reference](../reference/captcha-api.md) for the full error catalogue and remediation. Call the service directly instead of `captcha.Verify` when you need the underlying `VerifyResponse`.

## See also

- [Captcha API reference](../reference/captcha-api.md) for the full API surface.
- [About captcha](../explanation/about-captcha.md) for score normalisation, why four providers ship, and composition with rate limiting.
- [How to security](security.md) for the wider CSRF and rate-limit context.
- [Scenario 030: CAPTCHA-protected action](../../examples/scenarios/030_captcha/) for a runnable example.
