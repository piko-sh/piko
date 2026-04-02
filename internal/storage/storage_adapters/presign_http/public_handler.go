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
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

const (
	// publicPathMinParts is the minimum number of path parts needed for a valid
	// public download URL (/_piko/storage/public/{provider}/{repository}/{key}).
	publicPathMinParts = 6

	// publicPathProviderIndex is the index of the provider in the path parts.
	publicPathProviderIndex = 3

	// publicPathRepositoryIndex is the position of the repository in URL path parts.
	publicPathRepositoryIndex = 4

	// publicPathKeyStartIndex is the start index for the key in the path parts.
	publicPathKeyStartIndex = 5
)

// PublicDownloadHandler serves files from public repositories without
// authentication. It implements http.Handler.
type PublicDownloadHandler struct {
	// storageService checks repository access and reads files from storage.
	storageService storage_domain.Service
}

// NewPublicDownloadHandler creates a new public download handler.
//
// Takes storageService (storage_domain.Service) which handles file retrieval.
//
// Returns *PublicDownloadHandler which is ready to serve public files.
func NewPublicDownloadHandler(
	storageService storage_domain.Service,
) *PublicDownloadHandler {
	return &PublicDownloadHandler{
		storageService: storageService,
	}
}

// ServeHTTP handles public file downloads without authentication.
// URL format: /_piko/storage/public/{provider}/{repository}/{key}.
//
// Takes w (http.ResponseWriter) which receives the file content and headers.
// Takes r (*http.Request) which contains the path and request details.
func (h *PublicDownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, l := logger_domain.From(r.Context(), log)
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET and HEAD methods are supported")
		return
	}

	provider, repository, key, ok := h.parsePath(r.URL.Path)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "invalid_path", "Path must be /_piko/storage/public/{provider}/{repository}/{key}")
		return
	}

	if !h.storageService.IsPublicRepository(repository) {
		l.Warn("Attempted access to non-public repository",
			logger_domain.String("repository", repository))
		h.writeError(w, http.StatusForbidden, "not_public", "This repository does not allow public access")
		return
	}

	info, err := h.statFile(ctx, provider, repository, key)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "not_found", "File not found")
		return
	}

	if h.checkConditionalRequest(w, r, info) {
		return
	}

	h.setPublicHeaders(w, repository, info)

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.streamFile(ctx, w, provider, repository, key)
}

// statFile retrieves file metadata.
//
// Takes provider (string) which identifies the storage provider.
// Takes repository (string) which identifies the repository.
// Takes key (string) which identifies the file.
//
// Returns *storage_domain.ObjectInfo which contains file metadata.
// Returns error when the file cannot be found.
func (h *PublicDownloadHandler) statFile(ctx context.Context, provider, repository, key string) (*storage_domain.ObjectInfo, error) {
	return h.storageService.StatObject(ctx, provider, storage_dto.GetParams{
		Repository: repository,
		Key:        key,
	})
}

// streamFile sends the content of a file to the client.
//
// Takes w (http.ResponseWriter) which receives the file data.
// Takes provider (string) which is the name of the storage provider.
// Takes repository (string) which is the name of the repository.
// Takes key (string) which is the name of the file.
func (h *PublicDownloadHandler) streamFile(ctx context.Context, w http.ResponseWriter, provider, repository, key string) {
	ctx, l := logger_domain.From(ctx, log)
	reader, err := h.storageService.GetObject(ctx, provider, storage_dto.GetParams{
		Repository: repository,
		Key:        key,
	})
	if err != nil {
		l.Error("Failed to retrieve file", logger_domain.Error(err))
		h.writeError(w, http.StatusInternalServerError, "retrieval_failed", "Failed to retrieve file")
		return
	}
	defer func() { _ = reader.Close() }()

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

// setPublicHeaders sets response headers with aggressive caching for public files.
//
// Takes w (http.ResponseWriter) which receives the headers.
// Takes repository (string) which identifies the repository for cache config.
// Takes info (*storage_domain.ObjectInfo) which provides file metadata.
func (h *PublicDownloadHandler) setPublicHeaders(w http.ResponseWriter, repository string, info *storage_domain.ObjectInfo) { //nolint:revive // needs receiver for storageService
	w.Header().Set(headerContentType, info.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size))

	if config, ok := h.storageService.GetRepositoryConfig(repository); ok && config.CacheControl != "" {
		w.Header().Set(headerCacheControl, config.CacheControl)
	} else {
		w.Header().Set(headerCacheControl, "public, max-age=31536000, immutable")
	}

	if info.Metadata != nil {
		if cd, ok := info.Metadata["Content-Disposition"]; ok {
			w.Header().Set(headerContentDisposition, cd)
		} else {
			h.setContentDisposition(w, info.ContentType)
		}
	} else {
		h.setContentDisposition(w, info.ContentType)
	}

	if info.ETag != "" {
		w.Header().Set(headerETag, info.ETag)
	}
	if !info.LastModified.IsZero() {
		w.Header().Set(headerLastModified, info.LastModified.UTC().Format(http.TimeFormat))
	}
}

