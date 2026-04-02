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

// LinkingResult holds the output of linking a template with its partials.
type LinkingResult struct {
	// LinkedAST is the flattened template AST with partials inlined
	// and types resolved.
	LinkedAST *ast_domain.TemplateAST

	// VirtualModule is the Go module structure built during the linking step.
	VirtualModule *VirtualModule

	// CombinedCSS is the merged CSS content from all components in the tree.
	CombinedCSS string

	// UniqueInvocations contains the partial invocations found during linking.
	UniqueInvocations []*PartialInvocation
}
