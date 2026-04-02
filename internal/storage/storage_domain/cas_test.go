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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package storage_domain_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

func TestPutObject_CAS_SHA256(t *testing.T) {
	ctx := context.Background()

	service, mock := setupTestService(t)

	content := []byte("test content for CAS with SHA256")
	expectedHash := sha256.Sum256(content)
	expectedHashHex := hex.EncodeToString(expectedHash[:])

	params := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "sha256",
	}

	err := service.PutObject(ctx, "default", params)
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	require.Len(t, calls, 1)

	expectedKeyPrefix := "cas/sha256/"
	assert.True(t, strings.HasPrefix(calls[0].Key, expectedKeyPrefix),
		"key should start with %q, got %q", expectedKeyPrefix, calls[0].Key)
	assert.Contains(t, calls[0].Key, expectedHashHex,
		"key should contain the SHA-256 hash")
}

func TestPutObject_CAS_MD5(t *testing.T) {
	ctx := context.Background()

	service, mock := setupTestService(t)

	content := []byte("test content for CAS with MD5")

	expectedHash := md5.Sum(content)
	expectedHashHex := hex.EncodeToString(expectedHash[:])

	params := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "md5",
	}

	err := service.PutObject(ctx, "default", params)
	require.NoError(t, err)

	calls := mock.GetPutCalls()
	require.Len(t, calls, 1)

	expectedKeyPrefix := "cas/md5/"
	assert.True(t, strings.HasPrefix(calls[0].Key, expectedKeyPrefix),
		"key should start with %q, got %q", expectedKeyPrefix, calls[0].Key)
	assert.Contains(t, calls[0].Key, expectedHashHex,
		"key should contain the MD5 hash")
}

func TestPutObject_CAS_Deduplication(t *testing.T) {
	ctx := context.Background()

	service, mock := setupTestService(t)

	content := []byte("deduplicated content")

	params1 := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "sha256",
	}

	err := service.PutObject(ctx, "default", params1)
	require.NoError(t, err)

	firstCalls := mock.GetPutCalls()
	require.Len(t, firstCalls, 1, "first upload should result in one Put call")

	params2 := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "sha256",
	}

	err = service.PutObject(ctx, "default", params2)
	require.NoError(t, err, "deduplicated upload should succeed without error")

	secondCalls := mock.GetPutCalls()
	assert.Len(t, secondCalls, 1,
		"second upload should be deduplicated and not result in another Put call")
}

func TestPutObject_CAS_HashMismatch(t *testing.T) {
	ctx := context.Background()

	service, _ := setupTestService(t)

	content := []byte("content to hash")

	params := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "sha256",
		ExpectedHash:         "0000000000000000000000000000000000000000000000000000000000000000",
	}

	err := service.PutObject(ctx, "default", params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")
}

func TestPutObject_CAS_UnsupportedAlgorithm(t *testing.T) {
	ctx := context.Background()

	service, _ := setupTestService(t)

	content := []byte("content to hash")

	params := &storage_dto.PutParams{
		Repository:           storage_dto.StorageRepositoryDefault,
		Key:                  "",
		Reader:               bytes.NewReader(content),
		Size:                 int64(len(content)),
		ContentType:          "text/plain",
		UseContentAddressing: true,
		HashAlgorithm:        "blake2b",
	}

	err := service.PutObject(ctx, "default", params)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported hash algorithm")
}

func TestPutObject_CAS_KeyFormat(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name           string
		algorithm      string
		expectedPrefix string
		hashLength     int
	}{
		{
			name:           "SHA-256 key format",
			algorithm:      "sha256",
			expectedPrefix: "cas/sha256/",
			hashLength:     storage_domain.SHA256HexLength,
		},
		{
			name:           "MD5 key format",
			algorithm:      "md5",
			expectedPrefix: "cas/md5/",
			hashLength:     storage_domain.MD5HexLength,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, mock := setupTestService(t)

			content := []byte("key format test content")
			params := &storage_dto.PutParams{
				Repository:           storage_dto.StorageRepositoryDefault,
				Key:                  "",
				Reader:               bytes.NewReader(content),
				Size:                 int64(len(content)),
				ContentType:          "text/plain",
				UseContentAddressing: true,
				HashAlgorithm:        tc.algorithm,
			}

			err := service.PutObject(ctx, "default", params)
			require.NoError(t, err)

			calls := mock.GetPutCalls()
			require.Len(t, calls, 1)

			key := calls[0].Key
			assert.True(t, strings.HasPrefix(key, tc.expectedPrefix),
				"key should start with %q, got %q", tc.expectedPrefix, key)

			hashPart := strings.TrimPrefix(key, tc.expectedPrefix)
			assert.Len(t, hashPart, tc.hashLength,
				"hash part should be %d hex characters", tc.hashLength)
		})
	}
}
