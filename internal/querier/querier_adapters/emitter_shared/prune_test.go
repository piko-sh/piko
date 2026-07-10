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

package emitter_shared

import (
	"bytes"
	"errors"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safedisk"
)

type fakeDirEntry struct {
	name  string
	isDir bool
}

func (entry fakeDirEntry) Name() string         { return entry.name }
func (entry fakeDirEntry) IsDir() bool          { return entry.isDir }
func (fakeDirEntry) Type() fs.FileMode          { return 0 }
func (fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type fakeFileHandle struct {
	safedisk.FileHandle
	reader  *bytes.Reader
	readErr error
}

func (handle *fakeFileHandle) Read(destination []byte) (int, error) {
	if handle.readErr != nil {
		return 0, handle.readErr
	}
	return handle.reader.Read(destination)
}

func (*fakeFileHandle) Close() error { return nil }

type fakeOutputDir struct {
	files      map[string][]byte
	dirs       map[string]bool
	openErr    map[string]error
	readErr    map[string]error
	removeErr  map[string]error
	readDirErr error
}

func (dir *fakeOutputDir) ReadDir(string) ([]fs.DirEntry, error) {
	if dir.readDirErr != nil {
		return nil, dir.readDirErr
	}
	entries := make([]fs.DirEntry, 0, len(dir.files))
	for name := range dir.files {
		entries = append(entries, fakeDirEntry{name: name, isDir: dir.dirs[name]})
	}
	return entries, nil
}

func (dir *fakeOutputDir) Open(name string) (safedisk.FileHandle, error) {
	if err := dir.openErr[name]; err != nil {
		return nil, err
	}
	if err := dir.readErr[name]; err != nil {
		return &fakeFileHandle{readErr: err}, nil
	}
	content, ok := dir.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &fakeFileHandle{reader: bytes.NewReader(content)}, nil
}

func (dir *fakeOutputDir) Remove(name string) error {
	if err := dir.removeErr[name]; err != nil {
		return err
	}
	delete(dir.files, name)
	return nil
}

func TestPruneStaleGeneratedFilesRemovesOnlyStaleGeneratedFiles(t *testing.T) {
	t.Parallel()

	generated := []byte(GeneratedFileHeader + "package db\n")
	dir := &fakeOutputDir{
		files: map[string][]byte{
			"stale.sql.go":   generated,
			"keep.sql.go":    generated,
			"models.go":      generated,
			"handwritten.go": []byte("package db\n"),
			"tiny.go":        []byte("//"),
			"subdir":         generated,
		},
		dirs: map[string]bool{"subdir": true},
	}

	keep := map[string]struct{}{"keep.sql.go": {}, "models.go": {}}

	require.NoError(t, PruneStaleGeneratedFiles(dir, keep))

	assert.NotContains(t, dir.files, "stale.sql.go", "a generated file absent from keep must be removed")
	assert.Contains(t, dir.files, "keep.sql.go", "a generated file still in keep must survive")
	assert.Contains(t, dir.files, "models.go", "a generated file still in keep must survive")
	assert.Contains(t, dir.files, "handwritten.go", "a file without the generated header must never be removed")
	assert.Contains(t, dir.files, "tiny.go", "a file shorter than the header must never be removed")
	assert.Contains(t, dir.files, "subdir", "a directory entry must never be removed")
}

func TestPruneStaleGeneratedFilesReturnsReadDirError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("cannot list directory")
	err := PruneStaleGeneratedFiles(&fakeOutputDir{readDirErr: sentinel}, nil)

	require.ErrorIs(t, err, sentinel)
	assert.ErrorContains(t, err, "listing output directory")
}

func TestPruneStaleGeneratedFilesReturnsInspectError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("permission denied")
	dir := &fakeOutputDir{
		files:   map[string][]byte{"candidate.sql.go": []byte(GeneratedFileHeader)},
		openErr: map[string]error{"candidate.sql.go": sentinel},
	}

	err := PruneStaleGeneratedFiles(dir, nil)

	require.ErrorIs(t, err, sentinel)
	assert.ErrorContains(t, err, `inspecting candidate stale file "candidate.sql.go"`)
	assert.Contains(t, dir.files, "candidate.sql.go", "a file that cannot be inspected must not be removed")
}

func TestPruneStaleGeneratedFilesReturnsReadError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("input/output error")
	dir := &fakeOutputDir{
		files:   map[string][]byte{"candidate.sql.go": []byte(GeneratedFileHeader + "package db\n")},
		readErr: map[string]error{"candidate.sql.go": sentinel},
	}

	err := PruneStaleGeneratedFiles(dir, nil)

	require.ErrorIs(t, err, sentinel)
	assert.ErrorContains(t, err, `inspecting candidate stale file "candidate.sql.go"`)
	assert.Contains(t, dir.files, "candidate.sql.go", "a file that cannot be read must not be removed")
}

func TestPruneStaleGeneratedFilesReturnsRemoveError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("device busy")
	dir := &fakeOutputDir{
		files:     map[string][]byte{"stale.sql.go": []byte(GeneratedFileHeader + "package db\n")},
		removeErr: map[string]error{"stale.sql.go": sentinel},
	}

	err := PruneStaleGeneratedFiles(dir, nil)

	require.ErrorIs(t, err, sentinel)
	assert.ErrorContains(t, err, `removing stale generated file "stale.sql.go"`)
}

func TestPruneStaleGeneratedFilesTreatsVanishedFileAsPruned(t *testing.T) {
	t.Parallel()

	dir := &fakeOutputDir{
		files:     map[string][]byte{"gone.sql.go": []byte(GeneratedFileHeader + "package db\n")},
		openErr:   map[string]error{"gone.sql.go": fs.ErrNotExist},
		removeErr: map[string]error{},
	}

	require.NoError(t, PruneStaleGeneratedFiles(dir, nil),
		"a file that vanished before it could be inspected must not fail the prune")
}

func TestPruneStaleGeneratedFilesIgnoresRemoveNotExist(t *testing.T) {
	t.Parallel()

	dir := &fakeOutputDir{
		files:     map[string][]byte{"stale.sql.go": []byte(GeneratedFileHeader + "package db\n")},
		removeErr: map[string]error{"stale.sql.go": fs.ErrNotExist},
	}

	require.NoError(t, PruneStaleGeneratedFiles(dir, nil),
		"a file removed by another writer between listing and removal must not fail the prune")
}

func TestGeneratedFileNameSetExcludesEmptyContent(t *testing.T) {
	t.Parallel()

	files := []querier_dto.GeneratedFile{
		{Name: "models.go", Content: []byte("x")},
		{Name: "empty.sql.go", Content: nil},
	}

	names := GeneratedFileNameSet(files)

	assert.Contains(t, names, "models.go")
	assert.NotContains(t, names, "empty.sql.go", "empty-content files are not written, so must not be kept")
}
