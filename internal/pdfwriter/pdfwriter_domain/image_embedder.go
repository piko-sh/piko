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

// Manages image XObject resources for PDF rendering. Follows the same
// register-track-write lifecycle as FontEmbedder: images are registered
// during painting, then written as PDF objects at the end.
//
// JPEG images are embedded using DCTDecode (raw passthrough, no re-encoding).
// PNG images are decoded to raw RGB pixels, compressed with FlateDecode,
// and if an alpha channel is present, a separate SMask XObject is written.

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image/png"
	"slices"
	"strings"

	"piko.sh/piko/wdk/safeconv"
)

// Image format byte signatures and constants.
const (
	// maxAlpha holds the maximum alpha channel value for a fully opaque pixel.
	maxAlpha = 255

	// jpegMarkerPrefix holds the byte that precedes every JPEG marker.
	jpegMarkerPrefix = 0xFF

	// jpegSOIMarker holds the Start of Image marker byte.
	jpegSOIMarker = 0xD8

	// jpegSOFRangeStart holds the first byte in the Start of Frame marker range.
	jpegSOFRangeStart = 0xC0

	// jpegSOFRangeEnd holds the last byte in the Start of Frame marker range.
	jpegSOFRangeEnd = 0xCF

	// jpegDHTMarker holds the Define Huffman Table marker byte, excluded from SOF detection.
	jpegDHTMarker = 0xC4

	// jpegJPGMarker holds the JPG extension marker byte, excluded from SOF detection.
	jpegJPGMarker = 0xC8

	// jpegDACMarker holds the Define Arithmetic Coding
	// marker byte, excluded from SOF detection.
	jpegDACMarker = 0xCC

	// jpegSOFDataOffset holds the byte offset within a
	// SOF segment where dimension data begins.
	jpegSOFDataOffset = 7

	// pngMinHeaderSize holds the minimum number of bytes required to read PNG dimensions.
	pngMinHeaderSize = 24

	// pngSignature0 holds the first byte of the PNG file signature.
	pngSignature0 = 0x89

	// pngSignature1 holds the second byte of the PNG file signature.
	pngSignature1 = 0x50

	// pngSignature2 holds the third byte of the PNG file signature.
	pngSignature2 = 0x4E

	// pngSignature3 holds the fourth byte of the PNG file signature.
	pngSignature3 = 0x47

	// rgbChannelCount holds the number of colour channels in an RGB image.
	rgbChannelCount = 3
)

// embeddedImageState holds the data for a single registered image.
type embeddedImageState struct {
	// format holds the image encoding type, either "jpeg" or "png".
	format string

	// data holds the raw image bytes.
	data []byte

	// pixelWidth holds the image width in pixels.
	pixelWidth int

	// pixelHeight holds the image height in pixels.
	pixelHeight int
}

// ImageEmbedder tracks image XObjects needed for PDF rendering and
// writes them as PDF objects.
type ImageEmbedder struct {
	// images maps resource names (Im1, Im2, ...) to their image data.
	images map[string]*embeddedImageState

	// sourceToName deduplicates registrations by mapping source paths
	// to their existing resource names.
	sourceToName map[string]string

	// nextIndex is the counter for generating resource names.
	nextIndex int
}

// NewImageEmbedder creates a new image embedder.
//
// Returns *ImageEmbedder ready to accept image registrations.
func NewImageEmbedder() *ImageEmbedder {
	return &ImageEmbedder{
		images:       make(map[string]*embeddedImageState),
		sourceToName: make(map[string]string),
	}
}

// RegisterImage registers an image for embedding and returns its
// resource name (e.g. "Im1").
//
// If the same source has already been registered, the existing name is
// returned without creating a duplicate entry.
//
// Takes source (string) which is the image source path for deduplication.
// Takes data ([]byte) which is the raw image bytes.
// Takes format (string) which is "jpeg" or "png".
// Takes pixelWidth (int) which is the image width in pixels.
// Takes pixelHeight (int) which is the image height in pixels.
//
// Returns string which is the resource name for use in PaintXObject.
func (e *ImageEmbedder) RegisterImage(source string, data []byte, format string, pixelWidth, pixelHeight int) string {
	if name, exists := e.sourceToName[source]; exists {
		return name
	}

	e.nextIndex++
	name := fmt.Sprintf("Im%d", e.nextIndex)
	e.images[name] = &embeddedImageState{
		data:        data,
		format:      format,
		pixelWidth:  pixelWidth,
		pixelHeight: pixelHeight,
	}
	e.sourceToName[source] = name
	return name
}

