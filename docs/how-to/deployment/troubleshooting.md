---
title: How to troubleshoot a deployment
description: Diagnose startup failures, port conflicts, database errors, health-check problems, and 5xx responses in production.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 50
---

# How to troubleshoot a deployment

Use this guide as a triage reference for common production failure modes. Each section starts with the symptom and walks through the commands and checks that narrow the cause.

## The multi-port architecture

A running Piko server listens on up to three ports:

| Server | Default port | Purpose | Config field |
|---|---|---|---|
| Main application | 8080 | HTTP requests, pages, actions. | `PIKO_PORT` |
| Health probe | 9090 | Liveness, readiness, Prometheus metrics. | `PIKO_HEALTH_PROBE_PORT` |
| gRPC monitoring | 9091 | TUI monitoring (gRPC). | Programmatic via `WithMonitoring(...)`. |

Environment variables cannot activate the gRPC monitoring listener. It requires the `WithMonitoring` bootstrap option in code.

## Application fails to start

### Binary exits immediately

Check the process manager and turn on trace logging:

```bash
# systemd
journalctl -u myapp -f
systemctl status myapp

# Turn up log level and run in the foreground for one attempt
PIKO_LOG_LEVEL=-8 ./app prod 2>&1 | head -200
```

Typical causes:

- Port already in use (see below).
- Missing config file (`piko.yaml` not found in the working directory).
- Database connection failure (see "database connection failed").
- Migration failure on startup.

### Permission denied

```bash
ls -la bin/app
stat bin/app
ls -la /path/to/certs /path/to/db

# File descriptor limits
ulimit -n
```

On containers, make sure the application user owns the working directory and any mounted volumes.

## Port conflicts

### Address already in use

```bash
sudo lsof -i :8080
sudo lsof -i :9090

# Linux-only alternative
ss -tlnp | grep -E ':8080|:9090'
```

Options:

- Stop the previous instance.
- Set `network.autoNextPort: true` in `piko.yaml` to fall back to the next free port. Not appropriate when traffic reaches the service through a fixed port.
- Change the configured port (`PIKO_PORT`, `PIKO_HEALTH_PROBE_PORT`).

### Cannot bind to 80 or 443

Running on privileged ports requires either root, `CAP_NET_BIND_SERVICE`, or an orchestrator that maps the ports. Prefer the capability:

```bash
sudo setcap 'cap_net_bind_service=+ep' ./bin/app
```

Systemd can grant it via `AmbientCapabilities=CAP_NET_BIND_SERVICE`.

## Database connection failed

```bash
# PostgreSQL
psql -h dbhost -U user -d database -c 'SELECT 1'

# SQLite
sqlite3 /path/to/registry.db '.tables'

# Inside a container: check it can reach the database service
docker compose exec myapp nc -zv postgres 5432
```

Check:

- `PIKO_DATABASE_POSTGRES_URL` (or equivalent) resolves.
- SSL mode matches the server (`sslmode=require` or `sslmode=disable`).
- Credentials in the secret resolver resolved successfully (try `PIKO_LOG_LEVEL=-6` and look for resolver diagnostics).
- Network policies do not block the egress.

## Migration failure

Piko runs SQL migrations at startup. If they fail, the server refuses to start.

```bash
# Check the startup logs at internal level for the exact error
PIKO_LOG_LEVEL=-6 ./app prod 2>&1 | grep -i migrat

# Inspect the migration tracking tables directly
#   PostgreSQL: registry_schema_migrations, orchestrator_schema_migrations
#   SQLite:     schema_migrations
```

The binary embeds every migration. You cannot skip one selectively. Fix the underlying issue (permissions, schema conflict, dialect difference) and restart.

## 404 errors

### Routes not found

Verify the generator ran before you built the binary:

```bash
ls -la .piko .out dist/generated.go 2>/dev/null
go run ./cmd/generator/main.go all
```

If the manifest or generated Go code is missing, the router has no routes to serve.

Check the base-path configuration if you mounted the app under a subpath:

```yaml
network:
  basePath: "/app"
```

### Static assets 404

Static asset manifests live under `.piko` and `.out`. Include them in the deployment artefact (the Dockerfile under `how-to: production build` does this). A missing manifest usually surfaces as assets 404 or the browser loading an outdated build.

## 500 errors

```bash
# Look for stack traces
PIKO_LOG_LEVEL=debug ./app prod 2>&1 | grep -i 'panic\|fatal\|error'

# A single failing action can be isolated via its endpoint
curl -v -X POST http://localhost:8080/actions/customer.Upsert -d '{}'
```

Check:

- External-service dependencies (payment gateway, email, LLM) returned unexpected responses.
- A recent code or config change landed. `git log` for the last deploy window.
- The error page at `pages/!500.pk` (or `pages/!error.pk`) is present and does not itself error.

## SSL/HTTPS issues

### Certificate errors

```bash
# Inspect the certificate served
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com < /dev/null 2>&1 | openssl x509 -noout -dates -subject -issuer

# Validate against a CA bundle
openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt server.pem
```

Common causes:

