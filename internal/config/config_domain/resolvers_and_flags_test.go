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
	"encoding/base64"
	"flag"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64ResolverResolveEmptyString(t *testing.T) {
	r := &Base64Resolver{}
	result, err := r.Resolve(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestBase64ResolverResolveMultilineContent(t *testing.T) {
	original := "line1\nline2\nline3"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))

	r := &Base64Resolver{}
	result, err := r.Resolve(context.Background(), encoded)
	require.NoError(t, err)
	assert.Equal(t, original, result)
}

func TestEnvResolverResolveWithSpecialCharacters(t *testing.T) {
	t.Setenv("TEST_SPECIAL_CHARS", "value with spaces & special=chars!")

	r := &EnvResolver{}
	result, err := r.Resolve(context.Background(), "TEST_SPECIAL_CHARS")
	require.NoError(t, err)
	assert.Equal(t, "value with spaces & special=chars!", result)
}

func TestStringSliceValueString(t *testing.T) {
	testCases := []struct {
		name     string
		slice    *[]string
		expected string
	}{
		{
			name:     "nil slice pointer",
			slice:    nil,
			expected: "",
		},
		{
			name:     "empty slice",
			slice:    func() *[]string { s := []string{}; return &s }(),
			expected: "",
		},
		{
			name:     "single element",
			slice:    func() *[]string { s := []string{"alpha"}; return &s }(),
			expected: "alpha",
		},
		{
			name:     "multiple elements joined by comma",
			slice:    func() *[]string { s := []string{"a", "b", "c"}; return &s }(),
			expected: "a,b,c",
		},
		{
			name:     "elements with spaces",
			slice:    func() *[]string { s := []string{"hello world", "foo bar"}; return &s }(),
			expected: "hello world,foo bar",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sv := &stringSliceValue{slice: tc.slice}
			assert.Equal(t, tc.expected, sv.String())
		})
	}
}

func TestStringSliceValueSet(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string clears slice",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single value",
			input:    "alpha",
			expected: []string{"alpha"},
		},
		{
			name:     "comma-separated values",
			input:    "a,b,c",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "values with spaces are not trimmed by Set",
			input:    "a, b, c",
			expected: []string{"a", " b", " c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var s []string
			sv := &stringSliceValue{slice: &s}

			err := sv.Set(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, s)
		})
	}
}

func TestStringSliceValueSetOverwritesPreviousValue(t *testing.T) {
	s := []string{"old1", "old2"}
	sv := &stringSliceValue{slice: &s}

	err := sv.Set("new1,new2,new3")
	require.NoError(t, err)
	assert.Equal(t, []string{"new1", "new2", "new3"}, s)
}

func TestStringMapValueString(t *testing.T) {
	testCases := []struct {
		name     string
		sMap     *map[string]string
		tags     reflect.StructTag
		contains []string
	}{
		{
			name: "nil map pointer",
			sMap: nil,
		},
		{
			name: "empty map",
			sMap: func() *map[string]string { m := map[string]string{}; return &m }(),
		},
		{
			name: "single entry with default delimiters",
			sMap: func() *map[string]string { m := map[string]string{"key": "val"}; return &m }(),
			tags: "",
			contains: []string{
				"key:val",
			},
		},
		{
			name: "single entry with custom delimiters",
			sMap: func() *map[string]string { m := map[string]string{"key": "val"}; return &m }(),
			tags: `delimiter:"|" separator:"="`,
			contains: []string{
				"key=val",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mv := &stringMapValue{sMap: tc.sMap, tags: tc.tags}
			result := mv.String()

			if tc.sMap == nil || len(*tc.sMap) == 0 {
				assert.Equal(t, "", result)
			} else {
				for _, expected := range tc.contains {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

func TestStringMapValueStringMultipleEntries(t *testing.T) {
	m := map[string]string{"a": "1", "b": "2"}
	mv := &stringMapValue{sMap: &m, tags: ""}
	result := mv.String()

	assert.Contains(t, result, "a:1")
	assert.Contains(t, result, "b:2")
	assert.Contains(t, result, ",")
}

func TestStringMapValueSet(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		input    string
		tags     reflect.StructTag
		wantErr  bool
	}{
		{
			name:     "single key-value pair with default delimiters",
			input:    "host:localhost",
			expected: map[string]string{"host": "localhost"},
		},
		{
			name:     "multiple key-value pairs",
			input:    "a:1,b:2",
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "custom delimiters",
			input:    "a=1|b=2",
			tags:     `delimiter:"|" separator:"="`,
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:    "invalid format missing separator",
			input:   "noseparator",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]string{}
			mv := &stringMapValue{sMap: &m, tags: tc.tags}

			err := mv.Set(tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, m)
			}
		})
	}
}

func TestProcessDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		tag      reflect.StructTag
		initial  string
		expected string
		wantErr  bool
	}{
		{
			name:     "sets default when tag is present",
			tag:      `default:"hello"`,
			initial:  "",
			expected: "hello",
		},
		{
			name:     "no-op when no default tag",
			tag:      `env:"FOO"`,
			initial:  "untouched",
			expected: "untouched",
		},
		{
			name:     "empty default tag sets empty string",
			tag:      `default:""`,
			initial:  "old",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.initial
			value := reflect.ValueOf(&s).Elem()
			field := &reflect.StructField{
				Name: "TestField",
				Tag:  tc.tag,
			}

			err := processDefaults(field, value, "", "")
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, s)
			}
		})
	}
}

