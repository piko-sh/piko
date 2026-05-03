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
	"path/filepath"

	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/wdk/safedisk"
)

// dirPermission is the default permission for directories created by generator
// adapters. Uses 0750 to restrict access to owner and group only.
const dirPermission = 0750

// DrivenCollectionEmitter implements CollectionEmitterPort.
//
// This adapter creates static collection files. It turns collection items into
// FlatBuffer binary format, writes the binary to dist/collections/{name}/, and
// creates a Go wrapper file with an embed directive. It uses ports for both
// serialisation and file writing to keep the generator separate from the
// FlatBuffers details. All file operations use a sandbox to prevent path
// traversal attacks.
type DrivenCollectionEmitter struct {
	// encoder converts collection items to binary format.
	encoder collection_domain.CollectionEncoderPort

	// fsWriter writes generated files to the filesystem.
	fsWriter generator_domain.FSWriterPort

	// sandbox provides safe file path handling and file system operations.
	sandbox safedisk.Sandbox

	// moduleName is the Go module name from go.mod (e.g. a GitHub-hosted module
	// path such as "example.com/user/project").
	moduleName string
}

// NewDrivenCollectionEmitter creates a new collection emitter instance.
//
// Takes encoder (CollectionEncoderPort) which provides encoding
// from the collection hexagon.
// Takes fsWriter (FSWriterPort) which handles filesystem write operations.
// Takes sandbox (Sandbox) which provides sandboxed filesystem access for
// directory operations.
// Takes moduleName (string) which specifies the Go module name for import
// paths.
//
// Returns *DrivenCollectionEmitter which is the configured emitter ready for
// use.
func NewDrivenCollectionEmitter(
	encoder collection_domain.CollectionEncoderPort,
	fsWriter generator_domain.FSWriterPort,
	sandbox safedisk.Sandbox,
	moduleName string,
) *DrivenCollectionEmitter {
	return &DrivenCollectionEmitter{
		encoder:    encoder,
		fsWriter:   fsWriter,
		sandbox:    sandbox,
		moduleName: moduleName,
	}
}

// EmitCollection generates the binary and Go wrapper files for a static
// collection.
//
// Workflow:
//  1. Create output directory (dist/collections/{collectionName}/)
//  2. Serialise items to binary using serialiser port
//  3. Write binary to data.bin
//  4. Generate Go wrapper file with //go:embed and init()
//  5. Write wrapper to generated.go
//
// Takes collectionName (string) which identifies the collection to generate.
// Takes items ([]collection_dto.ContentItem) which contains the data to embed.
// Takes outputDir (string) which specifies the base output directory.
//
// Returns string which is the full Go package path for the generated
// collection.
// Returns error when any step in the generation workflow fails.
func (e *DrivenCollectionEmitter) EmitCollection(
	ctx context.Context,
	collectionName string,
	items []collection_dto.ContentItem,
	outputDir string,
) (string, error) {
	relOutputDir := e.sandbox.RelPath(outputDir)
	collectionDir := filepath.Join(relOutputDir, "collections", collectionName)
	dataFilePath := filepath.Join(collectionDir, "data.bin")
	goFilePath := filepath.Join(collectionDir, "generated.go")

	if err := e.sandbox.MkdirAll(collectionDir, dirPermission); err != nil {
		return "", fmt.Errorf("failed to create collection directory %s: %w", collectionDir, err)
	}

	binaryData, err := e.encoder.EncodeCollection(items)
	if err != nil {
		return "", fmt.Errorf("failed to encode collection %q: %w", collectionName, err)
	}

	if err := e.fsWriter.WriteFile(ctx, dataFilePath, binaryData); err != nil {
		return "", fmt.Errorf("failed to write binary data for collection %q: %w", collectionName, err)
	}

	goCode := e.generateGoWrapper(collectionName)

	if err := e.fsWriter.WriteFile(ctx, goFilePath, []byte(goCode)); err != nil {
		return "", fmt.Errorf("failed to write Go wrapper for collection %q: %w", collectionName, err)
	}

	packagePath := filepath.Join(e.moduleName, "dist", "collections", collectionName)

	return packagePath, nil
}

// generateGoWrapper creates the Go source code for the collection wrapper.
//
// Takes collectionName (string) which specifies the package name and registry
// key for the generated code.
//
// Returns string which contains the complete Go source file.
//
// The generated file:
//   - Uses //go:embed to embed data.bin into the binary
//   - Registers the blob with the runtime registry in init()
//   - Is minimal and contains no business logic
func (*DrivenCollectionEmitter) generateGoWrapper(collectionName string) string {
	return generator_dto.AnalysisBuildConstraint + fmt.Sprintf(`// Code generated by Piko. DO NOT EDIT.
// This file embeds the binary collection data and registers it with the
// runtime.

package %s

import (
	"context"
	_ "embed"
	pikoruntime "piko.sh/piko/wdk/runtime"
)

//go:embed data.bin
var collectionBlob []byte

func init() {
	pikoruntime.RegisterStaticCollectionBlob(context.Background(), %q, collectionBlob)
}
`, collectionName, collectionName)
}
