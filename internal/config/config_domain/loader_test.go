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
	"context"
	"encoding"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingValidator struct{}

func (failingValidator) Struct(any) error {
	return errors.New("validation failed: mock validation error")
}

type passingValidator struct{}

func (passingValidator) Struct(any) error { return nil }

type customType struct {
	value string
}

func (c *customType) UnmarshalText(text []byte) error {
	if string(text) == "invalid" {
		return errors.New("invalid custom value")
	}
	c.value = "custom-" + string(text)
	return nil
}

var _ encoding.TextUnmarshaler = (*customType)(nil)

type mockResolver struct{}

func (m *mockResolver) GetPrefix() string {
	return "mock:"
}
func (m *mockResolver) Resolve(_ context.Context, value string) (string, error) {
	if value == "secret-value" {
		return "resolved-secret-from-mock", nil
	}
	return "", fmt.Errorf("mock secret %q not found", value)
}

var _ Resolver = (*mockResolver)(nil)

func TestIsNil(t *testing.T) {
	type testStruct struct{}
	var typedNil *testStruct = nil

	testCases := []struct {
		input    any
		name     string
		expected bool
	}{
		{name: "true nil interface", input: nil, expected: true},
		{name: "typed nil pointer", input: typedNil, expected: true},
		{name: "non-nil pointer", input: &testStruct{}, expected: false},
		{name: "zero value struct", input: testStruct{}, expected: false},
		{name: "non-nil map", input: make(map[string]string), expected: false},
		{name: "nil map", input: (map[string]string)(nil), expected: true},
		{name: "non-nil slice", input: make([]int, 0), expected: false},
		{name: "nil slice", input: ([]int)(nil), expected: true},
		{name: "empty string", input: "", expected: false},
		{name: "zero int", input: 0, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isNil(tc.input))
		})
	}
}

