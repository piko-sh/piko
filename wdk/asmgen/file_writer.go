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
	"os"
	"path/filepath"
)

// directoryPermission is the file mode used when creating parent directories.
const directoryPermission = 0o750

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

// WriteFile writes data to the given path, creating parent
// directories as needed. It uses mode 0o600 for files and 0o750 for
// directories.
//
// Takes path (string) which is the output file path.
// Takes data ([]byte) which is the file content.
//
// Returns error when directory creation or file writing fails.
func (*DiskWriter) WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, directoryPermission); err != nil {
		return err
	}
	return os.WriteFile(path, data, filePermission)
}
