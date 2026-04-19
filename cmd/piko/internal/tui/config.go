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

// Config holds TUI settings loaded from tui.yaml or PIKO_TUI_* environment
// variables. The config_domain loader populates the fields using the struct
// tags below.
type Config struct {
	// Endpoint is the API endpoint URL for the TUI service.
	Endpoint string `json:"endpoint" yaml:"endpoint" env:"PIKO_TUI_ENDPOINT" default:"http://localhost:8080"`

	// RefreshInterval specifies how often the TUI updates; empty uses the default.
	RefreshInterval string `json:"refreshInterval" yaml:"refreshInterval" env:"PIKO_TUI_REFRESH_INTERVAL" default:"2s"`

	// Theme specifies the visual theme name for the TUI.
	Theme string `json:"theme" yaml:"theme" env:"PIKO_TUI_THEME" default:"default"`

	// Title is the text displayed in the TUI window title bar.
	Title string `json:"title" yaml:"title" env:"PIKO_TUI_TITLE"`
}
