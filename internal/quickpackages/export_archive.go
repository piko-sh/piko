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

package quickpackages

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	// archiveSignature opens every archive file the Go compiler writes.
	archiveSignature = "!<arch>\n"

	// archiveHeaderSize is the length of an ar member header.
	archiveHeaderSize = 60

	// packageDefinitionName is the archive member holding the package's export data. The
	// compiler always writes it first.
	packageDefinitionName = "__.PKGDEF"

	// exportDataStart marks the start of the binary export data section.
	exportDataStart = "$$B\n"

	// exportDataEnd marks the end of the export data section. It is not part of the data.
	exportDataEnd = "\n$$\n"
)

var (
	// errNotGoArchive reports a file that is not a compiler-written archive.
	errNotGoArchive = errors.New("not a Go archive file")

	// errNoPackageDefinition reports an archive whose first member is not __.PKGDEF.
	errNoPackageDefinition = errors.New("archive does not start with a package definition")

	// errNoBinaryExportData reports a package definition without a binary export section.
	errNoBinaryExportData = errors.New("package definition has no binary export data")
)

// exportDataReader positions r at the export data inside an archive written by the Go
// compiler, as named by `go list -export`, and returns a reader limited to that data.
//
// The archive's first member, __.PKGDEF, holds header lines ("go object ...", build ID)
// followed by "$$B\n", the export data, and an "\n$$\n" end marker. The returned reader
// yields exactly the export data, in the form gcexportdata.Read expects.
//
// Takes r (*bufio.Reader) which must be positioned at the start of the archive.
//
// Returns io.Reader which reads only the export data.
// Returns error when the file is not a Go archive or holds no binary export data.
func exportDataReader(r *bufio.Reader) (io.Reader, error) {
	signature := make([]byte, len(archiveSignature))
	if _, err := io.ReadFull(r, signature); err != nil || string(signature) != archiveSignature {
		return nil, errNotGoArchive
	}

	header := make([]byte, archiveHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, errNoPackageDefinition
	}
	if !strings.HasPrefix(strings.TrimSpace(string(header[0:16])), packageDefinitionName) {
		return nil, errNoPackageDefinition
	}
	remaining, err := strconv.Atoi(strings.TrimSpace(string(header[48:58])))
	if err != nil || remaining <= 0 {
		return nil, errNoPackageDefinition
	}

	for {
		line, err := r.ReadSlice('\n')
		if err != nil {
			return nil, fmt.Errorf("reading package definition headers: %w", err)
		}
		remaining -= len(line)
		if string(line) == exportDataStart {
			break
		}
		if strings.HasPrefix(string(line), "$$") {
			return nil, errNoBinaryExportData
		}
	}

	remaining -= len(exportDataEnd)
	if remaining < 0 {
		return nil, errNoBinaryExportData
	}
	return io.LimitReader(r, int64(remaining)), nil
}
