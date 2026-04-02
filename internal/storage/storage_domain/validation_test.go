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

package storage_domain

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		errorMessage string
		expectError  bool
	}{
		{
			name:        "Valid simple key",
			key:         "file.txt",
			expectError: false,
		},
		{
			name:        "Valid nested key",
			key:         "folder/subfolder/file.txt",
			expectError: false,
		},
		{
			name:        "Valid key with underscores",
			key:         "my_file_2024.txt",
			expectError: false,
		},
		{
			name:        "Valid key with dashes",
			key:         "my-file-2024.txt",
			expectError: false,
		},
		{
			name:         "Empty key",
			key:          "",
			expectError:  true,
			errorMessage: "key cannot be empty",
		},
		{
			name:         "Path traversal with double dots",
			key:          "../etc/passwd",
			expectError:  true,
			errorMessage: "path traversal",
		},
		{
			name:         "Path traversal in middle",
			key:          "folder/../../../etc/passwd",
			expectError:  true,
			errorMessage: "path traversal",
		},
		{
			name:         "Absolute path",
			key:          "/etc/passwd",
			expectError:  true,
			errorMessage: "absolute path",
		},
		{
			name:         "Null byte injection",
			key:          "file.txt\x00.exe",
			expectError:  true,
			errorMessage: "dangerous characters",
		},
		{
			name:         "Carriage return injection",
			key:          "file\r.txt",
			expectError:  true,
			errorMessage: "dangerous characters",
		},
		{
			name:         "Newline injection",
			key:          "file\n.txt",
			expectError:  true,
			errorMessage: "dangerous characters",
		},
		{
			name:         "Key exceeds maximum length",
			key:          strings.Repeat("a", 1025),
			expectError:  true,
			errorMessage: "exceeds maximum length",
		},
		{
			name:        "Valid key at maximum length",
			key:         strings.Repeat("a", 1024),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKey(tt.key)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitiseKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Already clean key",
			input:    "folder/file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "Windows backslashes converted to forward slashes",
			input:    "folder\\subfolder\\file.txt",
			expected: "folder/subfolder/file.txt",
		},
		{
			name:     "Leading slash removed",
			input:    "/folder/file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "Multiple leading slashes become single slash",
			input:    "///folder/file.txt",
			expected: "/folder/file.txt",
		},
		{
			name:     "Current directory dots removed",
			input:    "./folder/./file.txt",
			expected: "folder/file.txt",
		},
		{
			name:     "Trailing slash removed",
			input:    "folder/file.txt/",
			expected: "folder/file.txt",
		},
		{
			name:     "Multiple slashes collapsed",
			input:    "folder//subfolder///file.txt",
			expected: "folder/subfolder/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitiseKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePutParams(t *testing.T) {
	defaultConfig := ServiceConfig{
		MaxUploadSizeBytes: 100 * 1024 * 1024,
		MaxBatchSize:       1000,
	}

	tests := []struct {
		name         string
		errorMessage string
		params       storage_dto.PutParams
		config       ServiceConfig
		expectError  bool
	}{
		{
			name: "Valid parameters",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        4,
				ContentType: "text/plain",
			},
			config:      defaultConfig,
			expectError: false,
		},
		{
			name: "Invalid key",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "../etc/passwd",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        4,
				ContentType: "text/plain",
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "path traversal",
		},
		{
			name: "Negative size",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        -1,
				ContentType: "text/plain",
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "size cannot be negative",
		},
		{
			name: "Size exceeds maximum",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        200 * 1024 * 1024,
				ContentType: "text/plain",
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "exceeds maximum allowed size",
		},
		{
			name: "Empty content type",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        4,
				ContentType: "",
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "content type cannot be empty",
		},
		{
			name: "Invalid content type",
			params: storage_dto.PutParams{
				Repository:  "media",
				Key:         "file.txt",
				Reader:      bytes.NewReader([]byte("test")),
				Size:        4,
				ContentType: "invalid",
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "invalid content type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePutParams(&tt.params, &tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCASParams(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		expectedAlg  string
		params       storage_dto.PutParams
		expectError  bool
	}{
		{
			name: "Valid CAS with sha256",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha256",
			},
			expectError: false,
			expectedAlg: "sha256",
		},
		{
			name: "Valid CAS with md5",
			params: storage_dto.PutParams{
				HashAlgorithm: "md5",
			},
			expectError: false,
			expectedAlg: "md5",
		},
		{
			name: "Default hash algorithm",
			params: storage_dto.PutParams{
				HashAlgorithm: "",
			},
			expectError: false,
			expectedAlg: "sha256",
		},
		{
			name: "Case insensitive algorithm",
			params: storage_dto.PutParams{
				HashAlgorithm: "SHA256",
			},
			expectError: false,
			expectedAlg: "sha256",
		},
		{
			name: "Unsupported hash algorithm",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha512",
			},
			expectError:  true,
			errorMessage: "unsupported hash algorithm",
		},
		{
			name: "Key provided with CAS",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha256",
				Key:           "custom-key",
			},
			expectError:  true,
			errorMessage: "key must be empty",
		},
		{
			name: "Valid expected hash SHA256",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha256",
				ExpectedHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
			expectError: false,
			expectedAlg: "sha256",
		},
		{
			name: "Invalid expected hash - not hex",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha256",
				ExpectedHash:  "not-a-hex-string",
			},
			expectError:  true,
			errorMessage: "must be a hexadecimal string",
		},
		{
			name: "Invalid expected hash - wrong length",
			params: storage_dto.PutParams{
				HashAlgorithm: "sha256",
				ExpectedHash:  "abc123",
			},
			expectError:  true,
			errorMessage: "must be 64 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.params
			err := validateCASParams(&params)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAlg, params.HashAlgorithm)
			}
		})
	}
}