// HasImages reports whether any images have been registered.
//
// Returns bool which is true if at least one image exists.
func (e *ImageEmbedder) HasImages() bool {
	return len(e.images) > 0
}

// WriteObjects writes all registered image XObjects as PDF objects
// and returns the resource entries string for inclusion in the page
// /Resources /XObject dictionary.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
//
// Returns string with entries like " /Im1 5 0 R /Im2 6 0 R", and an
// error if any PNG image fails to compress.
func (e *ImageEmbedder) WriteObjects(writer *PdfDocumentWriter) (string, error) {
	sortedNames := make([]string, 0, len(e.images))
	for name := range e.images {
		sortedNames = append(sortedNames, name)
	}
	slices.Sort(sortedNames)

	var entries strings.Builder
	for _, name := range sortedNames {
		state := e.images[name]
		switch state.format {
		case "jpeg":
			objectNumber := e.writeJPEGXObject(writer, state)
			fmt.Fprintf(&entries, " /%s %s", name, FormatReference(objectNumber))
		case "png":
			objectNumber, compressError := e.writePNGXObject(writer, state)
			if compressError != nil {
				return "", fmt.Errorf("pdfwriter: compressing PNG image %s: %w", name, compressError)
			}
			fmt.Fprintf(&entries, " /%s %s", name, FormatReference(objectNumber))
		}
	}
	return entries.String(), nil
}

// writeJPEGXObject writes a JPEG image as an XObject with DCTDecode.
// The raw JPEG bytes are passed through directly (no re-encoding).
//
// Takes writer (*PdfDocumentWriter) which receives the PDF object.
// Takes state (*embeddedImageState) which holds the JPEG data and dimensions.
//
// Returns int which is the object number of the written XObject.
func (*ImageEmbedder) writeJPEGXObject(writer *PdfDocumentWriter, state *embeddedImageState) int {
	objectNumber := writer.AllocateObject()
	dictionary := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>",
		state.pixelWidth, state.pixelHeight, len(state.data))

	writer.WriteRawStreamObject(objectNumber, dictionary, state.data)
	return objectNumber
}

// writePNGXObject decodes a PNG image, extracts raw RGB pixels,
// compresses with zlib, and writes as an XObject with FlateDecode.
//
// If the PNG has an alpha channel, a separate SMask XObject is
// written for transparency.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
// Takes state (*embeddedImageState) which holds the PNG data and dimensions.
//
// Returns int which is the object number of the written
// XObject.
// Returns error which is non-nil if zlib compression
// fails.
func (*ImageEmbedder) writePNGXObject(writer *PdfDocumentWriter, state *embeddedImageState) (int, error) {
	img, decodeError := png.Decode(bytes.NewReader(state.data))
	if decodeError != nil {
		objectNumber := writer.AllocateObject()
		writer.WriteObject(objectNumber, "<< /Type /XObject /Subtype /Image /Width 1 /Height 1 /ColorSpace /DeviceRGB /BitsPerComponent 8 >>")
		return objectNumber, nil
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	rgbPixels := make([]byte, 0, width*height*rgbChannelCount)
	alphaPixels := make([]byte, 0, width*height)
	hasAlpha := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()

			rgbPixels = append(rgbPixels, safeconv.Uint32ToByte(r>>8), safeconv.Uint32ToByte(g>>8), safeconv.Uint32ToByte(b>>8))
			alphaByte := safeconv.Uint32ToByte(a >> 8)
			alphaPixels = append(alphaPixels, alphaByte)
			if alphaByte != maxAlpha {
				hasAlpha = true
			}
		}
	}

	var rgbCompressed bytes.Buffer
	zlibWriter := zlib.NewWriter(&rgbCompressed)
	if rgbError := writeAndCloseZlib(zlibWriter, rgbPixels); rgbError != nil {
		return 0, fmt.Errorf("compressing RGB pixels: %w", rgbError)
	}

	var smaskRef string
	if hasAlpha {
		smaskNumber := writer.AllocateObject()
		var alphaCompressed bytes.Buffer
		alphaZlibWriter := zlib.NewWriter(&alphaCompressed)
		if alphaError := writeAndCloseZlib(alphaZlibWriter, alphaPixels); alphaError != nil {
			return 0, fmt.Errorf("compressing alpha channel: %w", alphaError)
		}

		smaskDict := fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>",
			width, height, alphaCompressed.Len())
		writer.WriteRawStreamObject(smaskNumber, smaskDict, alphaCompressed.Bytes())
		smaskRef = fmt.Sprintf(" /SMask %s", FormatReference(smaskNumber))
	}

	objectNumber := writer.AllocateObject()
	dictionary := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d%s >>",
		width, height, rgbCompressed.Len(), smaskRef)
	writer.WriteRawStreamObject(objectNumber, dictionary, rgbCompressed.Bytes())

	return objectNumber, nil
}

