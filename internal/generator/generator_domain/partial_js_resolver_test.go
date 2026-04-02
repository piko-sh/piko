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

package generator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestNewPartialJSDependencyResolver(t *testing.T) {
	t.Parallel()

	t.Run("creates resolver with empty maps from empty artefacts", func(t *testing.T) {
		t.Parallel()

		resolver := newPartialJSDependencyResolver([]*generator_dto.GeneratedArtefact{})

		require.NotNil(t, resolver)
		assert.Empty(t, resolver.artefactsByHashedName)
		assert.Empty(t, resolver.partialJSLookup)
		assert.Empty(t, resolver.importPathToHashedName)
	})

	t.Run("skips artefacts without component", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				SuggestedPath: "/output/orphan.go",
				Component:     nil,
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		assert.Empty(t, resolver.artefactsByHashedName)
	})

	t.Run("indexes artefact by hashed name", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_abc123",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		require.Contains(t, resolver.artefactsByHashedName, "page_abc123")
		assert.Equal(t, artefacts[0], resolver.artefactsByHashedName["page_abc123"])
	})

	t.Run("maps module import path to hashed name", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "card_hash",
					Source: &annotator_dto.ParsedComponent{
						ModuleImportPath: "github.com/example/components/card",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		assert.Equal(t, "card_hash", resolver.importPathToHashedName["github.com/example/components/card"])
	})

	t.Run("adds partial with JS to lookup", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				JSArtefactID: "pk-js/components/card.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "card_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "github.com/example/components/card",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		assert.Equal(t, "pk-js/components/card.js", resolver.partialJSLookup["card_hash"])
	})

	t.Run("skips partial without JS from lookup", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				JSArtefactID: "",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "static_partial",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "partial",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		assert.NotContains(t, resolver.partialJSLookup, "static_partial")
	})

	t.Run("skips non-partial from JS lookup even with JS artefact ID", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				JSArtefactID: "pk-js/pages/home.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)

		assert.NotContains(t, resolver.partialJSLookup, "page_hash")
	})
}

func TestResolveForPage(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for page with no dependencies", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports:   nil,
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		assert.Nil(t, result)
	})

	t.Run("returns nil for unknown page", func(t *testing.T) {
		t.Parallel()

		resolver := newPartialJSDependencyResolver([]*generator_dto.GeneratedArtefact{})
		result := resolver.ResolveForPage("nonexistent_page")

		assert.Nil(t, result)
	})

	t.Run("returns JS artefact ID for direct partial dependency", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/card"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/card.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "card_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/card",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		require.Len(t, result, 1)
		assert.Equal(t, "pk-js/partials/card.js", result[0])
	})

	t.Run("excludes partial without JS", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/static"},
						},
					},
				},
			},
			{
				JSArtefactID: "",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "static_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/static",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		assert.Nil(t, result)
	})

	t.Run("resolves nested dependencies", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/a"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/a.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_a_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/a",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/b"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/b.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_b_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/b",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		require.Len(t, result, 2)
		assert.Contains(t, result, "pk-js/partials/a.js")
		assert.Contains(t, result, "pk-js/partials/b.js")
	})

	t.Run("handles circular dependencies without infinite loop", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/a"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/a.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_a_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/a",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/b"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/b.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_b_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/b",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/a"},
						},
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		require.Len(t, result, 2)
		assert.Contains(t, result, "pk-js/partials/a.js")
		assert.Contains(t, result, "pk-js/partials/b.js")
	})

	t.Run("deduplicates dependencies used multiple times", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/a"},
							{Path: "example.com/partials/b"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/a.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_a_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/a",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/shared"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/b.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "partial_b_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/b",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/shared"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/shared.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "shared_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/shared",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		require.Len(t, result, 3)
		assert.Contains(t, result, "pk-js/partials/a.js")
		assert.Contains(t, result, "pk-js/partials/b.js")
		assert.Contains(t, result, "pk-js/partials/shared.js")
	})

	t.Run("handles import path that does not match any artefact", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/missing/partial"},
						},
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		assert.Nil(t, result)
	})

	t.Run("handles mixed dependencies with some having JS and some not", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType: "page",
						PikoImports: []annotator_dto.PikoImport{
							{Path: "example.com/partials/with-js"},
							{Path: "example.com/partials/no-js"},
						},
					},
				},
			},
			{
				JSArtefactID: "pk-js/partials/with-js.js",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "with_js_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/with-js",
					},
				},
			},
			{
				JSArtefactID: "",
				Component: &annotator_dto.VirtualComponent{
					HashedName: "no_js_hash",
					Source: &annotator_dto.ParsedComponent{
						ComponentType:    "partial",
						ModuleImportPath: "example.com/partials/no-js",
					},
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		require.Len(t, result, 1)
		assert.Equal(t, "pk-js/partials/with-js.js", result[0])
	})

	t.Run("handles artefact with nil source", func(t *testing.T) {
		t.Parallel()

		artefacts := []*generator_dto.GeneratedArtefact{
			{
				Component: &annotator_dto.VirtualComponent{
					HashedName: "page_hash",
					Source:     nil,
				},
			},
		}

		resolver := newPartialJSDependencyResolver(artefacts)
		result := resolver.ResolveForPage("page_hash")

		assert.Nil(t, result)
	})
}
