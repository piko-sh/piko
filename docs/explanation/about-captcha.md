---
title: About captcha
description: Score normalisation, the four-provider shape, HMAC as a development backstop, and composition with rate limiting and CSRF.
nav:
  sidebar:
    section: "explanation"
    subsection: "security"
    order: 30
---

# About captcha

Captchas guard forms and actions against automated abuse. Piko ships a small captcha service that sits in front of four providers and normalises their signals into one shape. The choices behind the service, the four providers it ships, and the composition with the rest of the security stack are deliberate. This page explains them.

## The one-function API

Application code rarely cares which captcha vendor is in use. It cares whether a request came from a human. The API reflects this:

```go
err := captcha.Verify(ctx, token, remoteIP, action)
```

One call. Three arguments. One error.

The absence of a response type is intentional. Most callers treat captcha as a gate. Pass the gate and continue, or fail the gate and reject the request. The service interface exposes the fine-grained response (score, timestamp, provider hostname) for the rare caller that needs it, but the default path sticks to the boolean-by-error pattern.

## Score normalisation as a portability trade

Captcha providers have different signals. Turnstile gives a pass/fail verdict. reCAPTCHA v3 gives a numeric score between 0.0 and 1.0. hCaptcha gives pass/fail plus an optional risk score. The HMAC dev provider gives pass/fail. Exposing all four shapes to application code would force every consumer to switch on provider type.

Piko normalises every provider onto the same scale. `VerifyResponse.Score` is `*float64` where `nil` means "no signal", `0.0` means "likely bot", and `1.0` means "likely human". A pass/fail verdict maps to `1.0` on success and absent on failure. A numeric score passes through. Score-based policies compare against `DefaultScoreThreshold` or an action-specific override, and application code stays provider-agnostic.

The cost of normalisation is that the unified shape drops some provider-specific detail. A reCAPTCHA v3 response with score `0.3` and an explicit risk reason becomes `0.3` without the reason in the default API. Callers that need the reason reach past `Verify` into the service layer.

The benefit is that swapping Turnstile for reCAPTCHA v3 is a one-line change at bootstrap, and every action that called `Verify` continues to work.

## Why four providers ship

Turnstile, reCAPTCHA v3, hCaptcha, and the HMAC challenge. Each serves a distinct need.

**Turnstile** is the pragmatic default. It runs invisibly in most cases, avoids a Google dependency, and has a generous free tier. Cloudflare owns the verify endpoint, which is a consideration for some operators but acceptable for most.

**reCAPTCHA v3** is the score-based incumbent. It scores every request between 0.0 and 1.0, and applications decide what to do with the number. It requires a Google dependency and a privacy statement. Sites already on Google's ecosystem often prefer it because the risk signals feed back into the same fraud-detection systems that power other Google products.

**hCaptcha** is the privacy-focused alternative. It collects less user data than reCAPTCHA and bills itself as a drop-in replacement. It still surfaces a challenge in some cases, so it leans more toward a "classic" captcha than Turnstile's invisible model.

**HMAC challenge** is the dev-only backstop. It produces a deterministic token from a shared secret with no external API call. It is safe for unit tests, integration tests, and local development. It is not safe for production because anyone who learns the secret can generate valid tokens.

Four providers cover the operational spectrum. Privacy-first (hCaptcha), invisible (Turnstile), score-based (reCAPTCHA), and development (HMAC). Adding a fifth is rarely worth the maintenance cost.

## Why HMAC is not production-safe

The HMAC provider's appeal in testing is the same thing that makes it dangerous in production. A valid token derives deterministically from the shared secret and the token payload. An attacker who learns the secret, whether through code leak, environment-variable exposure, or repository scanning, can generate valid tokens without ever running the captcha challenge.

Piko makes this warning explicit in the provider's documentation. The code itself does not check which environment it runs in. An operator who runs HMAC in production is free to do so, and free to pay the consequences. The library's role is to be loud about the risk, not to second-guess the operator.

## Action labels as fraud-detection hints

reCAPTCHA v3 and Turnstile accept an action name alongside the token. The action name tells the provider which form or flow generated the token. The provider echoes the name back in the verification response. A mismatch between the claimed action and the echoed action is a strong signal of a replayed token.

Piko passes the action name through as the third argument of `Verify`. Applications that care about action labels set a distinct name for every protected flow (`contact_submit`, `checkout_pay`, `signup`), and score-based providers treat the labels as separate behavioural features. Applications that do not care about action labels pass the empty string, and the provider treats the request generically.

## Composition with rate limiting and CSRF

Captcha is one layer of the security stack. It answers "is this request from a human". CSRF answers "is this request from the correct origin". Rate limiting answers "is this client behaving reasonably over time". The three layers defend against different threats.

The typical order on an action is:

1. CSRF check. Runs first because it is cheap and eliminates off-site forgery.
2. Rate limit. Runs second because it costs nothing beyond a counter bump.
3. Captcha verify. Runs third because it talks to an external API.
4. Business validation. Runs fourth because it may talk to a database.
5. The action itself.

Piko runs these in the dispatch middleware pipeline. Applications rarely need to think about the order. The framework applies CSRF and rate limit automatically based on action metadata, and the captcha verify is an explicit call from the action body. This explicit call is intentional. Captchas cost more than most middleware, and the application is best-placed to decide which actions warrant one.

## When not to reach for a captcha

A captcha is a tax on users. Some of that tax pays for security. Some of it is pure friction. Before adding a captcha, exhaust cheaper defences:

- Server-side validation (rejecting malformed input before it reaches a database).
- Rate limiting (rejecting clients that hit an endpoint too often).
- IP-based heuristics (rejecting known bot networks).
- Honeypot fields (hidden form fields that only bots fill in).

A captcha wins when abuse clears these cheaper defences but still runs automatically. Humans paid to manually solve captchas (captcha farms) exist and defeat any captcha eventually. A captcha is not a silver bullet for determined adversaries.

## Failure modes to expect

Three failure paths deserve explicit handling.

**Token expired**. The user left the form open for a long time. The captcha widget generates short-lived tokens. The user must re-challenge. For UX treatment, show the captcha widget again with a clear "verify again" message.

**Score below threshold**. The provider thinks the request is likely bot-generated. The user may be on a VPN, using a browser the provider distrusts, or may be a bot. For UX treatment, challenge the user again with a more interactive widget, or, for borderline scores, let the action proceed but flag it for review.

**Provider unavailable**. The provider's API is down. The action must decide between fail-open (let the request through) and fail-closed (block it). Fail-closed is the safer default for high-value actions (financial transactions). Fail-open is acceptable for low-value actions (newsletter signup). The decision is application-level, not framework-level.

## See also

- [Captcha API reference](../reference/captcha-api.md) for the full surface.
- [How to captcha](../how-to/captcha.md) for wiring Turnstile, integrating with an action, and swapping providers.
- [How to security](../how-to/security.md) for CSRF and rate-limit context.
- [Scenario 030: CAPTCHA-protected action](../showcase/030-captcha.md) for a runnable example.
