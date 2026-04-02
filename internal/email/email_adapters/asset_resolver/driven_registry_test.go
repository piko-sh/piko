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

package asset_resolver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestExtractFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourcePath string
		expected   string
	}{
		{
			name:       "nested path returns last segment",
			sourcePath: "assets/images/logo.png",
			expected:   "logo.png",
		},
		{
			name:       "single filename without slashes",
			sourcePath: "banner.jpg",
			expected:   "banner.jpg",
		},
		{
			name:       "deeply nested path",
			sourcePath: "a/b/c/d/e/photo.webp",
			expected:   "photo.webp",
		},
		{
			name:       "trailing slash returns empty string",
			sourcePath: "assets/images/",
			expected:   "",
		},
		{
			name:       "empty string returns empty string",
			sourcePath: "",
			expected:   "",
		},
		{
			name:       "single slash returns empty string",
			sourcePath: "/",
			expected:   "",
		},
		{
			name:       "leading slash with filename",
			sourcePath: "/logo.png",
			expected:   "logo.png",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := extractFilename(testCase.sourcePath)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestVariantMatchesProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		variantID string
		profile   string
		expected  bool
	}{
		{
			name:      "exact match returns true",
			variantID: "email-default",
			profile:   "email-default",
			expected:  true,
		},
		{
			name:      "different profile returns false",
			variantID: "email-default",
			profile:   "email-outlook",
			expected:  false,
		},
		{
			name:      "empty profile matches empty variant ID",
			variantID: "",
			profile:   "",
			expected:  true,
		},
		{
			name:      "source variant does not match named profile",
			variantID: "source",
			profile:   "email-default",
			expected:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			variant := &registry_dto.Variant{VariantID: testCase.variantID}
			result := variantMatchesProfile(variant, testCase.profile)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestVariantMatchesWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tagWidth string
		width    int
		expected bool
	}{
		{
			name:     "zero width always matches",
			tagWidth: "",
			width:    0,
			expected: true,
		},
		{
			name:     "px format matches",
			tagWidth: "300px",
			width:    300,
			expected: true,
		},
		{
			name:     "raw number format matches",
			tagWidth: "300",
			width:    300,
			expected: true,
		},
		{
			name:     "width mismatch returns false",
			tagWidth: "600px",
			width:    300,
			expected: false,
		},
		{
			name:     "empty tag with nonzero width returns false",
			tagWidth: "",
			width:    300,
			expected: false,
		},
		{
			name:     "non-numeric tag with valid width returns false",
			tagWidth: "large",
			width:    300,
			expected: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tags := makeTags(testCase.tagWidth, "")
			variant := &registry_dto.Variant{MetadataTags: tags}
			result := variantMatchesWidth(variant, testCase.width)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestVariantMatchesDensity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tagDensity string
		density    string
		expected   bool
	}{
		{
			name:       "empty density always matches",
			tagDensity: "",
			density:    "",
			expected:   true,
		},
		{
			name:       "matching density returns true",
			tagDensity: "x2",
			density:    "x2",
			expected:   true,
		},
		{
			name:       "mismatched density returns false",
			tagDensity: "x1",
			density:    "x2",
			expected:   false,
		},
		{
			name:       "empty tag with requested density returns false",
			tagDensity: "",
			density:    "x2",
			expected:   false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tags := makeTags("", testCase.tagDensity)
			variant := &registry_dto.Variant{MetadataTags: tags}
			result := variantMatchesDensity(variant, testCase.density)
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestFindExactMatch(t *testing.T) {
	t.Parallel()

	t.Run("returns variant matching profile width and density", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
			makeVariant("email-default", "image/png", makeTags("300px", "x2")),
			makeVariant("email-outlook", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findExactMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "email-default", result.VariantID)
		assert.Equal(t, "x2", result.MetadataTags.Get(registry_dto.TagDensity))
	})

	t.Run("returns nil when no exact match exists", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findExactMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("returns nil for empty variants slice", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findExactMatch(nil, request)
		assert.Nil(t, result)
	})

	t.Run("matches when width and density are zero or empty", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   0,
			Density: "",
		}

		result := findExactMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "email-default", result.VariantID)
	})
}

func TestFindProfileAndWidthMatch(t *testing.T) {
	t.Parallel()

	t.Run("returns variant matching profile and width ignoring density", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x3",
		}

		result := findProfileAndWidthMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "email-default", result.VariantID)
	})

	t.Run("returns nil when density is empty", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "",
		}

		result := findProfileAndWidthMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("returns nil when profile does not match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-outlook", "image/png", makeTags("300px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findProfileAndWidthMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("returns nil when width does not match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findProfileAndWidthMatch(variants, request)
		assert.Nil(t, result)
	})
}

