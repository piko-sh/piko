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

package presign_http

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"piko.sh/piko/internal/contextaware"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// httpStatusOK is the HTTP status code for a successful response.
	httpStatusOK = http.StatusOK

	// httpStatusBadRequest indicates an invalid token format or missing
	// parameters.
	httpStatusBadRequest = http.StatusBadRequest

	// httpStatusUnauthorised is the HTTP 401 status code for invalid credentials.
	httpStatusUnauthorised = http.StatusUnauthorized

	// httpStatusForbidden is HTTP status 403, used for expired or reused tokens.
	httpStatusForbidden = http.StatusForbidden

	// httpStatusPayloadTooLarge is the HTTP status code for when a file is too
	// large.
	httpStatusPayloadTooLarge = http.StatusRequestEntityTooLarge

	// httpStatusUnsupportedMediaType indicates a mismatch between the expected
	// and actual Content-Type header values.
	httpStatusUnsupportedMediaType = http.StatusUnsupportedMediaType

	// httpStatusInternalError indicates a storage write failure.
	httpStatusInternalError = http.StatusInternalServerError

	// headerContentType is the HTTP header name for content type.
	headerContentType = "Content-Type"

	// logFieldKey is the log field name for storage object keys.
	logFieldKey = "key"

	// headerETag is the HTTP header name for the ETag field.
	headerETag = "ETag"

	// headerLastModified is the HTTP header name for the last modified time.
	headerLastModified = "Last-Modified"

	// headerIfNoneMatch is the HTTP header name for conditional requests using
	// ETags.
	headerIfNoneMatch = "If-None-Match"

	// headerIfModifiedSince is the HTTP header name for requests that check
	// whether a resource has changed since a given time.
	headerIfModifiedSince = "If-Modified-Since"

	// headerCacheControl is the HTTP header name for cache control settings.
	headerCacheControl = "Cache-Control"

	// headerContentDisposition is the HTTP header key for Content-Disposition.
	headerContentDisposition = "Content-Disposition"

	// defaultCacheControl is the Cache-Control header value used when no metadata
	// is set.
	defaultCacheControl = "private, max-age=3600"
)

var (
	// inlineTopLevelTypes lists MIME type categories (top-level types) that should
	// be displayed inline. These cover the majority of browser-renderable content.
	inlineTopLevelTypes = map[string]struct{}{
		"image": {},
		"video": {},
		"audio": {},
		"text":  {},
		"font":  {},
	}

	// inlineApplicationTypes lists specific application/* MIME types that browsers
	// can typically render inline.
	inlineApplicationTypes = map[string]struct{}{
		"application/pdf":                   {},
		"application/json":                  {},
		"application/xml":                   {},
		"application/javascript":            {},
		"application/ecmascript":            {},
		"application/x-javascript":          {},
		"application/xhtml+xml":             {},
		"application/rss+xml":               {},
		"application/atom+xml":              {},
		"application/mathml+xml":            {},
		"application/svg+xml":               {},
		"application/wasm":                  {},
		"application/manifest+json":         {},
		"application/ld+json":               {},
		"application/geo+json":              {},
		"application/vnd.api+json":          {},
		"application/x-www-form-urlencoded": {},
	}
)

// UploadResponse holds the data returned after a file upload succeeds.
type UploadResponse struct {
	// TempKey is the storage key where the file was uploaded.
	TempKey string `json:"temp_key"`

	// ContentType is the MIME type of the uploaded file.
	ContentType string `json:"content_type"`

	// Size is the number of bytes written.
	Size int64 `json:"size"`
}

// ErrorResponse holds the JSON structure returned when an upload fails.
type ErrorResponse struct {
	// Error is a short error code such as "token_expired" or "invalid_signature".
	Error string `json:"error"`

	// Message is a human-readable description of the error.
	Message string `json:"message"`
}

// errorWriteFunction is a callback that writes an HTTP error response.
type errorWriteFunction func(
	ctx context.Context,
	w http.ResponseWriter,
	status int,
	errorCode string,
	message string,
)

