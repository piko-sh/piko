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
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	errTransformerNil = errors.New("transformer cannot be nil")

	errNoProviders = errors.New("at least one provider must be registered")

	errDefaultProviderNotSet = errors.New("default provider not set")

	errNoTransformers = errors.New("at least one image transformer must be provided")

	errDefaultProviderEmpty = errors.New("default image provider cannot be empty")

	errMaxWidthNegative = errors.New("max image width cannot be negative")

	errMaxHeightNegative = errors.New("max image height cannot be negative")

	errMaxPixelsNegative = errors.New("max image pixels cannot be negative")

	errMaxFileSizeNegative = errors.New("max file size cannot be negative")

	errTimeoutNegative = errors.New("transform timeout cannot be negative")

	errQualityOutOfRange = errors.New("quality must be between 1 and 100")

	errVariantNameEmpty = errors.New("variant name cannot be empty")
)
