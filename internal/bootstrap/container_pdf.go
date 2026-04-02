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

package bootstrap

import "piko.sh/piko/internal/pdfwriter/pdfwriter_domain"

// SetPdfWriterService sets the PDF writer service.
// Called by the daemon builder to set the service built using the
// selected manifest runner and layouter.
//
// Takes s (PdfWriterService) which provides PDF rendering operations.
func (c *Container) SetPdfWriterService(s pdfwriter_domain.PdfWriterService) {
	c.pdfWriterService = s
}

// GetPdfWriterService returns the PdfWriterService previously registered
// by the daemon builder.
//
// Returns pdfwriter_domain.PdfWriterService which provides PDF rendering
// operations, or nil if not yet initialised.
func (c *Container) GetPdfWriterService() pdfwriter_domain.PdfWriterService {
	return c.pdfWriterService
}
