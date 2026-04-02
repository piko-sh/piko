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

package goroutine_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goroutine"
)

func triggerPanic(message string) {
	panic(message)
}

func TestRecoverPanic_RecoversPanic(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanic(context.Background(), "test.panickyGoroutine")
		triggerPanic("something went wrong")
	})

	wg.Wait()
}

func TestRecoverPanic_NoPanic(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanic(context.Background(), "test.normalGoroutine")
	})

	wg.Wait()
}

func TestRecoverPanicToChannel_SendsError(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanicToChannel(context.Background(), "test.channelPanic", errCh)
		triggerPanic("channel panic")
	})

	wg.Wait()

	require.Len(t, errCh, 1)
	err := <-errCh
	assert.Contains(t, err.Error(), "panic in test.channelPanic")
	assert.Contains(t, err.Error(), "channel panic")
}

func TestRecoverPanicToChannel_NilChannel(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanicToChannel(context.Background(), "test.nilChannel", nil)
		triggerPanic("nil channel panic")
	})

	wg.Wait()
}

func TestRecoverPanicToChannel_FullChannel(t *testing.T) {
	t.Parallel()

	errCh := make(chan error)
	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanicToChannel(context.Background(), "test.fullChannel", errCh)
		triggerPanic("full channel panic")
	})

	wg.Wait()
}

func TestRecoverPanicToChannel_NoPanic(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Go(func() {
		defer goroutine.RecoverPanicToChannel(context.Background(), "test.noPanic", errCh)
	})

	wg.Wait()
	assert.Empty(t, errCh)
}
