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

package image_domain

import "errors"

var (
	// errProviderNameEmpty is returned when an image provider is registered
	// with an empty name.
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	// errTransformerNil is returned when a nil transformer is provided during
	// registration.
	errTransformerNil = errors.New("transformer cannot be nil")

	// errNoProviders is returned when the image service is started without any
	// registered providers.
	errNoProviders = errors.New("at least one provider must be registered")

	// errDefaultProviderNotSet is returned when an operation requires a default
	// provider but none has been configured.
	errDefaultProviderNotSet = errors.New("default provider not set")

	// errNoTransformers is returned when the image service is started without
	// any registered transformers.
	errNoTransformers = errors.New("at least one image transformer must be provided")

	// errDefaultProviderEmpty is returned when the default image provider name
	// is set to an empty string.
	errDefaultProviderEmpty = errors.New("default image provider cannot be empty")

	// errMaxWidthNegative is returned when the configured maximum image width
	// is negative.
	errMaxWidthNegative = errors.New("max image width cannot be negative")

	// errMaxHeightNegative is returned when the configured maximum image
	// height is negative.
	errMaxHeightNegative = errors.New("max image height cannot be negative")

	// errMaxPixelsNegative is returned when the configured maximum pixel count
	// is negative.
	errMaxPixelsNegative = errors.New("max image pixels cannot be negative")

	// errMaxFileSizeNegative is returned when the configured maximum file size
	// is negative.
	errMaxFileSizeNegative = errors.New("max file size cannot be negative")

	// errTimeoutNegative is returned when the configured transform timeout is
	// negative.
	errTimeoutNegative = errors.New("transform timeout cannot be negative")

	// errQualityOutOfRange is returned when the image quality setting is
	// outside the valid range of 1 to 100.
	errQualityOutOfRange = errors.New("quality must be between 1 and 100")

	// errVariantNameEmpty is returned when an image variant is created with an
	// empty name.
	errVariantNameEmpty = errors.New("variant name cannot be empty")
)
