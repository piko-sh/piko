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

package pml_domain

import (
	"context"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/pml/pml_dto"
)

// Transformer is the primary driving port for the PikoML engine. It
// orchestrates the entire transformation process from a PikoML-flavoured AST
// to a standard HTML AST.
type Transformer interface {
	// Transform performs a multi-pass transformation with style projection.
	//
	// It takes the initial TemplateAST (containing pml-* tags) and configuration,
	// returning a new TemplateAST containing only email-safe HTML, the final CSS
	// string (with media queries and MSO conditionals), and any diagnostics.
	//
	// Takes ast (*ast_domain.TemplateAST) which is the parsed template to
	// transform.
	// Takes config (*pml_dto.Config) which provides the parsed CSS and settings.
	//
	// Returns *ast_domain.TemplateAST which contains the transformed email-safe
	// HTML.
	// Returns string which is the final CSS with media queries and MSO
	// conditionals.
	// Returns []*Error which contains any diagnostics from the transformation.
	Transform(
		ctx context.Context,
		ast *ast_domain.TemplateAST,
		config *pml_dto.Config,
	) (*ast_domain.TemplateAST, string, []*Error)

	// TransformForEmail transforms a template AST for email rendering.
	//
	// It performs the same transformation as Transform but also initialises an
	// email-specific context that tracks asset embedding requests from <pml-img>
	// tags.
	//
	// Takes ast (*TemplateAST) which is the parsed template to transform.
	// Takes config (*Config) which specifies the transformation settings.
	//
	// Returns *TemplateAST which is the transformed abstract syntax tree.
	// Returns string which is the generated CSS.
	// Returns []*EmailAssetRequest which contains asset requests for CID
	// embedding.
	// Returns []*Error which contains any diagnostics from the transformation.
	TransformForEmail(
		ctx context.Context,
		ast *ast_domain.TemplateAST,
		config *pml_dto.Config,
	) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error)
}

// ComponentRegistry is a driven port for PikoML component lookup. The domain uses
// the registry to find components without being coupled to implementations.
type ComponentRegistry interface {
	// Register adds a component to the registry.
	//
	// Takes ctx (context.Context) which carries logging context for trace/request
	// ID propagation.
	// Takes comp (Component) which is the component to add.
	//
	// Returns error when registration fails.
	Register(ctx context.Context, comp Component) error

	// Get retrieves a component by its tag name.
	//
	// Takes tagName (string) which is the HTML tag name, such as "pml-row".
	//
	// Returns Component which is the matching component.
	// Returns bool which is true if found, or false if not present.
	Get(tagName string) (Component, bool)

	// GetAll returns all registered components.
	//
	// Returns []Component which contains all components that have been registered.
	GetAll() []Component

	// MustGet retrieves a component by tag name or panics if not found.
	//
	// Takes tagName (string) which identifies the component to retrieve.
	//
	// Returns Component which is the requested component.
	MustGet(tagName string) Component
}

// validatorPort is a driven port responsible for validating the structure and
// attributes of a PikoML AST.
type validatorPort interface {
	// Validates the TemplateAST by traversing it, checking for valid component
	// usage, parent-child relationships, and attributes against the rules
	// defined by registered components.
	//
	// Takes ast (*ast_domain.TemplateAST) which is the parsed template to check.
	//
	// Returns []*Error which contains any validation failures found.
	Validate(ast *ast_domain.TemplateAST) []*Error
}

// MediaQueryCollector is a driven port that tracks CSS classes needing
// responsive media queries and generates CSS for the document head. It
// implements pml_domain.MediaQueryCollector.
//
// This is the core infrastructure for PikoML's responsive email functionality.
// Components call RegisterClass during transformation to indicate that a CSS
// class needs a mobile media query. After all transformations complete,
// GenerateCSS produces the final style block.
//
// Key behaviours:
//   - Automatic deduplication: multiple columns with the same class generate
//     only one media query.
//   - Group-aware: columns inside pml-no-stack should not register to prevent
//     mobile stacking.
//   - Configurable breakpoint: uses the breakpoint from pml_dto.Config.
type MediaQueryCollector interface {
	// RegisterClass adds a CSS class that needs a mobile-stacking media query.
	//
	// Takes className (string) which is the CSS class (e.g., "pml-col-50").
	// Takes mobileStyles (string) which are the styles to apply on mobile
	// (e.g., "width: 100% !important;"). If the same className is registered
	// multiple times, only one entry is kept.
	RegisterClass(className string, mobileStyles string)

	// RegisterFluidClass adds a CSS class for fluid-on-mobile images or elements.
	//
	// This is a specialised variant for the "pml-fluid-mobile" class used by
	// <pml-img>. Functionally equivalent to RegisterClass but semantically
	// distinct for clarity.
	//
	// Takes className (string) which is the CSS class name to register.
	// Takes mobileStyles (string) which defines the mobile-specific styles.
	RegisterFluidClass(className string, mobileStyles string)

	// GenerateCSS produces the final style block with all media queries.
	//
	// Takes breakpoint (string) which determines the max-width for mobile
	// targeting (e.g., "480px").
	//
	// Returns string which contains complete CSS ready to inject into the
	// document head.
	GenerateCSS(breakpoint string) string
}

// MSOConditionalCollector collects CSS rules that need wrapping in Outlook/MSO
// conditional comments and generates the final conditional block.
//
// Microsoft Outlook uses the Word rendering engine, which has significant CSS
// limitations and requires special handling for common elements like lists. MSO
// conditional comments allow us to target Outlook specifically without
// affecting other email clients.
//
// This collector enables components to register Outlook-specific CSS rules
// during transformation. After all transformations are complete,
// GenerateConditionalBlock produces the final conditional block to inject into
// the document head.
//
// Key behaviours:
//   - Automatic deduplication: The same CSS rule registered multiple times
//     is only output once.
//   - Selector-based organisation: Rules are organised by CSS selector for
//     clean output.
//   - Thread-safe: Can be called concurrently during transformation.
//
// Usage pattern:
// // During component transformation (e.g., in pml_list.go):
//
//	if ctx.MSOConditionalCollector != nil {
//	    ctx.MSOConditionalCollector.RegisterStyle("ul", "margin: 0 !important;")
//	    ctx.MSOConditionalCollector.RegisterStyle("li", "margin-left: 40px !important;")
//	}
//
// // After all transformations (in main engine):
// msoCSS := collector.GenerateConditionalBlock()
// // Inject msoCSS into <head> of final HTML
type MSOConditionalCollector interface {
	// RegisterStyle adds a CSS rule to be wrapped in an MSO conditional comment.
	//
	// Takes selector (string) which is the CSS selector (e.g., "ul",
	// "li.firstListItem").
	// Takes styles (string) which contains the CSS properties to apply
	// (e.g., "margin: 0 !important;").
	//
	// Duplicate selector and styles combinations are kept as a single entry.
	RegisterStyle(selector string, styles string)

	// GenerateConditionalBlock produces the final MSO conditional comment block
	// with all collected styles.
	//
	// Returns string which is a complete conditional block ready to inject into
	// the document head.
	GenerateConditionalBlock() string
}