// Handler provides HTTP handling for presigned URL uploads.
// It implements http.Handler.
type Handler struct {
	// storageService writes uploaded files to storage.
	storageService storage_domain.Service

	// config holds the presign URL settings, including the secret and replay
	// cache.
	config storage_domain.PresignConfig
}

// NewHandler creates a new presigned upload handler.
//
// Takes storageService (storage_domain.Service) which handles file storage.
// Takes config (storage_domain.PresignConfig) which provides token verification
// settings.
//
// Returns *Handler which is ready to handle upload requests.
func NewHandler(
	storageService storage_domain.Service,
	config storage_domain.PresignConfig,
) *Handler {
	return &Handler{
		storageService: storageService,
		config:         config,
	}
}

// ServeHTTP handles presigned upload requests.
//
// It validates the token, checks rate limits, verifies content-type, enforces
// size limits, and streams the upload to the storage provider.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes r (*http.Request) which contains the upload request.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, l := logger_domain.From(r.Context(), log)
	tokenData, providerName, ok := h.validateUploadRequest(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, tokenData.MaxSize)

	bytesWritten, err := h.uploadFile(ctx, r.Body, tokenData, providerName, r.ContentLength)
	if err != nil {
		h.handleUploadError(ctx, w, err, tokenData.MaxSize)
		return
	}

	l.Info("Presigned upload completed",
		logger_domain.String("temp_key", tokenData.TempKey),
		logger_domain.Int64("size", bytesWritten),
		logger_domain.String("content_type", tokenData.ContentType))

	h.writeJSON(ctx, w, httpStatusOK, UploadResponse{
		TempKey:     tokenData.TempKey,
		Size:        bytesWritten,
		ContentType: tokenData.ContentType,
	})
}

// validateUploadRequest checks the upload request and parses the token data.
// Returns false for ok if checking fails and an error response was written.
//
// Takes w (http.ResponseWriter) which receives error responses on failure.
// Takes r (*http.Request) which is the upload request to check.
//
// Returns *storage_domain.PresignTokenData which contains the parsed token.
// Returns string which is the provider name from the query, or "default".
// Returns bool which is false when checking fails.
func (h *Handler) validateUploadRequest(w http.ResponseWriter, r *http.Request) (*storage_domain.PresignTokenData, string, bool) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		h.writeError(r.Context(), w, httpStatusBadRequest, "method_not_allowed", "Only PUT and POST methods are supported")
		return nil, "", false
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeError(r.Context(), w, httpStatusBadRequest, "missing_token", "Token query parameter is required")
		return nil, "", false
	}

	providerName := cmp.Or(r.URL.Query().Get("provider"), "default")

	tokenData, err := storage_domain.ParseAndVerifyPresignToken(h.config.Secret, token)
	if err != nil {
		h.handleTokenError(r.Context(), w, err)
		return nil, "", false
	}

	if !h.checkReplayProtection(r.Context(), w, tokenData) {
		return nil, "", false
	}

	if !h.validateRequestContentType(w, r, tokenData) {
		return nil, "", false
	}

	return tokenData, providerName, true
}

// checkReplayProtection verifies the token has not been used before.
//
// Takes w (http.ResponseWriter) which receives error responses on replay.
// Takes tokenData (*storage_domain.PresignTokenData) which contains the token
// to check.
//
// Returns bool which is true if the token is valid, or false if it was already
// used.
func (h *Handler) checkReplayProtection(ctx context.Context, w http.ResponseWriter, tokenData *storage_domain.PresignTokenData) bool {
	if h.config.RIDCache == nil {
		return true
	}
	expiresAt := time.Unix(tokenData.ExpiresAt, 0)
	if !h.config.RIDCache.Add(tokenData.RID, expiresAt) {
		ctx, l := logger_domain.From(ctx, log)
		l.Warn("Replay attempt detected",
			logger_domain.String("rid", tokenData.RID),
			logger_domain.String("temp_key", tokenData.TempKey))
		h.writeError(ctx, w, httpStatusForbidden, "token_replay", "Token has already been used")
		return false
	}
	return true
}

