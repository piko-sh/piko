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

func runSafeModeProgram(t *testing.T, code string) (any, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return NewService(WithSafeMode()).Eval(ctx, code)
}

func TestSafeModeGoroutinesSerialiseSharedState(t *testing.T) {
	t.Parallel()

	t.Run("concurrent map writes do not crash", func(t *testing.T) {
		t.Parallel()
		code := `
shared := make(map[int]int)
done := make(chan int, 8)
for w := 0; w < 8; w++ {
	go func(id int) {
		for k := 0; k < 3000; k++ {
			shared[k % 32] = id
		}
		done <- id
	}(w)
}
for i := 0; i < 8; i++ {
	<-done
}
len(shared)`
		result, err := runSafeModeProgram(t, code)
		require.NoError(t, err)
		require.Equal(t, int64(32), result)
	})

	t.Run("shared upvalue mutation completes", func(t *testing.T) {
		t.Parallel()
		code := `
var boxed interface{} = 0
done := make(chan int, 8)
for w := 0; w < 8; w++ {
	go func(id int) {
		for k := 0; k < 3000; k++ {
			if id % 2 == 0 {
				boxed = 1234567
			} else {
				boxed = "a string payload"
			}
			_ = boxed
		}
		done <- id
	}(w)
}
total := 0
for i := 0; i < 8; i++ {
	total += <-done
}
total`
		result, err := runSafeModeProgram(t, code)
		require.NoError(t, err)
		require.Equal(t, int64(28), result)
	})
}

func TestSafeModeUnbufferedProducerConsumer(t *testing.T) {
	t.Parallel()

	code := `
ch := make(chan int)
go func() {
	for i := 1; i <= 100; i++ {
		ch <- i
	}
	close(ch)
}()
sum := 0
for v := range ch {
	sum += v
}
sum`
	result, err := runSafeModeProgram(t, code)
	require.NoError(t, err)
	require.Equal(t, int64(5050), result)
}

func TestSafeModeSelectFanIn(t *testing.T) {
	t.Parallel()
	code := `
a := make(chan int)
b := make(chan int)
go func() { a <- 10 }()
go func() { b <- 20 }()
total := 0
for i := 0; i < 2; i++ {
	select {
	case x := <-a:
		total += x
	case y := <-b:
		total += y
	}
}
total`
	result, err := runSafeModeProgram(t, code)
	require.NoError(t, err)
	require.Equal(t, int64(30), result)
}
