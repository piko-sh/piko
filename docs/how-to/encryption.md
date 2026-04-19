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

For files or large payloads, use the streaming builder:

```go
import "piko.sh/piko/wdk/crypto"

func encryptUpload(ctx context.Context, in io.Reader, out io.Writer) error {
    builder, err := crypto.NewStreamEncryptBuilderFromDefault()
    if err != nil {
        return err
    }
    return builder.From(in).To(out).Do(ctx)
}
```

Memory usage stays around 64 KB regardless of file size.

## Register a non-default backend

Local AES-256-GCM is the default. For KMS-backed keys, register a provider at bootstrap:

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/crypto/crypto_provider_aws_kms"
)

func main() {
    ssr := piko.New(
        piko.WithCryptoProvider("kms", crypto_provider_aws_kms.New("alias/app-data")),
        piko.WithDefaultCryptoProvider("kms"),
    )
    ssr.Run()
}
```

The KMS provider caches data keys through the cache service. Ensure you also register a cache provider.

## Rotate a key

Rotations happen behind the API. When the configured rotation policy fires, the service marks the old key `KeyStatusDeprecated` and mints a new one. Ciphertexts encrypted with the deprecated key still decrypt, and new ciphertexts use the fresh key. No application code changes.

## See also

- [Crypto API reference](../reference/crypto-api.md).
- [Secrets API reference](../reference/secrets-api.md) for the inputs the crypto service consumes.
- [How to security](security.md) for broader hardening.
