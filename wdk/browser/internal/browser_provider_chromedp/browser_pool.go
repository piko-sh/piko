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
	"sync/atomic"
)

const (
	// maxPoolSize caps the number of browser instances to avoid excessive memory
	// usage (~100-300 MB per Chrome process).
	maxPoolSize = 16

	// defaultPagesPerBrowser is the default maximum number of concurrent pages
	// allowed per browser instance, applied when MaxConcurrentPages is not set.
	defaultPagesPerBrowser = 12
)

// BrowserPoolConfig configures optional behaviour of a BrowserPool.
type BrowserPoolConfig struct {
	// MaxConcurrentPages limits the number of pages that can be active
	// simultaneously across all pool members. When the limit is reached,
	// NewIncognitoPage blocks until an existing page is closed.
	//
	// Defaults to poolSize * MaxPagesPerBrowser when zero or unset.
	// Set to -1 for unlimited.
	MaxConcurrentPages int

	// MaxPagesPerBrowser limits the number of concurrent pages per browser
	// instance. Used to compute the default MaxConcurrentPages as
	// poolSize * MaxPagesPerBrowser when MaxConcurrentPages is not set.
	//
	// Defaults to 12 when zero or unset. Set to -1 for unlimited.
	MaxPagesPerBrowser int
}

// BrowserPool manages multiple Browser instances and distributes page creation
// across them via round-robin. This enables genuine multi-process parallelism
// by spreading work across separate Chrome processes, each of which can
// utilise its own CPU core.
//
// Safe for concurrent use.
type BrowserPool struct {
	// sem limits concurrent active pages. nil means unlimited.
	sem chan struct{}

	// browsers holds the pool of Browser instances.
	browsers []*Browser

	// index is the atomic round-robin counter for distributing page creation.
	index atomic.Uint64
}

// NewBrowserPool creates a pool of size browser instances in parallel sharing
// the same options, closing any successfully started browsers and returning
// the first error if any browser fails to start.
//
// An optional BrowserPoolConfig may be provided to limit the number of
// concurrent active pages. When MaxConcurrentPages is set, NewIncognitoPage
// blocks until a slot is available and releases it automatically when the
// page is closed.
//
// Takes opts (BrowserOptions) which specifies the browser settings.
// Takes size (int) which is the number of browser instances to create.
// Takes config (optional BrowserPoolConfig) which configures pool behaviour.
//
// Returns *BrowserPool which is ready for concurrent page creation.
// Returns error when any browser instance fails to start.
func NewBrowserPool(opts BrowserOptions, size int, config ...BrowserPoolConfig) (*BrowserPool, error) {
	browsers, err := startBrowsers(opts, size)
	if err != nil {
		return nil, err
	}

	pagesPerBrowser := defaultPagesPerBrowser
	if len(config) > 0 && config[0].MaxPagesPerBrowser != 0 {
		pagesPerBrowser = config[0].MaxPagesPerBrowser
	}

	maxPages := size * pagesPerBrowser
	if len(config) > 0 && config[0].MaxConcurrentPages != 0 {
		maxPages = config[0].MaxConcurrentPages
	}

	var sem chan struct{}
	if maxPages > 0 {
		sem = make(chan struct{}, maxPages)
	}

	return &BrowserPool{
		browsers: browsers,
		sem:      sem,
	}, nil
}

// NewIncognitoPage creates a new isolated browser page, distributing across
// pool members via round-robin. Each call selects the next browser in the
// pool, ensuring even load distribution.
//
// When MaxConcurrentPages is configured, this method blocks until a slot is
// available (or ctx is cancelled) and automatically releases it when the
// returned page is closed.
//
// Takes ctx (context.Context) which controls the semaphore wait timeout.
//
// Returns *IncognitoPage which provides an isolated browsing context.
// Returns error when page creation fails or ctx is cancelled while waiting.
func (bp *BrowserPool) NewIncognitoPage(ctx context.Context) (*IncognitoPage, error) {
	if bp.sem != nil {
		select {
		case bp.sem <- struct{}{}:
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for pool slot: %w", ctx.Err())
		}
	}

	index := bp.index.Add(1) - 1
	b := bp.browsers[index%uint64(len(bp.browsers))]
	page, err := b.NewIncognitoPage()
	if err != nil {
		if bp.sem != nil {
			<-bp.sem
		}
		return nil, err
	}

	if bp.sem != nil {
		sem := bp.sem
		page.releaseFunction = func() { <-sem }
	}

	return page, nil
}

