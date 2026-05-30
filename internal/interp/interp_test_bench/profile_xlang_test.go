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
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

func profileXLangBench(t *testing.T, benchDirName string, kInner int, iterations int) {
	t.Helper()
	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	profileXLangBenchWithRoot(t, repoRoot, benchDirName, kInner, iterations)
}

func profileXLangBenchWithRoot(t *testing.T, repoRoot string, benchDirName string, kInner int, iterations int) {
	t.Helper()

	sourcePath := filepath.Join(repoRoot,
		"tests/benchmarks/cross_language/benchmarks",
		benchDirName,
		"go", "piko_source.go")
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read %s: %v", sourcePath, err)
	}

	wrapperSource := fmt.Sprintf(`package main

func EntrypointRun() (string, int64) {
	return RunInner(%d)
}
`, kInner)

	service := interp_domain.NewService(interp_domain.WithMaxCallDepth(200000))
	symbolRegistry := interp_domain.NewSymbolRegistry(driven_system_symbols.NewProvider().Exports())
	service.UseSymbols(symbolRegistry)

	compiledFileSet, err := service.CompileFileSet(context.Background(), map[string]string{
		"main.go":       string(sourceBytes),
		"entrypoint.go": wrapperSource,
	})
	if err != nil {
		t.Fatalf("compile %s: %v", benchDirName, err)
	}

	if _, err := service.ExecuteEntrypoint(context.Background(), compiledFileSet, "EntrypointRun"); err != nil {
		t.Fatalf("warmup %s: %v", benchDirName, err)
	}

	cpuPath := fmt.Sprintf("/tmp/piko_xlang_%s.prof", benchDirName)
	cpuFile, err := os.Create(cpuPath)
	if err != nil {
		t.Fatalf("create %s: %v", cpuPath, err)
	}
	defer cpuFile.Close()

	memBefore := readAllocs()

	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		t.Fatalf("start cpu profile: %v", err)
	}

	wallStart := time.Now()
	for range iterations {
		if _, err := service.ExecuteEntrypoint(context.Background(), compiledFileSet, "EntrypointRun"); err != nil {
			pprof.StopCPUProfile()
			t.Fatalf("execute %s: %v", benchDirName, err)
		}
	}
	wallElapsed := time.Since(wallStart)

	pprof.StopCPUProfile()

	memAfter := readAllocs()
	t.Logf("%s: iterations=%d k_inner=%d wall=%v alloc_bytes=%d alloc_count=%d",
		benchDirName, iterations, kInner, wallElapsed,
		memAfter.totalAlloc-memBefore.totalAlloc,
		memAfter.mallocs-memBefore.mallocs)

	memPath := cpuPath + ".mem"
	memFile, err := os.Create(memPath)
	if err != nil {
		t.Fatalf("create %s: %v", memPath, err)
	}
	defer memFile.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(memFile); err != nil {
		t.Fatalf("write heap profile: %v", err)
	}

	t.Logf("%s: wrote %s and %s", benchDirName, cpuPath, memPath)
}

func repoRootFromHere() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for range 16 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found")
		}
		dir = parent
	}
	return "", fmt.Errorf("walked too many parents")
}

func TestProfileXLangExpressionEval(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}

	profileXLangBench(t, "05_expression_eval_10k", 3, 5)
}

func TestProfileXLangLRUCache(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}

	profileXLangBench(t, "06_lru_cache_100k_ops", 3, 10)
}

func TestProfileXLangDijkstra(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}

	profileXLangBench(t, "07_dijkstra_10k_nodes", 5, 30)
}

func TestProfileXLangMarkov(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}

	profileXLangBench(t, "10_markov_text_gen_10k_words", 5, 80)
}

func TestProfileXLangPolyAst(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}
	profileXLangBench(t, "13_polymorphic_ast_eval_500k", 5, 3)
}

func TestProfileXLangParallelWordCount(t *testing.T) {
	if testing.Short() {
		t.Skip("profile run")
	}

	repoRoot, err := repoRootFromHere()
	if err != nil {
		t.Fatalf("locate repo root: %v", err)
	}
	prevWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	benchDir := filepath.Join(repoRoot, "tests/benchmarks/cross_language/benchmarks/16_parallel_word_count_montecristo")
	if err := os.Chdir(benchDir); err != nil {
		t.Fatalf("chdir bench dir: %v", err)
	}
	defer os.Chdir(prevWd)

	profileXLangBenchWithRoot(t, repoRoot, "16_parallel_word_count_montecristo", 3, 5)
}
