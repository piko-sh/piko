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

package querier_dto

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// didYouMeanFormat is the standard suggestion-suffix format string used by every "did
	// you mean ...?" diagnostic emitted by the builder. Centralised so a phrasing change
	// ripples through every site that appends a Levenshtein suggestion.
	didYouMeanFormat = "%s; did you mean %q?"
)

// ErrorBuilder produces SourceError values with consistent shape and optional "did you
// mean" suggestions. Every directive validation site routes through this builder so
// message wording, severity, and suggestion behaviour stay uniform across the parser.
type ErrorBuilder struct {
	// Filename is recorded on every emitted SourceError so callers do not need to pass the
	// path at every site.
	Filename string
}

// NewErrorBuilder returns a builder bound to the given source file.
//
// Takes filename (string) which is the source file path recorded on every emitted error.
//
// Returns ErrorBuilder which is bound to the supplied filename.
func NewErrorBuilder(filename string) ErrorBuilder {
	return ErrorBuilder{Filename: filename}
}

// At builds a SourceError at the given span with the given code and message and
// SeverityError.
//
// Use this for low-level errors that do not have a canonical builder method.
//
// Takes span (TextSpan) which is the source span the error points at.
// Takes code (string) which is the diagnostic code to record.
// Takes message (string) which is the human-readable error message.
//
// Returns SourceError which carries the supplied span, code, and message.
func (builder ErrorBuilder) At(span TextSpan, code, message string) SourceError {
	return SourceError{
		Filename:  builder.Filename,
		Message:   message,
		Code:      code,
		Line:      span.Line,
		Column:    span.Column,
		EndLine:   span.EndLine,
		EndColumn: span.EndColumn,
		Severity:  SeverityError,
	}
}

// UnknownDirective reports a directive name that is not in the registry.
//
// When a Levenshtein neighbour is found in candidates the Suggestion field is populated
// with the closest match.
//
// Takes span (TextSpan) which is the source span of the offending directive.
// Takes given (string) which is the unknown directive name.
// Takes candidates ([]string) which are the known directive names to suggest from.
//
// Returns SourceError describing the unknown directive, with a suggestion when one is
// near.
func (builder ErrorBuilder) UnknownDirective(span TextSpan, given string, candidates []string) SourceError {
	suggestion := findClosestSuggestion(given, candidates)
	message := fmt.Sprintf("unknown directive %q", given)
	if suggestion != "" {
		message = fmt.Sprintf(didYouMeanFormat, message, suggestion)
	}
	return builder.withSuggestion(span, CodeUnknownDirective, message, suggestion)
}

// UnknownKeywordArgument reports a keyword argument key that the directive's spec does
// not accept.
//
// Suggestion carries the closest-matching keyword argument name.
//
// Takes span (TextSpan) which is the source span of the offending key.
// Takes directiveName (string) which is the directive the key was given on.
// Takes given (string) which is the unknown keyword argument key.
// Takes candidates ([]string) which are the accepted keyword argument names.
//
// Returns SourceError describing the unknown keyword argument, with a suggestion when one
// is near.
func (builder ErrorBuilder) UnknownKeywordArgument(span TextSpan, directiveName, given string, candidates []string) SourceError {
	suggestion := findClosestSuggestion(given, candidates)
	message := fmt.Sprintf("unknown keyword argument %q on %s", given, directiveName)
	if suggestion != "" {
		message = fmt.Sprintf(didYouMeanFormat, message, suggestion)
	}
	return builder.withSuggestion(span, CodeUnknownKeywordArgument, message, suggestion)
}

