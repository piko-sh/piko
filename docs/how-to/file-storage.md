---
title: How to upload and serve files
description: Register a storage provider, upload files from actions, and return presigned download URLs.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 80
---

# How to upload and serve files

This guide shows how to upload a file from a server action, return a download link, and swap the backend between local disk and S3. See the [storage reference](../reference/storage-api.md) for the full API.

## Register a disk provider for local development

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/storage/storage_provider_disk"
)

func main() {
    ssr := piko.New(
        piko.WithStorageProvider("default", storage_provider_disk.New("./uploads")),
        piko.WithDefaultStorageProvider("default"),
    )
    ssr.Run()
}
```

## Upload from an action

```go
package upload

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/storage"
)

type UploadInput struct {
    File *piko.FormFile `json:"file"`
}

type UploadResponse struct {
    Key string `json:"key"`
}

type UploadAction struct {
    piko.ActionMetadata
}

func (a *UploadAction) Call(input UploadInput) (UploadResponse, error) {
    reader, err := input.File.Open()
    if err != nil {
        return UploadResponse{}, err
    }
    defer reader.Close()

    builder, err := storage.NewUploadBuilderFromDefault(reader)
    if err != nil {
        return UploadResponse{}, err
    }

    key := "uploads/" + input.File.Filename()
    if err := builder.Key(key).ContentType(input.File.ContentType()).Do(a.Ctx()); err != nil {
        return UploadResponse{}, err
    }

    return UploadResponse{Key: key}, nil
}
```

## Return a presigned download URL

```go
func presignedURL(ctx context.Context, key string) (string, error) {
    builder, err := storage.NewRequestBuilderFromDefault("default", key)
    if err != nil {
        return "", err
    }
    return builder.Presign(15 * time.Minute).Do(ctx)
}
```

Presign URLs are time-limited. Pick a TTL that matches the user flow (short for single-use downloads, longer for embedded media).

## Swap to S3

Change the bootstrap options:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/storage/storage_provider_s3"
)

ssr := piko.New(
    piko.WithStorageProvider("default", storage_provider_s3.New(storage_provider_s3.Config{
        Region: "eu-west-1",
        Bucket: "my-app-uploads",
    })),
    piko.WithDefaultStorageProvider("default"),
)
```

Application code stays the same. The same upload and presign calls now hit S3.

## Encrypt uploads

Attach a transformer:

```go
import "piko.sh/piko/wdk/storage/storage_transformer_crypto"

builder.Transform(storage.TransformConfig{
    Type: storage.TransformerEncryption,
    Config: storage_transformer_crypto.Config{KeyID: "default"},
}).Do(ctx)
```

Reads automatically decrypt through the same transformer.

## See also

- [Storage API reference](../reference/storage-api.md) for the full API.
- [How to assets](assets.md) for image-specific pipelines.
- [Scenario 012: S3 file upload](../showcase/012-file-upload.md) for a runnable example with direct and presigned flows.
