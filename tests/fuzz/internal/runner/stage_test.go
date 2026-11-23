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

package runner

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

func TestCollectionStagedWithLocalModuleReplacement(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	app, workspaceFile, err := stage(repo, t.TempDir(), new(generate.New("collection-routes", 7, 1)))
	require.NoError(t, err)
	source := filepath.Join(app, casefile.CollectionDir)
	data, err := os.ReadFile(workspaceFile)
	require.NoError(t, err)
	workspace, err := modfile.ParseWork(workspaceFile, data, nil)
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(workspace.Use, func(use *modfile.Use) bool { return use.Path == source }), "collection absent from private workspace")
	data, err = os.ReadFile(filepath.Join(app, "go.mod"))
	require.NoError(t, err)
	module, err := modfile.Parse("go.mod", data, nil)
	require.NoError(t, err)
	assert.True(t, slices.ContainsFunc(module.Replace, func(replacement *modfile.Replace) bool {
		return replacement.Old.Path == casefile.CollectionModule && replacement.New.Path == source && replacement.New.Version == ""
	}), "missing offline source resolution")
}

func TestNativeHostFixtureEmissionBaseline(t *testing.T) {
	const digest = "c0782fdf6bc9e901edcab10afcb41de0feda5dd1c9d5532fb6c000a0c51572a3"
	assert.Equal(t, digest, fmt.Sprintf("%x", sha256.Sum256([]byte(nativeMain))))
}

func TestLegacyGoFixtureSourceStagedVerbatim(t *testing.T) {
	const path = "internal/formfixture/provider.go"
	const source = "package formfixture\nconst ActionName=\"saved-π\" // archived reduced case\n"
	saved := casefile.Case{
		Version:   casefile.Version,
		Generator: "piko-project/5",
		Profile:   "small-project/5",
		Target:    "form-shim-contracts",
		Files:     map[string]string{path: source},
	}
	directory := filepath.Join(t.TempDir(), "saved")
	require.NoError(t, casefile.Write(directory, &saved))
	replayed, err := casefile.Read(directory)
	require.NoError(t, err)
	require.Equal(t, saved.Generator, replayed.Generator, "saved fixture identity changed")
	require.Equal(t, saved.Profile, replayed.Profile, "saved fixture identity changed")
	require.Equal(t, saved.Hash(), replayed.Hash(), "saved fixture identity changed")
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	app, _, err := stage(repo, t.TempDir(), &replayed)
	require.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(app, path))
	require.NoError(t, err)
	assert.Equal(t, source, string(data), "staged source must keep the archived bytes")
}

func TestCopyRequiredRefusesOversizedFile(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "source")
	destination := filepath.Join(directory, "destination")
	require.NoError(t, os.WriteFile(source, []byte("abcd"), 0600))

	data, err := readBoundedFile(directory, "source", 3)
	assert.ErrorIs(t, err, errFileOversize)
	assert.Nil(t, data)

	data, err = readBoundedFile(directory, "source", 4)
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), data)

	require.NoError(t, copyRequired(directory, "source", destination))
	copied, err := os.ReadFile(destination)
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), copied)
	assert.NoError(t, copyIfPresent(directory, "absent", destination))
}

func TestReadBoundedFileStaysInsideItsRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	require.NoError(t, os.Mkdir(root, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(parent, "outside"), []byte("secret"), 0600))
	require.NoError(t, os.Symlink(filepath.Join(parent, "outside"), filepath.Join(root, "link")))

	_, err := readBoundedFile(root, "../outside", 64)
	assert.Error(t, err)

	_, err = readBoundedFile(root, "link", 64)
	assert.Error(t, err)
}
