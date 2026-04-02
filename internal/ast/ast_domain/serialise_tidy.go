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

// Provides formatting and tidying utilities for serialised Go code to improve readability and maintainability.
// Applies context-aware formatting rules, manages line breaks, and maintains consistent indentation in generated struct literals.

import (
	"strings"
	"unicode"
)

// tidyContext is a context key type used to pass tidy options through context.
type tidyContext int

const (
	// ctxDefault is the default tidy context with no special formatting rules.
	ctxDefault tidyContext = iota

	// ctxCompositeLiteral is the context type for composite literal expressions.
	ctxCompositeLiteral

	// ctxBlock is the block type for context-related code sections.
	ctxBlock

	// ctxTypeDecl indicates that the context is a type declaration.
	ctxTypeDecl

	// ctxStructTypeDef is the context key for struct type definitions.
	ctxStructTypeDef
)

// defaultTabWidth is the number of spaces used for tab indentation.
const defaultTabWidth = 4

// tidier tracks state while cleaning and formatting Go source code.
type tidier struct {
	// builder builds the reformatted output text.
	builder strings.Builder

	// runes holds the input text as a slice of runes for processing.
	runes []rune

	// contextStack tracks nested contexts during AST traversal.
	contextStack []tidyContext

	// cursor is the current position in the runes slice.
	cursor int

	// stringDelim is the delimiter character for the current string literal.
	stringDelim rune

	// inString tracks whether the cursor is inside a string literal.
	inString bool

	// inComment indicates whether the cursor is inside a comment block.
	inComment bool

	// isMultiLineComment indicates whether the comment uses /* */ style.
	isMultiLineComment bool
}

// run processes the runes and returns the tidied output.
//
// Returns string which contains the processed content.
func (t *tidier) run() string {
	t.builder.Grow(len(t.runes) * 2)

	for t.cursor < len(t.runes) {
		if t.inString {
			t.processString()
			continue
		}
		if t.inComment {
			t.processComment()
			continue
		}
		t.processCode()
	}
	return t.builder.String()
}

// processString handles a character within a string literal.
func (t *tidier) processString() {
	r := t.runes[t.cursor]
	_, _ = t.builder.WriteRune(r)
	if r == t.stringDelim {
		slashes := 0
		for j := t.cursor - 1; j >= 0 && t.runes[j] == '\\'; j-- {
			slashes++
		}
		if slashes%2 == 0 {
			t.inString = false
		}
	}
	t.cursor++
}

// processComment handles a single character within a comment block.
func (t *tidier) processComment() {
	r := t.runes[t.cursor]

	if t.isMultiLineComment {
		_, _ = t.builder.WriteRune(r)
		if r == '*' && t.peekRune() == '/' {
			_, _ = t.builder.WriteRune(t.peekRune())
			t.cursor++
			t.inComment = false
		}
	} else {
		if r == '\n' {
			_, _ = t.builder.WriteRune(r)
			t.inComment = false
		} else if r == '\\' && t.peekRune() == 'n' {
			_, _ = t.builder.WriteRune('\n')
			t.cursor++
			t.inComment = false
		} else {
			_, _ = t.builder.WriteRune(r)
		}
	}
	t.cursor++
}

// processCode handles a single character when outside strings and comments.
func (t *tidier) processCode() {
	r := t.runes[t.cursor]
	nextRune := t.peekRune()

	switch r {
	case '"', '\'', '`':
		t.inString = true
		t.stringDelim = r
		_, _ = t.builder.WriteRune(r)
	case '/':
		isStartingComment := nextRune == '/' || nextRune == '*'
		if isStartingComment {
			t.handleCommentStart()
		} else {
			_, _ = t.builder.WriteRune(r)
		}
	case '{':
		t.handleOpeningBrace()
	case '}':
		t.handleClosingBrace()
	case '[':
		t.handleOpeningBracket()
	case ']':
		t.handleClosingBracket()
	case ',':
		t.handleComma()
	default:
		_, _ = t.builder.WriteRune(r)
	}
	t.cursor++
}