// setContentDisposition sets the Content-Disposition header based on the
// content type.
//
// Takes w (http.ResponseWriter) which receives the header.
// Takes contentType (string) which determines if the response is shown inline
// or as a download attachment.
func (*PublicDownloadHandler) setContentDisposition(w http.ResponseWriter, contentType string) {
	if isInlineContentType(contentType) {
		w.Header().Set(headerContentDisposition, "inline")
	} else {
		w.Header().Set(headerContentDisposition, "attachment")
	}
}

// checkConditionalRequest checks If-None-Match and If-Modified-Since headers.
//
// Takes w (http.ResponseWriter) which receives the 304 response if matched.
// Takes r (*http.Request) which provides the conditional headers.
// Takes info (*storage_domain.ObjectInfo) which contains the file metadata.
//
// Returns bool which is true if 304 was sent, false otherwise.
func (*PublicDownloadHandler) checkConditionalRequest(w http.ResponseWriter, r *http.Request, info *storage_domain.ObjectInfo) bool {
	if inm := r.Header.Get(headerIfNoneMatch); inm != "" && info.ETag != "" {
		if etagMatches(inm, info.ETag) {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}

	if ims := r.Header.Get(headerIfModifiedSince); ims != "" && !info.LastModified.IsZero() {
		if t, err := time.Parse(http.TimeFormat, ims); err == nil {
			if !info.LastModified.After(t) {
				w.WriteHeader(http.StatusNotModified)
				return true
			}
		}
	}

	return false
}

// parsePath extracts provider, repository, and key from a URL path.
// The expected format is /_piko/storage/public/{provider}/{repository}/{key...}.
//
// Takes path (string) which is the URL path to parse.
//
// Returns provider (string) which identifies the storage provider.
// Returns repository (string) which identifies the repository.
// Returns key (string) which identifies the file within the repository.
// Returns ok (bool) which is true if parsing succeeded.
func (*PublicDownloadHandler) parsePath(path string) (provider, repository, key string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < publicPathMinParts {
		return "", "", "", false
	}

	if parts[0] != "_piko" || parts[1] != "storage" || parts[2] != "public" {
		return "", "", "", false
	}

	provider = parts[publicPathProviderIndex]
	repository = parts[publicPathRepositoryIndex]
	key = strings.Join(parts[publicPathKeyStartIndex:], "/")

	return provider, repository, key, true
}

// writeError sends a JSON error response to the client.
//
// Takes w (http.ResponseWriter) which receives the error response.
// Takes status (int) which is the HTTP status code to send.
// Takes code (string) which is the error code for the response.
// Takes message (string) which describes the error.
func (*PublicDownloadHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set(headerContentType, "application/json")
	w.WriteHeader(status)

	errorResp := map[string]string{
		"error":   code,
		"message": message,
	}

	if jsonBytes, err := json.Marshal(errorResp); err == nil {
		_, _ = w.Write(jsonBytes)
	} else {
		_, _ = fmt.Fprintf(w, `{"error":"%s","message":"%s"}`, code, message)
	}
}
