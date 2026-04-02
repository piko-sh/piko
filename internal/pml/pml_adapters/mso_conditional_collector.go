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

package pml_adapters

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/pml/pml_domain"
)

// msoConditionalCollector implements the pml_domain.MSOConditionalCollector
// interface. It tracks CSS rules that need to be wrapped in MSO (Microsoft
// Outlook) conditional comments and generates the final conditional block.
//
// The collector is thread-safe to support potential concurrent transformations
// in the future. It automatically deduplicates rules, so multiple
// registrations of the same selector and styles combination only generate one
// CSS rule.
type msoConditionalCollector struct {
	// rules maps CSS selectors to their style values.
	rules map[string]string

	// mu guards concurrent access to the styles map.
	mu sync.RWMutex
}

var _ pml_domain.MSOConditionalCollector = (*msoConditionalCollector)(nil)

// RegisterStyle adds a CSS rule to be wrapped in an MSO conditional comment.
//
// If the same selector is registered multiple times, the last registration
// wins. Components should register consistent styles for each selector.
//
// Takes selector (string) which specifies the CSS selector for the rule.
// Takes styles (string) which contains the CSS properties to apply.
//
// Safe for concurrent use.
func (m *msoConditionalCollector) RegisterStyle(selector string, styles string) {
	if selector == "" || styles == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.rules[selector] = styles
}

// GenerateConditionalBlock produces the final MSO conditional comment block
// with all collected styles. The output follows the standard MSO conditional
// comment format used in email templates.
//
// The generated block follows this structure:
// <!--[if mso]>
// <style type="text/css">
//
//	ul {margin: 0 !important;}
//	li {margin-left: 40px !important;}
//	li.firstListItem {margin-top: 20px !important;}
//	li.lastListItem {margin-bottom: 20px !important;}
//
// </style>
// <![endif]-->
// Selectors are sorted alphabetically for deterministic output.
//
// Returns string which contains the MSO conditional block, or an empty string
// if no rules have been registered.
//
// Thread-safe: Safe to call concurrently with RegisterStyle.
func (m *msoConditionalCollector) GenerateConditionalBlock() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.rules) == 0 {
		return ""
	}

	var builder strings.Builder

	builder.WriteString("<!--[if mso]>\n")
	builder.WriteString("<style type=\"text/css\">\n")

	selectors := make([]string, 0, len(m.rules))
	for selector := range m.rules {
		selectors = append(selectors, selector)
	}

	slices.Sort(selectors)

	for _, selector := range selectors {
		styles := m.rules[selector]
		_, _ = fmt.Fprintf(&builder, "  %s {%s}\n", selector, styles)
	}

	builder.WriteString("</style>\n")
	builder.WriteString("<![endif]-->")

	return builder.String()
}

// NewMSOConditionalCollector creates a new, empty MSOConditionalCollector.
// This should be called once at the start of a transformation pass.
//
// Returns pml_domain.MSOConditionalCollector which is ready for use.
func NewMSOConditionalCollector() pml_domain.MSOConditionalCollector {
	return &msoConditionalCollector{
		rules: make(map[string]string),
		mu:    sync.RWMutex{},
	}
}
