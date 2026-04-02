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

// Package pdf provides a builder-based interface for rendering PDF documents
// from Piko templates, with support for metadata, watermarks, PDF/A
// conformance, accessibility tagging, page labels, SVG vector rendering,
// custom fonts, and post-processing transformations.
//
// If the Piko framework has been bootstrapped (daemon mode),
// [GetDefaultService] returns the pre-configured service instance. For
// standalone use (tests, CLI), create a service from a compiled manifest
// with [NewServiceFromManifest].
//
// # Daemon usage
//
//	service, err := pdf.GetDefaultService()
//	if err != nil {
//	    return err
//	}
//	result, err := service.NewRender().
//	    Template("pdfs/invoice.pk").
//	    Request(r).
//	    Props(invoiceData).
//	    Metadata(pdf.Metadata{Title: "Invoice #123"}).
//	    Do(ctx)
//
// # Standalone usage (tests / CLI)
//
//	service, err := pdf.NewServiceFromManifest("dist/manifest.bin")
//	if err != nil {
//	    return err
//	}
//	result, err := service.NewRender().
//	    Template("pdfs/invoice.pk").
//	    PdfA(pdf.PdfA2B).
//	    Do(ctx)
//
// # Thread safety
//
// [Service] and [RenderBuilder] are safe for concurrent use. Each call
// to NewRender creates an independent builder instance.
package pdf
