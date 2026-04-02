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

package provider_mock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"sync"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/internal/storage/storage_domain"
	"piko.sh/piko/internal/storage/storage_dto"
)

// ErrObjectNotFound is a sentinel error returned by the mock provider when an
// object does not exist. Tests should use `errors.Is(err,
// provider_mock.ErrObjectNotFound)` to check for this condition.
var ErrObjectNotFound = errors.New("object not found")

// mockObject represents a file stored in the mock storage provider's memory.
type mockObject struct {
	// lastModified is when the object was last changed.
	lastModified time.Time

	// contentType is the MIME type of the stored object.
	contentType string

	// metadata holds custom key-value pairs linked to the object.
	metadata map[string]string

	// data holds the raw bytes of the stored object.
	data []byte
}

// MockStorageProvider is a thread-safe, in-memory implementation of
// StorageProviderPort for testing. It supports call inspection, state
// verification, and error simulation.
type MockStorageProvider struct {
	// copyError is returned by copyInternal when set; nil means no error.
	copyError error

	// removeManyError is the error to return from RemoveMany; nil returns success.
	removeManyError error

	// putManyError is returned by PutMany when set; nil allows normal operation.
	putManyError error

	// presignError is returned by PresignURL and PresignDownloadURL when set;
	// nil means no error.
	presignError error

	// getHashError is the error returned by GetHash when set; nil means no error.
	getHashError error

	// removeError is returned by Remove when set; nil allows normal behaviour.
	removeError error

	// getError is the error to return from Get; nil means no error.
	getError error

	// statError is returned by Stat when set; nil means Stat succeeds.
	statError error

	// errToReturn is a general error returned by any method when set.
	errToReturn error

	// putError is the error returned by Put when set; nil allows normal behaviour.
	putError error

	// storage maps repository names to their objects, keyed by object key.
	storage map[string]map[string]*mockObject

	// hashToReturn is the hash value to return from GetHash; empty uses
	// default behaviour.
	hashToReturn string

	// presignedURLToReturn is the URL to return from PresignURL and
	// PresignDownloadURL; empty uses default behaviour.
	presignedURLToReturn string

	// copyCalls records each copy operation for test verification.
	copyCalls []storage_dto.CopyParams

	// removeManyCalls records parameters from each RemoveMany call for test
	// verification.
	removeManyCalls []storage_dto.RemoveManyParams

	// putManyCalls stores the arguments from each PutMany call for later checking.
	putManyCalls []storage_dto.PutManyParams

	// presignURLCalls records the parameters from each PresignURL call.
	presignURLCalls []storage_dto.PresignParams

	// getHashCalls records the parameters passed to each GetHash call.
	getHashCalls []storage_dto.GetParams

	// removeCalls records the parameters of each Remove call for test
	// verification.
	removeCalls []storage_dto.GetParams

	// statCalls stores the parameters from each Stat method call.
	statCalls []storage_dto.GetParams

	// getCalls stores the parameters from each Get call for test checks.
	getCalls []storage_dto.GetParams

	// putCalls stores the parameters from each Put call for test checks.
	putCalls []storage_dto.PutParams

	// mu guards concurrent access to the mock's internal state.
	mu sync.RWMutex
}

var _ storage_domain.StorageProviderPort = (*MockStorageProvider)(nil)
var _ provider_domain.ProviderMetadata = (*MockStorageProvider)(nil)

// NewMockStorageProvider creates a new mock storage provider for use in tests.
//
// Returns *MockStorageProvider which is ready for use with all error fields
// set to nil.
func NewMockStorageProvider() *MockStorageProvider {
	return &MockStorageProvider{
		copyError:            nil,
		removeManyError:      nil,
		putManyError:         nil,
		presignError:         nil,
		getHashError:         nil,
		removeError:          nil,
		getError:             nil,
		statError:            nil,
		errToReturn:          nil,
		putError:             nil,
		storage:              make(map[string]map[string]*mockObject),
		hashToReturn:         "",
		presignedURLToReturn: "",
		copyCalls:            nil,
		removeManyCalls:      nil,
		putManyCalls:         nil,
		presignURLCalls:      nil,
		getHashCalls:         nil,
		removeCalls:          nil,
		statCalls:            nil,
		getCalls:             nil,
		putCalls:             nil,
		mu:                   sync.RWMutex{},
	}
}

