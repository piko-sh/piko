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

package config_domain

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"

	"piko.sh/piko/wdk/safedisk"
)

// tokenType identifies the kind of token in a configuration value.
type tokenType int

const (
	// tokenIllegal marks a character that the lexer does not recognise.
	tokenIllegal tokenType = iota

	// tokenEOF signals the end of the input stream.
	tokenEOF

	// tokenNewline represents a newline character in the lexer output.
	tokenNewline

	// tokenComment is the token type for comment lines.
	tokenComment

	// tokenIdentifier represents a key name or unquoted value in a dotenv file.
	tokenIdentifier

	// tokenEquals represents the equals sign token used in key-value assignments.
	tokenEquals

	// tokenQuotedValue represents a quoted string value in the dotenv file.
	tokenQuotedValue
)

const (
	// charEOF is the marker value that shows the input has ended.
	charEOF = 0

	// charNewline is the newline character that marks the end of a line.
	charNewline = '\n'

	// charEquals is the equals sign used for assignment in dotenv files.
	charEquals = '='

	// charComment is the hash character that marks the start of a comment.
	charComment = '#'

	// charSpace is the ASCII space character.
	charSpace = ' '

	// charTab is the horizontal tab character.
	charTab = '\t'

	// charCR is the carriage return character.
	charCR = '\r'

	// quoteSingle is the single quote character used to mark string values.
	quoteSingle = '\''

	// quoteDouble is the double quote character used to mark quoted values.
	quoteDouble = '"'

	// defaultDotEnvFilename is the default filename for loading environment
	// variables from a dotenv file.
	defaultDotEnvFilename = ".env"

	// escapeNewline is the character used to mark a newline escape sequence.
	escapeNewline = 'n'

	// escapeDollar is the dollar sign character used in escape sequences.
	escapeDollar = '$'

	// escapeQuote is the double quote character used in escape sequences.
	escapeQuote = '"'

	// escapeBackslash is the backslash escape sequence character.
	escapeBackslash = '\\'
)

// token represents a single piece of text from the configuration file parser.
type token struct {
	// Literal is the raw text of the token as it appears in the input.
	Literal string

	// Type identifies the kind of token.
	Type tokenType

	// IsSingleQuoted indicates whether the token was wrapped in single quotes.
	IsSingleQuoted bool
}

// lexer breaks configuration domain input into tokens for parsing.
type lexer struct {
	// input holds the source string as runes for tokenisation.
	input []rune

	// position is the current byte position in the input string.
	position int

	// readPosition is the position of the next character to read from input.
	readPosition int

	// character is the current character being examined.
	character rune
}

// NextToken returns the next token from the input stream.
//
// Returns token which is the next lexical token parsed from the input.
func (l *lexer) NextToken() token {
	l.skipWhitespace()

	switch l.character {
	case charEquals:
		l.readChar()
		return token{Literal: "", Type: tokenEquals, IsSingleQuoted: false}
	case charComment:
		return l.readComment()
	case charNewline:
		l.readChar()
		return token{Literal: "", Type: tokenNewline, IsSingleQuoted: false}
	case charEOF:
		return token{Literal: "", Type: tokenEOF, IsSingleQuoted: false}
	case quoteSingle, quoteDouble:
		return l.readQuotedValue()
	default:
		if isKeyCharacter(l.character) {
			return token{Literal: l.readIdentifier(), Type: tokenIdentifier, IsSingleQuoted: false}
		}
		l.readChar()
		return token{Literal: "", Type: tokenIllegal, IsSingleQuoted: false}
	}
}

