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
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

	"piko.sh/piko/internal/pml/pml_domain"
)

// mediaQueryCollector implements the pml_domain.MediaQueryCollector
// interface. It tracks CSS classes that need responsive media queries and
// generates the final CSS.
//
// The collector is thread-safe to support potential concurrent transformations
// in the future. It automatically deduplicates classes, so multiple columns
// with "pml-col-50" only generate one media query entry.
type mediaQueryCollector struct {
	// classes maps class names to mobile styles for standard responsive classes
	// such as column widths.
	classes map[string]string

	// fluidClasses maps class names to mobile styles for classes that need
	// fluid behaviour on mobile, such as full-width images.
	fluidClasses map[string]string

	// mu guards concurrent access to the class maps.
	mu sync.RWMutex
}

var _ pml_domain.MediaQueryCollector = (*mediaQueryCollector)(nil)

// RegisterClass adds a CSS class that needs a mobile-stacking media query.
// If the same class name is registered more than once, only one entry is kept,
// which provides automatic deduplication.
//
// Takes className (string) which is the CSS class name to register.
// Takes mobileStyles (string) which contains the mobile-specific CSS rules.
//
// Safe for concurrent use from multiple goroutines.
func (m *mediaQueryCollector) RegisterClass(className string, mobileStyles string) {
	if className == "" || mobileStyles == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.classes[className] = mobileStyles
}

// RegisterFluidClass adds a CSS class for fluid-on-mobile images and
// elements.
//
// This is the same as RegisterClass but kept separate for clarity. Used by
// <pml-img> for the "pml-fluid-mobile" class.
//
// Takes className (string) which specifies the CSS class name to register.
// Takes mobileStyles (string) which provides the CSS styles for mobile view.
//
// Safe for concurrent use.
func (m *mediaQueryCollector) RegisterFluidClass(className string, mobileStyles string) {
	if className == "" || mobileStyles == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.fluidClasses[className] = mobileStyles
}

// GenerateCSS produces the final style block with all media queries.
//
// Takes breakpoint (string) which sets the max-width for mobile targeting
// (e.g. "480px"). Defaults to "480px" if empty.
//
// Returns string which contains the generated CSS media query block. Returns
// an empty string if no classes have been registered.
//
// The generated CSS follows this pattern:
//
//	@media only screen and (max-width: 480px) {
//	  .pml-col-50 { width: 100% !important; }
//	  .pml-fluid-mobile { width: 100% !important; }
//	}
//
// Classes are sorted alphabetically for deterministic output.
//
// Safe for concurrent use with RegisterClass and RegisterFluidClass.
// Protected by a read lock on the collector's mutex.
func (m *mediaQueryCollector) GenerateCSS(breakpoint string) string {
	breakpoint = cmp.Or(breakpoint, "480px")

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.classes) == 0 && len(m.fluidClasses) == 0 {
		return ""
	}

	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "@media only screen and (max-width: %s) {\n", breakpoint)

	allClassNames := make([]string, 0, len(m.classes)+len(m.fluidClasses))
	for className := range m.classes {
		allClassNames = append(allClassNames, className)
	}
	for className := range m.fluidClasses {
		allClassNames = append(allClassNames, className)
	}

	slices.Sort(allClassNames)

	for _, className := range allClassNames {
		var mobileStyles string
		var found bool

		if styles, ok := m.classes[className]; ok {
			mobileStyles = styles
			found = true
		} else if styles, ok := m.fluidClasses[className]; ok {
			mobileStyles = styles
			found = true
		}

		if found {
			if strings.Contains(className, ".") {
				_, _ = fmt.Fprintf(&builder, "  %s { %s }\n", className, mobileStyles)
			} else {
				_, _ = fmt.Fprintf(&builder, "  .%s { %s }\n", className, mobileStyles)
			}
		}
	}

	builder.WriteString("}")

	return builder.String()
}

// NewMediaQueryCollector creates a new, empty MediaQueryCollector.
// This should be called once at the start of a transformation pass.
//
// Returns pml_domain.MediaQueryCollector which is ready to collect media
// queries during transformation.
func NewMediaQueryCollector() pml_domain.MediaQueryCollector {
	return &mediaQueryCollector{
		classes:      make(map[string]string),
		fluidClasses: make(map[string]string),
		mu:           sync.RWMutex{},
	}
}
