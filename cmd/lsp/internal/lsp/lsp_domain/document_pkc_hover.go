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
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/wdk/safeconv"
)

// checkPKCStatePropertyHoverContext detects state.propName references for hover
// in PKC template content.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character offset of the cursor.
// Takes position (protocol.Position) which is the cursor position.
//
// Returns *PKHoverContext for a state property, or nil if not matched.
func (*document) checkPKCStatePropertyHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	name, dotIndex, nameEnd, ok := scanStatePropertyAccess(line, cursor)
	if !ok {
		return nil
	}

	return &PKHoverContext{
		Name: name,
		Kind: PKDefPKCStateProperty,
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      position.Line,
				Character: safeconv.IntToUint32(dotIndex),
			},
			End: protocol.Position{
				Line:      position.Line,
				Character: safeconv.IntToUint32(nameEnd),
			},
		},
		Position: position,
	}
}

// getPKCStatePropertyHover returns hover info for a state property in a PKC
// file, showing the type, nullability, and default value.
//
// Takes ctx (*PKHoverContext) which identifies the property.
//
// Returns *protocol.Hover which contains the property's type information.
// Returns error which is always nil.
func (d *document) getPKCStatePropertyHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("state.%s", ctx.Name))
	}

	prop, exists := meta.StateProperties[ctx.Name]
	if !exists {
		return d.makeSimpleHover(ctx, fmt.Sprintf("state.%s (unknown)", ctx.Name))
	}

	typeString := getPKCTypeString(prop)
	if prop.IsNullable {
		typeString += " | null"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("**state.%s** `%s`", prop.Name, typeString))

	if prop.InitialValue != "" {
		parts = append(parts, fmt.Sprintf("Default: `%s`", prop.InitialValue))
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: strings.Join(parts, "\n\n"),
		},
		Range: &ctx.Range,
	}, nil
}

// getPKCHandlerHover returns hover info for a function in a PKC file using
// cached metadata, showing the function signature with parameter names.
//
// Takes ctx (*PKHoverContext) which identifies the function.
//
// Returns *protocol.Hover which contains the function signature.
// Returns error which is always nil.
func (d *document) getPKCHandlerHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	meta := d.getPKCMetadata()
	if meta == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Event handler `%s`", ctx.Name))
	}

	function, exists := meta.Functions[ctx.Name]
	if !exists {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Event handler `%s`", ctx.Name))
	}

	params := strings.Join(function.ParamNames, ", ")
	signature := fmt.Sprintf("function %s(%s)", function.Name, params)

	return d.makeCodeHover(ctx, signature, "typescript")
}
