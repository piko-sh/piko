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

	"github.com/stretchr/testify/require"
)

func TestTypedChannelFastPathCorrectness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "chan_int_send_recv",
			code: `func run() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	a := <-ch
	b := <-ch
	c := <-ch
	return a + b + c
}
run()`,
			expect: int64(60),
		},
		{
			name: "chan_string_send_recv",
			code: `func run() string {
	ch := make(chan string, 2)
	ch <- "hello"
	ch <- "world"
	a := <-ch
	b := <-ch
	return a + " " + b
}
run()`,
			expect: "hello world",
		},
		{
			name: "chan_bool_send_recv",
			code: `func run() bool {
	ch := make(chan bool, 2)
	ch <- true
	ch <- false
	a := <-ch
	b := <-ch
	return a && !b
}
run()`,
			expect: true,
		},
		{
			name: "chan_float_send_recv",
			code: `func run() float64 {
	ch := make(chan float64, 2)
	ch <- 1.5
	ch <- 2.5
	a := <-ch
	b := <-ch
	return a + b
}
run()`,
			expect: 4.0,
		},
		{
			name: "chan_int_loop_pingpong",
			code: `func run() int {
	ch := make(chan int, 1)
	s := 0
	for i := 0; i < 100; i++ {
		ch <- i
		s += <-ch
	}
	return s
}
run()`,
			expect: int64(4950),
		},
		{
			name: "chan_int_comma_ok_after_close",
			code: `func run() int {
	ch := make(chan int, 2)
	ch <- 7
	ch <- 13
	close(ch)
	a, ok1 := <-ch
	b, ok2 := <-ch
	c, ok3 := <-ch
	if !ok1 || !ok2 || ok3 {
		return -1
	}
	return a + b + c
}
run()`,
			expect: int64(20),
		},
		{
			name: "chan_int_range_over",
			code: `func run() int {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)
	s := 0
	for v := range ch {
		s += v
	}
	return s
}
run()`,
			expect: int64(6),
		},
		{
			name: "chan_int_goroutine_pingpong",
			code: `func run() int {
	ch := make(chan int)
	go func() {
		for i := 0; i < 50; i++ {
			ch <- i
		}
		close(ch)
	}()
	s := 0
	for v := range ch {
		s += v
	}
	return s
}
run()`,
			expect: int64(1225),
		},
		{
			name: "chan_string_unbuffered_goroutine",
			code: `func run() string {
	ch := make(chan string)
	go func() {
		ch <- "alpha"
		ch <- "beta"
		ch <- "gamma"
		close(ch)
	}()
	result := ""
	for v := range ch {
		result += v
	}
	return result
}
run()`,
			expect: "alphabetagamma",
		},
		{
			name: "named_chan_int_type",
			code: `type IntChan chan int
func run() int {
	var ch IntChan = make(chan int, 2)
	ch <- 11
	ch <- 22
	return <-ch + <-ch
}
run()`,
			expect: int64(33),
		},
		{
			name: "chan_int64_explicit_type",
			code: `func run() int64 {
	ch := make(chan int64, 1)
	ch <- int64(99)
	return <-ch
}
run()`,
			expect: int64(99),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}
