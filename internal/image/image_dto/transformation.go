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

package image_dto

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// FitMode determines how an image is resized when both width and height are
// set.
type FitMode string

const (
	// defaultQuality is the default image compression quality, from 1 to 100.
	defaultQuality = 80

	// FitCover fills dimensions, cropping excess.
	FitCover FitMode = "cover"

	// FitContain fits within dimensions, letterboxing if needed.
	FitContain FitMode = "contain"

	// FitFill stretches to exact dimensions, ignoring aspect ratio.
	FitFill FitMode = "fill"

	// FitInside resizes to be <= dimensions, preserving aspect ratio.
	FitInside FitMode = "inside"

	// FitOutside resizes to be >= dimensions, preserving aspect ratio.
	FitOutside FitMode = "outside"
)

// ResponsiveSpec holds configuration for generating responsive image variants.
// It enables generation of multiple image sizes for srcset attributes.
type ResponsiveSpec struct {
	// Screens maps breakpoint names to pixel widths (e.g., {"sm": 640, "md":
	// 768}). Used to convert the Sizes string into actual pixel widths.
	Screens map[string]int

	// Sizes defines viewport-based widths, such as "100vw sm:50vw md:400px".
	// The parser finds each distinct width for every breakpoint.
	Sizes string

	// Densities is a list of pixel density multipliers (e.g., ["x1", "x2", "x3"]).
	// For each base width, a variant is created at each density.
	Densities []string
}

// PlaceholderSpec holds configuration for generating LQIP (Low Quality Image
// Placeholder). Placeholders are tiny, heavily blurred versions of images used
// during initial page load.
type PlaceholderSpec struct {
	// Enabled indicates whether to generate a placeholder image.
	Enabled bool

	// Width is the target width for the placeholder in pixels (typically very
	// small, e.g., 20px). If 0, a default of 20px will be used.
	Width int

	// Height is the target height in pixels for the placeholder image.
	// If 0, it is worked out from the aspect ratio.
	Height int

	// Quality is the compression quality for the placeholder.
	// Valid range is 1-100; typically very low, like 10.
	Quality int

	// BlurSigma is the blur strength applied to the placeholder.
	// Typically 5.0 or higher; must not be negative.
	BlurSigma float64
}

// ResponsiveVariant represents a single image variant generated for responsive
// images. It contains the transformation result and metadata for building
// srcset attributes.
type ResponsiveVariant struct {
	// Density is the pixel density multiplier (e.g., "x1", "x2", "x3").
	Density string

	// URL is the address or path for this variant, usually set by the caller.
	URL string

	// SrcsetEntry is the full srcset entry for this variant, in the format
	// "url widthw" or "url densityx" (e.g. "/images/hero-800.webp 800w").
	SrcsetEntry string

	// Specification is the full transformation specification used
	// to generate this variant.
	Specification TransformationSpec

	// Width is the width of this variant in pixels.
	Width int

	// Height is the height of this variant in pixels.
	Height int
}

// TransformationSpec holds all desired operations for an image, such as
// resizing, format conversion, and quality adjustments. It is the central
// configuration object for any image processing task.
type TransformationSpec struct {
	// Placeholder holds configuration for generating a low-quality image
	// placeholder (LQIP). If nil, no placeholder is generated.
	Placeholder *PlaceholderSpec

	// Responsive holds configuration for generating multiple responsive image
	// variants. If nil, only a single image is generated.
	Responsive *ResponsiveSpec

	// Modifiers holds provider-specific key/value options not covered by standard
	// fields.
	Modifiers map[string]string

	// Format is the target output format, e.g., "webp", "jpeg", "png", "avif".
	// It will be normalised to lowercase.
	Format string

	// Fit determines how the image is resized when both width and height are
	// set. Defaults to FitContain.
	Fit FitMode

	// AspectRatio specifies the target aspect ratio such as "16:9", "4:3", or
	// "1:1". When set, one dimension may be left out and will be worked out from
	// the ratio.
	AspectRatio string

	// Provider is the name of the image transformer to use, such as "vips" or
	// "imaging". If empty, the domain service default is used.
	Provider string

	// Background is the hex colour (e.g., "#FFFFFF") used for letterboxing in
	// "contain" mode or to fill transparent areas. If empty, defaults to
	// transparent or black depending on format.
	Background string

	// Quality controls the compression level (1-100). Meaning varies by format:
	// JPEG/WebP/AVIF use 80 as a good default; PNG maps to compression effort.
	Quality int

	// Height is the output image height in pixels; 0 keeps the aspect ratio.
	Height int

	// Width is the output image width in pixels. If 0, the aspect ratio is
	// kept based on Height.
	Width int

	// WithoutEnlargement prevents images from being scaled up beyond their
	// original size. When true, images smaller than the target size keep their
	// original dimensions.
	WithoutEnlargement bool
}

// String returns a text form of the transformation settings for use as a
// cache key.
//
// Returns string which contains all transformation fields in a fixed format.
func (s TransformationSpec) String() string {
	format := strings.ToLower(s.Format)
	fit := strings.ToLower(string(s.Fit))
	b := &strings.Builder{}

	_, _ = fmt.Fprintf(b, "p=%s,w=%d,h=%d,fit=%s,ar=%s,we=%t,bg=%s,fmt=%s,q=%d",
		s.Provider, s.Width, s.Height, fit, s.AspectRatio,
		s.WithoutEnlargement, s.Background, format, s.Quality)

	if s.Responsive != nil {
		_, _ = fmt.Fprintf(b, ",resp_sizes=%s", s.Responsive.Sizes)
		if len(s.Responsive.Densities) > 0 {
			densities := strings.Join(s.Responsive.Densities, "+")
			_, _ = fmt.Fprintf(b, ",resp_dens=%s", densities)
		}
	}

	if s.Placeholder != nil && s.Placeholder.Enabled {
		_, _ = fmt.Fprintf(b, ",ph=t,ph_w=%d,ph_h=%d,ph_q=%d,ph_b=%.1f",
			s.Placeholder.Width, s.Placeholder.Height, s.Placeholder.Quality, s.Placeholder.BlurSigma)
	}

	if len(s.Modifiers) > 0 {
		keys := slices.Sorted(maps.Keys(s.Modifiers))
		for _, k := range keys {
			_, _ = fmt.Fprintf(b, ",%s=%s", k, s.Modifiers[k])
		}
	}
	return b.String()
}

// DefaultTransformationSpec returns a TransformationSpec with sensible default
// values.
//
// Returns TransformationSpec which uses WebP format and contain fit mode.
func DefaultTransformationSpec() TransformationSpec {
	return TransformationSpec{
		Provider:           "",
		Width:              0,
		Height:             0,
		Fit:                FitContain,
		AspectRatio:        "",
		WithoutEnlargement: false,
		Background:         "",
		Format:             "webp",
		Quality:            defaultQuality,
		Modifiers:          map[string]string{},
		Responsive:         nil,
		Placeholder:        nil,
	}
}
