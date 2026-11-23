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
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	historyStepBudget = 3 * time.Second
)

func newHistoryServer() *httptest.Server {
	return newTestServerWithRoutes(map[string]string{
		"/":      `<!DOCTYPE html><html><head><title>Home</title></head><body><p>home page body</p></body></html>`,
		"/page2": `<!DOCTYPE html><html><head><title>Page 2</title></head><body><p>second page body</p></body></html>`,
	})
}

func requireTitle(t *testing.T, actx *ActionContext, want string) {
	t.Helper()

	title, err := GetTitle(actx)
	require.NoError(t, err)
	assert.Equal(t, want, title)
}

func TestHistoryBackForward(t *testing.T) {
	t.Parallel()
	server := newHistoryServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL
		require.NoError(t, Navigate(actx, "/page2"))

		start := time.Now()
		require.NoError(t, GoBack(actx))
		assert.Less(t, time.Since(start), historyStepBudget)
		requireTitle(t, actx, "Home")

		start = time.Now()
		require.NoError(t, GoForward(actx))
		assert.Less(t, time.Since(start), historyStepBudget)
		requireTitle(t, actx, "Page 2")
	})
}

func TestHistoryNoEntry(t *testing.T) {
	t.Parallel()
	server := newHistoryServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)

		require.ErrorIs(t, GoForward(actx), errNoHistoryEntry)
	})
}

func TestHistoryCancelled(t *testing.T) {
	t.Parallel()
	server := newHistoryServer()
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		actx := newActionContext(page)
		actx.ServerURL = server.URL
		require.NoError(t, Navigate(actx, "/page2"))

		cause := errors.New("caller gave up")
		cancelled, cancel := context.WithCancelCause(page.Ctx())
		cancel(cause)
		actx.Ctx = cancelled

		require.ErrorIs(t, GoBack(actx), cause)
	})
}
