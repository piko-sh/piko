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

// Package layouter_domain contains the core domain types, port interfaces, and
// algorithms for the Piko layout engine.
//
// The layout engine transforms a post-render [ast_domain.TemplateAST] into a
// tree of positioned [LayoutBox] nodes with resolved CSS styles. External
// consumers (such as pdfwriter) use this positioned box tree to produce
// visual output in their chosen format.
//
// # Pipeline
//
// The layout pipeline runs in four phases:
//
//  1. Style resolution: CSS cascade, inheritance, and unit conversion
//  2. Box tree construction: AST nodes become [LayoutBox] nodes
//  3. Layout: boxes are sized and positioned on an infinite canvas
//  4. Pagination: the infinite canvas is split into discrete pages
//
// # Design rationale
//
// PDF rendering requires CSS layout computation that browsers perform
// natively but Go does not provide. Rather than shelling out to a headless
// browser, this package implements a CSS layout engine directly so that
// Piko can produce paginated PDF output from the same HTML/CSS templates
// used for web rendering, without external dependencies or child processes.
package layouter_domain
