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

package browser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareGolden(t *testing.T) {
	t.Chdir(t.TempDir())

	err := compareGolden(t, "card", "<p>one</p>")
	require.ErrorContains(t, err, "PIKO_UPDATE_GOLDEN=1")

	t.Setenv("PIKO_UPDATE_GOLDEN", "1")
	require.NoError(t, compareGolden(t, "nested/card", "<p>one</p>"))
	written, err := os.ReadFile(filepath.Join(goldenDir, "nested", "card.html"))
	require.NoError(t, err)
	assert.Equal(t, "<p>one</p>", string(written))

	t.Setenv("PIKO_UPDATE_GOLDEN", "")
	require.NoError(t, compareGolden(t, "nested/card", "<p>one</p>"))
	require.ErrorContains(t, compareGolden(t, "nested/card", "<p>two</p>"), "golden file mismatch")
}

func TestCompareGoldenReportsUnwritableDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.WriteFile("testdata", []byte("not a directory"), 0o600))
	t.Setenv("PIKO_UPDATE_GOLDEN", "1")
	require.ErrorContains(t, compareGolden(t, "card", "<p>one</p>"), "creating golden directory")
}

func TestCompareGoldenReportsUnwritableFile(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(filepath.Join(goldenDir, "card.html"), 0o750))
	t.Setenv("PIKO_UPDATE_GOLDEN", "1")
	require.ErrorContains(t, compareGolden(t, "card", "<p>one</p>"), "writing golden file")
}