func TestProcessDefaultsExpandsEnvVars(t *testing.T) {
	t.Setenv("MY_TEST_VAR", "/custom/path")

	s := ""
	value := reflect.ValueOf(&s).Elem()
	field := &reflect.StructField{
		Name: "Path",
		Tag:  `default:"$MY_TEST_VAR/app"`,
	}

	err := processDefaults(field, value, "", "")
	require.NoError(t, err)
	assert.Equal(t, "/custom/path/app", s)
}

func TestProcessDefaultsIntegerField(t *testing.T) {
	var port int
	value := reflect.ValueOf(&port).Elem()
	field := &reflect.StructField{
		Name: "Port",
		Tag:  `default:"8080"`,
	}

	err := processDefaults(field, value, "", "")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)
}

func TestMakeFlagAttributionProcessor(t *testing.T) {
	t.Run("records source when flag name matches", func(t *testing.T) {
		ctx := &LoadContext{
			FieldSources: make(map[string]string),
		}
		visited := &flag.Flag{Name: "host"}
		processor := makeFlagAttributionProcessor(ctx, visited)

		field := &reflect.StructField{
			Name: "Host",
			Tag:  `flag:"host"`,
		}
		err := processor(field, reflect.Value{}, "", "Host")
		require.NoError(t, err)
		assert.Equal(t, "flag: -host", ctx.FieldSources["Host"])
	})

	t.Run("does not record source when flag name does not match", func(t *testing.T) {
		ctx := &LoadContext{
			FieldSources: make(map[string]string),
		}
		visited := &flag.Flag{Name: "port"}
		processor := makeFlagAttributionProcessor(ctx, visited)

		field := &reflect.StructField{
			Name: "Host",
			Tag:  `flag:"host"`,
		}
		err := processor(field, reflect.Value{}, "", "Host")
		require.NoError(t, err)
		assert.Empty(t, ctx.FieldSources)
	})

	t.Run("does not record source when no flag tag", func(t *testing.T) {
		ctx := &LoadContext{
			FieldSources: make(map[string]string),
		}
		visited := &flag.Flag{Name: "host"}
		processor := makeFlagAttributionProcessor(ctx, visited)

		field := &reflect.StructField{
			Name: "Host",
			Tag:  `env:"HOST"`,
		}
		err := processor(field, reflect.Value{}, "", "Host")
		require.NoError(t, err)
		assert.Empty(t, ctx.FieldSources)
	})
}

func TestPassString(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		pass     Pass
	}{
		{name: "defaults", pass: PassDefaults, expected: "defaults"},
		{name: "files", pass: PassFiles, expected: "files"},
		{name: "dotenv", pass: PassDotEnv, expected: "dotenv"},
		{name: "env", pass: PassEnv, expected: "env"},
		{name: "flags", pass: PassFlags, expected: "flags"},
		{name: "resolvers", pass: PassResolvers, expected: "resolvers"},
		{name: "validation", pass: PassValidation, expected: "validation"},
		{name: "programmatic", pass: PassProgrammatic, expected: "programmatic"},
		{name: "unknown pass value", pass: Pass(99), expected: "Pass(99)"},
		{name: "negative pass value", pass: Pass(-1), expected: "Pass(-1)"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.pass.String())
		})
	}
}

func TestGetFileReaderDefault(t *testing.T) {
	reader := getFileReader(nil)
	require.NotNil(t, reader)
	_, isOS := reader.(osFileReader)
	assert.True(t, isOS, "nil input should return osFileReader")
}