// GetProviderType returns the type identifier for this storage provider.
//
// Returns string which is "mock".
func (*MockStorageProvider) GetProviderType() string {
	return "mock"
}

// GetProviderMetadata returns metadata about this mock storage provider.
//
// Returns map[string]any which describes the provider configuration.
func (*MockStorageProvider) GetProviderMetadata() map[string]any {
	return map[string]any{
		"description": "In-memory mock storage provider for testing",
	}
}

// Put simulates an object upload, storing the data in memory and recording the
// call.
//
// Takes params (*storage_dto.PutParams) which specifies the repository, key,
// reader, and metadata for the object.
//
// Returns error when reading from the reader fails or a configured error is
// set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) Put(_ context.Context, params *storage_dto.PutParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := io.ReadAll(params.Reader)
	if err != nil {
		return fmt.Errorf("mock provider failed to read from reader: %w", err)
	}

	paramsCopy := *params
	paramsCopy.Reader = nil
	m.putCalls = append(m.putCalls, paramsCopy)

	if m.putError != nil {
		return m.putError
	}
	if m.errToReturn != nil {
		return m.errToReturn
	}

	if _, ok := m.storage[params.Repository]; !ok {
		m.storage[params.Repository] = make(map[string]*mockObject)
	}

	var metadata map[string]string
	if params.Metadata != nil {
		metadata = make(map[string]string, len(params.Metadata))
		maps.Copy(metadata, params.Metadata)
	}

	m.storage[params.Repository][params.Key] = &mockObject{
		data:         data,
		contentType:  params.ContentType,
		metadata:     metadata,
		lastModified: time.Now(),
	}
	return nil
}

// Get simulates retrieving an object, returning its data from memory.
//
// Takes params (storage_dto.GetParams) which specifies the repository, key,
// and optional byte range.
//
// Returns io.ReadCloser which provides the object data as a stream.
// Returns error when the object is not found or a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) Get(_ context.Context, params storage_dto.GetParams) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getCalls = append(m.getCalls, params)

	if m.getError != nil {
		return nil, m.getError
	}
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}

	repo, ok := m.storage[params.Repository]
	if !ok {
		return nil, fmt.Errorf("repository %s not found: %w", params.Repository, ErrObjectNotFound)
	}
	storageObject, ok := repo[params.Key]
	if !ok {
		return nil, fmt.Errorf("object '%s' not found: %w", params.Key, ErrObjectNotFound)
	}

	data := storageObject.data
	if params.ByteRange != nil {
		start, end := params.ByteRange.Start, params.ByteRange.End
		if start < 0 || start > int64(len(data)) {
			start = int64(len(data))
		}
		if end == -1 || end >= int64(len(data)) {
			end = int64(len(data)) - 1
		}
		if end < start {
			data = []byte{}
		} else {
			data = data[start : end+1]
		}
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

// Stat simulates retrieving object metadata.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key to query.
//
// Returns *storage_domain.ObjectInfo which contains the object metadata.
// Returns error when the object is not found or a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) Stat(_ context.Context, params storage_dto.GetParams) (*storage_domain.ObjectInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.statCalls = append(m.statCalls, params)

	if m.statError != nil {
		return nil, m.statError
	}
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}

	repo, ok := m.storage[params.Repository]
	if !ok {
		return nil, fmt.Errorf("repository %s not found: %w", params.Repository, ErrObjectNotFound)
	}
	storageObject, ok := repo[params.Key]
	if !ok {
		return nil, fmt.Errorf("object '%s' not found: %w", params.Key, ErrObjectNotFound)
	}

	return &storage_domain.ObjectInfo{
		Size:         int64(len(storageObject.data)),
		ContentType:  storageObject.contentType,
		LastModified: storageObject.lastModified,
		ETag:         "",
		Metadata:     storageObject.metadata,
	}, nil
}

