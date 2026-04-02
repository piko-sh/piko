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

//go:build !js

package render_domain

import "piko.sh/piko/internal/daemon/daemon_frontend"

// getModulePreloadHTML returns the HTML preload link tags
// for the frontend module.
//
// Returns string which holds the preload link HTML.
func getModulePreloadHTML() string { return daemon_frontend.GetModulePreloadHTML() }

// getModuleConfigHTML returns the HTML configuration script
// tag for the frontend module.
//
// Returns string which holds the config script HTML.
func getModuleConfigHTML() string { return daemon_frontend.GetModuleConfigHTML() }

// getModuleScriptHTML returns the HTML script tag that loads
// the frontend module.
//
// Returns string which holds the module script HTML.
func getModuleScriptHTML() string { return daemon_frontend.GetModuleScriptHTML() }

// getDevWidgetHTML returns the HTML for the development
// widget overlay.
//
// Returns string which holds the dev widget HTML.
func getDevWidgetHTML() string { return daemon_frontend.GetDevWidgetHTML() }

// getCoreJSSRIHash returns the SRI hash for the core framework JS module.
//
// Returns string which holds the integrity hash, or empty if SRI is disabled.
func getCoreJSSRIHash() string {
	return daemon_frontend.GetSRIHash("built/ppframework.core.es.js")
}

// getActionsJSSRIHash returns the SRI hash for the actions.gen.js module.
// This returns a hash stored via SetSRIHash when the actions JS artefact is
// registered by the build pipeline.
//
// Returns string which holds the integrity hash, or empty if SRI is disabled
// or the hash has not been computed.
func getActionsJSSRIHash() string {
	return daemon_frontend.GetSRIHash("pk-js/pk/actions.gen.js")
}

// getThemeCSSSRIHash returns the SRI hash for the theme.css stylesheet.
// This returns a hash stored via SetSRIHash when the theme CSS artefact is
// registered.
//
// Returns string which holds the integrity hash, or empty if SRI is disabled
// or the hash has not been computed.
func getThemeCSSSRIHash() string {
	return daemon_frontend.GetSRIHash("theme.css")
}
