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

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/config/config_domain"
)

type MockFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
	error error
}

func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *MockFileSystem) AddFile(path string, content []byte) {
	m.files[path] = content
}

func (m *MockFileSystem) AddDir(path string) {
	m.dirs[path] = true
}

func (m *MockFileSystem) SetError(err error) {
	m.error = err
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if m.error != nil {
		return nil, m.error
	}
	if content, exists := m.files[name]; exists {
		return content, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.error != nil {
		return nil, m.error
	}
	if _, exists := m.dirs[name]; exists {
		return &MockFileInfo{name: name, isDir: true}, nil
	}
	if _, exists := m.files[name]; exists {
		return &MockFileInfo{name: name, isDir: false}, nil
	}
	return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
}

type MockFileInfo struct {
	name  string
	isDir bool
}

func (m *MockFileInfo) Name() string       { return filepath.Base(m.name) }
func (m *MockFileInfo) Size() int64        { return 0 }
func (m *MockFileInfo) Mode() os.FileMode  { return 0755 }
func (m *MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *MockFileInfo) IsDir() bool        { return m.isDir }
func (m *MockFileInfo) Sys() any           { return nil }

func TestNewConfigProvider(t *testing.T) {
	provider := NewConfigProvider()

	if provider == nil {
		t.Fatal("NewConfigProvider returned nil")
	}

	if provider.ServerConfig.Paths.BaseDir != nil {
		t.Errorf("Expected zero ServerConfig, but BaseDir was: %q", *provider.ServerConfig.Paths.BaseDir)
	}
	if provider.WebsiteConfig.Name != "" {
		t.Errorf("Expected zero WebsiteConfig, but Name was: %q", provider.WebsiteConfig.Name)
	}
}

func TestLoadConfig(t *testing.T) {
	config_domain.ResetGlobalFlagCoordinator()
	config_domain.ResetGlobalResolverRegistry()
	config_domain.ResetDotEnvCache()

	mockFS := NewMockFileSystem()
	mockFS.AddDir("/app")
	mockFS.AddFile("piko.yaml", []byte("network:\n  port: 9000"))

	t.Setenv("PIKO_PORT", "3000")

	provider := newTestConfigProvider(mockFS)

	t.Setenv("PIKO_BASE_DIR", "/app")

	_, err := provider.LoadConfig(nil, nil)

	if err != nil {
		t.Fatalf("LoadConfig() returned an unexpected error: %v", err)
	}

	if *provider.ServerConfig.Build.DefaultServeMode != "render" {
		t.Errorf("Expected DefaultServeMode to be 'render' (from default), but got %q", *provider.ServerConfig.Build.DefaultServeMode)
	}

	if *provider.ServerConfig.Network.Port != "3000" {
		t.Errorf("Expected Port to be '3000' (from env var), but got %q", *provider.ServerConfig.Network.Port)
	}
}

func TestLoadConfig_NoValidatorSkipsValidation(t *testing.T) {
	config_domain.ResetGlobalFlagCoordinator()
	config_domain.ResetGlobalResolverRegistry()
	config_domain.ResetDotEnvCache()

	t.Setenv("PIKO_DEFAULT_SERVE_MODE", "invalid-mode")
	t.Setenv("PIKO_BASE_DIR", t.TempDir())

	provider := NewConfigProvider()

	_, err := provider.LoadConfig(nil, nil)

	if err != nil {
		errString := err.Error()
		isFlagError := strings.Contains(errString, "flags have been parsed")
		if !isFlagError {
			t.Errorf("Expected either no error or a flag registration error, but got: %v", err)
		}
	}
}

func TestValidateServerPaths(t *testing.T) {
	testCases := []struct {
		setupFS       func(fs *MockFileSystem, baseDir string)
		name          string
		baseDir       string
		errorContains string
		expectError   bool
	}{
		{
			name:    "valid base directory",
			baseDir: "/app/valid",
			setupFS: func(fs *MockFileSystem, baseDir string) {
				fs.AddDir(baseDir)
			},
			expectError: false,
		},
		{
			name:          "base directory does not exist",
			baseDir:       "/app/nonexistent",
			setupFS:       func(fs *MockFileSystem, baseDir string) {},
			expectError:   true,
			errorContains: "website base directory not found",
		},
		{
			name:    "base directory is a file",
			baseDir: "/app/file",
			setupFS: func(fs *MockFileSystem, baseDir string) {
				fs.AddFile(baseDir, []byte("i am a file"))
			},
			expectError:   true,
			errorContains: "baseDir is not a directory",
		},
		{
			name:    "filesystem stat error",
			baseDir: "/app/error",
			setupFS: func(fs *MockFileSystem, baseDir string) {
				fs.SetError(errors.New("permission denied"))
			},
			expectError:   true,
			errorContains: "cannot access website base directory",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			tc.setupFS(mockFS, tc.baseDir)

			provider := newTestConfigProvider(mockFS)
			provider.ServerConfig.Paths.BaseDir = &tc.baseDir

			err := provider.validateServerPaths()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, but got: %v", tc.errorContains, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
		})
	}
}