func TestLoad(t *testing.T) {
	type precedenceConfig struct {
		Source string `json:"source" yaml:"source" default:"default" env:"SOURCE" flag:"source"`
	}

	type typesConfig struct {
		Map       map[string]string `env:"MAP" flag:"map"`
		String    string            `env:"STRING"`
		Bytes     []byte            `env:"BYTES"`
		Slice     []string          `env:"SLICE" flag:"slice"`
		Int       int               `env:"INT"`
		Int64     int64             `env:"INT64"`
		Uint64    uint64            `env:"UINT64"`
		Float64   float64           `env:"FLOAT64"`
		Duration  time.Duration     `env:"DURATION"`
		BoolTrue  bool              `env:"BOOL_TRUE"`
		BoolFalse bool              `env:"BOOL_FALSE"`
	}

	type nestedConfig struct {
		Host string `env:"HOST"`
		Port int64  `env:"PORT" default:"5432"`
	}
	type prefixConfig struct {
		DB nestedConfig
	}

	type validationConfig struct {
		ZeroVal  *int    `env:"ZERO_VAL" validate:"required"`
		EmptyStr *string `env:"EMPTY_STR" validate:"required"`
		IsSet    string  `env:"IS_SET" validate:"required"`
		NotSet   string  `env:"NOT_SET" validate:"required"`
		Mode     string  `env:"MODE" validate:"oneof=dev prod qa"`
		Port     int     `env:"PORT" validate:"min=1024,max=65535"`
	}

	type customDelimiterConfig struct {
		Map   map[string]string `env:"MAP" delimiter:"|" separator:"="`
		Slice []string          `env:"SLICE" delimiter:";"`
	}

	type overwriteConfig struct {
		NoOverwrite  string `default:"default" env:"NO_OVERWRITE" overwrite:"false"`
		DoOverwrite  string `default:"default" env:"DO_OVERWRITE" overwrite:"true"`
		JSONWins     string `json:"json_wins" default:"default"`
		EnvStillWins string `json:"env_wins" default:"default" env:"ENV_WINS" overwrite:"true"`
	}

	type noInitConfig struct {
		Nested *nestedConfig `noinit:"true"`
	}
	type doInitConfig struct {
		Nested *nestedConfig
	}

	type customDecoderConfig struct {
		Decoder    customType `env:"DECODER"`
		Empty      customType `env:"EMPTY"`
		Invalid    customType `env:"INVALID"`
		StillUnset customType `env:"STILL_UNSET"`
	}

	type defaultExpansionConfig struct {
		Path string `default:"$HOME/app"`
	}

	type strictModeConfig struct {
		KnownField string `yaml:"knownField"`
	}

	type resolverConfig struct {
		APIKey string `default:"mock:secret-value"`
	}

	type progBasicConfig struct {
		Name string
		Port int
	}
	type progInner struct {
		A string
		B int
	}
	type progDeepConfig struct {
		PtrInner *progInner
		Inner    progInner
	}
	type progPrecedenceConfig struct {
		Val string `env:"VAL"`
	}
	type progTagDefaultConfig struct {
		Value string `default:"tagDefault"`
	}

	testCases := []struct {
		target        any
		want          any
		files         map[string]string
		env           map[string]string
		wantSources   map[string]string
		setup         func(t *testing.T)
		name          string
		arguments     []string
		errorContains []string
		opts          LoaderOptions
		wantErr       bool
	}{
		{
			name:   "programmatic: basic merge into zero target",
			target: &progBasicConfig{},
			opts:   LoaderOptions{ProgrammaticDefaults: &progBasicConfig{Name: "prog", Port: 8080}},
			want:   &progBasicConfig{Name: "prog", Port: 8080},
		},
		{
			name: "programmatic: deep merge with nested and pointer struct",
			target: &progDeepConfig{
				PtrInner: &progInner{A: "preset"},
			},
			opts: LoaderOptions{ProgrammaticDefaults: &progDeepConfig{Inner: progInner{A: "X"}, PtrInner: &progInner{B: 2}}},
			want: &progDeepConfig{Inner: progInner{A: "X", B: 0}, PtrInner: &progInner{A: "preset", B: 2}},
		},
		{
			name:   "programmatic: env overrides programmatic defaults",
			target: &progPrecedenceConfig{},
			opts:   LoaderOptions{ProgrammaticDefaults: &progPrecedenceConfig{Val: "from_prog"}},
			env:    map[string]string{"VAL": "from_env"},
			want:   &progPrecedenceConfig{Val: "from_env"},
		},
		{
			name:   "programmatic: placed after defaults overrides tag default",
			target: &progTagDefaultConfig{},
			opts: LoaderOptions{
				PassOrder:            []Pass{PassDefaults, PassProgrammatic, PassValidation},
				ProgrammaticDefaults: &progTagDefaultConfig{Value: "from_prog"},
			},
			want: &progTagDefaultConfig{Value: "from_prog"},
		},
		{
			name:   "programmatic: tag default overrides programmatic when order is reversed",
			target: &progTagDefaultConfig{},
			opts: LoaderOptions{
				PassOrder:            []Pass{PassProgrammatic, PassDefaults, PassValidation},
				ProgrammaticDefaults: &progTagDefaultConfig{Value: "from_prog"},
			},
			want: &progTagDefaultConfig{Value: "tagDefault"},
		},
		{
			name:   "overrides: basic override into zero target",
			target: &progBasicConfig{},
			opts:   LoaderOptions{ProgrammaticOverrides: &progBasicConfig{Name: "override", Port: 9090}},
			want:   &progBasicConfig{Name: "override", Port: 9090},
			wantSources: map[string]string{
				"Name": "programmatic_override",
				"Port": "programmatic_override",
			},
		},
		{
			name:   "overrides: override wins over env var",
			target: &progPrecedenceConfig{},
			opts:   LoaderOptions{ProgrammaticOverrides: &progPrecedenceConfig{Val: "from_override"}},
			env:    map[string]string{"VAL": "from_env"},
			want:   &progPrecedenceConfig{Val: "from_override"},
			wantSources: map[string]string{
				"Val": "programmatic_override",
			},
		},
		{
			name:   "overrides: zero-value override does not overwrite target",
			target: &progBasicConfig{},
			opts: LoaderOptions{
				ProgrammaticDefaults:  &progBasicConfig{Name: "default_name", Port: 3000},
				ProgrammaticOverrides: &progBasicConfig{Name: "override_name"},
			},
			want: &progBasicConfig{Name: "override_name", Port: 3000},
		},
		{
			name: "overrides: deep override with nested and pointer struct",
			target: &progDeepConfig{
				PtrInner: &progInner{A: "preset", B: 1},
			},
			opts: LoaderOptions{ProgrammaticOverrides: &progDeepConfig{
				Inner:    progInner{A: "overridden"},
				PtrInner: &progInner{B: 99},
			}},
			want: &progDeepConfig{
				Inner:    progInner{A: "overridden"},
				PtrInner: &progInner{A: "preset", B: 99},
			},
		},
		{
			name:   "overrides: nil overrides is a no-op",
			target: &progBasicConfig{},
			opts: LoaderOptions{
				ProgrammaticDefaults:  &progBasicConfig{Name: "default_name", Port: 3000},
				ProgrammaticOverrides: nil,
			},
			want: &progBasicConfig{Name: "default_name", Port: 3000},
		},
		{
			name:   "precedence: default only",
			target: &precedenceConfig{},
			want:   &precedenceConfig{Source: "default"},
			wantSources: map[string]string{
				"Source": "default",
			},
		},
		{
			name:   "precedence: yaml overrides default",
			target: &precedenceConfig{},
			files:  map[string]string{"config.yaml": `source: yaml`},
			want:   &precedenceConfig{Source: "yaml"},
			wantSources: map[string]string{
				"Source": "file: config.yaml",
			},
		},
		{
			name:   "precedence: env overrides yaml",
			target: &precedenceConfig{},
			files:  map[string]string{"config.yaml": `source: yaml`},
			env:    map[string]string{"SOURCE": "env"},
			want:   &precedenceConfig{Source: "env"},
			wantSources: map[string]string{
				"Source": "env",
			},
		},
		{
			name:      "precedence: flag overrides all",
			target:    &precedenceConfig{},
			files:     map[string]string{"config.yaml": `source: yaml`},
			env:       map[string]string{"SOURCE": "env"},
			arguments: []string{"--source", "flag"},
			want:      &precedenceConfig{Source: "flag"},
			wantSources: map[string]string{
				"Source": "flag: -source",
			},
		},
		{
			name:   "types: all primitives",
			target: &typesConfig{},
			env: map[string]string{
				"STRING":     "hello world",
				"INT":        "-42",
				"INT64":      "-12345",
				"UINT64":     "67890",
				"FLOAT64":    "3.14159",
				"BOOL_TRUE":  "true",
				"BOOL_FALSE": "false",
				"DURATION":   "1m30s",
				"BYTES":      "byte slice",
				"SLICE":      "a, b, c",
				"MAP":        "key1:val1,key2:val2",
			},
			want: &typesConfig{
				String:    "hello world",
				Int:       -42,
				Int64:     -12345,
				Uint64:    67890,
				Float64:   3.14159,
				BoolTrue:  true,
				BoolFalse: false,
				Duration:  90 * time.Second,
				Bytes:     []byte("byte slice"),
				Slice:     []string{"a", "b", "c"},
				Map:       map[string]string{"key1": "val1", "key2": "val2"},
			},
		},
		{
			name:   "types: flags for slice and map",
			target: &typesConfig{},
			arguments: []string{
				"--slice", "flagA,flagB",
				"--map", "flagKey:flagVal",
			},
			want: &typesConfig{
				Slice: []string{"flagA", "flagB"},
				Map:   map[string]string{"flagKey": "flagVal"},
			},
		},
		{
			name:   "features: resolver pass",
			target: &resolverConfig{},
			opts:   LoaderOptions{Resolvers: []Resolver{&mockResolver{}}},
			want:   &resolverConfig{APIKey: "resolved-secret-from-mock"},
			wantSources: map[string]string{
				"APIKey": "resolver:mock",
			},
		},
		{
			name:   "features: custom delimiters",
			target: &customDelimiterConfig{},
			env: map[string]string{
				"SLICE": "a; b; c",
				"MAP":   "k1=v1|k2=v2",
			},
			want: &customDelimiterConfig{
				Slice: []string{"a", "b", "c"},
				Map:   map[string]string{"k1": "v1", "k2": "v2"},
			},
		},
		{
			name:   "features: prefix for nested structs",
			target: &prefixConfig{},
			opts:   LoaderOptions{EnvPrefix: "DB_"},
			env:    map[string]string{"DB_HOST": "localhost"},
			want:   &prefixConfig{DB: nestedConfig{Host: "localhost", Port: 5432}},
		},
		{
			name:   "features: merge multiple files (yaml and json)",
			target: &precedenceConfig{},
			files: map[string]string{
				"base.yaml": `source: from_yaml`,
				"prod.json": `{"source": "from_json"}`,
			},
			opts: LoaderOptions{FilePaths: []string{"base.yaml", "prod.json"}},
			want: &precedenceConfig{Source: "from_json"},
		},
		{
			name:          "features: strict mode fails on unknown field",
			target:        &strictModeConfig{},
			files:         map[string]string{"config.yaml": "knownField: hello\nunknownField: world"},
			opts:          LoaderOptions{StrictFile: true},
			wantErr:       true,
			errorContains: []string{"unknown configuration keys found: unknownField"},
		},
		{
			name:   "features: strict mode succeeds when disabled",
			target: &strictModeConfig{},
			files:  map[string]string{"config.yaml": "knownField: hello\nunknownField: world"},
			opts:   LoaderOptions{StrictFile: false},
			want:   &strictModeConfig{KnownField: "hello"},
		},
		{
			name:    "features: validation fails on missing required",
			target:  &validationConfig{},
			env:     map[string]string{"IS_SET": "value", "PORT": "8080", "MODE": "dev"},
			opts:    LoaderOptions{Validator: failingValidator{}},
			wantErr: true,
			errorContains: []string{
				"validation failed",
			},
		},
		{
			name:    "features: validation fails on rule",
			target:  &validationConfig{},
			env:     map[string]string{"IS_SET": "v", "NOT_SET": "v", "PORT": "80", "MODE": "staging"},
			opts:    LoaderOptions{Validator: failingValidator{}},
			wantErr: true,
			errorContains: []string{
				"validation failed",
			},
		},
		{
			name:   "features: validation succeeds with zero values",
			target: &validationConfig{},
			opts:   LoaderOptions{Validator: passingValidator{}},
			env: map[string]string{
				"IS_SET":    "value",
				"NOT_SET":   "value too",
				"ZERO_VAL":  "0",
				"EMPTY_STR": "",
				"PORT":      "8080",
				"MODE":      "prod",
			},
			want: &validationConfig{
				IsSet:    "value",
				NotSet:   "value too",
				ZeroVal:  func(i int) *int { return &i }(0),
				EmptyStr: func(s string) *string { return &s }(""),
				Port:     8080,
				Mode:     "prod",
			},
		},
		{
			name: "features: overwrite behaviour",
			target: &overwriteConfig{
				NoOverwrite: "preset",
				DoOverwrite: "preset",
			},
			env: map[string]string{
				"NO_OVERWRITE": "env",
				"DO_OVERWRITE": "env",
				"ENV_WINS":     "env",
			},
			files: map[string]string{"config.json": `{"json_wins": "json", "env_wins": "json"}`},
			want: &overwriteConfig{
				NoOverwrite:  "preset",
				DoOverwrite:  "env",
				JSONWins:     "json",
				EnvStillWins: "env",
			},
		},
		{
			name:   "features: noinit keeps pointer nil",
			target: &noInitConfig{},
			want:   &noInitConfig{Nested: nil},
		},
		{
			name:   "features: default init creates struct",
			target: &doInitConfig{},
			want:   &doInitConfig{Nested: &nestedConfig{Port: 5432}},
		},
		{
			name:   "features: custom decoders",
			target: &customDecoderConfig{},
			env: map[string]string{
				"DECODER": "hello",
				"EMPTY":   "",
			},
			want: &customDecoderConfig{
				Decoder:    customType{value: "custom-hello"},
				Empty:      customType{value: "custom-"},
				StillUnset: customType{},
			},
		},
		{
			name:          "features: custom decoder error",
			target:        &customDecoderConfig{},
			env:           map[string]string{"INVALID": "invalid"},
			wantErr:       true,
			errorContains: []string{"invalid custom value"},
		},
		{
			name: "features: default value expansion",
			setup: func(t *testing.T) {
				t.Setenv("HOME", "/users/test")
			},
			target: &defaultExpansionConfig{},
			want:   &defaultExpansionConfig{Path: "/users/test/app"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			opts := setupTestEnvironment(t, &tc)
			ctx, err := Load(context.Background(), tc.target, opts)

			if tc.wantErr {
				require.Error(t, err, "Expected an error, but got nil")
				if len(tc.errorContains) > 0 {
					for _, expectedErr := range tc.errorContains {
						assert.Contains(t, err.Error(), expectedErr, "Error message did not contain expected text")
					}
				}
				return
			}

			require.NoError(t, err, "Expected no error, but got one")
			assert.Equal(t, tc.want, tc.target, "Configuration struct mismatch")

			if len(tc.wantSources) > 0 {
				assert.Len(t, ctx.FieldSources, len(tc.wantSources), "Field source count mismatch")
				for key, wantSource := range tc.wantSources {
					gotSource, ok := ctx.FieldSources[key]
					require.True(t, ok, "Missing expected source for key %q", key)

					if strings.HasPrefix(wantSource, "file:") {
						wantBase := strings.TrimPrefix(wantSource, "file: ")
						gotParts := strings.SplitN(gotSource, ": ", 2)
						require.Len(t, gotParts, 2, "Invalid file source format for key %q: got %q", key, gotSource)
						gotBase := filepath.Base(gotParts[1])
						assert.Equal(t, wantBase, gotBase, "Field source file mismatch for key %q", key)
					} else {
						assert.Equal(t, wantSource, gotSource, "Field source mismatch for key %q", key)
					}
				}
			}
		})
	}
}

