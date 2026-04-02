---
title: "012: S3 file upload"
description: File uploads to S3-compatible storage with direct and presigned URL patterns
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 130
---

# 012: S3 file upload

Upload files to S3-compatible storage using Piko's storage system. The server self-hosts a LocalStack testcontainer so S3 works out of the box with no external setup (requires Docker).

Two upload patterns are demonstrated:

- **Direct multipart upload** (`/`): file sent via multipart form data to a Piko action, streamed to S3 via `UploadBuilder`
- **Presigned URL upload** (`/presigned`): client requests a presigned PUT URL, uploads directly to S3 via XHR with progress tracking

## What this demonstrates

- **`piko.WithStorageProvider`**: registering an S3 storage provider at startup
- **`piko.WithStoragePresignBaseURL`**: configuring presigned URL routing
- **`piko.FileUpload`**: receiving uploaded files in actions via multipart form data
- **`storage.NewUploadBuilder`**: streaming file content to S3
- **`svc.GeneratePresignedUploadURL`**: generating presigned PUT URLs for client-side uploads
- **`storage.NewRequestBuilder`**: stat, copy, and remove operations on stored objects

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

**Direct flow**: User selects a file, the browser sends multipart form data, the action streams it to S3 via `UploadBuilder`, then registers it in an in-memory file list.

**Presigned flow**: Browser requests a presigned PUT URL from the server, PUTs the file directly to S3 (with XHR progress tracking), then calls a finalise action to verify and register the file.

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
