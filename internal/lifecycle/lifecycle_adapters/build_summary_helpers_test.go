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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
)

func TestExtractGutterContent(t *testing.T) {
	t.Parallel()

	t.Run("strips numbered gutter prefix", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("    25 | const v = 2;")

		assert.Equal(t, "const v = 2;", result)
	})

	t.Run("strips empty gutter prefix", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("       | ")

		assert.Equal(t, "", result)
	})

	t.Run("strips caret gutter prefix", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("       |           ^")

		assert.Equal(t, "          ^", result)
	})

	t.Run("returns trimmed content when no pipe", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("  no pipe here  ")

		assert.Equal(t, "no pipe here", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("")

		assert.Equal(t, "", result)
	})

	t.Run("handles trailing pipe with no content", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("       |")

		assert.Equal(t, "", result)
	})

	t.Run("handles pipe with spaces only after", func(t *testing.T) {
		t.Parallel()

		result := extractGutterContent("   | some code")

		assert.Equal(t, "some code", result)
	})
}

func TestCleanBuildError_ExtendedCases(t *testing.T) {
	t.Parallel()

	t.Run("strips both sentinels in chain", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("some error: capability: fatal error: orchestrator: fatal error")

		assert.Equal(t, "some error", got)
	})

	t.Run("preserves partial match of sentinel", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("error: orchestrator: not fatal")

		assert.Equal(t, "error: orchestrator: not fatal", got)
	})
}

func TestExtractBuildErrorSnippet_ExtendedCases(t *testing.T) {
	t.Parallel()

	t.Run("extracts location from simple message", func(t *testing.T) {
		t.Parallel()

		input := `Unexpected token (line 10, col 3)`

		snippet := extractBuildErrorSnippet(input)

		require.NotNil(t, snippet)
		assert.Equal(t, 10, snippet.line)
		assert.Equal(t, 3, snippet.column)
		assert.Equal(t, "Unexpected token", snippet.coreMessage)
	})

	t.Run("extracts with source but no caret", func(t *testing.T) {
		t.Parallel()

		input := "Error message (line 5, col 0)\n    5 | let x = bad;"

		snippet := extractBuildErrorSnippet(input)

		require.NotNil(t, snippet)
		assert.Equal(t, 5, snippet.line)
		assert.Equal(t, 0, snippet.column)
		assert.Equal(t, "let x = bad;", snippet.sourceText)
		assert.Empty(t, snippet.caretLine)
	})

	t.Run("handles multi-digit line and column", func(t *testing.T) {
		t.Parallel()

		input := `Something failed (line 999, col 123)`

		snippet := extractBuildErrorSnippet(input)

		require.NotNil(t, snippet)
		assert.Equal(t, 999, snippet.line)
		assert.Equal(t, 123, snippet.column)
	})
}

func TestExtractCoreMessage_ExtendedCases(t *testing.T) {
	t.Parallel()

	t.Run("handles deeply nested chain", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage("level1: level2: level3: Final message")

		assert.Equal(t, "Final message", got)
	})

	t.Run("handles chain with lowercase after colon", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage("wrapper: error parsing: unexpected token")

		assert.Equal(t, "wrapper: error parsing: unexpected token", got)
	})

	t.Run("handles empty input", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage("")

		assert.Equal(t, "", got)
	})

	t.Run("handles single character", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage("X")

		assert.Equal(t, "X", got)
	})
}

func TestBuildJSArtefactToPartialNameMap(t *testing.T) {
	t.Parallel()

	t.Run("builds map from partials with JS artefact IDs", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Partials: map[string]generator_dto.ManifestPartialEntry{
				"partials/card.pk": {
					JSArtefactID: "js-card-abc123",
					PartialName:  "card",
				},
				"partials/header.pk": {
					JSArtefactID: "js-header-def456",
					PartialName:  "header",
				},
			},
		}

		result := buildJSArtefactToPartialNameMap(manifest)

		assert.Len(t, result, 2)
		assert.Equal(t, "card", result["js-card-abc123"])
		assert.Equal(t, "header", result["js-header-def456"])
	})

	t.Run("skips partials without JS artefact ID", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Partials: map[string]generator_dto.ManifestPartialEntry{
				"partials/card.pk": {
					JSArtefactID: "js-card-abc123",
					PartialName:  "card",
				},
				"partials/static.pk": {
					JSArtefactID: "",
					PartialName:  "static",
				},
			},
		}

		result := buildJSArtefactToPartialNameMap(manifest)

		assert.Len(t, result, 1)
		assert.Equal(t, "card", result["js-card-abc123"])
	})

	t.Run("returns empty map for no partials", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{
			Partials: map[string]generator_dto.ManifestPartialEntry{},
		}

		result := buildJSArtefactToPartialNameMap(manifest)

		assert.Empty(t, result)
	})

	t.Run("returns empty map for nil partials", func(t *testing.T) {
		t.Parallel()

		manifest := &generator_dto.Manifest{}

		result := buildJSArtefactToPartialNameMap(manifest)

		assert.Empty(t, result)
	})
}

func TestFormatSourceSnippet(t *testing.T) {
	t.Parallel()

	t.Run("formats snippet with caret", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		snippet := &buildErrorSnippet{
			sourceText: "    const v = 2;",
			caretLine:  "          ^",
			line:       25,
		}

		formatSourceSnippet(&builder, snippet)
		output := builder.String()

		assert.Contains(t, output, "25 |")
		assert.Contains(t, output, "const v = 2;")
		assert.Contains(t, output, "^")
	})

	t.Run("formats snippet without caret", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		snippet := &buildErrorSnippet{
			sourceText: "let x = bad;",
			caretLine:  "",
			line:       5,
		}

		formatSourceSnippet(&builder, snippet)
		output := builder.String()

		assert.Contains(t, output, "5 |")
		assert.Contains(t, output, "let x = bad;")
	})

	t.Run("formats snippet with multi-digit line number", func(t *testing.T) {
		t.Parallel()

		var builder strings.Builder
		snippet := &buildErrorSnippet{
			sourceText: "code here",
			caretLine:  "    ^^^",
			line:       1234,
		}

		formatSourceSnippet(&builder, snippet)
		output := builder.String()

		assert.Contains(t, output, "1234 |")
	})
}
