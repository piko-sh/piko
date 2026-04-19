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
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestParseDotEnvStream(t *testing.T) {

	t.Setenv("HOME", "/users/test")

	testCases := []struct {
		expected map[string]string
		setup    func(t *testing.T)
		name     string
		input    string
		wantErr  bool
	}{
		{
			name:     "Basic assignment",
			input:    "KEY=VALUE",
			expected: map[string]string{"KEY": "VALUE"},
		},
		{
			name:     "Comments and empty lines",
			input:    "# This is a comment\n\nKEY=VALUE\n\n ANOTHER=KEY ",
			expected: map[string]string{"KEY": "VALUE", "ANOTHER": "KEY"},
		},
		{
			name:     "Double quoted value with spaces",
			input:    `DB_HOST="local host"`,
			expected: map[string]string{"DB_HOST": "local host"},
		},
		{
			name:     "Single quoted value is literal",
			input:    `SINGLE_QUOTED='$HOME'`,
			expected: map[string]string{"SINGLE_QUOTED": "$HOME"},
		},
		{
			name:     "Multiline value",
			input:    "PRIVATE_KEY=\"-----BEGIN KEY-----\nline two\n-----END KEY-----\"",
			expected: map[string]string{"PRIVATE_KEY": "-----BEGIN KEY-----\nline two\n-----END KEY-----"},
		},
		{
			name:     "Inline comments",
			input:    "PORT=8080 # The port to listen on",
			expected: map[string]string{"PORT": "8080"},
		},
		{
			name:     "Password with special characters",
			input:    `PASSWORD="ab#c=d$e"`,
			expected: map[string]string{"PASSWORD": "ab#c=d$e"},
		},
		{
			name:     "Self-referential variable expansion",
			input:    "HOST=localhost\nPORT=3000\nURL=http://${HOST}:${PORT}/api",
			expected: map[string]string{"HOST": "localhost", "PORT": "3000", "URL": "http://localhost:3000/api"},
		},
		{
			name:     "Fallback to shell environment variable expansion",
			input:    "APP_PATH=$HOME/my-app",
			expected: map[string]string{"APP_PATH": "/users/test/my-app"},
		},
		{
			name:     "No equals sign",
			input:    "INVALID_LINE",
			expected: map[string]string{},
		},

		{
			name:     "Double quoted with escape sequences",
			input:    `MESSAGE="Hello\nWorld\$123\"\\"`,
			expected: map[string]string{"MESSAGE": "Hello\nWorld$123\"\\"},
		},
		{
			name:     "Invalid escape sequences treated literally",
			input:    `TEXT="invalid\x\z sequences"`,
			expected: map[string]string{"TEXT": "invalid\\x\\z sequences"},
		},
		{
			name:     "Single quoted ignores escape sequences",
			input:    `LITERAL='no\nescapes\$here'`,
			expected: map[string]string{"LITERAL": "no\\nescapes\\$here"},
		},

		{
			name:     "Unterminated double quoted string",
			input:    `UNTERMINATED="missing quote`,
			expected: map[string]string{"UNTERMINATED": "missing quote"},
		},
		{
			name:     "Unterminated single quoted string",
			input:    `UNTERMINATED='missing quote`,
			expected: map[string]string{"UNTERMINATED": "missing quote"},
		},

		{
			name:     "Bare variable format $VAR",
			input:    "BASE=test\nVALUE=$BASE-suffix",
			expected: map[string]string{"BASE": "test", "VALUE": "test-suffix"},
		},
		{
			name:     "Mixed variable formats",
			input:    "A=one\nB=two\nMIXED=$A-${B}-end",
			expected: map[string]string{"A": "one", "B": "two", "MIXED": "$A-two-end"},
		},
		{
			name:     "Undefined variable expansion",
			input:    "VALUE=${UNDEFINED_VAR}",
			expected: map[string]string{"VALUE": ""},
		},
		{
			name:     "Self-referential variable (circular reference)",
			input:    "SELF=${SELF}",
			expected: map[string]string{"SELF": ""},
		},
		{
			name:     "Variable dependency chain",
			input:    "A=first\nB=${A}_second\nC=${B}_third",
			expected: map[string]string{"A": "first", "B": "first_second", "C": "first_second_third"},
		},
		{
			name:     "Deep variable nesting",
			input:    "L1=base\nL2=${L1}-2\nL3=${L2}-3\nL4=${L3}-4",
			expected: map[string]string{"L1": "base", "L2": "base-2", "L3": "base-2-3", "L4": "base-2-3-4"},
		},

		{
			name:     "Single letter variables treated as literal",
			input:    "VALUE=$A$B$C",
			expected: map[string]string{"VALUE": "$A$B$C"},
		},
		{
			name:  "Common single letter env var expansion (fallback)",
			input: "PATH_VALUE=$PATH",
			setup: func(t *testing.T) {
				t.Setenv("PATH", "/usr/bin:/bin")
			},
			expected: map[string]string{"PATH_VALUE": "/usr/bin:/bin"},
		},

		{
			name:     "Empty key",
			input:    "=value",
			expected: map[string]string{},
		},
		{
			name:     "Key with invalid characters",
			input:    "KEY@INVALID=value\nVALID_KEY=valid",
			expected: map[string]string{"VALID_KEY": "valid"},
		},
		{
			name:     "Multiple equals signs",
			input:    "KEY=value=with=equals",
			expected: map[string]string{"KEY": "value=with=equals"},
		},
		{
			name:     "Whitespace around equals",
			input:    "KEY1 = value1\n KEY2=  value2  \nKEY3 =value3",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2", "KEY3": "value3"},
		},

		{
			name:     "Mixed whitespace types",
			input:    "KEY1=value1\r\n\t  \r\nKEY2=value2",
			expected: map[string]string{"KEY1": "value1", "KEY2": "value2"},
		},
		{
			name:     "Comments with special characters",
			input:    "# Comment with $VAR and = symbols\nKEY=value # inline comment",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "Empty lines and comments only",
			input:    "\n\n# Just comments\n\n# More comments\n\n",
			expected: map[string]string{},
		},

		{
			name:     "Incomplete ${} pattern",
			input:    "VALUE=${INCOMPLETE",
			expected: map[string]string{"VALUE": "${INCOMPLETE"},
		},
		{
			name:     "Empty variable name in ${}",
			input:    "VALUE=${}",
			expected: map[string]string{"VALUE": "${}"},
		},
		{
			name:     "Dollar at end of line",
			input:    "VALUE=test$",
			expected: map[string]string{"VALUE": "test$"},
		},
		{
			name:     "Variable name with numbers",
			input:    "VAR123=base\nVALUE=$VAR123",
			expected: map[string]string{"VAR123": "base", "VALUE": "base"},
		},

		{
			name: "Database connection string",
			input: `DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASS="p@ssw0rd!#$"
DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/mydb"`,
			expected: map[string]string{
				"DB_HOST":      "localhost",
				"DB_PORT":      "5432",
				"DB_USER":      "admin",
				"DB_PASS":      "p@ssw0rd!#$",
				"DATABASE_URL": "postgresql://admin:p@ssw0rd!#$@localhost:5432/mydb",
			},
		},
		{
			name: "API configuration with nested variables",
			input: `ENV=production
API_HOST=${ENV}.api.example.com
API_PORT=443
API_VERSION=v1
BASE_URL=https://${API_HOST}:${API_PORT}
API_ENDPOINT=${BASE_URL}/${API_VERSION}`,
			expected: map[string]string{
				"ENV":          "production",
				"API_HOST":     "production.api.example.com",
				"API_PORT":     "443",
				"API_VERSION":  "v1",
				"BASE_URL":     "https://production.api.example.com:443",
				"API_ENDPOINT": "https://production.api.example.com:443/v1",
			},
		},

		{
			name:     "Variable expansion in single quotes (no expansion)",
			input:    "BASE=test\nVALUE='${BASE} $BASE'",
			expected: map[string]string{"BASE": "test", "VALUE": "${BASE} $BASE"},
		},
		{
			name: "Mixed quotes and variables",
			input: `VAR1=hello
VAR2="$VAR1 world"
VAR3='$VAR1 literal'`,
			expected: map[string]string{"VAR1": "hello", "VAR2": "hello world", "VAR3": "$VAR1 literal"},
		},

		{
			name:     "Unquoted value with spaces",
			input:    "MESSAGE=hello world with spaces",
			expected: map[string]string{"MESSAGE": "hello world with spaces"},
		},
		{
			name:     "Unquoted value with inline comment",
			input:    "PORT=8080 # This is the port",
			expected: map[string]string{"PORT": "8080"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			reader := strings.NewReader(tc.input)
			actual, err := parseDotEnvStream(reader)

			if (err != nil) != tc.wantErr {
				t.Fatalf("parseDotEnvStream() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("parseDotEnvStream() got = %v, want %v", actual, tc.expected)
			}
		})
	}
}

func TestParseDotEnvStreamErrors(t *testing.T) {
	testCases := []struct {
		reader  io.Reader
		name    string
		wantErr bool
	}{
		{
			name:    "Reader error",
			reader:  &errorReader{},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDotEnvStream(tc.reader)
			if (err != nil) != tc.wantErr {
				t.Errorf("parseDotEnvStream() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestParseDotEnvStreamLargeInput(t *testing.T) {

	var builder strings.Builder
	for i := range 1000 {
		builder.WriteString("VAR" + strconv.Itoa(i))
		builder.WriteString("=value")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteByte('\n')
	}

	reader := strings.NewReader(builder.String())
	result, err := parseDotEnvStream(reader)

	if err != nil {
		t.Fatalf("parseDotEnvStream() failed with large input: %v", err)
	}

	if len(result) != 1000 {
		t.Errorf("Expected 1000 variables, got %d", len(result))
	}
}

func TestVariableExpansionComplexScenarios(t *testing.T) {

	t.Setenv("TEST_ENV_VAR", "env_value")

	testCases := []struct {
		expected map[string]string
		name     string
		input    string
	}{
		{
			name: "Multiple variable references in one value",
			input: `A=one
B=two
C=three
RESULT=${A}_${B}_${C}`,
			expected: map[string]string{
				"A":      "one",
				"B":      "two",
				"C":      "three",
				"RESULT": "one_two_three",
			},
		},
		{
			name:  "Variable expansion with environment fallback",
			input: "VALUE=${TEST_ENV_VAR}_suffix",
			expected: map[string]string{
				"VALUE": "env_value_suffix",
			},
		},
		{
			name: "Nested variable expansion",
			input: `LEVEL1=base
LEVEL2=${LEVEL1}_level2
LEVEL3=${LEVEL2}_level3
FINAL=${LEVEL3}_final`,
			expected: map[string]string{
				"LEVEL1": "base",
				"LEVEL2": "base_level2",
				"LEVEL3": "base_level2_level3",
				"FINAL":  "base_level2_level3_final",
			},
		},
		{
			name: "Variable expansion order independence",
			input: `RESULT=${BASE}_${SUFFIX}
BASE=hello
SUFFIX=world`,
			expected: map[string]string{
				"RESULT": "hello_world",
				"BASE":   "hello",
				"SUFFIX": "world",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			actual, err := parseDotEnvStream(reader)

			if err != nil {
				t.Fatalf("parseDotEnvStream() error = %v", err)
			}

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("parseDotEnvStream() got = %v, want %v", actual, tc.expected)
			}
		})
	}
}

func TestParseDotEnvStreamCycleDetection(t *testing.T) {
	t.Run("two-step bare mutual cycle returns empty values", func(t *testing.T) {
		input := "ALPHA=$BETA\nBETA=$ALPHA"

		actual, err := parseDotEnvStream(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"ALPHA": "", "BETA": ""}, actual)
	})

	t.Run("two-step braced mutual cycle returns empty values", func(t *testing.T) {
		input := "A=${B}\nB=${A}"

		actual, err := parseDotEnvStream(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "", "B": ""}, actual)
	})

	t.Run("three-step cycle returns empty values", func(t *testing.T) {
		input := "A=${B}\nB=${C}\nC=${A}"

		actual, err := parseDotEnvStream(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "", "B": "", "C": ""}, actual)
	})

	t.Run("cycle does not affect unrelated variables", func(t *testing.T) {
		input := "A=${B}\nB=${A}\nC=clean"

		actual, err := parseDotEnvStream(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"A": "", "B": "", "C": "clean"}, actual)
	})
}

