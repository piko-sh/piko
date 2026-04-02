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

package pikotest_dto

import (
	"piko.sh/piko/internal/render/render_domain"
)

// ComponentOption configures optional behaviour for a ComponentTester.
type ComponentOption func(*ComponentConfig)

// ComponentConfig holds the configuration state for a ComponentTester.
// It is populated by applying ComponentOption functions.
type ComponentConfig struct {
	// Renderer is the service that renders templates to HTML during tests.
	// When nil, HTML() calls on TestView will fail.
	Renderer render_domain.RenderService

	// PageID is the unique identifier for the page, used in error messages
	// and debugging.
	PageID string
}

// DefaultComponentConfig returns a ComponentConfig with sensible defaults.
//
// Returns ComponentConfig which is ready for use with default values.
func DefaultComponentConfig() ComponentConfig {
	return ComponentConfig{
		Renderer: nil,
		PageID:   "test-component",
	}
}
