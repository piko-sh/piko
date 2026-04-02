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

// Package pdfwriter_domain contains the core domain types, port interfaces,
// and service implementation for the PDF writer.
//
// The PDF writer orchestrates the full pipeline from a compiled template AST
// to a rendered PDF document. It uses the layouter as a driven adapter for
// CSS resolution, box tree construction, and layout, then paints the result
// to PDF.
//
// The pipeline has three stages: template execution produces an AST, layout
// resolves CSS and builds a positioned box tree, and painting serialises
// that tree into PDF bytes.
package pdfwriter_domain
