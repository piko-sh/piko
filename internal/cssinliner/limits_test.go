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

package cssinliner

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestLimits_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		given         Limits
		wantDepth     int
		wantTotalSize int
	}{
		{
			name:          "a zero value takes both defaults",
			given:         Limits{},
			wantDepth:     DefaultMaxImportDepth,
			wantTotalSize: DefaultMaxInlinedBytes,
		},
		{
			name:          "a set depth keeps the default size",
			given:         Limits{MaxDepth: 3},
			wantDepth:     3,
			wantTotalSize: DefaultMaxInlinedBytes,
		},
		{
			name:          "a set size keeps the default depth",
			given:         Limits{MaxTotalBytes: 512},
			wantDepth:     DefaultMaxImportDepth,
			wantTotalSize: 512,
		},
		{
			name:          "a negative value falls back to the default",
			given:         Limits{MaxDepth: -1, MaxTotalBytes: -1},
			wantDepth:     DefaultMaxImportDepth,
			wantTotalSize: DefaultMaxInlinedBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.given.withDefaults()

			assert.Equal(t, tt.wantDepth, got.MaxDepth)
			assert.Equal(t, tt.wantTotalSize, got.MaxTotalBytes)
		})
	}
}

func TestCSSInliner_StopsAtTheImportDepthLimit(t *testing.T) {
	t.Parallel()

	fsReader := newTestFSReader()
	for depth := range 10 {
		fsReader.addFile(fmt.Sprintf("/test/level%d.css", depth),
			fmt.Sprintf("@import %q;\n.level%d { color: red }", fmt.Sprintf("/test/level%d.css", depth+1), depth))
	}
	fsReader.addFile("/test/level10.css", ".leaf { color: blue }")

	processor := newTestProcessor(newPassthroughResolver())
	inliner := GetInliner(processor.GetResolver(), processor.GetParserOptions(),
		fsReader, "TEST", "TEST-IMPORT", nil, Limits{MaxDepth: 3})

	tree, diagnostics := inliner.InlineAndParse(context.Background(),
		`@import "/test/level0.css";`, "/test/main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

	assert.Nil(t, tree, "a chain past the depth limit must not produce a tree")
	require.NotEmpty(t, diagnostics, "the limit must be reported, never applied silently")
	assert.Contains(t, diagnostics[0].Message, "nest deeper than the limit")
}

func TestCSSInliner_StopsAtTheTotalSizeLimit(t *testing.T) {
	t.Parallel()

	fsReader := newTestFSReader()
	fsReader.addFile("/test/big.css", ".a { color: red } /*"+strings.Repeat("x", 4096)+"*/")

	processor := newTestProcessor(newPassthroughResolver())
	inliner := GetInliner(processor.GetResolver(), processor.GetParserOptions(),
		fsReader, "TEST", "TEST-IMPORT", nil, Limits{MaxTotalBytes: 64})

	tree, diagnostics := inliner.InlineAndParse(context.Background(),
		`@import "/test/big.css";`, "/test/main.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0})

	assert.Nil(t, tree, "imports past the size limit must not produce a tree")
	require.NotEmpty(t, diagnostics, "the limit must be reported, never applied silently")
	assert.Contains(t, diagnostics[0].Message, "total more than the limit")
}

func TestCSSInliner_InlinesASharedImportOnce(t *testing.T) {
	t.Parallel()

	fsReader := newTestFSReader()
	fsReader.addFile("/test/shared.css", ".shared { color: red }")
	fsReader.addFile("/test/a.css", "@import \"/test/shared.css\";\n.a { color: blue }")
	fsReader.addFile("/test/b.css", "@import \"/test/shared.css\";\n.b { color: green }")

	processor := newTestProcessor(newPassthroughResolver())

	result, diagnostics, err := processor.Process(context.Background(),
		"@import \"/test/a.css\";\n@import \"/test/b.css\";", "/test/main.css",
		ast_domain.Location{Line: 1, Column: 1, Offset: 0}, fsReader)

	require.NoError(t, err)
	assert.Empty(t, diagnostics)
	assert.Contains(t, result, ".a")
	assert.Contains(t, result, ".b")
	assert.Equal(t, 1, strings.Count(result, ".shared"),
		"a stylesheet reached by two import chains must be inlined once")
}
