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
)

func TestIsValidJSIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{

		{name: "simple identifier", input: "myRef", expected: true},
		{name: "single letter", input: "x", expected: true},
		{name: "underscore prefix", input: "_private", expected: true},
		{name: "dollar prefix", input: "$special", expected: true},
		{name: "with numbers", input: "ref123", expected: true},
		{name: "underscore only", input: "_", expected: true},
		{name: "dollar only", input: "$", expected: true},
		{name: "mixed case", input: "MyRefName", expected: true},
		{name: "camelCase", input: "camelCase", expected: true},
		{name: "snake_case", input: "snake_case", expected: true},
		{name: "dollar and underscore", input: "$_value", expected: true},
		{name: "unicode letters", input: "αβγ", expected: true},
		{name: "unicode with ascii", input: "myVarα", expected: true},
		{name: "japanese", input: "変数", expected: true},
		{name: "emoji prefix underscore", input: "_emoji", expected: true},

		{name: "empty string", input: "", expected: false},
		{name: "starts with number", input: "123abc", expected: false},
		{name: "number only", input: "123", expected: false},
		{name: "hyphenated", input: "my-ref", expected: false},
		{name: "dotted", input: "state.ref", expected: false},
		{name: "contains space", input: "my ref", expected: false},
		{name: "contains at sign", input: "my@ref", expected: false},
		{name: "contains hash", input: "my#ref", expected: false},
		{name: "contains exclamation", input: "my!ref", expected: false},
		{name: "contains percent", input: "my%ref", expected: false},
		{name: "starts with hyphen", input: "-myRef", expected: false},
		{name: "contains parentheses", input: "func()", expected: false},
		{name: "contains brackets", input: "arr[0]", expected: false},
		{name: "contains colon", input: "obj:key", expected: false},
		{name: "whitespace only", input: "   ", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidJSIdentifier(tt.input)
			assert.Equal(t, tt.expected, result, "IsValidJSIdentifier(%q) = %v, expected %v", tt.input, result, tt.expected)
		})
	}
}

func TestDirectiveTypeString(t *testing.T) {
	tests := []struct {
		expected string
		dtype    DirectiveType
	}{
		{expected: "If", dtype: DirectiveIf},
		{expected: "ElseIf", dtype: DirectiveElseIf},
		{expected: "Else", dtype: DirectiveElse},
		{expected: "For", dtype: DirectiveFor},
		{expected: "Show", dtype: DirectiveShow},
		{expected: "Bind", dtype: DirectiveBind},
		{expected: "Model", dtype: DirectiveModel},
		{expected: "On", dtype: DirectiveOn},
		{expected: "Event", dtype: DirectiveEvent},
		{expected: "Class", dtype: DirectiveClass},
		{expected: "Style", dtype: DirectiveStyle},
		{expected: "Text", dtype: DirectiveText},
		{expected: "Html", dtype: DirectiveHTML},
		{expected: "Ref", dtype: DirectiveRef},
		{expected: "Key", dtype: DirectiveKey},
		{expected: "Context", dtype: DirectiveContext},
		{expected: "Scaffold", dtype: DirectiveScaffold},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.dtype.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePrefixedDirectiveEventModifiers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		key           string
		wantArg       string
		wantModifiers []string
		wantType      DirectiveType
		wantFound     bool
	}{
		{
			name:          "p-on with no modifier",
			key:           "p-on:click",
			wantType:      DirectiveOn,
			wantArg:       "click",
			wantModifiers: nil,
			wantFound:     true,
		},
		{
			name:          "p-on with single modifier",
			key:           "p-on:click.prevent",
			wantType:      DirectiveOn,
			wantArg:       "click",
			wantModifiers: []string{"prevent"},
			wantFound:     true,
		},
		{
			name:          "p-on with two modifiers",
			key:           "p-on:click.prevent.stop",
			wantType:      DirectiveOn,
			wantArg:       "click",
			wantModifiers: []string{"prevent", "stop"},
			wantFound:     true,
		},
		{
			name:          "p-on submit with prevent and once",
			key:           "p-on:submit.prevent.once",
			wantType:      DirectiveOn,
			wantArg:       "submit",
			wantModifiers: []string{"prevent", "once"},
			wantFound:     true,
		},
		{
			name:          "p-on with all four modifiers",
			key:           "p-on:click.prevent.stop.once.self",
			wantType:      DirectiveOn,
			wantArg:       "click",
			wantModifiers: []string{"prevent", "stop", "once", "self"},
			wantFound:     true,
		},
		{
			name:          "p-event with single modifier",
			key:           "p-event:update.stop",
			wantType:      DirectiveEvent,
			wantArg:       "update",
			wantModifiers: []string{"stop"},
			wantFound:     true,
		},
		{
			name:          "p-event with no modifier",
			key:           "p-event:custom",
			wantType:      DirectiveEvent,
			wantArg:       "custom",
			wantModifiers: nil,
			wantFound:     true,
		},
		{
			name:      "non-event directive not matched",
			key:       "p-if",
			wantFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d, found := parsePrefixedDirective([]byte(tc.key), "handler()", Location{})
			assert.Equal(t, tc.wantFound, found, "found mismatch")

			if !found {
				return
			}

			assert.Equal(t, tc.wantType, d.Type, "directive type mismatch")
			assert.Equal(t, tc.wantArg, d.Arg, "argument mismatch")
			assert.Equal(t, tc.wantModifiers, d.EventModifiers, "event modifiers mismatch")
			assert.Empty(t, d.Modifier, "internal modifier should be empty from parser")
		})
	}
}

func TestResolveDirectiveType(t *testing.T) {
	tests := []struct {
		key      string
		expected DirectiveType
		found    bool
	}{
		{key: "p-if", expected: DirectiveIf, found: true},
		{key: "p-else-if", expected: DirectiveElseIf, found: true},
		{key: "p-else", expected: DirectiveElse, found: true},
		{key: "p-for", expected: DirectiveFor, found: true},
		{key: "p-show", expected: DirectiveShow, found: true},
		{key: "p-bind", expected: DirectiveBind, found: true},
		{key: "p-model", expected: DirectiveModel, found: true},
		{key: "p-on", expected: DirectiveOn, found: true},
		{key: "p-event", expected: DirectiveEvent, found: true},
		{key: "p-class", expected: DirectiveClass, found: true},
		{key: "p-style", expected: DirectiveStyle, found: true},
		{key: "p-text", expected: DirectiveText, found: true},
		{key: "p-html", expected: DirectiveHTML, found: true},
		{key: "p-ref", expected: DirectiveRef, found: true},
		{key: "p-key", expected: DirectiveKey, found: true},
		{key: "p-context", expected: DirectiveContext, found: true},
		{key: "p-scaffold", expected: DirectiveScaffold, found: true},
		{key: "unknown", expected: DirectiveIf, found: false},
		{key: "", expected: DirectiveIf, found: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result, found := resolveDirectiveType([]byte(tt.key))
			assert.Equal(t, tt.found, found)
			if tt.found {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
