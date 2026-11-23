---
title: Captcha API
description: Multi-provider captcha verification with normalised scoring.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 200
---

# Captcha API

Piko's captcha service verifies tokens from Cloudflare Turnstile, Google reCAPTCHA v3, hCaptcha, and a built-in HMAC challenge useful for development. The service normalises provider scores onto a 0.0-1.0 scale so downstream policy code does not have to know which provider produced a token. For the design rationale see [about captcha](../explanation/about-captcha.md). For task recipes see [how to captcha](../how-to/captcha.md) and [how to security](../how-to/security.md). Source file: [`wdk/captcha/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/captcha/facade.go).

## Entry points

```go
func Verify(ctx context.Context, token, remoteIP, action string) error
func GetDefaultService() (ServicePort, error)
```

`Verify` is the usual call site. It looks up the default service, verifies the token, and returns a typed error. Piko passes the action label through to providers that support action-based fraud detection (reCAPTCHA v3, Turnstile).

## Types

| Type | Purpose |
|---|---|
| `ServicePort` | Service interface. Wraps provider selection, rate limiting, and score enforcement. |
| `Provider` | Provider interface. Implemented by each backend adapter. |
| `ProviderType` | Enum identifying a provider kind. |
| `VerifyRequest` | Full verification request. See fields below. |
| `VerifyResponse` | Provider response including normalised score. See fields below. |
| `ServiceConfig` | Service configuration. See fields below. |
| `CaptchaError` | Typed error. Pairs with the error constants listed below. |

### Verify-request fields

| Field | Type | Purpose |
|---|---|---|
| `Token` | `string` | Captcha response token from the client widget. |
| `RemoteIP` | `string` | Client IP, forwarded to the provider for extra validation. |
| `Action` | `string` | Optional action name (reCAPTCHA v3, Turnstile). Identifies the form or flow the token guards. |

### Verify-response fields

| Field | Type | Purpose |
|---|---|---|
| `Timestamp` | `time.Time` | When the user solved the challenge. |
| `Score` | `*float64` | Normalised confidence score. `0.0` indicates a likely bot, `1.0` a likely human. The pointer is always non-nil; binary providers fill in `1.0` on success and `0.0` on failure. |
| `Action` | `string` | Action echoed back by the provider. Confirms the token matches the expected action. |
| `Hostname` | `string` | Hostname the provider issued the token for, as the provider reports it. |
| `ErrorCodes` | `[]string` | Provider-specific error codes when verification fails. Diagnostic only; do not show to users. |
| `Success` | `bool` | Whether verification passed. |

### ServiceConfig fields

| Field | Type | Default | Purpose |
|---|---|---|---|
| `DefaultScoreThreshold` | `float64` | `0.5` | Minimum score required for score-based providers. Individual actions may override. |
| `VerifyRateLimit` | `int` | `20` / IP / min | Verification calls per IP per minute. Zero disables. |
| `ChallengeRateLimit` | `int` | `30` / IP / min | Challenge-token generation calls per IP per minute. Zero disables. |

`captcha_dto.DefaultServiceConfig()` returns a `*ServiceConfig` populated with the defaults above. The helper lives on the data transfer object (DTO) package `captcha_dto`. The `wdk/captcha` facade does not re-export it.

## Provider constants

| Constant | Meaning |
|---|---|
| `ProviderTypeHMACChallenge` | Built-in HMAC challenge. Development only. |
| `ProviderTypeTurnstile` | Cloudflare Turnstile. |
| `ProviderTypeRecaptchaV3` | Google reCAPTCHA v3. |
| `ProviderTypeHCaptcha` | hCaptcha. |

## Score normalisation

Providers report confidence in different ways. Piko maps them into a single `0.0-1.0` scale. `VerifyResponse.Score` is a `*float64` and is always populated by every provider (the binary providers fill in `1.0` on success and `0.0` on failure instead of leaving the pointer nil).

