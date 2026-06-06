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

// validateCall enforces directive spec rules against the parsed call.
//
// At most len(spec.Positionals) positional arguments are accepted. A keyword argument
// whose key matches a positional name fills that slot when it has not already been filled
// positionally. A required slot that ends up empty (no positional and no matching keyword
// argument) produces Q031. A keyword argument whose key matches neither a positional name
// nor a keyword argument name produces Q027 with a suggestion, and the same keyword
// argument key seen twice produces Q029. Positionals carry a Kind that is validated the
// same way keyword argument values are.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the directive's parameters.
// Takes call (*callArgs) which holds the parsed call arguments.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes directiveName (string) which prefixes the messages.
//
// Returns []querier_dto.SourceError which holds the collected diagnostics.
func validateCall(spec *querier_dto.DirectiveSpec, call *callArgs, errorBuilder querier_dto.ErrorBuilder, directiveName string) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError

	if extra := len(call.positionals) - len(spec.Positionals); extra > 0 {
		first := call.positionals[len(spec.Positionals)]
		diagnostics = append(diagnostics, errorBuilder.DirectiveSyntax(first.span, fmt.Sprintf("%s accepts at most %d positional argument(s)", directiveName, len(spec.Positionals))))
	}

	filled := make([]bool, len(spec.Positionals))
	diagnostics = append(diagnostics, validatePositionalCallArgs(spec, call, filled, errorBuilder)...)
	diagnostics = append(diagnostics, validateKeywordCallArgs(spec, call, filled, errorBuilder, directiveName)...)
	diagnostics = append(diagnostics, collectMissingRequired(spec, call, filled, errorBuilder, directiveName)...)
	diagnostics = append(diagnostics, validateMutuallyExclusiveKeywords(spec, call, errorBuilder, directiveName)...)
	return diagnostics
}

// validateMutuallyExclusiveKeywords enforces that `default` and `optional` are not both
// supplied on the same parameter directive.
//
// A parameter cannot simultaneously carry a fallback value and be dropped when the caller
// passes nil. The check only applies to directives whose spec declares both keyword
// arguments, so it stays silent for directives that define neither.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the directive's keyword
// arguments.
// Takes call (*callArgs) which holds the parsed keyword arguments.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
// Takes directiveName (string) which prefixes the message.
//
// Returns []querier_dto.SourceError with one diagnostic when both keywords are present.
func validateMutuallyExclusiveKeywords(spec *querier_dto.DirectiveSpec, call *callArgs, errorBuilder querier_dto.ErrorBuilder, directiveName string) []querier_dto.SourceError {
	if _, hasDefault := querier_dto.LookupKeywordArgument(spec, "default"); !hasDefault {
		return nil
	}
	if _, hasOptional := querier_dto.LookupKeywordArgument(spec, "optional"); !hasOptional {
		return nil
	}
	var optionalArgument *parsedKeywordArgument
	sawDefault := false
	for index := range call.keywordArguments {
		switch call.keywordArguments[index].key {
		case "default":
			sawDefault = true
		case "optional":
			optionalArgument = &call.keywordArguments[index]
		}
	}
	if sawDefault && optionalArgument != nil {
		return []querier_dto.SourceError{
			errorBuilder.MutuallyExclusiveKeywordArgument(
				optionalArgument.keySpan, directiveName, "default", "optional",
			),
		}
	}
	return nil
}

// validatePositionalCallArgs walks the parsed positionals, marking the filled slots and
// emitting value-shape diagnostics.
//
// Positionals beyond spec.Positionals are ignored here because the outer validateCall
// already reported the arity overflow.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the positional slots.
// Takes call (*callArgs) which holds the parsed positionals.
// Takes filled ([]bool) which is marked true for each slot a positional fills.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
//
// Returns []querier_dto.SourceError which holds the value-shape diagnostics.
func validatePositionalCallArgs(spec *querier_dto.DirectiveSpec, call *callArgs, filled []bool, errorBuilder querier_dto.ErrorBuilder) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	for index := range call.positionals {
		if index >= len(spec.Positionals) {
			break
		}
		filled[index] = true
		if valueDiagnostic := validatePositionalValue(&spec.Positionals[index], &call.positionals[index], errorBuilder); valueDiagnostic != nil {
			diagnostics = append(diagnostics, *valueDiagnostic)
		}
	}
	return diagnostics
}

