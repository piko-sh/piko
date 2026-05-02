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

package querier_domain

import (
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// keywordName is the keyword argument key that supplies a directive's name slot. The
	// same key fills both the positional name slot and the explicit `name:` argument, so
	// concentrating it here keeps the routing consistent across the apply helpers.
	keywordName = "name"
)

// applyTopCall populates the DirectiveBlock from a parsed `piko(...)` call.
//
// Positional arguments at index 0 and 1 fill the Name and Command slots respectively,
// mirroring the spec's Positionals order. Any keyword argument whose key matches a
// positional name fills the same slot, and other keyword arguments become Metadata
// entries.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the populated directive.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
func applyTopCall(result *querier_dto.DirectiveBlock, call *callArgs) {
	span := mergeSpan(call.openSpan, call.closeSpan)

	applyTopCallPositionals(result, call)

	for index := range call.keywordArguments {
		applyTopCallKeyword(result, &call.keywordArguments[index])
	}
	if result.Name != nil && result.Name.Span.Line == 0 {
		result.Name.Span = span
	}
}

// applyTopCallPositionals fills the Name (slot 0) and Command (slot 1) fields from
// positional arguments, mirroring the spec ordering. A non-parseable command leaves the
// Command slot empty for the keyword loop to retry later if a `command:` keyword argument
// is supplied.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the Name and Command slots.
// Takes call (*callArgs) which holds the parsed positional arguments.
func applyTopCallPositionals(result *querier_dto.DirectiveBlock, call *callArgs) {
	if len(call.positionals) > 0 {
		positional := &call.positionals[0]
		result.Name = &querier_dto.NameDirective{
			Value:     positional.value.raw,
			Span:      positional.span,
			KeySpan:   positional.span,
			ValueSpan: positional.span,
		}
	}
	if len(call.positionals) >= 2 {
		positional := &call.positionals[1]
		command, parseErr := parseCommandValue(positional.value.raw)
		if parseErr == nil {
			result.Command = &querier_dto.CommandDirective{
				Value:     positional.value.raw,
				Command:   command,
				Span:      positional.span,
				KeySpan:   positional.span,
				ValueSpan: positional.span,
			}
		}
	}
}

// applyTopCallKeyword routes one keyword argument from the top-level piko(...) call into
// the matching Directive field, falling back to a metadata entry for keys that do not
// correspond to a known slot.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the routed field or metadata.
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword argument to route.
func applyTopCallKeyword(result *querier_dto.DirectiveBlock, keywordArgument *parsedKeywordArgument) {
	switch keywordArgument.key {
	case keywordName:
		if result.Name != nil {
			return
		}
		result.Name = &querier_dto.NameDirective{
			Value:     keywordArgument.value.raw,
			Span:      keywordArgument.span,
			KeySpan:   keywordArgument.keySpan,
			ValueSpan: keywordArgument.valueSpan,
		}
	case "command":
		if result.Command != nil {
			return
		}
		command, parseErr := parseCommandValue(keywordArgument.value.raw)
		if parseErr != nil {
			return
		}
		result.Command = &querier_dto.CommandDirective{
			Value:     keywordArgument.value.raw,
			Command:   command,
			Span:      keywordArgument.span,
			KeySpan:   keywordArgument.keySpan,
			ValueSpan: keywordArgument.valueSpan,
		}
	default:
		result.Metadata = append(result.Metadata, &querier_dto.MetadataDirective{
			Directive: keywordArgument.key,
			Value:     keywordArgument.value.raw,
			Span:      keywordArgument.span,
			KeySpan:   keywordArgument.keySpan,
			ValueSpan: keywordArgument.valueSpan,
		})
	}
}

// applyHeaderCall populates the DirectiveBlock from a parsed header directive call.
//
// The call dispatches to piko.embed, piko.column, or piko.sortable. The table/name
// positional may be passed by name (`table:` / `name:`), in which case it is routed
// through the keyword argument loop into the same slot.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the populated directive.
// Takes spec (*querier_dto.DirectiveSpec) which selects the header directive to apply.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
// Takes errorBuilder (querier_dto.ErrorBuilder) which builds source-mapped diagnostics.
// Takes diagnostics (*[]querier_dto.SourceError) which accumulates emitted diagnostics.
func applyHeaderCall(result *querier_dto.DirectiveBlock, spec *querier_dto.DirectiveSpec, call *callArgs, errorBuilder querier_dto.ErrorBuilder, diagnostics *[]querier_dto.SourceError) {
	switch spec.Name {
	case "piko.embed":
		applyEmbedHeaderCall(result, call)
	case "piko.column":
		applyColumnHeaderCall(result, call, errorBuilder, diagnostics)
	case "piko.sortable":
		applySortableHeaderCall(result, spec, call)
	}
}

