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

package config_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/config/config_domain"
)

type mockValidator struct{}

func (mockValidator) Struct(any) error {
	return errors.New("mock validation error")
}

type TestSpec struct {
	ValidateFields   map[string]any    `json:"validateFields,omitempty"`
	Environment      map[string]string `json:"environment,omitempty"`
	Description      string            `json:"description"`
	ErrorContains    string            `json:"errorContains,omitempty"`
	ConfigFiles      []string          `json:"configFiles,omitempty"`
	ExpectError      bool              `json:"expectError"`
	ExpectedDefaults bool              `json:"expectedDefaults,omitempty"`
	UseValidator     bool              `json:"useValidator,omitempty"`
}

func loadTestSpec(testDir string) (*TestSpec, error) {
	specPath := filepath.Join(testDir, "testspec.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read testspec.json: %w", err)
	}

	var spec TestSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse testspec.json: %w", err)
	}

	return &spec, nil
}

func createTestFilePaths(testDir string, configFiles []string) []string {
	var filePaths []string

	for _, file := range configFiles {
		fullPath := filepath.Join(testDir, file)
		if _, err := os.Stat(fullPath); err == nil {
			filePaths = append(filePaths, fullPath)
		}
	}

	return filePaths
}

func validateFields(t *testing.T, config *config.ServerConfig, expected map[string]any) {
	for field, expectedValue := range expected {
		actual := derefPointer(getFieldValue(config, field))

		if expectedFloat, ok := expectedValue.(float64); ok {
			if actualInt, ok := actual.(int); ok {
				assert.Equal(t, int(expectedFloat), actualInt, "Field %s should match expected value", field)
				continue
			}
		}

		assert.Equal(t, expectedValue, actual, "Field %s should match expected value", field)
	}
}

func derefPointer(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return v
}

func getFieldValue(config *config.ServerConfig, fieldPath string) any {

	switch fieldPath {
	case "I18nDefaultLocale":
		return config.I18nDefaultLocale
	case "CSRFSecret":
		return config.CSRFSecret
	case "Network.Port":
		return config.Network.Port
	case "Network.PublicDomain":
		return config.Network.PublicDomain
	case "Paths.BaseDir":
		return config.Paths.BaseDir
	case "Otlp.Enabled":
		return config.Otlp.Enabled
	default:
		return nil
	}
}

func TestConfigLoaderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping config loader integration tests in short mode")
	}

	testDataDir := "testdata"

	testDirs, err := filepath.Glob(filepath.Join(testDataDir, "*"))
	require.NoError(t, err, "Failed to find test directories")
	require.NotEmpty(t, testDirs, "No test directories found in testdata")

	for _, testDir := range testDirs {
		if !isDirectory(testDir) {
			continue
		}

		testName := filepath.Base(testDir)
		t.Run(testName, func(t *testing.T) {
			config_domain.ResetGlobalFlagCoordinator()

			spec, err := loadTestSpec(testDir)
			require.NoError(t, err, "Failed to load test spec for %s", testName)

			for key, value := range spec.Environment {
				t.Setenv(key, value)
			}

			filePaths := createTestFilePaths(testDir, spec.ConfigFiles)

			serverConfig := &config.ServerConfig{}
			opts := config_domain.LoaderOptions{
				FilePaths: filePaths,
			}

			if spec.UseValidator {
				opts.Validator = mockValidator{}
			}

			loadContext, err := config_domain.Load(context.Background(), serverConfig, opts)

			if spec.ExpectError {
				assert.Error(t, err, "Expected error for test case: %s", spec.Description)
				if spec.ErrorContains != "" {
					assert.Contains(t, err.Error(), spec.ErrorContains, "Error should contain expected text")
				}
				return
			}

			require.NoError(t, err, "Unexpected error for test case: %s", spec.Description)
			require.NotNil(t, loadContext, "Load context should not be nil")

			if len(spec.ValidateFields) > 0 {
				validateFields(t, serverConfig, spec.ValidateFields)
			}

			if spec.ExpectedDefaults {

				assert.NotEmpty(t, serverConfig.I18nDefaultLocale, "I18nDefaultLocale default should be set")
				assert.NotEmpty(t, serverConfig.Network.Port, "Network.Port default should be set")
			}
		})
	}
}

