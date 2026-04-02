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

package pdfwriter_adapters

import (
	"context"
	"fmt"
	"net/http"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// TemplateRunnerAdapter adapts templater_domain.ManifestRunnerPort to
// pdfwriter_domain.TemplateRunnerPort.
type TemplateRunnerAdapter struct {
	// runner is the manifest runner that executes compiled templates.
	runner templater_domain.ManifestRunnerPort
}

// NewTemplateRunnerAdapter creates a new template runner adapter.
//
// Takes runner (templater_domain.ManifestRunnerPort) which provides
// access to the compiled template manifest.
//
// Returns *TemplateRunnerAdapter which implements
// pdfwriter_domain.TemplateRunnerPort.
func NewTemplateRunnerAdapter(runner templater_domain.ManifestRunnerPort) *TemplateRunnerAdapter {
	return &TemplateRunnerAdapter{runner: runner}
}

// RunPdfWithProps executes a PDF template with the given props and
// returns the AST and styling.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes templatePath (string) which is the path to the PDF template.
// Takes request (*http.Request) which provides the HTTP context.
// Takes props (any) which contains the data to pass to the template.
//
// Returns *ast_domain.TemplateAST which is the compiled template tree.
// Returns string which is the CSS styling from the template.
// Returns error when the template cannot be found or executed.
func (adapter *TemplateRunnerAdapter) RunPdfWithProps(
	ctx context.Context,
	templatePath string,
	request *http.Request,
	props any,
) (*ast_domain.TemplateAST, string, error) {
	pageDefinition := templater_dto.PageDefinition{
		OriginalPath:   templatePath,
		NormalisedPath: "",
		TemplateHTML:   "",
	}

	templateAST, _, styling, err := adapter.runner.RunPartialWithProps(ctx, pageDefinition, request, props)
	if err != nil {
		return nil, "", fmt.Errorf("failed to run PDF template '%s': %w", templatePath, err)
	}

	return templateAST, styling, nil
}
