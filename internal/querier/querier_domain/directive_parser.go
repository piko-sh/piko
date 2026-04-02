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
	"fmt"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// pikoPrefix holds the common prefix for all piko directive keywords.
const pikoPrefix = "piko."

// directiveParser parses directive comment blocks into
// structured directive representations.
type directiveParser struct {
	// prefixLookup maps parameter prefix bytes to their directive definitions.
	prefixLookup map[byte]querier_dto.DirectiveParameterPrefix

	// commentStyle holds the comment style used by the SQL engine.
	commentStyle querier_dto.CommentStyle
}

// newDirectiveParser creates a directive parser configured
// with the given parameter prefixes and comment style.
//
// Takes prefixes ([]querier_dto.DirectiveParameterPrefix)
// which holds the engine-specific parameter prefix
// definitions.
//
// Takes commentStyle (querier_dto.CommentStyle) which
// specifies the SQL comment syntax used by the engine.
//
// Returns *directiveParser which holds the initialised
// parser.
func newDirectiveParser(
	prefixes []querier_dto.DirectiveParameterPrefix,
	commentStyle querier_dto.CommentStyle,
) *directiveParser {
	lookup := make(map[byte]querier_dto.DirectiveParameterPrefix, len(prefixes))
	for _, prefix := range prefixes {
		lookup[prefix.Prefix] = prefix
	}
	return &directiveParser{
		commentStyle: commentStyle,
		prefixLookup: lookup,
	}
}

// Parse parses a query block's directive comments into
// a structured DirectiveBlock.
//
// Takes block (queryBlock) which holds the raw SQL text
// and line information to parse.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Returns *querier_dto.DirectiveBlock which holds the
// parsed directives including name, command, parameters,
// and metadata.
//
// Returns []querier_dto.SourceError which holds any syntax
// errors or validation warnings found during parsing.
func (p *directiveParser) Parse(
	block queryBlock,
	filename string,
) (*querier_dto.DirectiveBlock, []querier_dto.SourceError) {
	result := &querier_dto.DirectiveBlock{}
	var diagnostics []querier_dto.SourceError

	lines := strings.Split(block.sql, "\n")
	firstLine := -1
	lastLine := -1

	for lineOffset, line := range lines {
		scanner := newDirectiveLineScanner(line, block.startLine+lineOffset)

		scanner.skipWhitespace()
		if !scanner.matchString(p.commentStyle.LinePrefix) {
			break
		}
		scanner.skipWhitespace()

		if firstLine == -1 {
			firstLine = scanner.lineNumber
		}
		lastLine = scanner.lineNumber

		if scanner.atEnd() {
			continue
		}

		if prefix, isParameterPrefix := p.prefixLookup[scanner.current()]; isParameterPrefix {
			p.handleParameterLine(scanner, result, &diagnostics, filename, prefix)
			continue
		}

		if scanner.lookingAt(pikoPrefix) {
			parsePikoDirective(scanner, result, &diagnostics, filename)
			continue
		}
	}

	setBlockSpan(result, firstLine, lastLine, lines, block.startLine)
	validateRequiredDirectives(result, &diagnostics, filename, block.startLine)

	return result, diagnostics
}

// handleParameterLine parses a single parameter directive
// line and appends the result or error.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the parameter prefix.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to append the parameter to.
//
// Takes diagnostics (*[]querier_dto.SourceError) which
// holds the diagnostics slice to append errors to.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes prefix (querier_dto.DirectiveParameterPrefix)
// which holds the matched prefix definition.
func (*directiveParser) handleParameterLine(
	scanner *directiveLineScanner,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	filename string,
	prefix querier_dto.DirectiveParameterPrefix,
) {
	var directive *querier_dto.ParameterDirective
	var parseError *querier_dto.SourceError

	if prefix.IsNamed {
		directive, parseError = parseNamedParameterDirective(scanner, filename, len(result.Parameters)+1)
	} else {
		directive, parseError = parseNumberedParameterDirective(scanner, filename, prefix.Prefix)
	}

	if parseError != nil {
		*diagnostics = append(*diagnostics, *parseError)
	} else {
		result.Parameters = append(result.Parameters, directive)
	}
}

