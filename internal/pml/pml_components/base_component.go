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

package pml_components

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/pml/pml_domain"
)

const (
	// paddingPartsAll is the count when padding has one value for all sides.
	paddingPartsAll = 1

	// paddingPartsVertHoriz is the count for vertical and horizontal padding
	// format.
	paddingPartsVertHoriz = 2

	// paddingPartsTopSideBot is the count when padding has top, sides, and
	// bottom values.
	paddingPartsTopSideBot = 3

	// paddingPartsFull is the count for full padding (top, right, bottom, left).
	paddingPartsFull = 4

	// paddingIndexFirst is the index of the first padding value in a parsed list.
	paddingIndexFirst = 0

	// paddingIndexSecond is the index for the second padding value
	// (right/horizontal).
	paddingIndexSecond = 1

	// paddingIndexThird is the index for the third padding value (bottom).
	paddingIndexThird = 2

	// paddingIndexFourth is the index for the left padding value.
	paddingIndexFourth = 3
)

// BaseComponent provides default implementations for common methods in the
// pml_domain.Component interface. Concrete components can embed this struct to
// reduce boilerplate code.
type BaseComponent struct{}

// IsEndingTag provides the default behaviour for most components, which is to
// allow child components. Components like <pml-p> or <pml-button> will override
// this method.
//
// Returns bool which is false, indicating that this component allows children.
func (*BaseComponent) IsEndingTag() bool {
	return false
}

// DefaultAttributes provides the default implementation returning no built-in
// defaults. Individual components can override this to provide built-in
// defaults.
//
// Returns map[string]string which is an empty map for the base implementation.
func (*BaseComponent) DefaultAttributes() map[string]string {
	return make(map[string]string)
}

// GetAttributePrecedence defines the standard order for merging style sources.
// The order is critical: inline attributes on the tag have the highest
// priority and override everything else.
//
// Returns []pml_domain.AttributeSource which lists sources from lowest to
// highest priority.
func (*BaseComponent) GetAttributePrecedence() []pml_domain.AttributeSource {
	return []pml_domain.AttributeSource{
		pml_domain.SourceDefault,
		pml_domain.SourceInline,
	}
}

// transferPikoDirectives moves all dynamic p-* directives and attributes from
// the original PML node to the root of the newly generated HTML AST subtree.
// This preserves Piko's dynamic features through the
// transformation.
//
// Takes from (*ast_domain.TemplateNode) which is the source node to copy from.
// Takes to (*ast_domain.TemplateNode) which is the target node to copy to.
func transferPikoDirectives(from *ast_domain.TemplateNode, to *ast_domain.TemplateNode) {
	if from == nil || to == nil {
		return
	}

	to.DirIf = from.DirIf
	to.DirElseIf = from.DirElseIf
	to.DirElse = from.DirElse
	to.DirFor = from.DirFor
	to.DirShow = from.DirShow
	to.DirModel = from.DirModel
	to.DirRef = from.DirRef
	to.DirKey = from.DirKey
	to.DirContext = from.DirContext
	to.DirScaffold = from.DirScaffold

	to.Binds = from.Binds
	to.OnEvents = from.OnEvents
	to.CustomEvents = from.CustomEvents
	to.DynamicAttributes = from.DynamicAttributes

	from.DirIf, from.DirElseIf, from.DirElse, from.DirFor, from.DirShow = nil, nil, nil, nil, nil
	from.DirModel, from.DirRef, from.DirKey, from.DirContext, from.DirScaffold = nil, nil, nil, nil, nil
	from.Binds = nil
	from.OnEvents = nil
	from.CustomEvents = nil
	from.DynamicAttributes = nil
}

// getNodeDataAttr reads a data attribute value from a node's attribute list.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes.
// Takes name (string) which is the attribute name to find.
//
// Returns string which is the attribute value, or empty if not found.
func getNodeDataAttr(node *ast_domain.TemplateNode, name string) string {
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			return node.Attributes[i].Value
		}
	}
	return ""
}

// removeNodeDataAttr removes a data attribute from a node's attribute list.
//
// Takes node (*ast_domain.TemplateNode) which contains the attributes.
// Takes name (string) which is the attribute name to remove.
func removeNodeDataAttr(node *ast_domain.TemplateNode, name string) {
	attrs := node.Attributes[:0]
	for i := range node.Attributes {
		if node.Attributes[i].Name != name {
			attrs = append(attrs, node.Attributes[i])
		}
	}
	node.Attributes = attrs
}

// mapToStyleString converts a map of CSS properties to a style attribute
// string, sorting properties alphabetically for deterministic output.
//
// Takes styles (map[string]string) which contains CSS property names as keys
// and their values.
//
// Returns string which is the formatted style attribute in "key:value;" format,
// or an empty string if the map is empty.
func mapToStyleString(styles map[string]string) string {
	if len(styles) == 0 {
		return ""
	}

	keys := slices.Sorted(maps.Keys(styles))

	var builder strings.Builder
	for _, key := range keys {
		value := styles[key]
		if value != "" {
			_, _ = fmt.Fprintf(&builder, "%s:%s;", key, value)
		}
	}
	return builder.String()
}

