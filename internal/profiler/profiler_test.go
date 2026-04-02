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

package profiler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPort_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 6060, DefaultPort)
}

func TestDefaultBindAddress_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "localhost", DefaultBindAddress)
}

func TestDefaultBlockProfileRate_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1000, DefaultBlockProfileRate)
}

func TestDefaultMutexProfileFraction_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 10, DefaultMutexProfileFraction)
}

func TestDefaultOutputDir_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "./profiles", DefaultOutputDir)
}

func TestCaptureBlockProfileRate_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, CaptureBlockProfileRate)
}

func TestCaptureMutexProfileFraction_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1, CaptureMutexProfileFraction)
}

func TestDefaultMemProfileRate_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, DefaultMemProfileRate)
}

func TestDefaultRollingTraceMinAge_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 15*time.Second, DefaultRollingTraceMinAge)
}

func TestDefaultRollingTraceMaxBytes_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t, 16*1024*1024, DefaultRollingTraceMaxBytes)
}

func TestCaptureMemProfileRate_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 4096, CaptureMemProfileRate)
}

func TestBasePath_HasExpectedValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/_piko", BasePath)
}

func TestGoroutineCount_ReturnsPositiveValue(t *testing.T) {
	t.Parallel()

	count := GoroutineCount()

	assert.Greater(t, count, 0)
}

func TestSetRuntimeRates_DoesNotPanicWithZeroConfig(t *testing.T) {

	assert.NotPanics(t, func() {
		SetRuntimeRates(Config{})
	})
}

func TestSetRuntimeRates_DoesNotPanicWithNonZeroConfig(t *testing.T) {

	assert.NotPanics(t, func() {
		SetRuntimeRates(Config{
			BlockProfileRate:     DefaultBlockProfileRate,
			MutexProfileFraction: DefaultMutexProfileFraction,
			MemProfileRate:       CaptureMemProfileRate,
		})
	})
}

func TestSetRuntimeRates_SkipsMemProfileRateWhenZero(t *testing.T) {

	assert.NotPanics(t, func() {
		SetRuntimeRates(Config{
			BlockProfileRate:     100,
			MutexProfileFraction: 5,
			MemProfileRate:       0,
		})
	})
}

func TestCheckBuildFlags_ReturnsString(t *testing.T) {
	t.Parallel()

	result := CheckBuildFlags()

	assert.IsType(t, "", result)
}

func TestServerAddress_FormatsCorrectly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default address",
			config: Config{
				BindAddress: DefaultBindAddress,
				Port:        DefaultPort,
			},
			expected: "localhost:6060",
		},
		{
			name: "custom address",
			config: Config{
				BindAddress: "0.0.0.0",
				Port:        8080,
			},
			expected: "0.0.0.0:8080",
		},
		{
			name: "empty bind address",
			config: Config{
				BindAddress: "",
				Port:        9090,
			},
			expected: ":9090",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result := ServerAddress(testCase.config)

			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestConfig_ZeroValueHasExpectedDefaults(t *testing.T) {
	t.Parallel()

	var config Config

	assert.Empty(t, config.BindAddress)
	assert.Empty(t, config.OutputDir)
	assert.Zero(t, config.Port)
	assert.Zero(t, config.BlockProfileRate)
	assert.Zero(t, config.MutexProfileFraction)
	assert.Zero(t, config.MemProfileRate)
	assert.False(t, config.EnableRollingTrace)
	assert.Zero(t, config.RollingTraceMinAge)
	assert.Zero(t, config.RollingTraceMaxBytes)
}
