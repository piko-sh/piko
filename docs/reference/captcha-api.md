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
| `Score` | `*float64` | Normalised confidence score. `0.0` indicates a likely bot, `1.0` a likely human. Always populated (defaults filled for binary providers). |
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

`DefaultServiceConfig()` returns the above defaults.

## Provider constants

| Constant | Meaning |
|---|---|
| `ProviderTypeHMACChallenge` | Built-in HMAC challenge. Development only. |
| `ProviderTypeTurnstile` | Cloudflare Turnstile. |
| `ProviderTypeRecaptchaV3` | Google reCAPTCHA v3. |
| `ProviderTypeHCaptcha` | hCaptcha. |

## Score normalisation

Providers report confidence in different ways. Piko maps them into a single `0.0-1.0` scale.

| Provider | Native signal | Normalised to |
|---|---|---|
| Turnstile | Pass/fail (no score) | `1.0` on success, absent on failure. |
| reCAPTCHA v3 | `0.0-1.0` score | Passed through unchanged. |
| hCaptcha | Pass/fail | `1.0` on success, absent on failure. |
| HMAC challenge | Pass/fail | `1.0` on success, absent on failure. |

Actions consume the score through the same `VerifyResponse.Score` field regardless of provider. Score-based policies (reCAPTCHA v3) compare against `DefaultScoreThreshold` or an action-specific override.

## Errors

`ErrCaptchaDisabled`, `ErrVerificationFailed`, `ErrTokenMissing`, `ErrTokenExpired`, `ErrProviderUnavailable`, `ErrScoreBelowThreshold`.

## Providers

Each provider sub-package exposes `NewProvider(Config) (captcha_domain.CaptchaProvider, error)`.

### `captcha_provider_turnstile`

```go
type Config struct {
    SiteKey   string // public key embedded in the widget
    SecretKey string // server-side verification key
}
```

Constructor: `captcha_provider_turnstile.NewProvider(Config)`.

### `captcha_provider_recaptcha_v3`

```go
type Config struct {
    SiteKey   string
    SecretKey string
}
```

Constructor: `captcha_provider_recaptcha_v3.NewProvider(Config)`.

### `captcha_provider_hcaptcha`

```go
type Config struct {
    SiteKey   string
    SecretKey string
}
```

Constructor: `captcha_provider_hcaptcha.NewProvider(Config)`.

### `captcha_provider_hmac_challenge`

```go
type Config struct {
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
- [Scenario 030: CAPTCHA-protected action](../showcase/030-captcha.md) for a runnable example.
- Source: [`wdk/captcha/`](https://github.com/piko-sh/piko/tree/master/wdk/captcha).