func TestFindProfileOnlyMatch(t *testing.T) {
	t.Parallel()

	t.Run("returns variant matching profile only when width was requested", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "",
		}

		result := findProfileOnlyMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "email-default", result.VariantID)
	})

	t.Run("returns variant matching profile only when density was requested", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   0,
			Density: "x2",
		}

		result := findProfileOnlyMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "email-default", result.VariantID)
	})

	t.Run("returns nil when no dimensions were requested", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   0,
			Density: "",
		}

		result := findProfileOnlyMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("returns nil when profile does not match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-outlook", "image/png", makeTags("300px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x1",
		}

		result := findProfileOnlyMatch(variants, request)
		assert.Nil(t, result)
	})
}

func TestFindDimensionMatch(t *testing.T) {
	t.Parallel()

	t.Run("returns variant matching width and density ignoring profile", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("other-profile", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findDimensionMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "other-profile", result.VariantID)
	})

	t.Run("returns nil when profile is empty", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("something", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "",
			Width:   300,
			Density: "x2",
		}

		result := findDimensionMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("returns nil when dimensions do not match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("other-profile", "image/png", makeTags("600px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		result := findDimensionMatch(variants, request)
		assert.Nil(t, result)
	})

	t.Run("matches with zero width and empty density", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("other-profile", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   0,
			Density: "",
		}

		result := findDimensionMatch(variants, request)
		require.NotNil(t, result)
		assert.Equal(t, "other-profile", result.VariantID)
	})
}

func TestFindSourceVariant(t *testing.T) {
	t.Parallel()

	t.Run("returns source variant when present", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
			makeVariant("source", "image/png", makeTags("", "")),
			makeVariant("email-outlook", "image/png", makeTags("300px", "x2")),
		}

		result := findSourceVariant(variants)
		require.NotNil(t, result)
		assert.Equal(t, "source", result.VariantID)
	})

	t.Run("returns nil when no source variant exists", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
			makeVariant("email-outlook", "image/png", makeTags("300px", "x2")),
		}

		result := findSourceVariant(variants)
		assert.Nil(t, result)
	})

	t.Run("returns nil for empty variants slice", func(t *testing.T) {
		t.Parallel()

		result := findSourceVariant(nil)
		assert.Nil(t, result)
	})
}

func TestFindBestMatchingVariant(t *testing.T) {
	t.Parallel()

	a := &adapter{}

	t.Run("returns nil for empty variants", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
		}

		variant, fallback := a.findBestMatchingVariant(nil, request)
		assert.Nil(t, variant)
		assert.Empty(t, fallback)
	})

	t.Run("returns exact match with no fallback", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x2")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Empty(t, fallback)
	})

	t.Run("falls back to profile and width match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x3",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Contains(t, fallback, "density")
	})

	t.Run("falls back to profile only match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Contains(t, fallback, "profile matches")
	})

	t.Run("falls back to dimension match", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("other-profile", "image/png", makeTags("300px", "x2")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "other-profile", variant.VariantID)
		assert.Contains(t, fallback, "profile")
		assert.Contains(t, fallback, "not configured")
	})

	t.Run("falls back to source variant", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("unrelated-profile", "image/png", makeTags("999px", "x3")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "source", variant.VariantID)
		assert.Contains(t, fallback, "source variant")
	})

	t.Run("falls back to first available variant", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("compressed", "image/webp", makeTags("999px", "x3")),
			makeVariant("thumbnail", "image/jpeg", makeTags("100px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "compressed", variant.VariantID)
		assert.Contains(t, fallback, "first available")
	})

	t.Run("profile-only request matches directly", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   0,
			Density: "",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Empty(t, fallback)
	})
}

