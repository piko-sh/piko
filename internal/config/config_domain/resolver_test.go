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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvResolverGetPrefix(t *testing.T) {
	r := &EnvResolver{}
	assert.Equal(t, "env:", r.GetPrefix())
}

func TestEnvResolverResolve(t *testing.T) {
	testCases := []struct {
		setup      func(t *testing.T)
		name       string
		input      string
		expected   string
		errMessage string
		wantErr    bool
	}{
		{
			name: "resolves existing environment variable",
			setup: func(t *testing.T) {
				t.Setenv("TEST_ENV_VAR", "test_value")
			},
			input:    "TEST_ENV_VAR",
			expected: "test_value",
		},
		{
			name: "resolves empty environment variable",
			setup: func(t *testing.T) {
				t.Setenv("EMPTY_VAR", "")
			},
			input:    "EMPTY_VAR",
			expected: "",
		},
		{
			name:       "fails for non-existent environment variable",
			input:      "NONEXISTENT_VAR_12345",
			wantErr:    true,
			errMessage: "environment variable \"NONEXISTENT_VAR_12345\" not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t)
			}

			r := &EnvResolver{}
			result, err := r.Resolve(context.Background(), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestBase64ResolverGetPrefix(t *testing.T) {
	r := &Base64Resolver{}
	assert.Equal(t, "base64:", r.GetPrefix())
}

func TestBase64ResolverResolve(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		expected   string
		errMessage string
		wantErr    bool
	}{
		{
			name:     "decodes valid base64 string",
			input:    base64.StdEncoding.EncodeToString([]byte("Hello World")),
			expected: "Hello World",
		},
		{
			name:     "decodes empty base64 string",
			input:    base64.StdEncoding.EncodeToString([]byte("")),
			expected: "",
		},
		{
			name:     "decodes base64 with special characters",
			input:    base64.StdEncoding.EncodeToString([]byte("secret!@#$%^&*()")),
			expected: "secret!@#$%^&*()",
		},
		{
			name:     "decodes base64 with unicode",
			input:    base64.StdEncoding.EncodeToString([]byte("Hello 世界")),
			expected: "Hello 世界",
		},
		{
			name:       "fails for invalid base64 string",
			input:      "not-valid-base64!!!",
			wantErr:    true,
			errMessage: "invalid base64 string",
		},
		{
			name:       "fails for truncated base64 string",
			input:      "SGVsbG8",
			wantErr:    true,
			errMessage: "invalid base64 string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Base64Resolver{}
			result, err := r.Resolve(context.Background(), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestFileResolverGetPrefix(t *testing.T) {
	r := NewFileResolver(nil)
	assert.Equal(t, "file:", r.GetPrefix())
}

func TestFileResolverResolve(t *testing.T) {
	testCases := []struct {
		name        string
		fileContent string
		errMessage  string
		wantErr     bool
	}{
		{
			name:        "reads file content",
			fileContent: "secret password",
		},
		{
			name:        "reads file with whitespace",
			fileContent: "  trimmed value  \n",
		},
		{
			name:        "reads empty file",
			fileContent: "",
		},
		{
			name:        "reads file with newlines",
			fileContent: "line1\nline2\nline3",
		},
		{
			name:        "reads file with unicode",
			fileContent: "Hello 世界 🌍",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			directory := t.TempDir()
			filePath := filepath.Join(directory, "test.txt")
			err := os.WriteFile(filePath, []byte(tc.fileContent), 0644)
			require.NoError(t, err)

			r := NewFileResolver(nil)
			result, err := r.Resolve(context.Background(), filePath)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMessage)
			} else {
				require.NoError(t, err)

				expected := tc.fileContent
				if tc.fileContent == "  trimmed value  \n" {
					expected = "trimmed value"
				}
				assert.Equal(t, expected, result)
			}
		})
	}
}

func TestFileResolverResolveNonExistent(t *testing.T) {
	r := NewFileResolver(nil)
	_, err := r.Resolve(context.Background(), "/nonexistent/path/to/file.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestFileResolverResolveBatch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		files   map[string]string
		values  []string
		wantErr bool
	}{
		{
			name: "resolves multiple files",
			files: map[string]string{
				"file1.txt": "content1",
				"file2.txt": "content2",
				"file3.txt": "content3",
			},
			values: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name: "deduplicates file reads",
			files: map[string]string{
				"file1.txt": "content1",
			},

			values: []string{"file1.txt", "file1.txt", "file1.txt"},
		},
		{
			name:   "handles empty values slice",
			files:  map[string]string{},
			values: []string{},
		},
		{
			name: "fails when any file not found",
			files: map[string]string{
				"file1.txt": "content1",
			},
			values:  []string{"file1.txt", "nonexistent.txt"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()

			var fullPaths []string
			for filename, content := range tc.files {
				filePath := filepath.Join(directory, filename)
				err := os.WriteFile(filePath, []byte(content), 0644)
				require.NoError(t, err)
			}

			for _, v := range tc.values {
				fullPaths = append(fullPaths, filepath.Join(directory, v))
			}

			r := NewFileResolver(nil)
			results, err := r.ResolveBatch(context.Background(), fullPaths)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				uniquePaths := make(map[string]struct{})
				for _, path := range fullPaths {
					uniquePaths[path] = struct{}{}
				}
				assert.Len(t, results, len(uniquePaths))

				for path := range uniquePaths {
					filename := filepath.Base(path)
					if content, ok := tc.files[filename]; ok {
						assert.Equal(t, content, results[path])
					}
				}
			}
		})
	}
}

func TestFileResolverBatchConcurrency(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	paths := make([]string, 0, 50)
	for i := range 50 {
		filePath := filepath.Join(directory, "file"+string(rune('a'+i%26))+string(rune('0'+i/26))+".txt")
		content := "content" + string(rune('0'+i))
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
		paths = append(paths, filePath)
	}

	r := NewFileResolver(nil)
	results, err := r.ResolveBatch(context.Background(), paths)

	require.NoError(t, err)
	assert.Len(t, results, len(paths))
}

func TestResolverInterfaceCompliance(t *testing.T) {
	testCases := []struct {
		name     string
		resolver Resolver
		prefix   string
	}{
		{
			name:     "EnvResolver implements Resolver",
			resolver: &EnvResolver{},
			prefix:   "env:",
		},
		{
			name:     "Base64Resolver implements Resolver",
			resolver: &Base64Resolver{},
			prefix:   "base64:",
		},
		{
			name:     "FileResolver implements Resolver",
			resolver: NewFileResolver(nil),
			prefix:   "file:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.prefix, tc.resolver.GetPrefix())
		})
	}
}

func TestBatchResolverInterfaceCompliance(t *testing.T) {
	r := NewFileResolver(nil)
	var _ BatchResolver = r

	_, ok := any(r).(BatchResolver)
	assert.True(t, ok, "FileResolver should implement BatchResolver")
}

func Test_clearResolutionPools(t *testing.T) {

	assert.NotPanics(t, func() {
		clearResolutionPools()
	})

	for range 100 {
		clearResolutionPools()
	}
}

func TestResolutionJobReset(t *testing.T) {
	job := &resolutionJob{
		keyPath:   "test.path",
		prefix:    "env:",
		lookupKey: "VAR",
	}

	job.Reset()

	assert.Empty(t, job.keyPath)
	assert.Empty(t, job.prefix)
	assert.Empty(t, job.lookupKey)
	assert.Nil(t, job.field)
}

func TestResolutionResultReset(t *testing.T) {
	job := &resolutionJob{keyPath: "test"}
	result := &resolutionResult{
		job:           job,
		resolvedValue: "value",
		err:           assert.AnError,
	}

	result.Reset()

	assert.Nil(t, result.job)
	assert.Empty(t, result.resolvedValue)
	assert.Nil(t, result.err)
}
