# WDK Data Services

Database access, file storage, and caching. All packages follow hexagonal architecture - the application depends on the port; swap adapters to switch backends.

## Database (`wdk/db`)

`wdk/db` provides typed SQL access via named registrations made during bootstrap. Drivers return a standard `*sql.DB`; engines supply dialect configuration.

### Drivers

| Driver | Package | Constructor |
|--------|---------|-------------|
| SQLite (CGO, fastest) | `wdk/db/db_driver_sqlite_cgo` | `Open(path string, cfg Config) (*sql.DB, error)` |
| SQLite (pure Go) | `wdk/db/db_driver_sqlite_nocgo` | `Open(path string, cfg Config) (*sql.DB, error)` |
| Cloudflare D1 | `wdk/db/db_driver_d1` | `Open(cfg Config) (*sql.DB, error)` |
| PostgreSQL / MySQL / others | stdlib | `sql.Open(driverName, dsn)` |

No `wdk/db` driver subpackage exists for Postgres or MySQL - open them via `database/sql` with a third-party driver (e.g. `github.com/jackc/pgx/v5/stdlib`).

### Engines

Each engine package exposes a factory returning `db.EngineConfig`:

| Engine | Package | Factory |
|--------|---------|---------|
| SQLite | `wdk/db/db_engine_sqlite` | `SQLite()` |
| PostgreSQL | `wdk/db/db_engine_postgres` | `Postgres()` / `PostgresPgBouncer()` |
| MySQL | `wdk/db/db_engine_mysql` | `MySQL()` |
| MariaDB | `wdk/db/db_engine_mariadb` | `MariaDB()` |
| CockroachDB | `wdk/db/db_engine_cockroachdb` | `CockroachDB()` |
| DuckDB | `wdk/db/db_engine_duckdb` | `DuckDB()` |

### Bootstrap

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_driver_sqlite_cgo"
    "piko.sh/piko/wdk/db/db_engine_sqlite"
)

sqlDB, err := db_driver_sqlite_cgo.Open("data/app.db", db_driver_sqlite_cgo.Config{})
// handle err

app := piko.New(
    piko.WithDatabase("default", &db.DatabaseRegistration{
        DB:           sqlDB,
        EngineConfig: db_engine_sqlite.SQLite(),
        MigrationFS:  migrationsFS,
    }),
)
```

Reserved names `db.DatabaseNameRegistry` and `db.DatabaseNameOrchestrator` back framework subsystems with SQL instead of the default in-memory store.

### Connections and migrations

Concurrency-safe accessors: `db.GetDatabaseConnection(name) (*sql.DB, error)`, `db.GetDatabaseReader(name) (DBTX, error)` (round-robin replicas), `db.GetDatabaseWriter(name) (DBTX, error)`.

Bootstrap auto-runs migrations when `MigrationFS` is set. To build manually:

```go
executor := db.NewMigrationExecutor(sqlDB, db.SQLiteDialect())
reader := db.NewFSFileReader(migrationsFS)
service := db.NewMigrationService(executor, reader, "migrations")
```

D1 migrations must use the Wrangler CLI - the framework cannot run them remotely.

## Storage (`wdk/storage`)

### Providers

| Provider | Package | Constructor |
|----------|---------|-------------|
| Disk | `storage_provider_disk` | `NewDiskProvider(config, opts...)` |
| S3 | `storage_provider_s3` | `NewS3Provider(ctx, *Config, opts...)` |
| GCS | `storage_provider_gcs` | `NewGCSProvider(ctx, config, opts...)` |
| R2 | `storage_provider_r2` | `NewR2Provider(ctx, *Config, opts...)` |
| Mock (tests) | `storage_provider_mock` | - |

### Setup

```go
import "piko.sh/piko/wdk/storage/storage_provider_s3"

provider, err := storage_provider_s3.NewS3Provider(ctx, &storage_provider_s3.Config{
    Region:             "eu-west-1",
    RepositoryMappings: map[string]string{"default": "my-bucket"},
})
// handle err

app := piko.New(piko.WithStorageProvider("default", provider))
```

### Upload, download, presign

Builder methods are bare names. Upload terminator is `Do(ctx)`. Request terminators take only `ctx`. Presigned downloads live on the service.

```go
import "piko.sh/piko/wdk/storage"

service, _ := storage.GetDefaultService()

uploader, _ := storage.NewUploadBuilder(service, reader)
err := uploader.
    Key("documents/report.pdf").
    ContentType("application/pdf").
    Size(fileSize).
    Do(ctx)

request, _ := storage.NewRequestBuilder(service, storage.StorageRepositoryDefault, "documents/report.pdf")
readCloser, err := request.Get(ctx)
defer readCloser.Close()