// validateRequestContentType checks if the request Content-Type header matches
// the expected value from the token.
//
// Takes w (http.ResponseWriter) which receives error responses when there is a
// mismatch.
// Takes r (*http.Request) which provides the Content-Type header to check.
// Takes tokenData (*storage_domain.PresignTokenData) which holds the expected
// content type.
//
// Returns bool which is true when the content type matches or when no content
// type is required.
func (h *Handler) validateRequestContentType(w http.ResponseWriter, r *http.Request, tokenData *storage_domain.PresignTokenData) bool {
	if tokenData.ContentType == "" {
		return true
	}
	requestCT := r.Header.Get(headerContentType)
	if !contentTypeMatches(requestCT, tokenData.ContentType) {
		h.writeError(r.Context(), w, httpStatusUnsupportedMediaType, "content_type_mismatch",
			fmt.Sprintf("Expected Content-Type %s, got %s", tokenData.ContentType, requestCT))
		return false
	}
	return true
}

// uploadFile streams the request body to the storage provider.
//
// Takes body (io.Reader) which provides the file content.
// Takes tokenData (*storage_domain.PresignTokenData) which specifies where
// to store the file.
// Takes providerName (string) which identifies the storage provider.
// Takes contentLength (int64) which is the expected size from the
// Content-Length header.
//
// Returns int64 which is the number of bytes written.
// Returns error when the upload fails.
func (h *Handler) uploadFile(ctx context.Context, body io.Reader, tokenData *storage_domain.PresignTokenData, providerName string, contentLength int64) (int64, error) {
	countingReader := &countingReader{reader: body}

	size := contentLength
	if size <= 0 {
		size = tokenData.MaxSize
	}

	params := &storage_dto.PutParams{
		Repository:  tokenData.Repository,
		Key:         tokenData.TempKey,
		Reader:      countingReader,
		Size:        size,
		ContentType: tokenData.ContentType,
	}

	if err := h.storageService.PutObject(ctx, providerName, params); err != nil {
		return countingReader.bytesRead, fmt.Errorf("storage write failed: %w", err)
	}

	return countingReader.bytesRead, nil
}

// handleTokenError writes an HTTP error response based on the token error type.
//
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the token error to handle.
func (h *Handler) handleTokenError(ctx context.Context, w http.ResponseWriter, err error) {
	handleTokenValidationError(ctx, w, err, h.writeError)
}

// handleUploadError converts upload errors to HTTP responses.
//
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the upload error to handle.
// Takes maxSize (int64) which is the largest allowed file size in bytes.
func (h *Handler) handleUploadError(ctx context.Context, w http.ResponseWriter, err error, maxSize int64) {
	errString := err.Error()

	if strings.Contains(errString, "http: request body too large") {
		h.writeError(ctx, w, httpStatusPayloadTooLarge, "file_too_large",
			fmt.Sprintf("File exceeds maximum size of %d bytes", maxSize))
		return
	}

	ctx, l := logger_domain.From(ctx, log)
	l.Error("Upload failed", logger_domain.Error(err))
	h.writeError(ctx, w, httpStatusInternalError, "upload_failed", "Failed to store uploaded file")
}

// writeJSON writes a JSON response with the given status code.
//
// Takes w (http.ResponseWriter) which receives the JSON response.
// Takes status (int) which sets the HTTP status code.
// Takes data (any) which is the value to encode as JSON.
func (*Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, data any) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)

	if err := json.ConfigDefault.NewEncoder(w).Encode(data); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to write JSON response", logger_domain.Error(err))
	}
}

// writeError writes a JSON error response.
//
// Takes w (http.ResponseWriter) which receives the JSON response.
// Takes status (int) which specifies the HTTP status code.
// Takes errorCode (string) which identifies the error type.
// Takes message (string) which provides a human-readable error description.
func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, status int, errorCode string, message string) {
	h.writeJSON(ctx, w, status, ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
}

// countingReader wraps an io.Reader and counts the bytes read.
type countingReader struct {
	// reader is the source from which bytes are read.
	reader io.Reader

	// bytesRead is the total number of bytes read so far.
	bytesRead int64
}

// Read reads up to len(p) bytes into p and tracks the total bytes read.
//
// Takes p ([]byte) which is the buffer to read into.
//
// Returns int which is the number of bytes read.
// Returns error when the underlying reader returns an error.
func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.bytesRead += int64(n)
	return n, err
}

