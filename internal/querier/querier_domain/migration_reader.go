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
	"bytes"
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// migrationFile holds a migration file's metadata and content.
type migrationFile struct {
	// filename holds the base name of the migration file.
	filename string

	// content holds the raw bytes of the migration file.
	content []byte

	// index holds the zero-based position of this file in the sorted list.
	index int
}

// queryBlock holds a single query extracted from a multi-query file, together
// with its starting line offset for source mapping.
type queryBlock struct {
	// sql holds the SQL text of this query block.
	sql string

	// startLine holds the one-based line number where this block begins in
	// the original file.
	startLine int
}

// readMigrationFiles reads and sorts SQL files from the given directory.
//
// Down migration files (.down.sql) are skipped. Files are sorted
// lexicographically by filename.
//
// Takes ctx (context.Context) for cancellation.
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files directory.
//
// Returns []migrationFile which holds the sorted migration files with their
// content.
// Returns error when the directory cannot be read or a file cannot be loaded.
func readMigrationFiles(
	ctx context.Context,
	fileReader FileReaderPort,
	directory string,
) ([]migrationFile, error) {
	entries, err := fileReader.ReadDir(ctx, directory)
	if err != nil {
		return nil, fmt.Errorf("reading migration directory %s: %w", directory, err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".down.sql") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		content, readError := fileReader.ReadFile(ctx, path)
		if readError != nil {
			return nil, fmt.Errorf("reading migration file %s: %w", path, readError)
		}
		files = append(files, migrationFile{
			filename: entry.Name(),
			content:  content,
		})
	}

	slices.SortFunc(files, func(a, b migrationFile) int {
		return cmp.Compare(a.filename, b.filename)
	})

	for i := range files {
		files[i].index = i
	}

	return files, nil
}

// warnNonConformingMigrationFiles checks a directory for .sql files that do
// not follow the .up.sql or .down.sql naming convention and returns warning
// diagnostics for each.
//
// Takes fileReader (FileReaderPort) which provides filesystem access.
// Takes directory (string) which is the path to the migration files directory.
//
// Returns []querier_dto.SourceError which holds warning diagnostics for each
// non-conforming file, or nil if all files conform.
func warnNonConformingMigrationFiles(
	ctx context.Context,
	fileReader FileReaderPort,
	directory string,
) []querier_dto.SourceError {
	entries, err := fileReader.ReadDir(ctx, directory)
	if err != nil {
		return nil
	}

	var diagnostics []querier_dto.SourceError
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		if strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".down.sql") {
			continue
		}
		diagnostics = append(diagnostics, querier_dto.SourceError{
			Filename: name,
			Line:     1,
			Column:   1,
			Message:  fmt.Sprintf("%s is ignored; migration files must end in .up.sql or .down.sql", name),
			Severity: querier_dto.SeverityWarning,
		})
	}

	return diagnostics
}

// downMigrationMarkers are the recognised markers that indicate the start of
// a down-migration section. Everything after a marker is stripped.
var downMigrationMarkers = [][]byte{
	[]byte("-- +migrate Down"),
	[]byte("-- +goose Down"),
	[]byte("-- migrate:down"),
}

// stripDownMigration removes everything after a recognised down-migration
// marker. If no marker is found, the content is returned unchanged.
//
// Takes content ([]byte) which holds the raw migration file bytes.
//
// Returns []byte which holds the content with the down-migration section
// stripped.
func stripDownMigration(content []byte) []byte {
	for _, marker := range downMigrationMarkers {
		if before, _, found := bytes.Cut(content, marker); found {
			return before
		}
	}
	return content
}

// queryNamePrefixForStyle returns the directive prefix that starts a new
// query block, derived from the given comment style.
//
// Takes style (querier_dto.CommentStyle) which specifies the SQL comment
// syntax to use.
//
// Returns string which holds the prefix string used to detect query name
// directives.
func queryNamePrefixForStyle(style querier_dto.CommentStyle) string {
	return style.LinePrefix + " piko.name:"
}

// splitQueryFile splits a SQL file containing multiple queries into individual
// query blocks.
//
// Queries are separated by the piko.name: directive on each subsequent query.
// Each block includes its starting line offset for source mapping. The comment
// style determines the directive prefix.
//
// Takes content ([]byte) which holds the raw SQL file bytes.
// Takes style (querier_dto.CommentStyle) which specifies the SQL comment
// syntax to use for directive detection.
//
// Returns []queryBlock which holds the individual query blocks with their
// line offsets.
func splitQueryFile(content []byte, style querier_dto.CommentStyle) []queryBlock {
	prefix := queryNamePrefixForStyle(style)
	lines := strings.Split(string(content), "\n")
	var blocks []queryBlock
	currentLines := make([]string, 0, len(lines))
	currentStart := 1

	for lineIndex, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) && len(currentLines) > 0 {
			sql := strings.TrimSpace(strings.Join(currentLines, "\n"))
			if sql != "" {
				blocks = append(blocks, queryBlock{
					sql:       sql,
					startLine: currentStart,
				})
			}
			currentLines = nil
			currentStart = lineIndex + 1
		}
		currentLines = append(currentLines, line)
	}

	if len(currentLines) > 0 {
		sql := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if sql != "" {
			blocks = append(blocks, queryBlock{
				sql:       sql,
				startLine: currentStart,
			})
		}
	}

	return blocks
}
