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
	"fmt"
	"io"
	"io/fs"

	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safedisk"
)

// GeneratedOutputDir is the minimal filesystem surface PruneStaleGeneratedFiles needs to
// remove obsolete generated files, satisfied by a safedisk sandbox rooted at the output
// directory.
type GeneratedOutputDir interface {
	// ReadDir lists the entries of the directory named name.
	//
	// Takes name (string) which is the directory path to list.
	//
	// Returns []fs.DirEntry which holds the directory entries.
	// Returns error when the directory cannot be read.
	ReadDir(name string) ([]fs.DirEntry, error)

	// Open opens the file named name for reading.
	//
	// Takes name (string) which is the path of the file to open.
	//
	// Returns safedisk.FileHandle which is the open file, and which the caller must close.
	// Returns error when the file cannot be opened.
	Open(name string) (safedisk.FileHandle, error)

	// Remove deletes the file named name.
	//
	// Takes name (string) which is the path of the file to delete.
	//
	// Returns error when the file cannot be deleted.
	Remove(name string) error
}

// GeneratedFileNameSet returns the names of the files a generation run will actually
// write.
//
// Only files carrying content are included; it is the keep set to pass to
// PruneStaleGeneratedFiles.
//
// Takes files ([]querier_dto.GeneratedFile) which are the freshly generated files.
//
// Returns map[string]struct{} which is the set of names that must survive a prune.
func GeneratedFileNameSet(files []querier_dto.GeneratedFile) map[string]struct{} {
	names := make(map[string]struct{}, len(files))
	for _, file := range files {
		if len(file.Content) == 0 {
			continue
		}
		names[file.Name] = struct{}{}
	}
	return names
}

// PruneStaleGeneratedFiles removes files left over from a previous generation run.
//
// A renamed or deleted query no longer collides with the fresh output. Only files that
// carry the piko GeneratedFileHeader and are absent from keep are removed, so
// hand-written files sharing the directory are never touched. A file that has vanished
// since the directory listing is treated as already pruned.
//
// Takes dir (GeneratedOutputDir) which is the output directory sandbox.
// Takes keep (map[string]struct{}) which is the set of filenames about to be written.
//
// Returns error when the directory cannot be listed or a stale file cannot be inspected
// or removed for an unexpected reason.
func PruneStaleGeneratedFiles(dir GeneratedOutputDir, keep map[string]struct{}) error {
	entries, err := dir.ReadDir(".")
	if err != nil {
		return fmt.Errorf("listing output directory: %w", err)
	}

	header := []byte(GeneratedFileHeader)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if _, wanted := keep[name]; wanted {
			continue
		}

		generated, inspectErr := fileHasGeneratedHeader(dir, name, header)
		if inspectErr != nil {
			return fmt.Errorf("inspecting candidate stale file %q: %w", name, inspectErr)
		}
		if !generated {
			continue
		}

		if removeErr := dir.Remove(name); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return fmt.Errorf("removing stale generated file %q: %w", name, removeErr)
		}
	}

	return nil
}

// fileHasGeneratedHeader reports whether name begins with the piko generated-file header.
//
// Only a header-length prefix is read so an unrelated large sibling is never loaded
// whole. A file shorter than the header, or one already removed, is reported as not
// generated.
//
// Takes dir (GeneratedOutputDir) which is the output directory sandbox.
// Takes name (string) which is the candidate file name.
// Takes header ([]byte) which is the generated-file header to match.
//
// Returns bool which is true when the file starts with the header.
// Returns error when an unexpected open or read failure occurs.
func fileHasGeneratedHeader(dir GeneratedOutputDir, name string, header []byte) (bool, error) {
	handle, openErr := dir.Open(name)
	if openErr != nil {
		if errors.Is(openErr, fs.ErrNotExist) {
			return false, nil
		}
		return false, openErr
	}
	defer func() { _ = handle.Close() }()

	prefix := make([]byte, len(header))
	read, readErr := io.ReadFull(handle, prefix)
	if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
		return false, nil
	}
	if readErr != nil {
		return false, readErr
	}

	return bytes.Equal(prefix[:read], header), nil
}
