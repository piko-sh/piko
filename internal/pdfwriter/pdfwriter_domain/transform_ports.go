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

package pdfwriter_domain

import (
	"context"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

// PdfTransformerPort defines the interface for a single PDF post-processing
// transformation step. Each transformer operates on complete PDF bytes,
// because PDF transformations (signing, encryption, linearisation) require
// random access to the cross-reference table and object tree.
//
// Transformers are composed into chains via PdfTransformerChain and execute
// in ascending priority order. The pattern mirrors
// storage_domain.StreamTransformerPort but uses []byte instead of io.Reader
// and omits Reverse since PDF transformations are one-directional.
type PdfTransformerPort interface {
	// Name returns the unique identifier for this transformer (e.g.
	// "watermark", "aes-256", "pades-b-b", "linearise").
	Name() string

	// Type returns the transformer's category for grouping and validation.
	//
	// Returns pdfwriter_dto.TransformerType which indicates the kind
	// of transformation.
	Type() pdfwriter_dto.TransformerType

	// Priority returns the execution order where lower values run first.
	//
	// Recommended ranges: 100-199 for content, 200-299 for compliance,
	// 300-399 for delivery, 400-499 for security.
	//
	// Returns int which indicates priority.
	Priority() int

	// Transform applies this transformation to the provided PDF bytes and
	// returns the modified PDF bytes.
	//
	// Takes ctx (context.Context) which carries cancellation and tracing.
	// Takes pdf ([]byte) which is the input PDF document.
	// Takes options (any) which is sourced from
	// TransformConfig.TransformerOptions[Name()].
	//
	// Returns []byte which is the transformed PDF.
	// Returns error when the transformation cannot be applied.
	Transform(ctx context.Context, pdf []byte, options any) ([]byte, error)
}
