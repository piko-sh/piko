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
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
)

// metaComponentTags defines a set of Piko-specific tags that should be ignored
// when searching for potential user-defined custom elements.
var metaComponentTags = map[string]bool{
	"piko:svg":   true,
	"piko:a":     true,
	"piko:video": true,
}

// CollectComponentComponents walks a given AST to find all potential custom
// element tags. A tag is considered a potential custom element if its name
// contains a hyphen or is prefixed with "piko:", following common conventions.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the template
// tree to walk.
//
// Returns []string which contains the unique custom element tag names
// found.
func CollectComponentComponents(templateAST *ast_domain.TemplateAST, _ string) []string {
	if templateAST == nil {
		return nil
	}

	seen := make(map[string]struct{})

	templateAST.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType != ast_domain.NodeElement {
			return true
		}

		if metaComponentTags[node.TagName] {
			return true
		}

		isPotentialCustomElement := strings.Contains(node.TagName, "-") || strings.HasPrefix(node.TagName, "piko:")

		if isPotentialCustomElement {
			seen[node.TagName] = struct{}{}
		}

		return true
	})

	if len(seen) == 0 {
		return nil
	}

	result := make([]string, 0, len(seen))
	for tag := range seen {
		result = append(result, tag)
	}
	return result
}
