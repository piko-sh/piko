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

package layouter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFontStyle_String(t *testing.T) {
	tests := []struct {
		name     string
		style    FontStyle
		expected string
	}{
		{"normal", FontStyleNormal, "normal"},
		{"italic", FontStyleItalic, "italic"},
		{"unknown value", FontStyle(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.style.String())
		})
	}
}

func TestFontDescriptor_String(t *testing.T) {
	tests := []struct {
		name       string
		descriptor FontDescriptor
		expected   string
	}{
		{
			"normal weight normal style",
			FontDescriptor{Family: "Helvetica", Weight: 400, Style: FontStyleNormal},
			"Helvetica 400 normal",
		},
		{
			"bold italic",
			FontDescriptor{Family: "Times", Weight: 700, Style: FontStyleItalic},
			"Times 700 italic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.descriptor.String())
		})
	}
}