// setBlockSpan sets the text span on the directive block
// result from the first and last directive lines.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to update.
//
// Takes firstLine (int) which specifies the first
// directive line number, or -1 if no directives were
// found.
//
// Takes lastLine (int) which specifies the last directive
// line number.
//
// Takes lines ([]string) which holds the split lines of
// the block for computing end column.
//
// Takes startLine (int) which specifies the starting line
// offset of the block.
func setBlockSpan(
	result *querier_dto.DirectiveBlock,
	firstLine int,
	lastLine int,
	lines []string,
	startLine int,
) {
	if firstLine == -1 {
		return
	}
	result.Span = querier_dto.TextSpan{
		Line:      firstLine,
		Column:    1,
		EndLine:   lastLine,
		EndColumn: len(lines[lastLine-startLine]) + 1,
	}
}

// validateRequiredDirectives checks that the mandatory
// piko.name and piko.command directives are present.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the parsed directive block to validate.
//
// Takes diagnostics (*[]querier_dto.SourceError) which
// holds the diagnostics slice to append errors to.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes startLine (int) which specifies the line number
// for error positioning.
func validateRequiredDirectives(
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	filename string,
	startLine int,
) {
	if result.Name == nil {
		*diagnostics = append(*diagnostics, querier_dto.SourceError{
			Filename: filename,
			Line:     startLine,
			Column:   1,
			Message:  "missing piko.name directive",
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeMissingDirective,
		})
	}

	if result.Command == nil {
		*diagnostics = append(*diagnostics, querier_dto.SourceError{
			Filename: filename,
			Line:     startLine,
			Column:   1,
			Message:  "missing piko.command directive",
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeMissingDirective,
		})
	}
}

// parsePikoDirective parses a piko-prefixed directive and
// dispatches to the appropriate sub-parser.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the piko prefix.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to populate.
//
// Takes diagnostics (*[]querier_dto.SourceError) which
// holds the diagnostics slice to append errors to.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
func parsePikoDirective(
	scanner *directiveLineScanner,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	filename string,
) {
	keyStart := scanner.column()
	scanner.matchString(pikoPrefix)
	directive, _ := scanner.readWord()
	keySpan := scanner.spanFrom(keyStart)

	switch {
	case scanner.current() == '(':
		parseParenthesisedDirective(scanner, result, directive, keySpan)
	case scanner.matchByte(':'):
		parseColonDirective(scanner, result, diagnostics, filename, directive, keySpan)
	default:
		parseBareDirective(scanner, result, directive, keySpan)
	}
}

// parseParenthesisedDirective parses a directive in
// piko.key(value) form and appends it as metadata.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the opening parenthesis.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to append metadata to.
//
// Takes directive (string) which specifies the directive
// keyword name.
//
// Takes keySpan (querier_dto.TextSpan) which holds the
// text span of the directive key.
func parseParenthesisedDirective(
	scanner *directiveLineScanner,
	result *querier_dto.DirectiveBlock,
	directive string,
	keySpan querier_dto.TextSpan,
) {
	scanner.advance()
	valueStart := scanner.column()
	value, _ := scanner.readUntilByte(')')
	value = strings.TrimSpace(value)
	valueSpan := scanner.spanFrom(valueStart)
	scanner.matchByte(')')

	result.Metadata = append(result.Metadata, &querier_dto.MetadataDirective{
		Directive: directive,
		Value:     value,
		Span:      directiveLineSpan(scanner),
		KeySpan:   keySpan,
		ValueSpan: valueSpan,
	})
}

// parseBareDirective parses a directive with no value
// syntax and records it as a boolean true metadata entry.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned after the directive keyword.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to append metadata to.
//
// Takes directive (string) which specifies the directive
// keyword name.
//
// Takes keySpan (querier_dto.TextSpan) which holds the
// text span of the directive key.
func parseBareDirective(
	scanner *directiveLineScanner,
	result *querier_dto.DirectiveBlock,
	directive string,
	keySpan querier_dto.TextSpan,
) {
	scanner.skipWhitespace()
	if !scanner.atEnd() {
		return
	}

	result.Metadata = append(result.Metadata, &querier_dto.MetadataDirective{
		Directive: directive,
		Value:     "true",
		Span:      directiveLineSpan(scanner),
		KeySpan:   keySpan,
		ValueSpan: keySpan,
	})
}

