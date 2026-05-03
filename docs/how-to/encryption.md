---
title: How to encrypt data at rest
description: Encrypt and decrypt strings and streams using Piko's crypto service with key rotation.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 70
---

# How to encrypt data at rest

This guide covers encrypting small fields, large streams, and batch payloads with Piko's crypto service. See the [crypto reference](../reference/crypto-api.md) for the full API.

## Encrypt a single field

```go
package customer

import (
    "context"

    "piko.sh/piko/wdk/crypto"
)

func storeSSN(ctx context.Context, customerID, ssn string) error {
    ciphertext, err := crypto.Encrypt(ctx, ssn)
    if err != nil {
        return err
    }
    return saveSSNCipher(ctx, customerID, ciphertext)
}

func readSSN(ctx context.Context, customerID string) (string, error) {
    ciphertext, err := loadSSNCipher(ctx, customerID)
    if err != nil {
        return "", err
    }
    return crypto.Decrypt(ctx, ciphertext)
}
```

Ciphertext is self-describing. Piko embeds the key ID and provider type, so `Decrypt` works across key rotations without schema changes.

## Encrypt a batch of fields at once

`EncryptBatch` wraps one data key for all inputs, which is cheaper for KMS backends:

```go
func storeManySSNs(ctx context.Context, inputs map[string]string) error {
    keys := make([]string, 0, len(inputs))
    plaintexts := make([]string, 0, len(inputs))
    for k, v := range inputs {
        keys = append(keys, k)
        plaintexts = append(plaintexts, v)
    }

    ciphertexts, err := crypto.EncryptBatch(ctx, plaintexts)
    if err != nil {
        return err
    }

    for i, k := range keys {
        if err := saveSSNCipher(ctx, k, ciphertexts[i]); err != nil {
            return err
        }
    }
    return nil
}
```

## Encrypt a stream

For files or large payloads, use the streaming builder. `Stream(ctx)` returns an `io.WriteCloser`. Write plaintext into it, and the builder writes the encrypted bytes to the destination passed to `Output`.

```go
import (
    "io"

    "piko.sh/piko/wdk/crypto"
)

func encryptUpload(ctx context.Context, in io.Reader, out io.Writer) error {
    builder, err := crypto.NewStreamEncryptBuilderFromDefault()
    if err != nil {
        return err
    }

    encryptor, err := builder.Output(out).Stream(ctx)
    if err != nil {
        return err
    }
    defer encryptor.Close()

    if _, err := io.Copy(encryptor, in); err != nil {
        return err
    }
    return encryptor.Close()
}
```

Call `KeyID(...)` before `Stream(ctx)` to encrypt with a specific key. Otherwise the call uses the active key on the default service. Memory usage stays around 64 KB regardless of file size.

## Register a non-default backend

Local AES-256-GCM is the default. For KMS-backed keys, register a provider at bootstrap:

```go
package main

import (
    "context"

    "piko.sh/piko"
    "piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"
)

func main() {
    ctx := context.Background()
    kmsProvider, err := crypto_provider_aws_kms.NewProvider(ctx, crypto_provider_aws_kms.Config{
        KeyID:  "alias/app-data",
        Region: "eu-west-1",
    })
    if err != nil {
        panic(err)
    }

    ssr := piko.New(
        piko.WithCryptoProvider("kms", kmsProvider),
        piko.WithDefaultCryptoProvider("kms"),
    )
    ssr.Run()
}
```

`Config.KeyID` accepts a key ID, key `Amazon Resource Name` (`ARN`), alias name (`alias/...`), or alias `ARN`. You must set `Region`. The provider caches data keys through the cache service, so register a cache provider too.

## Rotate a key

Rotations happen behind the API. When the configured rotation policy fires, the service marks the old key `KeyStatusDeprecated` and mints a new one. Ciphertexts encrypted with the deprecated key still decrypt, and new ciphertexts use the fresh key. No application code changes.

## See also

- [Crypto API reference](../reference/crypto-api.md).
- [Secrets API reference](../reference/secrets-api.md) for the inputs the crypto service consumes.
- [How to security](security.md) for broader hardening.
