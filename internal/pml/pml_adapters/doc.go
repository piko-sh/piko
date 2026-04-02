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

// Package pml_adapters implements the collector interfaces defined in
// pml_domain, handling responsive CSS media query generation and Microsoft
// Outlook conditional comment blocks for email compatibility.
//
// # Usage
//
// Collectors are typically created at the start of a transformation pass
// and passed through the rendering pipeline:
//
//	mqCollector := pml_adapters.NewMediaQueryCollector()
//	mqCollector.RegisterClass("pml-col-50", "width: 100% !important;")
//	css := mqCollector.GenerateCSS("480px")
//
//	msoCollector := pml_adapters.NewMSOConditionalCollector()
//	msoCollector.RegisterStyle("ul", "margin: 0 !important;")
//	block := msoCollector.GenerateConditionalBlock()
//
// # Thread safety
//
// All collector implementations are safe for concurrent use. Methods use
// internal synchronisation to protect shared state.
package pml_adapters