url, err := service.GeneratePresignedDownloadURL(ctx, "default", storage.PresignDownloadParams{
    Key:       "documents/report.pdf",
    ExpiresIn: 1 * time.Hour,
})
```

Other request terminators: `Stat(ctx) (*ObjectInfo, error)`, `Remove(ctx) error`, `Hash(ctx) (string, error)`.

### Stream transformers

Register on the service, opt in per upload. Run in priority order on upload (compress then encrypt) and reverse on download.

```go
import "piko.sh/piko/wdk/storage/storage_transformer_zstd"

transformer, _ := storage_transformer_zstd.NewZstdTransformer(storage_transformer_zstd.Config{})
_ = service.RegisterTransformer(ctx, transformer)

err = uploader.Key("backup.sql.zst").Transformer("zstd", nil).Do(ctx)
```

## Cache (`wdk/cache`)

Type-safe caching with generic key/value parameters.

### Providers

| Provider | Package | Use case |
|----------|---------|----------|
| Otter | `cache_provider_otter` | In-memory, S3-FIFO, high performance |
| Redis | `cache_provider_redis` | Distributed |
| Redis Cluster | `cache_provider_redis_cluster` | Horizontally scaled Redis |
| Valkey | `cache_provider_valkey` | Distributed (Redis-compatible OSS fork) |
| Valkey Cluster | `cache_provider_valkey_cluster` | Horizontally scaled Valkey |
| Multilevel | (built-in) | L1 in-memory + L2 distributed |
| Mock | `cache_provider_mock` | Unit testing |

### Setup and basic usage

Builder methods are bare names. `Build(ctx)`, `RegisterProvider(ctx, name, provider)`, and `RegisterTransformer(ctx, transformer)` all require ctx.

```go
import (
    "piko.sh/piko/wdk/cache"
    "piko.sh/piko/wdk/cache/cache_provider_otter"
)

service := cache.NewService("otter")
_ = service.RegisterProvider(ctx, "otter", cache_provider_otter.NewOtterProvider())

builder, _ := cache.NewCacheBuilder[string, User](service)
userCache, err := builder.
    Provider("otter").
    Namespace("users").
    MaximumSize(10000).
    WriteExpiration(1 * time.Hour).
    Build(ctx)
```

### Cache interface

All operations take `ctx`; mutations return `error`. `Get` with a loader prevents stampede - one in-flight load per key. Use `Compute(ctx, key, fn)` for atomic read-modify-write.

```go
if err := userCache.Set(ctx, "user:123", user, "user", "active"); err != nil {
    return err
}

value, found, err := userCache.GetIfPresent(ctx, "user:123")

loader := cache.LoaderFunc[string, User](func(ctx context.Context, key string) (User, error) {
    return loadUserFromDB(ctx, key)
})
user, err := userCache.Get(ctx, "user:123", loader)

removed, err := userCache.InvalidateByTags(ctx, "user")
```

### Multilevel and search

For L1+L2: chain `.MultiLevel("otter", "redis").L2CircuitBreaker(5, 30*time.Second)` before `Build(ctx)`.

For searchable caches: build a `*SearchSchema` with field constructors (`TagField`, `NumericField`, `TextField`, `GeoField`, `VectorField(name, dim)`, `VectorFieldWithMetric(name, dim, metric)`), pass to `.Searchable(schema)` on the builder, then call `cache.Search(ctx, query, *SearchOptions)`.

## LLM mistake checklist

- Importing `wdk/persistence` - the package is `wdk/db`
- Calling `db.WithDriver` - use `piko.WithDatabase(name, &db.DatabaseRegistration{...})`
- Adding `With*` prefixes on upload builders - methods are bare (`Key`, `ContentType`, `Size`, `Transformer`)
- Calling `.Upload(ctx, log)` - the terminator is `.Do(ctx)`
- Passing `log` to request builder terminators - they take only `ctx`
- Calling `.PresignDownload(...)` on `RequestBuilder` - it lives on the service (`service.GeneratePresignedDownloadURL`)
- Forgetting `ctx` on `Cache.Set`/`GetIfPresent`/`InvalidateByTags`, builder `Build`, `RegisterProvider`, `RegisterTransformer`
- Ignoring the `error` from `Set`, `(int, error)` from `InvalidateByTags`, or `(V, bool, error)` from `GetIfPresent`
- `GetIfPresent` + manual `Set` instead of `Get` with loader (causes stampede)
- Importing `db_driver_sqlite_cgo` without CGO - use `db_driver_sqlite_nocgo`

## Related

- `references/project-structure.md` - where config files live
- `references/server-actions.md` - using data services from actions
- `docs/reference/cache-api.md`, `docs/reference/storage-api.md`, `docs/reference/querier.md` - full API references
