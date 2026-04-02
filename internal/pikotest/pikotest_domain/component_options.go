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

package pikotest_domain

import (
	"piko.sh/piko/internal/pikotest/pikotest_dto"
	"piko.sh/piko/internal/render/render_domain"
)

// WithRenderer attaches a RenderService to enable full HTML rendering.
//
// Takes renderer (render_domain.RenderService) which provides the HTML
// rendering capability.
//
// Returns pikotest_dto.ComponentOption which configures the component to use
// the given renderer.
//
// Without this option, HTML() calls on TestView will fail. Most tests should
// use AST queries instead, which do not require a renderer.
func WithRenderer(renderer render_domain.RenderService) pikotest_dto.ComponentOption {
	return func(config *pikotest_dto.ComponentConfig) {
		config.Renderer = renderer
	}
}

// WithPageID sets the page identifier for this component test. This is used
// mainly for error messages and debugging.
//
// Takes pageID (string) which specifies the identifier for the page.
//
// Returns pikotest_dto.ComponentOption which configures the tester with the
// page ID.
func WithPageID(pageID string) pikotest_dto.ComponentOption {
	return func(config *pikotest_dto.ComponentConfig) {
		config.PageID = pageID
	}
}
