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
	"strings"
	"testing"
)

func TestDefaultTransformationSpec(t *testing.T) {
	t.Parallel()

	spec := DefaultTransformationSpec()

	if spec.Format != "webp" {
		t.Errorf("Format = %q, want webp", spec.Format)
	}
	if spec.Fit != FitContain {
		t.Errorf("Fit = %q, want contain", spec.Fit)
	}
	if spec.Quality != 80 {
		t.Errorf("Quality = %d, want 80", spec.Quality)
	}
	if spec.Width != 0 {
		t.Errorf("Width = %d, want 0", spec.Width)
	}
	if spec.Height != 0 {
		t.Errorf("Height = %d, want 0", spec.Height)
	}
	if spec.WithoutEnlargement {
		t.Error("WithoutEnlargement should be false")
	}
	if spec.Provider != "" {
		t.Errorf("Provider = %q, want empty", spec.Provider)
	}
	if spec.Background != "" {
		t.Errorf("Background = %q, want empty", spec.Background)
	}
	if spec.AspectRatio != "" {
		t.Errorf("AspectRatio = %q, want empty", spec.AspectRatio)
	}
	if spec.Modifiers == nil {
		t.Error("Modifiers should not be nil")
	}
	if len(spec.Modifiers) != 0 {
		t.Errorf("Modifiers length = %d, want 0", len(spec.Modifiers))
	}
	if spec.Responsive != nil {
		t.Error("Responsive should be nil")
	}
	if spec.Placeholder != nil {
		t.Error("Placeholder should be nil")
	}
}

func TestTransformationSpec_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contains []string
		absent   []string
		spec     TransformationSpec
	}{
		{
			name: "base case",
			spec: TransformationSpec{
				Format:  "webp",
				Fit:     FitContain,
				Quality: 80,
			},
			contains: []string{
				"fmt=webp",
				"fit=contain",
				"q=80",
				"w=0",
				"h=0",
				"we=false",
			},
			absent: []string{"resp_", "ph="},
		},
		{
			name: "with provider and dimensions",
			spec: TransformationSpec{
				Provider: "vips",
				Width:    800,
				Height:   600,
				Format:   "jpeg",
				Fit:      FitCover,
				Quality:  90,
			},
			contains: []string{
				"p=vips",
				"w=800",
				"h=600",
				"fmt=jpeg",
				"fit=cover",
				"q=90",
			},
		},
		{
			name: "with responsive sizes only",
			spec: TransformationSpec{
				Format:  "webp",
				Fit:     FitContain,
				Quality: 80,
				Responsive: &ResponsiveSpec{
					Sizes: "100vw sm:50vw",
				},
			},
			contains: []string{"resp_sizes=100vw sm:50vw"},
			absent:   []string{"resp_dens="},
		},
		{
			name: "with responsive sizes and densities",
			spec: TransformationSpec{
				Format:  "webp",
				Fit:     FitContain,
				Quality: 80,
				Responsive: &ResponsiveSpec{
					Sizes:     "100vw",
					Densities: []string{"x1", "x2"},
				},
			},
			contains: []string{"resp_sizes=100vw", "resp_dens=x1+x2"},
		},
		{
			name: "with placeholder enabled",
			spec: TransformationSpec{
				Format:  "webp",
				Fit:     FitContain,
				Quality: 80,
				Placeholder: &PlaceholderSpec{
					Enabled:   true,
					Width:     20,
					Height:    15,
					Quality:   10,
					BlurSigma: 5.0,
				},
			},
			contains: []string{"ph=t", "ph_w=20", "ph_h=15", "ph_q=10", "ph_b=5.0"},
		},
		{
			name: "with placeholder disabled",
			spec: TransformationSpec{
				Format:  "webp",
				Fit:     FitContain,
				Quality: 80,
				Placeholder: &PlaceholderSpec{
					Enabled: false,
					Width:   20,
				},
			},
			absent: []string{"ph=t"},
		},
		{
			name: "with modifiers sorted",
			spec: TransformationSpec{
				Format:  "png",
				Fit:     FitFill,
				Quality: 70,
				Modifiers: map[string]string{
					"z_last":  "3",
					"a_first": "1",
					"m_mid":   "2",
				},
			},
			contains: []string{"a_first=1", "m_mid=2", "z_last=3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.spec.String()

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("String() = %q, missing %q", result, substr)
				}
			}
			for _, substr := range tt.absent {
				if strings.Contains(result, substr) {
					t.Errorf("String() = %q, should not contain %q", result, substr)
				}
			}
		})
	}
}
