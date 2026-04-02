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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestStorageError_Error(t *testing.T) {
	tests := []struct {
		name           string
		storageErr     *storageError
		expectedOutput string
	}{
		{
			name: "Standard error message",
			storageErr: &storageError{
				Err:        errors.New("connection timeout"),
				Key:        "path/to/file.txt",
				Repository: "media",
			},
			expectedOutput: "storage error for key 'path/to/file.txt' in repository 'media': connection timeout",
		},
		{
			name: "Empty key",
			storageErr: &storageError{
				Err:        errors.New("not found"),
				Key:        "",
				Repository: "events",
			},
			expectedOutput: "storage error for key '' in repository 'events': not found",
		},
		{
			name: "Different repository",
			storageErr: &storageError{
				Err:        errors.New("permission denied"),
				Key:        "secure/document.pdf",
				Repository: "published",
			},
			expectedOutput: "storage error for key 'secure/document.pdf' in repository 'published': permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.storageErr.Error()
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestMultiError_New(t *testing.T) {
	multiErr := newMultiError()
	require.NotNil(t, multiErr)
	assert.False(t, multiErr.HasErrors())
	assert.Equal(t, 0, len(multiErr.Errors))
}

func TestMultiError_Add(t *testing.T) {
	multiErr := newMultiError()

	multiErr.Add("media", "file1.txt", errors.New("error 1"))
	assert.True(t, multiErr.HasErrors())
	assert.Equal(t, 1, len(multiErr.Errors))

	multiErr.Add("events", "file2.txt", errors.New("error 2"))
	assert.Equal(t, 2, len(multiErr.Errors))

	assert.Equal(t, "media", multiErr.Errors[0].Repository)
	assert.Equal(t, "file1.txt", multiErr.Errors[0].Key)
	assert.Equal(t, "error 1", multiErr.Errors[0].Err.Error())

	assert.Equal(t, "events", multiErr.Errors[1].Repository)
	assert.Equal(t, "file2.txt", multiErr.Errors[1].Key)
	assert.Equal(t, "error 2", multiErr.Errors[1].Err.Error())
}

func TestMultiError_HasErrors(t *testing.T) {
	multiErr := newMultiError()
	assert.False(t, multiErr.HasErrors())

	multiErr.Add("media", "key", errors.New("error"))
	assert.True(t, multiErr.HasErrors())
}

func TestMultiError_Error(t *testing.T) {
	tests := []struct {
		name           string
		setup          func() *multiError
		expectedOutput string
	}{
		{
			name: "No errors",
			setup: func() *multiError {
				return newMultiError()
			},
			expectedOutput: "no storage errors",
		},
		{
			name: "Single error",
			setup: func() *multiError {
				me := newMultiError()
				me.Add("media", "file.txt", errors.New("not found"))
				return me
			},
			expectedOutput: "storage error for key 'file.txt' in repository 'media': not found",
		},
		{
			name: "Multiple errors",
			setup: func() *multiError {
				me := newMultiError()
				me.Add("media", "file1.txt", errors.New("error 1"))
				me.Add("events", "file2.txt", errors.New("error 2"))
				return me
			},
			expectedOutput: "2 storage errors occurred: storage error for key 'file1.txt' in repository 'media': error 1; storage error for key 'file2.txt' in repository 'events': error 2",
		},
		{
			name: "Three errors",
			setup: func() *multiError {
				me := newMultiError()
				me.Add("media", "a.txt", errors.New("err a"))
				me.Add("media", "b.txt", errors.New("err b"))
				me.Add("media", "c.txt", errors.New("err c"))
				return me
			},
			expectedOutput: "3 storage errors occurred: storage error for key 'a.txt' in repository 'media': err a; storage error for key 'b.txt' in repository 'media': err b; storage error for key 'c.txt' in repository 'media': err c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multiErr := tt.setup()
			result := multiErr.Error()
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}

func TestBatchResultToMultiError(t *testing.T) {
	tests := []struct {
		name           string
		batchResult    *storage_dto.BatchResult
		repo           string
		expectedNil    bool
		expectedErrors int
	}{
		{
			name:           "Nil batch result",
			batchResult:    nil,
			repo:           "media",
			expectedNil:    true,
			expectedErrors: 0,
		},
		{
			name: "Batch result with no errors",
			batchResult: &storage_dto.BatchResult{
				SuccessfulKeys: []string{"key1", "key2"},
				FailedKeys:     []storage_dto.BatchFailure{},
			},
			repo:           "media",
			expectedNil:    true,
			expectedErrors: 0,
		},
		{
			name: "Batch result with one error",
			batchResult: &storage_dto.BatchResult{
				SuccessfulKeys: []string{"key1"},
				FailedKeys: []storage_dto.BatchFailure{
					{Key: "key2", Error: "failed to upload"},
				},
				TotalFailed: 1,
			},
			repo:           "media",
			expectedNil:    false,
			expectedErrors: 1,
		},
		{
			name: "Batch result with multiple errors",
			batchResult: &storage_dto.BatchResult{
				SuccessfulKeys: []string{},
				FailedKeys: []storage_dto.BatchFailure{
					{Key: "key1", Error: "timeout"},
					{Key: "key2", Error: "not found"},
					{Key: "key3", Error: "permission denied"},
				},
				TotalFailed: 3,
			},
			repo:           "events",
			expectedNil:    false,
			expectedErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := batchResultToMultiError(tt.repo, tt.batchResult)

			if tt.expectedNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.True(t, result.HasErrors())
				assert.Equal(t, tt.expectedErrors, len(result.Errors))

				for _, err := range result.Errors {
					assert.Equal(t, tt.repo, err.Repository)
				}

				if tt.batchResult != nil {
					for i, keyErr := range tt.batchResult.FailedKeys {
						assert.Equal(t, keyErr.Key, result.Errors[i].Key)
						assert.Contains(t, result.Errors[i].Err.Error(), keyErr.Error)
					}
				}
			}
		})
	}
}
