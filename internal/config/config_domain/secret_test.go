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
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/safeerror"
)

func TestSecret_UnmarshalText(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("parses resolver prefix and key", func(t *testing.T) {
		var s Secret[string]
		err := s.UnmarshalText([]byte("env:MY_SECRET"))

		require.NoError(t, err)
		assert.Equal(t, "env:MY_SECRET", s.rawValue)
		assert.Equal(t, "env:", s.resolverPrefix)
		assert.Equal(t, "MY_SECRET", s.resolverKey)
		assert.True(t, s.IsSet())
	})

	t.Run("handles literal value without prefix", func(t *testing.T) {
		var s Secret[string]
		err := s.UnmarshalText([]byte("literal-value"))

		require.NoError(t, err)
		assert.Equal(t, "literal-value", s.rawValue)
		assert.Equal(t, "", s.resolverPrefix)
		assert.Equal(t, "literal-value", s.resolverKey)
	})

	t.Run("registers with secret manager", func(t *testing.T) {
		ResetSecretManager()
		initialCount := GetSecretManager().Count()

		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST"))

		assert.Equal(t, initialCount+1, GetSecretManager().Count())
	})
}

func TestSecret_IsSet(t *testing.T) {
	t.Run("returns false for zero value", func(t *testing.T) {
		var s Secret[string]
		assert.False(t, s.IsSet())
	})

	t.Run("returns true after unmarshal", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST"))
		assert.True(t, s.IsSet())
	})
}

func TestSecret_Acquire_String(t *testing.T) {
	t.Cleanup(ResetSecretManager)
	t.Cleanup(ResetGlobalResolverRegistry)

	registry := GetGlobalResolverRegistry()
	err := registry.Register(&EnvResolver{})
	require.NoError(t, err)

	t.Run("acquires literal value", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("my-literal-secret"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		defer handle.Release()

		value, err := handle.Value()
		require.NoError(t, err)
		assert.Equal(t, "my-literal-secret", value)
	})

	t.Run("acquires env value", func(t *testing.T) {
		t.Setenv("TEST_SECRET_KEY", "secret-from-env")

		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST_SECRET_KEY"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		defer handle.Release()

		value, err := handle.Value()
		require.NoError(t, err)
		assert.Equal(t, "secret-from-env", value)
	})

	t.Run("returns error for unset secret", func(t *testing.T) {
		var s Secret[string]

		_, err := s.Acquire(context.Background())
		assert.ErrorIs(t, err, ErrSecretNotSet)
	})

	t.Run("returns error for missing resolver", func(t *testing.T) {
		registry.Clear()

		var s Secret[string]
		_ = s.UnmarshalText([]byte("unknown:KEY"))

		_, err := s.Acquire(context.Background())
		assert.ErrorIs(t, err, ErrNoResolver)
	})

	t.Run("caches resolved value", func(t *testing.T) {
		t.Setenv("TEST_CACHED_KEY", "original-value")

		registry.Clear()
		_ = registry.Register(&EnvResolver{})

		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST_CACHED_KEY"))

		handle1, err := s.Acquire(context.Background())
		require.NoError(t, err)
		value1, err := handle1.Value()
		require.NoError(t, err)
		assert.Equal(t, "original-value", value1)
		handle1.Release()

		_ = os.Setenv("TEST_CACHED_KEY", "new-value")

		handle2, err := s.Acquire(context.Background())
		require.NoError(t, err)
		value2, err := handle2.Value()
		require.NoError(t, err)
		assert.Equal(t, "original-value", value2)
		handle2.Release()
	})
}

func TestSecret_Acquire_Bytes(t *testing.T) {
	t.Cleanup(ResetSecretManager)
	t.Cleanup(ResetGlobalResolverRegistry)

	registry := GetGlobalResolverRegistry()
	err := registry.Register(&EnvResolver{})
	require.NoError(t, err)

	t.Run("acquires env value as bytes", func(t *testing.T) {
		t.Setenv("TEST_BYTES_SECRET", "binary-secret-data")

		var s Secret[[]byte]
		_ = s.UnmarshalText([]byte("env:TEST_BYTES_SECRET"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		defer handle.Release()

		value, err := handle.Value()
		require.NoError(t, err)
		assert.Equal(t, []byte("binary-secret-data"), value)
	})
}

func TestSecretHandle_Release(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("decrements ref count", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), s.refCount.Load())

		handle.Release()
		assert.Equal(t, int64(0), s.refCount.Load())
	})

	t.Run("is idempotent", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)

		handle.Release()
		handle.Release()
		handle.Release()

		assert.Equal(t, int64(0), s.refCount.Load())
	})

	t.Run("Close is alias for Release", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)

		err = handle.Close()
		assert.NoError(t, err)
		assert.Equal(t, int64(0), s.refCount.Load())
	})
}

