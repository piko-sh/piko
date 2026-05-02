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
	"path"
	"regexp"
	"slices"
	"strconv"

	"piko.sh/piko/internal/querier/querier_dto"
)

var (
	// migrationFilePattern matches filenames in the {version}_{name}.{up|down}.sql
	// convention. The version is a numeric prefix, the name is a descriptive segment, and
	// the direction is either "up" or "down".
	migrationFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)
)

// migrationVersionKey identifies a migration file slot by its numeric version and
// direction. Two files mapping to the same key collide and are rejected at read time.
type migrationVersionKey struct {
	// version is the numeric migration version parsed from the filename prefix.
	version int64

	// direction records whether the slot is the up or down migration.
	direction querier_dto.MigrationDirection
}

// parsedMigrationFilename holds the fields extracted from a
// {version}_{name}.{up|down}.sql filename.
type parsedMigrationFilename struct {
	// name is the descriptive segment between the version prefix and the direction suffix.
	name string

	// version is the numeric migration version parsed from the filename prefix.
	version int64

	// direction records whether the file is the up or down migration.
	direction querier_dto.MigrationDirection
}

// readMigrationFilesVersioned reads migration files matching the
// {version}_{name}.{up|down}.sql naming convention from the given directory.
// Returns files sorted by version ascending, then up before down within the same version.
//
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files.
//
// Returns []querier_dto.MigrationFile which contains the parsed migration files with
// checksums.
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
	seenVersions := make(map[migrationVersionKey]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		parsed, matched, parseError := parseMigrationFilename(entry.Name())
		if parseError != nil {
			return nil, parseError
		}
		if !matched {
			continue
		}

		key := migrationVersionKey{version: parsed.version, direction: parsed.direction}
		if existing, duplicate := seenVersions[key]; duplicate {
			return nil, &DuplicateMigrationVersionError{
				FirstFilename:  existing,
				SecondFilename: entry.Name(),
				Version:        parsed.version,
				Direction:      parsed.direction,
			}
		}
		seenVersions[key] = entry.Name()

		filePath := path.Join(directory, entry.Name())
		content, fileError := fileReader.ReadFile(ctx, filePath)
		if fileError != nil {
			return nil, fmt.Errorf("reading migration file %s: %w", filePath, fileError)
		}

		files = append(files, querier_dto.MigrationFile{
			Version:   parsed.version,
			Name:      parsed.name,
			Direction: parsed.direction,
			Filename:  entry.Name(),
			Content:   content,
			Checksum:  computeChecksum(content),
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

// parseMigrationFilename extracts the version, descriptive name, and direction from a
// migration filename following the {version}_{name}.{up|down}.sql convention.
//
// Takes filename (string) which is the directory entry name.
//
// Returns parsedMigrationFilename which holds the extracted fields (zero value when no
// match).
// Returns bool which is true when the filename matched the convention.
// Returns error when the numeric version prefix overflows int64.
func parseMigrationFilename(filename string) (parsedMigrationFilename, bool, error) {
	matches := migrationFilePattern.FindStringSubmatch(filename)
	if matches == nil {
		return parsedMigrationFilename{}, false, nil
	}

	version, parseError := strconv.ParseInt(matches[1], 10, 64)
	if parseError != nil {
		return parsedMigrationFilename{}, false, fmt.Errorf("parsing version from %s: %w", filename, parseError)
	}

	const directionGroupIndex = 3
	direction := querier_dto.MigrationDirectionUp
	if matches[directionGroupIndex] == "down" {
		direction = querier_dto.MigrationDirectionDown
	}

	return parsedMigrationFilename{name: matches[2], version: version, direction: direction}, true, nil
}

// computeChecksum returns the SHA-256 hex digest of the given content.
//
// Takes content ([]byte) which holds the raw bytes to hash.
//
// Returns string which is the hex-encoded SHA-256 digest.
func computeChecksum(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
