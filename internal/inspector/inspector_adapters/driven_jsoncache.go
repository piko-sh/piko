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

package inspector_adapters

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/wdk/safedisk"
)

// JSONCache provides JSON-based file caching for TypeData.
// It implements TypeDataProvider and is simpler than FlatBufferCache but
// slower and uses more disk space.
type JSONCache struct {
	// sandbox provides sandboxed file system operations for cache storage.
	sandbox safedisk.Sandbox
}

var _ inspector_domain.TypeDataProvider = (*JSONCache)(nil)

// NewJSONCache creates a new JSONCache with the given sandbox.
// The sandbox root is the cache directory.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem access
// to the cache directory.
//
// Returns *JSONCache which is ready for use.
func NewJSONCache(sandbox safedisk.Sandbox) *JSONCache {
	return &JSONCache{sandbox: sandbox}
}

// GetTypeData fetches cached type data for the given cache key.
//
// Takes cacheKey (string) which identifies the cached type data to fetch.
//
// Returns *inspector_dto.TypeData which holds the cached type information.
// Returns error when the cache directory or key is empty, the cache file does
// not exist, or the cached data cannot be read.
func (fc *JSONCache) GetTypeData(_ context.Context, cacheKey string) (*inspector_dto.TypeData, error) {
	if fc.sandbox == nil || cacheKey == "" {
		return nil, errors.New("file cache provider requires a sandbox and key")
	}

	fileName := fmt.Sprintf("typedata-%s.json", cacheKey)
	data, err := fc.sandbox.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("cache miss or read error for key %s: %w", cacheKey, err)
	}

	var typeData inspector_dto.TypeData
	if err := inspector_dto.CacheAPI.Unmarshal(data, &typeData); err != nil {
		_ = fc.sandbox.Remove(fileName)
		return nil, fmt.Errorf("failed to unmarshal corrupt cache file for key %s: %w", cacheKey, err)
	}
	return &typeData, nil
}

// SaveTypeData serialises and stores TypeData to the cache with the given key.
// The write is atomic via the sandbox's WriteFileAtomic method.
//
// Takes cacheKey (string) which identifies the cache entry.
// Takes data (*inspector_dto.TypeData) which contains the type data to store.
//
// Returns error when the sandbox or key is empty, or when any file
// operation fails.
func (fc *JSONCache) SaveTypeData(_ context.Context, cacheKey string, data *inspector_dto.TypeData) error {
	if fc.sandbox == nil || cacheKey == "" {
		return errors.New("file cache saver requires a sandbox and key")
	}

	if err := fc.sandbox.MkdirAll(".", defaultDirPerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	encodedData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal type data for caching: %w", err)
	}

	fileName := fmt.Sprintf("typedata-%s.json", cacheKey)
	if err := fc.sandbox.WriteFileAtomic(fileName, encodedData, defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write cache file atomically: %w", err)
	}

	return nil
}

// InvalidateCache removes the cached TypeData for the given key.
//
// Takes cacheKey (string) which identifies the cache entry to remove.
//
// Returns error when the sandbox or key is empty, or when the file
// cannot be removed.
func (fc *JSONCache) InvalidateCache(_ context.Context, cacheKey string) error {
	if fc.sandbox == nil || cacheKey == "" {
		return errors.New("file cache invalidator requires a sandbox and key")
	}

	fileName := fmt.Sprintf("typedata-%s.json", cacheKey)
	err := fc.sandbox.Remove(fileName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove cache file %s: %w", fileName, err)
	}
	return nil
}

// ClearCache removes all cached data from the cache directory.
//
// Returns error when the sandbox is not set or the directory cannot be removed.
func (fc *JSONCache) ClearCache(_ context.Context) error {
	if fc.sandbox == nil {
		return errors.New("file cache requires a sandbox to clear")
	}

	if err := fc.sandbox.RemoveAll("."); err != nil {
		return fmt.Errorf("failed to clear cache directory: %w", err)
	}
	return nil
}