// InvalidEnum reports a keyword argument value that is not in the allowed enum set.
//
// Suggestion carries the closest-matching allowed value.
//
// Takes span (TextSpan) which is the source span of the offending value.
// Takes keywordArgument (string) which is the keyword argument the value was given for.
// Takes given (string) which is the rejected value.
// Takes allowed ([]string) which are the permitted enum values.
//
// Returns SourceError describing the invalid value, with a suggestion when one is near.
func (builder ErrorBuilder) InvalidEnum(span TextSpan, keywordArgument, given string, allowed []string) SourceError {
	suggestion := findClosestSuggestion(given, allowed)
	allowedDisplay := strings.Join(allowed, ", ")
	message := fmt.Sprintf("invalid value %q for keyword argument %q; allowed: %s", given, keywordArgument, allowedDisplay)
	if suggestion != "" {
		message = fmt.Sprintf("invalid value %q for keyword argument %q; did you mean %q? allowed: %s", given, keywordArgument, suggestion, allowedDisplay)
	}
	return builder.withSuggestion(span, CodeInvalidKeywordArgumentValue, message, suggestion)
}

// InvalidValueShape reports a keyword argument value that fails the kind-specific shape
// check, such as a non-integer for KeywordArgumentInt or a non-list for
// KeywordArgumentList.
//
// Used when there is no enum to suggest from.
//
// Takes span (TextSpan) which is the source span of the offending value.
// Takes keywordArgument (string) which is the keyword argument the value was given for.
// Takes expected (string) which describes the shape the value should have taken.
//
// Returns SourceError describing the malformed value.
func (builder ErrorBuilder) InvalidValueShape(span TextSpan, keywordArgument, expected string) SourceError {
	message := fmt.Sprintf("invalid value for keyword argument %q: expected %s", keywordArgument, expected)
	return builder.At(span, CodeInvalidKeywordArgumentValue, message)
}

// Duplicate reports the same keyword argument key appearing twice in one call.
//
// Takes span (TextSpan) which is the source span of the repeated key.
// Takes key (string) which is the duplicated keyword argument name.
//
// Returns SourceError describing the duplicate keyword argument.
func (builder ErrorBuilder) Duplicate(span TextSpan, key string) SourceError {
	message := fmt.Sprintf("duplicate keyword argument %q", key)
	return builder.At(span, CodeDuplicateKeywordArgument, message)
}

// MissingRequired reports a required positional argument or keyword argument that is
// absent from a directive call.
//
// Takes span (TextSpan) which is the source span of the directive call.
// Takes directiveName (string) which is the directive missing the argument.
// Takes what (string) which describes the missing required argument.
//
// Returns SourceError describing the missing required argument.
func (builder ErrorBuilder) MissingRequired(span TextSpan, directiveName, what string) SourceError {
	message := fmt.Sprintf("missing required %s on %s", what, directiveName)
	return builder.At(span, CodeMissingRequired, message)
}

// Unclosed reports a multi-line directive call whose paren depth never returns to zero
// before the directive header terminates.
//
// Takes span (TextSpan) which is the source span of the unclosed call.
// Takes opener (string) which names the construct that was left open.
//
// Returns SourceError describing the unclosed directive.
func (builder ErrorBuilder) Unclosed(span TextSpan, opener string) SourceError {
	message := fmt.Sprintf("unclosed %s; expected matching ')'", opener)
	return builder.At(span, CodeUnclosedDirective, message)
}

// DirectiveSyntax reports a low-level lexer or parser failure that does not fit one of
// the structured codes above, such as an unexpected character or unterminated string.
//
// Falls back to the CodeDirectiveSyntax code.
//
// Takes span (TextSpan) which is the source span of the syntax failure.
// Takes message (string) which is the human-readable failure description.
//
// Returns SourceError carrying the supplied message under the directive-syntax code.
func (builder ErrorBuilder) DirectiveSyntax(span TextSpan, message string) SourceError {
	return builder.At(span, CodeDirectiveSyntax, message)
}

