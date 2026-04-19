---
title: "030: CAPTCHA-protected action"
description: Protect a server action with CAPTCHA verification using the configured provider.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 500
---

# 030: CAPTCHA-protected actions

A gallery of CAPTCHA-protected actions, one page per provider. Each page demonstrates a working CAPTCHA flow (pass case) and, for most providers, the failure case that the action rejects.

## What this demonstrates

- `WithCaptchaProvider` with each supported provider implementation.
- An action that implements `piko.CaptchaProtected` and declares its `CaptchaConfig`.
- The `<piko:captcha />` template tag that renders the CAPTCHA widget.
- Automatic token verification by the action-dispatch layer.
- Graceful failure: an invalid token returns HTTP 400 with a user-visible message.

## Project structure

```text
src/
  cmd/main/main.go      Bootstrap with WithCaptchaProvider for each provider.
  pages/
    index.pk            Navigation to every provider page.
    hmac.pk             HMAC provider demo.
    turnstile-pass.pk   Cloudflare Turnstile, pass case.
    turnstile-fail.pk   Cloudflare Turnstile, fail case.
    hcaptcha-pass.pk    hCaptcha, pass case.
    hcaptcha-fail.pk    hCaptcha, fail case.
    recaptcha-pass.pk   Google reCAPTCHA, pass case.
  actions/              One CAPTCHA-protected submission action per provider.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/030_captcha/src/
go mod tidy
air
```

Set each provider's site and secret keys via environment variables (or local secrets configuration) before starting. The index page links to every provider demo.

## See also

- [Server actions reference](../reference/server-actions.md).
- [Bootstrap options reference: Captcha](../reference/bootstrap-options.md#captcha).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/030_captcha).
