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

//go:build bench

package render_adapters

import (
	"testing"

	"piko.sh/piko/internal/registry/registry_dto"
)

func BenchmarkExtractTagContent(b *testing.B) {
	testCases := []struct {
		name    string
		rawHTML string
	}{
		{
			name:    "SimpleSVG",
			rawHTML: `<svg viewBox="0 0 24 24"><path d="M12 2L2 7"></path></svg>`,
		},
		{
			name:    "ComplexSVG",
			rawHTML: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="icon-home" data-testid="home-icon"><path d="M12 2L2 7v10l10 5 10-5V7l-10-5z"></path><circle cx="12" cy="12" r="5"></circle><rect x="5" y="5" width="14" height="14"></rect></svg>`,
		},
		{
			name:    "SVGWithWhitespace",
			rawHTML: "   \n\t<svg viewBox=\"0 0 24 24\">\n\t\t<path d=\"M12 2\"></path>\n\t</svg>\n   ",
		},
		{
			name:    "MixedCaseSVG",
			rawHTML: `<SVG viewBox="0 0 24 24"><path d="M12 2"></path></SVG>`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				_, _ = extractTagContent(tc.rawHTML, "svg")
			}
		})
	}
}

func BenchmarkFindJSVariant(b *testing.B) {
	variants := []registry_dto.Variant{
		{
			VariantID:  "v1",
			StorageKey: "dist/style.css",
			MetadataTags: registry_dto.TagsFromMap(map[string]string{
				"type": "component-css",
				"role": "stylesheet",
			}),
		},
		{
			VariantID:  "v2",
			StorageKey: "dist/component.js",
			MetadataTags: registry_dto.TagsFromMap(map[string]string{
				"type": "component-js",
				"role": "entrypoint",
			}),
		},
		{
			VariantID:  "v3",
			StorageKey: "dist/component.min.js",
			MetadataTags: registry_dto.TagsFromMap(map[string]string{
				"type": "component-js",
				"role": "minified",
			}),
		},
	}

	b.Run("Found", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = findJSVariant(variants)
		}
	})

	b.Run("NotFound", func(b *testing.B) {
		noJSVariants := []registry_dto.Variant{
			{
				VariantID:  "v1",
				StorageKey: "dist/style.css",
				MetadataTags: registry_dto.TagsFromMap(map[string]string{
					"type": "component-css",
				}),
			},
		}

		b.ReportAllocs()
		for b.Loop() {
			_ = findJSVariant(noJSVariants)
		}
	})
}
