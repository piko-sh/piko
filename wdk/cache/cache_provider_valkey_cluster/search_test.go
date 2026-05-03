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

package cache_provider_valkey_cluster

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEscapeTagValue_EscapesAllSpecialCharacters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		want string
	}{
		{"plain alphanumerics untouched", "hello123", "hello123"},
		{"comma escaped", "a,b", "a\\,b"},
		{"colons escaped", "ns:value", "ns\\:value"},
		{"dot escaped", "a.b", "a\\.b"},
		{"space escaped", "hello world", "hello\\ world"},
		{"hyphen escaped", "x-y", "x\\-y"},
		{"plus escaped", "1+2", "1\\+2"},
		{"complex value escapes everything", "(foo!)*", "\\(foo\\!\\)\\*"},
		{"integer formatted", 42, "42"},
		{"float formatted", 3.14, "3\\.14"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, escapeTagValue(tc.in))
		})
	}
}

func TestIsUnknownIndexError_DetectsUnknownPhrase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("Unknown index name"), true},
		{errors.New("ERR Unknown command"), true},
		{errors.New("connection refused"), false},
		{errors.New(""), false},
	}

	for _, tc := range cases {
		require.Equalf(t, tc.want, isUnknownIndexError(tc.err), "error: %q", tc.err.Error())
	}
}

func TestExtractJSONFromDocData_ReturnsEmptyForNilInput(t *testing.T) {
	t.Parallel()

	require.Empty(t, extractJSONFromDocData(nil))
}
