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

package ast_domain

const (
	// CodeInvalidNumberLiteral indicates a malformed numeric literal.
	CodeInvalidNumberLiteral = "T001"

	// CodeInvalidStringLiteral indicates a malformed string literal.
	CodeInvalidStringLiteral = "T002"

	// CodeInvalidEscapeSequence indicates an invalid escape sequence in a
	// rune literal.
	CodeInvalidEscapeSequence = "T003"

	// CodeInvalidTemporalFormat indicates an invalid datetime, date, time,
	// or duration format.
	CodeInvalidTemporalFormat = "T004"

	// CodeInvalidRuneLiteral indicates a rune literal that does not contain
	// exactly one character.
	CodeInvalidRuneLiteral = "T005"

	// CodeInvalidTemplateLiteral indicates a malformed template literal
	// (e.g. missing backticks).
	CodeInvalidTemplateLiteral = "T006"

	// CodeUnterminatedInterpolation indicates an unterminated expression
	// inside a template literal or text interpolation.
	CodeUnterminatedInterpolation = "T007"

	// CodeUnexpectedToken indicates a token was found where a different token
	// was expected.
	CodeUnexpectedToken = "T008"

	// CodeUnexpectedEOF indicates the expression ended prematurely while
	// parsing.
	CodeUnexpectedEOF = "T009"

	// CodeMissingIdentifier indicates an identifier was expected after a dot,
	// optional chain, or at-sign operator.
	CodeMissingIdentifier = "T010"

	// CodeMissingOperand indicates an operand was expected after an operator
	// but was not found.
	CodeMissingOperand = "T011"

	// CodeMissingClosingDelimiter indicates a closing bracket, parenthesis,
	// or brace was expected but not found.
	CodeMissingClosingDelimiter = "T012"

	// CodeMissingSeparator indicates a colon, comma, or other separator was
	// expected in a structured expression.
	CodeMissingSeparator = "T013"

	// CodeIncompleteConstruct indicates an array, object, function call, or
	// ternary expression was not fully formed.
	CodeIncompleteConstruct = "T014"

	// CodeExpressionDepthExceeded indicates the expression nesting depth has
	// exceeded the maximum allowed limit.
	CodeExpressionDepthExceeded = "T015"

	// CodeDisallowedFeature indicates an expression feature that is not
	// permitted in the current context.
	CodeDisallowedFeature = "T016"

	// CodeLexerError indicates an error from the expression lexer.
	CodeLexerError = "T017"

	// CodeTrailingTokens indicates unexpected tokens remaining after the
	// expression was fully parsed.
	CodeTrailingTokens = "T018"

	// CodeUnexpectedComma indicates a comma was used where it is not a valid
	// operator.
	CodeUnexpectedComma = "T019"

	// CodeConflictingDirectives indicates two mutually exclusive directives
	// on the same element (e.g. p-text and p-html, or p-else and p-else-if).
	CodeConflictingDirectives = "T020"

	// CodeDuplicateDirective indicates the same directive appears more than
	// once on a single element.
	CodeDuplicateDirective = "T021"

	// CodeInvalidDirectiveValue indicates a directive has an empty or
	// syntactically invalid value.
	CodeInvalidDirectiveValue = "T022"

	// CodeInvalidDirectivePlacement indicates a conditional directive like
	// p-else is not preceded by the required p-if.
	CodeInvalidDirectivePlacement = "T023"

	// CodeDirectivePrecedence indicates a directive appears before p-for but
	// will be evaluated after it.
	CodeDirectivePrecedence = "T024"

	// CodeInvalidDirectiveTarget indicates a directive is applied to an
	// element type that does not support it (e.g. p-model on a <div>).
	CodeInvalidDirectiveTarget = "T025"

	// CodeMissingLoopKey indicates a p-for loop is missing a p-key binding
	// for efficient list rendering.
	CodeMissingLoopKey = "T026"

	// CodeAttributeConflict indicates a static attribute conflicts with a
	// dynamic p-bind on the same property.
	CodeAttributeConflict = "T027"

	// CodeDirectiveOverwritesContent indicates a directive like p-text or
	// p-html will overwrite existing child content.
	CodeDirectiveOverwritesContent = "T028"

	// CodeCSSParseError indicates an error while parsing a CSS selector or
	// expression.
	CodeCSSParseError = "T029"

	// CodeInternalParserError indicates an unexpected internal state in the
	// parser. This should not normally fire.
	CodeInternalParserError = "T030"
)