func TestGetFileReaderCustom(t *testing.T) {
	custom := &stubFileReader{}
	reader := getFileReader(custom)
	assert.Same(t, custom, reader)
}

type stubFileReader struct{}

func (m *stubFileReader) ReadFile(_ string) ([]byte, error) {
	return nil, nil
}

func TestGetFlagCoordinatorDefault(t *testing.T) {
	ResetGlobalFlagCoordinator()
	defer ResetGlobalFlagCoordinator()

	fc := getFlagCoordinator(nil)
	require.NotNil(t, fc)
	assert.Same(t, GetGlobalFlagCoordinator(), fc)
}

func TestGetFlagCoordinatorCustom(t *testing.T) {
	custom := newFlagCoordinator()
	fc := getFlagCoordinator(custom)
	assert.Same(t, custom, fc)
}

func TestGetResolverRegistryDefault(t *testing.T) {
	ResetGlobalResolverRegistry()
	defer ResetGlobalResolverRegistry()

	rr := getResolverRegistry(nil)
	require.NotNil(t, rr)
	assert.Same(t, GetGlobalResolverRegistry(), rr)
}

func TestGetResolverRegistryCustom(t *testing.T) {
	custom := newResolverRegistry()
	rr := getResolverRegistry(custom)
	assert.Same(t, custom, rr)
}

func TestShouldUseFlagCoordinator(t *testing.T) {
	testCases := []struct {
		name      string
		passOrder []Pass
		expected  bool
	}{
		{
			name:      "empty pass order defaults to true",
			passOrder: nil,
			expected:  true,
		},
		{
			name:      "pass order with PassFlags returns true",
			passOrder: []Pass{PassDefaults, PassFlags, PassValidation},
			expected:  true,
		},
		{
			name:      "pass order without PassFlags returns false",
			passOrder: []Pass{PassDefaults, PassEnv, PassValidation},
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := &Loader{
				opts: LoaderOptions{
					PassOrder: tc.passOrder,
				},
			}
			assert.Equal(t, tc.expected, l.shouldUseFlagCoordinator())
		})
	}
}

func TestBuildPassesDefaultOrder(t *testing.T) {
	resetGlobalState(t)

	l := NewLoader(LoaderOptions{})
	defer l.Close()

	passes, err := l.buildPasses()
	require.NoError(t, err)
	assert.Len(t, passes, 9)
	assert.Equal(t, "defaults", passes[0].name)
	assert.Equal(t, "programmatic", passes[1].name)
	assert.Equal(t, "files", passes[2].name)
	assert.Equal(t, "dotenv", passes[3].name)
	assert.Equal(t, "env", passes[4].name)
	assert.Equal(t, "flags", passes[5].name)
	assert.Equal(t, "resolvers", passes[6].name)
	assert.Equal(t, "programmatic_overrides", passes[7].name)
	assert.Equal(t, "validation", passes[8].name)
}

func TestBuildPassesCustomOrder(t *testing.T) {
	resetGlobalState(t)

	l := NewLoader(LoaderOptions{
		PassOrder: []Pass{PassEnv, PassDefaults},
	})
	defer l.Close()

	passes, err := l.buildPasses()
	require.NoError(t, err)
	require.Len(t, passes, 2)
	assert.Equal(t, "env", passes[0].name)
	assert.Equal(t, "defaults", passes[1].name)
}

func TestBuildPassesInvalidPass(t *testing.T) {
	resetGlobalState(t)

	l := NewLoader(LoaderOptions{
		PassOrder: []Pass{Pass(99)},
	})
	defer l.Close()

	_, err := l.buildPasses()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or unknown pass")
}

func TestCreateResolverCacheDefault(t *testing.T) {
	cache := createResolverCache(nil)
	require.NotNil(t, cache, "nil TTL should create cache with default TTL")
	cache.StopAllGoroutines()
}

func TestCreateResolverCacheDisabled(t *testing.T) {
	cache := createResolverCache(new(-1 * defaultResolverCacheTTL))
	assert.Nil(t, cache, "negative TTL should disable cache")
}

func TestWithDefaultResolvers(t *testing.T) {
	resetGlobalState(t)

	l := NewLoader(LoaderOptions{}, WithDefaultResolvers())
	defer l.Close()

	assert.Contains(t, l.resolverMap, "env:")
	assert.Contains(t, l.resolverMap, "base64:")
	assert.Contains(t, l.resolverMap, "file:")
}
