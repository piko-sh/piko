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

package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// uploadLimit bounds uploaded, stored and downloaded fixture bytes.
	uploadLimit = 4096

	// storedUploadName is the stored upload file, relative to the working directory.
	storedUploadName = "upload-store/uploads/payload.bin"

	// binaryUploadMarker is a byte that is never valid UTF-8, so the upload payload is
	// binary rather than text.
	binaryUploadMarker = 0xff
)

// UploadRead records uploaded bytes, storage checks and public responses.
type UploadRead struct {
	// DiskDigest is the hex SHA-256 of the stored bytes.
	DiskDigest string

	// Name is the uploaded file name.
	Name string

	// Media is the uploaded media type.
	Media string

	// Digest is the hex SHA-256 of the uploaded bytes.
	Digest string

	// Removed records the removed upload name.
	Removed string `json:",omitempty"`

	// RangeDigest is the hex SHA-256 of the stored byte range.
	RangeDigest string `json:",omitempty"`

	// PublicDigest is the hex SHA-256 of the public download body.
	PublicDigest string `json:",omitempty"`

	// PublicMedia is the public download media type.
	PublicMedia string `json:",omitempty"`

	// DeletedPublicStatus is the public status after removing the upload.
	DeletedPublicStatus int `json:",omitempty"`

	// Size is the uploaded file size in bytes.
	Size int64

	// PublicStatus is the public resource HTTP status.
	PublicStatus int `json:",omitempty"`

	// PublicLength is the public resource length in bytes.
	PublicLength int64 `json:",omitempty"`

	// DiskAbsent reports whether upload removal deleted the stored file.
	DiskAbsent bool `json:",omitempty"`
}

// uploadRemoval is the decoded upload-removal action result.
type uploadRemoval struct {
	// Removed is the removed upload key.
	Removed string
}

// uploadPayload constructs deterministic upload bytes from the case input.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
//
// Returns []byte which holds the bounded source or response bytes.
func uploadPayload(input casefile.ActionValue) []byte {
	return append([]byte{binaryUploadMarker, 0, '\r', '\n'}, fmt.Appendf(nil, "%s\n%d", input.Label, input.Quantity)...)
}

// expectedUpload models upload metadata and delivered byte identities.
//
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
// Takes public (bool) which enables public upload delivery checks.
//
// Returns UploadRead which holds expected upload metadata and byte identities.
func expectedUpload(input casefile.ActionValue, public bool) UploadRead {
	data := uploadPayload(input)
	digest := sha256Hex(data)
	var value UploadRead
	value.Name, value.Media, value.Size, value.Digest, value.DiskDigest = "case.bin", "application/octet-stream", int64(len(data)), digest, digest
	if public {
		value.RangeDigest = sha256Hex(data[2 : len(data)-1])
		value.PublicStatus, value.PublicDigest, value.PublicMedia, value.PublicLength = http.StatusOK, digest, value.Media, value.Size
	}
	return value
}

// checkUploadedObject checks upload storage and public delivery.
//
// Takes want (*casefile.ActionContract) which holds the independently expected result.
// Takes got (*UploadRead) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkUploadedObject(want *casefile.ActionContract, got *UploadRead) error {
	expected := expectedUpload(want.Input, want.UploadPublic)
	if want.UploadRemoval {
		expected.Removed, expected.DeletedPublicStatus, expected.DiskAbsent = "payload.bin", http.StatusNotFound, true
		if got != nil {
			switch {
			case got.Removed != expected.Removed:
				return fmt.Errorf("action/delete-result: got %q, want %q", got.Removed, expected.Removed)
			case got.DeletedPublicStatus != expected.DeletedPublicStatus:
				return fmt.Errorf("action/deleted-public-status: got %d, want %d", got.DeletedPublicStatus, expected.DeletedPublicStatus)
			case !got.DiskAbsent:
				return errors.New("action/deleted-disk: object still present")
			}
		}
	}
	if got != nil && want.UploadPublic {
		switch {
		case got.RangeDigest != expected.RangeDigest:
			return fmt.Errorf("action/storage-range: got %q, want %q", got.RangeDigest, expected.RangeDigest)
		case got.PublicStatus != expected.PublicStatus:
			return fmt.Errorf("action/public-download-status: got %d, want %d", got.PublicStatus, expected.PublicStatus)
		case got.PublicMedia != expected.PublicMedia || got.PublicLength != expected.PublicLength:
			return fmt.Errorf("action/public-download-headers: got %q %d, want %q %d", got.PublicMedia, got.PublicLength, expected.PublicMedia, expected.PublicLength)
		case got.PublicDigest != expected.PublicDigest:
			return fmt.Errorf("action/public-download-body: got %q, want %q", got.PublicDigest, expected.PublicDigest)
		}
	}
	if got == nil || *got != expected {
		return fmt.Errorf("action/upload: got %+v, want %+v", got, expected)
	}
	return nil
}