// Copy simulates an intra-repository copy.
//
// Takes repo (string) which identifies the repository containing the objects.
// Takes srcKey (string) which specifies the source object key.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the source object is not found or a configured error
// is set.
func (m *MockStorageProvider) Copy(_ context.Context, repo string, srcKey, dstKey string) error {
	return m.copyInternal(storage_dto.CopyParams{
		SourceRepository:      repo,
		SourceKey:             srcKey,
		DestinationRepository: repo,
		DestinationKey:        dstKey,
	})
}

// CopyToAnotherRepository simulates an inter-repository copy.
//
// Takes srcRepo (string) which identifies the source repository.
// Takes srcKey (string) which specifies the source object key.
// Takes dstRepo (string) which identifies the destination repository.
// Takes dstKey (string) which specifies the destination object key.
//
// Returns error when the source object is not found or a configured error
// is set.
func (m *MockStorageProvider) CopyToAnotherRepository(_ context.Context, srcRepo string, srcKey string, dstRepo string, dstKey string) error {
	return m.copyInternal(storage_dto.CopyParams{
		SourceRepository:      srcRepo,
		SourceKey:             srcKey,
		DestinationRepository: dstRepo,
		DestinationKey:        dstKey,
	})
}

// Remove simulates deleting an object from memory. It is idempotent.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key to delete.
//
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) Remove(_ context.Context, params storage_dto.GetParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeCalls = append(m.removeCalls, params)

	if m.removeError != nil {
		return m.removeError
	}
	if m.errToReturn != nil {
		return m.errToReturn
	}

	if repo, ok := m.storage[params.Repository]; ok {
		delete(repo, params.Key)
	}
	return nil
}

// Rename simulates moving an object from one key to another within the same
// repository.
//
// Takes repo (string) which identifies the repository containing the object.
// Takes oldKey (string) which specifies the current object key.
// Takes newKey (string) which specifies the target object key.
//
// Returns error when the copy phase fails.
func (m *MockStorageProvider) Rename(ctx context.Context, repo string, oldKey, newKey string) error {
	if err := m.Copy(ctx, repo, oldKey, newKey); err != nil {
		return fmt.Errorf("rename copy phase failed: %w", err)
	}

	params := storage_dto.GetParams{
		Repository:      repo,
		Key:             oldKey,
		ByteRange:       nil,
		TransformConfig: nil,
	}
	_ = m.Remove(ctx, params)

	return nil
}

// Exists checks if an object exists in the mock storage.
// Returns (true, nil) if exists, (false, nil) if not exists, (false, error) on
// failure.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key to check.
//
// Returns bool which is true if the object exists, false otherwise.
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// read lock.
func (m *MockStorageProvider) Exists(_ context.Context, params storage_dto.GetParams) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.errToReturn != nil {
		return false, m.errToReturn
	}

	repo, ok := m.storage[params.Repository]
	if !ok {
		return false, nil
	}
	_, ok = repo[params.Key]
	return ok, nil
}

// GetHash simulates calculating an object's MD5 hash.
//
// Takes params (storage_dto.GetParams) which specifies the repository and
// key to hash.
//
// Returns string which is the hex-encoded hash of the object data, or a
// configured hash value.
// Returns error when the object is not found or a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) GetHash(_ context.Context, params storage_dto.GetParams) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getHashCalls = append(m.getHashCalls, params)

	if m.getHashError != nil {
		return "", m.getHashError
	}
	if m.errToReturn != nil {
		return "", m.errToReturn
	}
	if m.hashToReturn != "" {
		return m.hashToReturn, nil
	}

	repo, ok := m.storage[params.Repository]
	if !ok {
		return "", fmt.Errorf("repository %s not found: %w", params.Repository, ErrObjectNotFound)
	}
	storageObject, ok := repo[params.Key]
	if !ok {
		return "", fmt.Errorf("object '%s' not found: %w", params.Key, ErrObjectNotFound)
	}
	hash := sha256.Sum256(storageObject.data)
	return hex.EncodeToString(hash[:]), nil
}

