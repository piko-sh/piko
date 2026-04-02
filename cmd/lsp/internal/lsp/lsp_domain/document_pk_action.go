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
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// openParenLiteral is the open-parenthesis character used to delimit action
	// call arguments.
	openParenLiteral = "("

	// minActionSegments is the minimum number of dot-separated segments for a
	// fully-qualified action expression (e.g. "action.ns.Name").
	minActionSegments = 3
)

// actionSegmentInfo holds the parsed segments and cursor position within an
// action expression like "action.email.Contact($form)".
type actionSegmentInfo struct {
	// segments holds the dot-separated parts, e.g. ["action", "email", "Contact"].
	segments []string

	// segmentIndex is the index of the segment the cursor falls on.
	segmentIndex int

	// segmentStart is the absolute character offset in the line where the
	// current segment begins.
	segmentStart int

	// segmentEnd is the absolute character offset in the line where the
	// current segment ends.
	segmentEnd int
}

// checkActionHoverContext checks if the cursor is on an action expression
// inside a quoted attribute value. It parses the expression into segments
// and determines which segment the cursor falls on.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context if the cursor is on
// an action expression, or nil if no action context is found.
func (*document) checkActionHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	info := findActionSegmentAtCursor(line, cursor)
	if info == nil {
		return nil
	}

	kind := actionSegmentToKind(info)
	name := buildActionName(info)

	return &PKHoverContext{
		Kind:     kind,
		Name:     name,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(info.segmentStart)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(info.segmentEnd)},
		},
	}
}

// checkActionDefinitionContext checks if the cursor is on an action name for
// go-to-definition and returns the definition context if found.
//
// Only the action name segment (e.g. "Contact") produces a definition context.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKDefinitionContext which provides definition context if the cursor
// is on an action name, or nil if no action context is found.
func (*document) checkActionDefinitionContext(line string, cursor int, position protocol.Position) *PKDefinitionContext {
	info := findActionSegmentAtCursor(line, cursor)
	if info == nil {
		return nil
	}

	kind := actionSegmentToKind(info)
	if kind != PKDefActionName {
		return nil
	}

	return &PKDefinitionContext{
		Kind:     PKDefActionName,
		Name:     buildActionName(info),
		Position: position,
	}
}

// getActionHover returns hover information for an action expression.
// The content varies based on which segment the cursor is on.
//
// Takes ctx (*PKHoverContext) which provides the hover context.
//
// Returns *protocol.Hover which contains the formatted hover content.
// Returns error when hover information cannot be retrieved.
func (d *document) getActionHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	switch ctx.Kind {
	case PKDefActionRoot:
		return d.getActionRootHover(ctx)
	case PKDefActionNamespace:
		return d.getActionNamespaceHover(ctx)
	case PKDefActionName:
		return d.getActionNameHover(ctx)
	default:
		return nil, nil
	}
}

// getActionRootHover returns hover for the "action" keyword.
//
// Takes ctx (*PKHoverContext) which provides the hover context.
//
// Returns *protocol.Hover which describes the action namespace.
// Returns error which is always nil.
func (*document) getActionRootHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	content := "```\n(namespace) action\n```\n\n" +
		"Server-side action namespace. Use `action.<package>.<Name>()` to invoke actions.\n\n" +
		"Actions are type-safe server calls discovered from the `actions/` directory."

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: content,
		},
		Range: &ctx.Range,
	}, nil
}

// getActionNamespaceHover returns hover for an action namespace segment
// like "email" in action.email.Contact. Lists all actions in that namespace.
//
// Takes ctx (*PKHoverContext) which provides the hover context including the
// action name.
//
// Returns *protocol.Hover which contains the namespace summary.
// Returns error when the hover cannot be built.
func (d *document) getActionNamespaceHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	namespaceName := extractNamespaceFromActionName(ctx.Name)
	actions := d.findActionsInNamespace(namespaceName)

	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "```\n(package) action.%s\n```\n\n", namespaceName)

	if len(actions) == 0 {
		_, _ = fmt.Fprintf(&b, "No actions found in namespace `%s`.", namespaceName)
	} else {
		_, _ = fmt.Fprintf(&b, "Action package with %d action(s):\n\n", len(actions))
		for i := range actions {
			action := &actions[i]
			sig := buildActionDisplaySignature(action)
			_, _ = fmt.Fprintf(&b, "- `%s`\n", sig)
		}
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.String(),
		},
		Range: &ctx.Range,
	}, nil
}

