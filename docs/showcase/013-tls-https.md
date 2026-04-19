---
title: "013: TLS and HTTPS"
description: Serve Piko over HTTPS with certificate hot-reload and an HTTP-to-HTTPS redirect.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 240
---

# 013: TLS and HTTPS

A Piko server configured to serve HTTPS directly, with certificate hot-reload so `letsencrypt` renewals land without a restart, and an HTTP listener that redirects to HTTPS.

## What this demonstrates

- `WithTLS` with a certificate and key file.
- `WithTLSHotReload` to watch the cert files and reload on change.
- `WithTLSRedirectHTTP` to redirect plain-HTTP requests to HTTPS.
- `WithTLSMinVersion("1.3")` for a modern default.

## Project structure

```text
src/
  cmd/main/main.go      Bootstraps the server with WithTLS options.
  pages/
    index.pk            A small landing page served only over HTTPS.
  cert.pem              Certificate (generated locally for this example).
  key.pem               Private key.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/013_tls_https/src/
go mod tidy
air
```

Visit the configured HTTPS URL. Accept the self-signed certificate in the browser. The example ships a locally generated one.

## See also

- [Bootstrap options reference: TLS](../reference/bootstrap-options.md#tls-options).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/013_tls_https).
