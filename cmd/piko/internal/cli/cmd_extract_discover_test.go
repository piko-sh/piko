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

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
)

func TestRenderDiscoverResultListFormat(t *testing.T) {
	t.Parallel()

	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/b/two"},
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderDiscoverResult(&stdout, &stderr, "list", result))

	require.Equal(t, "github.com/a/one\ngithub.com/b/two\n", stdout.String())
	require.Empty(t, stderr.String())
}

func TestRenderDiscoverResultYAMLFormat(t *testing.T) {
	t.Parallel()

	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/b/two"},
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderDiscoverResult(&stdout, &stderr, "yaml", result))

	out := stdout.String()
	require.True(t, strings.HasPrefix(out, "packages:\n"), "yaml output must start with packages: header")
	require.Contains(t, out, "  - github.com/a/one")
	require.Contains(t, out, "  - github.com/b/two")
}

func TestRenderDiscoverResultJSONFormat(t *testing.T) {
	t.Parallel()

	result := driver_symbols_extract.DiscoverResult{
		RequiredImports:   []string{"github.com/a/one"},
		SkippedCgo:        []string{"github.com/example/cgo"},
		GenericCandidates: []string{"github.com/example/generic"},
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderDiscoverResult(&stdout, &stderr, "json", result))

	out := stdout.String()
	require.Contains(t, out, `"RequiredImports"`)
	require.Contains(t, out, `"SkippedCgo"`)
	require.Contains(t, out, `"GenericCandidates"`)
	require.Contains(t, out, "github.com/a/one")
}

func TestRenderDiscoverResultRoutesWarningsToStderr(t *testing.T) {
	t.Parallel()

	result := driver_symbols_extract.DiscoverResult{
		RequiredImports:   []string{"github.com/a/one"},
		SkippedCgo:        []string{"github.com/example/cgo"},
		GenericCandidates: []string{"github.com/example/generic"},
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderDiscoverResult(&stdout, &stderr, "list", result))

	require.Contains(t, stdout.String(), "github.com/a/one")
	require.NotContains(t, stdout.String(), "cgo")

	require.Contains(t, stderr.String(), "github.com/example/cgo")
	require.Contains(t, stderr.String(), "uses cgo")
	require.Contains(t, stderr.String(), "github.com/example/generic")
	require.Contains(t, stderr.String(), "generic types")
}

func TestRenderDiscoverResultUnknownFormatErrors(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := renderDiscoverResult(&stdout, &stderr, "xml", driver_symbols_extract.DiscoverResult{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported format")
}

func TestSplitIgnoreList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "", want: nil},
		{name: "single", in: "github.com/a/one", want: []string{"github.com/a/one"}},
		{name: "multiple", in: "github.com/a/one,github.com/b/two", want: []string{"github.com/a/one", "github.com/b/two"}},
		{name: "whitespace trimmed", in: "  github.com/a/one  , github.com/b/two  ", want: []string{"github.com/a/one", "github.com/b/two"}},
		{name: "empty entries dropped", in: "a,,b,", want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, splitIgnoreList(tt.in))
		})
	}
}