// sortHTMLAttributes sorts HTML attributes alphabetically by name.
// This produces consistent output across all PML components, which prevents
// test failures from random attribute ordering.
//
// The function sorts in place and also returns the sorted slice for ease of
// use.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to
// sort.
//
// Returns []ast_domain.HTMLAttribute which is the sorted slice.
func sortHTMLAttributes(attrs []ast_domain.HTMLAttribute) []ast_domain.HTMLAttribute {
	slices.SortFunc(attrs, func(a, b ast_domain.HTMLAttribute) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return attrs
}

// mustParsePixels extracts the number from a CSS pixel value string.
//
// Takes value (string) which is the CSS value to parse, such as "10px" or "20".
//
// Returns int which is the parsed pixel value, or 0 if parsing fails or value
// is empty.
func mustParsePixels(value string) int {
	if value == "" {
		return 0
	}
	value = strings.TrimSuffix(strings.TrimSpace(value), "px")
	i, _ := strconv.Atoi(value)
	return i
}

// copyStyle copies a style value from the style manager to a destination map.
// Use it when mapping PML attributes to different CSS property names.
//
// Takes sm (*pml_domain.StyleManager) which provides the source styles.
// Takes dest (map[string]string) which receives the copied style value.
// Takes key (string) which identifies the style to copy from the manager.
// Takes destKey (...string) which optionally specifies a different key for the
// destination map.
func copyStyle(sm *pml_domain.StyleManager, dest map[string]string, key string, destKey ...string) {
	if value, ok := sm.Get(key); ok {
		dKey := key
		if len(destKey) > 0 {
			dKey = destKey[0]
		}
		dest[dKey] = value
	}
}

// mustGetStyle retrieves a style value or returns an empty string if not found.
// This makes it simpler to set HTML attributes that may or may not have values.
//
// Takes sm (*pml_domain.StyleManager) which provides access to style values.
// Takes key (string) which identifies the style to retrieve.
//
// Returns string which is the style value, or an empty string if not found.
func mustGetStyle(sm *pml_domain.StyleManager, key string) string {
	value, _ := sm.Get(key)
	return value
}

// parsePadding splits a padding string into individual directional values.
// Supports CSS shorthand: "10px" (all), "10px 20px" (vertical horizontal),
// "10px 20px 30px" (top horizontal bottom), or "10px 20px 30px 40px"
// (top right bottom left).
//
// Takes padding (string) which is the CSS-style padding shorthand to parse.
//
// Returns top (string) which is the top padding value.
// Returns right (string) which is the right padding value.
// Returns bottom (string) which is the bottom padding value.
// Returns left (string) which is the left padding value.
func parsePadding(padding string) (top, right, bottom, left string) {
	if padding == "" {
		return "", "", "", ""
	}
	parts := strings.Fields(padding)
	switch len(parts) {
	case paddingPartsAll:
		return parts[paddingIndexFirst], parts[paddingIndexFirst], parts[paddingIndexFirst], parts[paddingIndexFirst]
	case paddingPartsVertHoriz:
		return parts[paddingIndexFirst], parts[paddingIndexSecond], parts[paddingIndexFirst], parts[paddingIndexSecond]
	case paddingPartsTopSideBot:
		return parts[paddingIndexFirst], parts[paddingIndexSecond], parts[paddingIndexThird], parts[paddingIndexSecond]
	case paddingPartsFull:
		return parts[paddingIndexFirst], parts[paddingIndexSecond], parts[paddingIndexThird], parts[paddingIndexFourth]
	default:
		return "", "", "", ""
	}
}

// expandPadding takes a padding value and any direction overrides, then
// returns the final top, right, bottom, and left values.
//
// Takes sm (*pml_domain.StyleManager) which provides access to padding styles.
//
// Returns top (string) which is the final top padding value.
// Returns right (string) which is the final right padding value.
// Returns bottom (string) which is the final bottom padding value.
// Returns left (string) which is the final left padding value.
func expandPadding(sm *pml_domain.StyleManager) (top, right, bottom, left string) {
	if basePadding, ok := sm.Get(AttrPadding); ok {
		top, right, bottom, left = parsePadding(basePadding)
	}

	if value, ok := sm.Get("padding-top"); ok {
		top = value
	}
	if value, ok := sm.Get("padding-right"); ok {
		right = value
	}
	if value, ok := sm.Get("padding-bottom"); ok {
		bottom = value
	}
	if value, ok := sm.Get("padding-left"); ok {
		left = value
	}

	return top, right, bottom, left
}