// DownloadHandler provides HTTP handlers for presigned URL downloads.
type DownloadHandler struct {
	// storageService handles file operations for reading objects from storage.
	storageService storage_domain.Service

	// config holds presigned URL settings such as the secret and expiry time.
	config storage_domain.PresignConfig
}

// NewDownloadHandler creates a new presigned download handler.
//
// Takes storageService (storage_domain.Service) which handles file retrieval.
// Takes config (storage_domain.PresignConfig) which provides token verification
// settings.
//
// Returns *DownloadHandler which is ready to handle download requests.
func NewDownloadHandler(
	storageService storage_domain.Service,
	config storage_domain.PresignConfig,
) *DownloadHandler {
	return &DownloadHandler{
		storageService: storageService,
		config:         config,
	}
}

// ServeHTTP handles presigned download requests.
//
// It validates the download token, verifies it has not expired, and streams
// the file to the client with appropriate headers.
//
// Takes w (http.ResponseWriter) which receives the file content and headers.
// Takes r (*http.Request) which contains the download token and request data.
func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tokenData, providerName, ok := h.validateDownloadRequest(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	info, err := h.statFile(ctx, w, providerName, tokenData)
	if err != nil {
		return
	}

	if h.checkConditionalRequest(w, r, info) {
		return
	}

	h.setDownloadHeaders(w, tokenData, info)

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.streamFile(ctx, w, providerName, tokenData)
}

// validateDownloadRequest checks that a download request is valid and extracts
// token data.
//
// Takes w (http.ResponseWriter) which receives error responses on failure.
// Takes r (*http.Request) which provides the request to check.
//
// Returns *storage_domain.PresignDownloadTokenData which contains the parsed
// token data on success, or nil on failure.
// Returns string which is the provider name from the query, or "default".
// Returns bool which shows whether the check passed.
func (h *DownloadHandler) validateDownloadRequest(w http.ResponseWriter, r *http.Request) (*storage_domain.PresignDownloadTokenData, string, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeError(r.Context(), w, httpStatusBadRequest, "method_not_allowed", "Only GET and HEAD methods are supported")
		return nil, "", false
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		h.writeError(r.Context(), w, httpStatusBadRequest, "missing_token", "Token query parameter is required")
		return nil, "", false
	}

	providerName := cmp.Or(r.URL.Query().Get("provider"), "default")

	tokenData, err := storage_domain.ParseAndVerifyPresignDownloadToken(h.config.Secret, token)
	if err != nil {
		h.handleTokenError(r.Context(), w, err)
		return nil, "", false
	}

	return tokenData, providerName, true
}

// statFile gets file metadata for setting response headers.
//
// Takes w (http.ResponseWriter) which receives error responses on failure.
// Takes providerName (string) which identifies the storage provider to query.
// Takes tokenData (*storage_domain.PresignDownloadTokenData) which contains
// the repository and key for the file to look up.
//
// Returns *storage_domain.ObjectInfo which contains the file metadata.
// Returns error when the storage service cannot get the file information.
func (h *DownloadHandler) statFile(ctx context.Context, w http.ResponseWriter, providerName string, tokenData *storage_domain.PresignDownloadTokenData) (*storage_domain.ObjectInfo, error) {
	ctx, l := logger_domain.From(ctx, log)
	info, err := h.storageService.StatObject(ctx, providerName, storage_dto.GetParams{
		Repository: tokenData.Repository,
		Key:        tokenData.Key,
	})
	if err != nil {
		l.Error("Failed to stat file for download",
			logger_domain.String(logFieldKey, tokenData.Key),
			logger_domain.Error(err))
		h.writeError(ctx, w, httpStatusInternalError, "stat_failed", "Failed to retrieve file information")
		return nil, fmt.Errorf("statting file %q for download: %w", tokenData.Key, err)
	}
	return info, nil
}

