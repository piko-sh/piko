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

package daemon_frontend

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"piko.sh/piko/internal/json"
)

// FrontendModule represents a built-in Piko frontend module
// that implements fmt.Stringer. These are optional JavaScript
// bundles that add extra features such as analytics, modals,
// or toasts to the core framework.
type FrontendModule uint8

const (
	// ModuleAnalytics is a frontend module that adds Google
	// Analytics (GA4) support.
	ModuleAnalytics FrontendModule = iota + 1

	// ModuleModals provides helper functions for modal dialogues. Exports:
	// showModal, closeModal, updateModal, and reloadPartial.
	ModuleModals

	// ModuleToasts provides toast notification helpers. Exports:
	// showToast(message, variant, duration).
	ModuleToasts

	// ModuleDev provides dev tools including auto-refresh on rebuild via SSE.
	// Only loaded in dev and dev-i modes.
	ModuleDev

	// ModuleComponents provides the PKC component runtime (PPElement, VDOM,
	// reactivity). Unlike other modules, this is not injected site-wide; it
	// is loaded on demand when a page contains PKC components.
	ModuleComponents
)

// AnalyticsConfig holds settings for the analytics module.
// It supports Google Analytics (GA4) direct integration and/or
// Google Tag Manager (GTM) container loading.
type AnalyticsConfig struct {
	// GTMContainerID is a Google Tag Manager container ID (e.g., "GTM-XXXXXXX").
	// When set, the GTM container script is loaded and SPA navigation events
	// are pushed to the dataLayer for GTM triggers to consume.
	GTMContainerID string `json:"gtmContainerId,omitempty"`

	// TrackingIDs holds GA4 measurement IDs (e.g., "G-XXXXXXXXXX").
	// Multiple IDs allow sending data to more than one property.
	TrackingIDs []string `json:"trackingIds"`

	// DebugMode enables detailed console logging for debugging.
	DebugMode bool `json:"debugMode,omitempty"`

	// AnonymiseIP hides user IP addresses before sending data
	// to Google Analytics.
	AnonymiseIP bool `json:"anonymiseIp,omitempty"`

	// DisablePageView stops automatic page view tracking when users move
	// between pages. Set to true to track page views yourself.
	DisablePageView bool `json:"disablePageView,omitempty"`
}

// ModalsConfig provides settings for the Modals module. All
// fields default to false, and the frontend applies its own
// defaults when fields are not set, so only set fields you
// want to change.
type ModalsConfig struct {
	// DisableCloseOnEscape stops modals from closing when the
	// Escape key is pressed. When false (the default), pressing
	// Escape closes the modal.
	DisableCloseOnEscape bool `json:"disableCloseOnEscape,omitempty"`

	// DisableCloseOnBackdrop stops modals from closing when
	// clicking the backdrop. When false (the default), clicking
	// the backdrop closes the modal.
	DisableCloseOnBackdrop bool `json:"disableCloseOnBackdrop,omitempty"`
}

// ToastsConfig provides settings for the Toasts module.
type ToastsConfig struct {
	// Position sets where toasts appear on the screen. Valid
	// values are "top-right", "top-left", "bottom-right",
	// "bottom-left", "top-centre", and "bottom-centre".
	Position string `json:"position,omitempty"`

	// DefaultDuration is the display time in milliseconds;
	// default is 5000.
	DefaultDuration int `json:"defaultDuration,omitempty"`

	// MaxVisible sets the maximum number of toasts shown at
	// once; default is 5.
	MaxVisible int `json:"maxVisible,omitempty"`
}

// ModuleEntry holds a built-in frontend module with its optional settings.
type ModuleEntry struct {
	// Config is the strongly-typed config (AnalyticsConfig,
	// ModalsConfig, etc.) or nil.
	Config any

	// Module is the built-in frontend module type.
	Module FrontendModule
}

// String returns the module name for logging and display.
//
// Returns string which is the name of the module in a readable form.
func (m FrontendModule) String() string {
	switch m {
	case ModuleAnalytics:
		return "analytics"
	case ModuleModals:
		return "modals"
	case ModuleToasts:
		return "toasts"
	case ModuleDev:
		return "dev"
	case ModuleComponents:
		return "components"
	default:
		return "unknown"
	}
}