func TestParseDotEnvStreamSizeLimit(t *testing.T) {
	t.Run("oversized stream returns ErrDotenvFileTooLarge", func(t *testing.T) {
		original := MaxDotenvBytes()
		t.Cleanup(func() { SetMaxDotenvBytes(original) })

		SetMaxDotenvBytes(64)
		payload := strings.Repeat("X", 128) + "=value"

		_, err := parseDotEnvStream(strings.NewReader(payload))
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrDotenvFileTooLarge), "expected ErrDotenvFileTooLarge, got %v", err)
	})

	t.Run("stream exactly at the limit is accepted", func(t *testing.T) {
		original := MaxDotenvBytes()
		t.Cleanup(func() { SetMaxDotenvBytes(original) })

		payload := "KEY=value"
		SetMaxDotenvBytes(int64(len(payload)))

		actual, err := parseDotEnvStream(strings.NewReader(payload))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"KEY": "value"}, actual)
	})

	t.Run("zero cap disables limit", func(t *testing.T) {
		original := MaxDotenvBytes()
		t.Cleanup(func() { SetMaxDotenvBytes(original) })

		SetMaxDotenvBytes(0)
		var builder strings.Builder
		builder.WriteString("KEY=")
		builder.WriteString(strings.Repeat("y", 8192))

		actual, err := parseDotEnvStream(strings.NewReader(builder.String()))
		require.NoError(t, err)
		assert.Len(t, actual, 1)
	})

	t.Run("negative cap is treated as disabled", func(t *testing.T) {
		original := MaxDotenvBytes()
		t.Cleanup(func() { SetMaxDotenvBytes(original) })

		SetMaxDotenvBytes(-1)
		assert.Equal(t, int64(0), MaxDotenvBytes())
	})
}

