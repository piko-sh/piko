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

package video_dto

import (
	"strings"
	"testing"
	"time"
)

func TestThumbnailSpec_Validate(t *testing.T) {
	valid := ThumbnailSpec{Format: "jpeg", Quality: 85}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid spec: %v", err)
	}
}

func TestThumbnailSpec_Validate_Errors(t *testing.T) {
	tests := []struct {
		name string
		want string
		spec ThumbnailSpec
	}{
		{name: "negative timestamp", want: "timestamp", spec: ThumbnailSpec{Timestamp: -time.Second}},
		{name: "negative width", want: "width", spec: ThumbnailSpec{Width: -1}},
		{name: "negative height", want: "height", spec: ThumbnailSpec{Height: -1}},
		{name: "quality too high", want: "quality", spec: ThumbnailSpec{Quality: 101}},
		{name: "quality negative", want: "quality", spec: ThumbnailSpec{Quality: -1}},
		{name: "bad format", want: "unsupported format", spec: ThumbnailSpec{Format: "bmp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Errorf("error %q should contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestThumbnailSpec_WithDefaults(t *testing.T) {
	spec := ThumbnailSpec{}
	result := spec.WithDefaults()
	if result.Format != "jpeg" {
		t.Errorf("Format = %q, want jpeg", result.Format)
	}
	if result.Quality != 85 {
		t.Errorf("Quality = %d, want 85", result.Quality)
	}

	spec2 := ThumbnailSpec{Format: "png", Quality: 50}
	result2 := spec2.WithDefaults()
	if result2.Format != "png" {
		t.Errorf("Format = %q, want png", result2.Format)
	}
	if result2.Quality != 50 {
		t.Errorf("Quality = %d, want 50", result2.Quality)
	}
}

func TestParseThumbnailTime(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{input: "", want: 0},
		{input: "5s", want: 5 * time.Second},
		{input: "1m30s", want: 90 * time.Second},
		{input: "1:30", want: 90 * time.Second},
		{input: "1:30:00", want: 90 * time.Minute},
		{input: "0:05.5", want: 5*time.Second + 500*time.Millisecond},
		{input: "30", want: 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseThumbnailTime(tt.input)
			if err != nil {
				t.Fatalf("ParseThumbnailTime(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseThumbnailTime_Errors(t *testing.T) {
	tests := []string{"abc:definition", "a:b:c"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseThumbnailTime(input)
			if err == nil {
				t.Errorf("expected error for %q", input)
			}
		})
	}
}