// checkConditionalRequest checks If-None-Match and If-Modified-Since headers
// and returns 304 Not Modified if the client's cached version is still valid.
//
// Takes w (http.ResponseWriter) which receives the 304 response if applicable.
// Takes r (*http.Request) which provides the conditional request headers.
// Takes info (*storage_domain.ObjectInfo) which provides ETag and LastModified.
//
// Returns bool which is true if a 304 response was sent and the caller should
// return early.
func (*DownloadHandler) checkConditionalRequest(
	w http.ResponseWriter,
	r *http.Request,
	info *storage_domain.ObjectInfo,
) bool {
	if checkETagMatch(w, r, info) {
		return true
	}
	return checkIfModifiedSince(w, r, info)
}

// setDownloadHeaders sets the response headers for a file download.
//
// It sets caching headers (ETag, Last-Modified, Cache-Control) and
// Content-Disposition based on the object metadata and token preferences.
//
// Takes w (http.ResponseWriter) which receives the response headers.
// Takes tokenData (*storage_domain.PresignDownloadTokenData) which provides
// content type and filename preferences.
// Takes info (*storage_domain.ObjectInfo) which provides object metadata
// including ETag, LastModified, and custom metadata for cache control.
func (*DownloadHandler) setDownloadHeaders(
	w http.ResponseWriter,
	tokenData *storage_domain.PresignDownloadTokenData,
	info *storage_domain.ObjectInfo,
) {
	contentType := resolveContentType(tokenData.ContentType, info.ContentType)
	if contentType != "" {
		w.Header().Set(headerContentType, contentType)
	}

	setBasicHeaders(w, info)
	setCacheControl(w, info)
	setContentDisposition(w, tokenData.FileName, contentType, info.Metadata)
}

// streamFile fetches a file from storage and writes it to the response.
//
// Takes w (http.ResponseWriter) which receives the file data.
// Takes providerName (string) which names the storage provider to use.
// Takes tokenData (*storage_domain.PresignDownloadTokenData) which holds
// the repository and key for the file to fetch.
func (h *DownloadHandler) streamFile(ctx context.Context, w http.ResponseWriter, providerName string, tokenData *storage_domain.PresignDownloadTokenData) {
	ctx, l := logger_domain.From(ctx, log)
	reader, err := h.storageService.GetObject(ctx, providerName, storage_dto.GetParams{
		Repository: tokenData.Repository,
		Key:        tokenData.Key,
	})
	if err != nil {
		l.Error("Failed to get file for download",
			logger_domain.String(logFieldKey, tokenData.Key),
			logger_domain.Error(err))
		h.writeError(ctx, w, httpStatusInternalError, "download_failed", "Failed to retrieve file")
		return
	}
	defer func() { _ = reader.Close() }()

	bytesWritten, err := io.Copy(w, contextaware.NewReader(ctx, reader))
	if err != nil {
		l.Error("Failed to stream file",
			logger_domain.String(logFieldKey, tokenData.Key),
			logger_domain.Error(err))
		return
	}

	l.Debug("Presigned download completed",
		logger_domain.String(logFieldKey, tokenData.Key),
		logger_domain.Int64("size", bytesWritten))
}

// handleTokenError converts token validation errors to HTTP responses.
//
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the token validation error to handle.
func (h *DownloadHandler) handleTokenError(ctx context.Context, w http.ResponseWriter, err error) {
	handleTokenValidationError(ctx, w, err, h.writeError)
}

// writeError sends a JSON error response to the client.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes status (int) which is the HTTP status code to send.
// Takes errorCode (string) which names the type of error.
// Takes message (string) which explains what went wrong.
func (*DownloadHandler) writeError(ctx context.Context, w http.ResponseWriter, status int, errorCode string, message string) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{
		Error:   errorCode,
		Message: message,
	}

	if err := json.ConfigDefault.NewEncoder(w).Encode(response); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to write JSON error response", logger_domain.Error(err))
	}
}

