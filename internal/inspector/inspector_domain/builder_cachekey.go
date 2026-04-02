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

package inspector_domain

// This file specifically holds the implementation for generating a stable cache key
// based on project dependencies, environment, and source code content.

import (
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/cespare/xxhash/v2"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/wdk/safedisk"
)

// generateCacheKey creates a stable hash from project dependencies, source
// content, and script block hashes. This is the core logic used by the default
// builderCacheKeyGenerator implementation.
//
// Uses xxhash for speed and to make clear this is not for cryptographic use.
//
// The sourceContents map holds stub Go files from the virtualiser. These files
// do not change when script blocks change, which would cause cache key clashes.
// To fix this, the function also hashes the scriptHashes map, which holds
// hashes of actual script block content from .pk files. This means script
// changes invalidate the cache.
//
// Takes config (inspector_dto.Config) which provides the base directory path.
// Takes sourceContents (map[string][]byte) which holds virtualised Go files.
// Takes scriptHashes (map[string]string) which maps script paths to their
// content hashes.
//
// Returns string which is the hex-encoded xxhash of all inputs.
// Returns error when hashing any part fails.
func generateCacheKey(config inspector_dto.Config, sourceContents map[string][]byte, scriptHashes map[string]string, factory safedisk.Factory) (string, error) {
	hasher := xxhash.New()

	if err := hashDependencyFiles(hasher, config.BaseDir, factory); err != nil {
		return "", fmt.Errorf("hashing dependency files: %w", err)
	}
	if err := hashEnvironmentVariables(hasher); err != nil {
		return "", fmt.Errorf("hashing environment variables: %w", err)
	}
	if err := hashSourceContents(hasher, sourceContents); err != nil {
		return "", fmt.Errorf("hashing source contents: %w", err)
	}
	if err := hashScriptHashes(hasher, scriptHashes); err != nil {
		return "", fmt.Errorf("hashing script hashes: %w", err)
	}
	if err := hashBuildFlags(hasher, config.BuildFlags); err != nil {
		return "", fmt.Errorf("hashing build flags: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// hashDependencyFiles adds the contents of go.mod and go.sum to the hash.
//
// Takes hasher (hash.Hash) which collects the hash of the file contents.
// Takes baseDir (string) which is the folder that contains the files.
//
// Returns error when a dependency file cannot be hashed.
func hashDependencyFiles(hasher hash.Hash, baseDir string, factory safedisk.Factory) error {
	for _, filename := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(baseDir, filename)
		if err := hashFile(hasher, path, factory); err != nil {
			return fmt.Errorf("could not hash dependency file %s: %w", filename, err)
		}
	}
	return nil
}

// hashEnvironmentVariables writes Go build environment variables to the given
// hasher for cache key generation.
//
// Takes hasher (hash.Hash) which receives the variable names and values.
//
// Returns error when writing to the hasher fails.
func hashEnvironmentVariables(hasher hash.Hash) error {
	goBuildEnvVars := []string{"CGO_ENABLED", "GOARCH", "GOEXPERIMENT", "GOFLAGS", "GOOS", "GOPROXY"}
	for _, key := range goBuildEnvVars {
		if _, err := hasher.Write([]byte(key)); err != nil {
			return fmt.Errorf("writing env var key %q to hasher: %w", key, err)
		}
		if _, err := hasher.Write([]byte(os.Getenv(key))); err != nil {
			return fmt.Errorf("writing env var value for %q to hasher: %w", key, err)
		}
	}
	return nil
}

// hashSourceContents hashes all source file contents in a fixed order.
//
// Takes hasher (hash.Hash) which receives the path and content bytes.
// Takes sourceContents (map[string][]byte) which maps file paths to their
// contents.
//
// Returns error when writing to the hasher fails.
func hashSourceContents(hasher hash.Hash, sourceContents map[string][]byte) error {
	for _, path := range slices.Sorted(maps.Keys(sourceContents)) {
		if _, err := hasher.Write([]byte(path)); err != nil {
			return fmt.Errorf("writing source path %q to hasher: %w", path, err)
		}
		if _, err := hasher.Write(sourceContents[path]); err != nil {
			return fmt.Errorf("writing source content for %q to hasher: %w", path, err)
		}
	}
	return nil
}

// hashScriptHashes adds script block hashes to the cache key.
// This means changes to scripts alone will make the cache invalid.
//
// When scriptHashes is nil, returns at once without error.
//
// Takes hasher (hash.Hash) which receives the hash data.
// Takes scriptHashes (map[string]string) which maps paths to their hashes.
//
// Returns error when writing to the hasher fails.
func hashScriptHashes(hasher hash.Hash, scriptHashes map[string]string) error {
	if scriptHashes == nil {
		return nil
	}
	for _, path := range slices.Sorted(maps.Keys(scriptHashes)) {
		if _, err := hasher.Write([]byte(path)); err != nil {
			return fmt.Errorf("writing script path %q to hasher: %w", path, err)
		}
		if _, err := hasher.Write([]byte(scriptHashes[path])); err != nil {
			return fmt.Errorf("writing script hash for %q to hasher: %w", path, err)
		}
	}
	return nil
}

// hashBuildFlags adds sorted build flags to the cache key. This means
// changes to build flags (such as adding or removing analysis tags) invalidate
// the cache.
//
// Takes hasher (hash.Hash) which receives the flag data.
// Takes flags ([]string) which contains the build flags to hash.
//
// Returns error when writing to the hasher fails.
func hashBuildFlags(hasher hash.Hash, flags []string) error {
	if len(flags) == 0 {
		return nil
	}
	sorted := make([]string, len(flags))
	copy(sorted, flags)
	slices.Sort(sorted)
	for _, flag := range sorted {
		if _, err := hasher.Write([]byte(flag)); err != nil {
			return fmt.Errorf("writing build flag %q to hasher: %w", flag, err)
		}
	}
	return nil
}

// hashFile reads a file and writes its contents to the given hash.
//
// When the file does not exist, returns nil without error. The path argument
// comes from trusted internal sources (go.mod, go.sum paths), not from user
// input.
//
// Takes h (hash.Hash) which receives the file contents for hashing.
// Takes path (string) which specifies the file path to read.
//
// Returns error when the file exists but cannot be read or copied.
func hashFile(h hash.Hash, path string, factory safedisk.Factory) error {
	parentDir := filepath.Dir(path)
	fileName := filepath.Base(path)

	var sandbox safedisk.Sandbox
	var sandboxErr error
	if factory != nil {
		sandbox, sandboxErr = factory.Create("hash file", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, sandboxErr = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if sandboxErr != nil {
		return nil
	}
	defer func() { _ = sandbox.Close() }()

	f, err := sandbox.Open(fileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("opening file %q for hashing: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("copying file %q to hasher: %w", path, err)
	}
	return nil
}
