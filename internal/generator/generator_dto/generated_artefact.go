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

package generator_dto

import (
	"piko.sh/piko/internal/annotator/annotator_dto"
)

const (
	// AnalysisBuildConstraint is the build constraint prepended to all generated
	// dist files. It excludes physical dist files from type-checking when the
	// "piko_analysis" build tag is active (used by the LSP inspector), while
	// keeping them included during normal builds where the tag is absent.
	AnalysisBuildConstraint = "//go:build !piko_analysis\n\n"
)

// GeneratedArtefact represents the complete output of the compilation pipeline
// for a single entry-point component. It bundles the generated Go source code
// with the rich analysis result that produced it.
type GeneratedArtefact struct {
	// Result holds the output from annotation, including style blocks and asset
	// references used when building manifest entries.
	Result *annotator_dto.AnnotationResult

	// Component holds the source component metadata; nil if not present.
	Component *annotator_dto.VirtualComponent

	// SuggestedPath is the relative file path where this artefact should be written.
	SuggestedPath string

	// JSArtefactID is the registry artefact ID for the client-side JavaScript
	// file, or empty if the component has no client script.
	JSArtefactID string

	// Content holds the generated Go source code as bytes.
	Content []byte
}
