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

package annotator_dto

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// ProjectAnnotationResult holds the results of a full project build. It
// contains the annotation results for each component and the overall project
// structure.
type ProjectAnnotationResult struct {
	// FinalGeneratedArtefacts holds the fully-emitted code for each component.
	//
	// This field is populated by the coordinator after code emission and contains
	// the complete, executable Go code (including BuildAST, init(), etc.) that is
	// either compiled (in dev/prod modes) or interpreted (in dev-i mode). This
	// field is critical for interpreted mode to receive the correct code.
	//
	// Type: any (actual type is []*generator_dto.GeneratedArtefact).
	// Uses any to avoid import cycles between annotator and generator.
	FinalGeneratedArtefacts any

	// ComponentResults maps each component's stable hashed name to its full
	// annotation result. This is the main output, containing results for both
	// pages and partials.
	ComponentResults map[string]*AnnotationResult

	// AllSourceContents maps each source file path to its content. This data is
	// used to show detailed error messages for the whole project build.
	AllSourceContents map[string][]byte

	// VirtualModule holds the full Go structure of the project. The generator uses
	// this to find all components, their package paths, and output file locations.
	VirtualModule *VirtualModule

	// AllDiagnostics holds all errors and warnings found across the entire
	// project during all compilation stages.
	AllDiagnostics []*ast_domain.Diagnostic

	// FinalAssetManifest holds the complete list of static asset dependencies
	// found across the project, with duplicates removed.
	FinalAssetManifest []*FinalAssetDependency

	// AnnotatedComponentCount tracks how many components were actually
	// processed during Phase 2 annotation. For scoped rebuilds this will be
	// less than the total number of components in the project.
	AnnotatedComponentCount int

	// GeneratedArtefactCount tracks how many code artefacts were generated.
	// For scoped rebuilds this matches the number of targeted components.
	GeneratedArtefactCount int
}
