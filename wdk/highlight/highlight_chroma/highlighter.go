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

package highlight_chroma

import (
	"bytes"
	"context"
	"hash/maphash"
	"html"
	"sync"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"piko.sh/piko/internal/highlight/highlight_domain"
	"piko.sh/piko/wdk/logger"
)

const (
	// defaultTabWidth is the number of spaces used for tab characters.
	defaultTabWidth = 4

	// defaultBufferSize is the initial buffer capacity for formatter output.
	defaultBufferSize = 4096
)

// Highlighter provides syntax highlighting using the Chroma library.
// It implements the highlight_domain.Highlighter interface.
type Highlighter struct {
	// style is the colour scheme used for syntax highlighting.
	style *chroma.Style

	// formatter converts tokenised code into HTML output.
	formatter *chromahtml.Formatter

	// lexerCache caches coalesced lexers by language name, avoiding repeated
	// lexers.Get + chroma.Coalesce calls.
	lexerCache sync.Map

	// bufPool pools bytes.Buffer instances for formatter output.
	bufPool sync.Pool

	// resultCache caches highlight results by a hash of (language, code),
	// avoiding redundant tokenisation and formatting for identical inputs.
	resultCache sync.Map

	// hashSeed is used for consistent hashing of cache keys.
	hashSeed maphash.Seed
}

var _ highlight_domain.Highlighter = (*Highlighter)(nil)

// Config holds configuration options for the Chroma highlighter.
type Config struct {
	// Style is the Chroma style name (for example "dracula", "monokai", or
	// a GitHub-themed style). Defaults to "dracula" if empty.
	Style string

	// WithClasses outputs CSS class names instead of inline styles. When true
	// (the default), you must include appropriate CSS in your page.
	WithClasses bool

	// WithLineNumbers enables line numbers in the output. Defaults to false.
	WithLineNumbers bool

	// LineNumbersInTable renders line numbers in a separate table cell when
	// WithLineNumbers is true. Defaults to false.
	LineNumbersInTable bool

	// TabWidth sets the number of spaces for tab characters. Defaults to 4.
	TabWidth int
}

// NewChromaHighlighter creates a new Highlighter with the given settings.
//
// Takes config (Config) which sets the highlighting style and format options.
// Zero values are replaced with sensible defaults.
//
// Returns *Highlighter which is set up and ready to use.
func NewChromaHighlighter(config Config) *Highlighter {
	_, l := logger.From(context.Background(), log)
	if config.Style == "" {
		config.Style = "dracula"
	}
	if config.TabWidth == 0 {
		config.TabWidth = defaultTabWidth
	}

	style := styles.Get(config.Style)
	if style == nil {
		style = styles.Fallback
		l.Warn("Unknown Chroma style, using fallback",
			logger.String("requested_style", config.Style))
	}

	opts := []chromahtml.Option{
		chromahtml.WithClasses(config.WithClasses),
		chromahtml.WithLineNumbers(config.WithLineNumbers),
		chromahtml.LineNumbersInTable(config.LineNumbersInTable),
		chromahtml.TabWidth(config.TabWidth),
	}

	formatter := chromahtml.New(opts...)

	l.Internal("Created Chroma highlighter",
		logger.String("style", config.Style),
		logger.Bool("with_classes", config.WithClasses),
		logger.Bool("line_numbers", config.WithLineNumbers))

	return &Highlighter{
		style:     style,
		formatter: formatter,
		bufPool: sync.Pool{
			New: func() any { return bytes.NewBuffer(make([]byte, 0, defaultBufferSize)) },
		},
		hashSeed: maphash.MakeSeed(),
	}
}

// Highlight implements the Highlighter interface.
//
// Takes code (string) which is the source code to highlight.
// Takes language (string) which specifies the programming language.
//
// Returns string which contains the highlighted HTML output, or a plain code
// block if highlighting fails.
func (h *Highlighter) Highlight(code, language string) string {
	key := h.cacheKey(language, code)
	if cached, ok := h.resultCache.Load(key); ok {
		if s, castOK := cached.(string); castOK {
			return s
		}
	}

	lexer := h.getLexer(language)
	if lexer == nil {
		return plainCodeBlock(code, language)
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		_, l := logger.From(context.Background(), log)
		l.Trace("Tokenisation failed, falling back to plain",
			logger.String("language", language),
			logger.Error(err))
		return plainCodeBlock(code, language)
	}

	buf, ok := h.bufPool.Get().(*bytes.Buffer)
	if !ok || buf == nil {
		buf = bytes.NewBuffer(make([]byte, 0, defaultBufferSize))
	}
	buf.Reset()
	defer h.bufPool.Put(buf)

	if err := h.formatter.Format(buf, h.style, iterator); err != nil {
		_, l := logger.From(context.Background(), log)
		l.Trace("Formatting failed, falling back to plain",
			logger.String("language", language),
			logger.Error(err))
		return plainCodeBlock(code, language)
	}

	result := buf.String()
	h.resultCache.Store(key, result)
	return result
}

// getLexer returns a cached, coalesced lexer for the given language.
//
// Takes language (string) which specifies the programming language to
// look up.
//
// Returns chroma.Lexer which is the coalesced lexer, or nil if no
// lexer is available.
func (h *Highlighter) getLexer(language string) chroma.Lexer {
	if cached, ok := h.lexerCache.Load(language); ok {
		if l, castOK := cached.(chroma.Lexer); castOK {
			return l
		}
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)
	h.lexerCache.Store(language, lexer)
	return lexer
}

// cacheKey returns a hash key for a (language, code) pair.
//
// Takes language (string) which identifies the programming language.
// Takes code (string) which is the source code to hash.
//
// Returns uint64 which is the combined hash of language and code.
func (h *Highlighter) cacheKey(language, code string) uint64 {
	var hash maphash.Hash
	hash.SetSeed(h.hashSeed)
	_, _ = hash.WriteString(language)
	_, _ = hash.Write([]byte{0})
	_, _ = hash.WriteString(code)
	return hash.Sum64()
}

// DefaultConfig returns a Config with sensible default values.
//
// Returns Config which contains the default syntax highlighting settings.
func DefaultConfig() Config {
	return Config{
		Style:              "dracula",
		WithClasses:        true,
		WithLineNumbers:    false,
		LineNumbersInTable: false,
		TabWidth:           defaultTabWidth,
	}
}

// plainCodeBlock returns a simple HTML code block without highlighting.
//
// Takes code (string) which is the source code to display.
// Takes language (string) which sets the language class attribute, or empty
// for no language class.
//
// Returns string which is the HTML-escaped code wrapped in pre and code tags.
func plainCodeBlock(code, language string) string {
	var buffer bytes.Buffer
	_, _ = buffer.WriteString(`<pre><code`)
	if language != "" {
		_, _ = buffer.WriteString(` class="language-`)
		_, _ = buffer.WriteString(html.EscapeString(language))
		_, _ = buffer.WriteString(`"`)
	}
	_, _ = buffer.WriteString(`>`)
	_, _ = buffer.WriteString(html.EscapeString(code))
	_, _ = buffer.WriteString(`</code></pre>`)
	return buffer.String()
}
