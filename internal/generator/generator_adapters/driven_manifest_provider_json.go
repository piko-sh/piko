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

package generator_adapters

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

// JSONManifestProvider implements ManifestProviderPort to load a project
// manifest from a JSON file on disk.
type JSONManifestProvider struct {
	// sandbox provides file system access for reading the manifest file.
	sandbox safedisk.Sandbox

	// factory creates sandboxes with validated paths. When set and sandbox is
	// nil, the factory is used before falling back to NewNoOpSandbox.
	factory safedisk.Factory

	// manifestFileName is the path to the manifest file within the sandbox.
	manifestFileName string
}

var _ generator_domain.ManifestProviderPort = (*JSONManifestProvider)(nil)

// JSONManifestProviderOption sets up a JSONManifestProvider during creation.
type JSONManifestProviderOption func(*JSONManifestProvider)

// NewJSONManifestProvider creates a provider that reads from a JSON manifest
// file at the given path.
//
// Takes manifestPath (string) which is the path to the JSON manifest file.
// Takes opts (...JSONManifestProviderOption) which provides optional
// configuration such as WithJSONManifestSandbox for testing.
//
// Returns *JSONManifestProvider which is ready to read from the given path.
func NewJSONManifestProvider(manifestPath string, opts ...JSONManifestProviderOption) *JSONManifestProvider {
	p := &JSONManifestProvider{
		sandbox:          nil,
		manifestFileName: filepath.Base(manifestPath),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.sandbox == nil {
		sandbox, err := createManifestSandbox(manifestPath, p.factory, "JSON manifest provider")
		if err != nil {
			return p
		}
		p.sandbox = sandbox
	}

	return p
}

// Load reads the manifest.json file from disk, parses it, and returns the
// Manifest DTO.
//
// Returns *generator_dto.Manifest which contains the parsed manifest data.
// Returns error when the file path is empty, sandbox is unavailable, the file
// cannot be read, or the JSON is malformed.
func (p *JSONManifestProvider) Load(_ context.Context) (*generator_dto.Manifest, error) {
	if p.manifestFileName == "" {
		return nil, errors.New("JSON manifest provider requires a valid file path")
	}

	if p.sandbox == nil {
		return nil, errors.New("JSON manifest provider sandbox not available")
	}

	data, err := p.sandbox.ReadFile(p.manifestFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("manifest file not found at %s: %w", p.manifestFileName, err)
		}
		return nil, fmt.Errorf("failed to read manifest file %s: %w", p.manifestFileName, err)
	}

	var manifest generator_dto.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse corrupt manifest file %s: %w", p.manifestFileName, err)
	}

	return &manifest, nil
}

// WithJSONManifestFactory sets the sandbox factory for the JSON manifest
// provider. When no sandbox is injected, the factory is tried before falling
// back to NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns JSONManifestProviderOption which configures the provider with the
// factory.
func WithJSONManifestFactory(factory safedisk.Factory) JSONManifestProviderOption {
	return func(p *JSONManifestProvider) {
		p.factory = factory
	}
}

// createManifestSandbox creates a read-only sandbox for the directory
// containing the manifest file. When factory is non-nil it is used to create
// the sandbox; otherwise a no-op sandbox is created as a fallback.
//
// Takes manifestPath (string) which is the path to the manifest file. The
// parent directory is used as the sandbox root.
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths. May be nil.
// Takes description (string) which identifies the sandbox in diagnostics.
//
// Returns safedisk.Sandbox which provides read access to the manifest
// directory.
// Returns error when the sandbox cannot be created.
func createManifestSandbox(manifestPath string, factory safedisk.Factory, description string) (safedisk.Sandbox, error) {
	manifestDir := filepath.Dir(manifestPath)
	if factory != nil {
		return factory.Create(description, manifestDir, safedisk.ModeReadOnly)
	}
	return safedisk.NewNoOpSandbox(manifestDir, safedisk.ModeReadOnly)
}

// WithJSONManifestSandbox sets a custom sandbox for the JSON manifest provider.
// Inject a mock sandbox to test file system operations.
//
// If not set, a real sandbox is created using safedisk.NewNoOpSandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides file system access for reading
// the manifest file.
//
// Returns JSONManifestProviderOption which sets up the provider with the
// given sandbox.
func WithJSONManifestSandbox(sandbox safedisk.Sandbox) JSONManifestProviderOption {
	return func(p *JSONManifestProvider) {
		p.sandbox = sandbox
	}
}