// parseColonDirective parses a directive in
// piko.key: value form and handles name, command, or
// metadata entries.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned after the colon.
//
// Takes result (*querier_dto.DirectiveBlock) which holds
// the directive block to populate.
//
// Takes diagnostics (*[]querier_dto.SourceError) which
// holds the diagnostics slice to append errors to.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes directive (string) which specifies the directive
// keyword name.
//
// Takes keySpan (querier_dto.TextSpan) which holds the
// text span of the directive key.
func parseColonDirective(
	scanner *directiveLineScanner,
	result *querier_dto.DirectiveBlock,
	diagnostics *[]querier_dto.SourceError,
	filename string,
	directive string,
	keySpan querier_dto.TextSpan,
) {
	scanner.skipWhitespace()

	valueStart := scanner.column()
	value, _ := scanner.readRemainder()
	value = strings.TrimSpace(value)
	valueSpan := scanner.spanFrom(valueStart)

	lineSpan := directiveLineSpan(scanner)

	switch directive {
	case "name":
		result.Name = &querier_dto.NameDirective{
			Value:     value,
			Span:      lineSpan,
			KeySpan:   keySpan,
			ValueSpan: valueSpan,
		}

	case "command":
		commandDirective, parseError := parseCommandValue(value, lineSpan, keySpan, valueSpan, scanner.lineNumber, filename)
		if parseError != nil {
			*diagnostics = append(*diagnostics, *parseError)
			return
		}
		result.Command = commandDirective

	default:
		result.Metadata = append(result.Metadata, &querier_dto.MetadataDirective{
			Directive: directive,
			Value:     value,
			Span:      lineSpan,
			KeySpan:   keySpan,
			ValueSpan: valueSpan,
		})
	}
}

// directiveLineSpan constructs a TextSpan covering the
// entire current line of the scanner.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner whose line number and content are used.
//
// Returns querier_dto.TextSpan which holds the span from
// column 1 to end of line.
func directiveLineSpan(scanner *directiveLineScanner) querier_dto.TextSpan {
	return querier_dto.TextSpan{
		Line:      scanner.lineNumber,
		Column:    1,
		EndLine:   scanner.lineNumber,
		EndColumn: len(scanner.line) + 1,
	}
}

// parseCommandValue parses a command directive value
// string into a CommandDirective.
//
// Takes value (string) which specifies the command name
// to parse.
//
// Takes lineSpan (querier_dto.TextSpan) which holds the
// span of the full directive line.
//
// Takes keySpan (querier_dto.TextSpan) which holds the
// span of the directive key.
//
// Takes valueSpan (querier_dto.TextSpan) which holds the
// span of the directive value.
//
// Takes lineNumber (int) which specifies the line number
// for error reporting.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Returns *querier_dto.CommandDirective which holds the
// parsed command directive, or nil on error.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseCommandValue(
	value string,
	lineSpan querier_dto.TextSpan,
	keySpan querier_dto.TextSpan,
	valueSpan querier_dto.TextSpan,
	lineNumber int,
	filename string,
) (*querier_dto.CommandDirective, *querier_dto.SourceError) {
	command, valid := parseQueryCommand(value)
	if !valid {
		return nil, &querier_dto.SourceError{
			Filename: filename,
			Line:     lineNumber,
			Column:   valueSpan.Column,
			Message:  fmt.Sprintf("unknown query command %q", value),
			Severity: querier_dto.SeverityError,
			Code:     querier_dto.CodeDirectiveSyntax,
		}
	}
	return &querier_dto.CommandDirective{
		Value:     value,
		Span:      lineSpan,
		KeySpan:   keySpan,
		ValueSpan: valueSpan,
		Command:   command,
	}, nil
}

// parseNumberedParameterDirective parses a numbered
// parameter directive such as $1 as piko.param(name).
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the prefix character.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes prefix (byte) which specifies the parameter
// prefix character.
//
// Returns *querier_dto.ParameterDirective which holds the
// parsed parameter directive, or nil on error.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseNumberedParameterDirective(
	scanner *directiveLineScanner,
	filename string,
	prefix byte,
) (*querier_dto.ParameterDirective, *querier_dto.SourceError) {
	numberStart := scanner.column()
	scanner.advance()

	number, numberSpan, numberError := parseParameterNumber(scanner, numberStart, filename, prefix)
	if numberError != nil {
		return nil, numberError
	}

	scanner.skipWhitespace()
	if !scanner.matchKeyword("as") {
		return nil, syntaxError(filename, scanner.lineNumber, scanner.column(), fmt.Sprintf("expected 'as' after parameter number %c%d", prefix, number))
	}

	scanner.skipWhitespace()
	kind, kindSpan, kindError := parseParameterKindFromScanner(scanner, filename)
	if kindError != nil {
		return nil, kindError
	}

	parameterName, nameSpan, nameError := parseParameterName(scanner, filename)
	if nameError != nil {
		return nil, nameError
	}

	options := parseAllOptions(scanner)

	directive := &querier_dto.ParameterDirective{
		Span: querier_dto.TextSpan{
			Line:      scanner.lineNumber,
			Column:    1,
			EndLine:   scanner.lineNumber,
			EndColumn: len(scanner.line) + 1,
		},
		Number:     number,
		NumberSpan: numberSpan,
		Kind:       kind,
		KindSpan:   kindSpan,
		Name:       parameterName,
		NameSpan:   nameSpan,
		Options:    options,
	}

	resolveParameterOptions(directive)

	return directive, nil
}