// AssetPath returns the path to the module's JS file in the embedded asset
// store.
//
// Returns string which is the asset path, or an empty string for unknown
// modules.
func (m FrontendModule) AssetPath() string {
	switch m {
	case ModuleAnalytics:
		return "built/ppframework.analytics.min.es.js"
	case ModuleModals:
		return "built/ppframework.modals.min.es.js"
	case ModuleToasts:
		return "built/ppframework.toasts.min.es.js"
	case ModuleDev:
		return "built/ppframework.dev.min.es.js"
	case ModuleComponents:
		return "built/ppframework.components.min.es.js"
	default:
		return ""
	}
}

// ServeURL returns the URL at which this module is served to browsers.
// This is used for generating <script> and <link rel="modulepreload"> tags.
//
// Returns string which is the URL path, or empty if the module is unknown.
func (m FrontendModule) ServeURL() string {
	switch m {
	case ModuleAnalytics:
		return "/_piko/dist/ppframework.analytics.min.es.js"
	case ModuleModals:
		return "/_piko/dist/ppframework.modals.min.es.js"
	case ModuleToasts:
		return "/_piko/dist/ppframework.toasts.min.es.js"
	case ModuleDev:
		return "/_piko/dist/ppframework.dev.min.es.js"
	case ModuleComponents:
		return "/_piko/dist/ppframework.components.min.es.js"
	default:
		return ""
	}
}

// CustomFrontendModule represents a user-provided frontend
// JavaScript module. The content is provided at compile time
// via go:embed and registered at startup.
type CustomFrontendModule struct {
	// Config holds optional settings passed to the module; nil
	// means no config.
	Config map[string]any

	// Name is the module identifier used in URL paths and asset file names.
	Name string

	// ETag is the computed value used for HTTP caching.
	ETag string

	// Content holds the JavaScript file content as raw bytes.
	Content []byte
}

// NewCustomFrontendModule creates a new custom module with computed ETag.
//
// Takes name (string) which identifies the module.
// Takes content ([]byte) which provides the module content.
// Takes config (map[string]any) which specifies module
// configuration options.
//
// Returns *CustomFrontendModule which is the initialised
// module ready for use.
func NewCustomFrontendModule(name string, content []byte, config map[string]any) *CustomFrontendModule {
	return &CustomFrontendModule{
		Name:    name,
		Content: content,
		ETag:    computeETag(content),
		Config:  config,
	}
}

// ServeURL returns the URL at which this custom module is served to browsers.
//
// Returns string which is the fully qualified path to the
// minified JavaScript file for this module.
func (m *CustomFrontendModule) ServeURL() string {
	return fmt.Sprintf("/_piko/dist/ppframework.%s.min.js", m.Name)
}

// AssetPath returns the path used for storing in the asset store.
//
// Returns string which is the full path to the minified
// JavaScript asset.
func (m *CustomFrontendModule) AssetPath() string {
	return fmt.Sprintf("built/ppframework.%s.min.js", m.Name)
}

var (
	// cachedModulePreloadHTML holds the pre-built HTML for module preload link elements.
	cachedModulePreloadHTML string

	// cachedModuleScriptHTML holds the pre-built HTML for module script elements.
	cachedModuleScriptHTML string

	// cachedModuleConfigHTML holds the pre-built HTML for module configuration script elements.
	cachedModuleConfigHTML string

	// cachedDevWidgetHTML holds the HTML element for the dev tools overlay
	// widget. Set once at startup in dev mode; empty in production.
	cachedDevWidgetHTML string
)

// SetModuleHTML sets the cached module HTML for site-wide
// injection. Call this once during startup after all modules
// are registered.
//
// Takes preloadHTML (string) which contains the preload link elements.
// Takes scriptHTML (string) which contains the script elements.
// Takes configHTML (string) which contains the configuration elements.
func SetModuleHTML(preloadHTML, scriptHTML, configHTML string) {
	cachedModulePreloadHTML = preloadHTML
	cachedModuleScriptHTML = scriptHTML
	cachedModuleConfigHTML = configHTML
}

// GetModulePreloadHTML returns the cached preload HTML for
// frontend modules.
//
// Returns string which contains the HTML link elements for
// module preloading.
func GetModulePreloadHTML() string {
	return cachedModulePreloadHTML
}

// GetModuleScriptHTML returns the cached script HTML for
// frontend modules.
//
// Returns string which contains the pre-built HTML script tags.
func GetModuleScriptHTML() string {
	return cachedModuleScriptHTML
}

