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

package annotator_domain

// Validates template structures and expressions against component contracts and Go type rules during semantic analysis.
// Performs compile-time checks for type safety, required props, valid member access, and expression correctness across the AST.

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// validatePMLUsage checks if PikoML tags are used outside email templates.
//
// PikoML (pml-*) tags are built for HTML email rendering. They use table-based
// layouts to work well in email clients. Using them in web pages causes layout
// problems and is not supported.
//
// When the component is an email template or has no template, returns nil.
//
// Takes component (*annotator_dto.ParsedComponent) which provides the parsed
// template and metadata to check.
//
// Returns []*ast_domain.Diagnostic which contains warnings for each pml-* tag
// found, or nil if there are no problems.
func validatePMLUsage(component *annotator_dto.ParsedComponent) []*ast_domain.Diagnostic {
	if component.ComponentType == "email" || component.Template == nil {
		return nil
	}

	var diagnostics []*ast_domain.Diagnostic

	component.Template.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.NodeType == ast_domain.NodeElement && strings.HasPrefix(node.TagName, "pml-") {
			message := fmt.Sprintf(
				"PikoML component <%s> is not supported outside of email templates. "+
					"PikoML components are specifically designed for HTML email rendering "+
					"and should only be used in .pk files within the 'emails' directory. "+
					"Using them in web pages may result in unexpected table-based layouts. "+
					"Consider using standard HTML elements or Piko components (<piko:*>) instead.",
				node.TagName,
			)

			diagnostic := ast_domain.NewDiagnosticWithCode(
				ast_domain.Warning,
				message,
				node.TagName,
				annotator_dto.CodeDeprecatedElement,
				node.Location,
				component.SourcePath,
			)
			diagnostics = append(diagnostics, diagnostic)
		}
		return true
	})

	return diagnostics
}