// applySortableHeaderCall builds a sortable parameter directive from a standalone
// piko.sortable(name, [columns]) call. Sortable does not bind a placeholder, so it
// carries no anchor or number here; ResolveParameters assigns it a number after the bound
// parameters so it appears last in the generated params struct.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the sortable directive.
// Takes spec (*querier_dto.DirectiveSpec) which supplies the directive name and slots.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
func applySortableHeaderCall(result *querier_dto.DirectiveBlock, spec *querier_dto.DirectiveSpec, call *callArgs) {
	name, nameSpan, ok := resolveDirectiveName(call)
	if !ok {
		return
	}
	directive := &querier_dto.ParameterDirective{
		Name:          name,
		DirectiveName: strings.TrimPrefix(spec.Name, "piko."),
		Kind:          querier_dto.ParameterDirectiveSortable,
		Span:          mergeSpan(call.openSpan, call.closeSpan),
		KindSpan:      mergeSpan(call.openSpan, call.closeSpan),
		NameSpan:      nameSpan,
	}
	if len(call.positionals) >= 2 && len(spec.Positionals) >= 2 {
		applyParameterValue(directive, spec.Positionals[1].Name, &call.positionals[1].value)
	}
	for index := range call.keywordArguments {
		keywordArgument := &call.keywordArguments[index]
		if keywordArgument.key == keywordName {
			continue
		}
		applyParameterValue(directive, keywordArgument.key, &keywordArgument.value)
	}
	result.Parameters = append(result.Parameters, directive)
}

// resolveDirectiveName returns the directive's parameter name from positional slot 0 when
// present, otherwise from a 'name:' keyword argument, so the apply phase honours both the
// positional and keyword spellings the validator accepts. ok is false when neither
// supplies a name.
//
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
//
// Returns the name, its source span, and whether a name was found.
func resolveDirectiveName(call *callArgs) (string, querier_dto.TextSpan, bool) {
	if len(call.positionals) > 0 {
		return call.positionals[0].value.raw, call.positionals[0].span, true
	}
	for index := range call.keywordArguments {
		if call.keywordArguments[index].key == keywordName {
			return call.keywordArguments[index].value.raw, call.keywordArguments[index].valueSpan, true
		}
	}
	return "", querier_dto.TextSpan{}, false
}

// applyEmbedHeaderCall populates an EmbedDirective from a parsed piko.embed(...) call.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the embed directive.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
func applyEmbedHeaderCall(result *querier_dto.DirectiveBlock, call *callArgs) {
	embed := &querier_dto.EmbedDirective{
		Span: mergeSpan(call.openSpan, call.closeSpan),
	}
	if len(call.positionals) > 0 {
		embed.Table = call.positionals[0].value.raw
		embed.TableSpan = call.positionals[0].span
	}
	for index := range call.keywordArguments {
		keywordArgument := &call.keywordArguments[index]
		switch keywordArgument.key {
		case "table":
			if embed.Table == "" {
				embed.Table = keywordArgument.value.raw
				embed.TableSpan = keywordArgument.valueSpan
			}
		case "from":
			embed.From = keywordArgument.value.raw
		case "as":
			embed.As = keywordArgument.value.raw
		}
	}
	result.Embeds = append(result.Embeds, embed)
	result.Metadata = append(result.Metadata, &querier_dto.MetadataDirective{
		Directive: "embed",
		Value:     embed.Table,
		Span:      embed.Span,
	})
}

