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
	"strings"
	"testing"

	"piko.sh/piko/internal/image/image_dto"
)

func TestValidateTransformationSpec_BasicValidation(t *testing.T) {
	tests := []struct {
		name        string
		errContains string
		spec        image_dto.TransformationSpec
		wantErr     bool
	}{
		{
			name: "valid spec with all fields",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "cover",
			},
			wantErr: false,
		},
		{
			name: "negative width",
			spec: image_dto.TransformationSpec{
				Width:   -100,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
			},
			wantErr:     true,
			errContains: "width cannot be negative",
		},
		{
			name: "negative height",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  -200,
				Quality: 85,
				Format:  "jpeg",
			},
			wantErr:     true,
			errContains: "height cannot be negative",
		},
		{
			name: "quality too low",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 0,
				Format:  "jpeg",
			},
			wantErr:     true,
			errContains: "quality must be between 1 and 100",
		},
		{
			name: "quality too high",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 101,
				Format:  "jpeg",
			},
			wantErr:     true,
			errContains: "quality must be between 1 and 100",
		},
		{
			name: "unsupported format",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "bmp",
			},
			wantErr:     true,
			errContains: "unsupported output format",
		},
		{
			name: "uppercase format normalised to lowercase",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "JPEG",
				Fit:     "cover",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateTransformationSpec(tt.spec, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTransformationSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransformationSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTransformationSpec() unexpected error = %v", err)
				}

				if result.Format != strings.ToLower(tt.spec.Format) {
					t.Errorf("ValidateTransformationSpec() format not normalised: got %q, want %q", result.Format, strings.ToLower(tt.spec.Format))
				}
			}
		})
	}
}

func TestValidateTransformationSpec_FitMode(t *testing.T) {
	tests := []struct {
		name        string
		expectedFit image_dto.FitMode
		errContains string
		spec        image_dto.TransformationSpec
		wantErr     bool
	}{
		{
			name: "fit mode cover",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "cover",
			},
			expectedFit: "cover",
			wantErr:     false,
		},
		{
			name: "fit mode contain",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "contain",
			},
			expectedFit: "contain",
			wantErr:     false,
		},
		{
			name: "fit mode fill",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "fill",
			},
			expectedFit: "fill",
			wantErr:     false,
		},
		{
			name: "fit mode inside",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "inside",
			},
			expectedFit: "inside",
			wantErr:     false,
		},
		{
			name: "fit mode outside",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "outside",
			},
			expectedFit: "outside",
			wantErr:     false,
		},
		{
			name: "empty fit defaults to contain",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
			},
			expectedFit: "contain",
			wantErr:     false,
		},
		{
			name: "uppercase fit normalised to lowercase",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "COVER",
			},
			expectedFit: "cover",
			wantErr:     false,
		},
		{
			name: "invalid fit mode",
			spec: image_dto.TransformationSpec{
				Width:   800,
				Height:  600,
				Quality: 85,
				Format:  "jpeg",
				Fit:     "stretch",
			},
			wantErr:     true,
			errContains: "unsupported fit mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateTransformationSpec(tt.spec, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTransformationSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransformationSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTransformationSpec() unexpected error = %v", err)
				}
				if result.Fit != tt.expectedFit {
					t.Errorf("ValidateTransformationSpec() fit = %q, want %q", result.Fit, tt.expectedFit)
				}
			}
		})
	}
}

