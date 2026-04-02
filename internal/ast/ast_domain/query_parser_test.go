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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func normaliseSelectorSetLocations(set SelectorSet) {
	for _, group := range set {
		for i := range group {
			group[i].Location = Location{}
			normaliseSimpleSelectorLocations(&group[i].Simple)
		}
	}
}

func normaliseSimpleSelectorLocations(ss *SimpleSelector) {
	ss.Location = Location{}
	for j := range ss.Attributes {
		ss.Attributes[j].Location = Location{}
	}
	for j := range ss.PseudoClasses {
		ss.PseudoClasses[j].Location = Location{}
		if ss.PseudoClasses[j].SubSelector != nil {
			normaliseSimpleSelectorLocations(ss.PseudoClasses[j].SubSelector)
		}
	}
}

func TestQueryParser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		input        string
		errContains  string
		expectedSet  SelectorSet
		expectedErrs int
	}{
		{
			name:  "Simple tag selector",
			input: "div",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "div"}}},
			},
		},
		{
			name:  "Tag with class and ID",
			input: "div.card#main",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "div", ID: "main", Classes: []string{"card"}}}},
			},
		},
		{
			name:  "Multiple classes",
			input: ".card.active.featured",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Classes: []string{"card", "active", "featured"}}}},
			},
		},
		{
			name:  "Universal selector with class",
			input: "*.message",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "*", Classes: []string{"message"}}}},
			},
		},
		{
			name:  "Attribute selector existence",
			input: "input[disabled]",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "input", Attributes: []AttributeSelector{{Name: "disabled"}}}}},
			},
		},
		{
			name:  "Attribute selector with value",
			input: `a[href="https://example.com"]`,
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "a", Attributes: []AttributeSelector{{Name: "href", Operator: "=", Value: "https://example.com"}}}}},
			},
		},
		{
			name:  "Attribute selector with unquoted value",
			input: `[type=text]`,
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Attributes: []AttributeSelector{{Name: "type", Operator: "=", Value: "text"}}}}},
			},
		},
		{
			name:  "Attribute Case-Insensitive Flag",
			input: `[data-value="foo" i]`,
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Attributes: []AttributeSelector{{Name: "data-value", Operator: "=", Value: "foo", CaseInsensitive: true}}}}},
			},
		},
		{
			name:  "Descendant combinator",
			input: "main .content p",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "main"}},
					{Combinator: " ", Simple: SimpleSelector{Classes: []string{"content"}}},
					{Combinator: " ", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "Child combinator",
			input: "ul > li",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "ul"}},
					{Combinator: ">", Simple: SimpleSelector{Tag: "li"}},
				},
			},
		},
		{
			name:  "Adjacent Sibling combinator",
			input: "h1 + p",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "h1"}},
					{Combinator: "+", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "General Sibling combinator",
			input: "h1 ~ p",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "h1"}},
					{Combinator: "~", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "Simple Selector List",
			input: "h1, .content, #app",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "h1"}}},
				{{Combinator: "", Simple: SimpleSelector{Classes: []string{"content"}}}},
				{{Combinator: "", Simple: SimpleSelector{ID: "app"}}},
			},
		},
		{
			name:  "Complex Selector List",
			input: `div#app > .container, footer p`,
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "div", ID: "app"}},
					{Combinator: ">", Simple: SimpleSelector{Classes: []string{"container"}}},
				},
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "footer"}},
					{Combinator: " ", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "Selector list with trailing comma",
			input: "h1, h2,",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "h1"}}},
				{{Combinator: "", Simple: SimpleSelector{Tag: "h2"}}},
			},
		},
		{
			name:  "Simple pseudo-class",
			input: "a:first-child",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "a", PseudoClasses: []PseudoClassSelector{{Type: "first-child"}}}}},
			},
		},
		{
			name:  "Chained pseudo-classes",
			input: "li:first-child:last-of-type",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "li", PseudoClasses: []PseudoClassSelector{{Type: "first-child"}, {Type: "last-of-type"}}}}},
			},
		},
		{
			name:  "Pseudo-class :not() with simple selector",
			input: "p:not(.special)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "p", PseudoClasses: []PseudoClassSelector{
					{Type: "not", SubSelector: &SimpleSelector{Classes: []string{"special"}}},
				}}}},
			},
		},
		{
			name:  "Pseudo-class :not() with complex simple selector",
			input: `div:not(#main[disabled])`,
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "div", PseudoClasses: []PseudoClassSelector{
					{Type: "not", SubSelector: &SimpleSelector{ID: "main", Attributes: []AttributeSelector{{Name: "disabled"}}}},
				}}}},
			},
		},
		{
			name:  "Pseudo-class :nth-child with keyword",
			input: "tr:nth-child(odd)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "tr", PseudoClasses: []PseudoClassSelector{{Type: "nth-child", Value: "odd"}}}}},
			},
		},
		{
			name:  "Pseudo-class :nth-of-type with number",
			input: "p:nth-of-type(3)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "p", PseudoClasses: []PseudoClassSelector{{Type: "nth-of-type", Value: "3"}}}}},
			},
		},
		{
			name:  "Pseudo-class :nth-child with formula",
			input: "li:nth-child(2n + 1)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "li", PseudoClasses: []PseudoClassSelector{{Type: "nth-child", Value: "2n+1"}}}}},
			},
		},
		{
			name:         "Error: Missing identifier after hash",
			input:        "#",
			expectedErrs: 1,
			errContains:  "expected identifier after #",
		},
		{
			name:         "Error: Missing identifier after dot",
			input:        "div.",
			expectedErrs: 1,
			errContains:  "expected identifier after .",
		},
		{
			name:         "Error: Unclosed attribute selector",
			input:        "[type='text'",
			expectedErrs: 1,
			errContains:  "expected ']' to close attribute selector",
		},
		{
			name:         "Error: Missing attribute name",
			input:        "[]",
			expectedErrs: 1,
			errContains:  "expected attribute name",
		},
		{
			name:         "Error: Empty :not() pseudo-class",
			input:        "p:not()",
			expectedErrs: 1,
			errContains:  "expected a selector inside :not()",
		},
		{
			name:         "Error: Unclosed pseudo-class function",
			input:        ":not(.special",
			expectedErrs: 1,
			errContains:  "expected ')' to close pseudo-class function",
		},
		{
			name:         "Error: Invalid pseudo-class name",
			input:        ":",
			expectedErrs: 1,
			errContains:  "expected identifier for pseudo-class name",
		},
		{
			name:         "Error: Trailing combinator",
			input:        "div >",
			expectedErrs: 1,
			errContains:  "Expected a selector after combinator",
		},
		{
			name:         "Error: Multiple combinators",
			input:        "div >> p",
			expectedErrs: 1,
			errContains:  "Expected a selector after combinator",
		},
		{
			name:         "Error: Unexpected token at start of selector",
			input:        ")",
			expectedErrs: 1,
			errContains:  "Expected a selector",
		},
		{
			name:         "Error: Unsupported functional pseudo-class",
			input:        "p:matches(.special)",
			expectedErrs: 1,
			errContains:  "unsupported functional pseudo-class",
		},
		{
			name:         "Error: Another unsupported pseudo-class",
			input:        "div:has(span)",
			expectedErrs: 1,
			errContains:  "unsupported functional pseudo-class",
		},
		{
			name:  "Adjacent sibling combinator",
			input: "h1 + p",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "h1"}},
					{Combinator: "+", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "General sibling combinator",
			input: "h1 ~ p",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "h1"}},
					{Combinator: "~", Simple: SimpleSelector{Tag: "p"}},
				},
			},
		},
		{
			name:  "Descendant combinator with whitespace",
			input: "div p span",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "div"}},
					{Combinator: " ", Simple: SimpleSelector{Tag: "p"}},
					{Combinator: " ", Simple: SimpleSelector{Tag: "span"}},
				},
			},
		},
		{
			name:  "Complex selector with all combinators",
			input: "div > p + span ~ a",
			expectedSet: SelectorSet{
				{
					{Combinator: "", Simple: SimpleSelector{Tag: "div"}},
					{Combinator: ">", Simple: SimpleSelector{Tag: "p"}},
					{Combinator: "+", Simple: SimpleSelector{Tag: "span"}},
					{Combinator: "~", Simple: SimpleSelector{Tag: "a"}},
				},
			},
		},
		{
			name:  "nth-child with even value",
			input: "li:nth-child(even)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "li", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-child", Value: "even"},
				}}}},
			},
		},
		{
			name:  "nth-child with odd value",
			input: "li:nth-child(odd)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "li", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-child", Value: "odd"},
				}}}},
			},
		},
		{
			name:  "nth-child with formula",
			input: "tr:nth-child(2n+1)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "tr", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-child", Value: "2n+1"},
				}}}},
			},
		},
		{
			name:  "nth-of-type pseudo-class",
			input: "p:nth-of-type(3)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "p", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-of-type", Value: "3"},
				}}}},
			},
		},
		{
			name:  "nth-last-child pseudo-class",
			input: "li:nth-last-child(2)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "li", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-last-child", Value: "2"},
				}}}},
			},
		},
		{
			name:  "nth-last-of-type pseudo-class",
			input: "span:nth-last-of-type(3n)",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "span", PseudoClasses: []PseudoClassSelector{
					{Type: "nth-last-of-type", Value: "3n"},
				}}}},
			},
		},
		{
			name:  "Multiple pseudo-classes",
			input: "a:hover:focus",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "a", PseudoClasses: []PseudoClassSelector{
					{Type: "hover"},
					{Type: "focus"},
				}}}},
			},
		},
		{
			name:        "Empty input returns empty set",
			input:       "",
			expectedSet: nil,
		},
		{
			name:         "Error: Unexpected token after selector",
			input:        "div @",
			expectedErrs: 1,
			errContains:  "Unexpected token",
		},
		{
			name:  "Selector list with multiple groups",
			input: "h1, h2, h3",
			expectedSet: SelectorSet{
				{{Combinator: "", Simple: SimpleSelector{Tag: "h1"}}},
				{{Combinator: "", Simple: SimpleSelector{Tag: "h2"}}},
				{{Combinator: "", Simple: SimpleSelector{Tag: "h3"}}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lexer := NewQueryLexer(tc.input)
			parser := NewQueryParser(lexer, "test.selector")
			set := parser.Parse()
			diagnostics := parser.Diagnostics()

			if tc.expectedErrs > 0 {
				require.Len(t, diagnostics, tc.expectedErrs, "Test case '%s' failed: incorrect number of errors", tc.name)
				assert.Contains(t, diagnostics[0].Message, tc.errContains, "Test case '%s' failed: error message mismatch", tc.name)
				return
			}

			require.Empty(t, diagnostics, "Expected no parsing errors for input: '%s'", tc.input)

			normaliseSelectorSetLocations(set)

			assert.Equal(t, tc.expectedSet, set, "Test case '%s' failed: parsed structure mismatch", tc.name)
		})
	}
}