| Provider | Native signal | Normalised to |
|---|---|---|
| Turnstile | Pass/fail (no score) | `1.0` on success, `0.0` on failure. |
| reCAPTCHA v3 | `0.0-1.0` score | Passed through unchanged. |
| hCaptcha | Pass/fail | `1.0` on success, `0.0` on failure. |
| HMAC challenge | Pass/fail | `1.0` on success, `0.0` on failure. |

Actions consume the score through the same `VerifyResponse.Score` field regardless of provider. Score-based policies (reCAPTCHA v3) compare against `DefaultScoreThreshold` or an action-specific override.

## Errors

Verification can return `ErrCaptchaDisabled`, `ErrVerificationFailed`,
`ErrTokenMissing`, `ErrTokenExpired`, `ErrProviderUnavailable`,
`ErrProviderMisconfigured` or `ErrScoreBelowThreshold`.

## Providers

Each provider sub-package exposes a `NewProvider` constructor returning `(captcha_domain.CaptchaProvider, error)`.

Turnstile, reCAPTCHA v3 and hCaptcha accept these options through
`NewProvider(config Config, options ...Option)`.

| Option | Purpose |
|---|---|
| `WithVerifyTimeout(timeout time.Duration)` | Bounds each verification call, including reading the response. Defaults to 10 seconds. |
| `WithMaxResponseBytes(limit int64)` | Caps the verification response body. Defaults to 64 KiB. A larger response fails with an error wrapping `ErrProviderUnavailable` without truncating the response. |
| `WithTransport(transport http.RoundTripper)` | Sends verification requests through `transport`, for example an egress proxy. Defaults to the standard transport. |

A nonpositive timeout or body limit keeps its default. A nil transport uses the
standard HTTP transport.

### `captcha_provider_turnstile`

```go
type Config struct {
    SiteKey   string // public key embedded in the widget
    SecretKey string // server-side verification key
}
```

Constructor: `captcha_provider_turnstile.NewProvider(Config, ...Option)`.

### `captcha_provider_recaptcha_v3`

```go
type Config struct {
    SiteKey      string
    SecretKey    string
    TestKeyScore *float64
}
```

Constructor: `captcha_provider_recaptcha_v3.NewProvider(Config, ...Option)`.

Piko preserves scores from `0.0` to `1.0`, including zero. Set `TestKeyScore` to
supply the score omitted by Google's test keys. The fallback applies only to
responses from Google's test key hostname. With no applicable fallback, a
successful response without a score returns an error wrapping `ErrScoreMissing`
and `ErrProviderMisconfigured`. `NewProvider` rejects a fallback outside the
inclusive range `0.0` to `1.0` or a fallback of NaN with
`ErrTestKeyScoreOutOfRange`.

### `captcha_provider_hcaptcha`

```go
type Config struct {
    SiteKey   string
    SecretKey string
}
```

Constructor: `captcha_provider_hcaptcha.NewProvider(Config, ...Option)`.

### `captcha_provider_hmac_challenge`

```go
type Config struct {
    Clock  clock.Clock   // time source for token generation and TTL checks; defaults to the real system clock when nil
    Secret []byte        // at least 16 bytes
    TTL    time.Duration // defaults to 5 minutes
}
```

Constructor: `captcha_provider_hmac_challenge.NewProvider(Config)`. Deterministic, no external calls, useful in tests and local development. Never ship to production.

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithCaptchaProvider(name, provider)` | Registers a provider under `name`. |
| `piko.WithDefaultCaptchaProvider(name)` | Marks the registered provider with `name` as the default. |

## See also

- [How to captcha](../how-to/captcha.md) for wiring Turnstile, integrating with an action, and swapping providers.
- [How to security](../how-to/security.md) for the wider rate-limit and CSRF context.
- [About captcha](../explanation/about-captcha.md) for the score-normalisation rationale and composition with other security layers.
- [Scenario 030: CAPTCHA-protected action](../../examples/scenarios/030_captcha/) for a runnable example.
- Source: [`wdk/captcha/`](https://github.com/piko-sh/piko/tree/master/wdk/captcha).
