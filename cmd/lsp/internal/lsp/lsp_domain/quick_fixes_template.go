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
	"bytes"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safeconv"
)

// maxTagSearchLines limits how many lines to search for tag closing characters
// (> or />).
const maxTagSearchLines = 10

// findComponentTagAtPosition finds the template node (component tag) that
// contains the given position. Locates the exact tag where a diagnostic was
// reported.
//
// Takes rootNodes ([]*ast_domain.TemplateNode) which is the list of
// root template nodes to search through.
// Takes line (uint32) which is the zero-based line number to match.
//
// Returns *ast_domain.TemplateNode which is the matching component tag
// node, or nil if no tag is found at the given position.
func findComponentTagAtPosition(rootNodes []*ast_domain.TemplateNode, line, _ uint32) *ast_domain.TemplateNode {
	if rootNodes == nil {
		return nil
	}

	targetLine := int(line) + 1

	var result *ast_domain.TemplateNode

	var traverse func(*ast_domain.TemplateNode)
	traverse = func(node *ast_domain.TemplateNode) {
		if node == nil {
			return
		}

		if node.Location.Line == targetLine {
			if node.TagName != "" {
				result = node
			}
		}

		for _, child := range node.Children {
			traverse(child)
		}
	}

	for _, rootNode := range rootNodes {
		traverse(rootNode)
		if result != nil {
			break
		}
	}

	return result
}

// calculateTagEndPosition finds the closing > or /> in a component tag.
// Determines where to insert missing attributes.
//
// Takes content ([]byte) which contains the template source.
// Takes node (*ast_domain.TemplateNode) which specifies the tag to search from.
//
// Returns line (uint32) which is the zero-based line of the closing bracket.
// Returns character (uint32) which is the zero-based column of the bracket.
// Returns isSelfClosing (bool) which indicates if the tag ends with />.
// Returns found (bool) which indicates if the closing bracket was found.
func calculateTagEndPosition(content []byte, node *ast_domain.TemplateNode) (line, character uint32, isSelfClosing bool, found bool) {
	if node == nil || node.Location.Line == 0 {
		return 0, 0, false, false
	}

	lines := bytes.Split(content, []byte("\n"))
	if node.Location.Line-1 >= len(lines) {
		return 0, 0, false, false
	}

	startLine := node.Location.Line - 1
	startCol := node.Location.Column - 1

	for lineIndex := startLine; lineIndex < len(lines) && lineIndex < startLine+maxTagSearchLines; lineIndex++ {
		lineContent := lines[lineIndex]
		searchStart := 0

		if lineIndex == startLine {
			searchStart = startCol
		}

		if index := bytes.Index(lineContent[searchStart:], []byte("/>")); index != -1 {
			return safeconv.IntToUint32(lineIndex), safeconv.IntToUint32(searchStart + index), true, true
		}

		if index := bytes.Index(lineContent[searchStart:], []byte(">")); index != -1 {
			return safeconv.IntToUint32(lineIndex), safeconv.IntToUint32(searchStart + index), false, true
		}
	}

	return 0, 0, false, false
}
