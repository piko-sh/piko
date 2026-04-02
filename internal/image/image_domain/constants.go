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

import "time"

const (
	// minQuality is the minimum allowed quality value for image transformations.
	minQuality = 1

	// maxQuality is the maximum allowed quality value for image transformations.
	maxQuality = 100

	// defaultMaxImageWidth is the default maximum width for images, in pixels.
	defaultMaxImageWidth = 8192

	// defaultMaxImageHeight is the default maximum image height in pixels.
	defaultMaxImageHeight = 8192

	// defaultMaxImagePixels is the default maximum total pixel count for images.
	// This equals 100 megapixels, which is roughly a 10000x10000 image.
	defaultMaxImagePixels = 100_000_000

	// defaultMaxFileSizeBytes is the default maximum file size of 50 megabytes.
	defaultMaxFileSizeBytes = 50 * bytesPerMB

	// defaultTransformTimeout is the default time limit for image transformations.
	defaultTransformTimeout = 30 * time.Second

	// hashPrefixLength is the number of characters used from a hash for sidecar keys.
	hashPrefixLength = 16

	// defaultPlaceholderWidth is the default width in pixels for placeholder images.
	defaultPlaceholderWidth = 20

	// defaultPlaceholderQuality is the default quality setting for placeholder images.
	defaultPlaceholderQuality = 10

	// defaultPlaceholderBlur is the default blur sigma value for placeholder images.
	defaultPlaceholderBlur = 5.0

	// bytesPerKB is the number of bytes per kilobyte.
	bytesPerKB = 1024

	// bytesPerMB is the number of bytes in a megabyte (1024 x 1024).
	bytesPerMB = 1024 * bytesPerKB

	// screenSizeSmall is the breakpoint width for small screens, set at 640 pixels.
	screenSizeSmall = 640

	// screenSizeMedium is the breakpoint for medium screens at 768 pixels.
	screenSizeMedium = 768

	// screenSizeLarge is the breakpoint for large screens (1024px).
	screenSizeLarge = 1024

	// screenSizeXLarge is the breakpoint for extra-large screens (1280px).
	screenSizeXLarge = 1280

	// screenSize2XLarge is the breakpoint for 2x extra-large screens at 1536px.
	screenSize2XLarge = 1536

	// responsiveWidthXSmall is the responsive width for extra-small screens (320px).
	responsiveWidthXSmall = 320

	// responsiveWidthSmall is the width for small screens (640 pixels).
	responsiveWidthSmall = 640

	// responsiveWidthMedium is the responsive width for medium screens (768px).
	responsiveWidthMedium = 768

	// responsiveWidthLarge is the responsive width for large screens (1024px).
	responsiveWidthLarge = 1024

	// responsiveWidthXLarge is the responsive width for extra-large screens.
	responsiveWidthXLarge = 1280

	// responsiveWidth2XLarge is the responsive width for 2x extra-large screens
	// (1920px).
	responsiveWidth2XLarge = 1920

	// defaultDensityMultiplier is the default pixel density multiplier value of 1.0,
	// used when parsing fails or the provided density is invalid.
	defaultDensityMultiplier = 1.0

	// metricAttributeProvider is the metric attribute key for the provider name.
	metricAttributeProvider = "provider"

	// variantSizeThumb100 is the dimension for 100px thumbnail variants.
	variantSizeThumb100 = 100

	// variantSizeThumb200 is the dimension for 200px thumbnail variants.
	variantSizeThumb200 = 200

	// variantSizeThumb400 is the dimension for 400px thumbnail variants.
	variantSizeThumb400 = 400

	// variantSizePreview800 is the width in pixels for 800px preview variants.
	variantSizePreview800 = 800

	// variantSizeLQIP is the dimension for low-quality image placeholder variants.
	variantSizeLQIP = 20

	// variantQualityLow is quality 75 for smaller thumbnails.
	variantQualityLow = 75

	// variantQualityMedium is quality 80 for medium thumbnails.
	variantQualityMedium = 80

	// variantQualityHigh is the quality level 85 for larger preview images.
	variantQualityHigh = 85

	// variantQualityLQIP is quality 20 for LQIP placeholders.
	variantQualityLQIP = 20

	// variantFormatWebP is the default output format for image variants.
	variantFormatWebP = "webp"

	// variantFitCover scales to cover the entire area.
	variantFitCover = "cover"

	// variantFitContain scales the image to fit within the given area.
	variantFitContain = "contain"
)
