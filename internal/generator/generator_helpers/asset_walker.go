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

package generator_helpers

import (
	"path"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// CollectAssetRefs walks a given AST and finds all asset references, such as
// those defined in `<piko:svg src="...">` tags. It uses the modern
// `ast_domain.TemplateAST.Walk` method for efficient traversal.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the template
// tree to walk.
//
// Returns []templater_dto.AssetRef which contains the deduplicated
// asset references found.
func CollectAssetRefs(templateAST *ast_domain.TemplateAST, _ string) []templater_dto.AssetRef {
	if templateAST == nil {
		return nil
	}

	seen := make(map[string]templater_dto.AssetRef)

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if ref, ok := extractSVGAssetRef(node); ok {
			key := ref.Kind + ":" + ref.Path
			if _, exists := seen[key]; !exists {
				seen[key] = ref
			}
		}
		return true
	})

	if len(seen) == 0 {
		return nil
	}

	result := make([]templater_dto.AssetRef, 0, len(seen))
	for _, ref := range seen {
		result = append(result, ref)
	}
	return result
}

// extractSVGAssetRef checks if a node is a piko:svg element and extracts its
// src attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns templater_dto.AssetRef which contains the SVG asset reference.
// Returns bool which indicates whether a valid SVG reference was found.
func extractSVGAssetRef(node *ast_domain.TemplateNode) (templater_dto.AssetRef, bool) {
	if node.NodeType != ast_domain.NodeElement || !strings.EqualFold(node.TagName, "piko:svg") {
		return templater_dto.AssetRef{}, false
	}

	for i := range node.Attributes {
		if strings.EqualFold(node.Attributes[i].Name, "src") && node.Attributes[i].Value != "" {
			return templater_dto.AssetRef{
				Kind: "svg",
				Path: path.Clean(node.Attributes[i].Value),
			}, true
		}
	}
	return templater_dto.AssetRef{}, false
}
