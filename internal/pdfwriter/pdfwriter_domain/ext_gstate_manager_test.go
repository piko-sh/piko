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

func TestNewExtGStateManager_HasNoStatesInitially(t *testing.T) {
	manager := NewExtGStateManager()

	assert.False(t, manager.HasStates(), "expected HasStates to return false for a new manager")
}

func TestRegisterOpacity_ReturnsGS1ForFirstRegistration(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterOpacity(0.5)

	assert.Equal(t, "GS1", name, "expected first registration to return \"GS1\"")
}

func TestRegisterOpacity_DeduplicatesSameOpacity(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterOpacity(0.75)
	second_name := manager.RegisterOpacity(0.75)

	assert.Equal(t, first_name, second_name, "expected duplicate registration to return the same name")
}

func TestRegisterOpacity_DifferentValuesGetDifferentNames(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterOpacity(0.5)
	second_name := manager.RegisterOpacity(0.8)

	assert.Equal(t, "GS1", first_name, "expected first registration to return \"GS1\"")
	assert.Equal(t, "GS2", second_name, "expected second registration to return \"GS2\"")
}

func TestHasStates_ReturnsTrueAfterRegistration(t *testing.T) {
	manager := NewExtGStateManager()

	manager.RegisterOpacity(0.5)

	assert.True(t, manager.HasStates(), "expected HasStates to return true after registering an opacity")
}

func TestWriteObjects_WritesCorrectPdfObjects(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.5)

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/GS1", "expected entries to contain \"/GS1\"")
	assert.Contains(t, entries, "0 R", "expected entries to contain an object reference (\"0 R\")")

	output := writer.buffer.String()

	assert.Contains(t, output, "/Type /ExtGState", "expected PDF output to contain \"/Type /ExtGState\"")
	assert.Contains(t, output, "/ca 0.50", "expected PDF output to contain \"/ca 0.50\"")
	assert.Contains(t, output, "/CA 0.50", "expected PDF output to contain \"/CA 0.50\"")
}

func TestWriteObjects_HandlesMultipleStates(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.3)
	manager.RegisterOpacity(1.0)

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/GS1", "expected entries to contain \"/GS1\"")
	assert.Contains(t, entries, "/GS2", "expected entries to contain \"/GS2\"")

	output := writer.buffer.String()

	assert.Contains(t, output, "/ca 0.30", "expected PDF output to contain \"/ca 0.30\"")
	assert.Contains(t, output, "/ca 1", "expected PDF output to contain \"/ca 1\"")
}

func TestRegisterBlendMode_ReturnsName(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterBlendMode("Multiply")

	assert.Equal(t, "GS1", name)
}

func TestRegisterBlendMode_DeduplicatesSameMode(t *testing.T) {
	manager := NewExtGStateManager()

	first_name := manager.RegisterBlendMode("Screen")
	second_name := manager.RegisterBlendMode("Screen")

	assert.Equal(t, first_name, second_name, "expected duplicate to return the same name")
}

func TestRegisterBlendMode_WritesBMInOutput(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterBlendMode("Multiply")

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/GS1", "expected entries to contain \"/GS1\"")

	output := writer.buffer.String()
	assert.Contains(t, output, "/BM /Multiply", "expected /BM /Multiply in output")
}

func TestRegisterBlendMode_DoesNotWriteOpacity(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterBlendMode("Overlay")

	writer := &PdfDocumentWriter{}
	manager.WriteObjects(writer)

	output := writer.buffer.String()
	assert.NotContains(t, output, "/ca", "expected no /ca in blend-mode-only state")
}

func TestRegisterSoftMask_ReturnsName(t *testing.T) {
	manager := NewExtGStateManager()

	name := manager.RegisterSoftMask(42)

	assert.Equal(t, "GS1", name)
	assert.True(t, manager.HasStates(), "expected HasStates to return true after RegisterSoftMask")
}

func TestRegisterSoftMask_WritesSMaskInOutput(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterSoftMask(42)

	writer := &PdfDocumentWriter{}
	writer.AllocateObject()
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/GS1", "expected entries to contain \"/GS1\"")

	output := writer.buffer.String()
	assert.Contains(t, output, "/SMask", "expected /SMask in output")
	assert.Contains(t, output, "/Luminosity", "expected /Luminosity in output")
	assert.Contains(t, output, "42 0 R", "expected reference to object 42 in output")
}

func TestRegisterSoftMask_DoesNotWriteOpacity(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterSoftMask(10)

	writer := &PdfDocumentWriter{}
	manager.WriteObjects(writer)

	output := writer.buffer.String()
	assert.NotContains(t, output, "/ca", "expected no /ca in soft-mask-only state")
}

func TestMixedOpacityAndBlendMode(t *testing.T) {
	manager := NewExtGStateManager()
	manager.RegisterOpacity(0.5)
	manager.RegisterBlendMode("Screen")

	writer := &PdfDocumentWriter{}
	entries := manager.WriteObjects(writer)

	assert.Contains(t, entries, "/GS1", "expected /GS1")
	assert.Contains(t, entries, "/GS2", "expected /GS2")

	output := writer.buffer.String()
	assert.Contains(t, output, "/ca 0.50", "expected /ca 0.50")
	assert.Contains(t, output, "/BM /Screen", "expected /BM /Screen")
}
