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

package generator_adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func newGoBlobEmbedFixture(t *testing.T) (*DrivenEmbedEmitter, string) {
	t.Helper()
	root := t.TempDir()
	pikoDir := filepath.Join(root, ".piko")
	distDir := filepath.Join(root, "dist")

	files := map[string]string{
		".piko/blobs/source/aaaa1111.go":   "package alpha\n",
		".piko/blobs/source/bbbb2222.go":   "package beta\n",
		".piko/blobs/source/cccc3333.GO":   "not compiled by the go tool\n",
		".piko/blobs/source/dddd4444.css":  "body{}\n",
		".piko/storage/uploads/helper.go":  "package gamma\n",
		".piko/storage/uploads/readme.txt": "storage-bytes",
	}
	for name, content := range files {
		full := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750), "fixture directories must create")
		require.NoError(t, os.WriteFile(full, []byte(content), 0o640), "fixture files must write")
	}
	require.NoError(t, os.MkdirAll(distDir, 0o750), "the dist fixture directory must create")

	factory, err := safedisk.NewCLIFactory(root)
	require.NoError(t, err, "the sandbox factory must create")
	source, err := factory.Create("embed-go-blob-source", pikoDir, safedisk.ModeReadOnly)
	require.NoError(t, err, "the source sandbox must create")
	destination, err := factory.Create("embed-go-blob-output", distDir, safedisk.ModeReadWrite)
	require.NoError(t, err, "the destination sandbox must create")

	return NewDrivenEmbedEmitter(source, destination), distDir
}

func TestEmbedEmitter_FailsWhenThePayloadHoldsGoSource(t *testing.T) {
	t.Parallel()

	emitter, _ := newGoBlobEmbedFixture(t)
	err := emitter.Emit(t.Context())

	require.Error(t, err, "a Go file in the payload must fail the build, not vanish from it")
	require.ErrorIs(t, err, ErrEmbedGoSourceAsset)
	assert.Contains(t, err.Error(), "blobs/source/aaaa1111.go")
	assert.Contains(t, err.Error(), "Move these files out of the")
}

func TestEmbedEmitter_GoSourceErrorNamesAtMostTenPaths(t *testing.T) {
	t.Parallel()

	paths := make([]string, embedGoSourceListLimit+5)
	for index := range paths {
		paths[index] = fmt.Sprintf("blobs/source/%02d.go", index)
	}

	err := newEmbedGoSourceError(paths)

	require.ErrorIs(t, err, ErrEmbedGoSourceAsset)
	assert.Contains(t, err.Error(), "and 5 more")
	assert.NotContains(t, err.Error(), "blobs/source/12.go")
}

func TestIsGoSourcePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "go blob is excluded", path: "blobs/source/aaaa1111.go", expected: true},
		{name: "bare go name is excluded", path: "x.go", expected: true},
		{name: "upper case extension is kept", path: "blobs/source/aaaa1111.GO", expected: false},
		{name: "css blob is kept", path: "blobs/source/aaaa1111.css", expected: false},
		{name: "extensionless blob is kept", path: "blobs/source/aaaa1111", expected: false},
		{name: "go inside the name is kept", path: "blobs/source/cargo.txt", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, isGoSourcePath(tt.path))
		})
	}
}
