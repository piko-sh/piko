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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInlineContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{name: "empty", contentType: "", want: false},
		{name: "image/png", contentType: "image/png", want: true},
		{name: "image/jpeg", contentType: "image/jpeg", want: true},
		{name: "video/mp4", contentType: "video/mp4", want: true},
		{name: "audio/mpeg", contentType: "audio/mpeg", want: true},
		{name: "text/html", contentType: "text/html", want: true},
		{name: "text/plain", contentType: "text/plain", want: true},
		{name: "font/woff2", contentType: "font/woff2", want: true},
		{name: "application/pdf", contentType: "application/pdf", want: true},
		{name: "application/json", contentType: "application/json", want: true},
		{name: "application/javascript", contentType: "application/javascript", want: true},
		{name: "application/svg+xml", contentType: "application/svg+xml", want: true},
		{name: "application/wasm", contentType: "application/wasm", want: true},
		{name: "application/octet-stream", contentType: "application/octet-stream", want: false},
		{name: "application/zip", contentType: "application/zip", want: false},
		{name: "with charset param", contentType: "text/html; charset=utf-8", want: true},
		{name: "invalid mime", contentType: "notamimetype", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isInlineContentType(tt.contentType))
		})
	}
}

func TestEtagMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		clientETag string
		serverETag string
		want       bool
	}{
		{name: "wildcard", clientETag: "*", serverETag: `"abc"`, want: true},
		{name: "exact match", clientETag: `"abc"`, serverETag: `"abc"`, want: true},
		{name: "no match", clientETag: `"abc"`, serverETag: `"definition"`, want: false},
		{name: "weak client match", clientETag: `W/"abc"`, serverETag: `"abc"`, want: true},
		{name: "weak server match", clientETag: `"abc"`, serverETag: `W/"abc"`, want: true},
		{name: "both weak match", clientETag: `W/"abc"`, serverETag: `W/"abc"`, want: true},
		{name: "comma separated match", clientETag: `"xxx", "abc", "yyy"`, serverETag: `"abc"`, want: true},
		{name: "comma separated no match", clientETag: `"xxx", "yyy"`, serverETag: `"abc"`, want: false},
		{name: "empty client", clientETag: "", serverETag: `"abc"`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, etagMatches(tt.clientETag, tt.serverETag))
		})
	}
}

func TestContentTypeMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		requestCT  string
		expectedCT string
		want       bool
	}{
		{name: "empty expected allows all", requestCT: "text/html", expectedCT: "", want: true},
		{name: "empty request fails", requestCT: "", expectedCT: "text/html", want: false},
		{name: "exact match", requestCT: "image/png", expectedCT: "image/png", want: true},
		{name: "no match", requestCT: "image/png", expectedCT: "image/jpeg", want: false},
		{name: "ignores charset", requestCT: "text/html; charset=utf-8", expectedCT: "text/html", want: true},
		{name: "case insensitive", requestCT: "IMAGE/PNG", expectedCT: "image/png", want: true},
		{name: "both have params", requestCT: "text/html; charset=utf-8", expectedCT: "text/html; charset=ascii", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, contentTypeMatches(tt.requestCT, tt.expectedCT))
		})
	}
}

func TestCountingReader_Read(t *testing.T) {
	t.Parallel()

	data := "hello world"
	cr := &countingReader{reader: strings.NewReader(data)}

	buffer := make([]byte, 5)
	n, err := cr.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(5), cr.bytesRead)

	n, err = cr.Read(buffer)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, int64(10), cr.bytesRead)
}

func TestParsePath(t *testing.T) {
	t.Parallel()

	h := &PublicDownloadHandler{}

	tests := []struct {
		name     string
		path     string
		wantProv string
		wantRepo string
		wantKey  string
		wantOK   bool
	}{
		{
			name:     "valid path",
			path:     "/_piko/storage/public/s3/images/photo.jpg",
			wantProv: "s3", wantRepo: "images", wantKey: "photo.jpg", wantOK: true,
		},
		{
			name:     "nested key",
			path:     "/_piko/storage/public/s3/uploads/2024/01/file.txt",
			wantProv: "s3", wantRepo: "uploads", wantKey: "2024/01/file.txt", wantOK: true,
		},
		{name: "too few parts", path: "/_piko/storage/public/s3", wantOK: false},
		{name: "wrong prefix", path: "/api/storage/public/s3/repo/key", wantOK: false},
		{name: "empty path", path: "", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			provider, repo, key, ok := h.parsePath(tt.path)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.wantProv, provider)
				assert.Equal(t, tt.wantRepo, repo)
				assert.Equal(t, tt.wantKey, key)
			}
		})
	}
}
