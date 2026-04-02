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

package pdfwriter_adapters

// Implements ImageDataPort by fetching image bytes from the registry service.
// Follows the same pattern as the email asset resolver at
// internal/email/email_adapters/asset_resolver/driven_registry.go.

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
	"piko.sh/piko/internal/registry/registry_domain"
)

const (
	// minimumMagicByteLength holds the minimum number of bytes needed to
	// detect an image format from its magic bytes.
	minimumMagicByteLength = 4

	// maxImageSize is the upper bound for image data reads to guard against
	// excessively large images (100 MiB).
	maxImageSize = 100 << 20
)

var (
	errImageArtefactNotFound = errors.New("artefact not found for image")

	errImageNoVariants = errors.New("no variants available for image")

	errImageUnsupportedFormat = errors.New("unsupported image format")
)

var (
	jpegMagicBytes = []byte{0xFF, 0xD8, 0xFF}

	pngMagicBytes = []byte{0x89, 0x50, 0x4E, 0x47}
)

var (
	_ pdfwriter_domain.ImageDataPort = (*RegistryImageDataAdapter)(nil)

	_ pdfwriter_domain.ImageDataPort = (*DataURIImageDataAdapter)(nil)

	_ pdfwriter_domain.ImageDataPort = (*MockImageDataAdapter)(nil)
)

// RegistryImageDataAdapter implements ImageDataPort by delegating to the
// registry service to fetch image bytes.
type RegistryImageDataAdapter struct {
	// registryService holds the registry service used to fetch artefacts
	// and variant data.
	registryService registry_domain.RegistryService
}

// NewRegistryImageDataAdapter creates an adapter that fetches image data
// from the registry service.
//
// Takes registryService (registry_domain.RegistryService) which provides
// artefact and variant data access.
//
// Returns *RegistryImageDataAdapter which implements ImageDataPort.
func NewRegistryImageDataAdapter(registryService registry_domain.RegistryService) *RegistryImageDataAdapter {
	return &RegistryImageDataAdapter{
		registryService: registryService,
	}
}

// GetImageData fetches the raw image bytes for the given source path
// from the registry. It looks up the artefact, selects the source
// variant, and returns the bytes with the detected format.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes source (string) which is the image source path.
//
// Returns []byte which contains the raw image data.
// Returns string which is the detected format ("jpeg" or "png").
// Returns error when the image cannot be fetched.
func (a *RegistryImageDataAdapter) GetImageData(ctx context.Context, source string) ([]byte, string, error) {
	artefactID := source
	if trimmed, found := strings.CutPrefix(artefactID, "/_piko/assets/"); found {
		artefactID = trimmed
	}

	artefact, err := a.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		return nil, "", fmt.Errorf("fetching artefact for image '%s': %w", source, err)
	}
	if artefact == nil {
		return nil, "", fmt.Errorf("image %q: %w", source, errImageArtefactNotFound)
	}

	var selectedVariant *struct {
		index int
	}
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == "source" {
			selectedVariant = &struct{ index int }{i}
			break
		}
	}

	if selectedVariant == nil && len(artefact.ActualVariants) > 0 {
		selectedVariant = &struct{ index int }{0}
	}
	if selectedVariant == nil {
		return nil, "", fmt.Errorf("image %q: %w", source, errImageNoVariants)
	}

	variant := &artefact.ActualVariants[selectedVariant.index]
	dataReader, err := a.registryService.GetVariantData(ctx, variant)
	if err != nil {
		return nil, "", fmt.Errorf("fetching variant data for image '%s': %w", source, err)
	}
	defer func() { _ = dataReader.Close() }()

	data, err := io.ReadAll(io.LimitReader(dataReader, maxImageSize))
	if err != nil {
		return nil, "", fmt.Errorf("reading image data for '%s': %w", source, err)
	}

	format := detectImageFormat(data)
	if format == "" {
		return nil, "", fmt.Errorf("image %q: %w", source, errImageUnsupportedFormat)
	}

	return data, format, nil
}

// detectImageFormat identifies the image format from magic bytes.
//
// Takes data ([]byte) which is the raw image bytes.
//
// Returns "jpeg", "png", or "" if unrecognised.
func detectImageFormat(data []byte) string {
	if len(data) < minimumMagicByteLength {
		return ""
	}

	if bytes.HasPrefix(data, jpegMagicBytes) {
		return "jpeg"
	}

	if bytes.HasPrefix(data, pngMagicBytes) {
		return "png"
	}

	return ""
}

// DataURIImageDataAdapter implements ImageDataPort by decoding inline
// data URIs (e.g. "data:image/png;base64,...").
//
// Non-data-URI sources are delegated to an optional fallback; if no
// fallback is configured the source is silently skipped.
type DataURIImageDataAdapter struct {
	// fallback holds the optional delegate adapter consulted for
	// non-data-URI sources.
	fallback pdfwriter_domain.ImageDataPort
}

// NewDataURIImageDataAdapter creates an adapter that decodes data URI
// image sources. An optional fallback handles non-data-URI sources.
//
// Takes fallback (ImageDataPort) which is consulted for non-data-URI
// sources, or nil to silently skip them.
//
// Returns *DataURIImageDataAdapter which implements ImageDataPort.
func NewDataURIImageDataAdapter(fallback pdfwriter_domain.ImageDataPort) *DataURIImageDataAdapter {
	return &DataURIImageDataAdapter{fallback: fallback}
}

// GetImageData decodes a data URI into raw image bytes and format.
// For non-data-URI sources, delegates to the fallback adapter.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes source (string) which is the image source (data URI or path).
//
// Returns []byte which contains the raw image data.
// Returns string which is the detected format ("jpeg" or "png").
// Returns error when the data URI is malformed or decoding fails.
func (a *DataURIImageDataAdapter) GetImageData(ctx context.Context, source string) ([]byte, string, error) {
	if !strings.HasPrefix(source, "data:") {
		if a.fallback != nil {
			return a.fallback.GetImageData(ctx, source)
		}
		return nil, "", fmt.Errorf("non-data-URI source and no fallback: '%s'", source)
	}

	commaIndex := strings.Index(source, ",")
	if commaIndex < 0 {
		return nil, "", errors.New("malformed data URI: no comma separator")
	}

	header := source[len("data:"):commaIndex]
	encoded := source[commaIndex+1:]

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, "", fmt.Errorf("decoding base64 data URI: %w", err)
	}

	format := detectImageFormat(data)
	if format == "" {
		mediaType := strings.SplitN(header, ";", 2)[0]
		switch mediaType {
		case "image/png":
			format = "png"
		case "image/jpeg", "image/jpg":
			format = "jpeg"
		default:
			return nil, "", fmt.Errorf("unsupported data URI media type: '%s'", mediaType)
		}
	}

	return data, format, nil
}

// MockImageDataAdapter is a test double for ImageDataPort that uses
// a function field for overriding behaviour.
type MockImageDataAdapter struct {
	// GetImageDataFunc is an optional override. When nil, returns an
	// error indicating no image data is available.
	GetImageDataFunc func(ctx context.Context, source string) ([]byte, string, error)
}

// GetImageData delegates to the function field if set, otherwise
// returns an error.
//
// Takes source (string) which is the image source path or data URI.
//
// Returns []byte which contains the raw image data.
// Returns string which is the detected format.
// Returns error when no function override is set or the override fails.
func (m *MockImageDataAdapter) GetImageData(ctx context.Context, source string) ([]byte, string, error) {
	if m.GetImageDataFunc != nil {
		return m.GetImageDataFunc(ctx, source)
	}
	return nil, "", fmt.Errorf("mock: no image data for '%s'", source)
}
