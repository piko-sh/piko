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

// Package fixturego supplies ordinary, parseable Go files for generated apps and their
// hosts.
package fixturego

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"

	"piko.sh/piko/internal/gosource"
)

var (
	// sources embeds the Go fixture assets and this materialiser.
	//
	//go:embed testdata/gosource sources.go
	sources embed.FS
)

// Fixed renders one asset at package initialisation.
//
// Its callers keep the resulting immutable string and hand independent byte slices to
// writers when necessary. Use Render wherever an error can be returned.
//
// Takes path (string) which is the asset path under testdata/gosource.
//
// Returns string which is the formatted asset source.
//
// Panics when the asset is missing or does not parse, since that is a defect in the
// repository's generator rather than in fuzz input.
func Fixed(path string) string {
	result, err := Render(path, nil)
	if err != nil {
		panic(err)
	}
	return result
}

// Render parses a fresh copy of an asset for every edit.
//
// A missing asset or a stale edit describes an error in the repository's generator,
// independent of fuzz input.
//
// Takes path (string) which is the asset path under testdata/gosource.
// Takes edit (func(*token.FileSet, *ast.File) error) which rewrites the parsed asset, or
// is nil to render it unchanged.
//
// Returns string which is the formatted, edited asset source.
// Returns error which reports a missing asset, a parse failure or a failed edit.
func Render(path string, edit func(*token.FileSet, *ast.File) error) (string, error) {
	source, err := sources.ReadFile("testdata/gosource/" + path)
	if err != nil {
		return "", fmt.Errorf("fixture %s: %w", path, err)
	}
	result, err := gosource.Render(path, source, edit)
	if err != nil {
		return "", fmt.Errorf("fixture %s: %w", path, err)
	}
	return string(result), nil
}

// SourceSHA256 digests each named source and this materialiser.
//
// Returns string which is the hex SHA-256 of every embedded source.
//
// Panics when the embedded file system cannot be walked, which cannot happen for a
// well-formed binary.
func SourceSHA256() string {
	digest, err := sourceSHA256(sources)
	if err != nil {
		panic(err)
	}
	return digest
}

// sourceSHA256 digests every regular file in a file system.
//
// Takes files (fs.FS) which holds the sources to digest.
//
// Returns string which is the hex SHA-256 digest.
// Returns error which reports a walk or read failure.
func sourceSHA256(files fs.FS) (string, error) {
	hasher := sha256.New()
	err := fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(files, path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		fmt.Fprintf(hasher, "%d:%s:%d:", len(path), path, len(data))
		_, _ = hasher.Write(data)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("digest fixture sources: %w", err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
