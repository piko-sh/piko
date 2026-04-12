// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package interp_domain

import (
	"context"
	"testing"
)

func TestDumpFibUintBytecode(t *testing.T) {
	const source = `
package main

const fibTermsToCompute = 100000
const fibModulusBitmask = (1 << 64) - 1

func computeFib(n int) uint64 {
	previous := uint64(0)
	current := uint64(1)
	for index := 0; index < n; index++ {
		next := (previous + current) & fibModulusBitmask
		previous = current
		current = next
	}
	return current
}

func EntrypointRun() uint64 {
	return computeFib(fibTermsToCompute)
}
`
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go": source,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dump := compiled.DisassembleAssembly()
	t.Logf("\n=== UINT64 fib bytecode ===\n%s", dump)
}

func TestDumpFibIntBytecode(t *testing.T) {
	const source = `
package main

const fibTermsToCompute = 100000

func computeFibInt(n int) int64 {
	previous := int64(0)
	current := int64(1)
	for index := 0; index < n; index++ {
		next := previous + current
		previous = current
		current = next
	}
	return current
}

func EntrypointRun() int64 {
	return computeFibInt(fibTermsToCompute)
}
`
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go": source,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dump := compiled.DisassembleAssembly()
	t.Logf("\n=== INT64 fib bytecode ===\n%s", dump)
}

func TestDumpArithIntBytecode(t *testing.T) {
	const source = `
package main

func EntrypointRun() int64 {
	var sum int64
	for index := int64(0); index < 5000000; index++ {
		sum = sum + index
	}
	return sum
}
`
	service := NewService()
	compiled, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go": source,
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	dump := compiled.DisassembleAssembly()
	t.Logf("\n=== Pure int64 arithmetic bytecode ===\n%s", dump)
}
