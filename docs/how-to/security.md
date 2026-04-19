---
title: How to configure security headers
description: Configure content security policy, trusted proxies, CSRF, and rate limiting.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 80
---

# How to configure security headers

Piko ships defaults for CSRF, rate limiting, and content security policy (CSP). This guide covers how to pick the right policy for a project and how to harden the configuration for production. See the [bootstrap options reference](../reference/bootstrap-options.md) for the full option list.

## Pick a CSP preset

CSP controls which resources a page can load. Piko offers four presets:

| Option | Use when |
|---|---|
| `piko.WithPikoDefaultCSP()` | Default balance of safety and convenience. Recommended starting point. |
| `piko.WithStrictCSP()` | Production sites with no inline scripts, no `eval`, and a locked-down allow-list. |
| `piko.WithRelaxedCSP()` | Legacy content that cannot yet meet strict rules; use as a migration stepping stone. |
| `piko.WithAPICSP()` | API-only deployments with no HTML output; minimal policy. |

```go
ssr := piko.New(
    piko.WithStrictCSP(),
)
```

## Customise the CSP

For fine-grained control, use `piko.WithCSP(configure)` and build the policy programmatically:

```go
ssr := piko.New(
    piko.WithCSP(func(b *piko.CSPBuilder) {
        b.ScriptSrc(piko.CSPSelf, piko.CSPHost("cdn.example.com"))
        b.StyleSrc(piko.CSPSelf, piko.CSPUnsafeInline)
        b.ImgSrc(piko.CSPSelf, piko.CSPData, piko.CSPHost("images.example.com"))
        b.ConnectSrc(piko.CSPSelf, piko.CSPHost("api.example.com"))
        b.FrameAncestors(piko.CSPNone)
    }),
)
```

CSP source helpers cover the common keywords and host patterns: `piko.CSPSelf`, `piko.CSPNone`, `piko.CSPUnsafeInline`, `piko.CSPUnsafeEval`, `piko.CSPData`, `piko.CSPBlob`, `piko.CSPHTTPS`, plus `piko.CSPHost("cdn.example.com")` and `piko.CSPScheme("wss:")` for explicit hosts and schemes. Each source is a value of the named type `Source`. Passing raw strings does not compile.

Or pass a raw policy string with `piko.WithCSPString(policy)`.

## Report CSP violations

Route violation reports to an endpoint:

```go
ssr := piko.New(
    piko.WithStrictCSP(),
    piko.WithReportingEndpoints(
        piko.ReportingEndpoint{Name: "csp-endpoint", URL: "https://reports.example.com/csp"},
    ),
)
```

The browser sends `report-to` POSTs to the endpoint for every blocked resource. Log them, alert on new patterns, and tighten the policy as the noise drops.

## Trusted proxies

If Piko sits behind a reverse proxy (Caddy, nginx, Cloudflare), tell it which proxies to trust so `r.ClientIP()` resolves correctly:

```go
ssr := piko.New(
    piko.WithTrustedProxies(
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
    ),
)
```

For Cloudflare specifically:

```go
ssr := piko.New(
    piko.WithCloudflareEnabled(true),
)
```

This enables trust of the `CF-Connecting-IP` header in addition to the trusted-proxy chain.

## CSRF

Piko validates CSRF tokens on every action request. Configure the signing secret and token lifetime:

```go
ssr := piko.New(
    piko.WithCSRFSecret([]byte(os.Getenv("CSRF_SECRET"))),
    piko.WithCSRFTokenMaxAge(24*time.Hour),
    piko.WithCSRFSecFetchSiteEnforcement(true),
)
```

- `WithCSRFSecret` takes a random 32-byte key. Generate once and store in your secret manager.
- `WithCSRFTokenMaxAge` defaults to a sensible production value; shorten for extra safety, lengthen for long-running SPA sessions.
- `WithCSRFSecFetchSiteEnforcement` adds `Sec-Fetch-Site` header checks on top of the token. Requires modern browsers; leave enabled unless you need to support legacy user agents.

## Rate limiting

Enable the built-in limiter:

```go
ssr := piko.New(
    piko.WithRateLimitEnabled(true),
)
```

The limiter applies to the action dispatch path. Actions can declare their own limits by implementing `piko.RateLimitable`. See the [server actions reference](../reference/server-actions.md).
## Cross-Origin Resource Policy
Set the `Cross-Origin-Resource-Policy` header for images, fonts, and other subresources:

```go
ssr := piko.New(
    piko.WithCrossOriginResourcePolicy("same-origin"),
)
```

Values: `"same-site"`, `"same-origin"`, `"cross-origin"`. Most projects want `"same-origin"`.

## TLS and HSTS

For direct HTTPS termination:

```go
ssr := piko.New(
    piko.WithTLS(
        piko.WithTLSCertFile("/etc/ssl/cert.pem"),
        piko.WithTLSKeyFile("/etc/ssl/key.pem"),
        piko.WithTLSMinVersion("1.3"),
        piko.WithTLSHotReload(true),
        piko.WithTLSRedirectHTTP(80),
    ),
)
```

For HSTS, add the header through your CSP builder or your reverse proxy. Most production Piko deployments terminate TLS at a reverse proxy (Caddy, nginx, a load balancer) instead of in-process. Choose based on operational preference.

## Secrets

Never embed credentials in source code or in any committed file. Use `Secret[T]` with a resolver and load the placeholder reference from your environment or secret store. See the [secrets how-to](secrets.md).

## See also

- [Bootstrap options reference: Security](../reference/bootstrap-options.md#security).
- [Server actions reference](../reference/server-actions.md) for per-action rate limiting and captcha protection.
- [How to secrets](secrets.md).
- [Scenario 013: TLS and HTTPS](../../examples/scenarios/013_tls_https/) for a runnable TLS setup.
- [How to frontend analytics](frontend-analytics.md) for GA4/GTM CSP recipes that extend the strict baseline.