// validateKeywordCallArgs verifies each parsed keyword argument.
//
// It catches duplicates, routes positional-named keywords into their matching slots, and
// dispatches value-shape checks for everything else. The filled slice is updated for
// downstream missing-required detection.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the directive's parameters.
// Takes call (*callArgs) which holds the parsed keyword arguments.
// Takes filled ([]bool) which is marked true for each positional slot a keyword fills.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes directiveName (string) which prefixes the messages.
//
// Returns []querier_dto.SourceError which holds the collected diagnostics.
func validateKeywordCallArgs(spec *querier_dto.DirectiveSpec, call *callArgs, filled []bool, errorBuilder querier_dto.ErrorBuilder, directiveName string) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	seenKeywordArguments := make(map[string]int, len(call.keywordArguments))
	for index := range call.keywordArguments {
		keywordArgument := &call.keywordArguments[index]
		if _, exists := seenKeywordArguments[keywordArgument.key]; exists {
			diagnostics = append(diagnostics, errorBuilder.Duplicate(keywordArgument.keySpan, keywordArgument.key))
			continue
		}
		seenKeywordArguments[keywordArgument.key] = index
		diagnostics = append(diagnostics, processKeywordCallArg(spec, keywordArgument, filled, errorBuilder, directiveName)...)
	}
	return diagnostics
}

// processKeywordCallArg classifies one keyword argument against spec.
//
// It may fill an unused positional slot, match a keyword-only spec, or be reported as
// unknown.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the directive's parameters.
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword argument to
// classify.
// Takes filled ([]bool) which is marked true when the keyword fills a positional slot.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes directiveName (string) which prefixes the messages.
//
// Returns []querier_dto.SourceError which holds the diagnostics for this keyword.
func processKeywordCallArg(
	spec *querier_dto.DirectiveSpec,
	keywordArgument *parsedKeywordArgument,
	filled []bool,
	errorBuilder querier_dto.ErrorBuilder,
	directiveName string,
) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	if positionalSpec, positionalIndex, isPositional := querier_dto.LookupPositional(spec, keywordArgument.key); isPositional {
		if filled[positionalIndex] {
			diagnostics = append(diagnostics, errorBuilder.Duplicate(keywordArgument.keySpan, keywordArgument.key))
			return diagnostics
		}
		filled[positionalIndex] = true
		synthetic := parsedPositional{value: keywordArgument.value, span: keywordArgument.valueSpan}
		if valueDiagnostic := validatePositionalValue(positionalSpec, &synthetic, errorBuilder); valueDiagnostic != nil {
			diagnostics = append(diagnostics, *valueDiagnostic)
		}
		return diagnostics
	}

	keywordArgumentSpec, found := querier_dto.LookupKeywordArgument(spec, keywordArgument.key)
	if !found {
		candidates := append(querier_dto.PositionalNames(spec), querier_dto.KeywordArgumentNames(spec)...)
		diagnostics = append(diagnostics, errorBuilder.UnknownKeywordArgument(keywordArgument.keySpan, directiveName, keywordArgument.key, candidates))
		return diagnostics
	}
	if valueDiagnostic := validateKeywordArgumentValue(keywordArgumentSpec, keywordArgument, errorBuilder); valueDiagnostic != nil {
		diagnostics = append(diagnostics, *valueDiagnostic)
	}
	return diagnostics
}

// collectMissingRequired emits one Q031 diagnostic per required positional slot the
// caller failed to fill.
//
// The diagnostic is anchored to the closing paren when present, or the opener when the
// call was unterminated.
//
// Takes spec (*querier_dto.DirectiveSpec) which declares the positional slots.
// Takes call (*callArgs) which supplies the spans for anchoring diagnostics.
// Takes filled ([]bool) which records which slots were filled.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostics.
// Takes directiveName (string) which prefixes the messages.
//
// Returns []querier_dto.SourceError which holds the missing-required diagnostics.
func collectMissingRequired(spec *querier_dto.DirectiveSpec, call *callArgs, filled []bool, errorBuilder querier_dto.ErrorBuilder, directiveName string) []querier_dto.SourceError {
	var diagnostics []querier_dto.SourceError
	missingSpan := call.closeSpan
	if !call.closed {
		missingSpan = call.openSpan
	}
	for index := range spec.Positionals {
		positional := &spec.Positionals[index]
		if !positional.Required || filled[index] {
			continue
		}
		diagnostics = append(diagnostics, errorBuilder.MissingRequired(missingSpan, directiveName, positional.Name))
	}
	return diagnostics
}

// validatePositionalValue runs the same kind-specific value-shape check as
// validateKeywordArgumentValue, against a positional slot's spec.
//
// The PositionalSpec mirrors the KeywordArgumentSpec value-shape fields, so this is a
// thin adapter onto the same logic.
//
// Takes spec (*querier_dto.PositionalSpec) which declares the positional's value shape.
// Takes positional (*parsedPositional) which is the positional value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the value-shape diagnostic, or nil when the
// value is well formed.
func validatePositionalValue(spec *querier_dto.PositionalSpec, positional *parsedPositional, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	adapter := querier_dto.KeywordArgumentSpec{
		Name:          spec.Name,
		Kind:          spec.Kind,
		Required:      spec.Required,
		AllowedValues: spec.AllowedValues,
		Description:   spec.Description,
	}
	syntheticKeywordArgument := parsedKeywordArgument{
		key:       spec.Name,
		value:     positional.value,
		valueSpan: positional.span,
	}
	return validateKeywordArgumentValue(&adapter, &syntheticKeywordArgument, errorBuilder)
}

