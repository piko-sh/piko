# WDK Security and Crypto

Use this guide when adding encryption, configuring security headers, rate limiting, or managing secrets.

## Crypto

The crypto package provides a unified encryption API with envelope encryption, streaming, key rotation, and multiple KMS providers.

### Supported providers

| Provider | Package | Use case |
|----------|---------|----------|
| Local AES-GCM | `crypto_provider_local_aes_gcm` | Development, single-server |
| AWS KMS | `crypto_provider_aws_kms` | Production AWS |
| GCP KMS | `crypto_provider_gcp_kms` | Production Google Cloud |

### Basic encryption

```go
import "piko.sh/piko/wdk/crypto"

// Convenience functions (use the default service)
encrypted, err := crypto.Encrypt(ctx, "sensitive-data")
plaintext, err := crypto.Decrypt(ctx, encrypted)
```

### Setup with local AES-GCM

```go
import "piko.sh/piko/wdk/crypto/crypto_provider_local_aes_gcm"

// 32-byte key: openssl rand -base64 32
provider, err := crypto_provider_local_aes_gcm.NewProvider(
    crypto_provider_local_aes_gcm.Config{
        Key:   key,      // []byte, exactly 32 bytes
        KeyID: "my-key",
    },
)

app := piko.New(
    piko.WithCryptoProvider("local", provider),
    piko.WithDefaultCryptoProvider("local"),
)
```

### Setup with AWS KMS

```go
import "piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"

provider, err := crypto_provider_aws_kms.NewProvider(ctx,
    crypto_provider_aws_kms.Config{
        KeyID:  "alias/my-app-key",
        Region: "eu-west-1",
    },
)
```

### Builder API

For advanced configuration, use the fluent builder:

```go
encrypted, err := crypto.NewEncryptBuilder(ctx, "sensitive-data").
    WithProvider("local").
    Do()
```

### Batch encryption (envelope encryption)

With envelope encryption (default), ONE KMS call encrypts the entire batch:

```go
svc, _ := crypto.GetDefaultService()
encrypted, err := svc.EncryptBatch(ctx, []string{"token1", "token2", "token3"})
decrypted, err := svc.DecryptBatch(ctx, encrypted)
```

### Streaming encryption

Memory stays constant (~64KB) regardless of file size:

```go
writer, err := svc.EncryptStream(ctx, outputFile, "key-id")
io.Copy(writer, largeInputFile)
writer.Close()

reader, err := svc.DecryptStream(ctx, encryptedFile)
plaintext, _ := io.ReadAll(reader)
```

### Key rotation

```go
err := svc.RotateKey(ctx, "old-key-id", "new-key-id")
// Old data still decryptable; new encryptions use the new key
```

## Security headers

Piko applies OWASP-recommended headers by default. Configure in `piko.yaml`:

```yaml
security:
  headers:
    enabled: true
    xFrameOptions: DENY
    xContentTypeOptions: nosniff
    contentSecurityPolicy: "default-src 'self'"
    stripServerHeader: true
```

Default headers: `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`, `Cross-Origin-Opener-Policy: same-origin`, `Strict-Transport-Security` (when HTTPS forced).

## Rate limiting

Disabled by default. Enable in `piko.yaml`:

```yaml
security:
  rateLimit:
    enabled: true
    storage: memory           # or "redis" for distributed
    trustedProxies:
      - "10.0.0.0/8"
    global:
      requestsPerMinute: 1000
      burstSize: 50
    actions:
      requestsPerMinute: 100
      burstSize: 20
    headersEnabled: true
```

Response headers when enabled: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, `Retry-After`.

## Cookie security

```yaml
security:
  cookies:
    forceHttpOnly: true
    forceSecureOnHttps: true
    defaultSameSite: Lax
```

## Filesystem sandboxing

Piko uses Go's `os.Root` for kernel-level path traversal protection:

```yaml
security:
  sandbox:
    enabled: true
    allowedPaths:
      - "/tmp/piko-cache"
```

## Secret resolution

String config fields can reference secrets via resolver prefixes:

| Prefix | Source | Example |
|--------|--------|---------|
| `env:` | Environment variable | `env:DATABASE_URL` |
| `file:` | File contents | `file:/run/secrets/api-key` |
| `base64:` | Base64 decode | `base64:c2VjcmV0` |
| `aws-secret:` | AWS Secrets Manager | `aws-secret:prod/db#password` |
| `gcp-secret:` | GCP Secret Manager | `gcp-secret:projects/p/secrets/s/versions/latest` |
| `azure-kv:` | Azure Key Vault | `azure-kv:my-vault/secret-name` |
| `vault:` | HashiCorp Vault | `vault:secret/data/prod/db#password` |
| `kubernetes-secret:` | Kubernetes Secrets | `kubernetes-secret:ns/secret#key` |

Register cloud resolvers:

```go
import "piko.sh/piko/wdk/config/config_resolver_aws"

func init() {
    config_resolver_aws.Register(context.Background())
}

type Config struct {
    DBPassword string `default:"aws-secret:prod/database/password"`
}
```

## LLM mistake checklist

- Using a key shorter or longer than 32 bytes for AES-GCM
- Committing encryption keys to version control
- Enabling rate limiting without setting `trustedProxies` (rate-limits your reverse proxy)
- Forgetting to register cloud secret resolvers in `init()`
- Using `#key` syntax for JSON key extraction without the provider supporting it
- Not closing the crypto service on shutdown (`svc.Close(ctx)`)
- Using streaming encryption from multiple goroutines (not thread-safe)

## Related

- `references/wdk-data.md` - storage encryption via transformers
- `references/project-structure.md` - piko.yaml configuration
- `references/configuration.md` - config loading and secret resolution
