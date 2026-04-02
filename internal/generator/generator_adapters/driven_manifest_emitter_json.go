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
	"fmt"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

// JSONManifestEmitter implements ManifestEmitterPort to write a JSON manifest.
// It uses atomic writes to prevent file corruption and sandboxes all file
// operations to prevent path traversal attacks.
type JSONManifestEmitter struct {
	// sandbox provides safe file system access for writing manifest files.
	sandbox safedisk.Sandbox
}

var _ generator_domain.ManifestEmitterPort = (*JSONManifestEmitter)(nil)

// NewJSONManifestEmitter creates an emitter that writes manifests in JSON
// format.
//
// Takes sandbox (safedisk.Sandbox) which is the output folder for the manifest
// file.
//
// Returns *JSONManifestEmitter which is ready to write manifests.
func NewJSONManifestEmitter(sandbox safedisk.Sandbox) *JSONManifestEmitter {
	return &JSONManifestEmitter{sandbox: sandbox}
}

// EmitCode generates the final manifest.json file by serialising the manifest
// to JSON and writing it atomically to the specified output path.
//
// Takes manifest (*generator_dto.Manifest) which contains the data to write.
// Takes outputPath (string) which specifies where to write the JSON file.
//
// Returns error when JSON serialisation fails or the file cannot be written.
func (e *JSONManifestEmitter) EmitCode(
	ctx context.Context,
	manifest *generator_dto.Manifest,
	outputPath string,
) error {
	bytes, err := json.ConfigStd.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest to JSON: %w", err)
	}

	relPath := e.sandbox.RelPath(outputPath)

	if err := generator_domain.AtomicWriteFile(ctx, e.sandbox, relPath, bytes, generator_domain.FilePermission); err != nil {
		return fmt.Errorf("failed to write manifest file atomically: %w", err)
	}

	return nil
}