// getActionNameHover returns rich hover for an action name like "Contact".
// Shows the full signature, input/output types, and a link to the source file.
//
// Takes ctx (*PKHoverContext) which provides the hover context.
//
// Returns *protocol.Hover which contains the action signature and details.
// Returns error when the hover cannot be built.
func (d *document) getActionNameHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	action := d.lookupAction(ctx.Name)
	if action == nil {
		return d.makeSimpleHover(ctx, fmt.Sprintf("Action `%s` (not found in manifest)", ctx.Name))
	}

	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "```go\n(action) %s\n```\n\n", action.Name)

	sig := buildActionDisplaySignature(action)
	_, _ = fmt.Fprintf(&b, "```\n%s %s\n```\n\n", action.HTTPMethod, sig)

	d.appendActionCapabilities(&b, action)

	if action.Description != "" {
		_, _ = fmt.Fprintf(&b, "%s\n\n", action.Description)
	}

	if action.FilePath != "" {
		absPath := d.resolveActionFilePath(action.FilePath)
		_, _ = fmt.Fprintf(&b, "[Open source file](%s)\n", uri.File(absPath))
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.String(),
		},
		Range: &ctx.Range,
	}, nil
}

// appendActionCapabilities appends capability badges to the builder.
//
// Takes b (*strings.Builder) which receives the badge text.
// Takes action (*annotator_dto.ActionDefinition) which provides capabilities.
func (*document) appendActionCapabilities(b *strings.Builder, action *annotator_dto.ActionDefinition) {
	caps := action.Capabilities
	var badges []string

	if caps.HasSSE {
		badges = append(badges, "SSE")
	}
	if caps.HasMiddlewares {
		badges = append(badges, "Middlewares")
	}
	if caps.HasRateLimit {
		badges = append(badges, "RateLimit")
	}
	if caps.HasResourceLimits {
		badges = append(badges, "ResourceLimits")
	}
	if caps.HasCacheConfig {
		badges = append(badges, "Cache")
	}

	if len(badges) > 0 {
		_, _ = fmt.Fprintf(b, "**Capabilities:** %s\n\n", strings.Join(badges, ", "))
	}
}

// findActionDefinition finds the Go source file location for an action name.
//
// Takes actionName (string) which is the dot-notation action name (e.g.
// "email.Contact").
//
// Returns []protocol.Location which points to the action struct in the Go
// file.
// Returns error when the lookup fails.
func (d *document) findActionDefinition(actionName string) ([]protocol.Location, error) {
	action := d.lookupAction(actionName)
	if action == nil {
		return nil, nil
	}

	absPath := d.resolveActionFilePath(action.FilePath)
	if absPath == "" {
		return nil, nil
	}

	line := action.StructLine
	if line == 0 {
		line = 1
	}

	return []protocol.Location{
		d.buildSymbolLocation(absPath, line, 1, action.StructName),
	}, nil
}

// lookupAction finds an action by name (case-insensitive) in the manifest.
//
// Takes name (string) which is the dot-notation action name.
//
// Returns *annotator_dto.ActionDefinition or nil if not found.
func (d *document) lookupAction(name string) *annotator_dto.ActionDefinition {
	if d.ProjectResult == nil || d.ProjectResult.VirtualModule == nil {
		return nil
	}

	manifest := d.ProjectResult.VirtualModule.ActionManifest
	if manifest == nil {
		return nil
	}

	if action := manifest.GetAction(name); action != nil {
		return action
	}

	for i := range manifest.Actions {
		if strings.EqualFold(manifest.Actions[i].Name, name) {
			return &manifest.Actions[i]
		}
	}

	return nil
}

