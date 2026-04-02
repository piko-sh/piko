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
	"testing"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

func TestArenaCheck(t *testing.T) {
	code := `import "strings"

const size = 100
var s string
for r := 0; r < size*2; r++ {
	if r%2 == 0 {
		s += string(rune(r))
	}
}
n := 0
for r := 0; r < size*2; r++ {
	if strings.ContainsRune(s, rune(r)) {
		n++
	}
}
n`
	symbols := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service := interp_domain.NewService()
	service.UseSymbols(symbols)
	compiledFunction, err := service.Compile(context.Background(), code)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Root function: %s\n", compiledFunction.FuncName())
	fmt.Printf("  NumRegisters: %v\n", compiledFunction.RegisterCounts())
	fmt.Printf("  Body length: %d\n", compiledFunction.BodyLen())
	fmt.Printf("  Functions count: %d\n", len(compiledFunction.SubFunctions()))
	for i, f := range compiledFunction.SubFunctions() {
		fmt.Printf("  Functions[%d]: %s, NumRegisters=%v, Body=%d\n", i, f.FuncName(), f.RegisterCounts(), f.BodyLen())
	}

	result, err := service.Execute(context.Background(), compiledFunction)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("  Result: %v\n", result)
}

var concatOnlyCode = `const size = 100
var s string
for r := 0; r < size*2; r++ {
	if r%2 == 0 {
		s += string(rune(r))
	}
}
len(s)`

func BenchmarkPiko_ConcatOnly(b *testing.B) {
	service := interp_domain.NewService()
	compiledFunction, err := service.Compile(context.Background(), concatOnlyCode)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err = service.Execute(context.Background(), compiledFunction)
		if err != nil {
			b.Fatal(err)
		}
	}
}
