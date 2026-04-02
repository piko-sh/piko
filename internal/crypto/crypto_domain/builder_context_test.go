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

package crypto_domain

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err = service.NewEncrypt().Data("secret").Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, provider.encryptCallCount(), "provider should not be called")
}

func TestEncryptBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err = service.NewEncrypt().Data("secret").Do(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 0, provider.encryptCallCount(), "provider should not be called")
}

func TestDecryptBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err = service.NewDecrypt().Data("ciphertext").Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestDecryptBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err = service.NewDecrypt().Data("ciphertext").Do(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestBatchEncryptBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	factory := newMockLocalProviderFactory()
	service, err := createTestService(provider, createTestConfig("key-1"), WithLocalProviderFactory(factory))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err = service.NewBatchEncrypt().Items([]string{"a", "b"}).Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, provider.encryptCallCount(), "provider should not be called")
}

func TestBatchEncryptBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	factory := newMockLocalProviderFactory()
	service, err := createTestService(provider, createTestConfig("key-1"), WithLocalProviderFactory(factory))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err = service.NewBatchEncrypt().Items([]string{"a", "b"}).Do(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 0, provider.encryptCallCount(), "provider should not be called")
}

func TestBatchDecryptBuilder_Do_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	factory := newMockLocalProviderFactory()
	service, err := createTestService(provider, createTestConfig("key-1"), WithLocalProviderFactory(factory))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err = service.NewBatchDecrypt().Items([]string{"a", "b"}).Do(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestBatchDecryptBuilder_Do_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	factory := newMockLocalProviderFactory()
	service, err := createTestService(provider, createTestConfig("key-1"), WithLocalProviderFactory(factory))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err = service.NewBatchDecrypt().Items([]string{"a", "b"}).Do(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStreamEncryptBuilder_Stream_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	var buffer bytes.Buffer
	_, err = service.NewStreamEncrypt().Output(&buffer).KeyID("key-1").Stream(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestStreamEncryptBuilder_Stream_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	var buffer bytes.Buffer
	_, err = service.NewStreamEncrypt().Output(&buffer).KeyID("key-1").Stream(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStreamDecryptBuilder_Stream_CancelledContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	_, err = service.NewStreamDecrypt().Input(strings.NewReader("data")).Stream(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestStreamDecryptBuilder_Stream_ExpiredContext(t *testing.T) {
	t.Parallel()

	provider := newMockProvider()
	service, err := createTestService(provider, createTestConfig("key-1"))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeoutCause(context.Background(), 0, fmt.Errorf("test: simulating expired deadline"))
	defer cancel()

	_, err = service.NewStreamDecrypt().Input(strings.NewReader("data")).Stream(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