func TestValidateMultipartConfig(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		config       storage_dto.MultipartUploadConfig
		expectError  bool
	}{
		{
			name: "Valid config",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    10 * 1024 * 1024,
				Concurrency: 5,
				MaxRetries:  3,
			},
			expectError: false,
		},
		{
			name: "Part size too small",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    1 * 1024 * 1024,
				Concurrency: 5,
				MaxRetries:  3,
			},
			expectError:  true,
			errorMessage: "must be at least 5 MB",
		},
		{
			name: "Part size too large",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    6 * 1024 * 1024 * 1024,
				Concurrency: 5,
				MaxRetries:  3,
			},
			expectError:  true,
			errorMessage: "cannot exceed 5 GB",
		},
		{
			name: "Concurrency too low",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    10 * 1024 * 1024,
				Concurrency: 0,
				MaxRetries:  3,
			},
			expectError:  true,
			errorMessage: "must be at least 1",
		},
		{
			name: "Concurrency too high",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    10 * 1024 * 1024,
				Concurrency: 101,
				MaxRetries:  3,
			},
			expectError:  true,
			errorMessage: "cannot exceed 100",
		},
		{
			name: "Negative max retries",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    10 * 1024 * 1024,
				Concurrency: 5,
				MaxRetries:  -1,
			},
			expectError:  true,
			errorMessage: "cannot be negative",
		},
		{
			name: "Max retries too high",
			config: storage_dto.MultipartUploadConfig{
				PartSize:    10 * 1024 * 1024,
				Concurrency: 5,
				MaxRetries:  11,
			},
			expectError:  true,
			errorMessage: "cannot exceed 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMultipartConfig(tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateByteRange(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		byteRange    storage_dto.ByteRange
		expectError  bool
	}{
		{
			name: "Valid range",
			byteRange: storage_dto.ByteRange{
				Start: 0,
				End:   100,
			},
			expectError: false,
		},
		{
			name: "Valid range to end of file",
			byteRange: storage_dto.ByteRange{
				Start: 100,
				End:   -1,
			},
			expectError: false,
		},
		{
			name: "Negative start",
			byteRange: storage_dto.ByteRange{
				Start: -1,
				End:   100,
			},
			expectError:  true,
			errorMessage: "must be non-negative",
		},
		{
			name: "End before start",
			byteRange: storage_dto.ByteRange{
				Start: 100,
				End:   50,
			},
			expectError:  true,
			errorMessage: "must be >= start",
		},
		{
			name: "Equal start and end",
			byteRange: storage_dto.ByteRange{
				Start: 100,
				End:   100,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateByteRange(tt.byteRange)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateGetParams(t *testing.T) {
	tests := []struct {
		params       storage_dto.GetParams
		name         string
		errorMessage string
		expectError  bool
	}{
		{
			name: "Valid params",
			params: storage_dto.GetParams{
				Repository: "media",
				Key:        "file.txt",
			},
			expectError: false,
		},
		{
			name: "Valid params with byte range",
			params: storage_dto.GetParams{
				Repository: "media",
				Key:        "file.txt",
				ByteRange: &storage_dto.ByteRange{
					Start: 0,
					End:   100,
				},
			},
			expectError: false,
		},
		{
			name: "Invalid key",
			params: storage_dto.GetParams{
				Repository: "media",
				Key:        "",
			},
			expectError:  true,
			errorMessage: "key cannot be empty",
		},
		{
			name: "Invalid byte range",
			params: storage_dto.GetParams{
				Repository: "media",
				Key:        "file.txt",
				ByteRange: &storage_dto.ByteRange{
					Start: 100,
					End:   50,
				},
			},
			expectError:  true,
			errorMessage: "must be >= start",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGetParams(tt.params)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCopyParams(t *testing.T) {
	tests := []struct {
		params       storage_dto.CopyParams
		name         string
		errorMessage string
		expectError  bool
	}{
		{
			name: "Valid params",
			params: storage_dto.CopyParams{
				SourceRepository:      "media",
				SourceKey:             "source.txt",
				DestinationRepository: "events",
				DestinationKey:        "dest.txt",
			},
			expectError: false,
		},
		{
			name: "Invalid source key",
			params: storage_dto.CopyParams{
				SourceRepository:      "media",
				SourceKey:             "",
				DestinationRepository: "events",
				DestinationKey:        "dest.txt",
			},
			expectError:  true,
			errorMessage: "key cannot be empty",
		},
		{
			name: "Invalid destination key",
			params: storage_dto.CopyParams{
				SourceRepository:      "media",
				SourceKey:             "source.txt",
				DestinationRepository: "events",
				DestinationKey:        "../etc/passwd",
			},
			expectError:  true,
			errorMessage: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCopyParams(tt.params)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidatePutManyParams(t *testing.T) {
	defaultConfig := ServiceConfig{
		MaxUploadSizeBytes: 100 * 1024 * 1024,
		MaxBatchSize:       1000,
	}

	tests := []struct {
		name         string
		errorMessage string
		params       storage_dto.PutManyParams
		config       ServiceConfig
		expectError  bool
	}{
		{
			name: "Valid params",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file1.txt",
						Reader:      bytes.NewReader([]byte("test1")),
						Size:        5,
						ContentType: "text/plain",
					},
					{
						Key:         "file2.txt",
						Reader:      bytes.NewReader([]byte("test2")),
						Size:        5,
						ContentType: "text/plain",
					},
				},
				Concurrency: 2,
			},
			config:      defaultConfig,
			expectError: false,
		},
		{
			name: "No objects",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects:    []storage_dto.PutObjectSpec{},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "no objects to upload",
		},
		{
			name: "Batch size exceeds maximum",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects:    make([]storage_dto.PutObjectSpec, 1001),
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "exceeds maximum",
		},
		{
			name: "Negative concurrency",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        4,
						ContentType: "text/plain",
					},
				},
				Concurrency: -1,
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "concurrency cannot be negative",
		},
		{
			name: "Concurrency too high",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        4,
						ContentType: "text/plain",
					},
				},
				Concurrency: 101,
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "concurrency cannot exceed 100",
		},
		{
			name: "Invalid key in object",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "../etc/passwd",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        4,
						ContentType: "text/plain",
					},
				},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "path traversal",
		},
		{
			name: "Negative size in object",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        -1,
						ContentType: "text/plain",
					},
				},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "size cannot be negative",
		},
		{
			name: "Size exceeds maximum in object",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        200 * 1024 * 1024,
						ContentType: "text/plain",
					},
				},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "exceeds maximum",
		},
		{
			name: "Empty content type in object",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        4,
						ContentType: "",
					},
				},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "content type cannot be empty",
		},
		{
			name: "Invalid content type in object",
			params: storage_dto.PutManyParams{
				Repository: "media",
				Objects: []storage_dto.PutObjectSpec{
					{
						Key:         "file.txt",
						Reader:      bytes.NewReader([]byte("test")),
						Size:        4,
						ContentType: "invalid",
					},
				},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "invalid content type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePutManyParams(&tt.params, &tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRemoveManyParams(t *testing.T) {
	defaultConfig := ServiceConfig{
		MaxBatchSize: 1000,
	}

	tests := []struct {
		name         string
		errorMessage string
		params       storage_dto.RemoveManyParams
		config       ServiceConfig
		expectError  bool
	}{
		{
			name: "Valid params",
			params: storage_dto.RemoveManyParams{
				Repository:  "media",
				Keys:        []string{"file1.txt", "file2.txt"},
				Concurrency: 2,
			},
			config:      defaultConfig,
			expectError: false,
		},
		{
			name: "No keys",
			params: storage_dto.RemoveManyParams{
				Repository: "media",
				Keys:       []string{},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "no keys to remove",
		},
		{
			name: "Batch size exceeds maximum",
			params: storage_dto.RemoveManyParams{
				Repository: "media",
				Keys:       make([]string, 1001),
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "exceeds maximum",
		},
		{
			name: "Negative concurrency",
			params: storage_dto.RemoveManyParams{
				Repository:  "media",
				Keys:        []string{"file.txt"},
				Concurrency: -1,
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "concurrency cannot be negative",
		},
		{
			name: "Concurrency too high",
			params: storage_dto.RemoveManyParams{
				Repository:  "media",
				Keys:        []string{"file.txt"},
				Concurrency: 101,
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "concurrency cannot exceed 100",
		},
		{
			name: "Invalid key",
			params: storage_dto.RemoveManyParams{
				Repository: "media",
				Keys:       []string{"../etc/passwd"},
			},
			config:       defaultConfig,
			expectError:  true,
			errorMessage: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRemoveManyParams(&tt.params, &tt.config)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
