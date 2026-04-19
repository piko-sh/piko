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

package asmgen

import (
	"fmt"
	"path/filepath"

	"piko.sh/piko/wdk/safedisk"
)

// filePermission is the file mode used when writing output files.
const filePermission = 0o600

// DiskWriter implements FileSystemWriterPort by writing files to the local
// filesystem.
type DiskWriter struct{}

var _ FileSystemWriterPort = (*DiskWriter)(nil)

// NewDiskWriter creates a new filesystem writer.
//
// Returns *DiskWriter ready for use.
func NewDiskWriter() *DiskWriter {
	return &DiskWriter{}
}

// WriteFile writes data to the given path atomically.
//
// It constructs a one-shot read-write sandbox at the file's parent directory
// so all writes go through safedisk path-traversal protection. The sandbox
// creates the parent directory if it does not yet exist. The data is written
// via WriteFileAtomic so an interrupted build cannot leave a half-written
// file on disk.
//
// Takes path (string) which is the output file path.
// Takes data ([]byte) which is the file content.
//
// Returns error when sandbox creation or file writing fails.
func (*DiskWriter) WriteFile(path string, data []byte) error {
	directory := filepath.Dir(path)
	sandbox, err := safedisk.NewSandbox(directory, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("preparing directory %q: %w", directory, err)
	}
	defer func() { _ = sandbox.Close() }()
	return sandbox.WriteFileAtomic(filepath.Base(path), data, filePermission)
}
