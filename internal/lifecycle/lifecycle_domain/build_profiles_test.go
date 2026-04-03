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
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/registry/registry_dto"
)

func Test_makeProfile(t *testing.T) {
	t.Parallel()

	t.Run("creates profile with all parameters", func(t *testing.T) {
		t.Parallel()

		profile := makeProfile(
			"test-profile",
			registry_dto.PriorityNeed,
			"test-capability",
			"source",
			map[string]string{"type": "test", "mimeType": "text/plain"},
			map[string]string{"param1": "value1"},
		)

		assert.Equal(t, "test-profile", profile.Name)
		assert.Equal(t, registry_dto.PriorityNeed, profile.Profile.Priority)
		assert.Equal(t, "test-capability", profile.Profile.CapabilityName)
		tagType, ok := profile.Profile.ResultingTags.GetByName("type")
		assert.True(t, ok)
		assert.Equal(t, "test", tagType)
		param1, ok := profile.Profile.Params.GetByName("param1")
		assert.True(t, ok)
		assert.Equal(t, "value1", param1)
	})

	t.Run("handles empty dependsOn", func(t *testing.T) {
		t.Parallel()

		profile := makeProfile(
			"independent",
			registry_dto.PriorityWant,
			"capability",
			"",
			nil,
			nil,
		)

		assert.Equal(t, "independent", profile.Name)
		assert.Empty(t, profile.Profile.DependsOn)
	})

	t.Run("handles nil maps", func(t *testing.T) {
		t.Parallel()

		profile := makeProfile(
			"minimal",
			registry_dto.PriorityWant,
			"capability",
			"source",
			nil,
			nil,
		)

		assert.Equal(t, "minimal", profile.Name)
	})
}
func TestGetProfilesForFile(t *testing.T) {
	t.Parallel()

	t.Run("CSS file gets minify and compress profiles", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("styles/main.css", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 3)

		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}

		assert.Contains(t, profileNames, "minified")
		assert.Contains(t, profileNames, "gzip")
		assert.Contains(t, profileNames, "br")
		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, capabilities_dto.CapabilityMinifyCSS.String(), p.Profile.CapabilityName)
			}
		}
	})

	t.Run("JavaScript file gets minify and compress profiles", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("scripts/app.js", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 3)
		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, capabilities_dto.CapabilityMinifyJS.String(), p.Profile.CapabilityName)
			}
		}
	})

	t.Run("PK JS file gets high priority profiles", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("pk-js/page-home.js", nil)

		require.NotEmpty(t, profiles)
		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, registry_dto.PriorityNeed, p.Profile.Priority)
			}
		}
	})

	t.Run("SVG file gets minify and compress profiles", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("icons/logo.svg", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 3)
		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, capabilities_dto.CapabilityMinifySVG.String(), p.Profile.CapabilityName)
			}
		}
	})

	t.Run("PKC component file gets compiled JS profile", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("components/button.pkc", nil)

		require.NotEmpty(t, profiles)
		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "compiled_js")
		for _, p := range profiles {
			if p.Name == "compiled_js" {
				assert.Equal(t, capabilities_dto.CapabilityCompileComponent.String(), p.Profile.CapabilityName)
				tagName, ok := p.Profile.ResultingTags.GetByName("tagName")
				assert.True(t, ok)
				assert.Equal(t, "button", tagName)
			}
		}
	})

	t.Run("ICO file gets compress profiles only", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("favicon.ico", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 2)

		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}
		assert.Contains(t, profileNames, "gzip")
		assert.Contains(t, profileNames, "br")
		assert.NotContains(t, profileNames, "minified")
	})

	t.Run("PNG file gets compress profiles only", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("images/logo.png", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 2)
	})

	t.Run("webmanifest file gets compress profiles", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("site.webmanifest", nil)

		require.NotEmpty(t, profiles)
		assert.Len(t, profiles, 2)
	})

	t.Run("unknown extension returns nil", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("readme.md", nil)
		assert.Nil(t, profiles)
	})

	t.Run("ignored extension returns nil", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("styles/main.css", []string{".css"})
		assert.Nil(t, profiles)
	})

	t.Run("case insensitive extension matching", func(t *testing.T) {
		t.Parallel()

		profiles := GetProfilesForFile("styles/main.CSS", nil)
		require.NotEmpty(t, profiles)

		profiles2 := GetProfilesForFile("icons/logo.SVG", nil)
		require.NotEmpty(t, profiles2)
	})
}
func TestGetMimeTypeForExtension(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext      string
		expected string
	}{
		{ext: ".png", expected: "image/png"},
		{ext: ".webmanifest", expected: "application/manifest+json"},
		{ext: ".ico", expected: "image/x-icon"},
		{ext: ".unknown", expected: "image/x-icon"},
	}

	for _, tc := range testCases {
		t.Run(tc.ext, func(t *testing.T) {
			t.Parallel()
			result := getMimeTypeForExtension(tc.ext)
			assert.Equal(t, tc.expected, result)
		})
	}
}
func TestBuildMinifyCompressChain(t *testing.T) {
	t.Parallel()

	profiles := buildMinifyCompressChain(
		capabilities_dto.CapabilityMinifyCSS.String(),
		"source",
		"minified-css",
		"text/css",
		".min.css",
	)

	require.Len(t, profiles, 3)
	assert.Equal(t, "minified", profiles[0].Name)
	assert.Equal(t, registry_dto.PriorityWant, profiles[0].Profile.Priority)
	for _, p := range profiles {
		if p.Name == "gzip" || p.Name == "br" {
			assert.Equal(t, "minified", p.Profile.DependsOn.First())
		}
	}
}