// PresignURL simulates generating a presigned URL.
//
// Takes params (storage_dto.PresignParams) which specifies the repository,
// key, and expiry settings.
//
// Returns string which is the generated presigned URL, or a configured URL.
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) PresignURL(_ context.Context, params storage_dto.PresignParams) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.presignURLCalls = append(m.presignURLCalls, params)

	if m.presignError != nil {
		return "", m.presignError
	}
	if m.errToReturn != nil {
		return "", m.errToReturn
	}
	if m.presignedURLToReturn != "" {
		return m.presignedURLToReturn, nil
	}

	return fmt.Sprintf("https://mock.storage.local/%s/%s?presigned=true", params.Repository, params.Key), nil
}

// PresignDownloadURL simulates generating a presigned download URL.
//
// Takes params (storage_dto.PresignDownloadParams) which specifies the
// repository, key, and download settings.
//
// Returns string which is the generated presigned download URL, or a
// configured URL.
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) PresignDownloadURL(_ context.Context, params storage_dto.PresignDownloadParams) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.presignError != nil {
		return "", m.presignError
	}
	if m.errToReturn != nil {
		return "", m.errToReturn
	}
	if m.presignedURLToReturn != "" {
		return m.presignedURLToReturn, nil
	}

	return fmt.Sprintf("https://mock.storage.local/%s/%s?presigned=true&download=true", params.Repository, params.Key), nil
}

// Name returns the display name of this provider.
//
// Returns string which is the human-readable name for this provider.
func (*MockStorageProvider) Name() string {
	return "StorageProvider (Mock)"
}

// Check implements the healthprobe_domain.Probe interface.
// The mock storage provider is always healthy as it operates in-memory.
//
// Returns healthprobe_dto.Status which always reports healthy.
func (m *MockStorageProvider) Check(_ context.Context, _ healthprobe_dto.CheckType) healthprobe_dto.Status {
	startTime := time.Now()

	return healthprobe_dto.Status{
		Name:      m.Name(),
		State:     healthprobe_dto.StateHealthy,
		Message:   "Mock storage provider operational",
		Timestamp: time.Now(),
		Duration:  time.Since(startTime).String(),
	}
}

// Close does nothing and always succeeds.
//
// Returns error which is always nil.
func (*MockStorageProvider) Close(_ context.Context) error {
	return nil
}

// SupportsMultipart returns whether the provider supports multipart uploads.
//
// Returns bool which is always true, as the mock can be used to test
// multipart logic.
func (*MockStorageProvider) SupportsMultipart() bool {
	return true
}

// SupportsBatchOperations returns whether this provider supports batch
// operations.
//
// Returns bool which is always true, as the mock provides batch
// implementations.
func (*MockStorageProvider) SupportsBatchOperations() bool {
	return true
}

// SupportsRetry returns false; the service layer should handle retries for
// testing purposes.
//
// Returns bool which indicates whether the provider supports automatic retries.
func (*MockStorageProvider) SupportsRetry() bool {
	return false
}

// SupportsCircuitBreaking returns false; the service layer handles circuit
// breaking for testing.
//
// Returns bool which indicates whether circuit breaking is supported.
func (*MockStorageProvider) SupportsCircuitBreaking() bool {
	return false
}

// SupportsRateLimiting reports whether the provider supports rate limiting.
//
// Returns bool which is always false; the service layer handles rate limiting
// for testing.
func (*MockStorageProvider) SupportsRateLimiting() bool {
	return false
}