// readChar moves the lexer forward by one character.
func (l *lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.character = charEOF
	} else {
		l.character = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

// skipWhitespace moves the lexer forward past any space, tab, or carriage
// return characters.
func (l *lexer) skipWhitespace() {
	for l.character == charSpace || l.character == charTab || l.character == charCR {
		l.readChar()
	}
}

// readIdentifier reads a sequence of key characters from the input.
//
// Returns string which is the identifier from the start position to the
// current position.
func (l *lexer) readIdentifier() string {
	startPosition := l.position
	for isKeyCharacter(l.character) {
		l.readChar()
	}
	return string(l.input[startPosition:l.position])
}

// readComment reads a comment from the current position until end of line.
//
// Returns token which holds the comment text with type tokenComment.
func (l *lexer) readComment() token {
	startPosition := l.position
	for l.character != charNewline && l.character != charEOF {
		l.readChar()
	}
	return token{Literal: string(l.input[startPosition:l.position]), Type: tokenComment, IsSingleQuoted: false}
}

// readQuotedValue reads a quoted string and returns it as a token.
//
// Returns token which holds the string content without the surrounding quotes.
func (l *lexer) readQuotedValue() token {
	quoteType := l.character
	isSingleQuoted := quoteType == quoteSingle
	l.readChar()

	var buffer bytes.Buffer
	for {
		if l.character == charEOF {
			return l.handleUnterminatedString(buffer.String(), isSingleQuoted)
		}

		if l.shouldProcessEscapeSequence(quoteType) {
			l.processEscapeSequence(&buffer)
		} else if l.character == quoteType {
			l.readChar()
			break
		} else {
			_, _ = buffer.WriteRune(l.character)
		}
		l.readChar()
	}
	return token{Type: tokenQuotedValue, Literal: buffer.String(), IsSingleQuoted: isSingleQuoted}
}

// handleUnterminatedString handles EOF before a closing quote.
//
// Takes value (string) which is the string content collected before EOF.
// Takes isSingleQuoted (bool) which indicates if the string used single quotes.
//
// Returns token which contains the unterminated string as a quoted value.
func (*lexer) handleUnterminatedString(value string, isSingleQuoted bool) token {
	return token{Type: tokenQuotedValue, Literal: value, IsSingleQuoted: isSingleQuoted}
}

// shouldProcessEscapeSequence checks if an escape sequence should be handled.
//
// Takes quoteType (rune) which specifies the quote character being processed.
//
// Returns bool which is true when the current character is a backslash and
// the quote type is a double quote.
func (l *lexer) shouldProcessEscapeSequence(quoteType rune) bool {
	return l.character == '\\' && quoteType == quoteDouble
}

// processEscapeSequence handles escape sequences in double-quoted strings.
//
// Takes buffer (*bytes.Buffer) which receives the decoded character output.
func (l *lexer) processEscapeSequence(buffer *bytes.Buffer) {
	l.readChar()
	switch l.character {
	case escapeNewline:
		_ = buffer.WriteByte('\n')
	case escapeDollar, escapeQuote, escapeBackslash:
		_, _ = buffer.WriteRune(l.character)
	default:
		_ = buffer.WriteByte('\\')
		_, _ = buffer.WriteRune(l.character)
	}
}

// parseResult holds a raw parsed value before variable expansion.
type parseResult struct {
	// value holds the raw string content before variable expansion.
	value string

	// isSingleQuoted indicates the value was in single quotes and should not have
	// variable expansion.
	isSingleQuoted bool
}

// parser turns configuration tokens into structured parse results.
type parser struct {
	// l is the lexer that splits input into tokens for parsing.
	l *lexer

	// rawValues stores parsed key-value pairs before variable expansion.
	rawValues map[string]parseResult
}

// parse processes all tokens and builds the result map.
func (p *parser) parse() {
	for {
		token := p.l.NextToken()

		if token.Type == tokenIdentifier && token.Literal == "export" {
			token = p.l.NextToken()
		}

		if token.Type == tokenEOF {
			break
		}
		if token.Type == tokenIdentifier {
			p.parseAssignment(token.Literal)
		}
	}
}

// parseAssignment parses a KEY=VALUE assignment from the input.
//
// Takes key (string) which is the variable name used to store the parsed value.
func (p *parser) parseAssignment(key string) {
	if p.l.NextToken().Type != tokenEquals {
		p.consumeLine()
		return
	}

	p.l.skipWhitespace()

	var valToken token

	if p.l.character == quoteSingle || p.l.character == quoteDouble {
		valToken = p.l.readQuotedValue()
	} else {
		startPosition := p.l.position
		for !p.isAtLineEnd() {
			p.l.readChar()
		}
		unquotedValue := string(p.l.input[startPosition:p.l.position])
		valToken = token{
			Type:           tokenIdentifier,
			Literal:        strings.TrimRight(unquotedValue, " \t\r"),
			IsSingleQuoted: false,
		}
	}

	p.rawValues[key] = parseResult{
		value:          valToken.Literal,
		isSingleQuoted: valToken.IsSingleQuoted,
	}
}

// isAtLineEnd checks if the parser has reached the end of the current line.
//
// Returns bool which is true when the current character is a newline, comment
// start, or end of file.
func (p *parser) isAtLineEnd() bool {
	return p.l.character == charNewline || p.l.character == charComment || p.l.character == charEOF
}

// consumeLine discards tokens until it reaches a newline or end of file.
func (p *parser) consumeLine() {
	for {
		token := p.l.NextToken()
		if token.Type == tokenNewline || token.Type == tokenEOF {
			break
		}
	}
}

// expandVariables performs the second pass, resolving variable substitutions.
//
// Returns map[string]string which contains the expanded key-value pairs.
func (p *parser) expandVariables() map[string]string {
	expandedMap := make(map[string]string, len(p.rawValues))

	for k, entry := range p.rawValues {
		if entry.isSingleQuoted {
			expandedMap[k] = entry.value
		}
	}

	for k, entry := range p.rawValues {
		if !entry.isSingleQuoted {
			expandedMap[k] = p.customExpand(entry.value, k, expandedMap)
		}
	}

	return expandedMap
}

// customExpand handles variable expansion for double-quoted strings.
// It expands variables in ${VAR} and $VAR formats, preserving literal $
// characters.
//
// Takes value (string) which is the string containing variables to expand.
// Takes currentKey (string) which identifies the key being processed.
// Takes expandedMap (map[string]string) which caches already expanded values.
//
// Returns string which is the input with all variables expanded.
func (p *parser) customExpand(value string, currentKey string, expandedMap map[string]string) string {
	result := strings.Builder{}
	i := 0
	for i < len(value) {
		if !p.isVariableStart(value, i) {
			_ = result.WriteByte(value[i])
			i++
			continue
		}

		expanded, consumed := p.tryExpandVariable(value, i, currentKey, expandedMap)
		if consumed > 0 {
			_, _ = result.WriteString(expanded)
			i += consumed
		} else {
			_ = result.WriteByte(value[i])
			i++
		}
	}
	return result.String()
}

// isVariableStart checks if position i marks a potential variable reference.
//
// Takes value (string) which is the string to examine.
// Takes i (int) which is the position to check.
//
// Returns bool which is true if a dollar sign exists at position i and there
// is at least one more character after it.
func (*parser) isVariableStart(value string, i int) bool {
	return value[i] == '$' && i+1 < len(value)
}

// tryExpandVariable expands a variable at position i in the value string.
// It tries the ${VAR} format first, then falls back to the $VAR format.
//
// Takes value (string) which contains the text with variables to expand.
// Takes i (int) which specifies the position of the dollar sign.
// Takes currentKey (string) which identifies the key being expanded.
// Takes expandedMap (map[string]string) which stores already expanded values.
//
// Returns string which contains the expanded variable value, or empty if none.
// Returns int which indicates the number of characters used, or 0 if no valid
// variable pattern is found.
func (p *parser) tryExpandVariable(value string, i int, currentKey string, expandedMap map[string]string) (string, int) {
	if expanded, consumed := p.tryExpandBracedVariable(value, i, currentKey, expandedMap); consumed > 0 {
		return expanded, consumed
	}

	if expanded, consumed := p.tryExpandBareVariable(value, i, currentKey, expandedMap); consumed > 0 {
		return expanded, consumed
	}

	return "", 0
}

// tryExpandBracedVariable handles ${VAR} format variable expansion.
//
// Takes value (string) which is the string containing the variable reference.
// Takes i (int) which is the position of the dollar sign in value.
// Takes currentKey (string) which is the key being expanded, used to detect
// cycles.
// Takes expandedMap (map[string]string) which holds variables that have already
// been expanded.
//
// Returns string which is the expanded variable value, or empty if this is not
// a braced variable.
// Returns int which is the number of characters consumed, or zero if this is
// not a braced variable.
func (p *parser) tryExpandBracedVariable(value string, i int, currentKey string, expandedMap map[string]string) (string, int) {
	if value[i+1] != '{' {
		return "", 0
	}

	closingBrace := p.findClosingBrace(value, i+2)
	if closingBrace < 0 {
		return "", 0
	}

	varName := value[i+2 : closingBrace]
	expanded := p.expandVariable(varName, currentKey, expandedMap)
	consumed := closingBrace - i + 1
	return expanded, consumed
}

// findClosingBrace finds the position of '}' starting from position.
//
// Takes value (string) which is the string to search within.
// Takes position (int) which is the starting position for the search.
//
// Returns int which is the position of the closing brace, or -1 if not found.
func (*parser) findClosingBrace(value string, position int) int {
	for j := position; j < len(value); j++ {
		if value[j] == '}' {
			return j
		}
	}
	return -1
}

// tryExpandBareVariable handles $VAR format expansion.
//
// Takes value (string) which contains the text being parsed.
// Takes i (int) which is the position of the dollar sign in value.
// Takes currentKey (string) which identifies the variable being expanded.
// Takes expandedMap (map[string]string) which tracks already expanded values.
//
// Returns string which is the expanded variable value, or empty if not valid.
// Returns int which is the number of characters used, or zero if not valid.
func (p *parser) tryExpandBareVariable(value string, i int, currentKey string, expandedMap map[string]string) (string, int) {
	if !isValidVarStart(value[i+1]) {
		return "", 0
	}

	j := i + 1
	for j < len(value) && isValidVarChar(value[j]) {
		j++
	}

	varName := value[i+1 : j]
	if !p.shouldExpandBareVariable(varName) {
		return "", 0
	}

	expanded := p.expandVariable(varName, currentKey, expandedMap)
	consumed := j - i
	return expanded, consumed
}

// shouldExpandBareVariable determines if a bare variable name should be
// expanded.
//
// Takes varName (string) which is the variable name to check.
//
// Returns bool which is true if the variable name is longer than one character
// or is a common single-letter variable.
func (*parser) shouldExpandBareVariable(varName string) bool {
	return len(varName) > 1 || isCommonSingleLetterVar(varName)
}

// expandVariable finds and expands a variable by name.
//
// The method first checks the parser's raw values, then falls back to the
// shell environment. It handles self-references by returning an empty string
// and returns the literal "${}" for empty variable names.
//
// Takes varName (string) which is the name of the variable to expand.
// Takes currentKey (string) which is the key being processed, used to detect
// self-references.
// Takes expandedMap (map[string]string) which caches expanded values to avoid
// repeated work and detect cycles.
//
// Returns string which is the expanded value.
func (p *parser) expandVariable(varName string, currentKey string, expandedMap map[string]string) string {
	if varName == "" {
		return "${}"
	}

	if rawEntry, ok := p.rawValues[varName]; ok {
		if varName == currentKey {
			return ""
		}
		if expandedValue, exists := expandedMap[varName]; exists {
			return expandedValue
		}
		if !rawEntry.isSingleQuoted {
			expandedMap[varName] = p.customExpand(rawEntry.value, varName, expandedMap)
		} else {
			expandedMap[varName] = rawEntry.value
		}
		return expandedMap[varName]
	}
	return os.Getenv(varName)
}

var (
	dotenvMap map[string]string

	dotenvOnce sync.Once

	dotenvParseErr error

	dotenvSandbox safedisk.Sandbox
)

// dotEnvLookuper implements lookuper to read values from .env files.
type dotEnvLookuper struct{}

// Lookup retrieves a value from the parsed .env file by key.
//
// Takes key (string) which specifies the environment variable name to look up.
//
// Returns string which is the value for the key, or empty if not found.
// Returns bool which is true if the key exists, false otherwise.
func (dotEnvLookuper) Lookup(key string) (string, bool) {
	dotenvOnce.Do(func() {
		filename := defaultDotEnvFilename
		if dotenvSandbox != nil {
			data, err := dotenvSandbox.ReadFile(filename)
			if err != nil {
				dotenvParseErr = err
				return
			}
			dotenvMap, dotenvParseErr = parseDotEnvStream(bytes.NewReader(data))
			return
		}
		file, err := os.Open(filename) //nolint:gosec // well-known config filename
		if err != nil {
			dotenvParseErr = err
			return
		}
		defer func() { _ = file.Close() }()

		dotenvMap, dotenvParseErr = parseDotEnvStream(file)
	})

	if dotenvParseErr != nil && !errors.Is(dotenvParseErr, os.ErrNotExist) {
		return "", false
	}

	value, ok := dotenvMap[key]
	return value, ok
}

// applyDotEnv applies environment variables from dotenv files to the struct.
//
// Takes ptr (any) which is a pointer to the struct to populate.
// Takes ctx (*LoadContext) which provides the loading context and state.
//
// Returns error when walking the struct fields fails.
func (l *Loader) applyDotEnv(ptr any, ctx *LoadContext) error {
	lookuper := dotEnvLookuper{}
	processor := func(field *reflect.StructField, value reflect.Value, _, _ string) error {
		return processEnv(field, value, l.opts.EnvPrefix, lookuper)
	}
	state := &walkState{
		processor: processor,
		ctx:       ctx,
		keyPrefix: "",
		source:    "dotenv",
	}
	return l.walk(reflect.ValueOf(ptr), state)
}

// SetDotEnvSandbox configures a sandbox for reading the .env file, causing
// the Lookup method to use it instead of os.Open.
//
// Takes sandbox (safedisk.Sandbox) which provides the file system sandbox
// for .env file access. Primarily useful for testing with MockSandbox.
func SetDotEnvSandbox(sandbox safedisk.Sandbox) {
	dotenvSandbox = sandbox
}

// ResetDotEnvCache clears the cached .env file data and sandbox.
// This is mainly for testing to ensure tests do not affect each other.
func ResetDotEnvCache() {
	dotenvOnce = sync.Once{}
	dotenvMap = nil
	dotenvParseErr = nil
	dotenvSandbox = nil
}

// newLexer creates a new lexer for splitting the given input string into
// tokens.
//
// Takes input (string) which contains the text to be tokenised.
//
// Returns *lexer which is ready to produce tokens from the input.
func newLexer(input string) *lexer {
	l := &lexer{
		input:        []rune(input),
		position:     0,
		readPosition: 0,
		character:    0,
	}
	l.readChar()
	return l
}

// isKeyCharacter reports whether the given rune is valid within a key name.
//
// Takes character (rune) which is the character to check.
//
// Returns bool which is true if character is a letter, digit, or underscore.
func isKeyCharacter(character rune) bool {
	return (character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '_'
}

// newParser creates a parser for processing .env file tokens.
//
// Takes l (*lexer) which provides the token stream to parse.
//
// Returns *parser which is ready to parse tokens.
func newParser(l *lexer) *parser {
	const estimatedEnvVars = 32
	return &parser{
		l:         l,
		rawValues: make(map[string]parseResult, estimatedEnvVars),
	}
}

// isValidVarStart checks if a character can start a variable name.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if the character is a letter or underscore.
func isValidVarStart(character byte) bool {
	return (character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z') || character == '_'
}

// isValidVarChar checks if a character can appear in a variable name.
//
// Takes character (byte) which is the character to check.
//
// Returns bool which is true if the character is valid for a variable name.
func isValidVarChar(character byte) bool {
	return isValidVarStart(character) || (character >= '0' && character <= '9')
}

// isCommonSingleLetterVar checks if a variable name is a common environment
// variable that should be expanded rather than treated as a literal string.
//
// Takes varName (string) which is the variable name to check.
//
// Returns bool which is true if varName is PATH, HOME, USER, LANG, or TERM.
func isCommonSingleLetterVar(varName string) bool {
	switch varName {
	case "PATH", "HOME", "USER", "LANG", "TERM":
		return true
	default:
		return false
	}
}

// parseDotEnvStream parses environment variables from a stream and returns
// the expanded key-value pairs.
//
// Takes r (io.Reader) which provides the .env file content to parse.
//
// Returns map[string]string which contains the parsed and expanded variables.
// Returns error when the stream cannot be read.
func parseDotEnvStream(r io.Reader) (map[string]string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read .env stream: %w", err)
	}

	dotEnvLexer := newLexer(string(content))
	doEnvParser := newParser(dotEnvLexer)

	doEnvParser.parse()
	expandedMap := doEnvParser.expandVariables()

	return expandedMap, nil
}
