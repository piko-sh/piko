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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrontendModule_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		module FrontendModule
	}{
		{name: "analytics", module: ModuleAnalytics, want: "analytics"},
		{name: "modals", module: ModuleModals, want: "modals"},
		{name: "toasts", module: ModuleToasts, want: "toasts"},
		{name: "unknown", module: FrontendModule(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.module.String())
		})
	}
}

func TestFrontendModule_AssetPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		module FrontendModule
	}{
		{name: "analytics", module: ModuleAnalytics, want: "built/ppframework.analytics.es.js"},
		{name: "modals", module: ModuleModals, want: "built/ppframework.modals.es.js"},
		{name: "toasts", module: ModuleToasts, want: "built/ppframework.toasts.es.js"},
		{name: "dev", module: ModuleDev, want: "built/ppframework.dev.es.js"},
		{name: "components", module: ModuleComponents, want: "built/ppframework.components.es.js"},
		{name: "unknown", module: FrontendModule(99), want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.module.AssetPath())
		})
	}
}

func TestFrontendModule_ServeURLNamesTheBuildsVersionedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		want   string
		module FrontendModule
		build  CoreRuntimeBuild
	}{
		{name: "analytics in production", module: ModuleAnalytics, build: CoreRuntimeBuildProduction, want: "/_piko/dist/" + CoreRuntimeBuildProduction.RuntimeVersion() + "/ppframework.analytics.es.js"},
		{name: "analytics in development", module: ModuleAnalytics, build: CoreRuntimeBuildDevelopment, want: "/_piko/dist/" + CoreRuntimeBuildDevelopment.RuntimeVersion() + "/ppframework.analytics.es.js"},
		{name: "modals", module: ModuleModals, build: CoreRuntimeBuildProduction, want: "/_piko/dist/" + CoreRuntimeBuildProduction.RuntimeVersion() + "/ppframework.modals.es.js"},
		{name: "toasts", module: ModuleToasts, build: CoreRuntimeBuildProduction, want: "/_piko/dist/" + CoreRuntimeBuildProduction.RuntimeVersion() + "/ppframework.toasts.es.js"},
		{name: "dev", module: ModuleDev, build: CoreRuntimeBuildDevelopment, want: "/_piko/dist/" + CoreRuntimeBuildDevelopment.RuntimeVersion() + "/ppframework.dev.es.js"},
		{name: "components", module: ModuleComponents, build: CoreRuntimeBuildProduction, want: "/_piko/dist/" + CoreRuntimeBuildProduction.RuntimeVersion() + "/ppframework.components.es.js"},
		{name: "unknown", module: FrontendModule(99), build: CoreRuntimeBuildProduction, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.module.ServeURL(tt.build))
		})
	}
}

func TestFrontendModule_ImportURLIsUnversioned(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/_piko/dist/ppframework.components.es.js", ModuleComponents.ImportURL())
	assert.Equal(t, "/_piko/dist/ppframework.dev.es.js", ModuleDev.ImportURL())
	assert.Empty(t, FrontendModule(99).ImportURL())
}

func TestNewCustomFrontendModule(t *testing.T) {
	t.Parallel()

	content := []byte("console.log('hello');")
	config := map[string]any{"key": "value"}

	m := NewCustomFrontendModule("mymod", content, config)

	assert.Equal(t, "mymod", m.Name)
	assert.Equal(t, content, m.Content)
	assert.Equal(t, config, m.Config)
	assert.NotEmpty(t, m.ETag, "ETag should be computed")
}

func TestCustomFrontendModule_ServeURL(t *testing.T) {
	t.Parallel()

	m := &CustomFrontendModule{Name: "widget"}
	assert.Equal(t, "/_piko/dist/ppframework.widget.min.js", m.ServeURL())
}

func TestCustomFrontendModule_AssetPath(t *testing.T) {
	t.Parallel()

	m := &CustomFrontendModule{Name: "widget"}
	assert.Equal(t, "built/ppframework.widget.min.js", m.AssetPath())
}