// handleCommentStart processes the start of a comment token.
//
// When inside a composite literal or struct type definition, inserts a comma
// and newline before the comment if needed. Sets comment state flags based on
// whether this is a single-line or multi-line comment.
func (t *tidier) handleCommentStart() {
	if t.context() == ctxCompositeLiteral || t.context() == ctxStructTypeDef {
		last := findLastMeaningfulChar(t.builder.String())
		if last != '{' && last != ',' {
			t.builder.WriteString(",\n")
		}
	}
	_, _ = t.builder.WriteRune(t.runes[t.cursor])

	if t.peekRune() == '/' {
		t.inComment = true
		t.isMultiLineComment = false
	} else if t.peekRune() == '*' {
		t.inComment = true
		t.isMultiLineComment = true
	}
}

// handleOpeningBrace processes an opening brace by checking what comes before
// it and pushing the correct context type onto the stack.
func (t *tidier) handleOpeningBrace() {
	trimmedBuffer := strings.TrimRightFunc(t.builder.String(), unicode.IsSpace)
	isBlock := isFunctionOrControlBlock(trimmedBuffer)

	if strings.HasSuffix(trimmedBuffer, "struct") {
		t.pushContext(ctxStructTypeDef)
	} else if isBlock {
		t.pushContext(ctxBlock)
	} else {
		t.pushContext(ctxCompositeLiteral)
	}

	_, _ = t.builder.WriteRune('{')
	if t.context() != ctxBlock && t.peekNextMeaningfulChar() != '}' {
		_, _ = t.builder.WriteRune('\n')
	}
}

// handleClosingBrace writes a closing brace and adds trailing punctuation
// when needed.
func (t *tidier) handleClosingBrace() {
	ctx := t.context()
	switch ctx {
	case ctxCompositeLiteral:
		lastMeaningful := findLastMeaningfulChar(t.builder.String())
		if lastMeaningful != '{' && lastMeaningful != ',' {
			t.builder.WriteString(",\n")
		}
	case ctxStructTypeDef:
		if findLastMeaningfulChar(t.builder.String()) != '{' {
			t.builder.WriteString("\n")
		}
	default:
	}

	if t.shouldPopContext(ctx) {
		t.popContext()
	}
	_, _ = t.builder.WriteRune('}')
}

// handleOpeningBracket processes a left bracket and updates the parsing state.
func (t *tidier) handleOpeningBracket() {
	if isPrecededByWord(t.builder.String(), "map") {
		t.pushContext(ctxTypeDecl)
	}
	_, _ = t.builder.WriteRune('[')
}

// handleClosingBracket writes a closing bracket and pops the type declaration
// context if one is active.
func (t *tidier) handleClosingBracket() {
	if t.context() == ctxTypeDecl {
		t.popContext()
	}
	_, _ = t.builder.WriteRune(']')
}

// handleComma writes a comma and adds the correct whitespace for the context.
func (t *tidier) handleComma() {
	_, _ = t.builder.WriteRune(',')
	ctx := t.context()
	switch ctx {
	case ctxCompositeLiteral, ctxStructTypeDef:
		_, _ = t.builder.WriteRune('\n')
	case ctxTypeDecl:
		_, _ = t.builder.WriteRune(' ')
	default:
	}
}

// context returns the current tidying context from the stack.
//
// Returns tidyContext which is the top context, or ctxDefault if empty.
func (t *tidier) context() tidyContext {
	if len(t.contextStack) == 0 {
		return ctxDefault
	}
	return t.contextStack[len(t.contextStack)-1]
}

// pushContext adds a context to the tidier's context stack.
//
// Takes ctx (tidyContext) which is the context to push onto the stack.
func (t *tidier) pushContext(ctx tidyContext) {
	t.contextStack = append(t.contextStack, ctx)
}

// popContext removes the most recent context from the stack.
func (t *tidier) popContext() {
	if len(t.contextStack) > 1 {
		t.contextStack = t.contextStack[:len(t.contextStack)-1]
	}
}

// shouldPopContext reports whether the given context should be popped.
//
// Takes ctx (tidyContext) which specifies the context type to check.
//
// Returns bool which is true for composite literals, struct type definitions,
// and blocks.
func (*tidier) shouldPopContext(ctx tidyContext) bool {
	return ctx == ctxCompositeLiteral || ctx == ctxStructTypeDef || ctx == ctxBlock
}

// peekRune returns the next rune without advancing the cursor.
//
// Returns rune which is the next rune, or 0 if at end of input.
func (t *tidier) peekRune() rune {
	if t.cursor+1 < len(t.runes) {
		return t.runes[t.cursor+1]
	}
	return 0
}