func TestSummarise(t *testing.T) {
	type summaryConfig struct {
		Server   string `flag:"server"`
		Password string `env:"PASSWORD" summary:"hide"`
		Mode     string `default:"dev"`
		Port     int    `env:"PORT"`
	}

	testCases := []struct {
		setupCtx    func() *LoadContext
		name        string
		wantExact   string
		wantContain []string
		wantErr     bool
	}{
		{
			name: "mixed sources with redaction",
			setupCtx: func() *LoadContext {
				config := &summaryConfig{
					Server:   "db.example.com",
					Port:     5432,
					Password: "super-secret-password",
					Mode:     "dev",
				}
				return &LoadContext{
					Target: config,
					FieldSources: map[string]string{
						"Server":   "flag: -server",
						"Port":     "env",
						"Password": "env",
						"Mode":     "default",
					},
				}
			},
			wantContain: []string{
				"--- Applied Configuration Summary ---",
				"[Source: env]",
				"Port                                     = 5432",
				"Password                                 = [REDACTED]",
				"[Source: flag: -server]",
				"Server                                   = db.example.com",
			},
		},
		{
			name: "only default values",
			setupCtx: func() *LoadContext {
				config := &summaryConfig{Mode: "dev"}
				return &LoadContext{
					Target: config,
					FieldSources: map[string]string{
						"Mode": "default",
					},
				}
			},
			wantExact: "No user-configured values were set. Using all defaults.",
		},
		{
			name: "no sources at all",
			setupCtx: func() *LoadContext {
				return &LoadContext{
					Target:       &summaryConfig{},
					FieldSources: map[string]string{},
				}
			},
			wantExact: "No user-configured values were set. Using all defaults.",
		},
		{
			name: "invalid context",
			setupCtx: func() *LoadContext {
				return nil
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.setupCtx()
			summary, err := Summarise(ctx)

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tc.wantExact != "" {
				assert.Equal(t, tc.wantExact, summary)
			}

			for _, expected := range tc.wantContain {
				assert.Contains(t, summary, expected)
			}
		})
	}
}

