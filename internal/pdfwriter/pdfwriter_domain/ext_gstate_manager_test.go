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

func TestNewExtGStateManager_HasNoStatesInitially(t *testing.T) {
	manager := NewExtGStateManager()

	if manager.HasStates() {
		t.Error("expected HasStates to return false for a new manager")
	}
}

func TestRegisterOpacity_ReturnsGS1ForFirstRegistration(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterOpacity(0.5)

	if name != "GS1" {
		t.Errorf("expected first registration to return \"GS1\", got %q", name)
	}
}

func TestRegisterOpacity_DeduplicatesSameOpacity(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterOpacity(0.75)
	second_name := manager.RegisterOpacity(0.75)

	if first_name != second_name {
		t.Errorf("expected duplicate registration to return %q, got %q", first_name, second_name)
	}
}

func TestRegisterOpacity_DifferentValuesGetDifferentNames(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterOpacity(0.5)
	second_name := manager.RegisterOpacity(0.8)

	if first_name != "GS1" {
		t.Errorf("expected first registration to return \"GS1\", got %q", first_name)
	}
	if second_name != "GS2" {
		t.Errorf("expected second registration to return \"GS2\", got %q", second_name)
	}
}

func TestHasStates_ReturnsTrueAfterRegistration(t *testing.T) {
	manager := NewExtGStateManager()

	manager.RegisterOpacity(0.5)

	if !manager.HasStates() {
		t.Error("expected HasStates to return true after registering an opacity")
	}
}

func TestWriteObjects_WritesCorrectPdfObjects(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.5)

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/GS1") {
		t.Error("expected entries to contain \"/GS1\"")
	}
	if !strings.Contains(entries, "0 R") {
		t.Error("expected entries to contain an object reference (\"0 R\")")
	}

	output := writer.buffer.String()

	if !strings.Contains(output, "/Type /ExtGState") {
		t.Error("expected PDF output to contain \"/Type /ExtGState\"")
	}
	if !strings.Contains(output, "/ca 0.50") {
		t.Error("expected PDF output to contain \"/ca 0.50\"")
	}
	if !strings.Contains(output, "/CA 0.50") {
		t.Error("expected PDF output to contain \"/CA 0.50\"")
	}
}

func TestWriteObjects_HandlesMultipleStates(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.3)
	manager.RegisterOpacity(1.0)

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/GS1") {
		t.Error("expected entries to contain \"/GS1\"")
	}
	if !strings.Contains(entries, "/GS2") {
		t.Error("expected entries to contain \"/GS2\"")
	}

	output := writer.buffer.String()

	if !strings.Contains(output, "/ca 0.30") {
		t.Error("expected PDF output to contain \"/ca 0.30\"")
	}
	if !strings.Contains(output, "/ca 1") {
		t.Error("expected PDF output to contain \"/ca 1\"")
	}
}

func TestRegisterBlendMode_ReturnsName(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterBlendMode("Multiply")

	if name != "GS1" {
		t.Errorf("expected \"GS1\", got %q", name)
	}
}

func TestRegisterBlendMode_DeduplicatesSameMode(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterBlendMode("Screen")
	second_name := manager.RegisterBlendMode("Screen")

	if first_name != second_name {
		t.Errorf("expected duplicate to return %q, got %q", first_name, second_name)
	}
}

func TestRegisterBlendMode_WritesBMInOutput(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterBlendMode("Multiply")

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/GS1") {
		t.Error("expected entries to contain \"/GS1\"")
	}

	output := writer.buffer.String()
	if !strings.Contains(output, "/BM /Multiply") {
		t.Errorf("expected /BM /Multiply in output, got %q", output)
	}
}

func TestRegisterBlendMode_DoesNotWriteOpacity(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterBlendMode("Overlay")

	writer := &PdfDocumentWriter{}
	manager.WriteObjects(writer)

	output := writer.buffer.String()
	if strings.Contains(output, "/ca") {
		t.Errorf("expected no /ca in blend-mode-only state, got %q", output)
	}
}

func TestRegisterSoftMask_ReturnsName(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterSoftMask(42)

	if name != "GS1" {
		t.Errorf("expected \"GS1\", got %q", name)
	}
	if !manager.HasStates() {
		t.Error("expected HasStates to return true after RegisterSoftMask")
	}
}

func TestRegisterSoftMask_WritesSMaskInOutput(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterSoftMask(42)

	writer := &PdfDocumentWriter{}
	writer.AllocateObject()
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/GS1") {
		t.Error("expected entries to contain \"/GS1\"")
	}

	output := writer.buffer.String()
	if !strings.Contains(output, "/SMask") {
		t.Errorf("expected /SMask in output, got %q", output)
	}
	if !strings.Contains(output, "/Luminosity") {
		t.Errorf("expected /Luminosity in output, got %q", output)
	}
	if !strings.Contains(output, "42 0 R") {
		t.Errorf("expected reference to object 42 in output, got %q", output)
	}
}

func TestRegisterSoftMask_DoesNotWriteOpacity(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterSoftMask(10)

	writer := &PdfDocumentWriter{}
	manager.WriteObjects(writer)

	output := writer.buffer.String()
	if strings.Contains(output, "/ca") {
		t.Errorf("expected no /ca in soft-mask-only state, got %q", output)
	}
}

func TestMixedOpacityAndBlendMode(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.5)
	manager.RegisterBlendMode("Screen")

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	if !strings.Contains(entries, "/GS1") {
		t.Error("expected /GS1")
	}
	if !strings.Contains(entries, "/GS2") {
		t.Error("expected /GS2")
	}

	output := writer.buffer.String()
	if !strings.Contains(output, "/ca 0.50") {
		t.Error("expected /ca 0.50")
	}
	if !strings.Contains(output, "/BM /Screen") {
		t.Error("expected /BM /Screen")
	}
}
