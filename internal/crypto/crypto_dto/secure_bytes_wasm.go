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

//go:build js && wasm

package crypto_dto

import (
	"context"
	"fmt"
	"runtime"

	"piko.sh/piko/internal/logger/logger_domain"
)

// secureBytesCleanupData holds the data needed for runtime.AddCleanup.
type secureBytesCleanupData struct {
	// data holds the raw memory region to be securely zeroed and unmapped.
	data []byte

	// id is a unique label used to track cleanup of secure memory.
	id string

	// size is the byte length of the memory region to clear.
	size int
}

// platformClose performs WASM-specific cleanup by zeroing memory.
//
// Returns error when the cleanup fails, though currently this always returns
// nil as Go's garbage collector handles deallocation.
func (secureBytes *SecureBytes) platformClose() error {
	zeroMemory(secureBytes.data)
	return nil
}

// NewSecureBytes creates a new SecureBytes instance using regular Go memory.
// WASM does not support mmap/mlock, so this is less secure than native
// platforms but functional for WASM environments.
//
// Takes size (int) which specifies the number of bytes to allocate.
// Takes opts (...Option) which provides optional configuration settings.
//
// Returns *SecureBytes which is the allocated memory buffer.
// Returns error when size is not positive.
func NewSecureBytes(size int, opts ...Option) (*SecureBytes, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: got %d", errSecureBytesInvalidSize, size)
	}

	data := make([]byte, size)

	secureBytes := &SecureBytes{
		data:      data,
		size:      size,
		allocSize: size,
	}

	for _, opt := range opts {
		opt(secureBytes)
	}

	cleanupData := &secureBytesCleanupData{
		data: data,
		id:   secureBytes.id,
		size: size,
	}
	secureBytes.cleanup = runtime.AddCleanup(secureBytes, secureBytesCleanup, cleanupData)

	return secureBytes, nil
}

// NewSecureBytesFromSlice creates a SecureBytes instance from existing data.
//
// Takes source ([]byte) which provides the data to copy into secure memory.
// Takes opts (...Option) which configures the secure bytes behaviour.
//
// Returns *SecureBytes which contains a protected copy of the source data.
// Returns error when the source slice is empty or secure memory allocation
// fails.
//
// Safe for concurrent use by multiple goroutines.
func NewSecureBytesFromSlice(source []byte, opts ...Option) (*SecureBytes, error) {
	if len(source) == 0 {
		return nil, fmt.Errorf("%w: source slice is empty", errSecureBytesInvalidSize)
	}

	secureBytes, err := NewSecureBytes(len(source), opts...)
	if err != nil {
		return nil, err
	}

	secureBytes.mu.Lock()
	copy(secureBytes.data, source)
	secureBytes.mu.Unlock()

	return secureBytes, nil
}

// secureBytesCleanup is called by the runtime when a SecureBytes becomes
// unreachable.
//
// Takes argument (*secureBytesCleanupData) which contains the data to clear.
func secureBytesCleanup(argument *secureBytesCleanupData) {
	_, l := logger_domain.From(context.Background(), log)
	l.Warn("SecureBytes finaliser called - Close() was not called explicitly",
		logger_domain.String("id", argument.id),
		logger_domain.Int("size", argument.size))

	zeroMemory(argument.data)
}