// SupportsPresignedURLs returns true since the mock provider can simulate
// native presigned URL generation for testing.
//
// Returns bool which is always true for this provider.
func (*MockStorageProvider) SupportsPresignedURLs() bool {
	return true
}

// PutMany simulates batch upload operations with full call tracking.
//
// Takes params (*storage_dto.PutManyParams) which specifies the repository
// and objects to upload.
//
// Returns *storage_dto.BatchResult which contains counts and details of
// successful and failed uploads.
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) PutMany(_ context.Context, params *storage_dto.PutManyParams) (*storage_dto.BatchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	paramsCopy := *params
	paramsCopy.Objects = make([]storage_dto.PutObjectSpec, len(params.Objects))
	for i, storageObject := range params.Objects {
		paramsCopy.Objects[i] = storage_dto.PutObjectSpec{Key: storageObject.Key, ContentType: storageObject.ContentType, Size: storageObject.Size, Reader: nil}
	}
	m.putManyCalls = append(m.putManyCalls, paramsCopy)

	if m.putManyError != nil {
		return nil, m.putManyError
	}
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}

	startTime := time.Now()
	result := &storage_dto.BatchResult{
		TotalRequested:  len(params.Objects),
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}
	for _, storageObject := range params.Objects {
		data, err := io.ReadAll(storageObject.Reader)
		if err != nil {
			result.FailedKeys = append(result.FailedKeys, storage_dto.BatchFailure{
				Key:       storageObject.Key,
				Error:     err.Error(),
				ErrorCode: "",
				Retryable: false,
			})
			result.TotalFailed++
			continue
		}
		if _, ok := m.storage[params.Repository]; !ok {
			m.storage[params.Repository] = make(map[string]*mockObject)
		}
		m.storage[params.Repository][storageObject.Key] = &mockObject{
			lastModified: time.Now(),
			contentType:  storageObject.ContentType,
			metadata:     nil,
			data:         data,
		}
		result.SuccessfulKeys = append(result.SuccessfulKeys, storageObject.Key)
		result.TotalSuccessful++
	}
	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// RemoveMany simulates batch delete operations. It is idempotent.
//
// Takes params (storage_dto.RemoveManyParams) which specifies the repository
// and keys to delete.
//
// Returns *storage_dto.BatchResult which contains counts and details of
// successful deletions.
// Returns error when a configured error is set.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (m *MockStorageProvider) RemoveMany(_ context.Context, params storage_dto.RemoveManyParams) (*storage_dto.BatchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.removeManyCalls = append(m.removeManyCalls, params)

	if m.removeManyError != nil {
		return nil, m.removeManyError
	}
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}

	startTime := time.Now()
	result := &storage_dto.BatchResult{
		TotalRequested:  len(params.Keys),
		SuccessfulKeys:  nil,
		FailedKeys:      nil,
		TotalSuccessful: 0,
		TotalFailed:     0,
		ProcessingTime:  0,
	}
	for _, key := range params.Keys {
		if repo, ok := m.storage[params.Repository]; ok {
			delete(repo, key)
		}
		result.SuccessfulKeys = append(result.SuccessfulKeys, key)
		result.TotalSuccessful++
	}
	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// SetError sets a generic error to be returned by any method that supports it.
//
// Takes err (error) which is the error to return from subsequent method calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errToReturn = err
}

// SetPutError sets an error to be returned specifically by Put.
//
// Takes err (error) which is the error that Put will return on subsequent
// calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetPutError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.putError = err
}

// SetGetError sets an error to be returned specifically by Get.
//
// Takes err (error) which is the error to return from subsequent Get calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getError = err
}

// SetStatError sets an error to be returned specifically by Stat.
//
// Takes err (error) which is the error that Stat will return.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetStatError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.statError = err
}

