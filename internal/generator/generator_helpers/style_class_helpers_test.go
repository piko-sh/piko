// Copyright 2026 PolitePixels Limited
// Tests for style_class_helpers.go

package generator_helpers

import (
	"strings"
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func warmupStylePool() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func TestBuildStyleStringBytes_FixedArityEquivalence(t *testing.T) {
	t.Parallel()

	warmupStylePool()

	testCases := []struct {
		name  string
		parts []string
	}{
		{
			name:  "two parts - property and value",
			parts: []string{"color: ", "red"},
		},
		{
			name:  "three parts - CSS variable",
			parts: []string{"--gradient-colour: var(", "blue", ")"},
		},
		{
			name:  "four parts - two properties",
			parts: []string{"color: ", "red", "; background: ", "white"},
		},
		{
			name:  "three parts - with semicolon",
			parts: []string{"display: ", "flex", ";"},
		},
		{
			name:  "four parts - complex property",
			parts: []string{"border: ", "1px", " solid ", "#ccc"},
		},
		{
			name:  "two parts - empty value",
			parts: []string{"color: ", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var builder strings.Builder
			for _, p := range tc.parts {
				builder.WriteString(p)
			}
			concat := builder.String()
			expectedPtr := StylesFromStringBytes(concat)
			var expected string
			if expectedPtr != nil {
				expected = string(*expectedPtr)
				ast_domain.PutByteBuf(expectedPtr)
			}

			var actualPtr *[]byte
			switch len(tc.parts) {
			case 2:
				actualPtr = BuildStyleStringBytes2(tc.parts[0], tc.parts[1])
			case 3:
				actualPtr = BuildStyleStringBytes3(tc.parts[0], tc.parts[1], tc.parts[2])
			case 4:
				actualPtr = BuildStyleStringBytes4(tc.parts[0], tc.parts[1], tc.parts[2], tc.parts[3])
			}

			var actual string
			if actualPtr != nil {
				actual = string(*actualPtr)
				ast_domain.PutByteBuf(actualPtr)
			}

			if actual != expected {
				t.Errorf("mismatch for parts %v:\n  expected: %q\n  actual:   %q", tc.parts, expected, actual)
			}
		})
	}
}

func TestBuildStyleStringBytesV_Equivalence(t *testing.T) {
	t.Parallel()

	warmupStylePool()

	testCases := []struct {
		name  string
		parts []string
	}{
		{
			name:  "one part",
			parts: []string{"color: red"},
		},
		{
			name:  "two parts",
			parts: []string{"color: ", "red"},
		},
		{
			name:  "three parts",
			parts: []string{"--var: ", "value", ";"},
		},
		{
			name:  "four parts",
			parts: []string{"color: ", "red", "; bg: ", "blue"},
		},
		{
			name:  "five parts - falls back to concat",
			parts: []string{"color: ", "red", "; bg: ", "blue", "; opacity: 1"},
		},
		{
			name:  "empty",
			parts: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var builder strings.Builder
			for _, p := range tc.parts {
				builder.WriteString(p)
			}
			concat := builder.String()

			var expected string
			if len(tc.parts) > 0 {
				expectedPtr := StylesFromStringBytes(concat)
				if expectedPtr != nil {
					expected = string(*expectedPtr)
					ast_domain.PutByteBuf(expectedPtr)
				}
			}

			actualPtr := BuildStyleStringBytesV(tc.parts...)
			var actual string
			if actualPtr != nil {
				actual = string(*actualPtr)
				ast_domain.PutByteBuf(actualPtr)
			}

			if actual != expected {
				t.Errorf("mismatch for parts %v:\n  expected: %q\n  actual:   %q", tc.parts, expected, actual)
			}
		})
	}
}

func TestBuildStyleStringBytes_SpecialCases(t *testing.T) {
	t.Parallel()

	warmupStylePool()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple property",
			input:    "color: red",
			expected: "color:red;",
		},
		{
			name:     "multiple properties - sorted alphabetically",
			input:    "color: red; background: blue",
			expected: "background:blue;color:red;",
		},
		{
			name:     "CSS variable",
			input:    "--gradient-colour: var(blue)",
			expected: "--gradient-colour:var(blue);",
		},
		{
			name:     "with extra whitespace - normalised",
			input:    "color:  red ;  background:   blue  ",
			expected: "background:blue;color:red;",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resultPtr := StylesFromStringBytes(tc.input)
			var result string
			if resultPtr != nil {
				result = string(*resultPtr)
				ast_domain.PutByteBuf(resultPtr)
			}

			if result != tc.expected {
				t.Errorf("StylesFromStringBytes(%q):\n  expected: %q\n  actual:   %q", tc.input, tc.expected, result)
			}
		})
	}
}