// Size returns the number of browser instances in the pool.
//
// Returns int which is the number of browsers in the pool.
func (bp *BrowserPool) Size() int {
	return len(bp.browsers)
}

// Close closes all browser instances in the pool and cleans up resources.
func (bp *BrowserPool) Close() {
	for _, b := range bp.browsers {
		if b != nil {
			b.Close()
		}
	}
}

// ExclusiveBrowserPool manages a pool of Browser instances where each browser
// is checked out to at most one consumer at a time. This guarantees that only
// a single tab is active per Chrome process, which is required for tests that
// depend on browser-level focus, blur, or mouse-capture semantics.
//
// Safe for concurrent use.
type ExclusiveBrowserPool struct {
	// avail is a buffered channel acting as a semaphore. Browsers are sent
	// back into the channel when released.
	avail chan *Browser

	// all holds every browser for cleanup.
	all []*Browser
}

// NewExclusiveBrowserPool creates a pool of size browser instances in parallel.
// All browsers are initially available for checkout via Acquire.
//
// Takes opts (BrowserOptions) which specifies the browser settings.
// Takes size (int) which is the number of browser instances to create.
//
// Returns *ExclusiveBrowserPool which is ready for exclusive browser
// checkout.
// Returns error when any browser instance fails to start.
func NewExclusiveBrowserPool(opts BrowserOptions, size int) (*ExclusiveBrowserPool, error) {
	browsers, err := startBrowsers(opts, size)
	if err != nil {
		return nil, err
	}

	avail := make(chan *Browser, len(browsers))
	for _, b := range browsers {
		avail <- b
	}

	return &ExclusiveBrowserPool{
		all:   browsers,
		avail: avail,
	}, nil
}

// Acquire checks out a browser for exclusive use, blocking until a browser
// is available or ctx is cancelled, where the caller must call Release when
// done.
//
// Returns *Browser which the caller has exclusive access to.
// Returns error when ctx is cancelled before a browser becomes available.
func (ep *ExclusiveBrowserPool) Acquire(ctx context.Context) (*Browser, error) {
	select {
	case b := <-ep.avail:
		return b, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Release returns a browser to the pool, making it available for other callers.
// Must be called exactly once for each successful Acquire.
//
// Takes b (*Browser) which is the browser to return to the pool.
func (ep *ExclusiveBrowserPool) Release(b *Browser) {
	ep.avail <- b
}

// Size returns the number of browser instances in the pool.
//
// Returns int which is the total number of browsers in the pool.
func (ep *ExclusiveBrowserPool) Size() int {
	return len(ep.all)
}

// Close closes all browser instances in the pool and cleans up resources.
func (ep *ExclusiveBrowserPool) Close() {
	for _, b := range ep.all {
		if b != nil {
			b.Close()
		}
	}
}

// DefaultPoolSize returns a sensible default number of browsers based on
// available CPU cores, capped at maxPoolSize and floored at 1. Uses half
// the CPU count so that a shared pool and an exclusive pool can each claim
// half of the available cores.
//
// Returns int which is the recommended pool size for the current machine.
func DefaultPoolSize() int {
	n := runtime.NumCPU() / 2
	if n > maxPoolSize {
		return maxPoolSize
	}
	if n < 1 {
		return 1
	}
	return n
}

// startBrowsers launches size Chrome processes in parallel and returns them.
// If any browser fails to start, successfully started ones are closed and
// the first error is returned.
//
// Takes opts (BrowserOptions) which specifies the browser launch settings.
// Takes size (int) which is the number of browsers to start.
//
// Returns []*Browser which contains the started browser instances.
// Returns error when any browser fails to start.
//
// Safe for concurrent use. Launches each browser in its own goroutine and
// waits for all to complete before returning.
func startBrowsers(opts BrowserOptions, size int) ([]*Browser, error) {
	if size < 1 {
		size = 1
	}

	browsers := make([]*Browser, size)
	errs := make([]error, size)

	var wg sync.WaitGroup
	wg.Add(size)
	for i := range size {
		go func() {
			defer wg.Done()
			b, err := NewBrowser(opts)
			browsers[i] = b
			errs[i] = err
		}()
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			for j, b := range browsers {
				if j != i && b != nil {
					b.Close()
				}
			}
			return nil, fmt.Errorf("starting browser %d/%d: %w", i+1, size, err)
		}
	}

	return browsers, nil
}
