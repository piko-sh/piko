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
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImageEmbedder_HasImagesReturnsFalse(t *testing.T) {
	embedder := NewImageEmbedder()

	assert.False(t, embedder.HasImages(), "expected HasImages to return false for a freshly created embedder")
}

func TestRegisterImage_FirstRegistrationReturnsIm1(t *testing.T) {
	embedder := NewImageEmbedder()

	name := embedder.RegisterImage("photo.jpg", []byte{0xFF}, "jpeg", 100, 50)

	assert.Equal(t, "Im1", name)
}

func TestRegisterImage_SameSourceReturnsSameName(t *testing.T) {
	embedder := NewImageEmbedder()

	first := embedder.RegisterImage("photo.jpg", []byte{0xFF}, "jpeg", 100, 50)
	second := embedder.RegisterImage("photo.jpg", []byte{0xFF}, "jpeg", 100, 50)

	assert.Equal(t, first, second, "expected duplicate registration to return same name")
}

func TestRegisterImage_DifferentSourcesReturnDifferentNames(t *testing.T) {
	embedder := NewImageEmbedder()

	first := embedder.RegisterImage("photo_a.jpg", []byte{0xFF}, "jpeg", 100, 50)
	second := embedder.RegisterImage("photo_b.jpg", []byte{0xFE}, "jpeg", 200, 100)

	assert.Equal(t, "Im1", first)
	assert.Equal(t, "Im2", second)
}

func TestHasImages_ReturnsTrueAfterRegistration(t *testing.T) {
	embedder := NewImageEmbedder()
	embedder.RegisterImage("photo.jpg", []byte{0xFF}, "jpeg", 100, 50)

	assert.True(t, embedder.HasImages(), "expected HasImages to return true after registering an image")
}

func TestExtractImageDimensions_JPEG(t *testing.T) {

	data := []byte{
		0xFF, 0xD8,
		0xFF, 0xC0,
		0x00, 0x11,
		0x08,
		0x00, 0xC8,
		0x01, 0x90,
		0x03,
		0x01, 0x22, 0x00,
		0x02, 0x11, 0x01,
		0x03, 0x11, 0x01,
	}

	width, height := ExtractImageDimensions(data, "jpeg")

	assert.Equal(t, 400, width)
	assert.Equal(t, 200, height)
}

func TestExtractImageDimensions_JPEG_Truncated(t *testing.T) {

	data := []byte{0xFF, 0xD8}

	width, height := ExtractImageDimensions(data, "jpeg")

	assert.Zero(t, width, "expected 0 width for truncated JPEG")
	assert.Zero(t, height, "expected 0 height for truncated JPEG")
}

func TestExtractImageDimensions_PNG(t *testing.T) {

	data := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x01, 0x90,
		0x00, 0x00, 0x00, 0xC8,
		0x08,
		0x02,
		0x00,
		0x00,
		0x00,
	}

	width, height := ExtractImageDimensions(data, "png")

	assert.Equal(t, 400, width)
	assert.Equal(t, 200, height)
}

func TestExtractImageDimensions_PNG_Truncated(t *testing.T) {

	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	width, height := ExtractImageDimensions(data, "png")

	assert.Zero(t, width, "expected 0 width for truncated PNG")
	assert.Zero(t, height, "expected 0 height for truncated PNG")
}

func TestExtractImageDimensions_UnknownFormat(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03}

	width, height := ExtractImageDimensions(data, "bmp")

	assert.Zero(t, width, "expected 0 width for unknown format")
	assert.Zero(t, height, "expected 0 height for unknown format")
}

func buildMinimalJPEG(width, height int) []byte {
	var buf bytes.Buffer

	buf.Write([]byte{0xFF, 0xD8})

	buf.Write([]byte{0xFF, 0xC0})

	buf.Write([]byte{0x00, 0x11})

	buf.WriteByte(0x08)

	hBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(hBytes, uint16(height))
	buf.Write(hBytes)

	wBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(wBytes, uint16(width))
	buf.Write(wBytes)

	buf.WriteByte(0x03)

	buf.Write([]byte{0x01, 0x22, 0x00})
	buf.Write([]byte{0x02, 0x11, 0x01})
	buf.Write([]byte{0x03, 0x11, 0x01})
	return buf.Bytes()
}

func buildMinimalPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func buildMinimalPNGWithAlpha() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	img.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 128})
	img.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 0})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 0, A: 64})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestWriteObjects_JPEG_ProducesValidPDFOutput(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	jpegData := buildMinimalJPEG(320, 240)
	embedder.RegisterImage("photo.jpg", jpegData, "jpeg", 320, 240)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Subtype /Image")
	assert.Contains(t, output, "/Filter /DCTDecode", "expected /Filter /DCTDecode for JPEG image")
	assert.Contains(t, output, "/Width 320")
	assert.Contains(t, output, "/Height 240")
	assert.Contains(t, entries, "/Im1")
}

