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

package casefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndReadRoundTrip(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "cases", "one")
	original := sampleCase()
	require.NoError(t, Write(directory, &original))
	loaded, err := Read(directory)
	require.NoError(t, err)
	assert.Equal(t, original.Files, loaded.Files)
	assert.Equal(t, []string{"cmd/main.go", "pages/index.pk"}, loaded.Paths)
	assert.Equal(t, original.Hash(), loaded.Hash())
	entries, err := os.ReadDir(filepath.Dir(directory))
	require.NoError(t, err)
	require.Len(t, entries, 1, "staging directory left behind")
}

func TestWriteRefusesExistingDestination(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "case")
	require.NoError(t, os.Mkdir(directory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "marker"), []byte("kept"), 0o600))
	err := Write(directory, new(sampleCase()))
	require.ErrorIs(t, err, errDestinationExists)
	assert.Contains(t, err.Error(), directory)
	data, err := os.ReadFile(filepath.Join(directory, "marker"))
	require.NoError(t, err)
	assert.Equal(t, "kept", string(data))
}

func TestWriteRemovesOnlyStaleStagingDirectories(t *testing.T) {
	parent := t.TempDir()
	stale := filepath.Join(parent, temporaryPrefix+"stale")
	fresh := filepath.Join(parent, temporaryPrefix+"fresh")
	require.NoError(t, os.Mkdir(stale, 0o700))
	require.NoError(t, os.Mkdir(fresh, 0o700))
	old := time.Now().Add(-2 * staleTemporaryAge)
	require.NoError(t, os.Chtimes(stale, old, old))
	require.NoError(t, Write(filepath.Join(parent, "case"), new(sampleCase())))
	assert.NoDirExists(t, stale)
	assert.DirExists(t, fresh)
}

func TestWriteRejectsLimitsAndUnsafePaths(t *testing.T) {
	tooMany := sampleCase()
	tooMany.Files = map[string]string{}
	for index := range maxFiles + 1 {
		tooMany.Files[filepath.Join("pages", strings.Repeat("a", index+1)+".pk")] = ""
	}
	oversized := sampleCase()
	oversized.Files["pages/big.pk"] = strings.Repeat("x", maxFile+1)
	tree := sampleCase()
	tree.Files = map[string]string{}
	for index := range 9 {
		tree.Files[filepath.Join("pages", strings.Repeat("b", index+1)+".pk")] = strings.Repeat("x", maxFile)
	}
	unsafe := sampleCase()
	unsafe.Files["../escape.go"] = "package escape"
	testCases := []struct {
		name string
		c    Case
		want error
	}{
		{name: "file count", c: tooMany, want: errFileCountLimit},
		{name: "file size", c: oversized, want: errSizeLimit},
		{name: "tree size", c: tree, want: errSizeLimit},
		{name: "traversal", c: unsafe, want: errInvalidPath},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := filepath.Join(t.TempDir(), "case")
			require.ErrorIs(t, Write(directory, &testCase.c), testCase.want)
			assert.NoDirExists(t, directory)
		})
	}
}

func TestValidatePath(t *testing.T) {
	for _, name := range []string{"pages/index.pk", "cmd/main/main.go", "a"} {
		assert.NoError(t, validatePath(name), name)
	}
	for _, name := range []string{"", ".", "..", "../x", "pages/../../x", "/abs", "pages//x", "pages/./x", "c:x", `pages\x`, "pages/"} {
		assert.ErrorIs(t, validatePath(name), errInvalidPath, name)
	}
}

