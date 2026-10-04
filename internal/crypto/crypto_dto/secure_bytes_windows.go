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

//go:build windows && !safe

package crypto_dto

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// pageSize caches the system page size.
	pageSize = windows.Getpagesize()
)

// secureBytesCleanupData holds the data needed for runtime.AddCleanup. It is passed as
// the argument to the cleanup function when SecureBytes becomes unreachable.
type secureBytesCleanupData struct {
	// id is the unique identifier for tracking this cleanup operation.
	id string

	// data is the memory region to be zeroed and released on cleanup.
	data []byte

	// allocSize is the allocated memory size in bytes used for secure cleanup.
	allocSize int

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

// NewSecureBytes creates a new SecureBytes instance with secure memory allocation. The
// memory is allocated via VirtualAlloc (outside Go heap), locked in physical memory via
// VirtualLock (prevents swapping), and zero-initialised.
//
// Takes size (int) which specifies the number of bytes to allocate.
// Takes opts (...Option) which provides optional configuration settings.
//
// Returns *SecureBytes which is the allocated secure memory buffer.
// Returns error when size is not positive or memory allocation fails.
//
// The caller MUST call Close() when done to release memory. A finaliser is set as a
// safety net, but explicit Close() is preferred.
func NewSecureBytes(size int, opts ...Option) (*SecureBytes, error) {
	if size <= 0 {
		return nil, fmt.Errorf("%w: got %d", errSecureBytesInvalidSize, size)
	}

	allocSize := ((size + pageSize - 1) / pageSize) * pageSize

	addr, err := windows.VirtualAlloc(0, uintptr(allocSize), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, fmt.Errorf("VirtualAlloc failed: %w", err)
	}

	data := virtualAllocSlice(addr, allocSize)

	if err := windows.VirtualLock(addr, uintptr(allocSize)); err != nil {
		_ = windows.VirtualFree(addr, 0, windows.MEM_RELEASE)
		return nil, fmt.Errorf("%w: %w", errSecureBytesLockFailed, err)
	}

	secureBytes := &SecureBytes{
		data:      data,
		size:      size,
		allocSize: allocSize,
		id:        "",
		cleanup:   runtime.Cleanup{},
		mu:        sync.RWMutex{},
		closed:    atomic.Bool{},
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

// virtualAllocSlice views a VirtualAlloc allocation as a byte slice without copying.
//
// This is the only place the package turns an integer address into a pointer, and it is
// valid only because of where the address comes from. VirtualAlloc returns memory outside
// the Go heap. The garbage collector never scans, moves or frees it, so the address stays
// fixed and valid until VirtualFree, exactly like the memory syscall.Mmap returns on
// Unix. The resulting pointer is never mistaken for a reference to a Go object, and the
// runtime's pointer checks accept it because it lies outside every heap span.
//
// The conversion is written as unsafe.Add on a nil pointer, which is the same integer to
// pointer conversion as unsafe.Pointer(address). go vet reports the latter form because,
// for an arbitrary uintptr, it cannot know the address is not a heap object that has
// since moved; that hazard cannot arise for VirtualAlloc memory. Builds with the safe tag
// do not use this file and keep secrets in ordinary heap memory instead.
//
// Takes address (uintptr) which is the base address returned by VirtualAlloc.
// Takes length (int) which is the number of bytes in the allocation.
//
// Returns []byte which spans the whole allocation.
func virtualAllocSlice(address uintptr, length int) []byte {
	return unsafe.Slice((*byte)(unsafe.Add(nil, address)), length)
}

// NewSecureBytesFromSlice creates a SecureBytes instance by copying existing data into
// secure memory.
//
// The caller should zero the source data after this call if it contains sensitive
// material.
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

// secureBytesCleanup is called by the runtime when a SecureBytes becomes unreachable
// without having Close called. This is a safety net.
//
// Takes argument (*secureBytesCleanupData) which contains the memory details needed to
// zero and release the protected memory.
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