func TestSecretHandle_Value_ErrorReturns(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("returns safeerror when handle is released", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		handle.Release()

		_, err = handle.Value()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrSecretHandleClosed)

		var safeErr safeerror.Error
		require.True(t, errors.As(err, &safeErr), "expected safeerror.Error")
		assert.Equal(t, "config secret unavailable", safeErr.SafeMessage())
	})

	t.Run("returns safeerror when secret is closed", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		require.NoError(t, s.Close())

		_, err = handle.Value()
		require.Error(t, err)

		var safeErr safeerror.Error
		require.True(t, errors.As(err, &safeErr), "expected safeerror.Error")
		assert.Equal(t, "config secret unavailable", safeErr.SafeMessage())

		assert.True(t, errors.Is(err, ErrSecretClosed))
	})
}

func TestSecretHandle_TryValue(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("returns value and nil error", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		defer handle.Release()

		value, err := handle.TryValue()
		assert.NoError(t, err)
		assert.Equal(t, "test-value", value)
	})

	t.Run("returns error when released", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		handle, err := s.Acquire(context.Background())
		require.NoError(t, err)
		handle.Release()

		_, err = handle.TryValue()
		assert.ErrorIs(t, err, ErrSecretHandleClosed)
	})
}

func TestSecret_Refresh(t *testing.T) {
	t.Cleanup(ResetSecretManager)
	t.Cleanup(ResetGlobalResolverRegistry)

	registry := GetGlobalResolverRegistry()
	_ = registry.Register(&EnvResolver{})

	t.Run("clears cache for re-resolution", func(t *testing.T) {
		t.Setenv("TEST_REFRESH_KEY", "original")

		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST_REFRESH_KEY"))

		handle1, _ := s.Acquire(context.Background())
		value1, err := handle1.Value()
		require.NoError(t, err)
		assert.Equal(t, "original", value1)
		handle1.Release()

		_ = os.Setenv("TEST_REFRESH_KEY", "updated")
		s.Refresh()

		handle2, _ := s.Acquire(context.Background())
		value2, err := handle2.Value()
		require.NoError(t, err)
		assert.Equal(t, "updated", value2)
		handle2.Release()
	})

	t.Run("does not refresh with active handles", func(t *testing.T) {
		t.Setenv("TEST_REFRESH_ACTIVE", "original")

		var s Secret[string]
		_ = s.UnmarshalText([]byte("env:TEST_REFRESH_ACTIVE"))

		handle, _ := s.Acquire(context.Background())
		assert.True(t, s.resolved.Load())

		s.Refresh()
		assert.True(t, s.resolved.Load())

		handle.Release()
	})
}

func TestSecret_Close(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("closes and unregisters", func(t *testing.T) {
		ResetSecretManager()
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		initialCount := GetSecretManager().Count()
		err := s.Close()

		assert.NoError(t, err)
		assert.True(t, s.closed.Load())
		assert.Equal(t, initialCount-1, GetSecretManager().Count())
	})

	t.Run("acquire fails after close", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))
		_ = s.Close()

		_, err := s.Acquire(context.Background())
		assert.ErrorIs(t, err, ErrSecretClosed)
	})

	t.Run("close is idempotent", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		err1 := s.Close()
		err2 := s.Close()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
	})
}

func TestSecretManager_Shutdown(t *testing.T) {
	t.Run("closes all registered secrets", func(t *testing.T) {
		ResetSecretManager()
		sm := GetSecretManager()

		var secrets [3]Secret[string]
		for i := range secrets {
			_ = secrets[i].UnmarshalText([]byte("test"))
		}

		assert.Equal(t, 3, sm.Count())

		err := sm.Shutdown(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 0, sm.Count())

		for i := range secrets {
			assert.True(t, secrets[i].closed.Load())
		}
	})
}

func TestSecret_FieldPathSetter(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("sets field path", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("test-value"))

		s.SetFieldPath("Config.Database.Password")

		assert.Equal(t, "Config.Database.Password", s.FieldPath())
	})
}

func TestSecret_ConcurrentAccess(t *testing.T) {
	t.Cleanup(ResetSecretManager)

	t.Run("handles concurrent acquires", func(t *testing.T) {
		var s Secret[string]
		_ = s.UnmarshalText([]byte("concurrent-test-value"))

		const numGoroutines = 100
		done := make(chan struct{}, numGoroutines)

		for range numGoroutines {
			go func() {
				defer func() { done <- struct{}{} }()

				handle, err := s.Acquire(context.Background())
				if err != nil {
					return
				}
				_, _ = handle.Value()
				handle.Release()
			}()
		}

		for range numGoroutines {
			<-done
		}

		assert.Equal(t, int64(0), s.refCount.Load())
	})
}
