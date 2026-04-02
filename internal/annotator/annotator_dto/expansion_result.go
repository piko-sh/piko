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

// ExpansionResult holds the output of expanding a component tree.
type ExpansionResult struct {
	// FlattenedAST is the fully processed AST for the entry-point component.
	// All partials are inlined and all nodes are marked with their source
	// locations.
	FlattenedAST *ast_domain.TemplateAST

	// CombinedCSS holds all merged and processed CSS from the component tree.
	CombinedCSS string

	// PotentialInvocations holds all unique partial invocations found during
	// expansion. The keys and props in these invocations are not yet in their
	// final form.
	PotentialInvocations []*PartialInvocation
}
