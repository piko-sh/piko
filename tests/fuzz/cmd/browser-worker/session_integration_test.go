//go:build integration

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

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/browser"
)

func requireChromium(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("google-chrome"); err == nil {
		return
	}
	if _, err := exec.LookPath("chromium"); err != nil {
		t.Skip("Chromium unavailable")
	}
}

func openTestBrowser(t *testing.T, budget time.Duration) (context.Context, *browser.Session) {
	t.Helper()
	ctx, session, closeBrowser, err := openBrowser(t.Context(), budget)
	require.NoError(t, err)
	t.Cleanup(closeBrowser)
	return ctx, session
}

func TestSessionBudgetEndsStepAsInfrastructure(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, `<p>never ready</p>`)
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx, session, closeBrowser, err := openBrowser(t.Context(), 8*time.Second)
	if err != nil {
		require.ErrorIs(t, err, errSessionBudget)
		return
	}
	defer closeBrowser()
	trace := driveCheckout(ctx, session, server.URL, casefile.CheckoutInput{Customer: "Alice", Quantity: 1})
	assert.Empty(t, trace.States)
	assert.True(t, strings.HasPrefix(trace.DriverError, errSessionBudget.Error()+": draft: "), trace.DriverError)
}
