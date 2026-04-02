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
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/crypto/crypto_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// logKeyFieldPath is the log key for the secret field path.
const logKeyFieldPath = "field_path"

var (
	secretLog = logger_domain.GetLogger("piko/internal/config/config_domain/secret")

	// ErrSecretNotSet is returned when trying to acquire a secret that was never
	// populated.
	ErrSecretNotSet = errors.New("secret value not set")

	// ErrSecretClosed is returned when trying to acquire a closed secret.
	ErrSecretClosed = errors.New("secret has been closed")

	// ErrSecretResolutionFailed is returned when the resolver fails to resolve the
	// secret.
	ErrSecretResolutionFailed = errors.New("secret resolution failed")

	// ErrSecretHandleClosed is returned when trying to use a handle that has
	// already been released.
	ErrSecretHandleClosed = errors.New("secret handle has been closed")

	// ErrNoResolver is returned when no resolver is registered for a secret's
	// prefix.
	ErrNoResolver = errors.New("no resolver registered for secret prefix")
)

// Secret provides lazy-loaded, secure handling for configuration values.
// It implements io.Closer and stores a resolver reference, resolving on-demand
// when Acquire is called rather than at startup.
//
// Type parameter T should be:
//   - string: For text secrets (memory zeroing is best-effort)
//   - []byte: For binary secrets (stored in SecureBytes with mmap+mlock)
//
// Security features:
//   - Lazy loading: Secrets do not enter memory until needed
//   - Clear lifecycle: Acquire + Release for scoped access
//   - Reference counting: Multiple concurrent Acquire calls are safe
//   - Auto-registration: Secrets register with SecretManager for shutdown
//   - Finaliser safety net: Unreleased handles are cleaned up by GC
//
// Usage:
//
//	type Config struct {
//	    APIKey Secret[string] `config:"api_key"`
//	}
//
// // Later, when you need the secret:
// handle, err := config.APIKey.Acquire(ctx)
// if err != nil { return err }
// defer handle.Release()
// apiKey := handle.Value()
type Secret[T any] struct {
	// cachedValue holds the resolved and parsed value; only used for string type.
	cachedValue *T

	// secureBytes holds the resolved value for []byte type in secure memory.
	secureBytes *crypto_dto.SecureBytes

	// rawValue is the unresolved resolver reference (e.g. "env:MY_SECRET").
	rawValue string

	// resolverPrefix is the prefix that identifies the resolver (e.g. "env:").
	resolverPrefix string

	// resolverKey is the key used to look up the secret (e.g., "MY_SECRET").
	resolverKey string

	// fieldPath is the config field path used for logging and audit.
	fieldPath string

	// secretID is the unique identifier for this secret.
	secretID string

	// refCount tracks the number of active handles to this secret.
	refCount atomic.Int64

	// mu guards the secret fields while they are being resolved.
	mu sync.RWMutex

	// resolved indicates whether the secret has been resolved at least once.
	resolved atomic.Bool

	// closed indicates whether this secret has been permanently closed.
	closed atomic.Bool
}

// String implements fmt.Stringer to prevent accidental secret leakage
// via fmt.Sprintf, log output, or any other string formatting.
//
// Returns string which is always "[REDACTED]".
func (*Secret[T]) String() string { return "[REDACTED]" }

// GoString implements fmt.GoStringer to prevent leakage via %#v formatting.
//
// Returns string which is always "Secret[REDACTED]".
func (*Secret[T]) GoString() string { return "Secret[REDACTED]" }

// MarshalJSON implements json.Marshaler to prevent accidental secret
// leakage via JSON serialisation.
//
// Returns []byte which contains the JSON string "[REDACTED]".
// Returns error which is always nil.
func (*Secret[T]) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }

// MarshalText implements encoding.TextMarshaler to prevent accidental
// secret leakage via text serialisation.
//
// Returns []byte which contains the text "[REDACTED]".
// Returns error which is always nil.
func (*Secret[T]) MarshalText() ([]byte, error) { return []byte("[REDACTED]"), nil }

// UnmarshalText implements encoding.TextUnmarshaler for config loading.
// It stores the resolver reference without actually resolving the secret.
//
// Takes text ([]byte) which contains the raw secret reference to parse.
//
// Returns error when parsing fails.
func (s *Secret[T]) UnmarshalText(text []byte) error {
	rawValue := string(text)
	s.rawValue = rawValue

	colonIndex := strings.Index(rawValue, ":")
	if colonIndex == -1 {
		s.resolverPrefix = ""
		s.resolverKey = rawValue
	} else {
		s.resolverPrefix = rawValue[:colonIndex+1]
		s.resolverKey = rawValue[colonIndex+1:]
	}

	s.secretID = fmt.Sprintf("secret-%p", s)

	GetSecretManager().register(s)

	return nil
}

// SetFieldPath sets the config field path for this secret.
// This is called by the config walker after field population.
//
// Takes path (string) which specifies the config field path.
func (s *Secret[T]) SetFieldPath(path string) {
	s.fieldPath = path
}

