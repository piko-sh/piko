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

package driven_transform_objstm

import (
	"bytes"
	"compress/lzw"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/pdfparse"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

const (
	// defaultPriority is the execution order for the object stream transformer.
	defaultPriority = 300

	// maxDecompressedStreamSize is the upper bound for decompressed stream
	// data to guard against zip-bomb or corrupt streams (256 MiB).
	maxDecompressedStreamSize = 256 << 20

	// transformerName is the identifier for this transformer.
	transformerName = "objstm-compress"

	// filterKey is the PDF dictionary key for the stream filter.
	filterKey = "Filter"

	// decodeParmsKey is the PDF dictionary key for filter decode parameters.
	decodeParmsKey = "DecodeParms"

	// filterLZW is the PDF name for LZW compression.
	filterLZW = "LZWDecode"

	// lzwLitWidth is the literal code width for LZW decompression as used
	// by PDF (always 8 bits).
	lzwLitWidth = 8
)

// ObjStmTransformer re-encodes PDF stream objects to ensure efficient
// FlateDecode compression.
//
// It decodes any LZW-compressed streams and removes the filter so the writer
// can re-compress them with FlateDecode. Streams that are already
// FlateDecode-compressed or use other filters are left unchanged.
// Uncompressed streams are automatically compressed by the writer.
type ObjStmTransformer struct {
	// name is the transformer identifier.
	name string

	// priority is the execution order.
	priority int
}

var _ pdfwriter_domain.PdfTransformerPort = (*ObjStmTransformer)(nil)

// New creates a new object stream compression transformer with default
// name and priority.
//
// Returns *ObjStmTransformer which is ready for use.
func New() *ObjStmTransformer {
	return &ObjStmTransformer{
		name:     transformerName,
		priority: defaultPriority,
	}
}

// Name returns the transformer's name.
//
// Returns string which identifies this transformer.
func (t *ObjStmTransformer) Name() string { return t.name }

// Type returns TransformerDelivery.
//
// Returns pdfwriter_dto.TransformerType which categorises this as a delivery
// transformer.
func (*ObjStmTransformer) Type() pdfwriter_dto.TransformerType {
	return pdfwriter_dto.TransformerDelivery
}

// Priority returns the execution priority.
//
// Returns int which is the transformer's position in the processing order.
func (t *ObjStmTransformer) Priority() int { return t.priority }

// Transform re-encodes stream objects in the PDF to use FlateDecode
// compression.
//
// LZW-compressed streams are decoded and re-compressed; uncompressed streams
// are compressed by the writer automatically. Options must be ObjStmOptions
// or *ObjStmOptions.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes pdf ([]byte) which is the input PDF document.
// Takes options (any) which must be ObjStmOptions or *ObjStmOptions.
//
// Returns []byte which is the re-compressed PDF.
// Returns error when the PDF cannot be parsed or re-compression fails.
func (*ObjStmTransformer) Transform(ctx context.Context, pdf []byte, options any) ([]byte, error) {
	if _, err := castOptions(options); err != nil {
		return nil, err
	}

	doc, err := pdfparse.Parse(pdf)
	if err != nil {
		return nil, fmt.Errorf("objstm-compress: parsing PDF: %w", err)
	}

	writer, err := pdfparse.NewWriterFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("objstm-compress: creating writer: %w", err)
	}

	for _, objNum := range doc.ObjectNumbers() {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("objstm-compress: %w", ctx.Err())
		}

		obj, err := doc.GetObject(objNum)
		if err != nil {
			slog.Warn("objstm-compress: skipping object that could not be retrieved",
				slog.Int("object", objNum), slog.String("error", err.Error()))
			continue
		}
		if obj.Type != pdfparse.ObjectStream {
			continue
		}

		rewritten, changed, err := reencodeStream(obj)
		if err != nil {
			slog.Warn("objstm-compress: skipping stream that could not be re-encoded",
				slog.Int("object", objNum), slog.String("error", err.Error()))
			continue
		}
		if changed {
			writer.SetObject(objNum, rewritten)
		}
	}

	output, err := writer.Write()
	if err != nil {
		return nil, fmt.Errorf("objstm-compress: writing PDF: %w", err)
	}
	return output, nil
}

// castOptions extracts ObjStmOptions from the generic options.
//
// Takes options (any) which is the untyped options value to assert.
//
// Returns pdfwriter_dto.ObjStmOptions which holds the typed options.
// Returns error when the options type does not match.
func castOptions(options any) (pdfwriter_dto.ObjStmOptions, error) {
	switch v := options.(type) {
	case pdfwriter_dto.ObjStmOptions:
		return v, nil
	case *pdfwriter_dto.ObjStmOptions:
		if v == nil {
			return pdfwriter_dto.ObjStmOptions{}, errors.New("objstm-compress: nil options pointer")
		}
		return *v, nil
	default:
		return pdfwriter_dto.ObjStmOptions{}, fmt.Errorf("objstm-compress: expected ObjStmOptions, got %T", options)
	}
}

// reencodeStream inspects a stream object's filter and, if it uses LZW
// compression, decodes the data and removes the filter so the writer will
// re-compress with FlateDecode.
//
// Takes obj (pdfparse.Object) which is the stream object to inspect.
//
// Returns pdfparse.Object which is the (possibly modified) stream object.
// Returns bool which indicates whether the object was changed.
// Returns error when LZW decoding fails.
func reencodeStream(obj pdfparse.Object) (pdfparse.Object, bool, error) {
	dict, ok := obj.Value.(pdfparse.Dict)
	if !ok {
		return obj, false, nil
	}

	filter := dict.GetName(filterKey)
	if filter != filterLZW {
		return obj, false, nil
	}

	decoded, err := decodeLZW(obj.StreamData)
	if err != nil {
		return obj, false, fmt.Errorf("decoding LZW stream: %w", err)
	}

	dict.Remove(filterKey)
	dict.Remove(decodeParmsKey)

	return pdfparse.StreamObj(dict, decoded), true, nil
}

// decodeLZW decompresses LZW-encoded data as specified by the PDF
// standard. PDF uses MSB (most significant bit first) byte ordering.
//
// Takes data ([]byte) which is the LZW-compressed stream bytes.
//
// Returns []byte which is the decompressed data.
// Returns error when decompression fails.
func decodeLZW(data []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(data), lzw.MSB, lzwLitWidth)
	defer reader.Close()

	decoded, err := io.ReadAll(io.LimitReader(reader, maxDecompressedStreamSize))
	if err != nil {
		return nil, fmt.Errorf("decompressing LZW stream: %w", err)
	}
	return decoded, nil
}
