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

package db_engine_duckdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectParenthesised(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantError    error
		name         string
		input        string
		wantValues   []string
		wantPosition int
		wantAnyError bool
	}{
		{name: "empty group", input: "() x", wantValues: nil, wantPosition: 2},
		{name: "nested groups", input: "(a, (b)) x", wantValues: []string{"a", ",", "(", "b", ")"}, wantPosition: 7},
		{name: "not a group", input: "a", wantAnyError: true},
		{name: "unbalanced group", input: "(a, (b)", wantError: errUnmatchedParenthesis},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tokens, err := tokenise(testCase.input)
			require.NoError(t, err)
			parser := newParser(tokens)

			inner, err := parser.collectParenthesised()

			switch {
			case testCase.wantError != nil:
				require.ErrorIs(t, err, testCase.wantError)
				return
			case testCase.wantAnyError:
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var values []string
			for _, tok := range inner {
				values = append(values, tok.value)
			}
			assert.Equal(t, testCase.wantValues, values)
			assert.Equal(t, testCase.wantPosition, parser.position)
			assert.Equal(t, len(inner), cap(inner), "the view must not expose the tokens after the group")
		})
	}
}
