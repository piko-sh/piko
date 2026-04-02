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
	"strings"
	"testing"
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

	if !strings.Contains(result, "q\n") {
		t.Error("expected SaveState (q)")
	}
	if !strings.Contains(result, "/GS1 gs\n") {
		t.Error("expected ExtGState reference")
	}
	if !strings.Contains(result, "BT\n") {
		t.Error("expected BeginText")
	}
	if !strings.Contains(result, "/FW 60 Tf\n") {
		t.Error("expected font selection")
	}
	if !strings.Contains(result, "0.85 0.85 0.85 rg\n") {
		t.Error("expected fill colour")
	}
	if !strings.Contains(result, "cm\n") {
		t.Error("expected transformation matrix")
	}
	if !strings.Contains(result, "(DRAFT) Tj\n") {
		t.Error("expected text showing")
	}
	if !strings.Contains(result, "ET\n") {
		t.Error("expected EndText")
	}
	if !strings.Contains(result, "Q\n") {
		t.Error("expected RestoreState (Q)")
	}
}

func TestWatermarkConfig_ApplyDefaults(t *testing.T) {
	wm := &WatermarkConfig{Text: "TEST"}
	wm.applyDefaults()

	if wm.FontSize != 60 {
		t.Errorf("expected FontSize 60, got %v", wm.FontSize)
	}
	if wm.ColourR != 0.85 {
		t.Errorf("expected ColourR 0.85, got %v", wm.ColourR)
	}
	if wm.Angle != 45 {
		t.Errorf("expected Angle 45, got %v", wm.Angle)
	}
	if wm.Opacity != 0.3 {
		t.Errorf("expected Opacity 0.3, got %v", wm.Opacity)
	}
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

	if wm.FontSize != 80 {
		t.Errorf("expected FontSize 80, got %v", wm.FontSize)
	}
	if wm.ColourR != 1 || wm.ColourG != 0 {
		t.Errorf("expected custom colour preserved, got R=%v G=%v", wm.ColourR, wm.ColourG)
	}
	if wm.Angle != 30 {
		t.Errorf("expected Angle 30, got %v", wm.Angle)
	}
	if wm.Opacity != 0.5 {
		t.Errorf("expected Opacity 0.5, got %v", wm.Opacity)
	}
}