// IsSet reports whether this secret has a value to resolve.
//
// Returns bool indicating whether the secret has a non-empty raw value.
func (s *Secret[T]) IsSet() bool {
	return s.rawValue != ""
}

// FieldPath returns the config field path for this secret.
//
// Returns string containing the dot-separated path to the config field.
func (s *Secret[T]) FieldPath() string {
	return s.fieldPath
}

// Acquire resolves the secret (if not already cached) and returns a handle. The
// handle provides access to the secret value and must be released when done.
//
// For []byte secrets, the value is stored in secure memory (mmap+mlock). For
// string secrets, the value is cached in regular memory.
//
// Returns *SecretHandle[T] which provides scoped access to the resolved
// secret value.
// Returns error when the secret is closed, not set, or resolution fails.
func (s *Secret[T]) Acquire(ctx context.Context) (*SecretHandle[T], error) {
	if s.closed.Load() {
		return nil, ErrSecretClosed
	}
	if !s.IsSet() {
		return nil, ErrSecretNotSet
	}

	if !s.resolved.Load() {
		if err := s.resolve(ctx); err != nil {
			return nil, fmt.Errorf("resolving secret for field %q: %w", s.fieldPath, err)
		}
	}

	s.refCount.Add(1)

	handle := &SecretHandle[T]{
		secret:   s,
		released: atomic.Bool{},
	}

	cleanupData := &secretHandleCleanupData[T]{
		secret:    s,
		fieldPath: s.fieldPath,
	}
	handle.cleanup = runtime.AddCleanup(handle, secretHandleCleanup[T], cleanupData)

	secretLog.Trace("Secret acquired",
		logger_domain.String(logKeyFieldPath, s.fieldPath),
		logger_domain.Int64("ref_count", s.refCount.Load()))

	return handle, nil
}

// resolve fetches the secret value using the registered resolver.
//
// Returns error when no resolver is registered, resolution fails, or storage
// fails.
//
// Safe for concurrent use. Acquires the mutex and uses double-checked locking
// to ensure the secret is resolved only once.
func (s *Secret[T]) resolve(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resolved.Load() {
		return nil
	}

	registry := GetGlobalResolverRegistry()

	var resolvedValue string
	var err error

	if s.resolverPrefix == "" {
		resolvedValue = s.resolverKey
	} else {
		resolver := registry.Get(s.resolverPrefix)
		if resolver == nil {
			return fmt.Errorf("%w: %s", ErrNoResolver, s.resolverPrefix)
		}

		resolvedValue, err = resolver.Resolve(ctx, s.resolverKey)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSecretResolutionFailed, err)
		}
	}

	if err := s.storeValue(resolvedValue); err != nil {
		return fmt.Errorf("storing resolved value for field %q: %w", s.fieldPath, err)
	}

	s.resolved.Store(true)

	secretLog.Trace("Secret resolved",
		logger_domain.String(logKeyFieldPath, s.fieldPath),
		logger_domain.String("resolver", s.resolverPrefix))

	return nil
}

// storeValue saves the resolved value in the correct format for its type.
//
// Takes resolvedValue (string) which is the secret value to store.
//
// Returns error when the value cannot be converted to the target type.
func (s *Secret[T]) storeValue(resolvedValue string) error {
	var t T
	switch any(t).(type) {
	case []byte:
		secureBytes, err := crypto_dto.NewSecureBytesFromSlice(
			[]byte(resolvedValue),
			crypto_dto.WithID(s.secretID),
		)
		if err != nil {
			return fmt.Errorf("failed to create secure bytes for secret: %w", err)
		}
		s.secureBytes = secureBytes
	case string:
		typedValue, ok := any(resolvedValue).(T)
		if !ok {
			return errors.New("unexpected type assertion failure for string secret")
		}
		s.cachedValue = &typedValue
	default:
		typedValue, ok := any(resolvedValue).(T)
		if !ok {
			return errors.New("unexpected type assertion failure for secret")
		}
		s.cachedValue = &typedValue
	}

	return nil
}

// Refresh clears the cached value, forcing re-resolution on next Acquire.
// Use it when secrets may have been rotated.
//
// When active handles exist, the refresh is skipped and a warning is logged.
//
// Safe for concurrent use.
func (s *Secret[T]) Refresh() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.refCount.Load() > 0 {
		secretLog.Warn("Cannot refresh secret with active handles",
			logger_domain.String(logKeyFieldPath, s.fieldPath),
			logger_domain.Int64("ref_count", s.refCount.Load()))
		return
	}

	s.resolved.Store(false)
	s.cachedValue = nil
	if s.secureBytes != nil {
		_ = s.secureBytes.Close()
		s.secureBytes = nil
	}

	secretLog.Trace("Secret cache cleared",
		logger_domain.String(logKeyFieldPath, s.fieldPath))
}

// isResolved reports whether the secret has been resolved at least once.
//
// Returns bool indicating whether the secret has been resolved.
func (s *Secret[T]) isResolved() bool {
	return s.resolved.Load()
}

