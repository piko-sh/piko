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
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func Test_parseSVGTagAttributes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []svgAttribute
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   \t\n  ",
			expected: nil,
		},
		{
			name:  "single attribute double quotes",
			input: `viewBox="0 0 24 24"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
		},
		{
			name:  "single attribute single quotes",
			input: `viewBox='0 0 24 24'`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
		},
		{
			name:  "multiple attributes",
			input: `viewBox="0 0 24 24" class="icon" fill="none"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "class", Value: "icon"},
				{Name: "fill", Value: "none"},
			},
		},
		{
			name:  "leading whitespace",
			input: `  viewBox="0 0 24 24"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
		},
		{
			name:  "extra whitespace between attributes",
			input: `viewBox="0 0 24 24"    class="icon"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "class", Value: "icon"},
			},
		},
		{
			name:  "whitespace around equals",
			input: `viewBox = "0 0 24 24"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
			},
		},
		{
			name:  "boolean attribute",
			input: `hidden class="icon"`,
			expected: []svgAttribute{
				{Name: "hidden", Value: ""},
				{Name: "class", Value: "icon"},
			},
		},
		{
			name:  "typical SVG tag content",
			input: ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"`,
			expected: []svgAttribute{
				{Name: "xmlns", Value: "http://www.w3.org/2000/svg"},
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "fill", Value: "none"},
				{Name: "stroke", Value: "currentColor"},
				{Name: "stroke-width", Value: "2"},
			},
		},
		{
			name:  "empty attribute value",
			input: `class=""`,
			expected: []svgAttribute{
				{Name: "class", Value: ""},
			},
		},
		{
			name:  "unquoted value",
			input: `width=24 height=24`,
			expected: []svgAttribute{
				{Name: "width", Value: "24"},
				{Name: "height", Value: "24"},
			},
		},
		{
			name:  "mixed quote styles",
			input: `viewBox="0 0 24 24" class='icon'`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "class", Value: "icon"},
			},
		},
		{
			name:  "value with special characters",
			input: `d="M12 2L2 7l10 5 10-5-10-5z"`,
			expected: []svgAttribute{
				{Name: "d", Value: "M12 2L2 7l10 5 10-5-10-5z"},
			},
		},
		{
			name:  "camelCase attributes preserved",
			input: `viewBox="0 0 24 24" strokeWidth="2" fillOpacity="0.5"`,
			expected: []svgAttribute{
				{Name: "viewBox", Value: "0 0 24 24"},
				{Name: "strokeWidth", Value: "2"},
				{Name: "fillOpacity", Value: "0.5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attrs := parseSVGTagAttributes(tt.input)
			defer putSVGAttrSlice(attrs)

			if tt.expected == nil {
				if attrs != nil && len(*attrs) > 0 {
					t.Errorf("expected nil/empty, got %v", *attrs)
				}
				return
			}

			if attrs == nil {
				t.Fatalf("expected %v, got nil", tt.expected)
			}

			if len(*attrs) != len(tt.expected) {
				t.Fatalf("expected %d attributes, got %d: %v", len(tt.expected), len(*attrs), *attrs)
			}

			for i, exp := range tt.expected {
				got := (*attrs)[i]
				if got.Name != exp.Name {
					t.Errorf("attribute %d: expected name %q, got %q", i, exp.Name, got.Name)
				}
				if got.Value != exp.Value {
					t.Errorf("attribute %d: expected value %q, got %q", i, exp.Value, got.Value)
				}
			}
		})
	}
}

func TestParseSVGAttributes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCount int
	}{
		{name: "empty", input: "", expectCount: 0},
		{name: "whitespace", input: "   ", expectCount: 0},
		{name: "single", input: `viewBox="0 0 24 24"`, expectCount: 1},
		{name: "multiple", input: `viewBox="0 0 24 24" class="icon"`, expectCount: 2},
		{name: "typical SVG", input: ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"`, expectCount: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSVGAttributes(tt.input)
			if len(result) != tt.expectCount {
				t.Errorf("expected %d attributes, got %d", tt.expectCount, len(result))
			}
		})
	}
}

