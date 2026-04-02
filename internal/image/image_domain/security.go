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

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

// securityViolationType groups different kinds of security violations.
type securityViolationType string

const (
	// violationDimensionExceeded indicates that an image dimension (width or
	// height) has exceeded the set limits.
	violationDimensionExceeded securityViolationType = "dimension_exceeded"

	// violationPixelCountExceeded means the image has too many pixels.
	violationPixelCountExceeded securityViolationType = "pixel_count_exceeded"

	// violationInvalidFormat indicates that a requested image format is not in the
	// allowed list.
	violationInvalidFormat securityViolationType = "invalid_format"
)

// ErrSizeLimitExceeded is returned by LimitedReader when the input exceeds the
// configured byte limit.
var ErrSizeLimitExceeded = errors.New("input size exceeded limit")

// securityViolation represents a security check failure. It implements the
// error interface and provides details about the specific security issue found.
type securityViolation struct {
	// details contains extra context about the violation.
	details map[string]any

	// message is a description of the security violation that users can read.
	message string

	// violationType is the category of security violation.
	violationType securityViolationType
}

// Error returns the formatted security violation message.
//
// Returns string which contains the violation type and message.
func (v securityViolation) Error() string {
	return fmt.Sprintf("security violation [%s]: %s", v.violationType, v.message)
}

// LimitedReader wraps an io.Reader and enforces a maximum read size.
// It prevents reading more than maxBytes, protecting against memory exhaustion.
type LimitedReader struct {
	// reader is the underlying source from which data is read.
	reader io.Reader

	// maxBytes is the maximum number of bytes that may be read.
	maxBytes int64

	// read tracks the total bytes read so far.
	read int64
}

// NewLimitedReader creates a reader that returns an error if more than the
// specified maximum bytes are read.
//
// Takes r (io.Reader) which is the underlying reader to wrap.
// Takes maxBytes (int64) which is the maximum number of bytes allowed.
//
// Returns *LimitedReader which wraps the reader with a byte limit check.
func NewLimitedReader(r io.Reader, maxBytes int64) *LimitedReader {
	return &LimitedReader{
		reader:   r,
		maxBytes: maxBytes,
		read:     0,
	}
}

// Read reads from the underlying reader and tracks the total bytes read.
// The read buffer is capped so that no more than maxBytes are ever read from
// the underlying source, preventing over-read past the configured limit.
//
// Takes p ([]byte) which is the buffer to read data into.
//
// Returns n (int) which is the number of bytes read into p.
// Returns err (error) when the underlying reader returns an error or
// the byte limit has been exceeded.
func (lr *LimitedReader) Read(p []byte) (n int, err error) {
	if lr.read >= lr.maxBytes {
		return 0, fmt.Errorf("%w: %d bytes", ErrSizeLimitExceeded, lr.maxBytes)
	}

	remaining := lr.maxBytes - lr.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = lr.reader.Read(p)
	lr.read += int64(n)

	return n, err
}

// ValidateImageDimensions checks if image dimensions and total pixel count are
// within allowed limits. This combines width/height checks with a pixel count
// check to prevent callers from accidentally skipping the pixel budget.
//
// Takes width (int) which specifies the image width in pixels.
// Takes height (int) which specifies the image height in pixels.
// Takes config (ServiceConfig) which provides the maximum dimension limits.
//
// Returns error when the width, height, or total pixel count exceeds the
// configured limits.
func ValidateImageDimensions(ctx context.Context, width, height int, config ServiceConfig) error {
	if width > config.MaxImageWidth {
		recordSecurityViolation(ctx, violationDimensionExceeded, map[string]any{
			"width": width,
			"limit": config.MaxImageWidth,
		})
		return securityViolation{
			violationType: violationDimensionExceeded,
			message:       fmt.Sprintf("image width %d exceeds maximum allowed %d", width, config.MaxImageWidth),
			details: map[string]any{
				"width":     width,
				"max_width": config.MaxImageWidth,
			},
		}
	}

	if height > config.MaxImageHeight {
		recordSecurityViolation(ctx, violationDimensionExceeded, map[string]any{
			"height": height,
			"limit":  config.MaxImageHeight,
		})
		return securityViolation{
			violationType: violationDimensionExceeded,
			message:       fmt.Sprintf("image height %d exceeds maximum allowed %d", height, config.MaxImageHeight),
			details: map[string]any{
				"height":     height,
				"max_height": config.MaxImageHeight,
			},
		}
	}

	return ValidateImagePixelCount(ctx, width, height, config)
}

// ValidateImagePixelCount checks if the total pixel count is within the allowed
// limit.
//
// Takes width (int) which is the image width in pixels.
// Takes height (int) which is the image height in pixels.
// Takes config (ServiceConfig) which provides the maximum pixel limit.
//
// Returns error when the total pixel count is greater than the set maximum.
func ValidateImagePixelCount(ctx context.Context, width, height int, config ServiceConfig) error {
	if config.MaxImagePixels <= 0 {
		return nil
	}

	pixels := int64(width) * int64(height)

	if pixels > config.MaxImagePixels {
		recordSecurityViolation(ctx, violationPixelCountExceeded, map[string]any{
			"pixels": pixels,
			"limit":  config.MaxImagePixels,
		})
		return securityViolation{
			violationType: violationPixelCountExceeded,
			message:       fmt.Sprintf("image size %d pixels (%dx%d) exceeds maximum allowed %d", pixels, width, height, config.MaxImagePixels),
			details: map[string]any{
				"width":      width,
				"height":     height,
				"pixels":     pixels,
				"max_pixels": config.MaxImagePixels,
			},
		}
	}

	return nil
}

// ValidateImageFormat checks if the requested output format is allowed.
//
// When the config has no allowed formats, returns nil without restriction.
//
// Takes format (string) which specifies the requested output format.
// Takes config (ServiceConfig) which provides the list of allowed formats.
//
// Returns error when the format is not in the allowed list. The error is a
// SecurityViolation and a security violation is recorded.
func ValidateImageFormat(ctx context.Context, format string, config ServiceConfig) error {
	if len(config.AllowedFormats) == 0 {
		return nil
	}

	for _, allowed := range config.AllowedFormats {
		if strings.EqualFold(allowed, format) {
			return nil
		}
	}

	recordSecurityViolation(ctx, violationInvalidFormat, map[string]any{
		"format":          format,
		"allowed_formats": config.AllowedFormats,
	})
	return securityViolation{
		violationType: violationInvalidFormat,
		message:       fmt.Sprintf("output format '%s' is not in allowed list: %v", format, config.AllowedFormats),
		details: map[string]any{
			"format":          format,
			"allowed_formats": config.AllowedFormats,
		},
	}
}

// recordSecurityViolation increases the security violation counter and logs
// the violation for audit purposes.
//
// Takes violationType (securityViolationType) which identifies the kind of
// security violation found.
// Takes details (map[string]any) which provides extra context about the
// violation.
func recordSecurityViolation(ctx context.Context, violationType securityViolationType, details map[string]any) {
	ctx, l := logger_domain.From(ctx, log)
	securityViolationCount.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("violation_type", string(violationType)),
		),
	)

	l.Warn("Security violation detected",
		logger_domain.String("violation_type", string(violationType)),
		logger_domain.String("details", fmt.Sprintf("%+v", details)),
	)
}
