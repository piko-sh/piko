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

package casefile

// DiagnosticPair pairs valid and invalid sources with an expected diagnostic.
type DiagnosticPair struct {
	// ID identifies the diagnostic pair.
	ID string

	// GoodPath is the source path that compiles cleanly.
	GoodPath string

	// BadPath is the source path that produces the diagnostic.
	BadPath string

	// Code is the expected diagnostic code.
	Code string

	// Expression is the diagnostic's source expression.
	Expression string `json:",omitempty"`

	// SectionPath names the source used for section-coordinate checks.
	SectionPath string `json:",omitempty"`

	// Sections holds expected source sections in document order.
	Sections []SourceSection `json:",omitempty"`

	// Line is the expected one-based diagnostic line.
	Line int

	// Column is the expected one-based Unicode column.
	Column int `json:",omitempty"`

	// ExpressionOffset is the expression's byte offset.
	ExpressionOffset int `json:",omitempty"`
}

// SourceSection records absolute byte offsets for a source section.
type SourceSection struct {
	// Attributes holds the element's public attributes.
	Attributes map[string]string

	// Tag is the source block tag.
	Tag string

	// Start is the section's absolute opening byte offset.
	Start int

	// ContentStart is the absolute byte offset where the section's content begins.
	ContentStart int

	// ContentEnd is the absolute byte offset where the content ends, before the closing tag.
	ContentEnd int
}
