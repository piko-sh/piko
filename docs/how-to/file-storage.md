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
    diskProvider, err := storage_provider_disk.NewDiskProvider(storage_provider_disk.Config{
        BaseDirectory: "./uploads",
    })
    if err != nil {
        panic(err)
    }

    ssr := piko.New(
        piko.WithStorageProvider("default", diskProvider),
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
    File piko.FileUpload `json:"file"`
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

    key := "uploads/" + input.File.Name
    if err := builder.Key(key).ContentType(input.File.ContentType).Do(a.Ctx()); err != nil {
        return UploadResponse{}, err
    }

    return UploadResponse{Key: key}, nil
}
```

`piko.FileUpload` exposes `Name`, `ContentType`, and `Size` fields plus the `Open()` and `ReadAll()` methods.

## Return a presigned download URL

Presigning lives on the storage service, not the request builder. Call `GeneratePresignedDownloadURL` directly:

```go
import (
    "context"
    "time"

    "piko.sh/piko/wdk/storage"
)

func presignedURL(ctx context.Context, key string) (string, error) {
    service, err := storage.GetDefaultService()
    if err != nil {
        return "", err
    }
    return service.GeneratePresignedDownloadURL(ctx, "default", storage.PresignDownloadParams{
        Repository: "default",
        Key:        key,
        ExpiresIn:  15 * time.Minute,
    })
}
```

Presigned URLs are time-limited. Pick a TTL that matches the user flow (short for single-use downloads, longer for embedded media).

## Swap to S3

Change the bootstrap options:

```go
import (
    "context"

    "piko.sh/piko"
    "piko.sh/piko/wdk/storage/storage_provider_s3"
)

ctx := context.Background()
s3Provider, err := storage_provider_s3.NewS3Provider(ctx, &storage_provider_s3.Config{
    Region: "eu-west-1",
    RepositoryMappings: map[string]string{
        "default": "my-app-uploads",
    },
})
if err != nil {
    panic(err)
}

ssr := piko.New(
    piko.WithStorageProvider("default", s3Provider),
    piko.WithDefaultStorageProvider("default"),
)
```

Application code stays the same. The same upload and presign calls now hit S3.

## Encrypt uploads

Attach a transformer by name on the upload builder. Piko registers the crypto transformer once you bootstrap a crypto provider.

```go
builder.
    Key(key).
    ContentType(input.File.ContentType).
    Transformer("crypto-service").
    Do(ctx)
```

`Transformer(name string, options ...any)` accepts a transformer name plus optional transformer-specific options. Reads automatically reverse the transformation when the metadata records the transformer used at write time.

## See also

- [Storage API reference](../reference/storage-api.md) for the full API.
- [How to assets](assets.md) for image-specific pipelines.
- [Scenario 012: S3 file upload](../../examples/scenarios/012_file_upload/) for a runnable example with direct and presigned flows.