// Close releases all resources held by this secret.
// After Close, Acquire returns ErrSecretClosed.
//
// Returns error when the underlying secure bytes cannot be closed.
//
// Safe for concurrent use; uses an atomic compare-and-swap for the closed flag
// and a mutex for resource cleanup.
func (s *Secret[T]) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedValue != nil {
		var zero T
		*s.cachedValue = zero
		s.cachedValue = nil
	}

	if s.secureBytes != nil {
		if err := s.secureBytes.Close(); err != nil {
			return fmt.Errorf("closing secure bytes for field %q: %w", s.fieldPath, err)
		}
		s.secureBytes = nil
	}

	GetSecretManager().unregister(s)

	return nil
}

// getValue returns the current value for use by SecretHandle.
// The caller must hold an active reference.
//
// Returns T which is a copy of the secret value.
// Returns error when the secret is closed or not set.
//
// Safe for concurrent use. Protected by a read lock.
func (s *Secret[T]) getValue() (T, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zero T

	if s.closed.Load() {
		return zero, ErrSecretClosed
	}

	var t T
	switch any(t).(type) {
	case []byte:
		if s.secureBytes == nil {
			return zero, ErrSecretNotSet
		}
		var result []byte
		err := s.secureBytes.WithAccess(func(data []byte) error {
			result = make([]byte, len(data))
			copy(result, data)
			return nil
		})
		if err != nil {
			return zero, fmt.Errorf("accessing secure bytes for field %q: %w", s.fieldPath, err)
		}
		typedValue, ok := any(result).(T)
		if !ok {
			return zero, errors.New("unexpected type assertion failure for []byte secret")
		}
		return typedValue, nil
	default:
		if s.cachedValue == nil {
			return zero, ErrSecretNotSet
		}
		return *s.cachedValue, nil
	}
}

// release decreases the reference count by one.
func (s *Secret[T]) release() {
	newCount := s.refCount.Add(-1)
	secretLog.Trace("Secret released",
		logger_domain.String(logKeyFieldPath, s.fieldPath),
		logger_domain.Int64("ref_count", newCount))
}

// secretCloser defines a type that can release its resources when closed.
// It is used by SecretManager to track and close registered secrets.
type secretCloser interface {
	// Close releases any resources held by the iterator.
	//
	// Returns error when the close operation fails.
	Close() error

	// isResolved reports whether the secret has been resolved at least once.
	isResolved() bool
}

// SecretHandle provides scoped access to a secret value.
// It must be released when no longer needed to allow the secret to be refreshed
// or cleaned up during shutdown.
//
// SecretHandle implements io.Closer for use with defer.
type SecretHandle[T any] struct {
	// secret stores the underlying secret value.
	secret *Secret[T]

	// released indicates whether the secret has been released and is no longer
	// valid.
	released atomic.Bool

	// cleanup releases resources when the handle is no longer reachable.
	cleanup runtime.Cleanup
}

// secretHandleCleanupData holds the data needed for runtime.AddCleanup.
// It is passed to the cleanup function when the SecretHandle becomes
// unreachable.
type secretHandleCleanupData[T any] struct {
	// secret is the secret being tracked for cleanup.
	secret *Secret[T]

	// fieldPath is the path to the field being cleaned up.
	fieldPath string
}

// Value returns the secret value.
//
// Returns T which is the decrypted secret value.
//
// Panics if the handle has been released or if the secret value cannot be
// retrieved.
func (h *SecretHandle[T]) Value() T {
	if h.released.Load() {
		panic("attempted to access released secret handle")
	}

	value, err := h.secret.getValue()
	if err != nil {
		panic(fmt.Sprintf("failed to get secret value: %v", err))
	}
	return value
}

// TryValue returns the secret value and any error.
// This is a non-panicking alternative to Value.
//
// Returns T which is the secret value.
// Returns error when the handle has been released or retrieval fails.
func (h *SecretHandle[T]) TryValue() (T, error) {
	var zero T
	if h.released.Load() {
		return zero, ErrSecretHandleClosed
	}
	return h.secret.getValue()
}

// Release frees this handle and lowers the secret's reference count.
// This method is idempotent; calling it more than once is safe.
func (h *SecretHandle[T]) Release() {
	if !h.released.CompareAndSwap(false, true) {
		return
	}

	h.secret.release()
	h.cleanup.Stop()
}

// Close implements io.Closer by calling Release, so SecretHandle can be
// used with defer.
//
// Returns error which is always nil.
func (h *SecretHandle[T]) Close() error {
	h.Release()
	return nil
}

// secretHandleCleanup is called by the runtime when a SecretHandle becomes
// unreachable without having Release() called. This acts as a safety net to
// ensure the secret is still released.
//
// Takes argument (*secretHandleCleanupData[T]) which contains the data needed to
// clean up the secret handle.
func secretHandleCleanup[T any](argument *secretHandleCleanupData[T]) {
	secretLog.Warn("SecretHandle finaliser called - Release() was not called explicitly",
		logger_domain.String("field_path", argument.fieldPath))

	argument.secret.release()
}