func TestBuildAttachment(t *testing.T) {
	t.Parallel()

	t.Run("builds attachment with correct fields", func(t *testing.T) {
		t.Parallel()
		variant := &registry_dto.Variant{
			MimeType: "image/png",
		}
		content := []byte("fake image data")

		attachment := buildAttachment("assets/images/logo.png", variant, content, "logo_abc123")

		assert.Equal(t, "logo.png", attachment.Filename)
		assert.Equal(t, "image/png", attachment.MIMEType)
		assert.Equal(t, content, attachment.Content)
		assert.Equal(t, "logo_abc123", attachment.ContentID)
	})

	t.Run("extracts filename from simple path", func(t *testing.T) {
		t.Parallel()
		variant := &registry_dto.Variant{
			MimeType: "image/jpeg",
		}

		attachment := buildAttachment("banner.jpg", variant, []byte("data"), "cid1")
		assert.Equal(t, "banner.jpg", attachment.Filename)
	})
}

func TestBuildSpanAttributes(t *testing.T) {
	t.Parallel()

	t.Run("includes base attributes", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "logo_abc",
			Width:      0,
			Density:    "",
		}

		attributes := buildSpanAttributes(request)
		assert.Len(t, attributes, 3)
	})

	t.Run("includes width when nonzero", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "logo_abc",
			Width:      300,
			Density:    "",
		}

		attributes := buildSpanAttributes(request)
		assert.Len(t, attributes, 4)
	})

	t.Run("includes density when nonempty", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "logo_abc",
			Width:      0,
			Density:    "x2",
		}

		attributes := buildSpanAttributes(request)
		assert.Len(t, attributes, 4)
	})

	t.Run("includes both width and density", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "logo_abc",
			Width:      300,
			Density:    "x2",
		}

		attributes := buildSpanAttributes(request)
		assert.Len(t, attributes, 5)
	})
}

func TestSelectVariant(t *testing.T) {
	t.Parallel()

	a := &adapter{}

	t.Run("returns matching variant with no error", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("email-default", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			Width:      300,
			Density:    "x2",
		}

		variant, fallback, err := a.selectVariant(variants, request)
		require.NoError(t, err)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Empty(t, fallback)
	})

	t.Run("returns error when no variants available", func(t *testing.T) {
		t.Parallel()
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
		}

		variant, _, err := a.selectVariant(nil, request)
		require.Error(t, err)
		assert.Nil(t, variant)
		assert.Contains(t, err.Error(), "no usable variants")
	})

	t.Run("returns fallback description when using fallback", func(t *testing.T) {
		t.Parallel()
		variants := []registry_dto.Variant{
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			Width:      300,
			Density:    "x2",
		}

		variant, fallback, err := a.selectVariant(variants, request)
		require.NoError(t, err)
		require.NotNil(t, variant)
		assert.Equal(t, "source", variant.VariantID)
		assert.NotEmpty(t, fallback)
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("returns non-nil adapter implementing AssetResolverPort", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{}
		resolver := New(mock)
		require.NotNil(t, resolver)

		var _ email_domain.AssetResolverPort = resolver
	})
}

