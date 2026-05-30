// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build bench

package interp_test_bench

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

const (
	fibIntInnerLoopSource = `
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
		var last int64
		for index := 0; index < 25; index++ {
			last = computeFibInt(fibTermsToCompute)
		}
		return last
	}
	`
	lruCacheSource = `
	package main
	type lruNode struct {
		key      int64
		value    int64
		previous *lruNode
		next     *lruNode
	}
	type lruCache struct {
		capacity int
		size     int
		nodes    map[int64]*lruNode
		head     *lruNode
		tail     *lruNode
	}
	func newCache(capacity int) *lruCache {
		return &lruCache{
			capacity: capacity,
			nodes:    map[int64]*lruNode{},
		}
	}
	func (c *lruCache) detach(node *lruNode) {
		if node.previous != nil {
			node.previous.next = node.next
		} else {
			c.head = node.next
		}
		if node.next != nil {
			node.next.previous = node.previous
		} else {
			c.tail = node.previous
		}
	}
	func (c *lruCache) attachFront(node *lruNode) {
		node.previous = nil
		node.next = c.head
		if c.head != nil {
			c.head.previous = node
		}
		c.head = node
		if c.tail == nil {
			c.tail = node
		}
	}
	func (c *lruCache) get(key int64) (int64, bool) {
		node, ok := c.nodes[key]
		if !ok {
			return 0, false
		}
		c.detach(node)
		c.attachFront(node)
		return node.value, true
	}
	func (c *lruCache) put(key int64, value int64) {
		if existing, ok := c.nodes[key]; ok {
			existing.value = value
			c.detach(existing)
			c.attachFront(existing)
			return
		}
		node := &lruNode{key: key, value: value}
		c.nodes[key] = node
		c.attachFront(node)
		c.size++
		if c.size > c.capacity {
			evict := c.tail
			c.detach(evict)
			delete(c.nodes, evict.key)
			c.size--
		}
	}
	func EntrypointRun() int64 {
		cache := newCache(1024)
		var sum int64
		for index := int64(0); index < 5000; index++ {
			cache.put(index, index*2)
		}
		for index := int64(0); index < 5000; index++ {
			v, _ := cache.get(index)
			sum += v
		}
		return sum
	}
	`
	arithIntSource = `
	package main
	func EntrypointRun() int64 {
		var sum int64
		for index := int64(0); index < 5000000; index++ {
			sum = sum + index
		}
		return sum
	}
	`
)

func runProfile(t *testing.T, name, source, profileFile string, iterations int) {
	t.Helper()
	service := interp_domain.NewService(interp_domain.WithMaxCallDepth(200000))
	symbolRegistry := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service.UseSymbols(symbolRegistry)

	compiledFileSet, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go": source,
	})
	if err != nil {
		t.Fatalf("%s compile: %v", name, err)
	}

	if _, err := service.ExecuteEntrypoint(context.Background(), compiledFileSet, "EntrypointRun"); err != nil {
		t.Fatalf("%s warmup: %v", name, err)
	}

	cpuFile, err := os.Create(profileFile)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	defer cpuFile.Close()

	memBefore := readAllocs()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatalf("start cpu profile: %v", err)
	}

	for range iterations {
		if _, err := service.ExecuteEntrypoint(context.Background(), compiledFileSet, "EntrypointRun"); err != nil {
			t.Fatalf("%s execute: %v", name, err)
		}
	}

	pprof.StopCPUProfile()

	memAfter := readAllocs()
	t.Logf("%s: allocs total=%d bytes count=%d (across %d iterations)",
		name,
		memAfter.totalAlloc-memBefore.totalAlloc,
		memAfter.mallocs-memBefore.mallocs,
		iterations)

	memProfFile := profileFile + ".mem"
	memFile, err := os.Create(memProfFile)
	if err != nil {
		t.Fatalf("create memprof: %v", err)
	}
	defer memFile.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		t.Fatalf("write heap profile: %v", err)
	}

	t.Logf("%s: wrote %s + .mem", name, profileFile)
}

func TestProfileFibInt(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}
	runProfile(t, "fib_int", fibIntInnerLoopSource, "/tmp/piko_fib_int.prof", 5)
}

func TestProfileLRUCache(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}
	runProfile(t, "lru_cache", lruCacheSource, "/tmp/piko_lru.prof", 20)
}

func TestProfileArithInt(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}
	runProfile(t, "arith_int", arithIntSource, "/tmp/piko_arith.prof", 5)
}

var (
	_ = fmt.Sprintf
)