// parseNamedParameterDirective parses a named parameter
// directive such as @name as piko.param(name).
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the prefix character.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes sequentialNumber (int) which specifies the
// sequential parameter number to assign.
//
// Returns *querier_dto.ParameterDirective which holds the
// parsed parameter directive, or nil on error.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseNamedParameterDirective(
	scanner *directiveLineScanner,
	filename string,
	sequentialNumber int,
) (*querier_dto.ParameterDirective, *querier_dto.SourceError) {
	referenceStart := scanner.column()
	scanner.advance()

	identifier, _ := scanner.readWord()
	if identifier == "" {
		return nil, syntaxError(filename, scanner.lineNumber, referenceStart, "expected identifier after parameter prefix")
	}
	referenceSpan := scanner.spanFrom(referenceStart)

	scanner.skipWhitespace()
	if !scanner.matchKeyword("as") {
		return nil, syntaxError(filename, scanner.lineNumber, scanner.column(), fmt.Sprintf("expected 'as' after parameter name %q", identifier))
	}

	scanner.skipWhitespace()
	kind, kindSpan, kindError := parseParameterKindFromScanner(scanner, filename)
	if kindError != nil {
		return nil, kindError
	}

	parameterName := identifier
	nameSpan := referenceSpan
	if !scanner.atEnd() && scanner.current() == '(' {
		overrideName, overrideSpan, nameError := parseParameterName(scanner, filename)
		if nameError != nil {
			return nil, nameError
		}
		parameterName = overrideName
		nameSpan = overrideSpan
	}

	options := parseAllOptions(scanner)

	directive := &querier_dto.ParameterDirective{
		Span: querier_dto.TextSpan{
			Line:      scanner.lineNumber,
			Column:    1,
			EndLine:   scanner.lineNumber,
			EndColumn: len(scanner.line) + 1,
		},
		Number:        sequentialNumber,
		NumberSpan:    referenceSpan,
		DirectiveName: identifier,
		IsNamed:       true,
		Kind:          kind,
		KindSpan:      kindSpan,
		Name:          parameterName,
		NameSpan:      nameSpan,
		Options:       options,
	}

	resolveParameterOptions(directive)

	return directive, nil
}

// parseParameterNumber reads and converts the numeric
// portion of a numbered parameter reference.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned after the prefix character.
//
// Takes numberStart (int) which specifies the column
// where the number begins for span tracking.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Takes prefix (byte) which specifies the parameter
// prefix character for error messages.
//
// Returns int which holds the parsed parameter number.
//
// Returns querier_dto.TextSpan which holds the text span
// covering the number.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseParameterNumber(
	scanner *directiveLineScanner,
	numberStart int,
	filename string,
	prefix byte,
) (int, querier_dto.TextSpan, *querier_dto.SourceError) {
	digits, _ := scanner.readDigits()
	if digits == "" {
		return 0, querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, numberStart, fmt.Sprintf("expected parameter number after '%c'", prefix))
	}

	number, conversionError := strconv.Atoi(digits)
	if conversionError != nil {
		return 0, querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, numberStart, fmt.Sprintf("invalid parameter number: %s", digits))
	}

	return number, scanner.spanFrom(numberStart), nil
}