func TestResolveAsset(t *testing.T) {
	t.Parallel()

	t.Run("resolves asset successfully with exact variant match", func(t *testing.T) {
		t.Parallel()
		expectedContent := []byte("png image bytes")
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				return &registry_dto.ArtefactMeta{
					ID:         artefactID,
					SourcePath: artefactID,
					ActualVariants: []registry_dto.Variant{
						makeVariant("email-default", "image/png", makeTags("300px", "x2")),
						makeVariant("source", "image/png", makeTags("", "")),
					},
				}, nil
			},
			GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(expectedContent)), nil
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/images/logo.png",
			Profile:    "email-default",
			Width:      300,
			Density:    "x2",
			CID:        "logo_abc123",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, attachment)
		assert.Equal(t, "logo.png", attachment.Filename)
		assert.Equal(t, "image/png", attachment.MIMEType)
		assert.Equal(t, expectedContent, attachment.Content)
		assert.Equal(t, "logo_abc123", attachment.ContentID)
		assert.Equal(t, int64(1), mock.GetArtefactCallCount)
		assert.Equal(t, int64(1), mock.GetVariantDataCallCount)
	})

	t.Run("returns error when artefact fetch fails", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
				return nil, errors.New("registry unavailable")
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "cid1",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.Error(t, err)
		assert.Nil(t, attachment)
		assert.Contains(t, err.Error(), "fetching artefact")
	})

	t.Run("returns error when artefact is nil", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
				return nil, nil
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/missing.png",
			Profile:    "email-default",
			CID:        "cid1",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.Error(t, err)
		assert.Nil(t, attachment)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error when no usable variants exist", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				return &registry_dto.ArtefactMeta{
					ID:             artefactID,
					ActualVariants: []registry_dto.Variant{},
				}, nil
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "cid1",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.Error(t, err)
		assert.Nil(t, attachment)
		assert.Contains(t, err.Error(), "selecting variant")
	})

	t.Run("returns error when variant data fetch fails", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				return &registry_dto.ArtefactMeta{
					ID: artefactID,
					ActualVariants: []registry_dto.Variant{
						makeVariant("email-default", "image/png", makeTags("", "")),
					},
				}, nil
			},
			GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
				return nil, errors.New("blob store error")
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			CID:        "cid1",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.Error(t, err)
		assert.Nil(t, attachment)
		assert.Contains(t, err.Error(), "fetching variant")
	})

	t.Run("uses fallback variant when exact match not available", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				return &registry_dto.ArtefactMeta{
					ID: artefactID,
					ActualVariants: []registry_dto.Variant{
						makeVariant("source", "image/png", makeTags("", "")),
					},
				}, nil
			},
			GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader([]byte("source data"))), nil
			},
		}

		resolver := New(mock)
		request := &email_dto.EmailAssetRequest{
			SourcePath: "assets/logo.png",
			Profile:    "email-default",
			Width:      300,
			Density:    "x2",
			CID:        "cid1",
		}

		attachment, err := resolver.ResolveAsset(context.Background(), request)
		require.NoError(t, err)
		require.NotNil(t, attachment)
		assert.Equal(t, []byte("source data"), attachment.Content)
	})
}

func TestResolveAssets(t *testing.T) {
	t.Parallel()

	t.Run("resolves multiple assets successfully", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				return &registry_dto.ArtefactMeta{
					ID: artefactID,
					ActualVariants: []registry_dto.Variant{
						makeVariant("email-default", "image/png", makeTags("", "")),
					},
				}, nil
			},
			GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader([]byte("data"))), nil
			},
		}

		resolver := New(mock)
		requests := []*email_dto.EmailAssetRequest{
			{SourcePath: "assets/logo.png", Profile: "email-default", CID: "cid1"},
			{SourcePath: "assets/banner.jpg", Profile: "email-default", CID: "cid2"},
		}

		attachments, errs := resolver.ResolveAssets(context.Background(), requests)
		assert.Len(t, attachments, 2)
		assert.Len(t, errs, 2)
		assert.Nil(t, errs[0])
		assert.Nil(t, errs[1])
	})

	t.Run("handles partial failures", func(t *testing.T) {
		t.Parallel()
		callCount := 0
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
				callCount++
				if callCount == 2 {
					return nil, errors.New("not found")
				}
				return &registry_dto.ArtefactMeta{
					ID: artefactID,
					ActualVariants: []registry_dto.Variant{
						makeVariant("email-default", "image/png", makeTags("", "")),
					},
				}, nil
			},
			GetVariantDataFunc: func(_ context.Context, _ *registry_dto.Variant) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader([]byte("data"))), nil
			},
		}

		resolver := New(mock)
		requests := []*email_dto.EmailAssetRequest{
			{SourcePath: "assets/logo.png", Profile: "email-default", CID: "cid1"},
			{SourcePath: "assets/missing.png", Profile: "email-default", CID: "cid2"},
			{SourcePath: "assets/banner.jpg", Profile: "email-default", CID: "cid3"},
		}

		attachments, errs := resolver.ResolveAssets(context.Background(), requests)
		assert.Len(t, attachments, 2)
		assert.Len(t, errs, 3)
		assert.Nil(t, errs[0])
		assert.Error(t, errs[1])
		assert.Nil(t, errs[2])
	})

	t.Run("handles empty request list", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{}
		resolver := New(mock)

		attachments, errs := resolver.ResolveAssets(context.Background(), nil)
		assert.Empty(t, attachments)
		assert.Empty(t, errs)
	})

	t.Run("all requests fail", func(t *testing.T) {
		t.Parallel()
		mock := &mockRegistryService{
			GetArtefactFunc: func(_ context.Context, _ string) (*registry_dto.ArtefactMeta, error) {
				return nil, errors.New("service down")
			},
		}

		resolver := New(mock)
		requests := []*email_dto.EmailAssetRequest{
			{SourcePath: "a.png", Profile: "p", CID: "c1"},
			{SourcePath: "b.png", Profile: "p", CID: "c2"},
		}

		attachments, errs := resolver.ResolveAssets(context.Background(), requests)
		assert.Empty(t, attachments)
		assert.Len(t, errs, 2)
		assert.Error(t, errs[0])
		assert.Error(t, errs[1])
	})
}

