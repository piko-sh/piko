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

package querier_domain

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"

	"piko.sh/piko/internal/querier/querier_dto"
)

// migrationFilePattern matches filenames in the {version}_{name}.{up|down}.sql
// convention. The version is a numeric prefix, the name is a descriptive
// segment, and the direction is either "up" or "down".
var migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

// readMigrationFilesVersioned reads migration files matching the
// {version}_{name}.{up|down}.sql naming convention from the given directory.
// Returns files sorted by version ascending, then up before down within the
// same version.
//
// Takes ctx (context.Context) for cancellation.
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files.
//
// Returns []querier_dto.MigrationFile which contains the parsed migration
// files with checksums.
// Returns error when the directory cannot be read or filenames are malformed.
func readMigrationFilesVersioned(
	ctx context.Context,
	fileReader FileReaderPort,
	directory string,
) ([]querier_dto.MigrationFile, error) {
	entries, readError := fileReader.ReadDir(ctx, directory)
	if readError != nil {
		return nil, fmt.Errorf("reading migration directory %s: %w", directory, readError)
	}

	var files []querier_dto.MigrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, parseError := strconv.ParseInt(matches[1], 10, 64)
		if parseError != nil {
			return nil, fmt.Errorf("parsing version from %s: %w", entry.Name(), parseError)
		}

		const directionGroupIndex = 3
		name := matches[2]
		direction := querier_dto.MigrationDirectionUp
		if matches[directionGroupIndex] == "down" {
			direction = querier_dto.MigrationDirectionDown
		}

		path := filepath.Join(directory, entry.Name())
		content, fileError := fileReader.ReadFile(ctx, path)
		if fileError != nil {
			return nil, fmt.Errorf("reading migration file %s: %w", path, fileError)
		}

		checksum := computeChecksum(content)

		files = append(files, querier_dto.MigrationFile{
			Version:   version,
			Name:      name,
			Direction: direction,
			Filename:  entry.Name(),
			Content:   content,
			Checksum:  checksum,
		})
	}

	slices.SortFunc(files, func(a, b querier_dto.MigrationFile) int {
		if result := cmp.Compare(a.Version, b.Version); result != 0 {
			return result
		}
		return cmp.Compare(a.Direction, b.Direction)
	})

	return files, nil
}

// computeChecksum returns the SHA-256 hex digest of the
// given content.
//
// Takes content ([]byte) which holds the raw bytes to hash.
//
// Returns string which is the hex-encoded SHA-256 digest.
func computeChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