// isInlineContentType checks whether a MIME type should display inline rather
// than as an attachment. Uses mime.ParseMediaType for parsing.
//
// Takes contentType (string) which is the MIME type to check.
//
// Returns bool which is true for images, videos, audio, text, fonts, and common
// browser-viewable types.
func isInlineContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType, _, _ = strings.Cut(contentType, ";")
		mediaType = strings.TrimSpace(strings.ToLower(mediaType))
	}

	parts := strings.SplitN(mediaType, "/", 2)
	if len(parts) != 2 {
		return false
	}
	topLevel := parts[0]

	if _, ok := inlineTopLevelTypes[topLevel]; ok {
		return true
	}

	if _, ok := inlineApplicationTypes[mediaType]; ok {
		return true
	}

	return false
}

// etagMatches checks if a client ETag matches a server ETag.
// Supports the "*" wildcard and comma-separated lists as per RFC 7232.
//
// Takes clientETag (string) which is the value from the If-None-Match header.
// Takes serverETag (string) which is the ETag of the current resource.
//
// Returns bool which is true if any client ETag matches the server ETag.
func etagMatches(clientETag, serverETag string) bool {
	if clientETag == "*" {
		return true
	}
	clientETag = strings.TrimPrefix(clientETag, "W/")
	serverETag = strings.TrimPrefix(serverETag, "W/")

	for etag := range strings.SplitSeq(clientETag, ",") {
		if strings.TrimSpace(etag) == serverETag {
			return true
		}
	}
	return false
}

// handleTokenValidationError maps token validation errors to HTTP
// responses using the provided write function.
//
// Takes ctx (context.Context) which carries the request context.
// Takes w (http.ResponseWriter) which receives the error response.
// Takes err (error) which is the token validation error.
// Takes writeFunction (errorWriteFunction) which writes the HTTP error response.
//
// Returns nothing.
func handleTokenValidationError(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
	writeFunction errorWriteFunction,
) {
	switch {
	case errors.Is(err, storage_domain.ErrPresignTokenExpired):
		writeFunction(ctx, w, httpStatusForbidden, "token_expired", "Token has expired")
	case errors.Is(err, storage_domain.ErrPresignTokenSignature):
		writeFunction(ctx, w, httpStatusUnauthorised, "invalid_signature", "Token signature is invalid")
	case errors.Is(err, storage_domain.ErrPresignTokenInvalid):
		writeFunction(ctx, w, httpStatusBadRequest, "invalid_token", "Token format is invalid")
	default:
		ctx, l := logger_domain.From(ctx, log)
		l.Error("Token validation failed", logger_domain.Error(err))
		writeFunction(ctx, w, httpStatusBadRequest, "token_error", "Token validation failed")
	}
}

// contentTypeMatches checks if a request content type matches an expected type.
// It compares only the media type part and ignores charset and other
// parameters.
//
// When expectedCT is empty, returns true (no restriction is set).
//
// When requestCT is empty but expectedCT is not, returns false.
//
// Takes requestCT (string) which is the content type from the request.
// Takes expectedCT (string) which is the expected content type to match.
//
// Returns bool which is true if the media types match.
func contentTypeMatches(requestCT, expectedCT string) bool {
	if expectedCT == "" {
		return true
	}
	if requestCT == "" {
		return false
	}

	requestMedia, _, _ := strings.Cut(requestCT, ";")
	expectedMedia, _, _ := strings.Cut(expectedCT, ";")

	return strings.TrimSpace(strings.ToLower(requestMedia)) == strings.TrimSpace(strings.ToLower(expectedMedia))
}