func TestBuildCompressChain(t *testing.T) {
	t.Parallel()

	profiles := buildCompressChain("source", "asset", "image/png", ".png")

	require.Len(t, profiles, 2)
	profileNames := make([]string, len(profiles))
	for i, p := range profiles {
		profileNames[i] = p.Name
	}
	assert.Contains(t, profileNames, "gzip")
	assert.Contains(t, profileNames, "br")
	for _, p := range profiles {
		assert.Equal(t, "source", p.Profile.DependsOn.First())
	}
}

func TestBuildMinifyCompressChainWithPriority(t *testing.T) {
	t.Parallel()

	profiles := buildMinifyCompressChainWithPriority(
		capabilities_dto.CapabilityMinifyJS.String(),
		"source",
		"minified-pk-js",
		"application/javascript",
		".min.js",
		registry_dto.PriorityNeed,
		"compressed-pk-js",
	)

	require.Len(t, profiles, 3)

	for _, p := range profiles {
		if p.Name == "minified" {
			assert.Equal(t, registry_dto.PriorityNeed, p.Profile.Priority)
		}
	}
}
func TestBuildComponentProfiles(t *testing.T) {
	t.Parallel()

	ctx := profileContext{
		artefactID: "components/my-button.pkc",
		ext:        ".pkc",
	}

	profiles := buildComponentProfiles(ctx)

	require.NotEmpty(t, profiles)
	var foundCompiledJS bool
	for _, p := range profiles {
		if p.Name == "compiled_js" {
			foundCompiledJS = true
			tagName, ok := p.Profile.ResultingTags.GetByName("tagName")
			assert.True(t, ok)
			assert.Equal(t, "my-button", tagName)
			sourcePath, ok := p.Profile.Params.GetByName("sourcePath")
			assert.True(t, ok)
			assert.Equal(t, "components/my-button.pkc", sourcePath)
		}
	}
	assert.True(t, foundCompiledJS)
}

func TestBuildStaticAssetProfiles(t *testing.T) {
	t.Parallel()

	ctx := profileContext{
		artefactID: "favicon.ico",
		ext:        ".ico",
	}

	profiles := buildStaticAssetProfiles(ctx)

	require.Len(t, profiles, 2)

	for _, p := range profiles {

		mimeType, ok := p.Profile.ResultingTags.GetByName("mimeType")
		assert.True(t, ok)
		assert.Equal(t, "image/x-icon", mimeType)
	}
}

