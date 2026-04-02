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

package templater_adapters

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// simpleRenderer implements RendererPort for testing purposes.
// It outputs debug information instead of full HTML rendering.
type simpleRenderer struct{}

// RenderPage writes debug information for a page template.
//
// Takes params (templater_domain.RenderPageParams) which contains the page
// data to render.
//
// Returns error when writing the debug output fails.
func (r *simpleRenderer) RenderPage(_ context.Context, params templater_domain.RenderPageParams) error {
	return r.renderDebug("Page", params)
}

// RenderPartial outputs debug information about the partial being rendered.
//
// Takes params (templater_domain.RenderPageParams) which specifies the partial
// to render and its context.
//
// Returns error when the debug output fails to write.
func (r *simpleRenderer) RenderPartial(_ context.Context, params templater_domain.RenderPageParams) error {
	return r.renderDebug("Partial", params)
}

// RenderEmail writes debug details about the email being rendered.
//
// Takes params (templater_domain.RenderEmailParams) which contains the email
// template data including metadata, AST, and styling to display.
//
// Returns error when the metadata cannot be marshalled or writing fails.
func (*simpleRenderer) RenderEmail(_ context.Context, params templater_domain.RenderEmailParams) error {
	snippetBytes, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("could not marshal snippet data: %w", err)
	}
	astNodeCount := 0
	if params.TemplateAST != nil {
		astNodeCount = len(params.TemplateAST.RootNodes)
	}
	result := fmt.Sprintf(
		"Email: %s\nSnippetData: %s\nAST RootNodes: %d\nStyling: %d bytes\n",
		params.PageID,
		string(snippetBytes),
		astNodeCount,
		len(params.Styling),
	)
	if _, err = params.Writer.Write([]byte(result)); err != nil {
		return fmt.Errorf("writing email render output: %w", err)
	}
	return nil
}

// CollectMetadata returns empty metadata for testing purposes.
//
// Returns []render_dto.LinkHeader which is always empty.
// Returns *ProbeData which is always nil.
// Returns error which is always nil.
func (*simpleRenderer) CollectMetadata(
	_ context.Context,
	_ *http.Request,
	_ *templater_dto.InternalMetadata,
	_ *config.WebsiteConfig,
) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
	return []render_dto.LinkHeader{}, nil, nil
}

// RenderASTToPlainText extracts plain text from an AST for email rendering.
//
// Takes templateAST (*ast_domain.TemplateAST) which is the parsed template to
// extract text from.
//
// Returns string which contains the extracted plain text content.
// Returns error which is always nil in the current implementation.
func (*simpleRenderer) RenderASTToPlainText(
	_ context.Context,
	templateAST *ast_domain.TemplateAST,
) (string, error) {
	if templateAST == nil {
		return "", nil
	}
	var b strings.Builder
	var walk func(n *ast_domain.TemplateNode)
	walk = func(n *ast_domain.TemplateNode) {
		switch n.NodeType {
		case ast_domain.NodeText:
			b.WriteString(n.TextContent)
		case ast_domain.NodeElement, ast_domain.NodeFragment:
			for _, c := range n.Children {
				walk(c)
			}
		default:
		}
	}
	for _, rn := range templateAST.RootNodes {
		walk(rn)
	}
	return b.String(), nil
}

// GetLastEmailAssetRequests returns an empty slice for testing.
//
// Returns []*email_dto.EmailAssetRequest which is always empty for this mock.
func (*simpleRenderer) GetLastEmailAssetRequests() []*email_dto.EmailAssetRequest {
	return []*email_dto.EmailAssetRequest{}
}

// renderDebug writes debug output about the page being rendered.
//
// Takes label (string) which identifies the type of debug output.
// Takes params (templater_domain.RenderPageParams) which contains the page
// definition, metadata, template AST, and writer for output.
//
// Returns error when marshalling metadata fails or writing output fails.
func (*simpleRenderer) renderDebug(label string, params templater_domain.RenderPageParams) error {
	snippetBytes, err := json.Marshal(params.Metadata)
	if err != nil {
		return fmt.Errorf("could not marshal snippet data: %w", err)
	}
	astNodeCount := 0
	if params.TemplateAST != nil {
		astNodeCount = len(params.TemplateAST.RootNodes)
	}
	result := fmt.Sprintf(
		"%s: %s\nSnippetData: %s\nAST RootNodes: %d\n",
		label,
		params.PageDefinition.NormalisedPath,
		string(snippetBytes),
		astNodeCount,
	)
	if _, err = params.Writer.Write([]byte(result)); err != nil {
		return fmt.Errorf("writing %s render output: %w", strings.ToLower(label), err)
	}
	return nil
}

// newSimpleRenderer creates a new simple test renderer.
//
// Returns templater_domain.RendererPort which is the renderer ready for use.
func newSimpleRenderer() templater_domain.RendererPort {
	return &simpleRenderer{}
}
