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

package piko

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
)

func TestFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"AnalyticsConfig": AnalyticsConfig{},
		"CachePolicy":     CachePolicy{},
		"FrontendModule":  FrontendModule(0),
		"HTTPMethod":      HTTPMethod(""),
		"Metadata":        Metadata{},
		"ModalsConfig":    ModalsConfig{},
		"NoProps":         NoProps{},
		"NoResponse":      NoResponse{},
		"PublicConfig":    PublicConfig{},
		"RequestData":     RequestData{},
		"SSRServer":       SSRServer{},
		"ToastsConfig":    ToastsConfig{},

		"ConfigResolver": (*ConfigResolver)(nil),
		"Container":      (*Container)(nil),

		"Option":                     (Option)(nil),
		"WithCapabilityService":      WithCapabilityService,
		"WithConfigResolvers":        WithConfigResolvers,
		"WithCSRFSecret":             WithCSRFSecret,
		"WithCustomFrontendModule":   WithCustomFrontendModule,
		"WithEventBus":               WithEventBus,
		"WithFrontendModule":         WithFrontendModule,
		"WithI18nService":            WithI18nService,
		"WithJSONTypeInspectorCache": WithJSONTypeInspectorCache,
		"WithMemoryRegistryCache":    WithMemoryRegistryCache,
		"WithOrchestratorService":    WithOrchestratorService,
		"WithRegistryService":        WithRegistryService,

		"New":         New,
		"WithSymbols": ((*SSRServer)(nil)).WithSymbols,
		"Configure":   ((*SSRServer)(nil)).Configure,
		"Setup":       ((*SSRServer)(nil)).Setup,
		"Generate":    ((*SSRServer)(nil)).Generate,
		"Run":         ((*SSRServer)(nil)).Run,
		"Stop":        ((*SSRServer)(nil)).Stop,

		"GenerateModeAll":       GenerateModeAll,
		"GenerateModeManifest":  GenerateModeManifest,
		"LevelDebug":            LevelDebug,
		"LevelError":            LevelError,
		"LevelInfo":             LevelInfo,
		"LevelWarn":             LevelWarn,
		"MethodDelete":          MethodDelete,
		"MethodGet":             MethodGet,
		"MethodHead":            MethodHead,
		"MethodOptions":         MethodOptions,
		"MethodPatch":           MethodPatch,
		"MethodPost":            MethodPost,
		"MethodPut":             MethodPut,
		"ModuleAnalytics":       ModuleAnalytics,
		"ModuleModals":          ModuleModals,
		"ModuleToasts":          ModuleToasts,
		"RunModeDev":            RunModeDev,
		"RunModeDevInterpreted": RunModeDevInterpreted,
		"RunModeProd":           RunModeProd,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
