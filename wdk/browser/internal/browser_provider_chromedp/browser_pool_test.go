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

package browser_provider_chromedp

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestDefaultPoolSize(t *testing.T) {
	t.Parallel()

	size := DefaultPoolSize()

	if size < 1 {
		t.Errorf("DefaultPoolSize() = %d, want >= 1", size)
	}
	if size > maxPoolSize {
		t.Errorf("DefaultPoolSize() = %d, want <= %d", size, maxPoolSize)
	}

	cpus := runtime.NumCPU()
	half := cpus / 2
	switch {
	case half < 1:
		if size != 1 {
			t.Errorf("DefaultPoolSize() = %d, want 1 for %d CPUs", size, cpus)
		}
	case half > maxPoolSize:
		if size != maxPoolSize {
			t.Errorf("DefaultPoolSize() = %d, want %d for %d CPUs", size, maxPoolSize, cpus)
		}
	default:
		if size != half {
			t.Errorf("DefaultPoolSize() = %d, want %d for %d CPUs", size, half, cpus)
		}
	}
}

func TestNewBrowserPool(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode")
	}

	t.Run("creates pool with requested size", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 2)
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		if pool.Size() != 2 {
			t.Errorf("Size() = %d, want 2", pool.Size())
		}
	})

	t.Run("clamps size below 1 to 1", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 0)
		if err != nil {
			t.Fatalf("NewBrowserPool(size=0) error = %v", err)
		}
		defer pool.Close()

		if pool.Size() != 1 {
			t.Errorf("Size() = %d, want 1", pool.Size())
		}
	})
}

func TestBrowserPool_NewIncognitoPage(t *testing.T) {
	t.Parallel()
	pool := requirePool(t)

	t.Run("creates page successfully", func(t *testing.T) {
		page, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		defer func() { _ = page.Close() }()

		if page.Ctx == nil {
			t.Error("page context should not be nil")
		}
	})

	t.Run("distributes across browsers", func(t *testing.T) {
		if pool.Size() < 2 {
			t.Skip("pool too small to test distribution")
		}

		pages := make([]*IncognitoPage, pool.Size()*2)
		for i := range pages {
			p, err := pool.NewIncognitoPage(context.Background())
			if err != nil {
				t.Fatalf("NewIncognitoPage() #%d error = %v", i, err)
			}
			pages[i] = p
		}
		defer func() {
			for _, p := range pages {
				_ = p.Close()
			}
		}()

		for i, p := range pages {
			if p.Ctx == nil {
				t.Errorf("page %d context should not be nil", i)
			}
		}
	})
}

func TestBrowserPool_Close(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode")
	}

	opts := DefaultBrowserOptions()
	pool, err := NewBrowserPool(opts, 2)
	if err != nil {
		t.Fatalf("NewBrowserPool() error = %v", err)
	}

	pool.Close()

	pool.Close()
}

func TestBrowserPool_ConcurrentPageCreation(t *testing.T) {
	t.Parallel()
	pool := requirePool(t)

	const numPages = 10
	errs := make(chan error, numPages)

	for range numPages {
		go func() {
			page, err := pool.NewIncognitoPage(context.Background())
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = page.Close() }()
			errs <- nil
		}()
	}

	for range numPages {
		if err := <-errs; err != nil {
			t.Errorf("concurrent NewIncognitoPage() error = %v", err)
		}
	}
}