func TestServerConfigLoading(t *testing.T) {
	tests := []struct {
		validate    func(t *testing.T, config *config.ServerConfig)
		env         map[string]string
		name        string
		expectError bool
	}{
		{
			name:        "default_config_loading",
			expectError: false,
			validate: func(t *testing.T, config *config.ServerConfig) {
				assert.Equal(t, "en", *config.I18nDefaultLocale)
				assert.Equal(t, "8080", *config.Network.Port)
				assert.Equal(t, "localhost:8080", *config.Network.PublicDomain)
				assert.Equal(t, ".", *config.Paths.BaseDir)
				assert.False(t, *config.Otlp.Enabled)
			},
		},
		{
			name: "environment_override",
			env: map[string]string{
				"PIKO_PORT": "9090",
			},
			expectError: false,
			validate: func(t *testing.T, config *config.ServerConfig) {
				assert.Equal(t, "9090", *config.Network.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config_domain.ResetGlobalFlagCoordinator()

			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			provider := config.NewConfigProvider()
			defaults := &config.ServerConfig{}

			loadContext, err := provider.LoadConfig(defaults, nil)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, loadContext)

			serverConfig, ok := loadContext.Target.(*config.ServerConfig)
			require.True(t, ok, "Target should be *config.ServerConfig")
			require.NotNil(t, serverConfig)

			if tt.validate != nil {
				tt.validate(t, serverConfig)
			}
		})
	}
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func changeCWD(newDir string) (func(), error) {
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	err = os.Chdir(newDir)
	if err != nil {
		return nil, fmt.Errorf("failed to change directory to %s: %w", newDir, err)
	}

	cleanup := func() {
		if err := os.Chdir(originalDir); err != nil {
			panic(fmt.Sprintf("failed to restore original directory %s: %v", originalDir, err))
		}
	}

	return cleanup, nil
}

func TestCWDConfigLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CWD config loading tests in short mode")
	}

	tests := []struct {
		setupEnv    map[string]string
		validate    func(t *testing.T, config *config.ServerConfig)
		name        string
		testDataDir string
		cwdSubPath  string
	}{
		{
			name:        "yaml_config_from_cwd",
			testDataDir: "testdata/05_cwd_yaml_loading",
			cwdSubPath:  "",
			setupEnv:    map[string]string{},
			validate: func(t *testing.T, config *config.ServerConfig) {
				assert.Equal(t, "fr", *config.I18nDefaultLocale)
				assert.Equal(t, "9000", *config.Network.Port)
				assert.Equal(t, "test.example.com:9000", *config.Network.PublicDomain)
				assert.Equal(t, ".", *config.Paths.BaseDir)
				assert.True(t, *config.Otlp.Enabled)
			},
		},
		{
			name:        "precedence_from_cwd",
			testDataDir: "testdata/06_cwd_precedence",
			cwdSubPath:  "",
			setupEnv:    map[string]string{"PIKO_ENV": "dev"},
			validate: func(t *testing.T, config *config.ServerConfig) {

				assert.Equal(t, "7777", *config.Network.Port)
				assert.Equal(t, "dev.example.com", *config.Network.PublicDomain)
			},
		},
		{
			name:        "nested_dir_parent_cwd",
			testDataDir: "testdata/07_cwd_nested_dirs",
			cwdSubPath:  "",
			setupEnv:    map[string]string{},
			validate: func(t *testing.T, config *config.ServerConfig) {

				assert.Equal(t, "8000", *config.Network.Port)
				assert.Equal(t, "parent.example.com", *config.Network.PublicDomain)
			},
		},
		{
			name:        "nested_dir_subdir_cwd",
			testDataDir: "testdata/07_cwd_nested_dirs",
			cwdSubPath:  "subdir",
			setupEnv:    map[string]string{},
			validate: func(t *testing.T, config *config.ServerConfig) {

				assert.Equal(t, "8001", *config.Network.Port)
				assert.Equal(t, "subdir.example.com", *config.Network.PublicDomain)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config_domain.ResetGlobalFlagCoordinator()

			for key, value := range tt.setupEnv {
				t.Setenv(key, value)
			}

			targetDir := tt.testDataDir
			if tt.cwdSubPath != "" {
				targetDir = filepath.Join(targetDir, tt.cwdSubPath)
			}

			cwdCleanup, err := changeCWD(targetDir)
			require.NoError(t, err, "Failed to change to directory: %s", targetDir)
			defer cwdCleanup()

			provider := config.NewConfigProvider()
			defaults := &config.ServerConfig{}

			loadContext, err := provider.LoadConfig(defaults, nil)
			require.NoError(t, err, "Failed to load config from CWD: %s", targetDir)
			require.NotNil(t, loadContext, "Load context should not be nil")

			serverConfig, ok := loadContext.Target.(*config.ServerConfig)
			require.True(t, ok, "Target should be *config.ServerConfig")
			require.NotNil(t, serverConfig, "Config should not be nil")

			if tt.validate != nil {
				tt.validate(t, serverConfig)
			}
		})
	}
}