func TestReadRejectsInvalidManifests(t *testing.T) {
	testCases := []struct {
		name     string
		manifest string
		files    map[string]string
		want     error
		message  string
	}{
		{name: "version", manifest: `{"Version":2,"Target":"pages"}`, want: errInvalidManifest},
		{name: "target", manifest: `{"Version":1}`, want: errInvalidManifest},
		{name: "traversal", manifest: `{"Version":1,"Target":"pages","Paths":["../outside.go"]}`, want: errInvalidPath},
		{name: "duplicate", manifest: `{"Version":1,"Target":"pages","Paths":["a.go","a.go"]}`, files: map[string]string{"a.go": "package a"}, want: errInvalidManifest},
		{name: "count", manifest: `{"Version":1,"Target":"pages","Paths":[` + strings.TrimSuffix(strings.Repeat(`"a",`, maxFiles+1), ",") + `]}`, want: errFileCountLimit},
		{name: "missing source", manifest: `{"Version":1,"Target":"pages","Paths":["absent.go"]}`, message: "absent.go"},
		{name: "malformed", manifest: `{`, message: "decoding manifest"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "case.json"), []byte(testCase.manifest), 0o600))
			for name, content := range testCase.files {
				target := filepath.Join(directory, "src", name)
				require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o700))
				require.NoError(t, os.WriteFile(target, []byte(content), 0o600))
			}
			_, err := Read(directory)
			require.Error(t, err)
			assert.Contains(t, err.Error(), directory)
			if testCase.want != nil {
				assert.ErrorIs(t, err, testCase.want)
			}
			if testCase.message != "" {
				assert.Contains(t, err.Error(), testCase.message)
			}
		})
	}
}

func TestReadRejectsOversizedSource(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "case.json"), []byte(`{"Version":1,"Target":"pages","Paths":["big.pk"]}`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "src"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "src", "big.pk"), make([]byte, maxFile+1), 0o600))
	_, err := Read(directory)
	require.ErrorIs(t, err, errSizeLimit)
}

func TestReadRejectsSymlinkEscape(t *testing.T) {
	outside := filepath.Join(t.TempDir(), "secret.go")
	require.NoError(t, os.WriteFile(outside, []byte("package secret"), 0o600))
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "case.json"), []byte(`{"Version":1,"Target":"pages","Paths":["link.go"]}`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(directory, "src"), 0o700))
	require.NoError(t, os.Symlink(outside, filepath.Join(directory, "src", "link.go")))
	_, err := Read(directory)
	require.Error(t, err)
}

func TestHashIsSensitiveToSources(t *testing.T) {
	c := sampleCase()
	digest := c.Hash()
	assert.Regexp(t, `^[0-9a-f]{64}$`, digest)
	assert.Equal(t, digest, c.Hash())
	c.Files["pages/index.pk"] += " "
	assert.NotEqual(t, digest, c.Hash())
}

func TestCanonicalSHA256IgnoresFieldOrder(t *testing.T) {
	type forward struct {
		Name  string
		Label string
		Count int64
	}
	type reversed struct {
		Label string
		Name  string
		Count int64
	}
	first, err := CanonicalSHA256(forward{Name: "a", Label: "b", Count: 1 << 60})
	require.NoError(t, err)
	second, err := CanonicalSHA256(reversed{Label: "b", Name: "a", Count: 1 << 60})
	require.NoError(t, err)
	assert.Equal(t, first, second)

	third, err := CanonicalSHA256(forward{Name: "a", Label: "b", Count: 1<<60 + 1})
	require.NoError(t, err)
	assert.NotEqual(t, first, third, "large integers must keep their exact value")
}

func TestCanonicalSHA256ReportsUnencodableValues(t *testing.T) {
	_, err := CanonicalSHA256(map[string]any{"value": make(chan int)})
	assert.Error(t, err)
}

func sampleCase() Case {
	return Case{
		Version: Version,
		Target:  "pages",
		Request: "/",
		Files:   map[string]string{"pages/index.pk": "<template><p>hi</p></template>", "cmd/main.go": "package main"},
	}
}

func TestRequiresProjectInterpreterHost(t *testing.T) {
	assert.False(t, (&Case{Target: "pages"}).RequiresProjectInterpreterHost())
	assert.True(t, (&Case{Target: "partial-components"}).RequiresProjectInterpreterHost())
	assert.True(t, (&Case{Target: "content-editions"}).RequiresProjectInterpreterHost())
	assert.True(t, (&Case{Target: "pages", Expected: Expected{FormWorkflow: &FormWorkflow{}}}).RequiresProjectInterpreterHost())
}