- Certificate expired.
- Hostname mismatch (`CN` or `SAN` does not match the request's hostname).
- Chain incomplete: serve the intermediate bundle too, or use a full-chain file.
- `hotReload: true` disabled and the certificate renewed on disk but not reloaded.

### Mixed-content warnings

When terminating TLS directly, set `network.forceHttps: true` so HTTP requests redirect. When fronting with a proxy, make sure the proxy sends `X-Forwarded-Proto: https` and the `network.trustedProxies` list includes it.

## Health-check failures

```bash
curl -v http://localhost:9090/live
curl -v http://localhost:9090/ready
curl http://localhost:9090/metrics | head
```

If the probe server itself does not respond:

- Confirm `healthProbe.enabled: true` (default `true`).
- Confirm the bind address: `127.0.0.1` rejects external probes. Use `0.0.0.0` for Kubernetes and Docker.

```yaml
healthProbe:
  enabled: true
  port: "9090"
  bindAddress: "0.0.0.0"
  livePath: "/live"
  readyPath: "/ready"
  metricsPath: "/metrics"
  metricsEnabled: true
  checkTimeoutSeconds: 5
```

All health-probe settings have environment-variable equivalents (`PIKO_HEALTH_PROBE_*`). Check them with `env | grep PIKO_HEALTH_PROBE`.

## Docker-specific issues

### Container exits immediately

```bash
docker logs myapp --tail 200
docker inspect myapp --format '{{.State.ExitCode}}'
```

Typical causes:

- `CMD` points at a path that does not exist in the image.
- Working directory missing (`WorkingDir` in the Dockerfile does not match the binary's expectations).
- Config file baked in at a different path than the runtime lookup.

### Health checks fail in the container

Set `healthProbe.bindAddress: "0.0.0.0"` and expose port 9090 in the `Dockerfile` and Kubernetes manifest. `127.0.0.1` (the safe default) refuses external probes.

## Performance issues

### Slow response times

```bash
# Look at request latency in the logs
tail -f /var/log/myapp/app.log | jq 'select(.message=="Request completed") | {path, duration}'

# Capture an on-demand profile
curl "http://localhost:9090/_piko/profiler/cpu?seconds=30" > cpu.pprof
go tool pprof cpu.pprof
```

If profiling is not enabled by default, enable it with `piko.WithProfiling()` or via `PIKO_PROFILING_ENABLED=true`. See the [profiling how-to](../profiling.md) for the workflow.

### High memory usage

```bash
curl "http://localhost:9090/_piko/profiler/heap" > heap.pprof
go tool pprof heap.pprof
```

Check for goroutine leaks:

```bash
curl "http://localhost:9090/_piko/profiler/goroutine?debug=1" | head -50
```

## Rollback

```bash
# Binary
mv bin/app bin/app-failed
mv bin/app-previous bin/app
systemctl restart myapp

# Docker
docker stop myapp
docker run -d --name myapp myapp:previous-tag

# Kubernetes
kubectl rollout undo deployment/myapp
```

Keep the previous binary or image reachable until the new deploy has been healthy for long enough to be confident in it.

## Common error messages

| Message | Likely cause |
|---|---|
| `bind: address already in use` | Another process on 8080 or 9090. |
| `connection refused` | App not running, wrong port, firewall. |
| `no such file or directory` | Missing generated assets, wrong working directory. |
| `permission denied` | File permissions, SELinux/AppArmor, privileged port without capability. |
| `database connection failed` | Wrong URL, DB down, network policy, SSL mismatch. |

## Environment-variable reference

### Core

| Variable | Default | Purpose |
|---|---|---|
| `PIKO_PORT` | `8080` | Main HTTP port. |
| `PIKO_LOG_LEVEL` | `info` | `trace`, `internal`, `debug`, `info`, `notice`, `warn`, `error`, or the numeric value. |
| `PIKO_FORCE_HTTPS` | `false` | Redirect HTTP to HTTPS. |
| `PIKO_ENV` | `dev` | Current environment name. |

### Health probe

| Variable | Default | Purpose |
|---|---|---|
| `PIKO_HEALTH_PROBE_ENABLED` | `true` | Enable the health-probe server. |
| `PIKO_HEALTH_PROBE_PORT` | `9090` | Port. |
| `PIKO_HEALTH_PROBE_BIND_ADDRESS` | `127.0.0.1` | Bind address. Use `0.0.0.0` for Kubernetes. |
| `PIKO_HEALTH_PROBE_LIVE_PATH` | `/live` | |
| `PIKO_HEALTH_PROBE_READY_PATH` | `/ready` | |
| `PIKO_HEALTH_PROBE_METRICS_PATH` | `/metrics` | |
| `PIKO_HEALTH_PROBE_METRICS_ENABLED` | `true` | Enable Prometheus metrics. |
| `PIKO_HEALTH_PROBE_CHECK_TIMEOUT` | `5` | Per-probe timeout in seconds. |
| `PIKO_HEALTH_PROBE_AUTO_PORT` | `false` | Try the next port if another process already holds the configured one. |

### Database

| Variable | Default | Purpose |
|---|---|---|
| `PIKO_DATABASE_DRIVER` | `sqlite` | `sqlite`, `postgres`, `d1`, and others. |
| `PIKO_DATABASE_POSTGRES_URL` | | PostgreSQL connection URL. |
| `PIKO_DATABASE_POSTGRES_MAX_CONNS` | `10` | |
| `PIKO_DATABASE_POSTGRES_MIN_CONNS` | `2` | |

## Debug checklist

A short list for running through during an incident:

1. Check the application logs at `internal` level (`PIKO_LOG_LEVEL=-6`).
2. Test both health endpoints (`/live` and `/ready` on port 9090).
3. Scrape `/metrics` for any out-of-range counters.
4. Check `env | grep PIKO` for the currently loaded variables.
5. Confirm ports 8080 and 9090 are available.
6. Exercise the database connection from the host.
7. Verify file permissions on the binary, certs, and data directories.
8. Check disk space (`df -h`).
9. Review the last deploy change.
10. Reproduce locally with the production config.
11. Regenerate assets if routes or static content are missing.

## See also

- [How to production build](production-build.md).
- [How to environment configuration](environment-config.md).
- [How to monitoring](monitoring.md).
- [How to profiling](../profiling.md).
- [CLI reference](../../reference/cli.md).
