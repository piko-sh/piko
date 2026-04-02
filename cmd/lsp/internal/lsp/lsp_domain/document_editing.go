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
	"go.lsp.dev/protocol"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/wdk/safeconv"
)

// GetLinkedEditingRanges finds HTML tag pairs for synchronised editing.
// When a user edits an opening tag name, the closing tag name should update
// automatically.
//
// Takes position (protocol.Position) which specifies the cursor location in the
// document.
//
// Returns *protocol.LinkedEditingRanges which contains the ranges of the
// opening and closing tag names that should be edited together.
// Returns error when the ranges cannot be determined.
func (d *document) GetLinkedEditingRanges(position protocol.Position) (*protocol.LinkedEditingRanges, error) {
	if d.AnnotationResult == nil || d.AnnotationResult.AnnotatedAST == nil {
		return nil, nil
	}

	targetNode := findNodeAtPosition(d.AnnotationResult.AnnotatedAST, position, d.URI.Filename())
	if targetNode == nil || targetNode.TagName == "" {
		return nil, nil
	}

	if d.isComponentTag(targetNode) || d.isSpecialTag(targetNode.TagName) {
		return nil, nil
	}

	if targetNode.ClosingTagRange.Start.IsSynthetic() {
		return nil, nil
	}

	ranges := make([]protocol.Range, 0, 2)

	openingTagLine := safeconv.IntToUint32(targetNode.Location.Line - 1)
	openingTagStartChar := safeconv.IntToUint32(targetNode.Location.Column)
	openingTagEndChar := openingTagStartChar + safeconv.IntToUint32(len(targetNode.TagName))

	ranges = append(ranges, protocol.Range{
		Start: protocol.Position{Line: openingTagLine, Character: openingTagStartChar},
		End:   protocol.Position{Line: openingTagLine, Character: openingTagEndChar},
	})

	closingTagLine := safeconv.IntToUint32(targetNode.ClosingTagRange.Start.Line - 1)
	closingTagStartChar := safeconv.IntToUint32(targetNode.ClosingTagRange.Start.Column - 1 + 2)
	closingTagEndChar := closingTagStartChar + safeconv.IntToUint32(len(targetNode.TagName))

	ranges = append(ranges, protocol.Range{
		Start: protocol.Position{Line: closingTagLine, Character: closingTagStartChar},
		End:   protocol.Position{Line: closingTagLine, Character: closingTagEndChar},
	})

	return &protocol.LinkedEditingRanges{
		Ranges: ranges,
	}, nil
}

// isComponentTag checks if a node represents a component invocation
// (partial or custom element).
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has an "is" attribute or partial
// info in annotations.
func (*document) isComponentTag(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}

	for i := range node.Attributes {
		if node.Attributes[i].Name == "is" {
			return true
		}
	}

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		return true
	}

	return false
}

// isSpecialTag checks if a tag name is a special Piko tag that should not
// have linked editing.
//
// Takes tagName (string) which is the tag name to check.
//
// Returns bool which is true if the tag is a special Piko tag.
func (*document) isSpecialTag(tagName string) bool {
	specialTags := map[string]bool{
		"template": true,
		"script":   true,
		"style":    true,
		"slot":     true,
	}
	return specialTags[tagName]
}
