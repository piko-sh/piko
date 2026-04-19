---
title: Storage API
description: Object-storage service, upload and request builders, and provider registration.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 110
---

# Storage API

Piko's storage service wraps object-storage backends (local disk, S3-compatible, GCS) behind one fluent builder. Uploads and reads flow through optional stream transformers that compress or encrypt content in-line. For task recipes see the [assets how-to](../how-to/assets.md). Source of truth: [`wdk/storage/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/storage/facade.go).

## Service

| Function | Returns |
|---|---|
| `storage.NewService(defaultProviderName string, opts ...ServiceOption) Service` | Constructs a new service with a named default provider. |
| `storage.GetDefaultService() (Service, error)` | Returns the service the bootstrap built from `WithStorageProvider` options. |

## Builders

```go
func NewUploadBuilder(service Service, reader io.Reader) (*UploadBuilder, error)
func NewUploadBuilderFromDefault(reader io.Reader) (*UploadBuilder, error)
func NewRequestBuilder(service Service, repository, key string) (*RequestBuilder, error)
func NewRequestBuilderFromDefault(repository, key string) (*RequestBuilder, error)
```

Both builders share `.Provider(name)` and `.Transformer(name, options...)` and clone via `.Clone()`. The two surfaces diverge in their fluent vocabulary and terminal calls.

### `UploadBuilder`

`UploadBuilder` is for PUT operations. Configure with `.Key(key)`, `.Repository(repo)`, `.ContentType(mime)`, `.Size(bytes)`, `.Metadata(map)`, `.CAS(algorithm, expectedHash...)`, `.Multipart(config)`, `.Transformer(name, options...)`, and `.Dispatch()` for asynchronous queuing. Execute with `.Do(ctx)`. Use `.Build()` to obtain a reusable `PutObjectSpec` for batch uploads.

### `RequestBuilder`

`RequestBuilder` is for read, stat, remove, and hash operations on a `(repository, key)` pair fixed at construction. Configure with `.ByteRange(start, end)`, `.Transformer(name, options)`, and `.DispatchRemove()` for asynchronous deletion. Terminal methods are:

| Method | Returns |
|---|---|
| `Get(ctx)` | `(io.ReadCloser, error)`. Caller closes the stream. |
| `Stat(ctx)` | `(*ObjectInfo, error)`. Metadata only. |
| `Remove(ctx)` | `error`. Deletes the object. |
| `Hash(ctx)` | `(string, error)`. Returns the content hash. |

`RequestBuilder` does not expose `.Key(...)`, `.ContentType(...)`, or `.Do(ctx)`. Those belong to `UploadBuilder`.

## Types

| Type | Purpose |
|---|---|
| `Service` | Entry point. Manages providers and resolves operations. |
| `ProviderPort` | Interface a provider implements (GET, PUT, DELETE, PRESIGN). |
| `DispatcherPort` | Interface for background upload dispatchers. |
| `StreamTransformerPort` | Interface for pipeline transformers (compression, encryption). |
| `UploadBuilder` | Fluent builder for PUT operations. |
| `RequestBuilder` | Fluent builder for GET, DELETE, COPY, PRESIGN operations. |
| `ObjectInfo` | Metadata returned by `Stat` and `List`. |
| `BatchResult` / `BatchFailure` | Returned by `PutMany`, `RemoveMany`, `Migrate`. |

Params structs: `PutParams`, `GetParams`, `CopyParams`, `PresignParams`, `PresignDownloadParams`, `PutManyParams`, `RemoveManyParams`, `MigrateParams`, `MultipartUploadConfig`, `ByteRange`.

## Dispatcher and resilience

Upload bursts go through an optional dispatcher configured via `DispatcherConfig`. Retries, rate limits, and circuit breakers take their configuration from `RetryConfig`, `ProviderRateLimitConfig`, and `CircuitBreakerConfig`. Stats land in `DispatcherStats`.

## Transformers

| Sub-package | Role |
|---|---|
| `storage_transformer_gzip` | Gzip (DEFLATE) compression on upload; automatic decompression on read. |
| `storage_transformer_zstd` | Zstandard compression on upload; automatic decompression on read. |
| `storage_transformer_crypto` | AES encryption with envelope keying. |

Attach transformers from a builder via `.Transformer(name, options...)`. Each transformer reports its kind through `TransformerType`, with constants `TransformerCompression`, `TransformerEncryption`, and `TransformerCustom`.

## Providers

| Sub-package | Backend |
|---|---|
| `storage_provider_disk` | Local filesystem. |
| `storage_provider_s3` | Generic S3-compatible (AWS S3, MinIO, any S3-API service). |
| `storage_provider_r2` | Cloudflare R2 (dedicated bindings; use this in preference to the generic S3 provider when targeting R2). |
| `storage_provider_gcs` | Google Cloud Storage. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithStorageProvider(name, provider)` | Registers a provider under a name. |
| `piko.WithDefaultStorageProvider(name)` | Marks a registered provider as the default. |
| `piko.WithStorageDispatcher(cfg)` | Enables the background dispatcher. |
| `piko.WithStoragePresignBaseURL(url)` | Overrides the base URL for pre-signed download responses. |
| `piko.WithStoragePublicBaseURL(url)` | Sets the public URL prefix for publicly readable objects. |

## Constants

| Name | Value |
|---|---|
| `StorageNameDefault` | Name for the Piko default provider. |
| `StorageNameSystem` | Name reserved for system-managed storage. |
| `StorageRepositoryDefault` | Default repository (bucket/directory) name. |

## See also

- [How to assets](../how-to/assets.md) for responsive-image recipes that use the storage service.
- [How to security](../how-to/security.md) for encryption-at-rest with the crypto transformer.
- [Scenario 012: S3 file upload](../../examples/scenarios/012_file_upload/) for a runnable example.
