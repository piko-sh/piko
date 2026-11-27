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

//go:build js && wasm

package render_domain

import (
	"piko.sh/piko/internal/daemon/daemon_frontend"
)

// In WASM builds, the frontend module HTML (preload links, config scripts, module
// scripts, dev widget) is not needed. The WASM build of daemon_frontend leaves out the
// heavy embedded bundles, so only the runtime URLs are taken from it.

// getModulePreloadHTML returns empty HTML since preload links are not needed in WASM
// builds.
//
// Returns string which is always empty.
func getModulePreloadHTML() string { return "" }

// getModuleConfigHTML returns empty HTML since the config script is not needed in WASM
// builds.
//
// Returns string which is always empty.
func getModuleConfigHTML() string { return "" }

// getModuleScriptHTML returns empty HTML since the module script is not needed in WASM
// builds.
//
// Returns string which is always empty.
func getModuleScriptHTML() string { return "" }

// getDevWidgetHTML returns empty HTML since the dev widget is not needed in WASM builds.
//
// Returns string which is always empty.
func getDevWidgetHTML() string { return "" }

// getCoreJSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Takes build (daemon_frontend.CoreRuntimeBuild) which is not used.
//
// Returns string which is always empty.
func getCoreJSSRIHash(daemon_frontend.CoreRuntimeBuild) string { return "" }

// getActionsJSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Returns string which is always empty.
func getActionsJSSRIHash() string { return "" }

// getThemeCSSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Returns string which is always empty.
func getThemeCSSSRIHash() string { return "" }

// getCoreJSURL returns the URL generated modules import the core framework JS module
// from. The playground maps that URL to its in-browser copy of the runtime, so the page
// must load the same one.
//
// Takes build (daemon_frontend.CoreRuntimeBuild) which is not used, as the page always
// loads the URL that generated modules import.
//
// Returns string which holds the URL the page loads the runtime from.
func getCoreJSURL(daemon_frontend.CoreRuntimeBuild) string {
	return daemon_frontend.CoreModuleImportURL()
}

// getComponentsJSURL returns the URL compiled components import the PKC components JS
// module from, which the playground maps to its in-browser copy.
//
// Takes build (daemon_frontend.CoreRuntimeBuild) which is not used, as the page always
// loads the URL that compiled components import.
//
// Returns string which holds the unversioned URL of the module.
func getComponentsJSURL(daemon_frontend.CoreRuntimeBuild) string {
	return daemon_frontend.ComponentsModuleImportURL()
}

// getRuntimeRecoveryURL returns empty, since the playground has no server to serve the
// runtime recovery script.
//
// Returns string which is always empty.
func getRuntimeRecoveryURL() string { return "" }
