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

package config_domain

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFileReader struct {
	files map[string][]byte
	err   error
}

func (m *mockFileReader) ReadFile(path string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func TestGetUnmarshaler(t *testing.T) {
	testCases := []struct {
		name    string
		path    string
		wantExt string
		wantNil bool
	}{
		{
			name:    "json file",
			path:    "config.json",
			wantNil: false,
			wantExt: ".json",
		},
		{
			name:    "yaml file",
			path:    "config.yaml",
			wantNil: false,
			wantExt: ".yaml",
		},
		{
			name:    "yml file",
			path:    "config.yml",
			wantNil: false,
			wantExt: ".yml",
		},
		{
			name:    "JSON uppercase",
			path:    "config.JSON",
			wantNil: false,
			wantExt: ".json",
		},
		{
			name:    "unsupported extension",
			path:    "config.toml",
			wantNil: true,
			wantExt: ".toml",
		},
		{
			name:    "no extension",
			path:    "config",
			wantNil: true,
			wantExt: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unmarshaler, ext := getUnmarshaler(tc.path)

			if tc.wantNil {
				assert.Nil(t, unmarshaler)
			} else {
				assert.NotNil(t, unmarshaler)
			}
			assert.Equal(t, tc.wantExt, ext)
		})
	}
}

func TestLoadFilesJSON(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.json": []byte(`{"host": "localhost", "port": 8080}`),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.json"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8080, config.Port)
	assert.Contains(t, ctx.FieldSources["Host"], "file:")
}

func TestLoadFilesYAML(t *testing.T) {
	type testConfig struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.yaml": []byte("host: localhost\nport: 8080"),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.yaml"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8080, config.Port)
}

func TestLoadFilesMultiple(t *testing.T) {
	type testConfig struct {
		Host string `json:"host" yaml:"host"`
		Port int    `json:"port" yaml:"port"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/base.yaml": []byte("host: basehost\nport: 8000"),
			"/prod.json": []byte(`{"port": 9000}`),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/base.yaml", "/prod.json"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.NoError(t, err)
	assert.Equal(t, "basehost", config.Host)
	assert.Equal(t, 9000, config.Port)
}

func TestLoadFilesMissingOptional(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/nonexistent.json"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.NoError(t, err)
}

func TestLoadFilesReadError(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{"/config.json": nil},
		err:   errors.New("read error"),
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.json"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadFilesUnsupportedFormat(t *testing.T) {
	type testConfig struct {
		Host string
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.toml": []byte("host = 'localhost'"),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.toml"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file format")
}

func TestLoadFilesInvalidJSON(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.json": []byte(`{"host": invalid}`),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.json"}},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal config file")
}

func TestCheckStrictSuccess(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.json": []byte(`{"host": "localhost", "port": 8080}`),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.json"}, StrictFile: true},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.NoError(t, err)
}

func TestCheckStrictFailure(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
	}

	reader := &mockFileReader{
		files: map[string][]byte{
			"/config.json": []byte(`{"host": "localhost", "unknownField": "value"}`),
		},
	}

	loader := &Loader{
		opts:       LoaderOptions{FilePaths: []string{"/config.json"}, StrictFile: true},
		fileReader: reader,
	}

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err := loader.loadFiles(config, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown configuration keys found")
	assert.Contains(t, err.Error(), "unknownField")
}

func TestCollectTags(t *testing.T) {
	type nested struct {
		NestedField string `json:"nestedField"`
	}

	type config struct {
		PtrNested  *nested `json:"ptrNested"`
		JSONField  string  `json:"jsonField"`
		YAMLField  string  `yaml:"yamlField"`
		PlainField string
		Ignored    string `json:"-"`
		Nested     nested `json:"nested"`
	}

	keys := make(map[string]struct{})
	collectTags(reflect.TypeFor[config](), keys)

	assert.Contains(t, keys, "jsonField")
	assert.Contains(t, keys, "yamlField")
	assert.Contains(t, keys, "plainfield")
	assert.NotContains(t, keys, "-")
	assert.Contains(t, keys, "nested")
	assert.Contains(t, keys, "nestedField")
	assert.Contains(t, keys, "ptrNested")
}

func TestCollectTagsWithOptions(t *testing.T) {
	type config struct {
		Field string `json:"field,omitempty"`
	}

	keys := make(map[string]struct{})
	collectTags(reflect.TypeFor[config](), keys)

	assert.Contains(t, keys, "field")
	assert.NotContains(t, keys, "field,omitempty")
}

func TestDetectChanges(t *testing.T) {
	type config struct {
		Host string
		Port int
	}

	before := config{Host: "old", Port: 8000}
	after := config{Host: "new", Port: 8000}

	ctx := &LoadContext{FieldSources: make(map[string]string)}
	detectChanges(reflect.ValueOf(before), reflect.ValueOf(after), "", "test-source", ctx)

	assert.Equal(t, "test-source", ctx.FieldSources["Host"])
	_, hasPort := ctx.FieldSources["Port"]
	assert.False(t, hasPort)
}

func TestDetectChangesNested(t *testing.T) {
	type nested struct {
		Port int
	}
	type config struct {
		Nested nested
	}

	before := config{Nested: nested{Port: 8000}}
	after := config{Nested: nested{Port: 9000}}

	ctx := &LoadContext{FieldSources: make(map[string]string)}
	detectChanges(reflect.ValueOf(before), reflect.ValueOf(after), "", "test-source", ctx)

	assert.Equal(t, "test-source", ctx.FieldSources["Nested.Port"])
}

func TestDetectChangesWithPrefix(t *testing.T) {
	type config struct {
		Host string
	}

	before := config{Host: "old"}
	after := config{Host: "new"}

	ctx := &LoadContext{FieldSources: make(map[string]string)}
	detectChanges(reflect.ValueOf(before), reflect.ValueOf(after), "Config", "test-source", ctx)

	assert.Equal(t, "test-source", ctx.FieldSources["Config.Host"])
}

func TestDetectChangesInvalidValues(t *testing.T) {
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	detectChanges(reflect.Value{}, reflect.Value{}, "", "test-source", ctx)
	assert.Empty(t, ctx.FieldSources)
}

func TestOsFileReaderReadFile(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "test.txt")

	content := []byte("test content")
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)

	reader := osFileReader{}
	data, err := reader.ReadFile(filePath)

	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestOsFileReaderReadFileNotFound(t *testing.T) {
	reader := osFileReader{}
	_, err := reader.ReadFile("/nonexistent/path/file.txt")

	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLoadFilesIntegration(t *testing.T) {
	type testConfig struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	}

	directory := t.TempDir()
	jsonPath := filepath.Join(directory, "config.json")

	content := []byte(`{"host": "localhost", "port": 3000}`)
	err := os.WriteFile(jsonPath, content, 0644)
	require.NoError(t, err)

	loader := NewLoader(LoaderOptions{
		FilePaths: []string{jsonPath},
	})
	defer loader.Close()

	config := &testConfig{}
	ctx := &LoadContext{FieldSources: make(map[string]string)}

	err = loader.loadFiles(config, ctx)
	require.NoError(t, err)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 3000, config.Port)
}
