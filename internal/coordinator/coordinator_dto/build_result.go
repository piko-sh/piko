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

package coordinator_dto

import (
	"piko.sh/piko/internal/annotator/annotator_dto"
)

// BuildResult wraps the annotator's full result to provide a stable
// contract for coordinator consumers. This DTO can be extended with metadata
// without breaking consumers, as long as the core AnnotationResult is preserved.
type BuildResult struct {
	// AnnotationResult is the primary payload of a successful build. It contains the
	// fully analysed and annotated ASTs for each page, the complete dependency graph,
	// the virtual Go module, and the final aggregated asset manifest. It is the
	// complete, authoritative representation of the compiled project.
	*annotator_dto.ProjectAnnotationResult
}
