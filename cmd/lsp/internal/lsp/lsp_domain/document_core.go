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

package lsp_domain

import (
	"sync"

	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safeconv"
)

// document represents a single, fully analysed .pk file.
// It is an immutable snapshot of a file at a specific version, containing
// the full semantic analysis results including the AST and analysis context map.
type document struct {
	// Resolver provides path resolution for finding partial components and asset
	// files in the document; nil turns off path resolution.
	Resolver resolver_domain.ResolverPort

	// TypeInspector provides access to type information for LSP features such as
	// member access completions, signature help, and go-to-definition.
	TypeInspector TypeInspectorPort

	// AnalysisMap links each TemplateNode to its AnalysisContext, which holds
	// the symbol table. This map enables code intelligence features such as
	// completions and identifier resolution.
	AnalysisMap map[*ast_domain.TemplateNode]*annotator_domain.AnalysisContext

	// ProjectResult holds the full project analysis from the coordinator; nil when
	// analysis has not yet finished.
	ProjectResult *annotator_dto.ProjectAnnotationResult

	// AnnotationResult holds the parsed AST and style blocks for this document.
	AnnotationResult *annotator_dto.AnnotationResult

	// SFCResult caches the parsed SFC structure (template, scripts, styles, i18n
	// blocks). This avoids re-parsing the file for each LSP operation.
	SFCResult *sfcparser.ParseResult

	// pkcMeta caches the extracted PKC metadata for this document snapshot.
	pkcMeta *pkcMetadata

	// URI is the unique identifier for this document in the language server protocol.
	URI protocol.DocumentURI

	// Content holds the raw document text as bytes.
	Content []byte

	// sfcOnce guards single execution of getSFCResult, even when called
	// from multiple goroutines at the same time.
	sfcOnce sync.Once

	// pkcOnce guards single execution of getPKCMetadata.
	pkcOnce sync.Once

	// dirty indicates whether the document needs to be analysed again.
	dirty bool
}

// getSFCResult returns the cached SFC parse result, parsing on first access.
// Returns nil if parsing fails.
//
// Returns *sfcparser.ParseResult which is the parsed SFC data, or nil on error.
func (d *document) getSFCResult() *sfcparser.ParseResult {
	d.sfcOnce.Do(func() {
		result, err := sfcparser.Parse(d.Content)
		if err != nil {
			return
		}
		d.SFCResult = result
	})
	return d.SFCResult
}

// isPositionInClientScript checks if the given position is inside a client-side
// script block (JavaScript or TypeScript).
//
// Takes position (protocol.Position) which is the cursor position to check.
//
// Returns bool which is true if the position is within a client script block.
func (d *document) isPositionInClientScript(position protocol.Position) bool {
	sfc := d.getSFCResult()
	if sfc == nil {
		return false
	}
	for i := range sfc.Scripts {
		script := &sfc.Scripts[i]
		if script.IsClientScript() && d.isPositionInScriptContent(script, position) {
			return true
		}
	}
	return false
}

// isPositionInScriptContent checks if a position falls within the content of a
// script block.
//
// Takes script (*sfcparser.Script) which is the script block to check.
// Takes position (protocol.Position) which is the cursor position to test.
//
// Returns bool which is true if the position is inside the script content.
func (*document) isPositionInScriptContent(script *sfcparser.Script, position protocol.Position) bool {
	startLine := safeconv.IntToUint32(script.ContentLocation.Line - 1)

	endLine := startLine + safeconv.IntToUint32(countNewlinesInContent(script.Content))

	if position.Line < startLine || position.Line > endLine {
		return false
	}

	if startLine == endLine {
		startCol := safeconv.IntToUint32(script.ContentLocation.Column - 1)
		endCol := startCol + safeconv.IntToUint32(len(script.Content))
		return position.Character >= startCol && position.Character <= endCol
	}

	return true
}

// countNewlinesInContent counts the number of newline characters in content.
//
// Takes content (string) which is the text to search for newlines.
//
// Returns int which is the count of newline characters found.
func countNewlinesInContent(content string) int {
	count := 0
	for i := range len(content) {
		if content[i] == '\n' {
			count++
		}
	}
	return count
}
