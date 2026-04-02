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
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// seedFilePattern matches seed files named like 001_demo_data.sql.
var seedFilePattern = regexp.MustCompile(`^(\d+)_(.+)\.sql$`)

// readSeedFiles reads and parses seed SQL files matching
// {version}_{name}.sql from the given directory, returning
// them sorted by version ascending.
//
// Takes ctx (context.Context) for cancellation.
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the seed files directory.
//
// Returns []querier_dto.SeedFile which holds the sorted seed files.
// Returns error when the directory cannot be read or a file cannot be loaded.
func readSeedFiles(
	ctx context.Context,
	fileReader FileReaderPort,
	directory string,
) ([]querier_dto.SeedFile, error) {
	entries, err := fileReader.ReadDir(ctx, directory)
	if err != nil {
		return nil, fmt.Errorf("reading seed directory %s: %w", directory, err)
	}

	var files []querier_dto.SeedFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		matches := seedFilePattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, parseErr := strconv.ParseInt(matches[1], 10, 64)
		if parseErr != nil {
			continue
		}

		path := filepath.Join(directory, entry.Name())
		content, readErr := fileReader.ReadFile(ctx, path)
		if readErr != nil {
			return nil, fmt.Errorf("reading seed file %s: %w", path, readErr)
		}

		files = append(files, querier_dto.SeedFile{
			Version:  version,
			Name:     matches[2],
			Filename: entry.Name(),
			Content:  content,
			Checksum: computeChecksum(content),
		})
	}

	slices.SortFunc(files, func(a, b querier_dto.SeedFile) int {
		return cmp.Compare(a.Version, b.Version)
	})

	return files, nil
}
