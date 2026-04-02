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

package tui

// TUIConfig holds TUI settings from piko.yaml.
type TUIConfig struct {
	// Endpoint is the API endpoint URL for the TUI service.
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// RefreshInterval specifies how often the TUI updates; empty uses the default.
	RefreshInterval string `json:"refreshInterval" yaml:"refreshInterval"`

	// Theme specifies the visual theme name for the TUI.
	Theme string `json:"theme" yaml:"theme"`

	// Title is the text displayed in the TUI window title bar.
	Title string `json:"title" yaml:"title"`
}

// pikoConfig is a minimal representation of piko.yaml for extracting TUI settings.
type pikoConfig struct {
	// TUI contains the terminal user interface configuration.
	TUI TUIConfig `json:"tui" yaml:"tui"`
}