func TestValidateTransformationSpec_BackgroundColour(t *testing.T) {
	tests := []struct {
		name        string
		background  string
		errContains string
		wantErr     bool
	}{
		{
			name:       "valid hex colour",
			background: "#FF0000",
			wantErr:    false,
		},
		{
			name:       "valid hex colour normalised to lowercase",
			background: "#FF0000",
			wantErr:    false,
		},
		{
			name:        "invalid hex colour without hash",
			background:  "FF0000",
			wantErr:     true,
			errContains: "background colour must be a hex colour starting with #",
		},
		{
			name:       "empty background is valid",
			background: "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := image_dto.TransformationSpec{
				Width:      800,
				Height:     600,
				Quality:    85,
				Format:     "jpeg",
				Fit:        "cover",
				Background: tt.background,
			}
			result, err := ValidateTransformationSpec(spec, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTransformationSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransformationSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTransformationSpec() unexpected error = %v", err)
				}
				if tt.background != "" && result.Background != strings.ToLower(tt.background) {
					t.Errorf("ValidateTransformationSpec() background not normalised: got %q, want %q", result.Background, strings.ToLower(tt.background))
				}
			}
		})
	}
}

func TestValidateTransformationSpec_AspectRatio(t *testing.T) {
	tests := []struct {
		name        string
		aspectRatio string
		errContains string
		wantErr     bool
	}{
		{
			name:        "valid aspect ratio 16:9",
			aspectRatio: "16:9",
			wantErr:     false,
		},
		{
			name:        "valid aspect ratio 4:3",
			aspectRatio: "4:3",
			wantErr:     false,
		},
		{
			name:        "valid aspect ratio 1:1",
			aspectRatio: "1:1",
			wantErr:     false,
		},
		{
			name:        "invalid aspect ratio missing colon",
			aspectRatio: "169",
			wantErr:     true,
			errContains: "invalid aspect ratio format",
		},
		{
			name:        "invalid aspect ratio too many parts",
			aspectRatio: "16:9:2",
			wantErr:     true,
			errContains: "invalid aspect ratio format",
		},
		{
			name:        "empty aspect ratio is valid",
			aspectRatio: "",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := image_dto.TransformationSpec{
				Width:       800,
				Height:      600,
				Quality:     85,
				Format:      "jpeg",
				Fit:         "cover",
				AspectRatio: tt.aspectRatio,
			}
			_, err := ValidateTransformationSpec(spec, nil)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTransformationSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransformationSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTransformationSpec() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateTransformationSpec_ProviderCapabilities(t *testing.T) {
	capabilities := map[string]providerCapability{
		"imaging": {
			supportedFormats: []string{"jpeg", "png", "webp"},
			supportedModifiers: map[string]bool{
				"blur":      true,
				"greyscale": true,
			},
		},
		"vips": {
			supportedFormats: []string{"jpeg", "png", "webp", "avif"},
			supportedModifiers: map[string]bool{
				"blur":      true,
				"greyscale": true,
				"hue":       true,
				"tint":      true,
			},
		},
	}

	tests := []struct {
		name        string
		errContains string
		spec        image_dto.TransformationSpec
		wantErr     bool
	}{
		{
			name: "valid format for provider",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "jpeg",
				Fit:      "cover",
				Provider: "imaging",
			},
			wantErr: false,
		},
		{
			name: "unsupported format for provider",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "avif",
				Fit:      "cover",
				Provider: "imaging",
			},
			wantErr:     true,
			errContains: "not supported by provider",
		},
		{
			name: "valid modifier for provider",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "jpeg",
				Fit:      "cover",
				Provider: "imaging",
				Modifiers: map[string]string{
					"blur": "2.0",
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported modifier for provider",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "jpeg",
				Fit:      "cover",
				Provider: "imaging",
				Modifiers: map[string]string{
					"hue": "180",
				},
			},
			wantErr:     true,
			errContains: "modifiers not supported by provider",
		},
		{
			name: "unknown provider skips validation",
			spec: image_dto.TransformationSpec{
				Width:    800,
				Height:   600,
				Quality:  85,
				Format:   "avif",
				Fit:      "cover",
				Provider: "unknown",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateTransformationSpec(tt.spec, capabilities)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateTransformationSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateTransformationSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateTransformationSpec() unexpected error = %v", err)
				}
			}
		})
	}
}

