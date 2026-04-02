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

package storage_domain

import (
	"cmp"
	"slices"
	"sync"
)

// RepositoryConfig defines metadata and access control for a storage repository.
type RepositoryConfig struct {
	// Name is the repository identifier (e.g., "media-public", "media-private").
	Name string

	// CacheControl is the default Cache-Control header for files in
	// this repository (e.g., "public, max-age=31536000, immutable"
	// for public repos or "private, max-age=3600" for private repos).
	CacheControl string

	// AllowedOrigins for CORS on public repositories (optional).
	AllowedOrigins []string

	// IsPublic indicates whether this repository allows unauthenticated access.
	// Public repositories serve files via permanent URLs without presigning.
	IsPublic bool
}

// RepositoryRegistry manages repository configurations.
type RepositoryRegistry struct {
	// repositories maps repository names to their configurations.
	repositories map[string]*RepositoryConfig

	// mu guards concurrent access to the repositories map.
	mu sync.RWMutex
}

// NewRepositoryRegistry creates a new registry.
//
// Returns *RepositoryRegistry which is an empty registry ready for use.
func NewRepositoryRegistry() *RepositoryRegistry {
	return &RepositoryRegistry{
		repositories: make(map[string]*RepositoryConfig),
	}
}

// Register adds a repository configuration.
//
// Takes config (*RepositoryConfig) which defines the repository metadata.
//
// Safe for concurrent use.
func (r *RepositoryRegistry) Register(config *RepositoryConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.repositories[config.Name] = config
}

// Get retrieves a repository configuration.
//
// Takes name (string) which identifies the repository.
//
// Returns *RepositoryConfig which contains the repository metadata.
// Returns bool which indicates whether the repository was found.
//
// Safe for concurrent use.
func (r *RepositoryRegistry) Get(name string) (*RepositoryConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	config, ok := r.repositories[name]
	return config, ok
}

// ListAll returns all repository configurations sorted by name.
//
// Returns []*RepositoryConfig which contains all registered repositories.
//
// Safe for concurrent use.
func (r *RepositoryRegistry) ListAll() []*RepositoryConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	configs := make([]*RepositoryConfig, 0, len(r.repositories))
	for _, config := range r.repositories {
		configs = append(configs, config)
	}

	slices.SortFunc(configs, func(a, b *RepositoryConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return configs
}

// IsPublic checks if a repository is marked as public.
//
// Takes name (string) which identifies the repository.
//
// Returns bool which is true if the repository is public, false otherwise.
// Defaults to false for unknown repositories.
//
// Safe for concurrent use.
func (r *RepositoryRegistry) IsPublic(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if config, ok := r.repositories[name]; ok {
		return config.IsPublic
	}
	return false
}
