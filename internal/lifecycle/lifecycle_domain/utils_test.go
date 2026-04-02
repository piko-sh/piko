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

package lifecycle_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hasPrefix(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		path     string
		prefix   string
		expected bool
	}{
		{
			name:     "path within directory",
			path:     "pages/home.pk",
			prefix:   "pages",
			expected: true,
		},
		{
			name:     "path within nested directory",
			path:     "pages/blog/article.pk",
			prefix:   "pages",
			expected: true,
		},
		{
			name:     "prefix with trailing slash",
			path:     "pages/home.pk",
			prefix:   "pages/",
			expected: true,
		},
		{
			name:     "path not in directory",
			path:     "components/button.pkc",
			prefix:   "pages",
			expected: false,
		},
		{
			name:     "empty prefix returns false",
			path:     "pages/home.pk",
			prefix:   "",
			expected: false,
		},
		{
			name:     "prefix longer than path",
			path:     "src",
			prefix:   "src/components",
			expected: false,
		},
		{
			name:     "partial match not at boundary",
			path:     "pagesExtra/home.pk",
			prefix:   "pages",
			expected: false,
		},
		{
			name:     "exact directory match",
			path:     "pages/index.pk",
			prefix:   "pages",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasPrefix(tc.path, tc.prefix)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_isRelevantFileForProcessing(t *testing.T) {
	t.Parallel()

	defaultPaths := &LifecyclePathsConfig{
		BaseDir:             "/project",
		PagesSourceDir:      "pages",
		PartialsSourceDir:   "partials",
		ComponentsSourceDir: "components",
		AssetsSourceDir:     "assets",
		I18nSourceDir:       "i18n",
	}

	testCases := []struct {
		paths    *LifecyclePathsConfig
		name     string
		relPath  string
		expected bool
	}{

		{
			name:     "go file is relevant anywhere",
			relPath:  "pkg/utils/helper.go",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "go file in root is relevant",
			relPath:  "main.go",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "test go file is not relevant",
			relPath:  "pkg/utils/helper_test.go",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "image in assets directory is relevant",
			relPath:  "assets/images/logo.png",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "font in assets directory is relevant",
			relPath:  "assets/fonts/roboto.woff2",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "css in assets directory is relevant",
			relPath:  "assets/css/global.css",
			paths:    defaultPaths,
			expected: true,
		},

		{
			name:     "pk in pages directory is relevant",
			relPath:  "pages/home.pk",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pk in partials directory is relevant",
			relPath:  "partials/header.pk",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pk outside pages/partials is not relevant",
			relPath:  "components/button.pk",
			paths:    defaultPaths,
			expected: false,
		},
		{
			name:     "pk in nested pages directory is relevant",
			relPath:  "pages/blog/article.pk",
			paths:    defaultPaths,
			expected: true,
		},

		{
			name:     "pkc in components directory is relevant",
			relPath:  "components/button.pkc",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pkc outside components is not relevant",
			relPath:  "pages/button.pkc",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "json in i18n directory is relevant",
			relPath:  "i18n/en.json",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "json outside i18n is not relevant",
			relPath:  "config/settings.json",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "markdown file is not relevant",
			relPath:  "README.md",
			paths:    defaultPaths,
			expected: false,
		},
		{
			name:     "hidden file is not relevant",
			relPath:  ".gitignore",
			paths:    defaultPaths,
			expected: false,
		},
		{
			name:     "txt file is not relevant",
			relPath:  "notes.txt",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:    "pk with empty pages directory config",
			relPath: "pages/home.pk",
			paths: &LifecyclePathsConfig{
				BaseDir:           "/project",
				PagesSourceDir:    "",
				PartialsSourceDir: "partials",
			},
			expected: false,
		},
		{
			name:    "asset with empty assets directory config",
			relPath: "assets/logo.png",
			paths: &LifecyclePathsConfig{
				BaseDir:         "/project",
				AssetsSourceDir: "",
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isRelevantFileForProcessing(tc.relPath, tc.paths)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func Test_isCoreSourceFile(t *testing.T) {
	t.Parallel()

	defaultPaths := &LifecyclePathsConfig{
		BaseDir:             "/project",
		PagesSourceDir:      "pages",
		PartialsSourceDir:   "partials",
		ComponentsSourceDir: "components",
		I18nSourceDir:       "i18n",
	}

	testCases := []struct {
		paths    *LifecyclePathsConfig
		name     string
		relPath  string
		expected bool
	}{

		{
			name:     "go file is core source",
			relPath:  "pkg/service.go",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "test go file is still core source",
			relPath:  "pkg/service_test.go",
			paths:    defaultPaths,
			expected: true,
		},

		{
			name:     "pk in pages is core source",
			relPath:  "pages/home.pk",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pk in partials is core source",
			relPath:  "partials/footer.pk",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pk elsewhere is not core source",
			relPath:  "templates/email.pk",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "pkc in components is core source",
			relPath:  "components/modal.pkc",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "pkc elsewhere is not core source",
			relPath:  "lib/widget.pkc",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "json in i18n is core source",
			relPath:  "i18n/fr.json",
			paths:    defaultPaths,
			expected: true,
		},
		{
			name:     "json elsewhere is not core source",
			relPath:  "config/db.json",
			paths:    defaultPaths,
			expected: false,
		},

		{
			name:     "image file is not core source",
			relPath:  "assets/logo.png",
			paths:    defaultPaths,
			expected: false,
		},
		{
			name:     "css file is not core source",
			relPath:  "assets/style.css",
			paths:    defaultPaths,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isCoreSourceFile(tc.relPath, tc.paths)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsValidPKFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "valid pk file",
			filename: "home.pk",
			expected: true,
		},
		{
			name:     "uppercase extension",
			filename: "home.PK",
			expected: true,
		},
		{
			name:     "mixed case extension",
			filename: "home.Pk",
			expected: true,
		},
		{
			name:     "underscore prefix is invalid",
			filename: "_layout.pk",
			expected: false,
		},
		{
			name:     "non-pk extension",
			filename: "home.html",
			expected: false,
		},
		{
			name:     "pk in name but wrong extension",
			filename: "pk.html",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isValidPKFile(tc.filename)
			assert.Equal(t, tc.expected, result)
		})
	}
}
