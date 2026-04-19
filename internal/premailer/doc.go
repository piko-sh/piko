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

// Package premailer handles CSS inlining and email compatibility
// transformations.
//
// It processes HTML email templates by extracting CSS from <style> tags,
// parsing it using esbuild's CSS parser, and inlining styles directly
// onto elements. It handles CSS specificity, variable resolution, shorthand
// expansion, and validates both CSS properties and HTML elements for email
// client compatibility.
//
// # Usage
//
//	p := premailer.New(templateAST,
//	    premailer.WithExpandShorthands(true),
//	    premailer.WithTheme(themeVars),
//	    premailer.WithLinkQueryParams(utmParams),
//	)
//	transformed, err := p.Transform()
//
// # Features
//
// The premailer performs several transformations critical for email
// compatibility:
//
//   - CSS Variable Resolution: Resolves var() functions using a theme map, as
//     email
//     clients cannot evaluate CSS variables at runtime
//   - Shorthand Expansion: Expands properties like margin and border into
//     longhands
//     for Outlook compatibility
//   - Colour Normalisation: Converts colour values to hex format for
//     consistency
//   - Link Parameter Injection: Appends UTM or tracking parameters to all links
//   - !important Handling: Optionally preserves !important rules in a <style>
//     block
//     for dual-path styling in webmail clients
//
// # Validation
//
// The package validates templates for email compatibility, producing
// diagnostics for:
//
//   - Unsupported CSS properties (flexbox, grid, animations, transforms)
//   - Problematic HTML elements (<script>, <form>, <iframe>, <svg>)
//   - CSS features with poor client support (gradients, calc(), shadows)
//
// # Leftover rules
//
// Rules that cannot be inlined (pseudo-classes like :hover, @media queries) are
// collected as "leftover rules" and reinserted into a <style> tag. The
// [WithMakeLeftoverImportant] option marks these as !important for Gmail
// compatibility.
//
// # Integration
//
// Depends on:
//
//   - ast_domain: Provides [ast_domain.TemplateAST] and node types that
//     the premailer walks and mutates
//   - css_parser/css_ast: esbuild's CSS parser for accurate CSS parsing
//     including specificity calculation and selector matching
//   - css_printer: Rebuilds CSS text from the AST for leftover rules
//
// # Thread safety
//
// A [Premailer] instance mutates the provided [ast_domain.TemplateAST] in place
// and must not be shared between goroutines.
package premailer
