//go:build !vips

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

package image_provider_vips

import (
	"context"
	"errors"
	"io"

	"piko.sh/piko/internal/image/image_dto"
	"piko.sh/piko/wdk/media"
)

// Config holds configuration options for the vips provider.
// When built without the "vips" tag, NewProvider returns an error.
type Config struct {
	// VipsConfig is unused in the stub; present for API compatibility.
	VipsConfig any

	// ImageServiceConfig holds security settings for input validation and resource limits.
	media.ImageServiceConfig

	// ConcurrencyLevel limits how many image tasks can run at the same time.
	ConcurrencyLevel int
}

// Provider is a stub that satisfies the ImageTransformerPort interface
// when built without the "vips" tag. All methods return errors.
type Provider struct{}

var _ media.ImageTransformerPort = (*Provider)(nil)

// errNotAvailable is returned by all stub methods.
var errNotAvailable = errors.New(
	"image_provider_vips: not available (built without -tags vips; install libvips and rebuild with -tags vips)",
)

// Close is a no-op in the stub.
//
// Returns error which is always nil.
func (*Provider) Close() error {
	return nil
}

// Transform returns an error indicating the vips provider is not available.
//
// Returns string which is always empty.
// Returns error when built without the vips build tag.
func (*Provider) Transform(
	_ context.Context, _ io.Reader, _ io.Writer, _ image_dto.TransformationSpec,
) (string, error) {
	return "", errNotAvailable
}

// GetSupportedFormats returns nil in the stub.
//
// Returns []string which is always nil.
func (*Provider) GetSupportedFormats() []string {
	return nil
}

// GetSupportedModifiers returns nil in the stub.
//
// Returns []string which is always nil.
func (*Provider) GetSupportedModifiers() []string {
	return nil
}

// GetDimensions returns an error indicating the vips provider
// is not available.
//
// Returns int which is the width, always 0.
// Returns int which is the height, always 0.
// Returns error when built without the vips build tag.
func (*Provider) GetDimensions(_ context.Context, _ io.Reader) (width int, height int, err error) {
	return 0, 0, errNotAvailable
}

// NewProvider returns an error when built without the "vips" tag.
//
// Takes config (Config) which is ignored in the stub.
//
// Returns *Provider which is nil.
// Returns error when built without the vips build tag.
func NewProvider(_ Config) (*Provider, error) {
	return nil, errNotAvailable
}
