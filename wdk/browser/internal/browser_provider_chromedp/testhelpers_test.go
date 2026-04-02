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
	"testing"
	"time"
)

const perTestTimeout = 90 * time.Second

func requirePool(t *testing.T) *BrowserPool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	if testPool == nil {
		t.Fatal("browser pool not initialised - TestMain may have failed")
	}

	return testPool
}

func requireBrowser(t *testing.T) *Browser {
	t.Helper()

	pool := requirePool(t)
	return pool.browsers[0]
}

func withTestPage(t *testing.T, url string, callback func(t *testing.T, page *PageHelper)) {
	t.Helper()

	pool := requirePool(t)

	incognito, err := pool.NewIncognitoPage(context.Background())
	if err != nil {
		t.Fatalf("creating incognito page: %v", err)
	}

	timer := time.AfterFunc(perTestTimeout, func() {
		t.Errorf("test %s exceeded %s - force cancelling page context", t.Name(), perTestTimeout)
		incognito.Cancel()
	})
	defer timer.Stop()
	defer func() { _ = incognito.Close() }()

	page := NewPageHelper(incognito.Ctx)
	defer page.Close()

	if err := page.Navigate(url); err != nil {
		t.Fatalf("navigating to %s: %v", url, err)
	}

	callback(t, page)
}

func withTestPageNoNav(t *testing.T, callback func(t *testing.T, page *PageHelper)) {
	t.Helper()

	pool := requirePool(t)

	incognito, err := pool.NewIncognitoPage(context.Background())
	if err != nil {
		t.Fatalf("creating incognito page: %v", err)
	}

	timer := time.AfterFunc(perTestTimeout, func() {
		t.Errorf("test %s exceeded %s - force cancelling page context", t.Name(), perTestTimeout)
		incognito.Cancel()
	})
	defer timer.Stop()
	defer func() { _ = incognito.Close() }()

	page := NewPageHelper(incognito.Ctx)
	defer page.Close()

	callback(t, page)
}

func requireExclusivePool(t *testing.T) *ExclusiveBrowserPool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	if testExclusivePool == nil {
		t.Fatal("exclusive browser pool not initialised - TestMain may have failed")
	}

	return testExclusivePool
}

func withExclusivePage(t *testing.T, url string, callback func(t *testing.T, page *PageHelper)) {
	t.Helper()

	pool := requireExclusivePool(t)

	browser, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquiring exclusive browser: %v", err)
	}
	defer pool.Release(browser)

	incognito, err := browser.NewIncognitoPage()
	if err != nil {
		t.Fatalf("creating incognito page: %v", err)
	}

	timer := time.AfterFunc(perTestTimeout, func() {
		t.Errorf("test %s exceeded %s - force cancelling page context", t.Name(), perTestTimeout)
		incognito.Cancel()
	})
	defer timer.Stop()
	defer func() { _ = incognito.Close() }()

	page := NewPageHelper(incognito.Ctx)
	defer page.Close()

	if err := page.Navigate(url); err != nil {
		t.Fatalf("navigating to %s: %v", url, err)
	}

	callback(t, page)
}

func newActionContext(page *PageHelper) *ActionContext {
	return &ActionContext{
		Ctx:        page.Ctx(),
		PageHelper: page,
	}
}