// SetRemoveError sets an error to be returned specifically by Remove.
//
// Takes err (error) which is the error to return from Remove calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetRemoveError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeError = err
}

// SetCopyError sets an error to be returned by Copy and
// CopyToAnotherRepository.
//
// Takes err (error) which is the error to return from copy operations.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetCopyError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.copyError = err
}

// SetGetHashError sets an error to be returned specifically by GetHash.
//
// Takes err (error) which specifies the error to return from GetHash calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetGetHashError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getHashError = err
}

// SetPresignError sets an error to be returned specifically by PresignURL.
//
// Takes err (error) which specifies the error that PresignURL will return.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetPresignError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presignError = err
}

// SetPresignedURL sets a specific URL to be returned by a successful
// PresignURL call.
//
// Takes url (string) which is the URL to return from future PresignURL calls.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetPresignedURL(url string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presignedURLToReturn = url
}

// SetHash sets a specific hash to be returned by a successful GetHash call.
//
// Takes hash (string) which specifies the hash value to return.
//
// Safe for concurrent use.
func (m *MockStorageProvider) SetHash(hash string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hashToReturn = hash
}

// GetObjectData is a test helper to directly access the stored data for an
// object.
//
// Takes repo (string) which identifies the repository to search in.
// Takes key (string) which identifies the object within the repository.
//
// Returns []byte which contains a copy of the stored data, or nil if not found.
// Returns bool which is true if the object was found, false otherwise.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetObjectData(repo string, key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if r, ok := m.storage[repo]; ok {
		if storageObject, ok := r[key]; ok {
			dataCopy := make([]byte, len(storageObject.data))
			copy(dataCopy, storageObject.data)
			return dataCopy, true
		}
	}
	return nil, false
}

// GetPutCalls returns a copy of all parameters passed to Put.
//
// Returns []storage_dto.PutParams which contains a copy of all recorded Put
// call parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetPutCalls() []storage_dto.PutParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.PutParams, len(m.putCalls))
	copy(callsCopy, m.putCalls)
	return callsCopy
}

// GetLastPutCall returns the most recent parameters passed to Put.
//
// Returns storage_dto.PutParams which contains the parameters from the last
// Put call, or an empty value if no calls have been made.
// Returns bool which is true if a Put call was recorded, false otherwise.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastPutCall() (storage_dto.PutParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.putCalls) == 0 {
		return storage_dto.PutParams{
			Repository:           "",
			Key:                  "",
			Reader:               nil,
			Size:                 0,
			ContentType:          "",
			TransformConfig:      nil,
			MultipartConfig:      nil,
			Metadata:             nil,
			HashAlgorithm:        "",
			ExpectedHash:         "",
			UseContentAddressing: false,
		}, false
	}
	return m.putCalls[len(m.putCalls)-1], true
}

// GetGetCalls returns a copy of all parameters passed to Get.
//
// Returns []storage_dto.GetParams which contains a copy of all call records.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetGetCalls() []storage_dto.GetParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.GetParams, len(m.getCalls))
	copy(callsCopy, m.getCalls)
	return callsCopy
}

// GetLastGetCall returns the most recent parameters passed to Get.
//
// Returns storage_dto.GetParams which contains the parameters from the last
// call.
// Returns bool which indicates whether a call was recorded.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastGetCall() (storage_dto.GetParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.getCalls) == 0 {
		return storage_dto.GetParams{
			Repository:      "",
			Key:             "",
			ByteRange:       nil,
			TransformConfig: nil,
		}, false
	}
	return m.getCalls[len(m.getCalls)-1], true
}

// GetStatCalls returns a copy of all parameters passed to Stat.
//
// Returns []storage_dto.GetParams which contains a copy of the recorded call
// parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetStatCalls() []storage_dto.GetParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.GetParams, len(m.statCalls))
	copy(callsCopy, m.statCalls)
	return callsCopy
}

