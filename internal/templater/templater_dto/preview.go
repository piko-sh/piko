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

package templater_dto

// PreviewScenario describes a single named preview scenario for a component.
// The Preview() convention function returns a slice of these, each providing
// sample data for rendering the component in the dev tools preview.
type PreviewScenario struct {
	// Props is the sample data to pass to the component's Render function.
	// For partials, emails, and PDFs, this is passed as the props argument
	// to RunPartialWithProps.
	Props any

	// Name is the display name for this scenario (e.g., "empty cart",
	// "with items").
	Name string

	// Description is an optional longer description shown in the dev tools.
	Description string
}