func TestBuildSVGProfiles(t *testing.T) {
	t.Parallel()

	ctx := profileContext{
		artefactID: "icons/logo.svg",
		ext:        ".svg",
	}

	profiles := buildSVGProfiles(ctx)

	require.NotEmpty(t, profiles)

	for _, p := range profiles {
		if p.Name == "minified" {
			assert.Equal(t, capabilities_dto.CapabilityMinifySVG.String(), p.Profile.CapabilityName)

			mimeType, ok := p.Profile.ResultingTags.GetByName("mimeType")
			assert.True(t, ok)
			assert.Equal(t, "image/svg+xml", mimeType)
		}
	}
}

func TestBuildCSSProfiles(t *testing.T) {
	t.Parallel()

	ctx := profileContext{
		artefactID: "styles/main.css",
		ext:        ".css",
	}

	profiles := buildCSSProfiles(ctx)

	require.NotEmpty(t, profiles)

	for _, p := range profiles {
		if p.Name == "minified" {
			assert.Equal(t, capabilities_dto.CapabilityMinifyCSS.String(), p.Profile.CapabilityName)

			mimeType, ok := p.Profile.ResultingTags.GetByName("mimeType")
			assert.True(t, ok)
			assert.Equal(t, "text/css", mimeType)
		}
	}
}

func TestIsPreMinified(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "regular JS", input: "scripts/app.js", expected: false},
		{name: "min.js", input: "lib/vendor.min.js", expected: true},
		{name: "min.es.js", input: "lib/vendor-editor.min.es.js", expected: true},
		{name: "min.umd.js", input: "lib/vendor-editor.min.umd.js", expected: true},
		{name: "es.js not minified", input: "lib/vendor-editor.es.js", expected: false},
		{name: "pk-js regular", input: "pk-js/page-home.js", expected: false},
		{name: "nested min.js", input: "lib/js/react.min.js", expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isPreMinified(tc.input))
		})
	}
}

func TestBuildJSProfiles(t *testing.T) {
	t.Parallel()

	t.Run("regular JS file", func(t *testing.T) {
		t.Parallel()

		ctx := profileContext{
			artefactID: "scripts/app.js",
			ext:        ".js",
		}

		profiles := buildJSProfiles(ctx)

		require.NotEmpty(t, profiles)

		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, registry_dto.PriorityWant, p.Profile.Priority)
			}
		}
	})

	t.Run("pk-js file has high priority", func(t *testing.T) {
		t.Parallel()

		ctx := profileContext{
			artefactID: "pk-js/page-home.js",
			ext:        ".js",
		}

		profiles := buildJSProfiles(ctx)

		require.NotEmpty(t, profiles)

		for _, p := range profiles {
			if p.Name == "minified" {
				assert.Equal(t, registry_dto.PriorityNeed, p.Profile.Priority)
			}
		}
	})

	t.Run("pre-minified JS skips minification", func(t *testing.T) {
		t.Parallel()

		ctx := profileContext{
			artefactID: "lib/js/vendor-editor.min.es.js",
			ext:        ".js",
		}

		profiles := buildJSProfiles(ctx)

		require.NotEmpty(t, profiles)

		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}

		assert.NotContains(t, profileNames, "minified")
		assert.Contains(t, profileNames, "gzip")
		assert.Contains(t, profileNames, "br")
	})

	t.Run("pre-minified min.js skips minification", func(t *testing.T) {
		t.Parallel()

		ctx := profileContext{
			artefactID: "lib/vendor.min.js",
			ext:        ".js",
		}

		profiles := buildJSProfiles(ctx)

		require.NotEmpty(t, profiles)

		profileNames := make([]string, len(profiles))
		for i, p := range profiles {
			profileNames[i] = p.Name
		}

		assert.NotContains(t, profileNames, "minified")
		assert.Contains(t, profileNames, "gzip")
		assert.Contains(t, profileNames, "br")
	})
}
