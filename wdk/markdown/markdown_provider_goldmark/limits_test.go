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

package markdown_provider_goldmark

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"

	"piko.sh/piko/internal/markdown/markdown_domain"
)

func TestParser_Parse_InputLimits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		input   string
		options []Option
	}{
		{name: "a document within the limits parses", input: "# Title\n\n> quoted\n- item\n"},
		{name: "a document at the size limit parses", input: strings.Repeat("a", 10), options: []Option{WithMaxInputSize(10)}},
		{name: "a document over the size limit is rejected", input: strings.Repeat("a", 11), options: []Option{WithMaxInputSize(10)}, wantErr: ErrInputTooLarge},
		{name: "a document over the default size is rejected", input: strings.Repeat("a", defaultMaxInputSize+1), wantErr: ErrInputTooLarge},
		{name: "a size below one keeps the default", input: strings.Repeat("a", 100), options: []Option{WithMaxInputSize(0)}},
		{name: "deeply nested blockquotes are rejected", input: strings.Repeat(">", 100_000) + " deep", wantErr: ErrNestingTooDeep},
		{name: "deeply nested bullets are rejected", input: strings.Repeat("- ", 100_000) + "deep", wantErr: ErrNestingTooDeep},
		{name: "deeply nested ordered items are rejected", input: strings.Repeat("1. ", 2_000) + "deep", wantErr: ErrNestingTooDeep},
		{name: "nesting at the limit parses", input: "> > > x", options: []Option{WithMaxNestingDepth(3)}},
		{name: "nesting past the limit on a later line is rejected", input: "fine\n> > > > x", options: []Option{WithMaxNestingDepth(3)}, wantErr: ErrNestingTooDeep},
		{name: "a depth below one keeps the default", input: "> > > > x", options: []Option{WithMaxNestingDepth(-1)}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, frontmatter, err := NewParser(tc.options...).Parse(context.Background(), []byte(tc.input))

			if tc.wantErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, doc)
				assert.NotNil(t, frontmatter)
				return
			}
			require.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, doc)
		})
	}
}

func TestParser_Parse_NestingErrorNamesTheLine(t *testing.T) {
	t.Parallel()

	_, _, err := NewParser(WithMaxNestingDepth(2)).Parse(context.Background(), []byte("a\nb\n> > > c"))

	require.ErrorIs(t, err, ErrNestingTooDeep)
	assert.Contains(t, err.Error(), "line 3")
}

func TestParser_Parse_LimitErrorsReachTheMarkdownService(t *testing.T) {
	t.Parallel()

	service := markdown_domain.NewMarkdownService(NewParser(WithMaxInputSize(8)), nil)

	_, err := service.Process(context.Background(), []byte("too long for the limit"), "page.md")

	require.ErrorIs(t, err, ErrInputTooLarge)
}

func TestNewParser_WithExtensions(t *testing.T) {
	t.Parallel()

	extender := &recordingExtender{}

	NewParser(WithExtensions(extender))

	assert.True(t, extender.extended)
}

func TestCountOpeningMarkers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		line  string
		limit int
		want  int
	}{
		{name: "plain text opens nothing", line: "hello", limit: 10, want: 0},
		{name: "empty line opens nothing", line: "", limit: 10, want: 0},
		{name: "mixed markers with spaces", line: "> > - 1. text", limit: 10, want: 4},
		{name: "adjacent quote markers after indentation", line: "   >>> x", limit: 10, want: 3},
		{name: "tab separated bullets", line: "*\t+\tx", limit: 10, want: 2},
		{name: "counting stops just past the limit", line: strings.Repeat("> ", 50), limit: 5, want: 6},
		{name: "emphasis is not a marker", line: "*bold*", limit: 10, want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, countOpeningMarkers([]byte(tc.line), tc.limit))
		})
	}
}

func TestContainerMarkerWidth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		rest string
		want int
	}{
		{name: "empty", rest: "", want: 0},
		{name: "blockquote", rest: "> x", want: 1},
		{name: "bullet with space", rest: "- x", want: 1},
		{name: "bullet with tab", rest: "+\tx", want: 1},
		{name: "bullet without space", rest: "-x", want: 0},
		{name: "bullet at end of line", rest: "-", want: 0},
		{name: "ordered with full stop", rest: "12. x", want: 3},
		{name: "ordered with bracket", rest: "7) x", want: 2},
		{name: "ordered without space", rest: "1.x", want: 0},
		{name: "ordered at end of line", rest: "1.", want: 0},
		{name: "too many digits", rest: "1234567890. x", want: 0},
		{name: "letter", rest: "a. x", want: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, containerMarkerWidth([]byte(tc.rest)))
		})
	}
}

type recordingExtender struct {
	extended bool
}

func (r *recordingExtender) Extend(goldmark.Markdown) {
	r.extended = true
}
