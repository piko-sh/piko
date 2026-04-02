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

// In WASM builds, the frontend module HTML (preload links, config scripts,
// module scripts, dev widget) is not needed. The daemon_frontend package
// embeds ~4.5MB of ppframework JS bundles that would bloat the WASM binary.

// getModulePreloadHTML returns empty HTML since preload
// links are not needed in WASM builds.
//
// Returns string which is always empty.
func getModulePreloadHTML() string { return "" }

// getModuleConfigHTML returns empty HTML since the config
// script is not needed in WASM builds.
//
// Returns string which is always empty.
func getModuleConfigHTML() string { return "" }

// getModuleScriptHTML returns empty HTML since the module
// script is not needed in WASM builds.
//
// Returns string which is always empty.
func getModuleScriptHTML() string { return "" }

// getDevWidgetHTML returns empty HTML since the dev widget
// is not needed in WASM builds.
//
// Returns string which is always empty.
func getDevWidgetHTML() string { return "" }

// getCoreJSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Returns string which is always empty.
func getCoreJSSRIHash() string { return "" }

// getActionsJSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Returns string which is always empty.
func getActionsJSSRIHash() string { return "" }

// getThemeCSSSRIHash returns empty since SRI is not needed in WASM builds.
//
// Returns string which is always empty.
func getThemeCSSSRIHash() string { return "" }
