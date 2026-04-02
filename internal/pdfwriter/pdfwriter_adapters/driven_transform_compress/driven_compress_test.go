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

package driven_transform_compress_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_transform_compress"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_dto"
)

var zstdMagic = []byte{0x28, 0xB5, 0x2F, 0xFD}

func newTransformer(t *testing.T) *driven_transform_compress.CompressTransformer {
	t.Helper()
	ct, err := driven_transform_compress.New()
	require.NoError(t, err)
	return ct
}

func decompressZstd(t *testing.T, data []byte) []byte {
	t.Helper()
	dec, err := zstd.NewReader(nil)
	require.NoError(t, err)
	defer dec.Close()
	out, err := dec.DecodeAll(data, nil)
	require.NoError(t, err)
	return out
}

func TestCompressTransformer_ImplementsPort(t *testing.T) {
	var _ pdfwriter_domain.PdfTransformerPort = newTransformer(t)
}

func TestCompressTransformer_Metadata(t *testing.T) {
	ct := newTransformer(t)
	assert.Equal(t, "compress", ct.Name())
	assert.Equal(t, pdfwriter_dto.TransformerCompression, ct.Type())
	assert.Equal(t, 900, ct.Priority())
}

func TestCompressTransformer_RoundTrip(t *testing.T) {
	ct := newTransformer(t)
	input := []byte("%PDF-1.7\n% some pdf content here for testing\n%%EOF\n")

	compressed, err := ct.Transform(
		context.Background(), input, pdfwriter_dto.CompressOptions{},
	)
	require.NoError(t, err)

	assert.True(t, bytes.HasPrefix(compressed, zstdMagic),
		"output should start with zstd magic bytes")
	assert.NotEqual(t, input, compressed)

	decompressed := decompressZstd(t, compressed)
	assert.Equal(t, input, decompressed)
}

func TestCompressTransformer_ReducesSize(t *testing.T) {
	ct := newTransformer(t)

	input := []byte(strings.Repeat("BT /Helv 12 Tf (Hello World) Tj ET\n", 100))

	compressed, err := ct.Transform(
		context.Background(), input, pdfwriter_dto.CompressOptions{},
	)
	require.NoError(t, err)
	assert.Less(t, len(compressed), len(input),
		"compressed output should be smaller than repetitive input")
}

func TestCompressTransformer_DefaultsToZstd(t *testing.T) {
	ct := newTransformer(t)
	input := []byte("test data")

	compressed, err := ct.Transform(
		context.Background(), input, pdfwriter_dto.CompressOptions{},
	)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(compressed, zstdMagic),
		"empty algorithm should default to zstd")
}

func TestCompressTransformer_ExplicitZstd(t *testing.T) {
	ct := newTransformer(t)
	input := []byte("test data")

	compressed, err := ct.Transform(
		context.Background(), input, pdfwriter_dto.CompressOptions{
			Algorithm: pdfwriter_dto.CompressZstd,
		},
	)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(compressed, zstdMagic))

	decompressed := decompressZstd(t, compressed)
	assert.Equal(t, input, decompressed)
}

func TestCompressTransformer_UnsupportedAlgorithm(t *testing.T) {
	ct := newTransformer(t)

	_, err := ct.Transform(
		context.Background(), []byte("data"), pdfwriter_dto.CompressOptions{
			Algorithm: "lz4",
		},
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

func TestCompressTransformer_EmptyInput(t *testing.T) {
	ct := newTransformer(t)

	compressed, err := ct.Transform(
		context.Background(), []byte{}, pdfwriter_dto.CompressOptions{},
	)
	require.NoError(t, err)
	assert.Empty(t, compressed)
}

func TestCompressTransformer_PointerOptions(t *testing.T) {
	ct := newTransformer(t)
	input := []byte("pointer options test")

	compressed, err := ct.Transform(
		context.Background(), input, &pdfwriter_dto.CompressOptions{},
	)
	require.NoError(t, err)

	decompressed := decompressZstd(t, compressed)
	assert.Equal(t, input, decompressed)
}

func TestCompressTransformer_InvalidOptions(t *testing.T) {
	ct := newTransformer(t)

	_, err := ct.Transform(context.Background(), []byte("data"), "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected CompressOptions")
}

func TestCompressTransformer_NilPointerOptions(t *testing.T) {
	ct := newTransformer(t)

	_, err := ct.Transform(
		context.Background(), []byte("data"), (*pdfwriter_dto.CompressOptions)(nil),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil options")
}
