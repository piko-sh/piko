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

// Package driven_transform_pdfua adds PDF/UA (Universal Accessibility)
// structural metadata as a post-processing transformer.
//
// All enhancements are additive: they add missing entries to the document
// catalog but never overwrite existing ones. This includes marking the PDF
// as tagged, adding a minimal structure tree root, setting a default
// document language, displaying the document title in viewer preferences,
// and adding a title entry to the document info dictionary.
package driven_transform_pdfua
