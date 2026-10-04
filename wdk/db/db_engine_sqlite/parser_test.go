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

package db_engine_sqlite

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectParenthesisedReturnsTheGroupWithoutCopying(t *testing.T) {
	t.Parallel()

	tokens, err := tokenise("(a, (b)) c")
	require.NoError(t, err)
	p := newParser(tokens)

	inner, err := p.collectParenthesised()

	require.NoError(t, err)
	require.Len(t, inner, 5)
	assert.Equal(t, "a", inner[0].value)
	assert.Same(t, &tokens[1], &inner[0], "the group must be a view of the statement's tokens")
	assert.Equal(t, len(inner), cap(inner), "the view must be capacity-capped so appends cannot overwrite later tokens")
	assert.Equal(t, "c", p.current().value)
}

func TestCollectParenthesisedRejectsMalformedGroups(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantErr error
		name    string
		sql     string
	}{
		{name: "unterminated group", sql: "(a, (b)", wantErr: errUnmatchedParenthesis},
		{name: "missing opening parenthesis", sql: "a)", wantErr: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			tokens, err := tokenise(testCase.sql)
			require.NoError(t, err)

			inner, err := newParser(tokens).collectParenthesised()

			require.Error(t, err)
			assert.Nil(t, inner)
			if testCase.wantErr != nil {
				assert.ErrorIs(t, err, testCase.wantErr)
			}
		})
	}
}

func TestRecordSyntaxErrorKeepsTheFirstError(t *testing.T) {
	t.Parallel()

	first := errors.New("first")
	parent := newParser(nil)
	child := newParser(nil)

	child.recordSyntaxError(first)
	child.recordSyntaxError(errors.New("second"))
	parent.adoptSyntaxError(newParser(nil))
	parent.adoptSyntaxError(child)

	assert.Same(t, first, parent.syntaxError)
}
