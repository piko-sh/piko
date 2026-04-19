---
title: "012: S3 file upload"
description: File uploads to S3-compatible storage with direct and presigned URL patterns
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 130
---

# 012: S3 file upload

Upload files to S3-compatible storage using Piko's storage system. The server self-hosts a LocalStack testcontainer, so S3 works with no external setup beyond Docker.

The page shows two upload patterns. The direct multipart upload at `/` sends a file through multipart form data to a Piko action, which streams it to S3 through `UploadBuilder`. The presigned URL upload at `/presigned` asks the server for a presigned PUT URL, then uploads directly to S3 through XHR with progress tracking.

## What this demonstrates

At startup, `piko.WithStorageProvider` registers an S3 storage provider, and `piko.WithStoragePresignBaseURL` configures presigned URL routing. Actions receive uploaded files through `piko.FileUpload` from multipart form data. The `storage.NewUploadBuilder` helper streams file content to S3. The `svc.GeneratePresignedUploadURL` call issues presigned PUT URLs for client-side uploads. The `storage.NewRequestBuilder` helper runs stat, copy, and remove operations on stored objects.

## Project structure

```text
src/
  cmd/main/
    main.go                 Starts LocalStack, configures S3 provider, runs Piko
  actions/files/
    upload.go               Direct multipart upload action
    request_upload.go       Presigned URL generation action
    finalize_upload.go      Presigned upload finalisation action
    list.go                 Returns all uploaded files
    registry.go             In-memory file registry
  pages/
    index.pk                Direct upload page
    presigned.pk            Presigned URL upload page with progress bar
```

## How it works

At startup, `main.go` launches a LocalStack container, creates an S3 bucket, and passes the provider to Piko via `WithStorageProvider("default", s3Provider)`.

In the direct flow, the user selects a file and the browser sends multipart form data. The action streams the data to S3 through `UploadBuilder`, then registers the file in an in-memory list.

In the presigned flow, the browser asks the server for a presigned PUT URL. The browser then PUTs the file directly to S3 with XHR progress tracking and calls a finalise action to verify and register the file.

## How to run this example

This is a runnable project in the Piko repository at
`examples/scenarios/012_file_upload/`. Requires Docker for the LocalStack container.

```bash
cd examples/scenarios/012_file_upload/src
go run ./cmd/main/
# Open http://localhost:8080
```

Integration test:

```bash
cd examples
GOWORK=off go test -v -tags 'cgo,integration' -run 'TestExamples_Integration/012_' -count=1 .
```

## See also

- [Storage API reference](../reference/storage-api.md).
- [How to file storage](../how-to/file-storage.md).