func Test_validateFormatForProvider(t *testing.T) {
	capabilities := map[string]providerCapability{
		"imaging": {
			supportedFormats: []string{"jpeg", "png", "webp"},
		},
		"vips": {
			supportedFormats: []string{"jpeg", "png", "webp", "avif"},
		},
	}

	tests := []struct {
		name        string
		format      string
		provider    string
		errContains string
		wantErr     bool
	}{
		{
			name:     "jpeg supported by imaging",
			format:   "jpeg",
			provider: "imaging",
			wantErr:  false,
		},
		{
			name:        "avif not supported by imaging",
			format:      "avif",
			provider:    "imaging",
			wantErr:     true,
			errContains: "not supported by provider",
		},
		{
			name:     "avif supported by vips",
			format:   "avif",
			provider: "vips",
			wantErr:  false,
		},
		{
			name:     "unknown provider returns no error",
			format:   "avif",
			provider: "unknown",
			wantErr:  false,
		},
		{
			name:     "format normalised to lowercase",
			format:   "JPEG",
			provider: "imaging",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFormatForProvider(tt.format, tt.provider, capabilities)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateFormatForProvider() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateFormatForProvider() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateFormatForProvider() unexpected error = %v", err)
				}
			}
		})
	}
}

func Test_validateModifiersForProvider(t *testing.T) {
	capabilities := map[string]providerCapability{
		"imaging": {
			supportedModifiers: map[string]bool{
				"blur":      true,
				"greyscale": true,
				"sharpen":   true,
			},
		},
		"vips": {
			supportedModifiers: map[string]bool{
				"blur":      true,
				"greyscale": true,
				"sharpen":   true,
				"hue":       true,
				"tint":      true,
				"gravity":   true,
			},
		},
	}

	tests := []struct {
		modifiers   map[string]string
		name        string
		provider    string
		errContains string
		wantErr     bool
	}{
		{
			name: "supported modifiers for imaging",
			modifiers: map[string]string{
				"blur":      "2.0",
				"greyscale": "true",
			},
			provider: "imaging",
			wantErr:  false,
		},
		{
			name: "unsupported modifier for imaging",
			modifiers: map[string]string{
				"hue": "180",
			},
			provider:    "imaging",
			wantErr:     true,
			errContains: "modifiers not supported by provider",
		},
		{
			name: "multiple unsupported modifiers",
			modifiers: map[string]string{
				"hue":     "180",
				"tint":    "#FF0000",
				"gravity": "center",
			},
			provider:    "imaging",
			wantErr:     true,
			errContains: "modifiers not supported by provider",
		},
		{
			name: "all modifiers supported by vips",
			modifiers: map[string]string{
				"blur":    "2.0",
				"hue":     "180",
				"gravity": "center",
			},
			provider: "vips",
			wantErr:  false,
		},
		{
			name: "unknown provider returns no error",
			modifiers: map[string]string{
				"hue": "180",
			},
			provider: "unknown",
			wantErr:  false,
		},
		{
			name:      "empty modifiers is valid",
			modifiers: map[string]string{},
			provider:  "imaging",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateModifiersForProvider(tt.modifiers, tt.provider, capabilities)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateModifiersForProvider() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateModifiersForProvider() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateModifiersForProvider() unexpected error = %v", err)
				}
			}
		})
	}
}

