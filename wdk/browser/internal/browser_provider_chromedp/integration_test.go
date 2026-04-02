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
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"go.uber.org/goleak"
)

var (
	testPool          *BrowserPool
	testExclusivePool *ExclusiveBrowserPool
)

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		os.Exit(m.Run())
	}

	opts := DefaultBrowserOptions()
	opts.Headless = true

	type poolResult struct {
		pool          *BrowserPool
		exclusivePool *ExclusiveBrowserPool
		err           error
	}

	poolSize := DefaultPoolSize()

	sharedDone := make(chan poolResult, 1)
	exclusiveDone := make(chan poolResult, 1)

	go func() {
		p, err := NewBrowserPool(opts, poolSize)
		sharedDone <- poolResult{pool: p, err: err}
	}()
	go func() {
		ep, err := NewExclusiveBrowserPool(opts, poolSize)
		exclusiveDone <- poolResult{exclusivePool: ep, err: err}
	}()

	timeout := time.After(60 * time.Second)

	select {
	case r := <-sharedDone:
		if r.err != nil {
			panic("failed to start shared browser pool: " + r.err.Error())
		}
		testPool = r.pool
	case <-timeout:
		_, _ = fmt.Fprintf(os.Stderr, "timeout: browser pools failed to start within 60s\n")
		os.Exit(1)
	}

	select {
	case r := <-exclusiveDone:
		if r.err != nil {
			testPool.Close()
			panic("failed to start exclusive browser pool: " + r.err.Error())
		}
		testExclusivePool = r.exclusivePool
	case <-timeout:
		testPool.Close()
		_, _ = fmt.Fprintf(os.Stderr, "timeout: browser pools failed to start within 60s\n")
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stderr, "browser pools: %d shared + %d exclusive instances\n",
		testPool.Size(), testExclusivePool.Size())

	code := m.Run()

	if testPool != nil {
		testPool.Close()
	}
	if testExclusivePool != nil {
		testExclusivePool.Close()
	}

	if code == 0 {
		if err := goleak.Find(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "goleak: %v\n", err)
			os.Exit(1)
		}
	}
	os.Exit(code)
}
