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

package lifecycle_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestExtractPackageName_SimplePackage(t *testing.T) {
	code := "package main\n\nfunc main() {}\n"
	name, err := extractPackageName(code)
	require.NoError(t, err)
	assert.Equal(t, "main", name)
}

func TestExtractPackageName_WithComment(t *testing.T) {
	code := "package mypackage // this is a comment\n\nimport \"fmt\"\n"
	name, err := extractPackageName(code)
	require.NoError(t, err)
	assert.Equal(t, "mypackage", name)
}

func TestExtractPackageName_WithLeadingWhitespace(t *testing.T) {
	code := "  package   mypackage  \n"
	name, err := extractPackageName(code)
	require.NoError(t, err)
	assert.Equal(t, "mypackage", name)
}

func TestExtractPackageName_WithCopyrightHeader(t *testing.T) {
	code := "// Copyright 2024\n// License header\n\npackage domain\n"
	name, err := extractPackageName(code)
	require.NoError(t, err)
	assert.Equal(t, "domain", name)
}

func TestExtractPackageName_NoPackageDeclaration(t *testing.T) {
	code := "// just a comment\nfunc main() {}\n"
	_, err := extractPackageName(code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing package declaration")
}

func TestExtractPackageName_EmptyString(t *testing.T) {
	_, err := extractPackageName("")
	assert.Error(t, err)
}

func TestExtractPackageName_PackageInCommentOnly(t *testing.T) {

	code := "// package main\nfunc init() {}\n"
	_, err := extractPackageName(code)
	assert.Error(t, err)
}

func TestCreatePageEntryFromManifest_FoundInPages(t *testing.T) {

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"src/pages/home.go": {
				PackagePath:        "example.com/pages/home",
				OriginalSourcePath: "src/pages/home.go",
				RoutePatterns:      map[string]string{"/": "/"},
			},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	entry := createPageEntryFromManifest(manifest, "src/pages/home.go")
	require.NotNil(t, entry)
	assert.Equal(t, "example.com/pages/home", entry.PackagePath)
	assert.Equal(t, "src/pages/home.go", entry.OriginalSourcePath)
	assert.Equal(t, map[string]string{"/": "/"}, entry.RoutePatterns)
}

func TestCreatePageEntryFromManifest_FoundInPartials(t *testing.T) {

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{
			"src/partials/header.go": {
				PackagePath:        "example.com/partials/header",
				OriginalSourcePath: "src/partials/header.go",
				PartialSrc:         "/_piko/partial/partials-header",
				StyleBlock:         "h1 { color: red }",
			},
		},
		Emails: map[string]generator_dto.ManifestEmailEntry{},
	}

	entry := createPageEntryFromManifest(manifest, "src/partials/header.go")
	require.NotNil(t, entry)
	assert.Equal(t, "example.com/partials/header", entry.PackagePath)
	assert.Equal(t, "src/partials/header.go", entry.OriginalSourcePath)
	assert.Equal(t, "h1 { color: red }", entry.StyleBlock)

	assert.Equal(t, map[string]string{"": "/_piko/partial/partials-header"}, entry.RoutePatterns)
}

func TestCreatePageEntryFromManifest_FoundInEmails(t *testing.T) {

	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails: map[string]generator_dto.ManifestEmailEntry{
			"src/emails/welcome.go": {
				PackagePath:        "example.com/emails/welcome",
				OriginalSourcePath: "src/emails/welcome.go",
				StyleBlock:         "body { font-family: sans-serif }",
			},
		},
	}

	entry := createPageEntryFromManifest(manifest, "src/emails/welcome.go")
	require.NotNil(t, entry)
	assert.Equal(t, "example.com/emails/welcome", entry.PackagePath)
	assert.Equal(t, "src/emails/welcome.go", entry.OriginalSourcePath)
	assert.Equal(t, "body { font-family: sans-serif }", entry.StyleBlock)
}

func TestCreatePageEntryFromManifest_NotFound(t *testing.T) {

	manifest := &generator_dto.Manifest{
		Pages:    map[string]generator_dto.ManifestPageEntry{},
		Partials: map[string]generator_dto.ManifestPartialEntry{},
		Emails:   map[string]generator_dto.ManifestEmailEntry{},
	}

	entry := createPageEntryFromManifest(manifest, "src/nonexistent.go")
	assert.Nil(t, entry)
}

func TestCreatePageEntryFromManifest_PagesHasPriority(t *testing.T) {

	manifest := &generator_dto.Manifest{
		Pages: map[string]generator_dto.ManifestPageEntry{
			"src/comp.go": {PackagePath: "page-pkg"},
		},
		Partials: map[string]generator_dto.ManifestPartialEntry{
			"src/comp.go": {PackagePath: "partial-pkg"},
		},
		Emails: map[string]generator_dto.ManifestEmailEntry{},
	}

	entry := createPageEntryFromManifest(manifest, "src/comp.go")
	require.NotNil(t, entry)
	assert.Equal(t, "page-pkg", entry.PackagePath)
}
