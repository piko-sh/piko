// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package daemon_dto

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"sync/atomic"
)

// DefaultMaxUploadFileBytes is the default per-file size cap (32 MiB) applied
// when ReadAll is called without an explicit maximum. Operators can override
// the package-level cap via SetDefaultMaxUploadFileBytes.
const DefaultMaxUploadFileBytes int64 = 32 << 20

// ErrFileUploadTooLarge is returned when a multipart upload's contents exceed
// the configured per-file size cap during ReadAll. Callers can use
// errors.Is to detect this condition without parsing the message.
var ErrFileUploadTooLarge = errors.New("uploaded file exceeds maximum allowed size")

// defaultMaxUploadFileBytes is the package-level cap consulted by ReadAll when
// no explicit maximum is supplied. Stored as int64 via atomic for safe
// concurrent reads and writes from tests or operator hooks.
var defaultMaxUploadFileBytes atomic.Int64

func init() {
	defaultMaxUploadFileBytes.Store(DefaultMaxUploadFileBytes)
}

// SetDefaultMaxUploadFileBytes overrides the package-level per-file size cap
// used by ReadAll when no explicit maximum is supplied. A non-positive value
// resets the cap to DefaultMaxUploadFileBytes.
//
// Takes maxBytes (int64) which is the new default cap in bytes.
func SetDefaultMaxUploadFileBytes(maxBytes int64) {
	if maxBytes <= 0 {
		defaultMaxUploadFileBytes.Store(DefaultMaxUploadFileBytes)
		return
	}
	defaultMaxUploadFileBytes.Store(maxBytes)
}

// CurrentMaxUploadFileBytes returns the active package-level per-file size
// cap consulted by ReadAll.
//
// Returns int64 which is the current cap in bytes.
func CurrentMaxUploadFileBytes() int64 {
	return defaultMaxUploadFileBytes.Load()
}

// FileUpload represents an uploaded file from a multipart form request.
// The action parser recognises uploads and generates appropriate multipart
// handling code in the wrapper functions.
type FileUpload struct {
	// header is the underlying multipart.FileHeader for advanced use.
	header *multipart.FileHeader

	// Name is the original filename as provided by the client.
	Name string

	// ContentType is the MIME type from the Content-Type header.
	ContentType string

	// Size is the file size in bytes.
	Size int64
}

// NewFileUpload creates a FileUpload from a multipart.FileHeader.
//
// This is called by generated wrapper code; users typically do not call this
// directly.
//
// Takes header (*multipart.FileHeader) which provides the uploaded file
// metadata.
//
// Returns FileUpload which contains the file information ready for processing.
func NewFileUpload(header *multipart.FileHeader) FileUpload {
	contentType := ""
	if header.Header != nil {
		contentType = header.Header.Get("Content-Type")
	}
	return FileUpload{
		Name:        header.Filename,
		Size:        header.Size,
		ContentType: contentType,
		header:      header,
	}
}

// Open returns a reader for the file content.
// The caller is responsible for closing the returned ReadCloser.
//
// Returns io.ReadCloser which provides access to the file content.
// Returns error when the underlying file cannot be opened.
func (f *FileUpload) Open() (io.ReadCloser, error) {
	if f.header == nil {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	return f.header.Open()
}

// ReadAll reads the entire file into memory using the package-level default
// per-file size cap.
//
// Use this for small files; for large files, prefer Open to stream. To
// override the cap for a single call, use ReadAllWithLimit.
//
// Returns []byte which contains the complete file contents.
// Returns error when the file cannot be opened or read, or wraps
// ErrFileUploadTooLarge when the file exceeds the configured cap.
func (f *FileUpload) ReadAll() ([]byte, error) {
	return f.ReadAllWithLimit(defaultMaxUploadFileBytes.Load())
}

// ReadAllWithLimit reads the entire file into memory but rejects payloads that
// exceed the supplied per-file size cap.
//
// A non-positive maxBytes falls back to the package-level default.
//
// Takes maxBytes (int64) which is the maximum number of bytes accepted from
// the uploaded file.
//
// Returns []byte which contains the complete file contents when within the
// cap.
// Returns error when the file cannot be opened, the read fails, or the
// upload exceeds maxBytes (in which case the error wraps ErrFileUploadTooLarge
// and identifies the offending filename).
func (f *FileUpload) ReadAllWithLimit(maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxUploadFileBytes.Load()
	}

	file, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("opening uploaded file %q: %w", f.Name, err)
	}
	defer func() { _ = file.Close() }()

	limited := io.LimitReader(file, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("reading uploaded file %q: %w", f.Name, err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("upload %q exceeds %d byte limit: %w", f.Name, maxBytes, ErrFileUploadTooLarge)
	}
	return data, nil
}

// Header returns the underlying multipart.FileHeader for advanced use cases
// that need direct access to the original header (e.g., accessing custom headers).
//
// Returns nil if the FileUpload was not created from a multipart form.
func (f *FileUpload) Header() *multipart.FileHeader {
	return f.header
}

// RawBody provides access to the unparsed request body. It implements
// fmt.Stringer.
//
// Use this when you need to verify signatures, parse custom formats, or
// access the exact bytes sent by the client.
type RawBody struct {
	// ContentType is the Content-Type header of the request.
	ContentType string

	// data holds the raw body bytes.
	data []byte

	// Size is the body size in bytes.
	Size int64
}

// NewRawBody creates a RawBody from raw data.
// This is called by the action handler; users typically do not call this
// directly.
//
// Takes contentType (string) which specifies the MIME type of the data.
// Takes data ([]byte) which contains the raw body content.
//
// Returns RawBody which wraps the data with its content type and size.
func NewRawBody(contentType string, data []byte) RawBody {
	return RawBody{
		ContentType: contentType,
		Size:        int64(len(data)),
		data:        data,
	}
}

// Bytes returns the raw body data.
//
// Returns []byte which is the underlying data. The returned slice should not
// be modified.
func (r *RawBody) Bytes() []byte {
	return r.data
}

// String returns the raw body as a string.
// This is a convenience method for text-based bodies.
//
// Returns string which contains the raw body data.
func (r *RawBody) String() string {
	return string(r.data)
}

// Reader returns an io.Reader for the body.
// Useful when passing to parsers that accept io.Reader.
//
// Returns io.Reader which provides access to the raw body data.
func (r *RawBody) Reader() io.Reader {
	return bytes.NewReader(r.data)
}