// ParameterBindingSyntax reports a malformed parameter-binding line where the expected
// keyword token ("as", "piko", or a role name) did not appear.
//
// When the candidate slice is non-empty and the offending token is a near-miss typo, a
// Levenshtein suggestion is attached so the LSP can offer a quick-fix.
//
// Takes span (TextSpan) which is the source span of the offending token.
// Takes expected (string) which describes the token that was expected.
// Takes given (string) which is the token that was actually found.
// Takes candidates ([]string) which are the keyword tokens to suggest from.
//
// Returns SourceError describing the binding syntax error, with a suggestion when one is
// near.
func (builder ErrorBuilder) ParameterBindingSyntax(span TextSpan, expected, given string, candidates []string) SourceError {
	message := fmt.Sprintf("expected %s, got %q", expected, given)
	suggestion := findClosestSuggestion(given, candidates)
	if suggestion != "" {
		message = fmt.Sprintf(didYouMeanFormat, message, suggestion)
		return builder.withSuggestion(span, CodeParameterBindingSyntax, message, suggestion)
	}
	return builder.At(span, CodeParameterBindingSyntax, message)
}

// InvalidListLiteral reports a bracketed list whose closing token did not match the
// expected ']'.
//
// The value is treated as malformed.
//
// Takes span (TextSpan) which is the source span of the list literal.
// Takes got (string) which is the unexpected closing token that was found.
//
// Returns SourceError describing the malformed list literal.
func (builder ErrorBuilder) InvalidListLiteral(span TextSpan, got string) SourceError {
	message := fmt.Sprintf("expected ']' to close list literal, got %q", got)
	return builder.At(span, CodeInvalidListLiteral, message)
}

// UnterminatedString reports a string literal whose opening quote was not matched before
// the end of the directive line or input.
//
// Takes span (TextSpan) which is the source span of the string literal.
// Takes quote (string) which is the closing quote that was expected.
//
// Returns SourceError describing the unterminated string literal.
func (builder ErrorBuilder) UnterminatedString(span TextSpan, quote string) SourceError {
	message := fmt.Sprintf("unterminated string literal; expected closing %s", quote)
	return builder.At(span, CodeUnterminatedString, message)
}

// UnknownOverrideColumn reports a piko.column(name, ...) directive in a query header that
// targets an output column not present in the query's SELECT projection.
//
// Suggestion carries the closest-matching actual output column name.
//
// Takes span (TextSpan) which is the source span of the directive.
// Takes given (string) which is the unknown output column name.
// Takes candidates ([]string) which are the actual output column names to suggest from.
//
// Returns SourceError describing the unknown override column, with a suggestion when one
// is near.
func (builder ErrorBuilder) UnknownOverrideColumn(span TextSpan, given string, candidates []string) SourceError {
	suggestion := findClosestSuggestion(given, candidates)
	message := fmt.Sprintf("piko.column references unknown output column %q", given)
	if suggestion != "" {
		message = fmt.Sprintf(didYouMeanFormat, message, suggestion)
		return builder.withSuggestion(span, CodeUnknownOverrideColumn, message, suggestion)
	}
	return builder.At(span, CodeUnknownOverrideColumn, message)
}

// UnknownOverrideMigrationColumn reports a migration-file piko.column directive whose
// qualified name does not match any column in the catalogue.
//
// Suggestion carries the closest-matching table.column reference.
//
// Takes span (TextSpan) which is the source span of the directive.
// Takes given (string) which is the unknown qualified column name.
// Takes candidates ([]string) which are the catalogue table.column references to suggest
// from.
//
// Returns SourceError describing the unknown migration column, with a suggestion when one
// is near.
func (builder ErrorBuilder) UnknownOverrideMigrationColumn(span TextSpan, given string, candidates []string) SourceError {
	suggestion := findClosestSuggestion(given, candidates)
	message := fmt.Sprintf("piko.column references unknown column %q", given)
	if suggestion != "" {
		message = fmt.Sprintf(didYouMeanFormat, message, suggestion)
		return builder.withSuggestion(span, CodeUnknownOverrideMigrationColumn, message, suggestion)
	}
	return builder.At(span, CodeUnknownOverrideMigrationColumn, message)
}