// parseParameterKindFromScanner reads and resolves a piko
// parameter kind keyword from the scanner.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the piko prefix.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Returns querier_dto.ParameterDirectiveKind which holds
// the resolved parameter kind.
//
// Returns querier_dto.TextSpan which holds the text span
// covering the kind keyword.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseParameterKindFromScanner(
	scanner *directiveLineScanner,
	filename string,
) (querier_dto.ParameterDirectiveKind, querier_dto.TextSpan, *querier_dto.SourceError) {
	kindStart := scanner.column()
	if !scanner.matchString(pikoPrefix) {
		return 0, querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, scanner.column(), "expected 'piko.' after 'as'")
	}

	kindName, _ := scanner.readWord()
	kindSpan := scanner.spanFrom(kindStart)

	kind, validKind := parseParameterDirectiveKind(kindName)
	if !validKind {
		return 0, querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, kindSpan.Column, fmt.Sprintf("unknown parameter kind %q", kindName))
	}

	return kind, kindSpan, nil
}

// parseParameterName reads a parenthesised parameter
// name from the scanner.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the opening parenthesis.
//
// Takes filename (string) which specifies the source file
// path for error reporting.
//
// Returns string which holds the extracted parameter
// name.
//
// Returns querier_dto.TextSpan which holds the text span
// covering the parameter name.
//
// Returns *querier_dto.SourceError which holds the parse
// error, or nil on success.
func parseParameterName(
	scanner *directiveLineScanner,
	filename string,
) (string, querier_dto.TextSpan, *querier_dto.SourceError) {
	if scanner.current() != '(' {
		return "", querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, scanner.column(), "expected '(' after parameter kind")
	}
	scanner.advance()

	nameStart := scanner.column()
	parameterName, _ := scanner.readWord()
	nameSpan := scanner.spanFrom(nameStart)

	if scanner.current() != ')' {
		return "", querier_dto.TextSpan{}, syntaxError(filename, scanner.lineNumber, scanner.column(), "expected ')' after parameter name")
	}
	scanner.advance()

	return parameterName, nameSpan, nil
}

// parseAllOptions reads all remaining key:value option
// pairs from the scanner.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned after the parameter kind.
//
// Returns []*querier_dto.DirectiveOption which holds the
// parsed options.
func parseAllOptions(scanner *directiveLineScanner) []*querier_dto.DirectiveOption {
	var options []*querier_dto.DirectiveOption
	scanner.skipWhitespace()
	for !scanner.atEnd() {
		option := parseOption(scanner)
		if option != nil {
			options = append(options, option)
		}
		scanner.skipWhitespace()
	}
	return options
}

// parseOption reads a single key:value option pair from
// the scanner.
//
// Takes scanner (*directiveLineScanner) which holds the
// scanner positioned at the start of the option.
//
// Returns *querier_dto.DirectiveOption which holds the
// parsed option, or nil if the syntax is invalid.
func parseOption(scanner *directiveLineScanner) *querier_dto.DirectiveOption {
	optionStart := scanner.column()
	keyStart := scanner.column()
	key, _ := scanner.readUntilByte(':')
	if key == "" {
		scanner.advance()
		return nil
	}
	keySpan := scanner.spanFrom(keyStart)

	if scanner.current() != ':' {
		return nil
	}
	scanner.advance()

	valueStart := scanner.column()
	value, _ := scanner.readUntilWhitespace()
	valueSpan := scanner.spanFrom(valueStart)

	return &querier_dto.DirectiveOption{
		Key:       key,
		Value:     value,
		Span:      scanner.spanFrom(optionStart),
		KeySpan:   keySpan,
		ValueSpan: valueSpan,
	}
}

// resolveParameterOptions applies parsed key:value
// options to the parameter directive's typed fields.
//
// Takes directive (*querier_dto.ParameterDirective)
// which holds the directive whose options are resolved
// in place.
func resolveParameterOptions(directive *querier_dto.ParameterDirective) {
	for _, option := range directive.Options {
		resolveOption(directive, option)
	}
}

// resolveOption applies a single key:value option to the
// appropriate field on the parameter directive.
//
// Takes directive (*querier_dto.ParameterDirective)
// which holds the directive to update.
//
// Takes option (*querier_dto.DirectiveOption) which holds
// the option key and value to apply.
func resolveOption(directive *querier_dto.ParameterDirective, option *querier_dto.DirectiveOption) {
	switch option.Key {
	case "type":
		directive.TypeHint = new(option.Value)
	case "nullable":
		directive.Nullable = new(option.Value == "true")
	case "columns":
		for column := range strings.SplitSeq(option.Value, ",") {
			trimmed := strings.TrimSpace(column)
			if trimmed != "" {
				directive.Columns = append(directive.Columns, trimmed)
			}
		}
	case "default":
		if intValue, conversionError := strconv.Atoi(option.Value); conversionError == nil {
			directive.DefaultVal = &intValue
		}
	case "max":
		if intValue, conversionError := strconv.Atoi(option.Value); conversionError == nil {
			directive.MaxVal = &intValue
		}
	}
}

