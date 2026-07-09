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

package pdfwriter_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildWatermarkStream_ContainsExpectedOperators(t *testing.T) {
	wm := &WatermarkConfig{
		Text:     "DRAFT",
		FontSize: 60,
		ColourR:  0.85,
		ColourG:  0.85,
		ColourB:  0.85,
		Angle:    45,
		Opacity:  0.3,
	}
	result := buildWatermarkStream(wm, "FW", "GS1", 595.28, 841.89)

	assert.Contains(t, result, "q\n", "expected SaveState (q)")
	assert.Contains(t, result, "/GS1 gs\n", "expected ExtGState reference")
	assert.Contains(t, result, "BT\n", "expected BeginText")
	assert.Contains(t, result, "/FW 60 Tf\n", "expected font selection")
	assert.Contains(t, result, "0.85 0.85 0.85 rg\n", "expected fill colour")
	assert.Contains(t, result, "cm\n", "expected transformation matrix")
	assert.Contains(t, result, "(DRAFT) Tj\n", "expected text showing")
	assert.Contains(t, result, "ET\n", "expected EndText")
	assert.Contains(t, result, "Q\n", "expected RestoreState (Q)")
}

func TestWatermarkConfig_ApplyDefaults(t *testing.T) {
	wm := &WatermarkConfig{Text: "TEST"}
	wm.applyDefaults()

	assert.Equal(t, 60.0, wm.FontSize, "expected FontSize 60")
	assert.Equal(t, 0.85, wm.ColourR, "expected ColourR 0.85")
	assert.Equal(t, 45.0, wm.Angle, "expected Angle 45")
	assert.Equal(t, 0.3, wm.Opacity, "expected Opacity 0.3")
}

func TestWatermarkConfig_PreservesCustomValues(t *testing.T) {
	wm := &WatermarkConfig{
		Text:     "CUSTOM",
		FontSize: 80,
		ColourR:  1,
		ColourG:  0,
		ColourB:  0,
		Angle:    30,
		Opacity:  0.5,
	}
	wm.applyDefaults()

	assert.Equal(t, 80.0, wm.FontSize, "expected FontSize 80")
	assert.Equal(t, 1.0, wm.ColourR, "expected custom colour preserved")
	assert.Equal(t, 0.0, wm.ColourG, "expected custom colour preserved")
	assert.Equal(t, 30.0, wm.Angle, "expected Angle 30")
	assert.Equal(t, 0.5, wm.Opacity, "expected Opacity 0.5")
}