// checkETagMatch checks the If-None-Match header and returns true if a 304
// response was sent.
//
// Takes w (http.ResponseWriter) which receives the 304 status if ETags match.
// Takes r (*http.Request) which provides the If-None-Match header value.
// Takes info (*storage_domain.ObjectInfo) which contains the current ETag.
//
// Returns bool which is true if a 304 Not Modified response was sent.
func checkETagMatch(w http.ResponseWriter, r *http.Request, info *storage_domain.ObjectInfo) bool {
	clientETag := r.Header.Get(headerIfNoneMatch)
	if clientETag == "" || info.ETag == "" {
		return false
	}
	if etagMatches(clientETag, info.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// checkIfModifiedSince checks the If-Modified-Since header and returns true if
// a 304 Not Modified response was sent.
//
// Takes w (http.ResponseWriter) which receives the 304 status if applicable.
// Takes r (*http.Request) which provides the If-Modified-Since header.
// Takes info (*storage_domain.ObjectInfo) which contains the last modified
// time.
//
// Returns bool which is true when a 304 response was sent, false otherwise.
func checkIfModifiedSince(w http.ResponseWriter, r *http.Request, info *storage_domain.ObjectInfo) bool {
	if r.Header.Get(headerIfNoneMatch) != "" {
		return false
	}

	ims := r.Header.Get(headerIfModifiedSince)
	if ims == "" || info.LastModified.IsZero() {
		return false
	}

	t, err := http.ParseTime(ims)
	if err != nil {
		return false
	}

	if !info.LastModified.Truncate(time.Second).After(t.Truncate(time.Second)) {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// resolveContentType returns the content type to use, preferring the token's
// value over the object's metadata.
//
// Takes tokenCT (string) which is the content type from the token.
// Takes infoCT (string) which is the content type from object metadata.
//
// Returns string which is the resolved content type.
func resolveContentType(tokenCT, infoCT string) string {
	return cmp.Or(tokenCT, infoCT)
}

// setBasicHeaders sets Content-Length, ETag, and Last-Modified headers.
//
// Takes w (http.ResponseWriter) which receives the headers.
// Takes info (*storage_domain.ObjectInfo) which provides the object metadata.
func setBasicHeaders(w http.ResponseWriter, info *storage_domain.ObjectInfo) {
	if info.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))
	}

	if info.ETag != "" {
		w.Header().Set(headerETag, info.ETag)
	}

	if !info.LastModified.IsZero() {
		w.Header().Set(headerLastModified, info.LastModified.UTC().Format(http.TimeFormat))
	}
}

// setCacheControl sets the Cache-Control header from metadata or uses default.
//
// Takes w (http.ResponseWriter) which receives the Cache-Control header.
// Takes info (*storage_domain.ObjectInfo) which provides metadata for cache
// settings.
func setCacheControl(w http.ResponseWriter, info *storage_domain.ObjectInfo) {
	cacheControl := defaultCacheControl
	if info.Metadata != nil {
		if cc, ok := info.Metadata[storage_domain.MetadataKeyCacheControl]; ok && cc != "" {
			cacheControl = cc
		}
	}
	w.Header().Set(headerCacheControl, cacheControl)
}

// setContentDisposition sets the Content-Disposition header based on metadata,
// content type, and optional filename.
//
// Takes w (http.ResponseWriter) which receives the header.
// Takes fileName (string) which specifies the optional filename for the header.
// Takes contentType (string) which determines the disposition type.
// Takes metadata (map[string]string) which provides additional disposition
// hints.
func setContentDisposition(w http.ResponseWriter, fileName, contentType string, metadata map[string]string) {
	disposition := determineDisposition(contentType, metadata)

	if fileName != "" {
		w.Header().Set(headerContentDisposition, fmt.Sprintf("%s; filename=%q", disposition, fileName))
	} else {
		w.Header().Set(headerContentDisposition, disposition)
	}
}

// determineDisposition returns "inline" or "attachment" based on metadata
// preference or content type.
//
// Takes contentType (string) which specifies the MIME type of the content.
// Takes metadata (map[string]string) which may contain a disposition
// preference.
//
// Returns string which is either "inline" or "attachment".
func determineDisposition(contentType string, metadata map[string]string) string {
	if metadata != nil {
		if pref, ok := metadata[storage_domain.MetadataKeyContentDisposition]; ok {
			if pref == "inline" || pref == "attachment" {
				return pref
			}
		}
	} else if isInlineContentType(contentType) {
		return "inline"
	}
	return "attachment"
}
