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

package layouter_domain

import (
	"strings"
	"unicode"
)

// tidyContext tracks the kind of brace-delimited block being processed.
type tidyContext int

const (
	// ctxDefault is the default context outside any block.
	ctxDefault tidyContext = iota

	// ctxCompositeLiteral is the context inside a composite
	// literal.
	ctxCompositeLiteral

	// ctxBlock is the context inside a function or
	// control-flow block.
	ctxBlock
)

// tidier tracks state while reformatting compact Go source
// into readable multi-line form.
type tidier struct {
	// builder accumulates the formatted output.
	builder strings.Builder

	// runes is the input source as a rune slice.
	runes []rune

	// contextStack tracks nested brace contexts.
	contextStack []tidyContext

	// cursor is the current read position in runes.
	cursor int

	// stringDelim is the delimiter that opened the current
	// string literal.
	stringDelim rune

	// inString is true when the cursor is inside a string
	// literal.
	inString bool

	// inComment is true when the cursor is inside a comment.
	inComment bool

	// isMultiLineComment is true when the current comment
	// uses block style.
	isMultiLineComment bool
}

// newTidier creates a tidier initialised with the given
// compact source string.
//
// Takes compact (string) which is the single-line Go
// source to reformat.
//
// Returns *tidier which is the initialised tidier.
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

// tidyGoLiteral formats a compact Go literal string by
// adding newlines and trailing commas inside composite
// literals.
//
// Takes compact (string) which is the single-line Go
// source to reformat.
//
// Returns the reformatted multi-line string.
func tidyGoLiteral(compact string) string {
	return newTidier(compact).run()
}

// run processes all runes and returns the formatted
// output.
//
// Returns string which is the reformatted source text.
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

// processString consumes one rune inside a string literal.
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

// processComment consumes one rune inside a comment.
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

// processCode dispatches the current rune to the
// appropriate handler based on its character.
func (t *tidier) processCode() {
	r := t.runes[t.cursor]
	nextRune := t.peekRune()

	switch r {
	case '"', '\'', '`':
		t.inString = true
		t.stringDelim = r
		_, _ = t.builder.WriteRune(r)
	case '/':
		if nextRune == '/' || nextRune == '*' {
			t.handleCommentStart()
		} else {
			_, _ = t.builder.WriteRune(r)
		}
	case '{':
		t.handleOpeningBrace()
	case '}':
		t.handleClosingBrace()
	case ',':
		t.handleComma()
	default:
		_, _ = t.builder.WriteRune(r)
	}
	t.cursor++
}

// handleCommentStart processes a slash that begins a
// comment sequence.
func (t *tidier) handleCommentStart() {
	if t.context() == ctxCompositeLiteral {
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

// handleOpeningBrace pushes the appropriate context and
// emits a newline for composite literals.
func (t *tidier) handleOpeningBrace() {
	trimmedBuffer := strings.TrimRightFunc(t.builder.String(), unicode.IsSpace)
	isBlock := isFunctionOrControlBlock(trimmedBuffer)

	if isBlock {
		t.pushContext(ctxBlock)
	} else {
		t.pushContext(ctxCompositeLiteral)
	}

	_, _ = t.builder.WriteRune('{')
	if t.context() != ctxBlock && t.peekNextMeaningfulChar() != '}' {
		_, _ = t.builder.WriteRune('\n')
	}
}

// handleClosingBrace inserts a trailing comma for
// composite literals and pops the context stack.
func (t *tidier) handleClosingBrace() {
	ctx := t.context()
	if ctx == ctxCompositeLiteral {
		lastMeaningful := findLastMeaningfulChar(t.builder.String())
		if lastMeaningful != '{' && lastMeaningful != ',' {
			t.builder.WriteString(",\n")
		}
	}

	if t.shouldPopContext(ctx) {
		t.popContext()
	}
	_, _ = t.builder.WriteRune('}')
}

// handleComma writes a comma and appends a newline when
// inside a composite literal.
func (t *tidier) handleComma() {
	_, _ = t.builder.WriteRune(',')
	if t.context() == ctxCompositeLiteral {
		_, _ = t.builder.WriteRune('\n')
	}
}

// context returns the current brace context from the
// top of the stack.
//
// Returns tidyContext which is the current context.
func (t *tidier) context() tidyContext {
	if len(t.contextStack) == 0 {
		return ctxDefault
	}
	return t.contextStack[len(t.contextStack)-1]
}

// pushContext appends a new context to the stack.
//
// Takes ctx (tidyContext) which is the context to push.
func (t *tidier) pushContext(ctx tidyContext) {
	t.contextStack = append(t.contextStack, ctx)
}

// popContext removes the topmost context from the stack.
func (t *tidier) popContext() {
	if len(t.contextStack) > 1 {
		t.contextStack = t.contextStack[:len(t.contextStack)-1]
	}
}

// shouldPopContext reports whether the given context
// should be removed when a closing brace is encountered.
//
// Takes ctx (tidyContext) which is the context to check.
//
// Returns bool which is true if the context should be
// popped.
func (*tidier) shouldPopContext(ctx tidyContext) bool {
	return ctx == ctxCompositeLiteral || ctx == ctxBlock
}

// peekRune returns the rune after the cursor without
// advancing.
//
// Returns rune which is the next rune, or 0 if at the
// end.
func (t *tidier) peekRune() rune {
	if t.cursor+1 < len(t.runes) {
		return t.runes[t.cursor+1]
	}
	return 0
}

// peekNextMeaningfulChar returns the first non-whitespace
// rune after the cursor.
//
// Returns rune which is the next non-whitespace rune, or
// 0 if none remains.
func (t *tidier) peekNextMeaningfulChar() rune {
	for i := t.cursor + 1; i < len(t.runes); i++ {
		r := t.runes[i]
		if !unicode.IsSpace(r) {
			return r
		}
	}
	return 0
}

// isFunctionOrControlBlock checks whether an opening
// brace belongs to a function or control-flow block
// rather than a composite literal.
//
// Takes s (string) which is the source text preceding
// the brace.
//
// Returns true when the brace opens a function or
// control-flow block.
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

// hasControlKeywordPrefix reports whether the segment
// starts with a Go control keyword or contains func.
//
// Takes segment (string) which is the text to inspect.
//
// Returns bool which is true if a control keyword is
// found.
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

// findLastMeaningfulChar returns the last character that
// is not whitespace or part of a comment.
//
// Takes s (string) which is the source text to inspect.
//
// Returns the last meaningful rune, or 0 if none exists.
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