// MutuallyExclusiveKeywordArgument reports two keyword arguments declared on the same
// directive call when only one of them may be set at a time.
//
// The span points at the second of the two; the quick-fix is to delete it.
//
// Takes span (TextSpan) which is the source span of the second argument.
// Takes directiveName (string) which is the directive both arguments were given on.
// Takes first (string) which is the first conflicting keyword argument name.
// Takes second (string) which is the second conflicting keyword argument name.
//
// Returns SourceError describing the mutually exclusive keyword arguments.
func (builder ErrorBuilder) MutuallyExclusiveKeywordArgument(span TextSpan, directiveName, first, second string) SourceError {
	message := fmt.Sprintf("%s: keyword arguments %q and %q are mutually exclusive; pick one", directiveName, first, second)
	return builder.At(span, CodeMutuallyExclusiveKeywordArgument, message)
}

// withSuggestion builds a SourceError at the span and attaches the given suggestion.
//
// Takes span (TextSpan) which is the source span the error points at.
// Takes code (string) which is the diagnostic code to record.
// Takes message (string) which is the human-readable error message.
// Takes suggestion (string) which is the closest-match suggestion to attach.
//
// Returns SourceError carrying the message and the attached suggestion.
func (builder ErrorBuilder) withSuggestion(span TextSpan, code, message, suggestion string) SourceError {
	source := builder.At(span, code, message)
	source.Suggestion = suggestion
	return source
}

// findClosestSuggestion returns the closest member of candidates to the input by
// Levenshtein distance, with a length-proportional threshold.
//
// The algorithm is a small local copy of the annotator_domain implementation so the
// querier_dto package keeps zero cross-domain coupling.
//
// Takes input (string) which is the token to find a near match for.
// Takes candidates ([]string) which are the known values to compare against.
//
// Returns string which is the closest candidate, or empty when none is close enough or
// the inputs are empty.
func findClosestSuggestion(input string, candidates []string) string {
	if input == "" || len(candidates) == 0 {
		return ""
	}
	const similarityThresholdDivisor = 3
	const minimumDistanceThreshold = 2

	const maxSuggestionInputRunes = 128

	inputLower := strings.ToLower(input)
	inputRunes := utf8.RuneCountInString(input)
	if inputRunes > maxSuggestionInputRunes {
		return ""
	}

	threshold := (inputRunes / similarityThresholdDivisor) + 1
	if threshold < minimumDistanceThreshold && inputRunes > 2 {
		threshold = minimumDistanceThreshold
	}

	bestDistance := threshold + 1
	bestMatch := ""

	for _, candidate := range candidates {
		distance := levenshteinDistance(inputLower, strings.ToLower(candidate))
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = candidate
		}
	}

	if bestDistance <= threshold {
		return bestMatch
	}
	return ""
}

// levenshteinDistance returns the minimum single-character edit distance between source
// and target.
//
// Uses a two-row dynamic programming table to keep allocation bounded.
//
// Takes source (string) which is the first string to compare.
// Takes target (string) which is the second string to compare.
//
// Returns int which is the minimum single-character edit distance between the two.
func levenshteinDistance(source, target string) int {
	sourceRunes := []rune(source)
	targetRunes := []rune(target)
	sourceLen := len(sourceRunes)
	targetLen := len(targetRunes)

	if sourceLen == 0 {
		return targetLen
	}
	if targetLen == 0 {
		return sourceLen
	}

	previous := make([]int, targetLen+1)
	current := make([]int, targetLen+1)
	for j := 0; j <= targetLen; j++ {
		previous[j] = j
	}
	for i := 1; i <= sourceLen; i++ {
		current[0] = i
		for j := 1; j <= targetLen; j++ {
			cost := 1
			if sourceRunes[i-1] == targetRunes[j-1] {
				cost = 0
			}
			deletion := previous[j] + 1
			insertion := current[j-1] + 1
			substitution := previous[j-1] + cost
			current[j] = min(deletion, insertion, substitution)
		}
		previous, current = current, previous
	}
	return previous[targetLen]
}
