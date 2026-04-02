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
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLimitedReader(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		maxBytes int64
		readSize int
		wantErr  error
	}{
		{
			name:     "read within limit",
			data:     []byte("hello world"),
			maxBytes: 100,
			readSize: 11,
		},
		{
			name:     "read exactly at limit",
			data:     []byte("hello"),
			maxBytes: 5,
			readSize: 5,
		},
		{
			name:     "read exceeds limit",
			data:     []byte("this is a very long string that exceeds the limit"),
			maxBytes: 10,
			readSize: 50,
			wantErr:  ErrSizeLimitExceeded,
		},
		{
			name:     "empty data",
			data:     []byte(""),
			maxBytes: 100,
			readSize: 0,
		},
		{
			name:     "zero max bytes still allows zero reads",
			data:     []byte(""),
			maxBytes: 0,
			readSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			limitedReader := NewLimitedReader(reader, tt.maxBytes)

			bufferSize := max(tt.readSize, len(tt.data))
			buffer := make([]byte, bufferSize)
			_, err := io.ReadFull(limitedReader, buffer[:tt.readSize])

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LimitedReader.Read() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil && !errors.Is(err, io.EOF) {
					t.Errorf("LimitedReader.Read() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestLimitedReader_NeverReadsPastLimit(t *testing.T) {
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	reader := bytes.NewReader(data)
	limitedReader := NewLimitedReader(reader, 10)

	buffer := make([]byte, 50)
	n, err := limitedReader.Read(buffer)
	if err != nil {
		t.Fatalf("Read() returned unexpected error: %v", err)
	}
	if n != 10 {
		t.Errorf("Read() read %d bytes, want exactly 10 (the limit)", n)
	}

	n2, err := limitedReader.Read(buffer)
	if err == nil {
		t.Errorf("Read() past limit should return error, got %d bytes", n2)
	}
	if n2 != 0 {
		t.Errorf("Read() past limit returned %d bytes, want 0", n2)
	}
}

func TestLimitedReader_MultipleReads(t *testing.T) {
	data := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	maxBytes := int64(10)
	reader := bytes.NewReader(data)
	limitedReader := NewLimitedReader(reader, maxBytes)

	buffer1 := make([]byte, 5)
	n1, err := limitedReader.Read(buffer1)
	if err != nil {
		t.Fatalf("First read failed: %v", err)
	}
	if n1 != 5 {
		t.Errorf("First read: got %d bytes, want 5", n1)
	}

	buffer2 := make([]byte, 5)
	n2, err := limitedReader.Read(buffer2)
	if err != nil {
		t.Fatalf("Second read failed: %v", err)
	}
	if n2 != 5 {
		t.Errorf("Second read: got %d bytes, want 5", n2)
	}

	buffer3 := make([]byte, 5)
	_, err = limitedReader.Read(buffer3)
	if err == nil {
		t.Errorf("Third read: expected error but got nil")
	} else if !errors.Is(err, ErrSizeLimitExceeded) {
		t.Errorf("Third read: error = %v, want %v", err, ErrSizeLimitExceeded)
	}
}

func TestValidateImageDimensions(t *testing.T) {
	config := ServiceConfig{
		MaxImageWidth:  2000,
		MaxImageHeight: 1500,
		MaxImagePixels: 25_000_000,
	}

	tests := []struct {
		name          string
		errContains   string
		violationType securityViolationType
		width         int
		height        int
		wantErr       bool
	}{
		{
			name:    "valid dimensions",
			width:   1920,
			height:  1080,
			wantErr: false,
		},
		{
			name:    "maximum allowed dimensions",
			width:   2000,
			height:  1500,
			wantErr: false,
		},
		{
			name:          "width exceeds limit",
			width:         2001,
			height:        1080,
			wantErr:       true,
			errContains:   "width",
			violationType: violationDimensionExceeded,
		},
		{
			name:          "height exceeds limit",
			width:         1920,
			height:        1501,
			wantErr:       true,
			errContains:   "height",
			violationType: violationDimensionExceeded,
		},
		{
			name:          "both dimensions exceed limit",
			width:         3000,
			height:        2000,
			wantErr:       true,
			errContains:   "width",
			violationType: violationDimensionExceeded,
		},
		{
			name:    "minimum dimensions",
			width:   1,
			height:  1,
			wantErr: false,
		},
		{
			name:    "zero dimensions are valid",
			width:   0,
			height:  0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := ValidateImageDimensions(ctx, tt.width, tt.height, config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageDimensions() expected error but got nil")
				} else {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("ValidateImageDimensions() error = %v, want error containing %q", err, tt.errContains)
					}
					if secViolation, ok := errors.AsType[securityViolation](err); ok {
						if secViolation.violationType != tt.violationType {
							t.Errorf("ValidateImageDimensions() violation type = %v, want %v", secViolation.violationType, tt.violationType)
						}
						if secViolation.details == nil {
							t.Errorf("ValidateImageDimensions() violation details is nil")
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateImageDimensions() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateImagePixelCount(t *testing.T) {
	config := ServiceConfig{
		MaxImagePixels: 25_000_000,
	}

	tests := []struct {
		name          string
		errContains   string
		violationType securityViolationType
		width         int
		height        int
		wantErr       bool
	}{
		{
			name:    "valid pixel count",
			width:   5000,
			height:  5000,
			wantErr: false,
		},
		{
			name:    "maximum allowed pixels",
			width:   5000,
			height:  5000,
			wantErr: false,
		},
		{
			name:          "pixel count exceeds limit",
			width:         6000,
			height:        5000,
			wantErr:       true,
			errContains:   "pixels",
			violationType: violationPixelCountExceeded,
		},
		{
			name:          "large dimensions exceed pixel limit",
			width:         10000,
			height:        10000,
			wantErr:       true,
			errContains:   "pixels",
			violationType: violationPixelCountExceeded,
		},
		{
			name:    "small dimensions well under limit",
			width:   100,
			height:  100,
			wantErr: false,
		},
		{
			name:    "one dimension zero results in zero pixels",
			width:   5000,
			height:  0,
			wantErr: false,
		},
		{
			name:    "both dimensions zero",
			width:   0,
			height:  0,
			wantErr: false,
		},
		{
			name:    "narrow but tall image",
			width:   100,
			height:  250000,
			wantErr: false,
		},
		{
			name:    "wide but short image",
			width:   250000,
			height:  100,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := ValidateImagePixelCount(ctx, tt.width, tt.height, config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImagePixelCount() expected error but got nil")
				} else {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("ValidateImagePixelCount() error = %v, want error containing %q", err, tt.errContains)
					}
					if secViolation, ok := errors.AsType[securityViolation](err); ok {
						if secViolation.violationType != tt.violationType {
							t.Errorf("ValidateImagePixelCount() violation type = %v, want %v", secViolation.violationType, tt.violationType)
						}
						if secViolation.details == nil {
							t.Errorf("ValidateImagePixelCount() violation details is nil")
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateImagePixelCount() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateImageFormat(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		errContains    string
		violationType  securityViolationType
		allowedFormats []string
		wantErr        bool
	}{
		{
			name:           "allowed format",
			format:         "jpeg",
			allowedFormats: []string{"jpeg", "png", "webp"},
			wantErr:        false,
		},
		{
			name:           "format in uppercase is allowed",
			format:         "JPEG",
			allowedFormats: []string{"jpeg", "png", "webp"},
			wantErr:        false,
		},
		{
			name:           "disallowed format",
			format:         "avif",
			allowedFormats: []string{"jpeg", "png", "webp"},
			wantErr:        true,
			errContains:    "not in allowed list",
			violationType:  violationInvalidFormat,
		},
		{
			name:           "empty allowed formats list allows all",
			format:         "avif",
			allowedFormats: []string{},
			wantErr:        false,
		},
		{
			name:           "nil allowed formats list allows all",
			format:         "bmp",
			allowedFormats: nil,
			wantErr:        false,
		},
		{
			name:           "multiple allowed formats",
			format:         "webp",
			allowedFormats: []string{"jpeg", "jpg", "png", "webp", "avif", "gif"},
			wantErr:        false,
		},
		{
			name:           "format not in restricted list",
			format:         "tiff",
			allowedFormats: []string{"jpeg", "png"},
			wantErr:        true,
			errContains:    "not in allowed list",
			violationType:  violationInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := ServiceConfig{
				AllowedFormats: tt.allowedFormats,
			}

			err := ValidateImageFormat(ctx, tt.format, config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateImageFormat() expected error but got nil")
				} else {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("ValidateImageFormat() error = %v, want error containing %q", err, tt.errContains)
					}
					if secViolation, ok := errors.AsType[securityViolation](err); ok {
						if secViolation.violationType != tt.violationType {
							t.Errorf("ValidateImageFormat() violation type = %v, want %v", secViolation.violationType, tt.violationType)
						}
						if secViolation.details == nil {
							t.Errorf("ValidateImageFormat() violation details is nil")
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateImageFormat() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSecurityViolation_Error(t *testing.T) {
	tests := []struct {
		name        string
		violation   securityViolation
		wantContain []string
	}{
		{
			name: "dimension exceeded",
			violation: securityViolation{
				violationType: violationDimensionExceeded,
				message:       "image width 3000 exceeds maximum allowed 2000",
				details: map[string]any{
					"width":     3000,
					"max_width": 2000,
				},
			},
			wantContain: []string{"security violation", "dimension_exceeded", "3000", "2000"},
		},
		{
			name: "pixel count exceeded",
			violation: securityViolation{
				violationType: violationPixelCountExceeded,
				message:       "image size 30000000 pixels exceeds maximum allowed 25000000",
				details: map[string]any{
					"pixels":     30000000,
					"max_pixels": 25000000,
				},
			},
			wantContain: []string{"security violation", "pixel_count_exceeded", "30000000", "25000000"},
		},
		{
			name: "invalid format",
			violation: securityViolation{
				violationType: violationInvalidFormat,
				message:       "output format 'bmp' is not in allowed list",
				details: map[string]any{
					"format":          "bmp",
					"allowed_formats": []string{"jpeg", "png", "webp"},
				},
			},
			wantContain: []string{"security violation", "invalid_format", "bmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errMessage := tt.violation.Error()

			for _, substr := range tt.wantContain {
				if !strings.Contains(errMessage, substr) {
					t.Errorf("SecurityViolation.Error() = %q, want to contain %q", errMessage, substr)
				}
			}

			if !strings.HasPrefix(errMessage, "security violation [") {
				t.Errorf("SecurityViolation.Error() = %q, want to start with 'security violation ['", errMessage)
			}
		})
	}
}

func TestSecurityViolationType_String(t *testing.T) {
	tests := []struct {
		name          string
		violationType securityViolationType
		wantString    string
	}{
		{
			name:          "dimension exceeded",
			violationType: violationDimensionExceeded,
			wantString:    "dimension_exceeded",
		},
		{
			name:          "pixel count exceeded",
			violationType: violationPixelCountExceeded,
			wantString:    "pixel_count_exceeded",
		},
		{
			name:          "invalid format",
			violationType: violationInvalidFormat,
			wantString:    "invalid_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(tt.violationType)
			if got != tt.wantString {
				t.Errorf("SecurityViolationType string = %q, want %q", got, tt.wantString)
			}
		})
	}
}

func TestValidateImageDimensions_AlsoChecksPixelCount(t *testing.T) {
	config := ServiceConfig{
		MaxImageWidth:  10000,
		MaxImageHeight: 10000,
		MaxImagePixels: 50_000_000,
	}

	ctx := context.Background()

	err := ValidateImageDimensions(ctx, 10000, 10000, config)
	if err == nil {
		t.Error("ValidateImageDimensions() should reject 100M pixels when limit is 50M")
	}
	if secV, ok := errors.AsType[securityViolation](err); ok {
		if secV.violationType != violationPixelCountExceeded {
			t.Errorf("expected pixel_count_exceeded violation, got %v", secV.violationType)
		}
	}

	err = ValidateImageDimensions(ctx, 5000, 5000, config)
	if err != nil {
		t.Errorf("ValidateImageDimensions() unexpected error for 25M pixels: %v", err)
	}

	configNoPixelLimit := ServiceConfig{
		MaxImageWidth:  10000,
		MaxImageHeight: 10000,
	}
	err = ValidateImageDimensions(ctx, 10000, 10000, configNoPixelLimit)
	if err != nil {
		t.Errorf("ValidateImageDimensions() should pass when MaxImagePixels is 0: %v", err)
	}
}

func TestValidateImageDimensions_EdgeCases(t *testing.T) {
	config := ServiceConfig{
		MaxImageWidth:  8192,
		MaxImageHeight: 8192,
		MaxImagePixels: 100_000_000,
	}

	tests := []struct {
		name    string
		width   int
		height  int
		wantErr bool
	}{
		{
			name:    "exactly at width limit",
			width:   8192,
			height:  4000,
			wantErr: false,
		},
		{
			name:    "exactly at height limit",
			width:   4000,
			height:  8192,
			wantErr: false,
		},
		{
			name:    "both at limit",
			width:   8192,
			height:  8192,
			wantErr: false,
		},
		{
			name:    "one pixel over width limit",
			width:   8193,
			height:  4000,
			wantErr: true,
		},
		{
			name:    "one pixel over height limit",
			width:   4000,
			height:  8193,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := ValidateImageDimensions(ctx, tt.width, tt.height, config)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateImageDimensions() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateImageDimensions() unexpected error = %v", err)
			}
		})
	}
}

func TestValidateImagePixelCount_EdgeCases(t *testing.T) {
	config := ServiceConfig{
		MaxImagePixels: 25_000_000,
	}

	tests := []struct {
		name    string
		width   int
		height  int
		wantErr bool
	}{
		{
			name:    "exactly at pixel limit",
			width:   5000,
			height:  5000,
			wantErr: false,
		},
		{
			name:    "one pixel over limit",
			width:   5001,
			height:  5000,
			wantErr: true,
		},
		{
			name:    "alternative dimensions at limit",
			width:   6250,
			height:  4000,
			wantErr: false,
		},
		{
			name:    "very wide and short at limit",
			width:   25000000,
			height:  1,
			wantErr: false,
		},
		{
			name:    "very tall and narrow at limit",
			width:   1,
			height:  25000000,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := ValidateImagePixelCount(ctx, tt.width, tt.height, config)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateImagePixelCount() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateImagePixelCount() unexpected error = %v", err)
			}
		})
	}
}
