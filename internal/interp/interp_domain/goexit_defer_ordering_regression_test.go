package interp_domain_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
	"piko.sh/piko/internal/interp/interp_domain"
)

func TestGoexitDeferOrderingRegression(t *testing.T) {
	const src = `package main

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
)

func run() string {
	var wg sync.WaitGroup
	var defersRan []string
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			r := recover()
			mu.Lock()
			defersRan = append(defersRan, fmt.Sprintf("recover=%v", r))
			mu.Unlock()
		}()
		defer func() {
			mu.Lock()
			defersRan = append(defersRan, "inner-defer")
			mu.Unlock()
		}()
		runtime.Goexit()
	}()
	wg.Wait()

	sort.Strings(defersRan)
	result := fmt.Sprintf("count=%d;", len(defersRan))
	for _, d := range defersRan {
		result += d + ";"
	}
	return result
}`
	const want = "count=2;inner-defer;recover=<nil>;"

	iters := 400
	if testing.Short() {
		iters = 50
	}
	const parallel = 8

	var mu sync.Mutex
	var mismatches int
	var wg sync.WaitGroup
	for range parallel {
		wg.Go(func() {
			for range iters {
				service := interp_domain.NewService()
				service.UseSymbolProviders(driven_system_symbols.NewProvider())
				result, err := service.EvalFile(context.Background(), src, "run")
				got := strings.TrimSpace(fmt.Sprint(result))
				if err != nil || got != want {
					mu.Lock()
					mismatches++
					if mismatches <= 3 {
						t.Errorf("goexit defer ordering broke: got=%q err=%v (want %q)", got, err, want)
					}
					mu.Unlock()
				}
			}
		})
	}
	wg.Wait()
	if mismatches > 0 {
		t.Errorf("total mismatches: %d/%d", mismatches, parallel*iters)
	}
}