func Test_validateAspectRatio(t *testing.T) {
	tests := []struct {
		name        string
		aspectRatio string
		errContains string
		wantErr     bool
	}{
		{
			name:        "valid 16:9",
			aspectRatio: "16:9",
			wantErr:     false,
		},
		{
			name:        "valid 4:3",
			aspectRatio: "4:3",
			wantErr:     false,
		},
		{
			name:        "valid 1:1",
			aspectRatio: "1:1",
			wantErr:     false,
		},
		{
			name:        "valid with decimals",
			aspectRatio: "16.5:9.5",
			wantErr:     false,
		},
		{
			name:        "missing colon",
			aspectRatio: "169",
			wantErr:     true,
			errContains: "invalid aspect ratio format",
		},
		{
			name:        "too many parts",
			aspectRatio: "16:9:2",
			wantErr:     true,
			errContains: "invalid aspect ratio format",
		},
		{
			name:        "non-numeric width",
			aspectRatio: "abc:9",
			wantErr:     true,
			errContains: "invalid aspect ratio width",
		},
		{
			name:        "non-numeric height",
			aspectRatio: "16:xyz",
			wantErr:     true,
			errContains: "invalid aspect ratio height",
		},
		{
			name:        "zero width",
			aspectRatio: "0:9",
			wantErr:     true,
			errContains: "aspect ratio dimensions must be positive",
		},
		{
			name:        "zero height",
			aspectRatio: "16:0",
			wantErr:     true,
			errContains: "aspect ratio dimensions must be positive",
		},
		{
			name:        "negative width",
			aspectRatio: "-16:9",
			wantErr:     true,
			errContains: "aspect ratio dimensions must be positive",
		},
		{
			name:        "negative height",
			aspectRatio: "16:-9",
			wantErr:     true,
			errContains: "aspect ratio dimensions must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAspectRatio(tt.aspectRatio)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateAspectRatio() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateAspectRatio() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validateAspectRatio() unexpected error = %v", err)
				}
			}
		})
	}
}

func Test_validatePlaceholderSpec(t *testing.T) {
	tests := []struct {
		spec        *image_dto.PlaceholderSpec
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "valid placeholder spec",
			spec: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     20,
				Height:    20,
				Quality:   10,
				BlurSigma: 5.0,
			},
			wantErr: false,
		},
		{
			name: "disabled placeholder is valid",
			spec: &image_dto.PlaceholderSpec{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "zero dimensions uses defaults",
			spec: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     0,
				Height:    0,
				Quality:   10,
				BlurSigma: 5.0,
			},
			wantErr: false,
		},
		{
			name: "negative width",
			spec: &image_dto.PlaceholderSpec{
				Enabled: true,
				Width:   -10,
				Height:  20,
				Quality: 10,
			},
			wantErr:     true,
			errContains: "placeholder width cannot be negative",
		},
		{
			name: "negative height",
			spec: &image_dto.PlaceholderSpec{
				Enabled: true,
				Width:   20,
				Height:  -10,
				Quality: 10,
			},
			wantErr:     true,
			errContains: "placeholder height cannot be negative",
		},
		{
			name: "quality too low",
			spec: &image_dto.PlaceholderSpec{
				Enabled: true,
				Width:   20,
				Height:  20,
				Quality: -1,
			},
			wantErr:     true,
			errContains: "placeholder quality must be between 0 and 100",
		},
		{
			name: "quality too high",
			spec: &image_dto.PlaceholderSpec{
				Enabled: true,
				Width:   20,
				Height:  20,
				Quality: 101,
			},
			wantErr:     true,
			errContains: "placeholder quality must be between 0 and 100",
		},
		{
			name: "negative blur sigma",
			spec: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     20,
				Height:    20,
				Quality:   10,
				BlurSigma: -2.0,
			},
			wantErr:     true,
			errContains: "placeholder blur sigma cannot be negative",
		},
		{
			name: "zero quality is valid",
			spec: &image_dto.PlaceholderSpec{
				Enabled: true,
				Width:   20,
				Height:  20,
				Quality: 0,
			},
			wantErr: false,
		},
		{
			name: "zero blur sigma is valid",
			spec: &image_dto.PlaceholderSpec{
				Enabled:   true,
				Width:     20,
				Height:    20,
				Quality:   10,
				BlurSigma: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlaceholderSpec(tt.spec)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePlaceholderSpec() expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validatePlaceholderSpec() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validatePlaceholderSpec() unexpected error = %v", err)
				}
			}
		})
	}
}
