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

//go:build !js

package daemon_frontend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMimeType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileExt  string
		expected string
	}{
		{name: "javascript", fileExt: ".js", expected: "application/javascript; charset=utf-8"},
		{name: "css", fileExt: ".css", expected: "text/css; charset=utf-8"},
		{name: "html", fileExt: ".html", expected: "text/html; charset=utf-8"},
		{name: "svg", fileExt: ".svg", expected: "image/svg+xml"},
		{name: "png", fileExt: ".png", expected: "image/png"},
		{name: "jpg", fileExt: ".jpg", expected: "image/jpeg"},
		{name: "jpeg", fileExt: ".jpeg", expected: "image/jpeg"},
		{name: "webp", fileExt: ".webp", expected: "image/webp"},
		{name: "unknown extension", fileExt: ".xyz", expected: "application/octet-stream"},
		{name: "uppercase js", fileExt: ".JS", expected: "application/javascript; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getMimeType(tt.fileExt))
		})
	}
}

func TestGetEncodingFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "brotli", path: "file.js.br", expected: "br"},
		{name: "gzip", path: "file.js.gz", expected: "gzip"},
		{name: "uncompressed", path: "file.js", expected: ""},
		{name: "no extension", path: "file", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getEncodingFromPath(tt.path))
		})
	}
}