// resolveActionFilePath resolves a relative action file path to an absolute
// path using the resolver's base directory.
//
// Takes relativePath (string) which is the path relative to the project root.
//
// Returns string which is the absolute file path.
func (d *document) resolveActionFilePath(relativePath string) string {
	if d.Resolver == nil {
		return relativePath
	}
	return filepath.Join(d.Resolver.GetBaseDir(), relativePath)
}

// findActionsInNamespace returns all actions whose name starts with the
// given namespace prefix.
//
// Takes namespace (string) which is the package namespace to filter by.
//
// Returns []annotator_dto.ActionDefinition containing the matching actions.
func (d *document) findActionsInNamespace(namespace string) []annotator_dto.ActionDefinition {
	if d.ProjectResult == nil || d.ProjectResult.VirtualModule == nil {
		return nil
	}

	manifest := d.ProjectResult.VirtualModule.ActionManifest
	if manifest == nil {
		return nil
	}

	prefix := namespace + "."
	var result []annotator_dto.ActionDefinition
	for i := range manifest.Actions {
		if strings.HasPrefix(manifest.Actions[i].Name, prefix) {
			result = append(result, manifest.Actions[i])
		}
	}

	return result
}

// checkActionParamKeyHoverContext checks if the cursor is on an object literal
// key inside the argument list of an action call.
//
// For example, in action.blueprint.FieldDelete({environment_id: state.X}),
// hovering over "environment_id" triggers this context.
//
// Takes line (string) which is the text content of the current line.
// Takes cursor (int) which is the character position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context if the cursor is on an
// action param key, or nil if no match.
func (*document) checkActionParamKeyHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	valueStart, valueEnd := findQuotedValueBounds(line, cursor)
	if valueStart == -1 {
		return nil
	}

	value := line[valueStart:valueEnd]
	if !strings.HasPrefix(value, "action.") {
		return nil
	}

	parenIndex := strings.Index(value, openParenLiteral)
	if parenIndex == -1 {
		return nil
	}

	absParenStart := valueStart + parenIndex
	if cursor <= absParenStart || cursor >= valueEnd {
		return nil
	}

	keyName, keyStart, keyEnd := findObjectKeyAtCursor(line, cursor)
	if keyName == "" {
		return nil
	}

	actionExpr := value[:parenIndex]
	segments := strings.Split(actionExpr, ".")
	if len(segments) < minActionSegments {
		return nil
	}
	actionName := strings.Join(segments[1:], ".")

	return &PKHoverContext{
		Kind:     PKDefActionParamKey,
		Name:     actionName + ":" + keyName,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(keyStart)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(keyEnd)},
		},
	}
}

// getActionParamKeyHover returns hover information for an action call parameter
// key. Shows the field's Go type, TypeScript type, validation rules, and
// description.
//
// Takes ctx (*PKHoverContext) which provides the hover context with the
// combined action name and key name.
//
// Returns *protocol.Hover which contains the formatted field information.
// Returns error when the hover cannot be built.
func (d *document) getActionParamKeyHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	actionName, keyName := splitActionParamKey(ctx.Name)

	action := d.lookupAction(actionName)
	if action == nil {
		return nil, nil
	}

	field := findActionFieldByJSONName(action.CallParams, keyName)
	if field == nil {
		return nil, nil
	}

	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "```go\n(field) %s: %s\n```\n\n", field.JSONName, field.GoType)

	if field.TSType != "" && field.TSType != field.GoType {
		_, _ = fmt.Fprintf(&b, "**TypeScript:** `%s`\n\n", field.TSType)
	}

	if field.Name != field.JSONName {
		_, _ = fmt.Fprintf(&b, "**Go field:** `%s`\n\n", field.Name)
	}

	if field.Validation != "" {
		_, _ = fmt.Fprintf(&b, "**Validation:** `%s`\n\n", field.Validation)
	}

	if field.Optional {
		b.WriteString("*Optional*\n\n")
	}

	if field.Description != "" {
		_, _ = fmt.Fprintf(&b, "%s\n", field.Description)
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: b.String(),
		},
		Range: &ctx.Range,
	}, nil
}