// writeAndCloseZlib writes data to the zlib writer and closes it.
//
// Takes zw (*zlib.Writer) which is the compression writer to write to.
// Takes data ([]byte) which is the raw bytes to compress.
//
// Returns error which is non-nil if writing or closing the zlib writer fails.
func writeAndCloseZlib(zw *zlib.Writer, data []byte) error {
	if _, err := zw.Write(data); err != nil {
		return err
	}
	return zw.Close()
}

// ExtractImageDimensions parses the dimensions from raw image data
// without fully decoding the pixel data. For JPEG it reads the SOF
// marker; for PNG it reads the IHDR chunk.
//
// Takes data ([]byte) which is the raw image bytes.
// Takes format (string) which is "jpeg" or "png".
//
// Returns (width, height int) in pixels, or (0, 0) on failure.
func ExtractImageDimensions(data []byte, format string) (width, height int) {
	switch format {
	case "jpeg":
		return extractJPEGDimensions(data)
	case "png":
		return extractPNGDimensions(data)
	default:
		return 0, 0
	}
}

// extractJPEGDimensions reads the SOF (Start of Frame) marker to
// get image dimensions without decoding the full image.
//
// Takes data ([]byte) which is the raw JPEG bytes.
//
// Returns (width, height int) in pixels, or (0, 0) if the data is
// invalid or no SOF marker is found.
func extractJPEGDimensions(data []byte) (width, height int) {
	if len(data) < 2 || data[0] != jpegMarkerPrefix || data[1] != jpegSOIMarker {
		return 0, 0
	}

	offset := 2
	for offset+1 < len(data) {
		if data[offset] != jpegMarkerPrefix {
			offset++
			continue
		}

		marker := data[offset+1]
		offset += 2

		if marker == jpegMarkerPrefix {
			continue
		}

		isSof := marker >= jpegSOFRangeStart && marker <= jpegSOFRangeEnd &&
			marker != jpegDHTMarker && marker != jpegJPGMarker && marker != jpegDACMarker

		if isSof {
			if offset+jpegSOFDataOffset > len(data) {
				return 0, 0
			}

			height = int(binary.BigEndian.Uint16(data[offset+3 : offset+5]))
			width = int(binary.BigEndian.Uint16(data[offset+5 : offset+7]))
			return width, height
		}

		if offset+1 >= len(data) {
			return 0, 0
		}
		segmentLength := int(binary.BigEndian.Uint16(data[offset : offset+2]))
		offset += segmentLength
	}

	return 0, 0
}

// extractPNGDimensions reads the IHDR chunk to get image dimensions.
//
// Takes data ([]byte) which is the raw PNG bytes.
//
// Returns (width, height int) in pixels, or (0, 0) if the data is
// too short or the PNG signature is invalid.
func extractPNGDimensions(data []byte) (width, height int) {
	if len(data) < pngMinHeaderSize {
		return 0, 0
	}

	if data[0] != pngSignature0 || data[1] != pngSignature1 || data[2] != pngSignature2 || data[rgbChannelCount] != pngSignature3 {
		return 0, 0
	}

	width = int(binary.BigEndian.Uint32(data[16:20]))
	height = int(binary.BigEndian.Uint32(data[20:24]))
	return width, height
}
