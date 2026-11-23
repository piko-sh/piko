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

package runtimecoverage

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/safedisk"
)

const (
	testProfile = "mode: atomic\npiko.sh/piko/serve.go:1.1,2.1 3 99\npiko.sh/piko/serve.go:3.1,5.1 4 0\nexample.org/pikofuzzcase/dist/page.go:1.1,1.3 2 1\npipit.sh/pipit/eval.go:1.1,2.1 1 0\nexample.org/pikofuzzcase/fuzzhost/main.go:1.1,2.1 2 1\n"
)

func TestGoDenominatorsAndOrdinaryCompressedProfile(t *testing.T) {
	directory := t.TempDir()
	input := filepath.Join(directory, "text")
	require.NoError(t, os.WriteFile(input, []byte(testProfile), 0o600))
	output := filepath.Join(directory, goArtefact)
	root := sandbox(t, directory)
	groups, total, err := compressGo(root, "text", root, goArtefact)
	require.NoError(t, err)
	assert.Equal(t, Counts{Covered: 7, Total: 12}, total)
	assert.Equal(t, Counts{Covered: 2, Total: 2}, groups["test host"])
	assert.Equal(t, Counts{Covered: 2, Total: 2}, groups["generated application"])
	assert.Equal(t, Counts{Covered: 3, Total: 7}, groups["Piko"])
	assert.Equal(t, Counts{Covered: 0, Total: 1}, groups["Pipit"])

	file, err := os.Open(output)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testProfile, string(decoded), "lost ordinary Go profile")
}

func TestCompressGoRejectsMalformedProfiles(t *testing.T) {
	for name, source := range map[string]string{
		"missing header": "mode: set\n",
		"short record":   "mode: atomic\npiko.sh/piko/a.go:1.1,2.1 3\n",
		"negative":       "mode: atomic\npiko.sh/piko/a.go:1.1,2.1 -1 1\n",
		"bad visits":     "mode: atomic\npiko.sh/piko/a.go:1.1,2.1 1 x\n",
		"no piko":        "mode: atomic\npipit.sh/pipit/a.go:1.1,2.1 1 1\n",
	} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			input := filepath.Join(directory, "text")
			require.NoError(t, os.WriteFile(input, []byte(source), 0o600))
			root := sandbox(t, directory)
			_, _, err := compressGo(root, "text", root, goArtefact)
			require.Error(t, err)
		})
	}
}

func TestRejectOversizedCoverageArtefactsAndCounterInconsistency(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, JSArtefact)
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, file.Truncate(MaxJS+1))
	require.NoError(t, file.Close())
	root := sandbox(t, directory)
	_, err = readBrowser(root, JSArtefact)
	require.ErrorIs(t, err, errFileTooLarge, "accepted oversized browser result")

	browser := validBrowser()
	write := func() {
		t.Helper()
		data, err := json.Marshal(browser)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0o600))
	}
	write()
	_, err = readBrowser(root, JSArtefact)
	require.NoError(t, err)
	browser.Functions.Covered = 2
	write()
	_, err = readBrowser(root, JSArtefact)
	require.ErrorIs(t, err, errBrowserTotals, "accepted summary disagreement")
}

func TestCompressGoRejectsOversizedRawProfile(t *testing.T) {
	directory := t.TempDir()
	file, err := os.Create(filepath.Join(directory, "text"))
	require.NoError(t, err)
	require.NoError(t, file.Truncate(MaxRawGo+1))
	require.NoError(t, file.Close())
	root := sandbox(t, directory)

	_, _, err = compressGo(root, "text", root, goArtefact)

	require.ErrorIs(t, err, errFileTooLarge)
	assert.NoFileExists(t, filepath.Join(directory, goArtefact))
}

func TestReadBoundRefusesSymlinkOutsideRoot(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.json")
	require.NoError(t, os.WriteFile(outside, []byte("{}"), 0o600))
	directory := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(directory, JSArtefact)))

	_, err := readBound(sandbox(t, directory), JSArtefact, MaxJS)

	require.Error(t, err)
}

func sandbox(t *testing.T, directory string) safedisk.Sandbox {
	t.Helper()
	root, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, root.Close()) })
	return root
}

func validBrowser() Browser {
	return Browser{
		Functions: Counts{Covered: 1, Total: 2},
		Scripts: []Script{{
			Path:   "/loaded.js",
			SHA256: strings.Repeat("1", 64),
			Functions: []Function{
				{Ranges: []Range{{Start: 0, End: 25, Count: 1}, {Start: 10, End: 12, Count: 0}}},
				{Ranges: []Range{{Start: 15, End: 20, Count: 0}}},
			},
		}},
	}
}