// extractQueryDirectives extracts query-level directives
// such as group_by, nullable, and runtime from a
// directive block.
//
// Takes block (*querier_dto.DirectiveBlock) which holds
// the parsed directive block to extract from.
//
// Returns *querier_dto.QueryDirectives which holds the
// extracted query-level directive settings.
func extractQueryDirectives(block *querier_dto.DirectiveBlock) *querier_dto.QueryDirectives {
	directives := &querier_dto.QueryDirectives{}
	for _, metadata := range block.Metadata {
		switch metadata.Directive {
		case "group_by":
			directives.GroupByKeys = append(directives.GroupByKeys, metadata.Value)
		case "nullable":
			setBoolOverride(&directives.NullableOverride, metadata.Value)
		case "readonly":
			setBoolOverride(&directives.ReadOnlyOverride, metadata.Value)
		case "runtime":
			switch metadata.Value {
			case "true":
				directives.DynamicRuntime = true
			case "false":
				directives.DynamicRuntime = false
			}
		case "dynamic":
			if metadata.Value == "runtime" {
				directives.DynamicRuntime = true
			}
		}
	}
	return directives
}

// setBoolOverride sets a boolean pointer override from a
// string value of "true" or "false".
//
// Takes target (**bool) which holds the pointer to the boolean override field to set.
//
// Takes value (string) which specifies the string representation of the boolean value.
func setBoolOverride(target **bool, value string) {
	switch value {
	case "true":
		*target = new(true)
	case "false":
		*target = new(false)
	}
}

// syntaxError constructs a SourceError for a directive syntax violation.
//
// Takes filename (string) which specifies the source file path.
//
// Takes lineNumber (int) which specifies the line number of the error.
//
// Takes column (int) which specifies the column number of the error.
//
// Takes message (string) which specifies the human-readable error description.
//
// Returns *querier_dto.SourceError which holds the
// constructed syntax error.
func syntaxError(filename string, lineNumber int, column int, message string) *querier_dto.SourceError {
	return &querier_dto.SourceError{
		Filename: filename,
		Line:     lineNumber,
		Column:   column,
		Message:  message,
		Severity: querier_dto.SeverityError,
		Code:     querier_dto.CodeDirectiveSyntax,
	}
}

// parseParameterDirectiveKind maps a parameter kind name to its enumerated constant.
//
// Takes name (string) which specifies the kind name to resolve.
//
// Returns querier_dto.ParameterDirectiveKind which holds
// the resolved kind.
//
// Returns bool which indicates whether the name was
// recognised.
func parseParameterDirectiveKind(name string) (querier_dto.ParameterDirectiveKind, bool) {
	switch name {
	case "param":
		return querier_dto.ParameterDirectiveParam, true
	case "optional":
		return querier_dto.ParameterDirectiveOptional, true
	case "slice":
		return querier_dto.ParameterDirectiveSlice, true
	case "sortable":
		return querier_dto.ParameterDirectiveSortable, true
	case "limit":
		return querier_dto.ParameterDirectiveLimit, true
	case "offset":
		return querier_dto.ParameterDirectiveOffset, true
	default:
		return 0, false
	}
}

// parseQueryCommand maps a command name string to its
// enumerated QueryCommand constant.
//
// Takes value (string) which specifies the command name
// to resolve, compared case-insensitively.
//
// Returns querier_dto.QueryCommand which holds the
// resolved command.
//
// Returns bool which indicates whether the command name
// was recognised.
func parseQueryCommand(value string) (querier_dto.QueryCommand, bool) {
	switch strings.ToLower(value) {
	case "one":
		return querier_dto.QueryCommandOne, true
	case "many":
		return querier_dto.QueryCommandMany, true
	case "exec":
		return querier_dto.QueryCommandExec, true
	case "execresult":
		return querier_dto.QueryCommandExecResult, true
	case "execrows":
		return querier_dto.QueryCommandExecRows, true
	case "batch":
		return querier_dto.QueryCommandBatch, true
	case "stream":
		return querier_dto.QueryCommandStream, true
	case "copyfrom":
		return querier_dto.QueryCommandCopyFrom, true
	default:
		return 0, false
	}
}
