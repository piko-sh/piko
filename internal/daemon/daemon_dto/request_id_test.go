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

package daemon_dto

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/sharded"
)

func TestAcceptForwardedRequestID(t *testing.T) {
	accepted := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"4bf92f3577b34da6a3ce929d0e0e4736",
		"trace.span_1",
	}
	for _, candidate := range accepted {
		t.Run("accepts "+candidate, func(t *testing.T) {
			got, ok := AcceptForwardedRequestID(candidate)

			require.True(t, ok)
			assert.Equal(t, forwardedRequestIDPrefix+candidate, got,
				"an accepted id is marked as forwarded rather than passed through")
			assert.True(t, IsForwardedRequestID(got))
		})
	}

	t.Run("rejects the server's own format", func(t *testing.T) {
		_, ok := AcceptForwardedRequestID("host-1/AbCdEfGhIj-000042")

		assert.False(t, ok)
	})

	t.Run("a generated id is not reported as forwarded", func(t *testing.T) {
		assert.False(t, IsForwardedRequestID(FormatRequestID(42)))
	})

	rejected := map[string]string{
		"empty":            "",
		"newline":          "abc\ndef",
		"carriage return":  "abc\rdef",
		"terminal escape":  "abc\x1b[31mdef",
		"null byte":        "abc\x00def",
		"space":            "abc def",
		"log field syntax": `abc" level="fatal`,
		"over length":      strings.Repeat("a", 129),
	}
	for name, candidate := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			got, ok := AcceptForwardedRequestID(candidate)

			assert.False(t, ok)
			assert.Empty(t, got, "a rejected value is not partially kept")
		})
	}
}

func TestNextRequestIDCounter(t *testing.T) {
	t.Parallel()

	t.Run("values are unique and non-zero under concurrency", func(t *testing.T) {
		t.Parallel()

		const goroutineCount = 16
		const perGoroutine = 2000

		results := make([][]uint64, goroutineCount)
		var waitGroup sync.WaitGroup
		for index := range goroutineCount {
			waitGroup.Go(func() {
				values := make([]uint64, 0, perGoroutine)
				for range perGoroutine {
					counter, err := NextRequestIDCounter()
					if err != nil {
						return
					}
					values = append(values, counter)
				}
				results[index] = values
			})
		}
		waitGroup.Wait()

		seen := make(map[uint64]struct{}, goroutineCount*perGoroutine)
		for _, values := range results {
			require.Len(t, values, perGoroutine, "every call should return a value")
			for _, counter := range values {
				require.NotZero(t, counter, "zero is reserved for no generated ID")
				_, duplicate := seen[counter]
				require.False(t, duplicate, "counter %d was handed out twice", counter)
				seen[counter] = struct{}{}
			}
		}
	})

	t.Run("generated ids keep the documented format", func(t *testing.T) {
		t.Parallel()

		pattern := regexp.MustCompile("^" + regexp.QuoteMeta(requestIDPrefix) + `-[0-9]{6,}$`)
		for range 100 {
			counter, err := NextRequestIDCounter()
			require.NoError(t, err)

			requestID := FormatRequestID(counter)
			assert.Regexp(t, pattern, requestID)
			assert.False(t, IsForwardedRequestID(requestID))
		}
	})
}

func TestNextRequestIDCounterFrom(t *testing.T) {
	t.Parallel()

	t.Run("returns values from the sequence", func(t *testing.T) {
		t.Parallel()

		sequence, err := sharded.NewSequence(sharded.WithStart(firstRequestIDCounter), sharded.WithBlockSize(1))
		require.NoError(t, err)

		counter, err := nextRequestIDCounterFrom(sequence, nil)

		require.NoError(t, err)
		assert.Equal(t, uint64(firstRequestIDCounter), counter)
	})

	t.Run("reports an exhausted sequence", func(t *testing.T) {
		t.Parallel()

		sequence, err := sharded.NewSequence(sharded.WithStart(math.MaxUint64-1), sharded.WithBlockSize(1))
		require.NoError(t, err)

		counter, err := nextRequestIDCounterFrom(sequence, nil)
		require.NoError(t, err)
		assert.Equal(t, uint64(math.MaxUint64-1), counter)

		counter, err = nextRequestIDCounterFrom(sequence, nil)
		require.ErrorIs(t, err, sharded.ErrSequenceExhausted)
		assert.Zero(t, counter)
	})

	t.Run("reports a creation error", func(t *testing.T) {
		t.Parallel()

		creationErr := errors.New("creation failed")

		counter, err := nextRequestIDCounterFrom(nil, creationErr)

		require.ErrorIs(t, err, creationErr)
		assert.Zero(t, counter)
	})

	t.Run("reports a nil sequence", func(t *testing.T) {
		t.Parallel()

		counter, err := nextRequestIDCounterFrom(nil, nil)

		require.ErrorIs(t, err, errNilRequestIDSequence)
		assert.Zero(t, counter)
	})
}

func TestFallbackRequestIDCounter(t *testing.T) {
	t.Parallel()

	for range 100 {
		assert.NotZero(t, FallbackRequestIDCounter())
	}
}

func TestFormatRequestID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		want    string
		counter uint64
	}{
		{name: "pads small values", counter: 1, want: requestIDPrefix + "-000001"},
		{name: "keeps six digit values", counter: 999999, want: requestIDPrefix + "-999999"},
		{name: "does not truncate long values", counter: 1234567, want: requestIDPrefix + "-1234567"},
		{
			name:    "formats the largest value",
			counter: math.MaxUint64,
			want:    requestIDPrefix + "-18446744073709551615",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, FormatRequestID(testCase.counter))
		})
	}
}

func BenchmarkNextRequestIDCounter(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var sink uint64
		for pb.Next() {
			counter, err := NextRequestIDCounter()
			if err != nil {
				b.Error(err)
				return
			}
			sink += counter
		}
		_ = sink
	})
}
