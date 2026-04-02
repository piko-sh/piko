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

package driven_svgwriter

import (
	"context"

	"piko.sh/piko/internal/layouter/layouter_domain"
	"piko.sh/piko/internal/pdfwriter/pdfwriter_domain"
)

const (
	// fallbackImageDimension holds the default width and height returned
	// when no inner resolver is available.
	fallbackImageDimension = 100
)

// SVGImageResolver wraps an inner ImageResolverPort and returns actual
// intrinsic dimensions for SVG sources by parsing the root <svg>
// element. Non-SVG sources are delegated to the inner resolver.
type SVGImageResolver struct {
	// inner holds the delegate resolver used for non-SVG sources.
	inner layouter_domain.ImageResolverPort

	// svgData holds the adapter that provides raw SVG markup for a given source.
	svgData pdfwriter_domain.SVGDataPort
}

var _ layouter_domain.ImageResolverPort = (*SVGImageResolver)(nil)

// NewSVGImageResolver creates a resolver that extracts SVG intrinsic
// dimensions for SVG sources and delegates all other sources to the
// inner resolver.
//
// Takes inner (ImageResolverPort) which handles non-SVG sources.
// Takes svgData (SVGDataPort) which provides raw SVG markup.
//
// Returns *SVGImageResolver which implements ImageResolverPort.
func NewSVGImageResolver(inner layouter_domain.ImageResolverPort, svgData pdfwriter_domain.SVGDataPort) *SVGImageResolver {
	return &SVGImageResolver{inner: inner, svgData: svgData}
}

// GetImageDimensions returns intrinsic width and height for the given source.
//
// For SVG sources, it parses the root <svg> element to extract
// width/height/viewBox. For other sources, delegates to the inner resolver.
//
// Takes source (string) which is the image source path or data URI.
//
// Returns width (float64) which is the intrinsic width in points.
// Returns height (float64) which is the intrinsic height in points.
// Returns err (error) when the inner resolver fails.
func (r *SVGImageResolver) GetImageDimensions(ctx context.Context, source string) (width, height float64, err error) {
	if w, h, ok := r.trySVGDimensions(ctx, source); ok {
		return w, h, nil
	}
	if r.inner != nil {
		return r.inner.GetImageDimensions(ctx, source)
	}
	return fallbackImageDimension, fallbackImageDimension, nil
}

// trySVGDimensions attempts to extract intrinsic dimensions
// from an SVG source by fetching the SVG data and parsing
// the root element.
//
// Takes source (string) which is the image source path or
// data URI.
//
// Returns width (float64) which is the SVG intrinsic width.
// Returns height (float64) which is the SVG intrinsic height.
// Returns ok (bool) which is true if valid SVG dimensions were found.
func (r *SVGImageResolver) trySVGDimensions(ctx context.Context, source string) (width float64, height float64, ok bool) {
	if r.svgData == nil {
		return 0, 0, false
	}
	svgXML, ok := r.svgData.GetSVGData(ctx, source)
	if !ok {
		return 0, 0, false
	}
	svg, err := ParseSVGString(svgXML)
	if err != nil {
		return 0, 0, false
	}
	w := svg.IntrinsicWidth()
	h := svg.IntrinsicHeight()
	if w > 0 && h > 0 {
		return w, h, true
	}
	return 0, 0, false
}
