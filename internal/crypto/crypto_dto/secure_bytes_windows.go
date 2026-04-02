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

//go:build windows

package crypto_dto

import (
	"context"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
	"piko.sh/piko/internal/logger/logger_domain"
)

// pageSize caches the system page size.
var pageSize int

// secureBytesCleanupData holds the data needed for runtime.AddCleanup.
// It is passed as the argument to the cleanup function when SecureBytes
// becomes unreachable.
type secureBytesCleanupData struct {
	// data is the memory region to be zeroed and unmapped on cleanup.
	data []byte

	// allocSize is the allocated memory size in bytes used for secure cleanup.
	allocSize int

	// id is the unique identifier for tracking this cleanup operation.
	id string

	// size is the number of bytes in the secure buffer.
	size int
}

// platformClose releases Windows memory by unlocking and freeing it.
//
// Returns error when VirtualFree fails to release the memory.
func (secureBytes *SecureBytes) platformClose() error {
	_, l := logger_domain.From(context.Background(), log)
	addr := uintptr(unsafe.Pointer(&secureBytes.data[0]))

	if err := windows.VirtualUnlock(addr, uintptr(secureBytes.allocSize)); err != nil {
		l.Warn("VirtualUnlock failed during SecureBytes close",
			logger_domain.String("id", secureBytes.id),
			logger_domain.Error(err))
	}

	if err := windows.VirtualFree(addr, 0, windows.MEM_RELEASE); err != nil {
		return fmt.Errorf("VirtualFree failed: %w", err)
	}

	return nil
}

// NewSecureBytes creates a new SecureBytes instance with secure memory allocation.
// The memory is allocated via VirtualAlloc (outside Go heap), locked in physical
// memory via VirtualLock (prevents swapping), and zero-initialised.
//
// Takes size (int) which specifies the number of bytes to allocate.
// Takes opts (...Option) which provides optional configuration settings.
//
// Returns *SecureBytes which is the allocated secure memory buffer.
// Returns error when size is not positive or memory allocation fails.
//
// The caller MUST call Close() when done to release memory. A finaliser is set
// as a safety net, but explicit Close() is preferred.
func NewSecureBytes(size int, opts ...Option) (*SecureBytes, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: got %d", errSecureBytesInvalidSize, size)
	}

	allocSize := ((size + pageSize - 1) / pageSize) * pageSize

	addr, err := windows.VirtualAlloc(0, uintptr(allocSize), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, fmt.Errorf("VirtualAlloc failed: %w", err)
	}

	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), allocSize)

	if err := windows.VirtualLock(addr, uintptr(allocSize)); err != nil {
		_ = windows.VirtualFree(addr, 0, windows.MEM_RELEASE)
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

// NewSecureBytesFromSlice creates a SecureBytes instance by copying existing
// data into secure memory.
//
// The caller should zero the source data after this call if it contains
// sensitive material.
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
		return nil, err
	}

	secureBytes.mu.Lock()
	copy(secureBytes.data, source)
	secureBytes.mu.Unlock()

	return secureBytes, nil
}

// secureBytesCleanup is called by the runtime when a SecureBytes becomes
// unreachable without having Close called. This is a safety net.
//
// Takes argument (*secureBytesCleanupData) which contains the
// memory details needed to zero and release the protected memory.
func secureBytesCleanup(argument *secureBytesCleanupData) {
	_, l := logger_domain.From(context.Background(), log)
	l.Warn("SecureBytes finaliser called - Close() was not called explicitly",
		logger_domain.String("id", argument.id),
		logger_domain.Int("size", argument.size))

	zeroMemory(argument.data)

	addr := uintptr(unsafe.Pointer(&argument.data[0]))

	_ = windows.VirtualUnlock(addr, uintptr(argument.allocSize))

	_ = windows.VirtualFree(addr, 0, windows.MEM_RELEASE)
}

func init() {
	pageSize = windows.Getpagesize()
}