func TestBrowserPool_MaxConcurrentPages(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode")
	}

	t.Run("defaults to size * 50", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 2)
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		if pool.sem == nil {
			t.Fatal("sem should not be nil with default config")
		}
		if cap(pool.sem) != 2*defaultPagesPerBrowser {
			t.Errorf("sem capacity = %d, want %d", cap(pool.sem), 2*defaultPagesPerBrowser)
		}
	})

	t.Run("respects explicit config", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 2, BrowserPoolConfig{MaxConcurrentPages: 3})
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		if pool.sem == nil {
			t.Fatal("sem should not be nil")
		}
		if cap(pool.sem) != 3 {
			t.Errorf("sem capacity = %d, want 3", cap(pool.sem))
		}
	})

	t.Run("unlimited with -1", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 2, BrowserPoolConfig{MaxConcurrentPages: -1})
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		if pool.sem != nil {
			t.Error("sem should be nil for unlimited")
		}
	})

	t.Run("context cancellation while waiting", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 1, BrowserPoolConfig{MaxConcurrentPages: 1})
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		page, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("first NewIncognitoPage() error = %v", err)
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 50*time.Millisecond, fmt.Errorf("test: simulating expired deadline"))
		defer cancel()

		_, err = pool.NewIncognitoPage(ctx)
		if err == nil {
			t.Error("NewIncognitoPage() should fail when pool is full and context expires")
		}

		_ = page.Close()

		page2, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() after release error = %v", err)
		}
		_ = page2.Close()
	})

	t.Run("page close releases semaphore slot", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 1, BrowserPoolConfig{MaxConcurrentPages: 2})
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		p1, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() #1 error = %v", err)
		}
		p2, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() #2 error = %v", err)
		}

		_ = p1.Close()

		p3, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() #3 after close error = %v", err)
		}
		_ = p2.Close()
		_ = p3.Close()
	})

	t.Run("double close does not double-release", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewBrowserPool(opts, 1, BrowserPoolConfig{MaxConcurrentPages: 1})
		if err != nil {
			t.Fatalf("NewBrowserPool() error = %v", err)
		}
		defer pool.Close()

		page, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}

		_ = page.Close()
		_ = page.Close()

		p1, err := pool.NewIncognitoPage(context.Background())
		if err != nil {
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 50*time.Millisecond, fmt.Errorf("test: simulating expired deadline"))
		defer cancel()

		_, err = pool.NewIncognitoPage(ctx)
		if err == nil {
			t.Error("double close leaked a semaphore slot - second page should not have been created")
		}
		_ = p1.Close()
	})
}

func TestNewExclusiveBrowserPool(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode")
	}

	t.Run("creates pool with requested size", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewExclusiveBrowserPool(opts, 2)
		if err != nil {
			t.Fatalf("NewExclusiveBrowserPool() error = %v", err)
		}
		defer pool.Close()

		if pool.Size() != 2 {
			t.Errorf("Size() = %d, want 2", pool.Size())
		}
	})

	t.Run("clamps size below 1 to 1", func(t *testing.T) {
		opts := DefaultBrowserOptions()
		pool, err := NewExclusiveBrowserPool(opts, 0)
		if err != nil {
			t.Fatalf("NewExclusiveBrowserPool(size=0) error = %v", err)
		}
		defer pool.Close()

		if pool.Size() != 1 {
			t.Errorf("Size() = %d, want 1", pool.Size())
		}
	})
}

func TestExclusiveBrowserPool_AcquireRelease(t *testing.T) {
	t.Parallel()
	pool := requireExclusivePool(t)

	t.Run("acquire and release", func(t *testing.T) {
		browser, err := pool.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire() error = %v", err)
		}

		page, err := browser.NewIncognitoPage()
		if err != nil {
			pool.Release(browser)
			t.Fatalf("NewIncognitoPage() error = %v", err)
		}
		_ = page.Close()

		pool.Release(browser)
	})

	t.Run("acquire respects context cancellation", func(t *testing.T) {

		browsers := make([]*Browser, pool.Size())
		for i := range browsers {
			b, err := pool.Acquire(context.Background())
			if err != nil {

				for j := range i {
					pool.Release(browsers[j])
				}
				t.Fatalf("Acquire() #%d error = %v", i, err)
			}
			browsers[i] = b
		}

		ctx, cancel := context.WithTimeoutCause(context.Background(), 50*time.Millisecond, fmt.Errorf("test: simulating expired deadline"))
		defer cancel()

		_, err := pool.Acquire(ctx)
		if err == nil {
			t.Error("Acquire() should fail when pool is empty and context expires")
		}

		for _, b := range browsers {
			pool.Release(b)
		}
	})
}

func TestExclusiveBrowserPool_ConcurrentAcquire(t *testing.T) {
	t.Parallel()
	pool := requireExclusivePool(t)

	const numWorkers = 20
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	errs := make(chan error, numWorkers)

	for range numWorkers {
		go func() {
			defer wg.Done()

			browser, err := pool.Acquire(context.Background())
			if err != nil {
				errs <- err
				return
			}
			defer pool.Release(browser)

			page, err := browser.NewIncognitoPage()
			if err != nil {
				errs <- err
				return
			}
			_ = page.Close()
			errs <- nil
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("concurrent Acquire() error = %v", err)
		}
	}
}

func TestExclusiveBrowserPool_Close(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode")
	}

	opts := DefaultBrowserOptions()
	pool, err := NewExclusiveBrowserPool(opts, 2)
	if err != nil {
		t.Fatalf("NewExclusiveBrowserPool() error = %v", err)
	}

	pool.Close()

	pool.Close()
}
