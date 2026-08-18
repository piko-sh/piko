---
title: How to run canary and rolling deploys
description: Give each build a release identity, share a registry and blob store between instances, and retire releases once they scale to zero.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 20
---

# How to run canary and rolling deploys

This guide sets up a deployment where two releases of one application serve traffic at the same time: a canary next to the stable release, or the overlap window of a rolling update. Each instance serves its own release from its own self-contained binary; a shared registry database and blob store let instances resolve each other's assets, so a browser that loaded HTML from one release completes its page even when the requests land on the other. See [about single-binary deployment](../../explanation/about-single-binary-deployment.md) for the model behind this.

## Prerequisites

- Binaries built with `piko build` (see [how to production build](production-build.md)).
- A Postgres database reachable by every instance, registered as the registry database.
- A shared blob store reachable by every instance (S3 or another external provider) when instances also need to share runtime uploads and variants.

## Step 1: Give every build a release identity

The release identifier is what keeps two builds apart on the shared backend. It defaults to the VCS revision of the build, which is correct for most pipelines. Set it explicitly when the revision is not available or when you tag releases independently of commits:

```go
ssr := piko.New(
    piko.WithReleaseID(os.Getenv("RELEASE_ID")),
)
```

Two different builds must never share one identifier. If they do, the second publisher detects the mismatch and logs an error naming the fix; its assets are not published for cross-release serving until the identifiers are distinct. A build with no `WithReleaseID` and no VCS stamp publishes as `unversioned` with a warning, which is fine for a single release but collides the moment two coexist.

## Step 2: Share the registry and the blob store

Register the shared backends in `internal/piko.go` so every binary carries the same configuration:

```go
import (
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_postgres"
    "piko.sh/piko/wdk/db/db_schema_registry_postgres"
)

piko.WithDatabase(db.DatabaseNameRegistry, &db.DatabaseRegistration{
    DB:           database,
    EngineConfig: db_engine_postgres.Postgres(),
    MigrationFS:  db_schema_registry_postgres.Migrations,
}),
piko.WithStorageProvider("s3", s3Provider),
```

Everything else is automatic. At boot, each instance publishes its release's records into the shared registry as an immutable layer, copies any blobs the shared store is missing, and starts a lease heartbeat. Publishing runs in the background and never delays readiness; it is idempotent, so any number of instances of one release publish exactly once between them.

## Step 3: Roll out

Run both releases behind the load balancer. No coordination step is required:

- Requests for a page render on whichever release serves them.
- Requests for assets are content-addressed, so an old-release asset URL resolves on a new-release instance through the shared registry, and a page loaded mid-rollout completes with the assets its HTML references.
- Runtime uploads land in the shared store and are visible to every instance.

Watch the logs on the first deploy. A healthy publish logs `Registry release publish complete` with the outcome; a release identity problem logs `Release digest conflict` at error level.

## Step 4: Retire the old release

Once the old release has scaled to zero, retire it from deploy tooling:

```go
if err := ssr.RetireRelease(ctx, "v1.4.2"); err != nil {
    return err
}
```

Retiring removes that release's records from the shared registry, releases its blob references, and drops its lease, all in one transaction. Blobs still referenced by another release survive; blobs nobody references any longer are collected. Retiring is idempotent, and a release that was retired can be deployed again later.

Releases that are never retired explicitly are reaped automatically: every instance heartbeats its release's lease, and a release whose heartbeat has been silent for thirty minutes is retired by the surviving instances. The reaper never touches the release an instance is itself serving, so an instance cannot reap itself.

## Troubleshooting

| Symptom | Cause and fix |
|---|---|
| `Release digest conflict` in the logs | Two different builds share one release identifier. Set `WithReleaseID` per build, or retire the conflicting release. |
| `Publishing as 'unversioned'` warning | The build has no `WithReleaseID` and no VCS stamp. Harmless for a single release; set an identifier before running a canary. |
| Old-release assets 404 on new instances | The shared registry or blob store is not configured on every instance, or the old release's publish failed. Check for `Registry release publish complete` in the old release's boot logs. |
| A live release was retired | Its instances could not reach the shared database to heartbeat for over thirty minutes. The instances keep serving their own release from their own binaries; redeploying republishes the layer. |

## See also

- [About single-binary deployment](../../explanation/about-single-binary-deployment.md) for why releases coexist as immutable layers.
- [How to build for production](production-build.md) for `piko build` and the Dockerfile.
- [How to troubleshooting deployment](troubleshooting.md) for general production triage.
- [Bootstrap options reference](../../reference/bootstrap-options.md) for `WithReleaseID` and the single-binary options.
