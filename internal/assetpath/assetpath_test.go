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

package assetpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsTransform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src  string
		want bool
	}{
		{"", false},
		{"github.com/user/app/assets/hero.png", true},
		{"assets/logo.svg", true},
		{"@/assets/icon.png", true},
		{"relative/path.png", true},

		{"/static/image.png", false},
		{"/already/absolute.ico", false},

		{"http://example.com/image.png", false},
		{"https://example.com/image.png", false},

		{"//cdn.example.com/image.png", false},

		{"data:image/png;base64,ABC123", false},
		{"data:image/svg+xml,%3Csvg%3E%3C/svg%3E", false},

		{"/_piko/assets/github.com/example/assets/image.png", false},
		{"/_piko/assets/already/transformed", false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, NeedsTransform(tt.src, DefaultServePath), "src=%q", tt.src)
	}
}

func TestNeedsCleaning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src  string
		want bool
	}{
		{"clean/path.png", false},
		{"github.com/user/app/assets/hero.png", false},
		{"simple/path.png", false},

		{"./relative/path.png", true},
		{"../parent/path.png", true},
		{"path//double/slash.png", true},
		{"path/with./dot", true},
		{"path/../parent", true},
		{"path/trailing/", true},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, NeedsCleaning(tt.src), "src=%q", tt.src)
	}
}

func TestResolveModuleAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		src        string
		moduleName string
		want       string
	}{
		{"assets/icon.png", "mymod", "assets/icon.png"},
		{"@/assets/icon.png", "mymod", "mymod/assets/icon.png"},
		{"@/assets/icon.png", "", "@/assets/icon.png"},
		{"@/deep/path/icon.svg", "github.com/org/repo", "github.com/org/repo/deep/path/icon.svg"},
		{"no-alias.png", "mymod", "no-alias.png"},
	}

	for _, tt := range tests {
		got := ResolveModuleAlias(tt.src, tt.moduleName)
		assert.Equal(t, tt.want, got, "src=%q moduleName=%q", tt.src, tt.moduleName)
	}
}

func TestTransform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		src        string
		moduleName string
		want       string
	}{
		{
			name: "module path",
			src:  "github.com/user/app/assets/hero.png",
			want: "/_piko/assets/github.com/user/app/assets/hero.png",
		},
		{
			name: "relative path",
			src:  "assets/logo.svg",
			want: "/_piko/assets/assets/logo.svg",
		},
		{
			name: "already absolute",
			src:  "/static/image.png",
			want: "/static/image.png",
		},
		{
			name: "http URL",
			src:  "http://example.com/image.png",
			want: "http://example.com/image.png",
		},
		{
			name: "https URL",
			src:  "https://example.com/image.png",
			want: "https://example.com/image.png",
		},
		{
			name: "data URI",
			src:  "data:image/png;base64,ABC123",
			want: "data:image/png;base64,ABC123",
		},
		{
			name: "empty string",
			src:  "",
			want: "",
		},
		{
			name: "path needing cleaning",
			src:  "assets/../images/hero.png",
			want: "/_piko/assets/images/hero.png",
		},
		{
			name: "protocol-relative URL",
			src:  "//cdn.example.com/image.png",
			want: "//cdn.example.com/image.png",
		},
		{
			name: "already transformed",
			src:  "/_piko/assets/github.com/example/image.png",
			want: "/_piko/assets/github.com/example/image.png",
		},
		{
			name:       "resolves @/ alias with module name",
			src:        "@/lib/images/hero.png",
			moduleName: "testmodule",
			want:       "/_piko/assets/testmodule/lib/images/hero.png",
		},
		{
			name:       "keeps @/ when no module name",
			src:        "@/lib/images/hero.png",
			moduleName: "",
			want:       "/_piko/assets/@/lib/images/hero.png",
		},
		{
			name: "trailing slash cleaned",
			src:  "github.com/example/assets/",
			want: "/_piko/assets/github.com/example/assets",
		},
		{
			name: "double slashes cleaned",
			src:  "github.com/example//assets/image.png",
			want: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name: "dot segments cleaned",
			src:  "github.com/example/./assets/../assets/image.png",
			want: "/_piko/assets/github.com/example/assets/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Transform(tt.src, tt.moduleName, DefaultServePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendTransformed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "transforms module path",
			src:  "github.com/example/image.png",
			want: "/_piko/assets/github.com/example/image.png",
		},
		{
			name: "nested directory structure",
			src:  "github.com/org/repo/assets/images/hero.jpg",
			want: "/_piko/assets/github.com/org/repo/assets/images/hero.jpg",
		},
		{
			name: "path with special chars in filename",
			src:  "github.com/example/assets/my-image_v2.png",
			want: "/_piko/assets/github.com/example/assets/my-image_v2.png",
		},
		{
			name: "skips already transformed path",
			src:  "/_piko/assets/github.com/example/assets/image.png",
			want: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name: "skips HTTP URL",
			src:  "http://cdn.example.com/image.png",
			want: "http://cdn.example.com/image.png",
		},
		{
			name: "skips HTTPS URL",
			src:  "https://cdn.example.com/image.png",
			want: "https://cdn.example.com/image.png",
		},
		{
			name: "skips protocol-relative URL",
			src:  "//cdn.example.com/image.png",
			want: "//cdn.example.com/image.png",
		},
		{
			name: "skips data URI with base64",
			src:  "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg...",
			want: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg...",
		},
		{
			name: "skips data URI with SVG",
			src:  "data:image/svg+xml,%3Csvg%3E%3C/svg%3E",
			want: "data:image/svg+xml,%3Csvg%3E%3C/svg%3E",
		},
		{
			name: "trailing slash removed",
			src:  "github.com/example/assets/",
			want: "/_piko/assets/github.com/example/assets",
		},
		{
			name: "double slashes cleaned",
			src:  "github.com/example//assets/image.png",
			want: "/_piko/assets/github.com/example/assets/image.png",
		},
		{
			name: "dot segments cleaned",
			src:  "github.com/example/./assets/../assets/image.png",
			want: "/_piko/assets/github.com/example/assets/image.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(AppendTransformed(nil, tt.src, DefaultServePath))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendTransformed_AppendsToExistingBuffer(t *testing.T) {
	t.Parallel()

	buffer := []byte("prefix:")
	buffer = AppendTransformed(buffer, "github.com/example/image.png", DefaultServePath)
	assert.Equal(t, "prefix:/_piko/assets/github.com/example/image.png", string(buffer))
}

func TestTransform_CustomServePath(t *testing.T) {
	t.Parallel()

	got := Transform("assets/icon.png", "", "/custom/path")
	assert.Equal(t, "/custom/path/assets/icon.png", got)
}

func TestAppendTransformed_CustomServePath(t *testing.T) {
	t.Parallel()

	got := string(AppendTransformed(nil, "assets/icon.png", "/custom/path"))
	assert.Equal(t, "/custom/path/assets/icon.png", got)
}