// GetLastStatCall returns the most recent parameters passed to Stat.
//
// Returns storage_dto.GetParams which contains the parameters from the last
// Stat call.
// Returns bool which indicates whether any Stat call has been recorded.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastStatCall() (storage_dto.GetParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.statCalls) == 0 {
		return storage_dto.GetParams{
			Repository:      "",
			Key:             "",
			ByteRange:       nil,
			TransformConfig: nil,
		}, false
	}
	return m.statCalls[len(m.statCalls)-1], true
}

// GetRemoveCalls returns a copy of all parameters passed to Remove.
//
// Returns []storage_dto.GetParams which contains the recorded call parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetRemoveCalls() []storage_dto.GetParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.GetParams, len(m.removeCalls))
	copy(callsCopy, m.removeCalls)
	return callsCopy
}

// GetLastRemoveCall returns the most recent parameters passed to Remove.
//
// Returns storage_dto.GetParams which contains the parameters from the last
// Remove call.
// Returns bool which indicates whether any Remove call has been recorded.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastRemoveCall() (storage_dto.GetParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.removeCalls) == 0 {
		return storage_dto.GetParams{
			Repository:      "",
			Key:             "",
			ByteRange:       nil,
			TransformConfig: nil,
		}, false
	}
	return m.removeCalls[len(m.removeCalls)-1], true
}

// GetCopyCalls returns a copy of all parameters passed to Copy or
// CopyToAnotherRepository.
//
// Returns []storage_dto.CopyParams which contains a copy of all recorded call
// parameters.
//
// Safe for concurrent use. Acquires a read lock to protect access to the
// internal call log.
func (m *MockStorageProvider) GetCopyCalls() []storage_dto.CopyParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.CopyParams, len(m.copyCalls))
	copy(callsCopy, m.copyCalls)
	return callsCopy
}

// GetLastCopyCall returns the most recent parameters passed to Copy or
// CopyToAnotherRepository.
//
// Returns storage_dto.CopyParams which contains the copy operation parameters.
// Returns bool which indicates whether a copy call was recorded.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastCopyCall() (storage_dto.CopyParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.copyCalls) == 0 {
		return storage_dto.CopyParams{
			SourceRepository:      "",
			SourceKey:             "",
			DestinationRepository: "",
			DestinationKey:        "",
		}, false
	}
	return m.copyCalls[len(m.copyCalls)-1], true
}

// GetGetHashCalls returns a copy of all parameters passed to GetHash.
//
// Returns []storage_dto.GetParams which contains a copy of the recorded call
// parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetGetHashCalls() []storage_dto.GetParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.GetParams, len(m.getHashCalls))
	copy(callsCopy, m.getHashCalls)
	return callsCopy
}

// GetLastGetHashCall returns the most recent parameters passed to GetHash.
//
// Returns storage_dto.GetParams which contains the parameters from the last
// GetHash call, or a zero value if no calls have been made.
// Returns bool which indicates whether a call was recorded.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastGetHashCall() (storage_dto.GetParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.getHashCalls) == 0 {
		return storage_dto.GetParams{
			Repository:      "",
			Key:             "",
			ByteRange:       nil,
			TransformConfig: nil,
		}, false
	}
	return m.getHashCalls[len(m.getHashCalls)-1], true
}

// GetPresignURLCalls returns a copy of all parameters passed to PresignURL.
//
// Returns []storage_dto.PresignParams which holds a copy of the recorded call
// parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetPresignURLCalls() []storage_dto.PresignParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.PresignParams, len(m.presignURLCalls))
	copy(callsCopy, m.presignURLCalls)
	return callsCopy
}

// GetLastPresignURLCall returns the most recent parameters passed to
// PresignURL.
//
// Returns storage_dto.PresignParams which contains the parameters from the
// last call.
// Returns bool which is true if a call was recorded, false otherwise.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetLastPresignURLCall() (storage_dto.PresignParams, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.presignURLCalls) == 0 {
		return storage_dto.PresignParams{
			Repository:  "",
			Key:         "",
			ContentType: "",
			ExpiresIn:   0,
		}, false
	}
	return m.presignURLCalls[len(m.presignURLCalls)-1], true
}