func TestDotEnvLookuper_Sandbox(t *testing.T) {
	t.Run("loads env file from sandbox", func(t *testing.T) {
		ResetDotEnvCache()
		defer ResetDotEnvCache()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile(".env", []byte("APP_NAME=myapp\nAPP_PORT=3000\n"))
		SetDotEnvSandbox(sandbox)

		lookuper := dotEnvLookuper{}

		value, ok := lookuper.Lookup("APP_NAME")
		require.True(t, ok)
		assert.Equal(t, "myapp", value)

		value, ok = lookuper.Lookup("APP_PORT")
		require.True(t, ok)
		assert.Equal(t, "3000", value)
	})

	t.Run("caches after first call", func(t *testing.T) {
		ResetDotEnvCache()
		defer ResetDotEnvCache()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile(".env", []byte("CACHED_KEY=cached_value\n"))
		SetDotEnvSandbox(sandbox)

		lookuper := dotEnvLookuper{}
		_, _ = lookuper.Lookup("CACHED_KEY")
		_, _ = lookuper.Lookup("CACHED_KEY")

		assert.Equal(t, 1, sandbox.CallCounts["ReadFile"])
	})

	t.Run("handles file not found gracefully", func(t *testing.T) {
		ResetDotEnvCache()
		defer ResetDotEnvCache()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadFileErr = os.ErrNotExist
		SetDotEnvSandbox(sandbox)

		lookuper := dotEnvLookuper{}
		_, ok := lookuper.Lookup("MISSING")
		assert.False(t, ok)
	})

	t.Run("returns false on read error", func(t *testing.T) {
		ResetDotEnvCache()
		defer ResetDotEnvCache()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadFileErr = errors.New("permission denied")
		SetDotEnvSandbox(sandbox)

		lookuper := dotEnvLookuper{}
		_, ok := lookuper.Lookup("ANY_KEY")
		assert.False(t, ok)
	})
}
