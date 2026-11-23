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
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko"
	"piko.sh/piko/tests/fuzz/internal/shippedhost"
)

func TestShippedTagsMatchSharedHelper(t *testing.T) {
	assert.Equal(t, shippedhost.Tags(), shippedTags)
	options, err := shippedOptions()
	require.NoError(t, err)
	assert.Len(t, options, len(shippedTags))
}

func TestReadyReturnsOnlyReadinessToken(t *testing.T) {
	t.Setenv(envReadyToken, "token")
	t.Setenv(envCoverageSecret, "secret")
	t.Setenv(envGoCoverage, "")
	recorder := httptest.NewRecorder()

	ready(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "token", recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), "secret")
}

func TestReadyRefusesReadinessTokenAsCoverageSecret(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(envReadyToken, "token")
	t.Setenv(envCoverageSecret, "secret")
	t.Setenv(envGoCoverage, directory)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil)
	request.Header.Set(coverageHeader, "token")
	recorder := httptest.NewRecorder()

	ready(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "token", recorder.Body.String())
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCoverageAuthorised(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		secret string
		header string
		want   bool
	}{
		{name: "matching secret", secret: "secret", header: "secret", want: true},
		{name: "wrong secret", secret: "secret", header: "token", want: false},
		{name: "missing header", secret: "secret", header: "", want: false},
		{name: "unset secret", secret: "", header: "", want: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv(envCoverageSecret, testCase.secret)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil)
			if testCase.header != "" {
				request.Header.Set(coverageHeader, testCase.header)
			}
			assert.Equal(t, testCase.want, coverageAuthorised(request))
		})
	}
}

func TestReadyReportsCoverageFailure(t *testing.T) {
	t.Setenv(envReadyToken, "token")
	t.Setenv(envCoverageSecret, "secret")
	t.Setenv(envGoCoverage, t.TempDir())
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil)
	request.Header.Set(coverageHeader, "secret")
	recorder := httptest.NewRecorder()
	ready(recorder, request)
	assert.Contains(t, []int{http.StatusNoContent, http.StatusInternalServerError}, recorder.Code)
}

func TestRunRejectsInvalidPort(t *testing.T) {
	t.Setenv("PIKO_FUZZ_PORT", "not-a-port")

	err := run(t.Context())

	require.ErrorContains(t, err, "reading PIKO_FUZZ_PORT")
}

func TestRunReportsWorkflowDatabaseFailureAfterBuildingOptions(t *testing.T) {
	failure := errors.New("database locked")
	original := workflowDatabase
	t.Cleanup(func() { workflowDatabase = original })
	workflowDatabase = func() (piko.Option, func(), error) { return nil, nil, failure }
	outbox := t.TempDir()
	t.Setenv("PIKO_FUZZ_PORT", "0")
	t.Setenv("PIKO_FUZZ_CHECKOUT_OUTBOX", outbox)
	t.Setenv("PIKO_FUZZ_SHIPPED", "true")
	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "true")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "true")

	err := run(t.Context())

	require.ErrorIs(t, err, failure)
	assert.ErrorContains(t, err, "opening workflow database")
}

func TestRunReportsMissingShippedComponent(t *testing.T) {
	original := shippedTags
	t.Cleanup(func() { shippedTags = original })
	shippedTags = []string{"piko-counter", "no-such-component"}
	t.Setenv("PIKO_FUZZ_PORT", "0")
	t.Setenv("PIKO_FUZZ_CHECKOUT_OUTBOX", "")
	t.Setenv("PIKO_FUZZ_SHIPPED", "true")

	err := run(t.Context())

	require.ErrorIs(t, err, errMissingShippedComponent)
	assert.ErrorContains(t, err, "no-such-component")
}

func TestReadyReportsUnwritableCoverageDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(directory, []byte("not a directory"), 0o600))
	t.Setenv(envCoverageSecret, "secret")
	t.Setenv(envGoCoverage, directory)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil)
	request.Header.Set(coverageHeader, "secret")
	recorder := httptest.NewRecorder()

	ready(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestRunServesConfiguredHostAndClosesDatabase(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PIKO_FUZZ_PORT", "0")
	t.Setenv("PIKO_FUZZ_CHECKOUT_OUTBOX", "")
	t.Setenv("PIKO_FUZZ_SHIPPED", "")
	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "")
	t.Setenv("PIKO_FUZZ_CATALOG_WATCH", "true")
	t.Setenv(envReadyToken, "host-token")
	t.Setenv(envGoCoverage, "")
	closed := false
	originalDatabase, originalServe := workflowDatabase, serveHost
	t.Cleanup(func() { workflowDatabase, serveHost = originalDatabase, originalServe })
	workflowDatabase = func() (piko.Option, func(), error) {
		return piko.WithHealthEnabled(false), func() { closed = true }, nil
	}
	stopped := errors.New("host stopped")
	var probe *httptest.ResponseRecorder
	serveHost = func(server *piko.SSRServer) error {
		assert.False(t, closed, "database closed before the host stopped")
		probe = httptest.NewRecorder()
		server.AppRouter.ServeHTTP(probe, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/__piko_fuzz_ready", nil))
		return stopped
	}

	err := run(t.Context())

	require.ErrorIs(t, err, stopped)
	assert.True(t, closed, "workflow database left open")
	require.NotNil(t, probe)
	assert.Equal(t, "host-token", probe.Body.String())
}

func TestRunReportsUnopenableCheckoutOutbox(t *testing.T) {
	outbox := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(outbox, []byte("not a directory"), 0o600))
	t.Setenv("PIKO_FUZZ_PORT", "0")
	t.Setenv("PIKO_FUZZ_CHECKOUT_OUTBOX", filepath.Join(outbox, "outbox"))

	err := run(t.Context())

	assert.ErrorContains(t, err, "opening checkout outbox")
}
