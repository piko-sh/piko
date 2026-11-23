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

package browser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/safedisk"
)

func TestHarnessOptions(t *testing.T) {
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{CWD: t.TempDir(), Enabled: true})
	require.NoError(t, err)
	options := defaultHarnessOptions()
	for _, option := range []HarnessOption{
		WithProjectDir("project"),
		WithOutputDir("output"),
		WithPort(8123),
		WithHeadless(false),
		WithInteractive(true),
		WithSimpleInteractive(),
		WithBuildTimeout(time.Minute),
		WithSkipBuild(true),
		WithEnv("KEY", "value"),
		WithEnv("OTHER", "second"),
		WithServerCommand("server", "--flag"),
		WithServerArgs("--port", "1"),
		WithSandboxFactory(factory),
	} {
		option(&options)
	}

	assert.Equal(t, "project", options.projectDir)
	assert.Equal(t, "output", options.outputDir)
	assert.Equal(t, 8123, options.port)
	assert.False(t, options.headless)
	assert.True(t, options.interactive)
	assert.False(t, options.interactiveTUI)
	assert.Equal(t, time.Minute, options.buildTimeout)
	assert.True(t, options.skipBuild)
	assert.Equal(t, map[string]string{"KEY": "value", "OTHER": "second"}, options.env)
	assert.Equal(t, []string{"server", "--flag"}, options.serverCommand)
	assert.Equal(t, []string{"--port", "1"}, options.serverArgs)
	assert.Same(t, factory, options.sandboxFactory)
}

func TestInteractiveOptionEnablesTheTUI(t *testing.T) {
	options := defaultHarnessOptions()
	WithInteractive(true)(&options)
	assert.True(t, options.interactiveTUI)
	assert.False(t, options.headless)
}

func TestWaitScreenshotAndSpecOptions(t *testing.T) {
	wait := defaultWaitConfig()
	assert.Equal(t, defaultWaitTimeout, wait.timeout)
	WithTimeout(time.Second)(&wait)
	assert.Equal(t, time.Second, wait.timeout)

	assert.NotNil(t, WithScreenshotTimeout(time.Second))

	spec := defaultSpecOptions()
	assert.Equal(t, defaultSpecTimeout, spec.timeout)
	WithUpdateGoldens(true)(&spec)
	WithSpecTimeout(time.Second)(&spec)
	assert.True(t, spec.updateGoldens)
	assert.Equal(t, time.Second, spec.timeout)
}