func TestLoadWebsiteConfig(t *testing.T) {
	baseDir := "/app"

	testCases := []struct {
		setupFS       func(fs *MockFileSystem)
		name          string
		errorContains string
		expectName    string
		expectError   bool
	}{
		{
			name:       "no config file is acceptable",
			setupFS:    func(fs *MockFileSystem) {},
			expectName: "",
		},
		{
			name: "valid config file",
			setupFS: func(fs *MockFileSystem) {
				fs.AddFile(filepath.Join(baseDir, "config.json"), []byte(`{"name": "Test Site"}`))
			},
			expectName: "Test Site",
		},
		{
			name: "invalid json content",
			setupFS: func(fs *MockFileSystem) {
				fs.AddFile(filepath.Join(baseDir, "config.json"), []byte(`{"name": "Test Site",`))
			},
			expectError:   true,
			errorContains: "failed to unmarshal site config JSON",
		},
		{
			name: "filesystem read error",
			setupFS: func(fs *MockFileSystem) {
				fs.SetError(errors.New("permission denied"))
			},
			expectError:   true,
			errorContains: "failed to read site config",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockFS := NewMockFileSystem()
			tc.setupFS(mockFS)

			provider := newTestConfigProvider(mockFS)
			provider.ServerConfig.Paths.BaseDir = &baseDir

			err := provider.LoadWebsiteConfig()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, but got: %v", tc.errorContains, err)
				}
			} else if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if provider.WebsiteConfig.Name != tc.expectName {
				t.Errorf("Expected website name to be %q, but got %q", tc.expectName, provider.WebsiteConfig.Name)
			}
		})
	}
}

func TestFindFlagValue(t *testing.T) {
	testCases := []struct {
		name         string
		flagName     string
		defaultValue string
		expected     string
		arguments    []string
	}{
		{name: "long flag with equals", arguments: []string{"--configFile=foo.yaml"}, flagName: "configFile", defaultValue: "piko.yaml", expected: "foo.yaml"},
		{name: "long flag with space", arguments: []string{"--configFile", "bar.yaml"}, flagName: "configFile", defaultValue: "piko.yaml", expected: "bar.yaml"},
		{name: "short flag with equals", arguments: []string{"-c=foo.yaml"}, flagName: "c", defaultValue: "piko.yaml", expected: "foo.yaml"},
		{name: "short flag with space", arguments: []string{"-c", "bar.yaml"}, flagName: "c", defaultValue: "piko.yaml", expected: "bar.yaml"},
		{name: "flag not present", arguments: []string{"--other-flag"}, flagName: "configFile", defaultValue: "piko.yaml", expected: "piko.yaml"},
		{name: "flag present but no value", arguments: []string{"--configFile"}, flagName: "configFile", defaultValue: "piko.yaml", expected: "piko.yaml"},
		{name: "empty arguments", arguments: []string{}, flagName: "configFile", defaultValue: "piko.yaml", expected: "piko.yaml"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalArgs := os.Args
			os.Args = append([]string{originalArgs[0]}, tc.arguments...)
			t.Cleanup(func() { os.Args = originalArgs })

			actual := findFlagValue(tc.flagName, tc.defaultValue)
			if actual != tc.expected {
				t.Errorf("Expected findFlagValue to return %q, but got %q", tc.expected, actual)
			}
		})
	}
}
