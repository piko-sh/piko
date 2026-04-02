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

//go:build bench

package interp_test_bench

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"piko.sh/piko/internal/interp/interp_domain"
)

func nativeSymbols() *interp_domain.SymbolRegistry {
	return interp_domain.NewSymbolRegistry(interp_domain.SymbolExports{
		"strings": {
			"ToUpper":    reflect.ValueOf(strings.ToUpper),
			"Contains":   reflect.ValueOf(strings.Contains),
			"HasPrefix":  reflect.ValueOf(strings.HasPrefix),
			"TrimSpace":  reflect.ValueOf(strings.TrimSpace),
			"ReplaceAll": reflect.ValueOf(strings.ReplaceAll),
		},
		"strconv": {
			"Itoa":      reflect.ValueOf(strconv.Itoa),
			"Atoi":      reflect.ValueOf(strconv.Atoi),
			"FormatInt": reflect.ValueOf(strconv.FormatInt),
		},
		"fmt": {
			"Sprintf": reflect.ValueOf(fmt.Sprintf),
		},
	})
}

func BenchmarkNativeCallExec(b *testing.B) {
	benchmarks := []struct {
		name   string
		source string
	}{
		{
			"string_to_string_1x",
			`package main

import "strings"

func run() string {
	return strings.ToUpper("hello")
}
`,
		},
		{
			"string_to_string_100x",
			`package main

import "strings"

func run() string {
	s := "hello"
	for i := 0; i < 100; i++ {
		s = strings.ToUpper(s)
	}
	return s
}
`,
		},
		{
			"string_to_string_1000x",
			`package main

import "strings"

func run() string {
	s := "hello"
	for i := 0; i < 1000; i++ {
		s = strings.ToUpper(s)
	}
	return s
}
`,
		},
		{
			"string_string_to_bool_100x",
			`package main

import "strings"

func run() int {
	count := 0
	for i := 0; i < 100; i++ {
		if strings.Contains("hello world", "world") {
			count++
		}
	}
	return count
}
`,
		},
		{
			"string_string_to_string_100x",
			`package main

import "strings"

func run() string {
	s := "hello world"
	for i := 0; i < 100; i++ {
		s = strings.ReplaceAll(s, "o", "0")
	}
	return s
}
`,
		},
		{
			"int_to_string_100x",
			`package main

import "strconv"

func run() string {
	s := ""
	for i := 0; i < 100; i++ {
		s = strconv.Itoa(i)
	}
	return s
}
`,
		},
		{
			"sprintf_100x",
			`package main

import "fmt"

func run() string {
	s := ""
	for i := 0; i < 100; i++ {
		s = fmt.Sprintf("value: %d, name: %s", i, "test")
	}
	return s
}
`,
		},
		{
			"format_int_100x",
			`package main

import "strconv"

func run() string {
	s := ""
	for i := 0; i < 100; i++ {
		s = strconv.FormatInt(int64(i), 10)
	}
	return s
}
`,
		},
		{
			"has_prefix_100x",
			`package main

import "strings"

func run() int {
	count := 0
	for i := 0; i < 100; i++ {
		if strings.HasPrefix("/api/v1/users", "/api") {
			count++
		}
	}
	return count
}
`,
		},
	}

	symbols := nativeSymbols()

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			service := interp_domain.NewService()
			service.UseSymbols(symbols)
			cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": bm.source})
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_, err = service.ExecuteEntrypoint(context.Background(), cfs, "run")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReflectCallBaseline(b *testing.B) {
	b.Run("reflect_string_to_string", func(b *testing.B) {
		reflectedFunction := reflect.ValueOf(strings.ToUpper)
		argument := reflect.ValueOf("hello")
		arguments := []reflect.Value{argument}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reflectedFunction.Call(arguments)
		}
	})

	b.Run("direct_string_to_string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = strings.ToUpper("hello")
		}
	})

	b.Run("typeassert_string_to_string", func(b *testing.B) {
		var untypedFunction any = strings.ToUpper
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			f, ok := untypedFunction.(func(string) string)
			_ = ok
			_ = f("hello")
		}
	})

	b.Run("reflect_string_string_to_bool", func(b *testing.B) {
		reflectedFunction := reflect.ValueOf(strings.Contains)
		arguments := []reflect.Value{reflect.ValueOf("hello world"), reflect.ValueOf("world")}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reflectedFunction.Call(arguments)
		}
	})

	b.Run("direct_string_string_to_bool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = strings.Contains("hello world", "world")
		}
	})

	b.Run("typeassert_string_string_to_bool", func(b *testing.B) {
		var untypedFunction any = strings.Contains
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			f, ok := untypedFunction.(func(string, string) bool)
			_ = ok
			_ = f("hello world", "world")
		}
	})

	b.Run("reflect_int_to_string", func(b *testing.B) {
		reflectedFunction := reflect.ValueOf(strconv.Itoa)
		argument := reflect.ValueOf(42)
		arguments := []reflect.Value{argument}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = reflectedFunction.Call(arguments)
		}
	})

	b.Run("direct_int_to_string", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = strconv.Itoa(42)
		}
	})

	b.Run("typeassert_int_to_string", func(b *testing.B) {
		var untypedFunction any = strconv.Itoa
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			f, ok := untypedFunction.(func(int) string)
			_ = ok
			_ = f(42)
		}
	})
}
