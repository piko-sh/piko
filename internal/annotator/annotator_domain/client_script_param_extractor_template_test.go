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

package annotator_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	extractionTimeout = 5 * time.Second
)

func TestExtractFunctionParams_TemplateLiterals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
	}{
		{
			name:   "non-ASCII text before a substitution",
			source: "function price(change: string) { return `£${change}`; }\nfunction after(count: number) {}",
		},
		{
			name:   "two templates with non-ASCII text in one expression",
			source: "const values = [`£${change.quoted}`, `£${change.current}`];\nfunction after(count: number) {}",
		},
		{
			name:   "non-ASCII text after a substitution",
			source: "const label = `${amount}£ — paid`;\nfunction after(count: number) {}",
		},
		{
			name:   "several substitutions",
			source: "const label = `${a} and ${b} cost £${c}`;\nfunction after(count: number) {}",
		},
		{
			name:   "object literal braces inside a substitution",
			source: "const label = `${ {quantity: 1}.quantity } × £${ {price: 2}.price }`;\nfunction after(count: number) {}",
		},
		{
			name:   "a template nested in a substitution",
			source: "const label = `outer ${`inner ${name} £`} done`;\nfunction after(count: number) {}",
		},
		{
			name:   "a block in an arrow function inside a substitution",
			source: "const label = `${(() => { return '£'; })()} £${total}`;\nfunction after(count: number) {}",
		},
		{
			name:   "a template without substitutions",
			source: "const label = `just £ text`;\nfunction after(count: number) {}",
		},
		{
			name:   "a template inside a function body",
			source: "function show(change: string) { const text = `£${change}`; return text; }\nfunction after(count: number) {}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractWithinTimeout(t, tc.source)

			require.Contains(t, result, "after", "a function declared after the template is still found")
			params := result["after"]
			require.Len(t, params, 1)
			assert.Equal(t, "count", params[0].Name)
			assert.Equal(t, categoryNumber, params[0].Category)
		})
	}
}

func TestExtractFunctionParams_TemplateInDefaultValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
	}{
		{name: "a comma expression in a substitution", source: "function f(x = `${a, b}`, y: number) {}"},
		{name: "a comma expression in a later substitution", source: "function f(x = `${a}-${b, c}`, y: number) {}"},
		{name: "a call with arguments in a substitution", source: "function f(x = `${fn(a, b)}`, y: number) {}"},
		{name: "a nested template with a comma expression", source: "function f(x = `${`${a, b}`}`, y: number) {}"},
		{name: "non-ASCII text and a comma expression", source: "function f(x = `£${a, b} €`, y: number) {}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractWithinTimeout(t, tc.source)

			require.Contains(t, result, "f")
			params := result["f"]
			require.Len(t, params, 2, "a comma inside a substitution does not start a parameter")
			assert.Equal(t, "x", params[0].Name)
			assert.True(t, params[0].Optional)
			assert.Equal(t, "y", params[1].Name)
			assert.Equal(t, categoryNumber, params[1].Category)
		})
	}
}

func TestExtractFunctionParams_StopsOnUnlexableSource(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
	}{
		{name: "a non-ASCII character outside any string", source: "function first(a: string) {}\nconst x = £;\nfunction after(count: number) {}"},
		{name: "an unterminated template", source: "function first(a: string) {}\nconst x = `£${"},
		{name: "a stray closing brace", source: "function first(a: string) {}\n}}} £ }"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractWithinTimeout(t, tc.source)

			require.Contains(t, result, "first", "functions found before the bad source are kept")
		})
	}
}

func extractWithinTimeout(t *testing.T, source string) map[string][]ParamInfo {
	t.Helper()

	done := make(chan map[string][]ParamInfo, 1)
	go func() {
		done <- ExtractFunctionParams(source)
	}()

	select {
	case result := <-done:
		return result
	case <-time.After(extractionTimeout):
		t.Fatalf("ExtractFunctionParams did not return within %s", extractionTimeout)
		return nil
	}
}
