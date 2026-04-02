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

package driven_transform_compress

import (
	"context"
	"errors"
	"fmt"

	"github.com/klauspost/compress/zstd"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the compression transformer.
	defaultPriority = 900
)

// CompressTransformer compresses the final PDF output using a
// configurable algorithm. The encoder is created once during
// construction and reused across calls; EncodeAll is safe for
// concurrent use.
//
// Because the output is compressed bytes (not a valid PDF), this
// transformer should be the last step in any transform chain.
type CompressTransformer struct {
	// encoder is the reusable zstd encoder.
	encoder *zstd.Encoder

	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*CompressTransformer)(nil)

// New creates a new compression transformer with a zstd encoder
// initialised at the default compression level.
//
// Returns *CompressTransformer which is ready for use.
// Returns error when the zstd encoder cannot be created.
func New() (*CompressTransformer, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("compress: creating zstd encoder: %w", err)
	}
	return &CompressTransformer{
		name:     "compress",
		priority: defaultPriority,
		encoder:  enc,
	}, nil
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *CompressTransformer) Name() string { return t.name }

// Type returns TransformerCompression.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a
// compression transformer.
func (*CompressTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerCompression
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *CompressTransformer) Priority() int { return t.priority }

// Transform compresses the PDF bytes using the algorithm specified in
// options. Defaults to zstd when the algorithm is empty.
//
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be CompressOptions or *CompressOptions.
//
// Returns []byte which is the compressed output.
// Returns error when the options are invalid or the algorithm is
// unsupported.
func (t *CompressTransformer) Transform(_ context.Context, pdf []byte, options any) ([]byte, error) {
	opts, err := castOptions(options)
	if err != nil {
		return nil, err
	}
	applyDefaults(&opts)

	if len(pdf) == 0 {
		return pdf, nil
	}

	switch opts.Algorithm {
	case pdfwriter_dto.CompressZstd:
		return t.encoder.EncodeAll(pdf, make([]byte, 0, len(pdf))), nil
	default:
		return nil, fmt.Errorf("compress: unsupported algorithm %q", opts.Algorithm)
	}
}

// castOptions extracts CompressOptions from the generic options.
//
// Takes options (any) which is the untyped options value to assert.
//
// Returns pdfwriter_dto.CompressOptions which holds the typed options.
// Returns error when the options type does not match.
func castOptions(options any) (pdfwriter_dto.CompressOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.CompressOptions:
		return v, nil
	case *pdfwriter_dto.CompressOptions:
		if v == nil {
			return pdfwriter_dto.CompressOptions{}, errors.New("compress: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.CompressOptions{}, fmt.Errorf(
			"compress: expected CompressOptions, got %T", options,
		)
	}
}

// applyDefaults fills zero-value fields with sensible defaults.
//
// Takes opts (*pdfwriter_dto.CompressOptions) which is the options struct
// to populate.
func applyDefaults(opts *pdfwriter_dto.CompressOptions) {
	if opts.Algorithm == "" {
		opts.Algorithm = pdfwriter_dto.CompressZstd
	}
}