// findActionSegmentAtCursor locates an action expression in a quoted attribute
// value and determines which segment the cursor is on.
//
// Takes line (string) which is the text to search.
// Takes cursor (int) which is the cursor position within the line.
//
// Returns *actionSegmentInfo describing the segment, or nil if the cursor is
// not on an action expression.
func findActionSegmentAtCursor(line string, cursor int) *actionSegmentInfo {
	valueStart, valueEnd := findQuotedValueBounds(line, cursor)
	if valueStart == -1 {
		return nil
	}

	value := line[valueStart:valueEnd]
	if !strings.HasPrefix(value, "action.") {
		return nil
	}

	actionExpr := value
	if parenIndex := strings.Index(actionExpr, openParenLiteral); parenIndex != -1 {
		actionExpr = actionExpr[:parenIndex]
	}

	segments := strings.Split(actionExpr, ".")

	for len(segments) > 0 && segments[len(segments)-1] == "" {
		segments = segments[:len(segments)-1]
	}

	if len(segments) < 2 {
		return nil
	}

	offset := valueStart
	for i, seg := range segments {
		segStart := offset
		segEnd := segStart + len(seg)

		if cursor >= segStart && cursor < segEnd {
			return &actionSegmentInfo{
				segments:     segments,
				segmentIndex: i,
				segmentStart: segStart,
				segmentEnd:   segEnd,
			}
		}

		offset = segEnd + 1
	}

	lastIndex := len(segments) - 1
	lastStart := valueStart
	for i := range lastIndex {
		lastStart += len(segments[i]) + 1
	}
	lastEnd := lastStart + len(segments[lastIndex])

	if cursor == lastEnd {
		return &actionSegmentInfo{
			segments:     segments,
			segmentIndex: lastIndex,
			segmentStart: lastStart,
			segmentEnd:   lastEnd,
		}
	}

	return nil
}

// findQuotedValueBounds finds the start and end positions of the quoted
// attribute value that the cursor is within.
//
// Takes line (string) which is the text to search.
// Takes cursor (int) which is the cursor position within the line.
//
// Returns int which is the start position of the value, excluding the quote.
// Returns int which is the end position of the value, excluding the quote.
// Both values are -1 if the cursor is not inside a quoted value.
func findQuotedValueBounds(line string, cursor int) (start int, end int) {
	if cursor > len(line) {
		return -1, -1
	}

	quoteChar := byte(0)
	valueStart := -1

	for i := cursor - 1; i >= 0; i-- {
		if line[i] == '"' || line[i] == '\'' {
			quoteChar = line[i]
			valueStart = i + 1
			break
		}
	}

	if valueStart == -1 {
		return -1, -1
	}

	for i := valueStart; i < len(line); i++ {
		if line[i] == quoteChar {
			if cursor >= valueStart && cursor <= i {
				return valueStart, i
			}
			return -1, -1
		}
	}

	return -1, -1
}

// actionSegmentToKind maps a segment index to the appropriate PKDefinitionKind.
//
// Takes info (*actionSegmentInfo) which describes the current segment.
//
// Returns PKDefinitionKind for the segment position.
func actionSegmentToKind(info *actionSegmentInfo) PKDefinitionKind {
	switch {
	case info.segmentIndex == 0:
		return PKDefActionRoot
	case info.segmentIndex == len(info.segments)-1 && len(info.segments) >= minActionSegments:
		return PKDefActionName
	default:
		return PKDefActionNamespace
	}
}

// buildActionName constructs the full dot-notation action name from segments.
// For "action.email.Contact", this returns "email.Contact".
//
// Takes info (*actionSegmentInfo) which contains the parsed segments.
//
// Returns string which is the action name (without the "action." prefix).
func buildActionName(info *actionSegmentInfo) string {
	if len(info.segments) < 2 {
		return ""
	}
	return strings.Join(info.segments[1:], ".")
}

// extractNamespaceFromActionName extracts the namespace portion from a
// dot-notation action name. For "email.Contact" it returns "email", and for
// "email" it returns "email" unchanged.
//
// Takes actionName (string) which is the full action name in dot notation.
//
// Returns string which is the namespace portion before the first dot.
func extractNamespaceFromActionName(actionName string) string {
	if namespace, _, found := strings.Cut(actionName, "."); found {
		return namespace
	}
	return actionName
}