// GetPutManyCalls returns a copy of all parameters passed to PutMany.
//
// Returns []storage_dto.PutManyParams which contains copies of the arguments
// from each PutMany call.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetPutManyCalls() []storage_dto.PutManyParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.PutManyParams, len(m.putManyCalls))
	copy(callsCopy, m.putManyCalls)
	return callsCopy
}

// GetRemoveManyCalls returns a copy of all parameters passed to RemoveMany.
//
// Returns []storage_dto.RemoveManyParams which contains the recorded call
// parameters.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetRemoveManyCalls() []storage_dto.RemoveManyParams {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]storage_dto.RemoveManyParams, len(m.removeManyCalls))
	copy(callsCopy, m.removeManyCalls)
	return callsCopy
}

// GetTotalCallCount returns the total number of calls made to any of the mock
// provider's main methods.
//
// Returns int which is the sum of all method call counts.
//
// Safe for concurrent use.
func (m *MockStorageProvider) GetTotalCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.putCalls) + len(m.getCalls) + len(m.statCalls) + len(m.copyCalls) +
		len(m.removeCalls) + len(m.getHashCalls) + len(m.presignURLCalls) +
		len(m.putManyCalls) + len(m.removeManyCalls)
}

// Reset clears all recorded calls, stored data, and configured errors,
// preparing the mock for a new test case.
//
// Safe for concurrent use.
func (m *MockStorageProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage = make(map[string]map[string]*mockObject)
	m.putCalls = nil
	m.getCalls = nil
	m.statCalls = nil
	m.copyCalls = nil
	m.removeCalls = nil
	m.getHashCalls = nil
	m.presignURLCalls = nil
	m.putManyCalls = nil
	m.removeManyCalls = nil

	m.errToReturn = nil
	m.putError = nil
	m.getError = nil
	m.statError = nil
	m.copyError = nil
	m.removeError = nil
	m.getHashError = nil
	m.presignError = nil
	m.putManyError = nil
	m.removeManyError = nil

	m.presignedURLToReturn = ""
	m.hashToReturn = ""
}

// copyInternal copies an object between repositories in the mock storage.
//
// Takes params (storage_dto.CopyParams) which specifies the source and
// destination locations.
//
// Returns error when the source repository or object does not exist.
//
// Safe for concurrent use; guards access with a mutex.
func (m *MockStorageProvider) copyInternal(params storage_dto.CopyParams) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.copyCalls = append(m.copyCalls, params)

	if m.copyError != nil {
		return m.copyError
	}
	if m.errToReturn != nil {
		return m.errToReturn
	}

	srcRepo, ok := m.storage[params.SourceRepository]
	if !ok {
		return fmt.Errorf("source repository %s not found: %w", params.SourceRepository, ErrObjectNotFound)
	}
	srcObj, ok := srcRepo[params.SourceKey]
	if !ok {
		return fmt.Errorf("source object '%s' not found: %w", params.SourceKey, ErrObjectNotFound)
	}

	dataCopy := make([]byte, len(srcObj.data))
	copy(dataCopy, srcObj.data)
	var metadataCopy map[string]string
	if srcObj.metadata != nil {
		metadataCopy = make(map[string]string, len(srcObj.metadata))
		maps.Copy(metadataCopy, srcObj.metadata)
	}

	if _, ok := m.storage[params.DestinationRepository]; !ok {
		m.storage[params.DestinationRepository] = make(map[string]*mockObject)
	}
	m.storage[params.DestinationRepository][params.DestinationKey] = &mockObject{
		data:         dataCopy,
		contentType:  srcObj.contentType,
		metadata:     metadataCopy,
		lastModified: time.Now(),
	}

	return nil
}