// validateKeywordArgumentValue dispatches per spec.Kind to the kind-specific value
// validator.
//
// It returns a pointer to the kind's Q-coded diagnostic so the caller can fold it into
// its own list.
//
// Takes spec (*querier_dto.KeywordArgumentSpec) which declares the value shape.
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the diagnostic, or nil when the value is well
// formed.
func validateKeywordArgumentValue(spec *querier_dto.KeywordArgumentSpec, keywordArgument *parsedKeywordArgument, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	switch spec.Kind {
	case querier_dto.KeywordArgumentList:
		return validateListKeywordValue(keywordArgument, errorBuilder)
	case querier_dto.KeywordArgumentInt:
		return validateIntKeywordValue(keywordArgument, errorBuilder)
	case querier_dto.KeywordArgumentBool:
		return validateBoolKeywordValue(spec, keywordArgument, errorBuilder)
	case querier_dto.KeywordArgumentIdent:
		return validateIdentKeywordValue(spec, keywordArgument, errorBuilder)
	case querier_dto.KeywordArgumentQualifiedIdent, querier_dto.KeywordArgumentString:
		return nil
	}
	return nil
}

// validateListKeywordValue checks that the value carries the list shape expected by the
// spec, returning the canonical Q-coded diagnostic when it does not.
//
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the diagnostic, or nil when the value is a
// list.
func validateListKeywordValue(keywordArgument *parsedKeywordArgument, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	if keywordArgument.value.isList {
		return nil
	}
	return new(errorBuilder.InvalidValueShape(keywordArgument.valueSpan, keywordArgument.key, "a list like [a, b, c]"))
}

// validateIntKeywordValue verifies the raw value parses as a decimal integer literal that
// fits in a 64-bit integer.
//
// Validating at a fixed 64-bit width (rather than the platform-dependent strconv.Atoi)
// keeps the accepted range identical regardless of the host architecture so validation
// and the apply phase always agree.
//
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the diagnostic, or nil when the value is a
// valid integer literal.
func validateIntKeywordValue(keywordArgument *parsedKeywordArgument, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	if _, parseErr := strconv.ParseInt(keywordArgument.value.raw, 10, 64); parseErr == nil {
		return nil
	}
	return new(errorBuilder.InvalidValueShape(keywordArgument.valueSpan, keywordArgument.key, "an integer literal"))
}

// validateBoolKeywordValue ensures the value is the lowercased literalTrue or
// literalFalse marker, returning an InvalidEnum diagnostic otherwise.
//
// Takes spec (*querier_dto.KeywordArgumentSpec) which supplies the allowed values.
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the diagnostic, or nil when the value is a
// boolean literal.
func validateBoolKeywordValue(spec *querier_dto.KeywordArgumentSpec, keywordArgument *parsedKeywordArgument, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	lower := strings.ToLower(keywordArgument.value.raw)
	if lower == literalTrue || lower == literalFalse {
		return nil
	}
	return new(errorBuilder.InvalidEnum(keywordArgument.valueSpan, keywordArgument.key, keywordArgument.value.raw, spec.AllowedValues))
}

// validateIdentKeywordValue checks the value against the spec's allowed-values
// enumeration, when one is present.
//
// An empty allow-list short-circuits to nil because the parser already accepted the
// identifier shape.
//
// Takes spec (*querier_dto.KeywordArgumentSpec) which supplies the allowed values.
// Takes keywordArgument (*parsedKeywordArgument) which is the keyword value to check.
// Takes errorBuilder (querier_dto.ErrorBuilder) which renders the diagnostic.
//
// Returns *querier_dto.SourceError which is the diagnostic, or nil when the value is
// allowed.
func validateIdentKeywordValue(spec *querier_dto.KeywordArgumentSpec, keywordArgument *parsedKeywordArgument, errorBuilder querier_dto.ErrorBuilder) *querier_dto.SourceError {
	if len(spec.AllowedValues) == 0 {
		return nil
	}
	for _, candidate := range spec.AllowedValues {
		if strings.EqualFold(candidate, keywordArgument.value.raw) {
			return nil
		}
	}
	return new(errorBuilder.InvalidEnum(keywordArgument.valueSpan, keywordArgument.key, keywordArgument.value.raw, spec.AllowedValues))
}