func TestParseSVGAttributes_PreservesCase(t *testing.T) {

	testCases := []struct {
		input    string
		expected []string
	}{
		{input: `viewBox="0 0 24 24"`, expected: []string{"viewBox"}},
		{input: `strokeWidth="2"`, expected: []string{"strokeWidth"}},
		{input: `xmlns="..." viewBox="0 0 24 24"`, expected: []string{"xmlns", "viewBox"}},
		{input: `class="" ID="test"`, expected: []string{"class", "ID"}},
		{input: `fillOpacity="0.5" strokeDasharray="5,5"`, expected: []string{"fillOpacity", "strokeDasharray"}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := ParseSVGAttributes(tc.input)

			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d attributes, got %d", len(tc.expected), len(result))
			}

			for i, exp := range tc.expected {
				if result[i].Name != exp {
					t.Errorf("attribute %d: expected name %q, got %q", i, exp, result[i].Name)
				}
			}
		})
	}
}

func TestParseSVGAttributes_PoolReuse(t *testing.T) {

	input1 := `viewBox="0 0 24 24" class="icon"`
	input2 := `fill="none"`

	result1 := ParseSVGAttributes(input1)
	if len(result1) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(result1))
	}

	result2 := ParseSVGAttributes(input2)
	if len(result2) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(result2))
	}
	if result2[0].Name != "fill" {
		t.Errorf("expected 'fill', got %q", result2[0].Name)
	}
}

func Test_convertSVGToHTMLAttributes(t *testing.T) {
	attrs := &[]svgAttribute{
		{Name: "viewBox", Value: "0 0 24 24"},
		{Name: "class", Value: "icon"},
	}

	result := convertSVGToHTMLAttributes(attrs)

	if len(result) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(result))
	}

	if result[0].Name != "viewBox" || result[0].Value != "0 0 24 24" {
		t.Errorf("first attribute mismatch: %+v", result[0])
	}
	if result[1].Name != "class" || result[1].Value != "icon" {
		t.Errorf("second attribute mismatch: %+v", result[1])
	}

	if result[0].Location.Line != 0 || result[0].Location.Column != 0 {
		t.Error("Location should be zero-valued")
	}
}

func Test_convertSVGToHTMLAttributes_NilInput(t *testing.T) {
	result := convertSVGToHTMLAttributes(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func Test_convertSVGToHTMLAttributes_EmptySlice(t *testing.T) {
	attrs := &[]svgAttribute{}
	result := convertSVGToHTMLAttributes(attrs)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func parseAttributesWithTokeniser(tagContent string) int {
	if strings.TrimSpace(tagContent) == "" {
		return 0
	}

	tokeniserInput := "<div " + tagContent + ">"
	tokeniser := html.NewTokenizer(strings.NewReader(tokeniserInput))

	if tokenType := tokeniser.Next(); tokenType == html.StartTagToken {
		tok := tokeniser.Token()
		return len(tok.Attr)
	}
	return 0
}

func BenchmarkParseSVGAttributes(b *testing.B) {
	input := ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-check"`

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = ParseSVGAttributes(input)
	}
}

func BenchmarkParseSVGAttributes_OldTokeniser(b *testing.B) {
	input := ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-check"`

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = parseAttributesWithTokeniser(input)
	}
}

func BenchmarkParseSVGAttributes_BySizes(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{
			name:  "minimal_1attr",
			input: `viewBox="0 0 24 24"`,
		},
		{
			name:  "small_3attrs",
			input: `viewBox="0 0 24 24" fill="none" class="icon"`,
		},
		{
			name:  "medium_6attrs",
			input: ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="icon"`,
		},
		{
			name:  "large_10attrs",
			input: ` xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-check" width="24" height="24"`,
		},
	}

	for _, tc := range testCases {
		b.Run("new/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = ParseSVGAttributes(tc.input)
			}
		})

		b.Run("old_tokeniser/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = parseAttributesWithTokeniser(tc.input)
			}
		})
	}
}

func Benchmark_parseSVGTagAttributes_PoolEfficiency(b *testing.B) {

	input := `viewBox="0 0 24 24" fill="none" class="icon"`

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			attrs := parseSVGTagAttributes(input)
			putSVGAttrSlice(attrs)
		}
	})
}

func Benchmark_convertSVGToHTMLAttributes(b *testing.B) {
	attrs := &[]svgAttribute{
		{Name: "xmlns", Value: "http://www.w3.org/2000/svg"},
		{Name: "viewBox", Value: "0 0 24 24"},
		{Name: "fill", Value: "none"},
		{Name: "stroke", Value: "currentColor"},
		{Name: "stroke-width", Value: "2"},
		{Name: "class", Value: "icon"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = convertSVGToHTMLAttributes(attrs)
	}
}
