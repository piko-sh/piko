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
)

const (
	poolStartTimeout = 60 * time.Second
)

var (
	testPool          *BrowserPool
	testExclusivePool *ExclusiveBrowserPool
)

type browserPoolMain struct {
	m *testing.M
}

type poolResult struct {
	pool          *BrowserPool
	exclusivePool *ExclusiveBrowserPool
	err           error
}

func newBrowserPoolMain(m *testing.M) *browserPoolMain {
	return &browserPoolMain{m: m}
}

func (b *browserPoolMain) Run() int {
	flag.Parse()

	if testing.Short() {
		return b.m.Run()
	}

	if err := startTestPools(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "starting browser pools: %v\n", err)
		closeTestPools()
		return 1
	}
	defer closeTestPools()

	_, _ = fmt.Fprintf(os.Stderr, "browser pools: %d shared + %d exclusive instances\n",
		testPool.Size(), testExclusivePool.Size())

	return b.m.Run()
}

func startTestPools() error {
	opts := DefaultBrowserOptions()
	opts.Headless = true
	poolSize := DefaultPoolSize()

	sharedDone := make(chan poolResult, 1)
	exclusiveDone := make(chan poolResult, 1)

	go func() {
		p, err := NewBrowserPool(opts, poolSize)
		sharedDone <- poolResult{pool: p, exclusivePool: nil, err: err}
	}()
	go func() {
		ep, err := NewExclusiveBrowserPool(opts, poolSize)
		exclusiveDone <- poolResult{pool: nil, exclusivePool: ep, err: err}
	}()

	timeout := time.NewTimer(poolStartTimeout)
	defer timeout.Stop()

	var startErr error
	for range 2 {
		select {
		case r := <-sharedDone:
			testPool = r.pool
			if r.err != nil && startErr == nil {
				startErr = fmt.Errorf("shared browser pool: %w", r.err)
			}
		case r := <-exclusiveDone:
			testExclusivePool = r.exclusivePool
			if r.err != nil && startErr == nil {
				startErr = fmt.Errorf("exclusive browser pool: %w", r.err)
			}
		case <-timeout.C:
			return fmt.Errorf("browser pools did not start within %s", poolStartTimeout)
		}
	}
	return startErr
}

func closeTestPools() {
	if testPool != nil {
		testPool.Close()
		testPool = nil
	}
	if testExclusivePool != nil {
		testExclusivePool.Close()
		testExclusivePool = nil
	}
}