// buildActionDisplaySignature builds a human-readable signature for an action
// using the dot-notation name used in templates.
//
// Takes action (*annotator_dto.ActionDefinition) which provides the action
// definition.
//
// Returns string which is the formatted signature.
func buildActionDisplaySignature(action *annotator_dto.ActionDefinition) string {
	params := buildActionParamList(action.CallParams, true)

	var outputType string
	if action.OutputType != nil {
		outputType = action.OutputType.Name
		if outputType == "" {
			outputType = action.OutputType.TSType
		}
	}

	if outputType != "" {
		return action.Name + openParenLiteral + params + "): ActionBuilder<" + outputType + ">"
	}
	return action.Name + openParenLiteral + params + "): ActionBuilder<void>"
}

// findObjectKeyAtCursor finds the identifier at the cursor position and
// verifies it is an object literal key (followed by a colon).
//
// Takes line (string) which is the source text to scan.
// Takes cursor (int) which is the character position to check.
//
// Returns key (string) which is the key name, or empty if no key is found.
// Returns start (int) which is the start character offset of the key.
// Returns end (int) which is the end character offset of the key.
func findObjectKeyAtCursor(line string, cursor int) (key string, start int, end int) {
	if cursor >= len(line) {
		return "", 0, 0
	}

	if !isIdentChar(line[cursor]) {
		if cursor <= 0 || !isIdentChar(line[cursor-1]) {
			return "", 0, 0
		}
		cursor--
	}

	start = cursor
	for start > 0 && isIdentChar(line[start-1]) {
		start--
	}

	end = cursor
	for end < len(line) && isIdentChar(line[end]) {
		end++
	}

	if start == end {
		return "", 0, 0
	}

	afterEnd := end
	for afterEnd < len(line) && (line[afterEnd] == ' ' || line[afterEnd] == '\t') {
		afterEnd++
	}
	if afterEnd >= len(line) || line[afterEnd] != ':' {
		return "", 0, 0
	}

	return line[start:end], start, end
}

// splitActionParamKey splits a combined "actionName:keyName" string into its
// two parts.
//
// Takes combined (string) which holds the action name and key name separated
// by a colon.
//
// Returns string which is the action name portion.
// Returns string which is the key name portion.
func splitActionParamKey(combined string) (actionName string, keyName string) {
	actionName, keyName, _ = strings.Cut(combined, ":")
	return actionName, keyName
}

// findActionFieldByJSONName searches the action's call parameters for a field
// whose JSON name matches the given key.
//
// Takes params ([]annotator_dto.ActionTypeInfo) which holds the action's
// parameter type information.
// Takes jsonName (string) which is the JSON field name to look up.
//
// Returns *annotator_dto.ActionFieldInfo for the matching field, or nil if
// not found.
func findActionFieldByJSONName(params []annotator_dto.ActionTypeInfo, jsonName string) *annotator_dto.ActionFieldInfo {
	for i := range params {
		for j := range params[i].Fields {
			if params[i].Fields[j].JSONName == jsonName {
				return &params[i].Fields[j]
			}
		}
	}
	return nil
}

// buildActionParamList builds a comma-separated list of parameter type names.
//
// When preferName is true, it uses the Go type name, falling back to TSType.
// When preferName is false, it uses TSType, falling back to Name.
//
// Takes params ([]annotator_dto.ActionTypeInfo) which are the call parameters.
// Takes preferName (bool) which controls whether to prefer Name over TSType.
//
// Returns string which is the comma-separated parameter types.
func buildActionParamList(params []annotator_dto.ActionTypeInfo, preferName bool) string {
	if len(params) == 0 {
		return ""
	}

	parts := make([]string, 0, len(params))
	for i := range params {
		var typeName string
		if preferName {
			typeName = params[i].Name
			if typeName == "" {
				typeName = params[i].TSType
			}
		} else {
			typeName = params[i].TSType
			if typeName == "" {
				typeName = params[i].Name
			}
		}
		parts = append(parts, typeName)
	}
	return strings.Join(parts, ", ")
}
