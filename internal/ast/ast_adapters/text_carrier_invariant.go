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

package ast_adapters

import (
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
)

// checkSingleTextCarrier rejects a decoded node that carries its text in more than one of
// TextContent, RichText and TextContentWriter.
//
// Takes node (*ast_domain.TemplateNode) which is the freshly decoded node.
//
// Returns error when more than one text field is populated.
func checkSingleTextCarrier(node *ast_domain.TemplateNode) error {
	carriers := 0
	if node.TextContent != "" {
		carriers++
	}
	if len(node.RichText) > 0 {
		carriers++
	}
	if node.TextContentWriter.HasParts() {
		carriers++
	}
	if carriers > 1 {
		return fmt.Errorf(
			"node type %d %q carries text in %d fields at once; TextContent, RichText and "+
				"TextContentWriter are mutually exclusive", node.NodeType, node.TagName, carriers)
	}
	return nil
}
