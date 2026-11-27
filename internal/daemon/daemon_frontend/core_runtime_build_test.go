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
	"github.com/stretchr/testify/require"
)

func TestCoreRuntimeBuild_ServedFileName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		build         CoreRuntimeBuild
		requestedFile string
		want          string
	}{
		{name: "production sends the minified core at the core URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.core.es.js", want: "ppframework.core.min.es.js"},
		{name: "the dev modes send the unminified core at the core URL", build: CoreRuntimeBuildDevelopment, requestedFile: "ppframework.core.es.js", want: "ppframework.core.es.js"},
		{name: "production sends the minified components runtime at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.components.es.js", want: "ppframework.components.min.es.js"},
		{name: "the dev modes send the unminified components runtime at its URL", build: CoreRuntimeBuildDevelopment, requestedFile: "ppframework.components.es.js", want: "ppframework.components.es.js"},
		{name: "production sends the minified analytics module at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.analytics.es.js", want: "ppframework.analytics.min.es.js"},
		{name: "production sends the minified animation module at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.animation.es.js", want: "ppframework.animation.min.es.js"},
		{name: "production sends the minified modals module at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.modals.es.js", want: "ppframework.modals.min.es.js"},
		{name: "production sends the minified toasts module at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.toasts.es.js", want: "ppframework.toasts.min.es.js"},
		{name: "production sends the minified dev module at its URL", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.dev.es.js", want: "ppframework.dev.min.es.js"},
		{name: "production sends a minified file named in the request as named", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.modals.min.es.js", want: "ppframework.modals.min.es.js"},
		{name: "production sends the UMD core as named", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.core.min.umd.js", want: "ppframework.core.min.umd.js"},
		{name: "production sends the minified source map as named", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.core.min.es.js.map", want: "ppframework.core.min.es.js.map"},
		{name: "production sends the unminified source map as named", build: CoreRuntimeBuildProduction, requestedFile: "ppframework.core.es.js.map", want: "ppframework.core.es.js.map"},
		{name: "a value that names no build sends the unminified core", build: CoreRuntimeBuild(99), requestedFile: "ppframework.core.es.js", want: "ppframework.core.es.js"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, testCase.build.ServedFileName(testCase.requestedFile))
		})
	}
}

func TestCoreRuntimeBuild_CoreModuleSRIHashNamesTheServedFile(t *testing.T) {
	ResetSRIState()
	t.Cleanup(ResetSRIState)
	SetSRIEnabled(true)
	SetSRIHash("built/ppframework.core.es.js", "sha384-unminified")
	SetSRIHash("built/ppframework.core.min.es.js", "sha384-minified")

	assert.Equal(t, "sha384-unminified", CoreRuntimeBuildDevelopment.CoreModuleSRIHash())
	assert.Equal(t, "sha384-minified", CoreRuntimeBuildProduction.CoreModuleSRIHash())
}

func TestCoreRuntimeBuild_RuntimeVersionIsOwnedByEachBuild(t *testing.T) {
	t.Parallel()

	bundleVersion := RuntimeVersion()
	require.True(t, IsRuntimeVersion(bundleVersion))
	development := CoreRuntimeBuildDevelopment.RuntimeVersion()
	production := CoreRuntimeBuildProduction.RuntimeVersion()

	assert.True(t, IsRuntimeVersion(development), "the version keeps the shape the browser runtime parses")
	assert.True(t, IsRuntimeVersion(production))
	assert.NotEqual(t, development, production, "a production and a dev server must never share a versioned core URL")
	assert.NotEqual(t, bundleVersion, development)
	assert.NotEqual(t, bundleVersion, production)
	assert.Equal(t, production, CoreRuntimeBuildProduction.VersionOf(bundleVersion), "the version must be stable")
	assert.NotEqual(t, production, CoreRuntimeBuildProduction.VersionOf("fedcba9876543210"), "the version must follow the bundles")
	assert.Empty(t, CoreRuntimeBuildProduction.VersionOf(""), "without a bundle version the URLs stay unversioned")
}

func TestCoreRuntimeBuild_ModuleSRIHashNamesTheServedFile(t *testing.T) {
	ResetSRIState()
	t.Cleanup(ResetSRIState)
	SetSRIEnabled(true)
	SetSRIHash("built/ppframework.components.es.js", "sha384-unminified")
	SetSRIHash("built/ppframework.components.min.es.js", "sha384-minified")

	assert.Equal(t, "sha384-unminified", CoreRuntimeBuildDevelopment.ModuleSRIHash("ppframework.components.es.js"))
	assert.Equal(t, "sha384-minified", CoreRuntimeBuildProduction.ModuleSRIHash("ppframework.components.es.js"))
}

func TestHasBuildVariants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		fileName string
		want     bool
	}{
		{name: "the core runtime", fileName: "ppframework.core.es.js", want: true},
		{name: "the components runtime", fileName: "ppframework.components.es.js", want: true},
		{name: "the analytics module", fileName: "ppframework.analytics.es.js", want: true},
		{name: "a minified file", fileName: "ppframework.core.min.es.js", want: false},
		{name: "a source map", fileName: "ppframework.components.es.js.map", want: false},
		{name: "an unknown file", fileName: "missing.js", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.want, HasBuildVariants(testCase.fileName))
		})
	}
}