// GetModuleConfigHTML returns the cached HTML script for
// frontend module settings.
//
// Returns string which contains the pre-rendered HTML script
// configuration.
func GetModuleConfigHTML() string {
	return cachedModuleConfigHTML
}

// SetDevWidgetHTML sets the HTML element for the dev tools overlay widget.
// Call once at startup in dev mode.
//
// Takes html (string) which contains the HTML element for the dev widget.
func SetDevWidgetHTML(html string) {
	cachedDevWidgetHTML = html
}

// GetDevWidgetHTML returns the cached dev widget HTML element.
// Empty in production mode.
//
// Returns string which holds the dev widget HTML, or empty in production.
func GetDevWidgetHTML() string {
	return cachedDevWidgetHTML
}

// GenerateModuleHTML creates the HTML needed to load frontend
// modules.
//
// Takes moduleEntries ([]ModuleEntry) which contains the built-in modules to
// include.
// Takes customModules (map[string]*CustomFrontendModule)
// which provides custom modules to include alongside the
// built-in ones.
//
// Returns preloadHTML (string) which contains link tags for
// module preloading.
// Returns scriptHTML (string) which contains script tags for
// module loading.
// Returns configHTML (string) which contains a JSON script tag
// with module settings, or an empty string if no modules have
// settings.
func GenerateModuleHTML(moduleEntries []ModuleEntry, customModules map[string]*CustomFrontendModule) (preloadHTML, scriptHTML, configHTML string) {
	var preloadBuf, scriptBuf strings.Builder

	allConfigs := make(map[string]any)

	for _, entry := range moduleEntries {
		url := entry.Module.ServeURL()
		if url == "" {
			continue
		}
		sriHash := GetSRIHash(entry.Module.AssetPath())
		writeModulePreloadTag(&preloadBuf, url, sriHash)
		writeModuleScriptTag(&scriptBuf, url, sriHash)

		if entry.Config != nil {
			allConfigs[entry.Module.String()] = entry.Config
		}
	}

	for _, module := range customModules {
		url := module.ServeURL()
		sriHash := GetSRIHash(module.AssetPath())
		writeModulePreloadTag(&preloadBuf, url, sriHash)
		writeModuleScriptTag(&scriptBuf, url, sriHash)

		if module.Config != nil {
			allConfigs[module.Name] = module.Config
		}
	}

	if len(allConfigs) > 0 {
		configJSON, err := json.Marshal(allConfigs)
		if err == nil {
			configHTML = fmt.Sprintf(`<script id="pk-module-config" type="application/json">%s</script>`, string(configJSON))
		}
	}

	return preloadBuf.String(), scriptBuf.String(), configHTML
}

// writeModulePreloadTag writes a modulepreload link tag, with optional SRI
// integrity and crossorigin attributes.
//
// Takes buf (*strings.Builder) which receives the generated HTML.
// Takes url (string) which is the module URL for the href attribute.
// Takes sriHash (string) which is the SRI hash for the integrity attribute,
// or empty to omit integrity.
func writeModulePreloadTag(buf *strings.Builder, url, sriHash string) {
	if sriHash != "" {
		_, _ = fmt.Fprintf(buf, "<link rel=\"modulepreload\" href=%q integrity=%q crossorigin=\"anonymous\">", url, sriHash)
	} else {
		_, _ = fmt.Fprintf(buf, "<link rel=\"modulepreload\" href=%q>", url)
	}
}

// writeModuleScriptTag writes a module script tag, with optional SRI integrity
// and crossorigin attributes.
//
// Takes buf (*strings.Builder) which receives the generated HTML.
// Takes url (string) which is the module URL for the src attribute.
// Takes sriHash (string) which is the SRI hash for the integrity attribute,
// or empty to omit integrity.
func writeModuleScriptTag(buf *strings.Builder, url, sriHash string) {
	if sriHash != "" {
		_, _ = fmt.Fprintf(buf, "<script type=\"module\" src=%q integrity=%q crossorigin=\"anonymous\"></script>", url, sriHash)
	} else {
		_, _ = fmt.Fprintf(buf, "<script type=\"module\" src=%q></script>", url)
	}
}

// computeETag creates an ETag from the given content using SHA256.
//
// Takes content ([]byte) which is the data to hash.
//
// Returns string which is the quoted hash prefix in
// hexadecimal format.
func computeETag(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%q", hex.EncodeToString(hash[:8]))
}
