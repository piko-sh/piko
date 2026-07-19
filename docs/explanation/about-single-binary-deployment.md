---
title: About single-binary deployment
description: Why the production binary carries its own assets, the base-and-overlay store model, what stays ephemeral, and when to add shared backends.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 30
---

# About single-binary deployment

`piko build` produces one file, and that file is the release. The compiled manifest, the pages, the collections, the asset blobs, and the registry of what exists all travel inside the binary. Nothing needs to sit beside it on disk, nothing gets unpacked at boot, and nothing on the server can drift out of agreement with it. This page explains why the binary takes that form, the store model that makes it work, and where the boundary sits between what the binary carries and what a deployment must still provide.

## The binary is the release

Most deployment pain is agreement pain. A binary that reads its assets from a folder must trust that the folder matches the build that produced it. Copy the folder late and old HTML references assets that do not exist yet. Copy it early and new HTML references assets that are already gone. Tools grow around the problem: sync steps, versioned buckets, boot-time checks that compare manifests and repair differences.

Piko removes the agreement problem instead of managing it. At build time the generator copies the runtime payload into the `dist/` package and emits a small generated file that embeds it. `piko build` compiles that payload in. The result is immutable in the way a Go binary is immutable: the assets a release serves are fixed the moment it is linked, and two copies of the binary are the same release by construction. Deployment becomes file copy plus restart, and rollback becomes running the previous file.

The payload is gated behind a build tag, so this costs development nothing. A plain `go build`, a test run, or the `air` loop never compiles the payload, and development serves from the working folders exactly as before. Only a production build carries the weight, and only a production run reads it.

## A deployment is a stack of stores

Inside the binary, the payload becomes a read-only base layer: records held in memory, bytes served straight from the embedded filesystem. Above it sits an optional writable overlay for anything created at runtime, such as uploads and on-demand image variants. Reads check the base and the overlay and merge the result. Writes only ever touch the overlay. The base cannot be modified, cannot be evicted, and cannot lose an asset while the process lives.

Every deployment shape is a choice of overlay, and nothing else changes:

- **No overlay.** A static site in a distroless container with a read-only root filesystem. The binary serves its base and writes nothing.
- **Local overlay.** A single server with a disk-backed store for runtime data. The base still serves the release; the overlay holds what users add.
- **Shared overlay.** Several instances behind a load balancer, with Postgres holding the records and S3 holding the bytes. Each instance serves its own release from its own base and shares runtime data through the overlay.

Development is the degenerate case: no base, only the working folders. That is why the model adds no friction to the edit loop. Dev and production run the same code over a different stack.

## Ephemeral by default, durable by choice

A container filesystem does not survive a restart, so Piko refuses to treat it as storage. With no overlay configured, the binary declines runtime writes rather than accepting them into a location that will vanish: on-demand variant generation steps aside and the page cache runs without persistence. Nothing errors per request, and nothing pretends to be durable.

Durability is therefore an explicit decision. Runtime data that must survive a restart needs a real backend: a storage provider for the bytes and a registry database for the records that describe them. Registering both is the whole opt-in. The framework will not silently write to a folder inside the container and let a rollout delete user uploads, because losing data quietly is worse than asking for one configuration decision.

## Releases can coexist

A rolling deploy always has a moment where two releases serve traffic at once, and a browser that loaded HTML from the old release is still fetching that page's assets. Piko handles this with release identity. Every build carries a release identifier, taken from the VCS revision by default and settable with `WithReleaseID`. When a shared registry database is configured, each instance publishes its release's records into it at boot, as its own layer keyed by that identifier. Publishing is idempotent, so any number of instances of one release publish exactly once between them.

Asset URLs are content-addressed, and a lookup by content address is deliberately exempt from freshness rules: it is a request for those exact bytes, made by HTML that was generated against them. So during a rollout, a request for an old-release asset that lands on a new-release instance still resolves, and the half-loaded page completes. Old releases are retired explicitly from deploy tooling, or reaped automatically once every instance of the release has stopped renewing its lease. Retiring one release removes only its own layer. Content shared between releases survives because the store counts references.

Publishing is an optimisation for multi-instance serving, never a requirement. An instance always serves its own release from its own base, so a failed publish degrades cross-release serving and nothing else. The binary that cannot reach the database still works.

## What the model costs

The trade-offs are real and worth naming.

**The binary grows with the site.** `EmbedAll` bakes every blob into the file, so a media-heavy site produces a heavy binary and a slower first build while the payload is copied. That is also precisely the site that should keep its media in external storage, which removes the weight again. For deployments that prefer files on disk, `piko build --no-embed` produces the previous shape: a binary that reads `dist/` and `.piko/` beside it.

**Regeneration must precede a tagged build.** The payload is frozen at generation time. `piko build` runs the generator first for exactly this reason; a manual `go build -tags piko_embed` against a stale payload ships stale assets.

**Runtime state inside the container is disposable.** This is a feature until the first time it surprises someone. The rule is simple: if it must survive a restart, it needs a backend.

## See also

- [How to build for production](../how-to/deployment/production-build.md) for the build commands, the Dockerfile, and the file-based alternative.
- [How to run canary and rolling deploys](../how-to/deployment/canary-and-rolling-deploys.md) for the multi-release flow on a shared backend.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithReleaseID`, `WithEmbedScope`, and the embed overrides.
- [CLI reference](../reference/cli.md) for `piko generate`, `piko build`, and `piko dev`.
- [About project structure](about-project-structure.md) for the source-versus-generated folder contract the payload is built from.