func TestFallbackPriorityOrdering(t *testing.T) {
	t.Parallel()

	t.Run("exact match takes priority over all fallbacks", func(t *testing.T) {
		t.Parallel()
		a := &adapter{}
		variants := []registry_dto.Variant{
			makeVariant("source", "image/png", makeTags("", "")),
			makeVariant("other-profile", "image/png", makeTags("300px", "x2")),
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
			makeVariant("email-default", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Equal(t, "x2", variant.MetadataTags.Get(registry_dto.TagDensity))
		assert.Equal(t, "300px", variant.MetadataTags.Get(registry_dto.TagWidth))
		assert.Empty(t, fallback)
	})

	t.Run("profile and width match takes priority over profile only", func(t *testing.T) {
		t.Parallel()
		a := &adapter{}
		variants := []registry_dto.Variant{
			makeVariant("source", "image/png", makeTags("", "")),
			makeVariant("email-default", "image/png", makeTags("600px", "x1")),
			makeVariant("email-default", "image/png", makeTags("300px", "x1")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x3",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Equal(t, "300px", variant.MetadataTags.Get(registry_dto.TagWidth))
		assert.Contains(t, fallback, "density")
	})

	t.Run("profile only match takes priority over dimension match", func(t *testing.T) {
		t.Parallel()
		a := &adapter{}
		variants := []registry_dto.Variant{
			makeVariant("source", "image/png", makeTags("", "")),
			makeVariant("other-profile", "image/png", makeTags("300px", "")),
			makeVariant("email-default", "image/png", makeTags("999px", "x9")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "email-default", variant.VariantID)
		assert.Contains(t, fallback, "profile matches")
	})

	t.Run("dimension match takes priority over source variant", func(t *testing.T) {
		t.Parallel()
		a := &adapter{}
		variants := []registry_dto.Variant{
			makeVariant("source", "image/png", makeTags("", "")),
			makeVariant("other-profile", "image/png", makeTags("300px", "x2")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "other-profile", variant.VariantID)
		assert.Contains(t, fallback, "not configured")
	})

	t.Run("source variant takes priority over first available", func(t *testing.T) {
		t.Parallel()
		a := &adapter{}
		variants := []registry_dto.Variant{
			makeVariant("compressed", "image/webp", makeTags("999px", "x9")),
			makeVariant("source", "image/png", makeTags("", "")),
		}
		request := &email_dto.EmailAssetRequest{
			Profile: "email-default",
			Width:   300,
			Density: "x2",
		}

		variant, fallback := a.findBestMatchingVariant(variants, request)
		require.NotNil(t, variant)
		assert.Equal(t, "source", variant.VariantID)
		assert.Contains(t, fallback, "source variant")
	})
}

func TestVariantMatchesWidthEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("width tag with px suffix but different value", func(t *testing.T) {
		t.Parallel()
		tags := makeTags("301px", "")
		variant := &registry_dto.Variant{MetadataTags: tags}
		assert.False(t, variantMatchesWidth(variant, 300))
	})

	t.Run("width tag as raw number matches", func(t *testing.T) {
		t.Parallel()
		tags := makeTags("1024", "")
		variant := &registry_dto.Variant{MetadataTags: tags}
		assert.True(t, variantMatchesWidth(variant, 1024))
	})

	t.Run("width of one pixel", func(t *testing.T) {
		t.Parallel()
		tags := makeTags("1px", "")
		variant := &registry_dto.Variant{MetadataTags: tags}
		assert.True(t, variantMatchesWidth(variant, 1))
	})
}
