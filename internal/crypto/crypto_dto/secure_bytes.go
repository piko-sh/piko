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

package crypto_dto

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/crypto/crypto_dto")

	// errSecureBytesClosed is returned when attempting to access a closed
	// SecureBytes instance.
	errSecureBytesClosed = errors.New("secure bytes already closed")

	// errSecureBytesLockFailed is returned when memory locking fails.
	errSecureBytesLockFailed = errors.New("failed to lock memory")

	// errSecureBytesInvalidSize is returned when attempting to create
	// SecureBytes with an invalid size.
	errSecureBytesInvalidSize = errors.New("size must be positive")
)

// SecureBytes provides secure memory allocation for sensitive cryptographic
// key material. It implements io.ReadCloser and allocates memory outside the
// Go heap using platform-specific mechanisms to prevent garbage collector
// copying, lock memory against swapping, and provide explicit zeroing on
// destruction.
type SecureBytes struct {
	// id is an optional identifier for logging and debugging.
	id string

	// data holds the actual secure memory region.
	// On Unix, this is mmap'd memory; on Windows, this is VirtualAlloc'd memory.
	data []byte

	// cleanup holds the runtime cleanup handle, used to stop the cleanup when
	// Close is called.
	cleanup runtime.Cleanup

	// size is the originally requested length in bytes; data may be larger due to
	// page alignment.
	size int

	// allocSize is the actual memory size allocated, aligned to page boundaries.
	allocSize int

	// mu guards read and write access to the secure data.
	mu sync.RWMutex

	// closed tracks whether this instance has been closed.
	closed atomic.Bool
}

var (
	_ io.Closer = (*SecureBytes)(nil)

	_ io.Reader = (*SecureBytes)(nil)
)

// Option configures a SecureBytes instance.
type Option func(*SecureBytes)

// Close releases all resources held by the secure bytes, zeroing and releasing
// memory. Implements io.Closer.
//
// Returns error when the platform-specific cleanup fails.
//
// Safe for concurrent use. Multiple calls are safe; only the first call
// performs cleanup.
func (secureBytes *SecureBytes) Close() error {
	if !secureBytes.closed.CompareAndSwap(false, true) {
		return nil
	}

	secureBytes.mu.Lock()
	defer secureBytes.mu.Unlock()

	zeroMemory(secureBytes.data)

	if err := secureBytes.platformClose(); err != nil {
		return fmt.Errorf("platform-specific cleanup: %w", err)
	}

	secureBytes.data = nil
	secureBytes.cleanup.Stop()

	return nil
}

// Len returns the size of the secure bytes (the originally requested size).
//
// Returns int which is the byte count.
func (secureBytes *SecureBytes) Len() int {
	return secureBytes.size
}

// ID returns the identifier for this SecureBytes instance.
//
// Returns string which is the unique identifier.
func (secureBytes *SecureBytes) ID() string {
	return secureBytes.id
}

// IsClosed returns true if this SecureBytes instance has been closed.
//
// Returns bool which indicates whether the instance has been closed.
func (secureBytes *SecureBytes) IsClosed() bool {
	return secureBytes.closed.Load()
}

// Read implements io.Reader for scoped access to the secure bytes.
// This copies data OUT of secure memory - use WithAccess for zero-copy access.
//
// Takes p ([]byte) which is the buffer to fill with the secure data.
//
// Returns n (int) which is the number of bytes copied into p.
// Returns err (error) which is io.EOF after all bytes are read, or
// io.ErrShortBuffer if p is too small for the full content.
//
// Safe for concurrent use; acquires a read lock during access.
func (secureBytes *SecureBytes) Read(p []byte) (n int, err error) {
	if secureBytes.closed.Load() {
		return 0, errSecureBytesClosed
	}

	secureBytes.mu.RLock()
	defer secureBytes.mu.RUnlock()

	n = copy(p, secureBytes.data[:secureBytes.size])
	if n < secureBytes.size {
		return n, io.ErrShortBuffer
	}
	return n, io.EOF
}

// WithAccess provides scoped access to the underlying bytes without copying.
//
// The callback receives a view of the data. The caller MUST NOT:
//   - Store references to the data beyond the callback scope
//   - Modify the data (modifications are allowed but discouraged)
//
// This is the preferred method for using the key material as it avoids copying.
//
// Takes operation (func(data []byte) error) which processes the
// byte data in place.
//
// Returns error when the secure bytes are closed or the operation fails.
//
// Safe for concurrent use; uses a read lock during access.
func (secureBytes *SecureBytes) WithAccess(operation func(data []byte) error) error {
	if secureBytes.closed.Load() {
		return errSecureBytesClosed
	}

	secureBytes.mu.RLock()
	defer secureBytes.mu.RUnlock()

	if err := operation(secureBytes.data[:secureBytes.size]); err != nil {
		return fmt.Errorf("accessing secure bytes: %w", err)
	}
	return nil
}

// Clone creates a new SecureBytes instance with a copy of the data.
// The caller is responsible for closing the returned SecureBytes.
//
// Returns *SecureBytes which contains a copy of the data.
// Returns error when the SecureBytes is already closed.
//
// Safe for concurrent use.
func (secureBytes *SecureBytes) Clone() (*SecureBytes, error) {
	if secureBytes.closed.Load() {
		return nil, errSecureBytesClosed
	}

	secureBytes.mu.RLock()
	defer secureBytes.mu.RUnlock()

	cloneID := secureBytes.id
	if cloneID != "" {
		cloneID += "-clone"
	}

	return NewSecureBytesFromSlice(secureBytes.data[:secureBytes.size], WithID(cloneID))
}

// WithID sets an identifier for the SecureBytes instance, aiding debugging
// and logging to track which secret is being accessed.
//
// Takes id (string) which specifies the identifier for the instance.
//
// Returns Option which configures the SecureBytes with the given identifier.
func WithID(id string) Option {
	return func(secureBytes *SecureBytes) {
		secureBytes.id = id
	}
}

// zeroMemory sets all bytes in a slice to zero in a way that the compiler
// cannot remove. Uses runtime.KeepAlive to prevent dead store elimination.
//
// Takes data ([]byte) which is the memory region to zero.
//
// See: https://github.com/golang/go/issues/33325
func zeroMemory(data []byte) {
	for i := range data {
		data[i] = 0
	}
	runtime.KeepAlive(data)
}
