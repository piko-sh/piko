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
	"flag"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagCoordinatorNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "creates new coordinator with empty registrations",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				require.NotNil(t, fc)
				assert.NotNil(t, fc.flagSet)
				assert.Empty(t, fc.registrations)
				assert.False(t, fc.parsed)
			},
		},
		{
			name: "each call creates independent coordinator",
			test: func(t *testing.T) {
				fc1 := newFlagCoordinator()
				fc2 := newFlagCoordinator()
				assert.NotSame(t, fc1, fc2)
				assert.NotSame(t, fc1.flagSet, fc2.flagSet)
			},
		},
		{
			name: "new coordinator is not parsed",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				assert.False(t, fc.IsParsed())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestFlagCoordinatorGlobalSingleton(t *testing.T) {
	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "returns same instance on multiple calls",
			test: func(t *testing.T) {
				ResetGlobalFlagCoordinator()
				defer ResetGlobalFlagCoordinator()

				fc1 := GetGlobalFlagCoordinator()
				fc2 := GetGlobalFlagCoordinator()
				assert.Same(t, fc1, fc2)
			},
		},
		{
			name: "reset creates new instance",
			test: func(t *testing.T) {
				ResetGlobalFlagCoordinator()
				fc1 := GetGlobalFlagCoordinator()

				ResetGlobalFlagCoordinator()
				fc2 := GetGlobalFlagCoordinator()

				assert.NotSame(t, fc1, fc2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestFlagCoordinatorRegisterStruct(t *testing.T) {
	type testConfig struct {
		Host string `flag:"host" default:"localhost"`
		Port int    `flag:"port" default:"8080"`
	}

	testCases := []struct {
		setupFunc  func(*FlagCoordinator)
		name       string
		prefix     string
		errMessage string
		wantErr    bool
	}{
		{
			name:   "registers struct without prefix",
			prefix: "",
		},
		{
			name:   "registers struct with prefix",
			prefix: "app",
		},
		{
			name:   "registers struct with prefix with separator",
			prefix: "app.",
		},
		{
			name:   "allows duplicate registration with same prefix",
			prefix: "app",
			setupFunc: func(fc *FlagCoordinator) {
				config := &testConfig{}
				loader := &Loader{opts: LoaderOptions{}}
				_ = fc.RegisterStruct(config, "app", loader)
			},
		},
		{
			name:   "fails when already parsed",
			prefix: "test",
			setupFunc: func(fc *FlagCoordinator) {
				_ = fc.Parse()
			},
			wantErr:    true,
			errMessage: "cannot register struct after flags have been parsed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := newFlagCoordinator()

			if tc.setupFunc != nil {
				tc.setupFunc(fc)
			}

			config := &testConfig{}
			loader := &Loader{opts: LoaderOptions{}}
			err := fc.RegisterStruct(config, tc.prefix, loader)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlagCoordinatorParse(t *testing.T) {
	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "parse is idempotent",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()

				err1 := fc.Parse()
				require.NoError(t, err1)

				err2 := fc.Parse()
				require.NoError(t, err2)

				assert.True(t, fc.IsParsed())
			},
		},
		{
			name: "parse sets parsed flag",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				assert.False(t, fc.IsParsed())

				_ = fc.Parse()
				assert.True(t, fc.IsParsed())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestFlagCoordinatorReset(t *testing.T) {
	type testConfig struct {
		Host string `flag:"host" default:"localhost"`
	}

	testCases := []struct {
		test func(t *testing.T)
		name string
	}{
		{
			name: "reset clears parsed state",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				_ = fc.Parse()
				assert.True(t, fc.IsParsed())

				fc.Reset()
				assert.False(t, fc.IsParsed())
			},
		},
		{
			name: "reset clears registrations",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				config := &testConfig{}
				loader := &Loader{opts: LoaderOptions{}}
				_ = fc.RegisterStruct(config, "app", loader)

				fc.Reset()
				assert.Empty(t, fc.registrations)
			},
		},
		{
			name: "reset allows new registrations",
			test: func(t *testing.T) {
				fc := newFlagCoordinator()
				_ = fc.Parse()

				fc.Reset()

				config := &testConfig{}
				loader := &Loader{opts: LoaderOptions{}}
				err := fc.RegisterStruct(config, "new", loader)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, tc.test)
	}
}

func TestFlagCoordinatorGetVisitedFlags(t *testing.T) {
	type testConfig struct {
		Host string `flag:"host" default:"localhost"`
		Port int    `flag:"port" default:"8080"`
	}

	testCases := []struct {
		name         string
		prefix       string
		queryPrefix  string
		wantFlagName string
	}{
		{
			name:         "returns flags without prefix when queried without prefix",
			prefix:       "",
			queryPrefix:  "",
			wantFlagName: "host",
		},
		{
			name:         "returns flags with prefix stripped when queried with matching prefix",
			prefix:       "app",
			queryPrefix:  "app",
			wantFlagName: "host",
		},
		{
			name:         "normalises query prefix without separator",
			prefix:       "app.",
			queryPrefix:  "app",
			wantFlagName: "host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := newFlagCoordinator()
			config := &testConfig{}
			loader := &Loader{opts: LoaderOptions{}}

			err := fc.RegisterStruct(config, tc.prefix, loader)
			require.NoError(t, err)

			visited := fc.GetVisitedFlags(tc.queryPrefix)

			assert.Empty(t, visited)
		})
	}
}

func TestFlagCoordinatorFlagTypes(t *testing.T) {
	type allTypesConfig struct {
		Map      map[string]string `flag:"map"`
		String   string            `flag:"str"`
		Slice    []string          `flag:"slice"`
		Int      int               `flag:"int"`
		Int64    int64             `flag:"int64"`
		Uint     uint              `flag:"uint"`
		Uint64   uint64            `flag:"uint64"`
		Duration time.Duration     `flag:"dur"`
		Bool     bool              `flag:"bool"`
	}

	fc := newFlagCoordinator()
	config := &allTypesConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	err := fc.RegisterStruct(config, "", loader)
	require.NoError(t, err)

	flagNames := []string{"str", "int", "int64", "uint", "uint64", "bool", "dur", "slice", "map"}
	for _, name := range flagNames {
		f := fc.flagSet.Lookup(name)
		assert.NotNil(t, f, "Flag %s should be registered", name)
	}
}

func TestFlagCoordinatorPointerFlagTypes(t *testing.T) {
	type pointerConfig struct {
		String   *string        `flag:"str"`
		Int      *int           `flag:"int"`
		Int64    *int64         `flag:"int64"`
		Uint     *uint          `flag:"uint"`
		Uint64   *uint64        `flag:"uint64"`
		Bool     *bool          `flag:"bool"`
		Duration *time.Duration `flag:"dur"`
	}

	t.Run("pointer fields register as flags", func(t *testing.T) {
		fc := newFlagCoordinator()
		config := &pointerConfig{}
		loader := &Loader{opts: LoaderOptions{}}

		err := fc.RegisterStruct(config, "", loader)
		require.NoError(t, err)

		flagNames := []string{"str", "int", "int64", "uint", "uint64", "bool", "dur"}
		for _, name := range flagNames {
			f := fc.flagSet.Lookup(name)
			assert.NotNil(t, f, "Pointer flag %s should be registered", name)
		}
	})

	t.Run("pointer flags parse values correctly", func(t *testing.T) {
		fc := newFlagCoordinator()
		config := &pointerConfig{}
		loader := &Loader{opts: LoaderOptions{}}

		err := fc.RegisterStruct(config, "", loader)
		require.NoError(t, err)

		err = fc.flagSet.Parse([]string{
			"-str=hello",
			"-int=42",
			"-int64=99",
			"-uint=7",
			"-uint64=8",
			"-bool=true",
			"-dur=5s",
		})
		require.NoError(t, err)

		require.NotNil(t, config.String)
		assert.Equal(t, "hello", *config.String)
		require.NotNil(t, config.Int)
		assert.Equal(t, 42, *config.Int)
		require.NotNil(t, config.Int64)
		assert.Equal(t, int64(99), *config.Int64)
		require.NotNil(t, config.Uint)
		assert.Equal(t, uint(7), *config.Uint)
		require.NotNil(t, config.Uint64)
		assert.Equal(t, uint64(8), *config.Uint64)
		require.NotNil(t, config.Bool)
		assert.Equal(t, true, *config.Bool)
		require.NotNil(t, config.Duration)
		assert.Equal(t, 5*time.Second, *config.Duration)
	})

	t.Run("pointer flags with prefix parse correctly", func(t *testing.T) {
		fc := newFlagCoordinator()
		config := &pointerConfig{}
		loader := &Loader{opts: LoaderOptions{}}

		err := fc.RegisterStruct(config, "app", loader)
		require.NoError(t, err)

		f := fc.flagSet.Lookup("app.str")
		assert.NotNil(t, f, "Prefixed pointer flag should be registered")

		err = fc.flagSet.Parse([]string{"-app.str=world", "-app.bool=false"})
		require.NoError(t, err)

		require.NotNil(t, config.String)
		assert.Equal(t, "world", *config.String)
		require.NotNil(t, config.Bool)
		assert.Equal(t, false, *config.Bool)
	})
}

func TestFlagCoordinatorPrefixIsolation(t *testing.T) {
	type config struct {
		Host string `flag:"host"`
	}

	fc := newFlagCoordinator()
	loader := &Loader{opts: LoaderOptions{}}

	cfg1 := &config{}
	cfg2 := &config{}

	err := fc.RegisterStruct(cfg1, "app1", loader)
	require.NoError(t, err)

	err = fc.RegisterStruct(cfg2, "app2", loader)
	require.NoError(t, err)

	assert.NotNil(t, fc.flagSet.Lookup("app1.host"))
	assert.NotNil(t, fc.flagSet.Lookup("app2.host"))
}

func TestBuildFlagUsage(t *testing.T) {
	testCases := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "usage only",
			tag:      `usage:"Host address"`,
			expected: "Host address",
		},
		{
			name:     "usage with env",
			tag:      `usage:"Host address" env:"HOST"`,
			expected: "Host address (env: HOST)",
		},
		{
			name:     "usage with default",
			tag:      `usage:"Host address" default:"localhost"`,
			expected: "Host address [default: localhost]",
		},
		{
			name:     "usage with env and default",
			tag:      `usage:"Host address" env:"HOST" default:"localhost"`,
			expected: "Host address (env: HOST) [default: localhost]",
		},
		{
			name:     "empty usage with env and default",
			tag:      `env:"HOST" default:"localhost"`,
			expected: " (env: HOST) [default: localhost]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tags := parseTagString(tc.tag)
			usage := buildFlagUsage(tags)
			assert.Equal(t, tc.expected, usage)
		})
	}
}

func parseTagString(tag string) reflect.StructTag {

	return reflect.StructTag(tag)
}

func TestFilterTestFlags(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		expected  []string
	}{
		{
			name:      "filters test.run flag",
			arguments: []string{"-test.run=TestFoo", "-host=localhost"},
			expected:  []string{"-host=localhost"},
		},
		{
			name:      "filters test.v flag",
			arguments: []string{"-test.v", "-port=8080"},
			expected:  []string{"-port=8080"},
		},
		{
			name:      "filters update flag",
			arguments: []string{"-update", "-host=localhost"},
			expected:  []string{"-host=localhost"},
		},
		{
			name:      "keeps non-test flags",
			arguments: []string{"-host=localhost", "-port=8080"},
			expected:  []string{"-host=localhost", "-port=8080"},
		},
		{
			name:      "handles empty arguments",
			arguments: []string{},
			expected:  []string{},
		},
		{
			name:      "filters multiple test flags",
			arguments: []string{"-test.run=Foo", "-test.v", "-update", "-host=localhost"},
			expected:  []string{"-host=localhost"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterTestFlags(tc.arguments)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDefineFlagOnFlagSet(t *testing.T) {
	testCases := []struct {
		setup   func() (any, reflect.Value)
		name    string
		wantErr bool
	}{
		{
			name: "string flag",
			setup: func() (any, reflect.Value) {
				s := "default"
				return &s, reflect.ValueOf(s)
			},
		},
		{
			name: "int flag",
			setup: func() (any, reflect.Value) {
				i := 42
				return &i, reflect.ValueOf(i)
			},
		},
		{
			name: "int64 flag",
			setup: func() (any, reflect.Value) {
				i := int64(42)
				return &i, reflect.ValueOf(i)
			},
		},
		{
			name: "uint flag",
			setup: func() (any, reflect.Value) {
				u := uint(42)
				return &u, reflect.ValueOf(u)
			},
		},
		{
			name: "uint64 flag",
			setup: func() (any, reflect.Value) {
				u := uint64(42)
				return &u, reflect.ValueOf(u)
			},
		},
		{
			name: "bool flag",
			setup: func() (any, reflect.Value) {
				b := true
				return &b, reflect.ValueOf(b)
			},
		},
		{
			name: "duration flag",
			setup: func() (any, reflect.Value) {
				d := 5 * time.Second
				return &d, reflect.ValueOf(d)
			},
		},
		{
			name: "string slice flag",
			setup: func() (any, reflect.Value) {
				s := []string{"a", "b"}
				return &s, reflect.ValueOf(s)
			},
		},
		{
			name: "string map flag",
			setup: func() (any, reflect.Value) {
				m := map[string]string{"key": "value"}
				return &m, reflect.ValueOf(m)
			},
		},
		{
			name: "unsupported type is silently skipped",
			setup: func() (any, reflect.Value) {
				f := 3.14
				return &f, reflect.ValueOf(f)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			ptr, value := tc.setup()

			err := defineFlagOnFlagSet(fs, "testflag", ptr, value, "", "test usage")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFilterKnownFlags(t *testing.T) {
	testCases := []struct {
		name      string
		arguments []string
		expected  []string
	}{
		{
			name:      "keeps known flags with equals syntax",
			arguments: []string{"-host=localhost", "-port=8080"},
			expected:  []string{"-host=localhost", "-port=8080"},
		},
		{
			name:      "keeps known flags with double dash",
			arguments: []string{"--host=localhost", "--port=8080"},
			expected:  []string{"--host=localhost", "--port=8080"},
		},
		{
			name:      "filters unknown flags with equals syntax",
			arguments: []string{"-host=localhost", "-unknown=value", "-port=8080"},
			expected:  []string{"-host=localhost", "-port=8080"},
		},
		{
			name:      "filters unknown flags with space syntax",
			arguments: []string{"-host=localhost", "-unknown", "value", "-port=8080"},
			expected:  []string{"-host=localhost", "-port=8080"},
		},
		{
			name:      "filters unknown double-dash flags",
			arguments: []string{"--tcp", "--verbose", "-host=localhost"},
			expected:  []string{"-host=localhost"},
		},
		{
			name:      "keeps non-flag arguments",
			arguments: []string{"-host=localhost", "positional", "-port=8080"},
			expected:  []string{"-host=localhost", "positional", "-port=8080"},
		},
		{
			name:      "preserves everything after double dash",
			arguments: []string{"-host=localhost", "--", "-unknown=value", "argument"},
			expected:  []string{"-host=localhost", "--", "-unknown=value", "argument"},
		},
		{
			name:      "handles empty arguments",
			arguments: []string{},
			expected:  []string{},
		},
		{
			name:      "handles only unknown flags",
			arguments: []string{"-tcp", "--debug", "--verbose"},
			expected:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fc := newFlagCoordinator()

			var host string
			var port int
			fc.flagSet.StringVar(&host, "host", "", "host")
			fc.flagSet.IntVar(&port, "port", 0, "port")

			result := fc.filterKnownFlags(tc.arguments)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractFlagName(t *testing.T) {
	testCases := []struct {
		name         string
		argument     string
		expectedName string
		expectedHas  bool
	}{
		{
			name:         "single dash with equals",
			argument:     "-host=localhost",
			expectedName: "host",
			expectedHas:  true,
		},
		{
			name:         "double dash with equals",
			argument:     "--port=8080",
			expectedName: "port",
			expectedHas:  true,
		},
		{
			name:         "single dash without equals",
			argument:     "-verbose",
			expectedName: "verbose",
			expectedHas:  false,
		},
		{
			name:         "double dash without equals",
			argument:     "--tcp",
			expectedName: "tcp",
			expectedHas:  false,
		},
		{
			name:         "equals with empty value",
			argument:     "-host=",
			expectedName: "host",
			expectedHas:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, hasValue := extractFlagName(tc.argument)
			assert.Equal(t, tc.expectedName, name)
			assert.Equal(t, tc.expectedHas, hasValue)
		})
	}
}