func TestWriteObjects_PNG_ProducesValidPDFOutput(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	pngData := buildMinimalPNG()
	embedder.RegisterImage("icon.png", pngData, "png", 1, 1)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	output := string(writer.Bytes())
	assert.Contains(t, output, "/Subtype /Image")
	assert.Contains(t, output, "/Filter /FlateDecode", "expected /Filter /FlateDecode for PNG image")
	assert.Contains(t, entries, "/Im1")
}

func TestWriteObjects_PNGWithAlpha_WritesSMask(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	pngData := buildMinimalPNGWithAlpha()
	embedder.RegisterImage("alpha.png", pngData, "png", 2, 2)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	_, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	output := string(writer.Bytes())
	assert.Contains(t, output, "/SMask", "expected /SMask reference for PNG with alpha channel")
	assert.Contains(t, output, "/ColorSpace /DeviceGray", "expected /ColorSpace /DeviceGray for SMask")
}

func TestWriteObjects_MultipleImages_SortedOrder(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	jpegData := buildMinimalJPEG(100, 100)
	pngData := buildMinimalPNG()

	embedder.RegisterImage("b.jpg", jpegData, "jpeg", 100, 100)
	embedder.RegisterImage("a.png", pngData, "png", 1, 1)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	assert.Contains(t, entries, "/Im1")
	assert.Contains(t, entries, "/Im2")
}

func TestWriteObjects_InvalidPNG_WritesPlaceholder(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()

	embedder.RegisterImage("broken.png", []byte{0x89, 0x50, 0x4E, 0x47, 0x00}, "png", 100, 100)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	output := string(writer.Bytes())

	assert.Contains(t, output, "/Subtype /Image", "expected placeholder /Subtype /Image for broken PNG")
	assert.Contains(t, entries, "/Im1")
}

func TestWriteObjects_Empty_ReturnsEmptyEntries(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	writer := &PdfDocumentWriter{}
	writer.WriteHeader()
	entries, writeError := embedder.WriteObjects(writer)
	require.NoError(t, writeError)

	assert.Empty(t, entries, "expected empty entries for no images")
}

func TestExtractJPEGDimensions_InvalidSignature(t *testing.T) {
	t.Parallel()

	w, h := ExtractImageDimensions([]byte{0x00, 0x00}, "jpeg")
	assert.Zero(t, w, "expected 0 width for invalid JPEG signature")
	assert.Zero(t, h, "expected 0 height for invalid JPEG signature")
}

func TestExtractJPEGDimensions_FFPadding(t *testing.T) {
	t.Parallel()

	data := []byte{
		0xFF, 0xD8,
		0xFF, 0xFF,
		0xFF, 0xC0,
		0x00, 0x11,
		0x08,
		0x00, 0x64,
		0x00, 0xC8,
		0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01,
	}

	w, h := ExtractImageDimensions(data, "jpeg")
	assert.Equal(t, 200, w)
	assert.Equal(t, 100, h)
}

func TestExtractJPEGDimensions_NonMarkerByte(t *testing.T) {
	t.Parallel()

	data := []byte{
		0xFF, 0xD8,
		0x00,
		0xFF, 0xC0,
		0x00, 0x11, 0x08,
		0x00, 0x32,
		0x00, 0x96,
		0x03, 0x01, 0x22, 0x00, 0x02, 0x11, 0x01, 0x03, 0x11, 0x01,
	}

	w, h := ExtractImageDimensions(data, "jpeg")
	assert.Equal(t, 150, w)
	assert.Equal(t, 50, h)
}

func TestExtractJPEGDimensions_TruncatedSOF(t *testing.T) {
	t.Parallel()

	data := []byte{
		0xFF, 0xD8,
		0xFF, 0xC0,
		0x00, 0x03,
		0x08,
	}

	w, h := ExtractImageDimensions(data, "jpeg")
	assert.Zero(t, w, "expected 0 width for truncated SOF")
	assert.Zero(t, h, "expected 0 height for truncated SOF")
}

func TestExtractPNGDimensions_BadSignature(t *testing.T) {
	t.Parallel()

	data := make([]byte, 30)
	data[0] = 0x00

	w, h := ExtractImageDimensions(data, "png")
	assert.Zero(t, w, "expected 0 width for bad PNG signature")
	assert.Zero(t, h, "expected 0 height for bad PNG signature")
}

func TestImageEmbedder_RejectsHugeDimensions(t *testing.T) {
	t.Parallel()

	embedder := NewImageEmbedder()
	embedder.RegisterImage("huge.png", buildMinimalPNG(), "png", 50000, 50000)

	writer := &PdfDocumentWriter{}
	writer.WriteHeader()

	_, err := embedder.WriteObjects(writer)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrImageDimensionsTooLarge)
}
