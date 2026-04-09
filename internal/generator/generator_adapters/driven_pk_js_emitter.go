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
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/registry/registry_domain"
)

// PKJSEmitter implements PKJSEmitterPort to transpile and emit JavaScript
// from PK client scripts. It uses JSTranspiler to convert TypeScript to
// JavaScript and stores output in the registry for minification and compression.
type PKJSEmitter struct {
	// transpiler converts TypeScript source code to JavaScript.
	transpiler *generator_domain.JSTranspiler

	// registryService stores and retrieves JS artefacts; nil disables output.
	registryService registry_domain.RegistryService

	// blobStoreID is the identifier for the blob store where compiled JS is stored.
	blobStoreID string
}

var _ generator_domain.PKJSEmitterPort = (*PKJSEmitter)(nil)

// NewPKJSEmitter creates a new PK JavaScript emitter.
//
// When registryService is nil, the emitter will be disabled and returns empty
// artefact IDs.
//
// Takes registryService (registry_domain.RegistryService) which provides access
// to the registry for artefact storage.
//
// Returns *PKJSEmitter which is the configured emitter ready for use.
func NewPKJSEmitter(registryService registry_domain.RegistryService) *PKJSEmitter {
	return &PKJSEmitter{
		transpiler:      generator_domain.NewJSTranspiler(),
		registryService: registryService,
		blobStoreID:     "local_disk_cache",
	}
}

// EmitJS transpiles TypeScript/JavaScript source and stores it in the registry.
// The artefact ID uses the pk-js/ prefix to trigger PK-specific profile handling.
//
// For example:
// pagePath: "pages/checkout"
// -> artefact ID: "pk-js/pages/checkout.js"
// -> profiles: minified (PriorityNeed), gzip, br
// The outputDir and minify parameters are ignored - registry handles storage and
// the capabilities pipeline handles minification/compression.
//
// Takes source (string) which is the TypeScript/JavaScript source code
// to transpile.
// Takes pagePath (string) which identifies the page this script
// belongs to.
//
// Returns string which is the artefact ID for the stored script.
// Returns error when transpilation or registry storage fails.
func (e *PKJSEmitter) EmitJS(
	ctx context.Context,
	source string,
	pagePath string,
	moduleName string,
	_ string,
	_ bool,
) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", nil
	}

	if e.registryService == nil {
		return "", nil
	}

	cleanPath := strings.TrimSuffix(pagePath, ".pk")

	componentName := strings.TrimPrefix(cleanPath, "partials/")
	if componentName == cleanPath {
		componentName = ""
	}

	transformedSource := generator_domain.TransformPKSource(source, componentName)

	filename := filepath.Base(cleanPath) + ".ts"
	result, err := e.transpiler.Transpile(ctx, transformedSource, generator_domain.TranspileOptions{
		Filename:   filename,
		Minify:     false,
		ModuleName: moduleName,
	})
	if err != nil {
		return "", fmt.Errorf("transpiling PK JS for %s: %w", pagePath, err)
	}

	artefactID := fmt.Sprintf("pk-js/%s.js", cleanPath)

	desiredProfiles := lifecycle_domain.GetProfilesForFile(artefactID, nil)

	blobSourcePath := "pk/" + filepath.Base(cleanPath)

	_, err = e.registryService.UpsertArtefact(
		ctx,
		artefactID,
		blobSourcePath,
		bytes.NewReader([]byte(result.Code)),
		e.blobStoreID,
		desiredProfiles,
	)
	if err != nil {
		return "", fmt.Errorf("storing PK JS artefact %s in registry: %w", artefactID, err)
	}

	return artefactID, nil
}
