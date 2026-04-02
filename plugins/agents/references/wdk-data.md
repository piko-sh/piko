# WDK Data Services

Use this guide when adding database persistence, file storage, or caching to a Piko application.

## Architecture pattern

All WDK data packages follow hexagonal architecture (ports and adapters). Your application depends on the port interface; the adapter implements it for a specific backend. Switching backends means swapping the import - no application code changes.

## Persistence

The persistence package provides database access for Piko's internal systems (Registry, Orchestrator). Register a driver during bootstrap:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/persistence"
    sqlite "piko.sh/piko/wdk/persistence/persistence_driver_sqlite"
)

app := piko.New(
    persistence.WithDriver(sqlite.New(sqlite.Config{})),
)
```

### Supported drivers

| Driver | Package | Best for |
|--------|---------|----------|
| SQLite (auto) | `persistence_driver_sqlite` | Development, single-server |
| SQLite (pure Go) | `persistence_driver_sqlite_nocgo` | Cross-compiling, no C compiler |
| SQLite (CGO) | `persistence_driver_sqlite_cgo` | Maximum performance |
| PostgreSQL | `persistence_driver_postgres` | Production, multi-server |
| Cloudflare D1 | `persistence_driver_d1` | Edge deployments |

### PostgreSQL example

```go
import postgres "piko.sh/piko/wdk/persistence/persistence_driver_postgres"

provider, err := postgres.New(postgres.Config{
    URL:      "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
    MaxConns: 10,
    MinConns: 2,
})
```

### Cloudflare D1 example

```go
import d1 "piko.sh/piko/wdk/persistence/persistence_driver_d1"

provider, err := d1.New(d1.Config{
    APIToken:   os.Getenv("CF_API_TOKEN"),
    AccountID:  os.Getenv("CF_ACCOUNT_ID"),
    DatabaseID: os.Getenv("CF_DATABASE_ID"),
})
```

Migrations run automatically for SQLite and PostgreSQL. D1 migrations must use the Wrangler CLI.

## Storage

The storage package provides object storage with a unified API across providers.

### Supported providers

| Provider | Package | Features |
|----------|---------|----------|
| Disk | `storage_provider_disk` | Atomic writes, metadata sidecars |
| S3 | `storage_provider_s3` | Presigned URLs, multipart uploads |
| GCS | `storage_provider_gcs` | Presigned URLs, multipart uploads |
| R2 | `storage_provider_r2` | Cloudflare R2 (S3-compatible) |

### Setup

```go
import "piko.sh/piko/wdk/storage/storage_provider_s3"

provider, err := storage_provider_s3.NewS3Provider(ctx, &storage_provider_s3.Config{
    Region: "eu-west-1",
    RepositoryMappings: map[string]string{
        "default": "my-bucket",
    },
})

app := piko.New(
    piko.WithStorageProvider("default", provider),
)
```

### Upload and download

```go
import "piko.sh/piko/wdk/storage"

// Upload
err := storage.NewUpload(reader).
    WithKey("documents/report.pdf").
    WithContentType("application/pdf").
    WithSize(fileSize).
    Upload(ctx, log)

// Download
reader, err := storage.NewRequest(storage.StorageRepositoryDefault, "documents/report.pdf").
    Get(ctx, log)

// Presigned download URL
url, err := storage.NewRequest(storage.StorageRepositoryDefault, key).
    PresignDownload(ctx, log, storage.PresignDownloadParams{
        ExpiresIn: 1 * time.Hour,
    })
```

### Stream transformers

Apply compression or encryption transparently:

```go
import "piko.sh/piko/wdk/storage/storage_transformer_zstd"

service.RegisterTransformer(compressor)

storage.NewUpload(data).
    WithKey("backup.sql.zst").
    WithTransformer("zstd", nil).
    Upload(ctx, log)
```

Transformers run in priority order on upload (compress then encrypt) and reverse on download.

## Cache

The cache package provides type-safe caching with multiple backends.

### Supported providers

| Provider | Package | Use case |
|----------|---------|----------|
| Otter | `cache_provider_otter` | In-memory, high-performance (S3-FIFO) |
| Redis | `cache_provider_redis` | Distributed |
| Redis Cluster | `cache_provider_redis_cluster` | Horizontally scaled |
| Multilevel | `cache_provider_multilevel` | L1 (Otter) + L2 (Redis) |
| Mock | `cache_provider_mock` | Unit testing |

### Basic usage

```go
import (
    "piko.sh/piko/wdk/cache"
    "piko.sh/piko/wdk/cache/cache_provider_otter"
)

svc := cache.NewService("otter")
provider := cache_provider_otter.NewOtterProvider()
svc.RegisterProvider("otter", provider)

userCache, err := cache.NewCacheBuilder[string, User](svc).
    Provider("otter").
    Namespace("users").
    MaximumSize(10000).
    WriteExpiration(1 * time.Hour).
    Build()

// Use the cache
userCache.Set("user:123", user)
if user, found := userCache.GetIfPresent("user:123"); found { ... }
```

### Cache with loader (prevents stampede)

```go
loader := cache.LoaderFunc[string, User](func(ctx context.Context, key string) (User, error) {
    return db.GetUser(ctx, key)
})
user, err := myCache.Get(ctx, "user:123", loader)
```

### Multilevel caching

```go
myCache, err := cache.NewCacheBuilder[string, User](svc).
    MultiLevel("otter", "redis").
    MaximumSize(10000).
    L2CircuitBreaker(5, 30*time.Second).
    Build()
```

### Tag-based invalidation

```go
myCache.Set("user:1", user, "user", "active")
myCache.InvalidateByTags("user")  // Removes all entries with "user" tag
```

## LLM mistake checklist

- Forgetting to register the provider before building caches
- Using `GetIfPresent` + manual set instead of `Get` with a loader (causes cache stampede)
- Mixing up `storage.StorageRepositoryDefault` with a custom repository name
- Forgetting `WithSize()` on uploads (disables automatic multipart for large files)
- Using `persistence_driver_sqlite_cgo` without CGO enabled
- Forgetting `defer provider.Close()` for PostgreSQL
- Not importing the correct driver subpackage (each is a separate Go module)

## Related

- `references/project-structure.md` - where config files live
- `references/server-actions.md` - using data services from actions
