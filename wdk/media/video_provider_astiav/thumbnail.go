//go:build ffmpeg

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

package video_provider_astiav

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/asticode/go-astiav"
	"golang.org/x/image/webp"
)

var _ = webp.Decode

const (
	// rgbChannelCount is the number of colour channels in an
	// RGB pixel.
	rgbChannelCount = 3

	// rgbaChannelCount is the number of colour channels in an
	// RGBA pixel.
	rgbaChannelCount = 4

	// alphaOpaque is the fully opaque alpha channel value.
	alphaOpaque = 255

	// defaultQuality is the default JPEG encoding quality
	// (1-100).
	defaultQuality = 85

	// rgbaAlphaOffset is the byte offset of the alpha channel within an RGBA
	// pixel.
	rgbaAlphaOffset = 3
)

// encodeRGBFrameToImage converts an RGB24 FFmpeg frame to an encoded image.
//
// Takes frame (*astiav.Frame) which contains the RGB24 pixel data.
// Takes format (string) which specifies the output format (jpeg, png, webp).
// Takes quality (int) which specifies the quality for lossy formats (1-100).
//
// Returns []byte which contains the encoded image data.
// Returns error when encoding fails.
func encodeRGBFrameToImage(frame *astiav.Frame, format string, quality int) ([]byte, error) {
	img, err := rgbFrameToRGBAImage(frame)
	if err != nil {
		return nil, err
	}

	if quality <= 0 {
		quality = defaultQuality
	}

	return encodeImageFormat(img, format, quality)
}

// rgbFrameToRGBAImage copies RGB24 pixel data from an FFmpeg
// frame into a standard library RGBA image.
//
// Takes frame (*astiav.Frame) which contains the RGB24
// source pixel data.
//
// Returns *image.RGBA which is the converted RGBA image.
// Returns error when frame data cannot be read.
func rgbFrameToRGBAImage(frame *astiav.Frame) (*image.RGBA, error) {
	width := frame.Width()
	height := frame.Height()

	data, err := frame.Data().Bytes(1)
	if err != nil {
		return nil, fmt.Errorf("getting frame data bytes: %w", err)
	}
	if len(data) == 0 {
		return nil, errors.New("frame has no data")
	}

	linesize := frame.Linesize()[0]
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		srcRowStart := y * linesize
		for x := range width {
			srcIndex := srcRowStart + x*rgbChannelCount
			if srcIndex+2 >= len(data) {
				break
			}
			dstIndex := y*img.Stride + x*rgbaChannelCount
			img.Pix[dstIndex+0] = data[srcIndex+0]
			img.Pix[dstIndex+1] = data[srcIndex+1]
			img.Pix[dstIndex+2] = data[srcIndex+2]
			img.Pix[dstIndex+rgbaAlphaOffset] = alphaOpaque
		}
	}

	return img, nil
}

// encodeImageFormat encodes an RGBA image into the specified
// format (png or jpeg).
//
// Takes img (*image.RGBA) which is the source image to
// encode.
// Takes format (string) which specifies the output format.
// Takes quality (int) which sets the JPEG quality (1-100).
//
// Returns []byte which contains the encoded image data.
// Returns error when encoding fails.
func encodeImageFormat(img *image.RGBA, format string, quality int) ([]byte, error) {
	var buffer bytes.Buffer

	switch strings.ToLower(format) {
	case "png":
		if err := png.Encode(&buffer, img); err != nil {
			return nil, fmt.Errorf("encoding PNG: %w", err)
		}
	default:

		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encoding JPEG: %w", err)
		}
	}

	return buffer.Bytes(), nil
}
