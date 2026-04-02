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

// Manages ExtGState (extended graphics state) resources for PDF opacity.
// Follows the same register-track-write lifecycle as FontEmbedder.

import (
	"fmt"
	"slices"
	"strings"
)

// extGStateEntry holds the properties for a single ExtGState dictionary.
type extGStateEntry struct {
	// blendMode holds the PDF blend mode name (e.g. "Multiply").
	blendMode string

	// softMaskSubtype holds the soft mask subtype (e.g. "Luminosity").
	softMaskSubtype string

	// opacity holds the opacity value in [0, 1].
	opacity float64

	// softMaskRef holds the object number of the transparency group XObject.
	softMaskRef int

	// hasOpacity indicates whether this entry carries an explicit opacity value.
	hasOpacity bool
}

// ExtGStateManager tracks graphics state parameter dictionaries needed
// for opacity, blend modes, and other graphics state properties, and
// writes them as PDF objects.
type ExtGStateManager struct {
	// states maps resource names (GS1, GS2, ...) to their entries.
	states map[string]extGStateEntry

	// opacityToName deduplicates opacity-only registrations by mapping
	// opacity values to their existing resource names.
	opacityToName map[float64]string

	// blendModeToName deduplicates blend-mode-only registrations.
	blendModeToName map[string]string

	// nextIndex is the counter for generating resource names.
	nextIndex int
}

// NewExtGStateManager creates a new ExtGState manager.
//
// Returns *ExtGStateManager ready to accept opacity registrations.
func NewExtGStateManager() *ExtGStateManager {
	return &ExtGStateManager{
		states:          make(map[string]extGStateEntry),
		opacityToName:   make(map[float64]string),
		blendModeToName: make(map[string]string),
	}
}

// RegisterOpacity registers an opacity value and returns its resource
// name (e.g. "GS1").
//
// If the same opacity has already been registered, returns the existing name.
//
// Takes opacity (float64) which is the opacity value in [0, 1].
//
// Returns string which is the resource name for use in SetExtGState.
func (m *ExtGStateManager) RegisterOpacity(opacity float64) string {
	if name, exists := m.opacityToName[opacity]; exists {
		return name
	}

	m.nextIndex++
	name := fmt.Sprintf("GS%d", m.nextIndex)
	m.states[name] = extGStateEntry{opacity: opacity, hasOpacity: true}
	m.opacityToName[opacity] = name
	return name
}

// RegisterBlendMode registers a blend mode and returns its resource
// name. Deduplicates: if the same blend mode has already been
// registered, returns the existing name.
//
// Takes blendMode (string) which is the PDF blend mode name (e.g.
// "Multiply", "Screen").
//
// Returns string which is the resource name for use in SetExtGState.
func (m *ExtGStateManager) RegisterBlendMode(blendMode string) string {
	if name, exists := m.blendModeToName[blendMode]; exists {
		return name
	}

	m.nextIndex++
	name := fmt.Sprintf("GS%d", m.nextIndex)
	m.states[name] = extGStateEntry{opacity: 1.0, blendMode: blendMode}
	m.blendModeToName[blendMode] = name
	return name
}

// RegisterSoftMask registers a luminosity soft mask that references a
// transparency group form XObject.
//
// Takes groupObjectNumber (int) which is the object number of the
// transparency group form XObject to use as the mask.
//
// Returns string which is the resource name (e.g. "GS3").
func (m *ExtGStateManager) RegisterSoftMask(groupObjectNumber int) string {
	m.nextIndex++
	name := fmt.Sprintf("GS%d", m.nextIndex)
	m.states[name] = extGStateEntry{
		softMaskRef:     groupObjectNumber,
		softMaskSubtype: "Luminosity",
	}
	return name
}

// HasStates reports whether any graphics states have been registered.
//
// Returns bool which is true if at least one state exists.
func (m *ExtGStateManager) HasStates() bool {
	return len(m.states) > 0
}

// WriteObjects writes all registered ExtGState dictionaries as PDF
// objects and returns the resource entries string for inclusion in
// the page /Resources /ExtGState dictionary.
//
// Takes writer (*PdfDocumentWriter) which receives the PDF objects.
//
// Returns string with entries like " /GS1 5 0 R /GS2 6 0 R".
func (m *ExtGStateManager) WriteObjects(writer *PdfDocumentWriter) string {
	names := make([]string, 0, len(m.states))
	for name := range m.states {
		names = append(names, name)
	}
	slices.Sort(names)

	var entries strings.Builder
	for _, name := range names {
		entry := m.states[name]
		objectNumber := writer.AllocateObject()
		var dict strings.Builder
		dict.WriteString("<< /Type /ExtGState")
		if entry.hasOpacity {
			fmt.Fprintf(&dict, " /ca %s /CA %s",
				FormatNumber(entry.opacity), FormatNumber(entry.opacity))
		}
		if entry.blendMode != "" {
			fmt.Fprintf(&dict, " /BM /%s", entry.blendMode)
		}
		if entry.softMaskRef > 0 {
			fmt.Fprintf(&dict, " /SMask << /S /%s /G %s >>",
				entry.softMaskSubtype, FormatReference(entry.softMaskRef))
		}
		dict.WriteString(" >>")
		writer.WriteObject(objectNumber, dict.String())
		fmt.Fprintf(&entries, " /%s %s", name, FormatReference(objectNumber))
	}
	return entries.String()
}
