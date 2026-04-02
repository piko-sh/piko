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

package render_domain

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestIsInternalAttribute(t *testing.T) {
	testCases := []struct {
		name          string
		attributeName string
		want          bool
	}{
		{name: "p-key is internal", attributeName: "p-key", want: true},
		{name: "partial is internal", attributeName: "partial", want: true},
		{name: "p-ref is internal", attributeName: "p-ref", want: true},
		{name: "class is not internal", attributeName: "class", want: false},
		{name: "style is not internal", attributeName: "style", want: false},
		{name: "id is not internal", attributeName: "id", want: false},
		{name: "href is not internal", attributeName: "href", want: false},
		{name: "empty string is not internal", attributeName: "", want: false},
		{name: "p-key-extra is not internal", attributeName: "p-key-extra", want: false},
		{name: "partial-extra is not internal", attributeName: "partial-extra", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isInternalAttribute(tc.attributeName)
			if got != tc.want {
				t.Errorf("isInternalAttribute(%q) = %v, want %v", tc.attributeName, got, tc.want)
			}
		})
	}
}

func TestWriteNodeAndFragmentAttributes_FiltersInternalAttributes_EmailMode(t *testing.T) {

	emailCtx := &renderContext{isEmailMode: true}

	testCases := []struct {
		name      string
		want      string
		nodeAttrs []ast_domain.HTMLAttribute
		fragAttrs []ast_domain.HTMLAttribute
	}{
		{
			name: "filters p-key from node attributes",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "test"},
				{Name: "p-key", Value: "abc123"},
			},
			fragAttrs: nil,
			want:      ` class="test"`,
		},
		{
			name: "filters partial from node attributes",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "main"},
				{Name: "partial", Value: "scope123"},
			},
			fragAttrs: nil,
			want:      ` id="main"`,
		},
		{
			name: "filters p-ref from node attributes",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "href", Value: "/page"},
				{Name: "p-ref", Value: "ref456"},
			},
			fragAttrs: nil,
			want:      ` href="/page"`,
		},
		{
			name:      "filters internal attributes from fragment attributes",
			nodeAttrs: nil,
			fragAttrs: []ast_domain.HTMLAttribute{
				{Name: "style", Value: "color: red"},
				{Name: "p-key", Value: "fragkey"},
				{Name: "partial", Value: "fragscope"},
			},
			want: ` style="color: red"`,
		},
		{
			name: "filters all internal attributes from mixed sources",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "p-key", Value: "nodekey"},
			},
			fragAttrs: []ast_domain.HTMLAttribute{
				{Name: "data-test", Value: "value"},
				{Name: "partial", Value: "fragscope"},
			},
			want: ` class="container" data-test="value"`,
		},
		{
			name: "keeps all regular attributes when no internal attributes present",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "test"},
				{Name: "id", Value: "main"},
			},
			fragAttrs: []ast_domain.HTMLAttribute{
				{Name: "data-foo", Value: "bar"},
			},
			want: ` class="test" id="main" data-foo="bar"`,
		},
		{
			name:      "empty attributes produce empty output",
			nodeAttrs: nil,
			fragAttrs: nil,
			want:      "",
		},
		{
			name: "only internal attributes produce empty output",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "p-key", Value: "key1"},
				{Name: "partial", Value: "scope1"},
			},
			fragAttrs: []ast_domain.HTMLAttribute{
				{Name: "p-ref", Value: "ref1"},
			},
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeNodeAndFragmentAttributes(tc.nodeAttrs, tc.fragAttrs, nil, qw, emailCtx)

			got := buffer.String()
			if got != tc.want {
				t.Errorf("writeNodeAndFragmentAttributes() output:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

func TestWriteNodeAndFragmentAttributes_PreservesInternalAttributes_WebMode(t *testing.T) {

	testCases := []struct {
		rctx      *renderContext
		name      string
		want      string
		nodeAttrs []ast_domain.HTMLAttribute
	}{
		{
			name: "preserves p-key when context is nil",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "test"},
				{Name: "p-key", Value: "abc123"},
			},
			rctx: nil,
			want: ` class="test" p-key="abc123"`,
		},
		{
			name: "preserves partial when context is nil",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "main"},
				{Name: "partial", Value: "scope123"},
			},
			rctx: nil,
			want: ` id="main" partial="scope123"`,
		},
		{
			name: "preserves internal attributes when isEmailMode is false",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "test"},
				{Name: "p-key", Value: "key1"},
				{Name: "partial", Value: "scope1"},
			},
			rctx: &renderContext{isEmailMode: false},
			want: ` class="test" p-key="key1" partial="scope1"`,
		},
		{
			name: "preserves all internal attributes in web mode",
			nodeAttrs: []ast_domain.HTMLAttribute{
				{Name: "p-key", Value: "key"},
				{Name: "partial", Value: "scope"},
				{Name: "p-ref", Value: "ref"},
			},
			rctx: &renderContext{isEmailMode: false},
			want: ` p-key="key" partial="scope" p-ref="ref"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			defer qt.ReleaseWriter(qw)

			writeNodeAndFragmentAttributes(tc.nodeAttrs, nil, nil, qw, tc.rctx)

			got := buffer.String()
			if got != tc.want {
				t.Errorf("writeNodeAndFragmentAttributes() output:\ngot:  %q\nwant: %q", got, tc.want)
			}
		})
	}
}

type testStringer struct {
	value string
}

func (s testStringer) String() string {
	return s.value
}

func TestWriteDirectWriterParts(t *testing.T) {
	t.Cleanup(ast_domain.ResetDirectWriterPool)

	testCases := []struct {
		name  string
		setup func() *ast_domain.DirectWriter
		want  string
	}{
		{
			name: "empty DirectWriter writes nothing",
			setup: func() *ast_domain.DirectWriter {
				return ast_domain.GetDirectWriter()
			},
			want: "",
		},
		{
			name: "single string part writes the string",
			setup: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.AppendString("hello")
				return dw
			},
			want: "hello",
		},
		{
			name: "multiple parts of different types are written in order",
			setup: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.AppendString("count:")
				dw.AppendInt(42)
				dw.AppendString(":done")
				return dw
			},
			want: "count:42:done",
		},
		{
			name: "string and bool parts are combined correctly",
			setup: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.AppendString("active=")
				dw.AppendBool(true)
				return dw
			},
			want: "active=true",
		},
		{
			name: "multiple string parts are concatenated",
			setup: func() *ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.AppendString("foo")
				dw.AppendString(" ")
				dw.AppendString("bar")
				return dw
			},
			want: "foo bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dw := tc.setup()
			defer ast_domain.PutDirectWriter(dw)

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			writeDirectWriterParts(dw, qw)
			qt.ReleaseWriter(qw)

			assert.Equal(t, tc.want, buffer.String())
		})
	}
}

func TestWriteWriterPart(t *testing.T) {
	testCases := []struct {
		name string
		part *ast_domain.WriterPart
		want string
	}{
		{
			name: "WriterPartString writes the string verbatim",
			part: &ast_domain.WriterPart{
				Type:        ast_domain.WriterPartString,
				StringValue: "hello world",
			},
			want: "hello world",
		},
		{
			name: "WriterPartEscapeString escapes HTML angle brackets",
			part: &ast_domain.WriterPart{
				Type:        ast_domain.WriterPartEscapeString,
				StringValue: "<b>bold</b>",
			},
			want: "&lt;b&gt;bold&lt;/b&gt;",
		},
		{
			name: "WriterPartEscapeString escapes ampersand and double quote",
			part: &ast_domain.WriterPart{
				Type:        ast_domain.WriterPartEscapeString,
				StringValue: `A & "B"`,
			},
			want: "A &amp; &quot;B&quot;",
		},
		{
			name: "WriterPartInt writes a positive integer",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartInt,
				IntValue: 42,
			},
			want: "42",
		},
		{
			name: "WriterPartInt writes a negative integer",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartInt,
				IntValue: -100,
			},
			want: "-100",
		},
		{
			name: "WriterPartInt writes zero",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartInt,
				IntValue: 0,
			},
			want: "0",
		},
		{
			name: "WriterPartUint writes an unsigned integer",
			part: &ast_domain.WriterPart{
				Type:      ast_domain.WriterPartUint,
				UintValue: 255,
			},
			want: "255",
		},
		{
			name: "WriterPartUint writes zero",
			part: &ast_domain.WriterPart{
				Type:      ast_domain.WriterPartUint,
				UintValue: 0,
			},
			want: "0",
		},
		{
			name: "WriterPartFloat writes a floating-point number",
			part: &ast_domain.WriterPart{
				Type:       ast_domain.WriterPartFloat,
				FloatValue: 3.14,
			},
			want: "3.14",
		},
		{
			name: "WriterPartFloat writes zero",
			part: &ast_domain.WriterPart{
				Type:       ast_domain.WriterPartFloat,
				FloatValue: 0,
			},
			want: "0",
		},
		{
			name: "WriterPartBool true writes true",
			part: &ast_domain.WriterPart{
				Type:      ast_domain.WriterPartBool,
				BoolValue: true,
			},
			want: "true",
		},
		{
			name: "WriterPartBool false writes false",
			part: &ast_domain.WriterPart{
				Type:      ast_domain.WriterPartBool,
				BoolValue: false,
			},
			want: "false",
		},
		{
			name: "WriterPartBytes writes bytes directly",
			part: &ast_domain.WriterPart{
				Type:       ast_domain.WriterPartBytes,
				BytesValue: []byte("raw bytes"),
			},
			want: "raw bytes",
		},
		{
			name: "WriterPartEscapeBytes escapes HTML in bytes",
			part: &ast_domain.WriterPart{
				Type:       ast_domain.WriterPartEscapeBytes,
				BytesValue: []byte("<div>content</div>"),
			},
			want: "&lt;div&gt;content&lt;/div&gt;",
		},
		{
			name: "WriterPartAny with fmt.Stringer calls String method",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: testStringer{value: "from stringer"},
			},
			want: "from stringer",
		},
		{
			name: "WriterPartAny with non-Stringer uses fmt.Sprint",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: 12345,
			},
			want: "12345",
		},
		{
			name: "WriterPartAny with nil writes nothing",
			part: &ast_domain.WriterPart{
				Type:     ast_domain.WriterPartAny,
				AnyValue: nil,
			},
			want: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			writeWriterPart(tc.part, qw)
			qt.ReleaseWriter(qw)

			assert.Equal(t, tc.want, buffer.String())
		})
	}
}

func TestWritePartBool(t *testing.T) {
	testCases := []struct {
		name  string
		want  string
		value bool
	}{
		{name: "true writes true", value: true, want: "true"},
		{name: "false writes false", value: false, want: "false"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			writePartBool(tc.value, qw)
			qt.ReleaseWriter(qw)

			assert.Equal(t, tc.want, buffer.String())
		})
	}
}

func TestWritePartAny(t *testing.T) {
	testCases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "nil writes nothing",
			value: nil,
			want:  "",
		},
		{
			name:  "fmt.Stringer calls String method",
			value: testStringer{value: "stringer output"},
			want:  "stringer output",
		},
		{
			name:  "integer uses fmt.Sprint",
			value: 42,
			want:  "42",
		},
		{
			name:  "string uses fmt.Sprint",
			value: "plain text",
			want:  "plain text",
		},
		{
			name:  "float uses fmt.Sprint",
			value: 2.718,
			want:  fmt.Sprint(2.718),
		},
		{
			name:  "boolean true uses fmt.Sprint",
			value: true,
			want:  "true",
		},
		{
			name:  "slice uses fmt.Sprint",
			value: []int{1, 2, 3},
			want:  "[1 2 3]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			writePartAny(tc.value, qw)
			qt.ReleaseWriter(qw)

			assert.Equal(t, tc.want, buffer.String())
		})
	}
}

func TestShouldExcludeAttribute(t *testing.T) {
	testCases := []struct {
		name          string
		attributeName string
		excludeNames  []string
		want          bool
	}{
		{
			name:          "name in exclude list returns true",
			attributeName: "src",
			excludeNames:  []string{"src", "alt"},
			want:          true,
		},
		{
			name:          "name not in exclude list returns false",
			attributeName: "class",
			excludeNames:  []string{"src", "alt"},
			want:          false,
		},
		{
			name:          "empty exclude list returns false",
			attributeName: "class",
			excludeNames:  []string{},
			want:          false,
		},
		{
			name:          "nil exclude list returns false",
			attributeName: "class",
			excludeNames:  nil,
			want:          false,
		},
		{
			name:          "empty attribute name not in list returns false",
			attributeName: "",
			excludeNames:  []string{"src"},
			want:          false,
		},
		{
			name:          "empty attribute name in list returns true",
			attributeName: "",
			excludeNames:  []string{""},
			want:          true,
		},
		{
			name:          "single item list matches",
			attributeName: "width",
			excludeNames:  []string{"width"},
			want:          true,
		},
		{
			name:          "single item list does not match",
			attributeName: "height",
			excludeNames:  []string{"width"},
			want:          false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldExcludeAttribute(tc.attributeName, tc.excludeNames)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWriteAttributeWritersExcluding(t *testing.T) {
	t.Cleanup(ast_domain.ResetDirectWriterPool)

	makeAttrWriter := func(name, value string) *ast_domain.DirectWriter {
		dw := ast_domain.GetDirectWriter()
		dw.Name = name
		dw.AppendString(value)
		return dw
	}

	testCases := []struct {
		setup        func() []*ast_domain.DirectWriter
		name         string
		want         string
		excludeNames []string
	}{
		{
			name: "no exclusions writes all attributes",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "container"),
					makeAttrWriter("id", "main"),
				}
			},
			excludeNames: nil,
			want:         ` class="container" id="main"`,
		},
		{
			name: "single exclusion skips matching attribute",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "container"),
					makeAttrWriter("src", "/image.png"),
					makeAttrWriter("alt", "photo"),
				}
			},
			excludeNames: []string{"src"},
			want:         ` class="container" alt="photo"`,
		},
		{
			name: "multiple exclusions skip all matching attributes",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "container"),
					makeAttrWriter("src", "/image.png"),
					makeAttrWriter("alt", "photo"),
				}
			},
			excludeNames: []string{"src", "alt"},
			want:         ` class="container"`,
		},
		{
			name: "nil DirectWriter in slice is skipped",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "active"),
					nil,
					makeAttrWriter("id", "header"),
				}
			},
			excludeNames: nil,
			want:         ` class="active" id="header"`,
		},
		{
			name: "DirectWriter with Len zero is skipped",
			setup: func() []*ast_domain.DirectWriter {
				empty := ast_domain.GetDirectWriter()
				empty.Name = "style"

				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "active"),
					empty,
				}
			},
			excludeNames: nil,
			want:         ` class="active"`,
		},
		{
			name: "DirectWriter with empty Name is skipped",
			setup: func() []*ast_domain.DirectWriter {
				unnamed := ast_domain.GetDirectWriter()
				unnamed.AppendString("some-value")

				return []*ast_domain.DirectWriter{
					makeAttrWriter("class", "active"),
					unnamed,
				}
			},
			excludeNames: nil,
			want:         ` class="active"`,
		},
		{
			name: "empty writers slice writes nothing",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{}
			},
			excludeNames: nil,
			want:         "",
		},
		{
			name: "all writers excluded writes nothing",
			setup: func() []*ast_domain.DirectWriter {
				return []*ast_domain.DirectWriter{
					makeAttrWriter("src", "/img.png"),
				}
			},
			excludeNames: []string{"src"},
			want:         "",
		},
		{
			name: "attribute with multiple parts renders all parts",
			setup: func() []*ast_domain.DirectWriter {
				dw := ast_domain.GetDirectWriter()
				dw.Name = "class"
				dw.AppendString("container")
				dw.AppendString(" ")
				dw.AppendString("active")
				return []*ast_domain.DirectWriter{dw}
			},
			excludeNames: nil,
			want:         ` class="container active"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			writers := tc.setup()
			defer func() {
				for _, w := range writers {
					ast_domain.PutDirectWriter(w)
				}
			}()

			var buffer bytes.Buffer
			qw := qt.AcquireWriter(&buffer)
			if len(tc.excludeNames) > 0 {
				writeAttributeWritersExcluding(writers, qw, tc.excludeNames...)
			} else {
				writeAttributeWritersExcluding(writers, qw)
			}
			qt.ReleaseWriter(qw)

			assert.Equal(t, tc.want, buffer.String())
		})
	}
}
