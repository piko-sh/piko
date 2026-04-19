---
title: About the hexagonal architecture
description: Why Piko's core defines interfaces and lets the application wire implementations.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 40
---

# About the hexagonal architecture

Piko follows the hexagonal-architecture pattern (sometimes called "ports and adapters"). The framework defines the interfaces it depends on, and the application supplies the implementations. Every external dependency, from the cache backend to the LLM provider, is swappable. This page explains why the framework takes that form and how it behaves.

<p align="center">
  <img src="../diagrams/hexagonal-ports-adapters.svg"
       alt="Piko Core at the centre with six ports around it: cache, storage, and email above, LLM, image, and notification below. Each port has two or three named adapters plugged in, such as Valkey, S3, OpenAI, and libvips."
       width="560"/>
</p>

The diagram above is the mental model for the rest of this page. The `Piko Core` in the middle is the framework. Around it sit the ports, which are Go interfaces describing what the framework needs from the outside world. The chips around each port are adapters, the concrete implementations the project supplies at bootstrap.

## What "hexagonal" means here

The core of Piko is a set of Go interfaces that describe what the framework needs from the outside world:

- A cache provider can get, set, and invalidate keys.
- A storage provider can put and get blobs with optional presigned URLs.
- An email provider can send a message.
- An LLM provider can complete a prompt.
- An image provider can resize and reformat an image.

Each interface is a port. Each implementation is an adapter. The framework does not know whether the cache is in-memory, Valkey, or Redis. It only knows the adapter satisfies the `cache.Provider` interface.

Application code wires the adapters at startup:

```go
ssr := piko.New(
    piko.WithCacheProvider("redis", redis.NewProvider(redisConfig)),
    piko.WithDefaultCacheProvider("redis"),
    piko.WithStorageProvider("s3", s3.NewProvider(s3Config)),
    piko.WithDefaultStorageProvider("s3"),
    piko.WithEmailProvider("ses", ses.NewProvider(sesConfig)),
    piko.WithDefaultEmailProvider("ses"),
)
```

## Why it matters in practice

Three things fall out of the shape.

**Testing is direct.** A test replaces the real storage provider with an in-memory fake in a single line. No test-database setup, no network mocking. The framework invites replacement at the port boundary, so tests do exactly that.

```go
ssr := pikotest.InitialiseForTesting(
    pikotest.WithStorageProvider("fake", inmem.NewProvider()),
    pikotest.WithCacheProvider("fake", inmem.NewCache()),
)
```

**Deployment shape follows choice.** A project might develop against SQLite and Valkey locally, deploy against Postgres and Redis in production, and run tests against in-memory versions of both. None of the PK or action code changes. Only the bootstrap does.

**Growth is additive.** A project that does not need email sends `WithEmailProvider` nothing, and the email service is never wired. Adding email later means registering a provider in the bootstrap and calling `piko.GetEmailService()` from an action. No big refactor comes with it. The framework just notices the new provider and wires it in.

## The bootstrap as the composition root

The bootstrap is the one place where concrete implementations meet the framework. Everywhere else, application code asks for an interface and receives whatever implementation the bootstrap wired. This is the "composition root" pattern from dependency-injection orthodoxy, and the framework gives it structure.

A typical bootstrap picks a provider per port from a library of adapters that ship with Piko or that the project writes in-house:

- Cache: in-memory, Valkey, Redis, DuckDB.
- Storage: local disk, S3, GCS, Cloudflare R2.
- Email: stdout (dev), SES, SendGrid, SMTP.
- LLM: OpenAI, Anthropic, local Ollama, custom HTTP client.
- Image: libvips, imaginary.

The full list lives in the [bootstrap options reference](../reference/bootstrap-options.md).

## What the pattern does not do

Hexagonal architecture does not pretend that all adapters behave identically. An in-memory cache behaves differently from Redis under concurrent load. A local-disk storage provider does not support presigned URLs. An SMTP email adapter has different failure modes from an SES adapter. The interfaces describe the operations, not the guarantees. An application that switches adapters in production must still test against the target adapter.

Hexagonal architecture does not eliminate cross-adapter coupling. An action that uses the cache and the storage provider together has to reason about the interaction (does invalidating the cache also invalidate any URLs we signed?). The framework cannot solve that for you.

## When to add a new adapter

- A new backend exists that the project wants to use (a new LLM provider, a new object-storage service).
- The project has a testing need the existing adapters do not cover (a fake that injects specific failures).
- An external service has a nonstandard API that does not fit the port's interface. In that case, consider whether the port needs to grow instead of wrapping a mismatched service.

## See also

- [Bootstrap options reference](../reference/bootstrap-options.md) for every `With*` option and its provider interface.
- [About PK files](about-pk-files.md) for the application shape.
- [About reactivity](about-reactivity.md) for how the PK/PKC split fits the architecture.
- [How to testing](../how-to/testing.md) for using fake adapters in tests.