// peekNextMeaningfulChar returns the next non-whitespace character without
// advancing the cursor.
//
// Returns rune which is the next meaningful character, or 0 if none remains.
func (t *tidier) peekNextMeaningfulChar() rune {
	for i := t.cursor + 1; i < len(t.runes); i++ {
		r := t.runes[i]
		if !unicode.IsSpace(r) {
			return r
		}
	}
	return 0
}

// tidyGoLiteral formats a compact Go literal string.
//
// It tracks nesting of composite literals and code blocks to add newlines and
// trailing commas in the right places.
//
// Takes compact (string) which is the compact Go literal to format.
//
// Returns string which is the formatted literal with proper newlines and
// trailing commas.
func tidyGoLiteral(compact string) string {
	t := newTidier(compact)
	return t.run()
}

// newTidier creates a tidier to process the given compact string.
//
// Takes compact (string) which is the input text to tidy.
//
// Returns *tidier which is ready for use.
func newTidier(compact string) *tidier {
	return &tidier{
		builder:            strings.Builder{},
		runes:              []rune(compact),
		contextStack:       []tidyContext{ctxDefault},
		cursor:             0,
		stringDelim:        0,
		inString:           false,
		inComment:          false,
		isMultiLineComment: false,
	}
}

// isFunctionOrControlBlock checks if an opening brace belongs to a code block
// such as if, for, or func, rather than a composite literal. It uses a
// two-stage check to tell these cases apart.
//
// Takes s (string) which is the source code segment to check.
//
// Returns bool which is true if the brace belongs to a function or control
// block, false if it belongs to a composite literal.
func isFunctionOrControlBlock(s string) bool {
	lastBrace := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '{' || s[i] == '}' {
			lastBrace = i
			break
		}
	}
	segmentFromBrace := strings.TrimSpace(s[lastBrace+1:])
	if strings.HasPrefix(segmentFromBrace, "for") {
		return true
	}

	lastSeparator := -1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '{' || s[i] == '}' || s[i] == ';' || s[i] == '\n' {
			lastSeparator = i
			break
		}
	}
	segment := strings.TrimSpace(s[lastSeparator+1:])

	return hasControlKeywordPrefix(segment)
}

// hasControlKeywordPrefix checks if a text segment contains or starts with a
// Go control flow keyword.
//
// Takes segment (string) which is the text to check.
//
// Returns bool which is true if the segment contains "func" or starts with
// "if", "for", "else", "switch", "case", or "default".
func hasControlKeywordPrefix(segment string) bool {
	if strings.Contains(segment, "func") {
		return true
	}
	return strings.HasPrefix(segment, "if") ||
		strings.HasPrefix(segment, "for") ||
		strings.HasPrefix(segment, "else") ||
		strings.HasPrefix(segment, "switch") ||
		strings.HasPrefix(segment, "case") ||
		strings.HasPrefix(segment, "default")
}

// isPrecededByWord checks whether a string ends with the given word after
// removing trailing whitespace.
//
// Takes s (string) which is the text to check.
// Takes word (string) which is the suffix to look for.
//
// Returns bool which is true if s ends with word after trimming whitespace.
func isPrecededByWord(s string, word string) bool {
	trimmed := strings.TrimRightFunc(s, unicode.IsSpace)
	return strings.HasSuffix(trimmed, word)
}

// findLastMeaningfulChar finds the last character in a string that is not
// whitespace or part of a comment.
//
// Takes s (string) which is the input string to search.
//
// Returns rune which is the last meaningful character, or 0 if none is found.
func findLastMeaningfulChar(s string) rune {
	for len(s) > 0 {
		s = strings.TrimRightFunc(s, unicode.IsSpace)
		if s == "" {
			return 0
		}

		if strings.HasSuffix(s, "*/") {
			if startIndex := strings.LastIndex(s, "/*"); startIndex != -1 {
				s = s[:startIndex]
				continue
			}
		}

		if startIndex := strings.LastIndex(s, "//"); startIndex != -1 {
			if lastNewline := strings.LastIndex(s, "\n"); startIndex > lastNewline {
				s = s[:startIndex]
				continue
			}
		}

		runes := []rune(s)
		return runes[len(runes)-1]
	}
	return 0
}
