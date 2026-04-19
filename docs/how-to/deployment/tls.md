---
title: How to terminate TLS directly
description: Configure TLS, mTLS, certificate hot reload, and health-probe TLS without a reverse proxy.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 30
---

# How to terminate TLS directly

Piko can serve HTTPS on the main application port without a reverse proxy. This guide covers basic TLS, certificate hot reload, mutual TLS, and an independently configured TLS listener for the health probe. For every TLS option see [`internal/bootstrap/options_tls.go`](https://github.com/piko-sh/piko/blob/master/internal/bootstrap/options_tls.go).

## Basic TLS

Point Piko at a certificate and key file:

```go
ssr := piko.New(
    piko.WithTLS(
        piko.WithTLSCertFile("/certs/server.pem"),
        piko.WithTLSKeyFile("/certs/server.key"),
    ),
)
```

Or configure through YAML:

```yaml
network:
  tls:
    enabled: true
    certFile: /certs/server.pem
    keyFile: /certs/server.key
```

With TLS on, Piko sets HSTS headers automatically, switches CORS to secure origins, and negotiates HTTP/2 natively via ALPN (skipping the h2c cleartext wrapper).

## Certificate hot reload

Rotate certificates without restarting the server:

```go
ssr := piko.New(
    piko.WithTLS(
        piko.WithTLSCertFile("/certs/server.pem"),
        piko.WithTLSKeyFile("/certs/server.key"),
        piko.WithTLSHotReload(true),
    ),
)
```

```yaml
network:
  tls:
    enabled: true
    certFile: /certs/server.pem
    keyFile: /certs/server.key
    hotReload: true
```

The loader watches the directories containing the certificate and key (not the files directly) so it catches symlink swaps. That matters for Kubernetes and cert-manager, which renew mounted Secrets by replacing symlinks.

The reload flow:

1. The filesystem watcher detects a change.
2. A 500 ms debounce waits for both files to settle.
3. The new certificate loads and validates.
4. An atomic swap happens: the next TLS handshake uses the new certificate.
5. If the new certificate fails validation, the old one stays in place and a warning logs.

### Kubernetes volume mount

Mount the certificate from a Secret:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
    - name: app
      volumeMounts:
        - name: tls-certs
          mountPath: /certs
          readOnly: true
  volumes:
    - name: tls-certs
      secret:
        secretName: app-tls
```

With `hotReload: true`, cert-manager renewals propagate without a rolling restart.

## Mutual TLS

Require every client to present a certificate signed by a trusted CA:

```go
ssr := piko.New(
    piko.WithTLS(
        piko.WithTLSCertFile("/certs/server.pem"),
        piko.WithTLSKeyFile("/certs/server.key"),
        piko.WithTLSClientCA("/certs/ca.pem"),
        piko.WithTLSClientAuth("require_and_verify"),
    ),
)
```

```yaml
network:
  tls:
    enabled: true
    certFile: /certs/server.pem
    keyFile: /certs/server.key
    clientCaFile: /certs/ca.pem
    clientAuthType: require_and_verify
```

Client authentication modes:

| Mode | Behaviour |
|---|---|
| `none` | No client certificate requested. Default. |
| `request` | Requested but not required. |
| `require` | Required but not verified against the CA. |
| `verify` | Verified against the CA if provided. |
| `require_and_verify` | Required and verified against the CA. |

Rejected clients fail at the TLS layer. No HTTP handler runs.

## Minimum TLS version

Piko defaults to TLS 1.2. To require TLS 1.3:

```go
piko.WithTLS(
    piko.WithTLSCertFile("/certs/server.pem"),
    piko.WithTLSKeyFile("/certs/server.key"),
    piko.WithTLSMinVersion("1.3"),
)
```

```yaml
network:
  tls:
    minVersion: "1.3"
```

## TLS for the health probe

The health probe runs on a separate port. Its TLS configuration is independent:

```go
ssr := piko.New(
    piko.WithHealthTLS(
        piko.WithHealthTLSCertFile("/certs/health.pem"),
        piko.WithHealthTLSKeyFile("/certs/health.key"),
    ),
)
```

```yaml
healthProbe:
  tls:
    enabled: true
    certFile: /certs/health.pem
    keyFile: /certs/health.key
```

Useful when the health probe is reachable on a different trust boundary than the main application, or when the orchestrator requires HTTPS for probes.

## Redirect HTTP to HTTPS

Run an HTTP-to-HTTPS redirector on a second port:

```go
piko.WithTLS(
    piko.WithTLSCertFile("/certs/server.pem"),
    piko.WithTLSKeyFile("/certs/server.key"),
    piko.WithTLSRedirectHTTP(80),
)
```

Requests to port 80 receive a 301 redirect to the HTTPS equivalent.

## Environment variables

Every TLS setting is reachable via an environment variable:

| Variable | Field |
|---|---|
| `PIKO_TLS_ENABLED` | Enable or disable TLS. |
| `PIKO_TLS_CERT_FILE` | Certificate path. |
| `PIKO_TLS_KEY_FILE` | Private key path. |
| `PIKO_TLS_CLIENT_CA_FILE` | Client CA bundle for mTLS. |
| `PIKO_TLS_CLIENT_AUTH_TYPE` | Client auth mode. |
| `PIKO_TLS_MIN_VERSION` | Minimum TLS version. |
| `PIKO_TLS_HOT_RELOAD` | Enable hot reload. |

## Direct TLS or a reverse proxy

Use direct TLS when:

- Running in Kubernetes with cert-manager.
- Deploying into a service mesh that expects TLS between services.
- You need mTLS for service-to-service authentication.
- You want fewer moving parts in the deployment.

Use a reverse proxy (nginx, Caddy, Cloudflare) when:

- You need advanced load balancing or path-based routing.
- Multiple services share a single domain.
- You want centralised certificate management outside the binary.
- You need buffering, WAF, or other proxy-layer features.

## See also

- [How to production build](production-build.md).
- [How to environment configuration](environment-config.md).
- [How to troubleshooting deployment](troubleshooting.md).
- [Bootstrap options reference](../../reference/bootstrap-options.md).
- Source: [`internal/bootstrap/options_tls.go`](https://github.com/piko-sh/piko/blob/master/internal/bootstrap/options_tls.go).
- [Scenario 013: TLS and HTTPS](../../showcase/013-tls-https.md).