// observeUpload uploads the generated payload and reads independent storage evidence.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes token (string) which is the action CSRF token.
// Takes ephemeral (string) which is the form's ephemeral CSRF token.
// Takes input (casefile.ActionValue) which supplies the action label and quantity.
// Takes public (bool) which enables public upload delivery checks.
//
// Returns *UploadRead which records uploaded bytes, independent storage checks and public
// delivery.
// Returns error which reports upload dispatch, independent storage or public-response
// failure, or nil on success.
func observeUpload(ctx context.Context, client *http.Client, base, token, ephemeral string, input casefile.ActionValue, public bool) (*UploadRead, error) {
	data := uploadPayload(input)
	if len(data) > uploadLimit {
		return nil, errors.New("action/upload-size: fixture exceeds 4 KiB")
	}
	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	if err := parts.WriteField("_csrf_ephemeral_token", ephemeral); err != nil {
		return nil, err
	}
	file, err := parts.CreateFormFile("file", "case.bin")
	if err != nil {
		return nil, err
	}
	if _, err = file.Write(data); err != nil {
		return nil, err
	}
	if err := parts.Close(); err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/fuzz.Store", &body)
	if err != nil {
		return nil, err
	}
	setActionHeaders(request, mediaJSON, parts.FormDataContentType(), token)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Join(fmt.Errorf("action/upload-status: HTTP %d", response.StatusCode), closeResponse(response))
	}
	var value UploadRead
	if err = decodeAction(response, &value); err != nil || !public {
		return &value, err
	}
	return &value, observePublicUpload(ctx, client, base, &value)
}

// observePublicUpload downloads the public upload anonymously and records its delivery.
//
// The public handler serves whole representations. ByteRange is verified separately
// through the WDK GetObject call in the generated action.
//
// Takes client (*http.Client) which is the HTTP client whose cookies are dropped.
// Takes base (string) which is the application base URL.
// Takes value (*UploadRead) which receives the public delivery evidence.
//
// Returns error which reports a failed or oversized public download, or nil on success.
func observePublicUpload(ctx context.Context, client *http.Client, base string, value *UploadRead) error {
	fresh := *client
	fresh.Jar = nil
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/_piko/storage/public/fuzz-upload/uploads/payload.bin", nil)
	if err != nil {
		return err
	}
	request.Host = requestHost
	publicResponse, err := fresh.Do(request)
	if err != nil {
		return fmt.Errorf("action/upload-public: %w", err)
	}
	value.PublicStatus = publicResponse.StatusCode
	value.PublicMedia = publicResponse.Header.Get(headerContentType)
	value.PublicLength = publicResponse.ContentLength
	persisted, err := io.ReadAll(io.LimitReader(publicResponse.Body, uploadLimit+1))
	if err = errors.Join(err, closeResponse(publicResponse)); err != nil {
		return fmt.Errorf("action/upload-public: %w", err)
	}
	if len(persisted) > uploadLimit {
		return errors.New("action/upload-public: download exceeds 4 KiB")
	}
	value.PublicDigest = sha256Hex(persisted)
	return nil
}

// observeUploadRemoval removes the upload and checks storage and public responses.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes page (string) which holds the rendered page markup.
// Takes work (string) which is the invocation-private working directory holding the
// upload store.
// Takes value (*UploadRead) which receives removal and independent storage evidence.
//
// Returns error which reports removal, storage-presence or public-response failure, or
// nil on success.
func observeUploadRemoval(ctx context.Context, client *http.Client, base, page, work string, value *UploadRead) error {
	var err error
	value.DiskDigest, err = readStoredUpload(work)
	if err != nil {
		return err
	}
	token, ephemeral, err := pageActionTokens(page)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"Key": "payload.bin", "_csrf_ephemeral_token": ephemeral})
	if err != nil {
		return fmt.Errorf("action/delete-encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/fuzz.RemoveUpload", bytes.NewReader(body))
	if err != nil {
		return err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, token)
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("action/delete-request: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return errors.Join(fmt.Errorf("action/delete-request: HTTP %d", response.StatusCode), closeResponse(response))
	}
	var result uploadRemoval
	if err := decodeAction(response, &result); err != nil {
		return err
	}
	value.Removed = result.Removed
	fresh := *client
	fresh.Jar = nil
	status, _, _, err := get(ctx, &fresh, base, "/_piko/storage/public/fuzz-upload/uploads/payload.bin")
	if err != nil {
		return fmt.Errorf("action/deleted-public-status: %w", err)
	}
	value.DeletedPublicStatus = status
	_, err = os.Stat(filepath.Join(work, storedUploadName))
	value.DiskAbsent = errors.Is(err, fs.ErrNotExist)
	if err != nil && !value.DiskAbsent {
		return fmt.Errorf("action/deleted-disk: %w", err)
	}
	return nil
}

// readStoredUpload hashes the bounded stored upload bytes.
//
// Takes work (string) which is the invocation-private working directory holding the
// upload store.
//
// Returns string which identifies the stored upload bytes.
// Returns error which reports an inaccessible or oversized stored upload, or nil on
// success.
func readStoredUpload(work string) (string, error) {
	data, err := readBoundedFile(work, storedUploadName, uploadLimit)
	if err != nil {
		return "", fmt.Errorf("action/upload-disk: %w", err)
	}
	return sha256Hex(data), nil
}

// sha256Hex returns the lowercase hexadecimal SHA-256 digest of data.
//
// Takes data ([]byte) which holds the bytes to hash.
//
// Returns string which is the hexadecimal digest.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
