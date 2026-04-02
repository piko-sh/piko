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

// Package driven_transform_pdfa applies PDF/A archival conformance as a
// post-processing transformer.
//
// It adds XMP metadata with the appropriate pdfaid namespace declarations,
// inserts an sRGB output intent into the document catalog, and removes
// prohibited features such as additional actions (/AA), JavaScript, and
// (for PDF/A-1b) transparency groups.
package driven_transform_pdfa
