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

package interp_domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func runWithDeadline(t *testing.T, ctx context.Context, source, entrypoint string) (any, error) {
	t.Helper()
	type outcome struct {
		result any
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := NewService().EvalFile(ctx, source, entrypoint)
		done <- outcome{result: result, err: err}
	}()
	select {
	case got := <-done:
		return got.result, got.err
	case <-time.After(10 * time.Second):
		t.Fatal("interpreter did not return: a blocking channel/select operation deadlocked")
		return nil, nil
	}
}

func TestChannelReceiveUnblocksOnContextCancel(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	return <-ch
}

func main() {}
`
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := runWithDeadline(t, ctx, source, "run")
	require.Error(t, err, "a blocked receive must surface ctx cancellation")
}

func TestChannelSendUnblocksOnContextCancel(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	ch <- 1
	return 0
}

func main() {}
`
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := runWithDeadline(t, ctx, source, "run")
	require.Error(t, err, "a blocked send must surface ctx cancellation")
}

func TestSelectUnblocksOnContextCancel(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	select {
	case v := <-ch:
		return v
	}
}

func main() {}
`
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := runWithDeadline(t, ctx, source, "run")
	require.Error(t, err, "a blocked select must surface ctx cancellation")
}

func TestBlockedReceiveUnblocksOnSiblingPanic(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	go func() {
		panic("sibling boom")
	}()
	return <-ch
}

func main() {}
`
	_, err := runWithDeadline(t, context.Background(), source, "run")
	require.Error(t, err, "a sibling goroutine panic must unblock the parked receive")
	require.Contains(t, err.Error(), "sibling boom", "the surfaced error should carry the panic value")
}

func TestSelectUnblocksOnSiblingPanic(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	go func() {
		panic("select boom")
	}()
	select {
	case v := <-ch:
		return v
	}
}

func main() {}
`
	_, err := runWithDeadline(t, context.Background(), source, "run")
	require.Error(t, err, "a sibling goroutine panic must unblock the parked select")
	require.Contains(t, err.Error(), "select boom")
}

func TestSelectWithDefaultStillFiresDefault(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	select {
	case v := <-ch:
		return v
	default:
		return 42
	}
}

func main() {}
`
	result, err := runWithDeadline(t, t.Context(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
}

func TestUnbufferedChannelRoundTripUnaffected(t *testing.T) {
	t.Parallel()

	source := `package main

func run() int {
	ch := make(chan int)
	go func() {
		ch <- 7
	}()
	return <-ch
}

func main() {}
`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runWithDeadline(t, ctx, source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(7), result)
	require.NoError(t, ctx.Err(), "the handshake should complete well before the deadline")
}
