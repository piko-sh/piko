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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestReadProjectCorpusRejectsMalformedEntries(t *testing.T) {
	for _, body := range []string{"wrong", "go test fuzz v1\nint(1)\n", "go test fuzz v1\n[]byte(\"a\")\n[]byte(\"b\")\n", "go test fuzz v1\n[]byte(\"\\q\")\n"} {
		_, err := readProjectCorpus(strings.NewReader(body))
		require.ErrorIs(t, err, errCorpusFormat, body)
	}
	_, err := readProjectCorpus(strings.NewReader(strings.Repeat("X", maxCorpusBytes+1)))
	require.ErrorIs(t, err, errCorpusTooLarge)
}

func TestPromotedCorpusMatchesNativeEntry(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "corpus")
	out := filepath.Join(directory, "case")
	require.NoError(t, os.WriteFile(input, []byte("go test fuzz v1\n[]byte(\"\\x02\\xff\")\n"), 0o600))
	for range 2 {
		require.NoError(t, promote(input, out))
	}
	promoted, err := casefile.Read(out)
	require.NoError(t, err)
	want := generate.NativeProject([]byte{2, 255})
	assert.Equal(t, want.Hash(), promoted.Hash(), "native and promoted cases differ")
	assert.Equal(t, uint64(32), promoted.Index)
	require.NoError(t, os.WriteFile(input, []byte("go test fuzz v1\n[]byte(\"other\")\n"), 0o600))
	require.Error(t, promote(input, out), "overwrote a different case")
	require.Error(t, promote("", out))
	require.Error(t, promote(filepath.Join(directory, "missing"), out))
}