// applyColumnHeaderCall populates a ColumnOverride from a parsed piko.column(...) call.
// Enforces the mutual exclusion between the `type:` and `go_type:` keyword arguments at
// parse time so downstream consumers see at most one of them set.
//
// Takes result (*querier_dto.DirectiveBlock) which receives the column override.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
// Takes errorBuilder (querier_dto.ErrorBuilder) which builds source-mapped diagnostics.
// Takes diagnostics (*[]querier_dto.SourceError) which accumulates emitted diagnostics.
func applyColumnHeaderCall(result *querier_dto.DirectiveBlock, call *callArgs, errorBuilder querier_dto.ErrorBuilder, diagnostics *[]querier_dto.SourceError) {
	override := &querier_dto.ColumnOverride{
		Span: mergeSpan(call.openSpan, call.closeSpan),
	}
	if len(call.positionals) > 0 {
		override.Name = call.positionals[0].value.raw
		override.NameSpan = call.positionals[0].span
	}

	var sawType, sawGoType bool
	var goTypeSpan querier_dto.TextSpan
	for index := range call.keywordArguments {
		keywordArgument := &call.keywordArguments[index]
		switch keywordArgument.key {
		case keywordName:
			if override.Name == "" {
				override.Name = keywordArgument.value.raw
				override.NameSpan = keywordArgument.valueSpan
			}
		case "type":
			override.SQLType = keywordArgument.value.raw
			override.TypeSpan = keywordArgument.valueSpan
			sawType = true
		case "go_type":
			override.GoType = keywordArgument.value.raw
			override.GoTypeSpan = keywordArgument.valueSpan
			goTypeSpan = keywordArgument.span
			sawGoType = true
		case "nullable":
			override.Nullable = new(strings.EqualFold(keywordArgument.value.raw, literalTrue))
		}
	}

	if sawType && sawGoType {
		*diagnostics = append(*diagnostics, errorBuilder.MutuallyExclusiveKeywordArgument(goTypeSpan, "piko.column", "type", "go_type"))
	}

	result.ColumnOverrides = append(result.ColumnOverrides, override)
}

// buildParameterDirective converts a parsed parameter-binding call into a
// ParameterDirective AST node.
//
// Positional 0 is always the parameter name. Positional 1, when present, is the second
// positional declared in the spec; for piko.sortable this carries the `columns` list. Any
// keyword argument whose key matches a positional name fills the same slot as its
// positional counterpart.
//
// Takes spec (*querier_dto.DirectiveSpec) which supplies the directive name and slots.
// Takes call (*callArgs) which holds the parsed positionals and keyword arguments.
// Takes anchorToken (token) which anchors the directive span to the placeholder.
// Takes kindSpan (querier_dto.TextSpan) which spans the directive kind keyword.
// Takes isNamed (bool) which records whether the bound placeholder is named.
// Takes number (int) which is the parameter number assigned to the directive.
//
// Returns *querier_dto.ParameterDirective which is the built directive, or nil when no
// name was supplied.
func buildParameterDirective(spec *querier_dto.DirectiveSpec, call *callArgs, anchorToken token, kindSpan querier_dto.TextSpan, isNamed bool, number int) *querier_dto.ParameterDirective {
	name, nameSpan, ok := resolveDirectiveName(call)
	if !ok {
		return nil
	}
	directive := &querier_dto.ParameterDirective{
		Name:          name,
		DirectiveName: strings.TrimPrefix(spec.Name, "piko."),
		Number:        number,
		IsNamed:       isNamed,
		Kind:          spec.ParamKind,
		Span:          mergeSpan(anchorToken.span, call.closeSpan),
		NumberSpan:    anchorToken.span,
		KindSpan:      kindSpan,
		NameSpan:      nameSpan,
	}
	if len(call.positionals) >= 2 && len(spec.Positionals) >= 2 {
		applyParameterValue(directive, spec.Positionals[1].Name, &call.positionals[1].value)
	}
	for index := range call.keywordArguments {
		keywordArgument := &call.keywordArguments[index]
		if keywordArgument.key == keywordName {
			continue
		}
		applyParameterValue(directive, keywordArgument.key, &keywordArgument.value)
	}
	return directive
}

// applyParameterValue routes a value (from positional slot 1 or from any keyword
// argument) into the corresponding ParameterDirective field.
//
// Takes directive (*querier_dto.ParameterDirective) which receives the routed value.
// Takes key (string) which selects the directive field to populate.
// Takes value (*parsedValue) which carries the raw value and any list elements.
func applyParameterValue(directive *querier_dto.ParameterDirective, key string, value *parsedValue) {
	switch key {
	case "type":
		directive.TypeHint = new(value.raw)
	case "nullable":
		directive.Nullable = new(strings.EqualFold(value.raw, literalTrue))
	case "default":
		if parsed, parseErr := strconv.ParseInt(value.raw, 10, 64); parseErr == nil {
			directive.DefaultVal = new(safeconv.Int64ToInt(parsed))
		}
	case "max":
		if parsed, parseErr := strconv.ParseInt(value.raw, 10, 64); parseErr == nil {
			directive.MaxVal = new(safeconv.Int64ToInt(parsed))
		}
	case "optional":
		directive.IsOptional = strings.EqualFold(value.raw, literalTrue)
	case "kind":
		directive.IsSlice = strings.EqualFold(value.raw, "slice")
	case "columns":
		directive.Columns = append(directive.Columns, value.asList...)
	}
}