func resetGlobalState(t *testing.T) {
	t.Helper()
	ResetGlobalFlagCoordinator()
	ResetGlobalResolverRegistry()
	ResetDotEnvCache()
	clearResolutionPools()
}

func setupTestEnvironment(t *testing.T, tc *struct {
	target        any
	want          any
	files         map[string]string
	env           map[string]string
	wantSources   map[string]string
	setup         func(t *testing.T)
	name          string
	arguments     []string
	errorContains []string
	opts          LoaderOptions
	wantErr       bool
}) LoaderOptions {
	t.Helper()

	resetGlobalState(t)

	for key, value := range tc.env {
		t.Setenv(key, value)
	}

	originalArgs := os.Args
	os.Args = append([]string{originalArgs[0]}, tc.arguments...)
	t.Cleanup(func() {
		os.Args = originalArgs
	})

	opts := tc.opts
	if len(tc.files) > 0 {
		tempDir := t.TempDir()
		var tempFilePaths []string
		if len(opts.FilePaths) > 0 {
			for _, p := range opts.FilePaths {
				baseName := filepath.Base(p)
				content, ok := tc.files[baseName]
				if !ok {
					t.Fatalf("File %s specified in opts.FilePaths but not found in files map", baseName)
				}
				path := filepath.Join(tempDir, baseName)
				require.NoError(t, os.WriteFile(path, []byte(content), 0644))
				tempFilePaths = append(tempFilePaths, path)
			}
		} else {
			for filename, content := range tc.files {
				path := filepath.Join(tempDir, filename)
				require.NoError(t, os.WriteFile(path, []byte(content), 0644))
				tempFilePaths = append(tempFilePaths, path)
			}
		}
		opts.FilePaths = tempFilePaths
	}
	return opts
}
