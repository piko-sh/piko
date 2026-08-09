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

//go:build vips

package image_provider_vips

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/media"
)

const (
	sourceResolutionPixelsPerMetre = 11811
	webResolutionPixelsPerMetre    = 2835
	bmpHeaderBytes                 = 14 + 40
	bmpBytesPerPixel               = 3
	testImageEdge                  = 8
)

func TestProvider_TransformNormalisesOutputResolution(t *testing.T) {
	provider, err := NewProvider(Config{ImageServiceConfig: media.DefaultImageServiceConfig()})
	require.NoError(t, err, "libvips must be installed to run this test")
	t.Cleanup(func() { assert.NoError(t, provider.Close()) })

	source := synthesiseBMP(t, testImageEdge, testImageEdge, sourceResolutionPixelsPerMetre)

	requireBMPResolution(t, source, sourceResolutionPixelsPerMetre)

	t.Run("png output carries the web resolution, not the source resolution", func(t *testing.T) {
		var output bytes.Buffer
		spec := media.DefaultTransformationSpec()
		spec.Format = "png"
		spec.Width = 4

		mimeType, transformErr := provider.Transform(
			context.Background(), bytes.NewReader(source), &output, spec)
		require.NoError(t, transformErr)
		assert.Equal(t, "image/png", mimeType)

		resolution, found := pngPixelsPerMetre(output.Bytes())
		require.True(t, found, "libvips writes a pHYs chunk, so the resolution is observable")
		assert.Equal(t, uint32(webResolutionPixelsPerMetre), resolution,
			"a print-resolution source must not stamp print resolution onto a web derivative")
	})

	t.Run("other formats still encode", func(t *testing.T) {
		for _, format := range []string{"jpeg", "webp"} {
			t.Run(format, func(t *testing.T) {
				var output bytes.Buffer
				spec := media.DefaultTransformationSpec()
				spec.Format = format
				spec.Width = 4

				_, transformErr := provider.Transform(
					context.Background(), bytes.NewReader(source), &output, spec)
				require.NoError(t, transformErr)
				assert.NotEmpty(t, output.Bytes())
			})
		}
	})
}

func TestOutputResolutionConstant_Is72DPI(t *testing.T) {
	t.Parallel()

	perMetre := outputResolutionPixelsPerMillimetre * 1000

	assert.InDelta(t, float64(webResolutionPixelsPerMetre), math.Round(perMetre), 1,
		"the constant is expressed per millimetre because that is the unit libvips stores")
}

func synthesiseBMP(t *testing.T, width, height int, pixelsPerMetre int32) []byte {
	t.Helper()

	rowStride := (width*bmpBytesPerPixel + 3) & ^3
	pixelBytes := rowStride * height
	buffer := make([]byte, bmpHeaderBytes+pixelBytes)

	copy(buffer, "BM")
	binary.LittleEndian.PutUint32(buffer[2:], uint32(len(buffer)))
	binary.LittleEndian.PutUint32(buffer[10:], bmpHeaderBytes)

	binary.LittleEndian.PutUint32(buffer[14:], 40)
	binary.LittleEndian.PutUint32(buffer[18:], uint32(width))
	binary.LittleEndian.PutUint32(buffer[22:], uint32(height))
	binary.LittleEndian.PutUint16(buffer[26:], 1)
	binary.LittleEndian.PutUint16(buffer[28:], 24)
	binary.LittleEndian.PutUint32(buffer[34:], uint32(pixelBytes))
	binary.LittleEndian.PutUint32(buffer[38:], uint32(pixelsPerMetre))
	binary.LittleEndian.PutUint32(buffer[42:], uint32(pixelsPerMetre))

	for index := bmpHeaderBytes; index < len(buffer); index++ {
		buffer[index] = byte(index % 251)
	}

	return buffer
}

func requireBMPResolution(t *testing.T, source []byte, expected int32) {
	t.Helper()

	require.Greater(t, len(source), 46)
	assert.Equal(t, uint32(expected), binary.LittleEndian.Uint32(source[38:]))
	assert.Equal(t, uint32(expected), binary.LittleEndian.Uint32(source[42:]))
}

func pngPixelsPerMetre(encoded []byte) (pixelsPerMetre uint32, found bool) {
	const (
		signatureBytes = 8
		lengthBytes    = 4
		typeBytes      = 4
		crcBytes       = 4
	)

	offset := signatureBytes
	for offset+lengthBytes+typeBytes+crcBytes <= len(encoded) {
		length := int(binary.BigEndian.Uint32(encoded[offset:]))
		if length < 0 {
			return 0, false
		}
		chunkType := string(encoded[offset+lengthBytes : offset+lengthBytes+typeBytes])
		dataStart := offset + lengthBytes + typeBytes

		if chunkType == "pHYs" {
			if length < 8 || dataStart+8 > len(encoded) {
				return 0, false
			}
			return binary.BigEndian.Uint32(encoded[dataStart:]), true
		}

		offset = dataStart + length + crcBytes
	}
	return 0, false
}
