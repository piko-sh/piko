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

package gopls_bridge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	sampleBlock = `package main

import (
	"piko.sh/piko"
	layout "github.com/example/site/partials/layout.pk"
)

type Response struct {
	Title string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
	return Response{Title: "Doodle Cards"}, piko.Metadata{}, nil
}
`
)

func TestRewriteBlockRewritesPkImportsPreservingLineCount(t *testing.T) {
	t.Parallel()

	aliasToCanonical := map[string]string{
		"layout": "github.com/example/site/.piko/partials/layout_abc123",
	}
	rewritten := RewriteBlock(sampleBlock, aliasToCanonical)

	assert.Contains(t, rewritten, `layout "github.com/example/site/.piko/partials/layout_abc123"`)
	assert.NotContains(t, rewritten, "layout.pk")
	assert.Contains(t, rewritten, `"piko.sh/piko"`, "non-pk imports are untouched")
	assert.Equal(t, strings.Count(sampleBlock, "\n"), strings.Count(rewritten, "\n"), "line count must be preserved")
	assert.Contains(t, rewritten, `func Render(r *piko.RequestData`, "the body is verbatim")
}

func TestRewriteBlockNoOpWhenNoMapping(t *testing.T) {
	t.Parallel()

	assert.Equal(t, sampleBlock, RewriteBlock(sampleBlock, nil))
	assert.Equal(t, sampleBlock, RewriteBlock(sampleBlock, map[string]string{"other": "x"}))
}

func TestRewriteBlockLeavesNonPkImports(t *testing.T) {
	t.Parallel()

	block := "package main\n\nimport \"fmt\"\n\nfunc F() { fmt.Println() }\n"
	assert.Equal(t, block, RewriteBlock(block, map[string]string{"fmt": "rewritten"}))
}

func TestBuildVirtualDocPlacesBlockInIsolatedPackageDir(t *testing.T) {
	t.Parallel()

	doc := BuildVirtualDoc("file:///site/pages/cards.pk", VirtualDocInput{
		AliasToCanonical: map[string]string{"layout": "github.com/example/site/.piko/partials/layout_abc123"},
		ModuleRoot:       "/site",
		HashedName:       "pages_cards_def456",
		BlockContent:     sampleBlock,
		ContentLine:      143,
		ContentColumn:    1,
	})

	assert.Contains(t, string(doc.Mapper.VirtualURI()), "/piko-lsp/pages_cards_def456/source.pk.go")
	assert.Contains(t, string(doc.Content), "layout_abc123")
	assert.Equal(t, 143, doc.Mapper.contentLine)
}
