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

//go:build linux || darwin || freebsd || openbsd || netbsd

package crypto_dto

import (
	"context"
	"fmt"
	"runtime"

	"golang.org/x/sys/unix"
	"piko.sh/piko/internal/logger/logger_domain"
)

// pageSize caches the system page size.
var pageSize = unix.Getpagesize()

// secureBytesCleanupData holds the data needed for runtime.AddCleanup.
// It is passed to the cleanup function when SecureBytes becomes unreachable.
type secureBytesCleanupData struct {
	// id is the unique identifier used when logging.
	id string

	// data holds the memory-mapped byte slice to be zeroed and unmapped.
	data []byte

	// allocSize is the total number of bytes set aside for the secure memory region.
	allocSize int

	// size is the number of bytes in the secure buffer.
	size int
}

// platformClose performs Unix-specific cleanup using munlock and munmap.
//
// Returns error when munmap fails.
func (secureBytes *SecureBytes) platformClose() error {
	_, l := logger_domain.From(context.Background(), log)
	if err := unix.Munlock(secureBytes.data[:secureBytes.allocSize]); err != nil {
		l.Warn("munlock failed during SecureBytes close",
			logger_domain.String("id", secureBytes.id),
			logger_domain.Error(err))
	}

	if err := unix.Munmap(secureBytes.data[:secureBytes.allocSize]); err != nil {
		return fmt.Errorf("munmap failed: %w", err)
	}

	return nil
}

// NewSecureBytes creates a new SecureBytes instance with secure memory
// allocation. The memory is allocated via mmap (outside Go heap), locked in
// physical memory via mlock (prevents swapping), and zero-initialised.
//
// The caller MUST call Close() when done to release memory. A finaliser is set
// as a safety net, but explicit Close() is preferred.
//
// Takes size (int) which specifies the number of bytes to allocate.
// Takes opts (...Option) which provides optional configuration settings.
//
// Returns *SecureBytes which is the secure memory buffer ready for use.
// Returns error when size is not positive or memory allocation fails.
func NewSecureBytes(size int, opts ...Option) (*SecureBytes, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: got %d", errSecureBytesInvalidSize, size)
	}

	allocSize := ((size + pageSize - 1) / pageSize) * pageSize

	data, err := unix.Mmap(-1, 0, allocSize, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	if err := unix.Mlock(data); err != nil {
		_ = unix.Munmap(data)
		return nil, fmt.Errorf("%w: %w", errSecureBytesLockFailed, err)
	}

	secureBytes := &SecureBytes{
		data:      data,
		size:      size,
		allocSize: allocSize,
	}

	for _, opt := range opts {
		opt(secureBytes)
	}

	cleanupData := &secureBytesCleanupData{
		data:      data,
		allocSize: allocSize,
		id:        secureBytes.id,
		size:      size,
	}
	secureBytes.cleanup = runtime.AddCleanup(secureBytes, secureBytesCleanup, cleanupData)

	return secureBytes, nil
}

// NewSecureBytesFromSlice creates a SecureBytes instance from an existing byte
// slice by copying the data into secure memory. The caller should zero the
// source data after this call if it contains sensitive material.
//
// Takes source ([]byte) which provides the data to copy into secure memory.
// Takes opts (...Option) which configures the SecureBytes behaviour.
//
// Returns *SecureBytes which contains a protected copy of the source data.
// Returns error when the source slice is empty or memory allocation fails.
//
// Safe for concurrent use by multiple goroutines.
func NewSecureBytesFromSlice(source []byte, opts ...Option) (*SecureBytes, error) {
	if len(source) == 0 {
		return nil, fmt.Errorf("%w: source slice is empty", errSecureBytesInvalidSize)
	}

	secureBytes, err := NewSecureBytes(len(source), opts...)
	if err != nil {
		return nil, fmt.Errorf("allocating secure bytes from slice: %w", err)
	}

	secureBytes.mu.Lock()
	copy(secureBytes.data, source)
	secureBytes.mu.Unlock()

	return secureBytes, nil
}

// secureBytesCleanup is called by the runtime when a SecureBytes becomes
// unreachable without Close having been called. This acts as a safety net to
// ensure secure memory is zeroed and freed.
//
// Takes argument (*secureBytesCleanupData) which contains the memory region to
// clean up.
func secureBytesCleanup(argument *secureBytesCleanupData) {
	_, l := logger_domain.From(context.Background(), log)
	l.Warn("SecureBytes finaliser called - Close() was not called explicitly",
		logger_domain.String("id", argument.id),
		logger_domain.Int("size", argument.size))

	zeroMemory(argument.data)

	_ = unix.Munlock(argument.data[:argument.allocSize])

	_ = unix.Munmap(argument.data[:argument.allocSize])
}
