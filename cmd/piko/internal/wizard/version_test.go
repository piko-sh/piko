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

package wizard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectVersion_PrefersStable(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{TagName: "v0.0.0-alpha.6", Prerelease: true},
		{TagName: "v0.1.0", Prerelease: false},
		{TagName: "v0.0.0-alpha.5", Prerelease: true},
	}

	version, err := selectVersion(releases)
	require.NoErrorf(t, err, "selectVersion() error = %v", err)
	assert.Equalf(t, "v0.1.0", version, "selectVersion() = %q, want %q", version, "v0.1.0")
}

func TestSelectVersion_FallsBackToPrerelease(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{TagName: "v0.0.0-alpha.6", Prerelease: true},
		{TagName: "v0.0.0-alpha.5", Prerelease: true},
	}

	version, err := selectVersion(releases)
	require.NoErrorf(t, err, "selectVersion() error = %v", err)
	assert.Equalf(t, "v0.0.0-alpha.6", version, "selectVersion() = %q, want %q", version, "v0.0.0-alpha.6")
}

func TestSelectVersion_SkipsDrafts(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{TagName: "v0.2.0", Draft: true},
		{TagName: "v0.1.0", Prerelease: false},
	}

	version, err := selectVersion(releases)
	require.NoErrorf(t, err, "selectVersion() error = %v", err)
	assert.Equalf(t, "v0.1.0", version, "selectVersion() = %q, want %q", version, "v0.1.0")
}

func TestSelectVersion_SkipsEmptyTagName(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{TagName: ""},
		{TagName: "v0.0.0-alpha.1", Prerelease: true},
	}

	version, err := selectVersion(releases)
	require.NoErrorf(t, err, "selectVersion() error = %v", err)
	assert.Equalf(t, "v0.0.0-alpha.1", version, "selectVersion() = %q, want %q", version, "v0.0.0-alpha.1")
}

func TestSelectVersion_NoReleases(t *testing.T) {
	t.Parallel()

	_, err := selectVersion(nil)
	assert.Error(t, err, "selectVersion(nil) should return error")
}

func TestSelectVersion_OnlyDrafts(t *testing.T) {
	t.Parallel()

	releases := []githubRelease{
		{TagName: "v0.1.0", Draft: true},
	}

	_, err := selectVersion(releases)
	assert.Error(t, err, "selectVersion() should return error when only drafts exist")
}

func TestFetchLatestVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		body        string
		wantVersion string
		wantErr     string
		status      int
	}{
		{
			name:        "the first stable release",
			status:      http.StatusOK,
			body:        `[{"tag_name":"v0.3.0-rc1","prerelease":true},{"tag_name":"v0.2.0"}]`,
			wantVersion: "v0.2.0",
		},
		{name: "a non-OK status", status: http.StatusForbidden, body: `{}`, wantErr: "GitHub API returned status 403"},
		{name: "an undecodable body", status: http.StatusOK, body: `{"tag_name":`, wantErr: "failed to decode releases"},
		{name: "no releases", status: http.StatusOK, body: `[]`, wantErr: "no releases found"},
		{name: "an oversized body", status: http.StatusOK, body: "[" + strings.Repeat(" ", maxReleasesResponseBytes) + "]", wantErr: "exceeds"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "application/vnd.github+json", request.Header.Get("Accept"))
				writer.WriteHeader(testCase.status)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)

			version, err := fetchLatestVersion(server.Client(), server.URL)

			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.wantVersion, version)
		})
	}
}

func TestFetchLatestVersionReportsTransportFailures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	address := server.URL
	server.Close()

	_, err := fetchLatestVersion(http.DefaultClient, address)
	require.ErrorContains(t, err, "failed to fetch releases")

	_, err = fetchLatestVersion(http.DefaultClient, "http://bad host/")
	require.ErrorContains(t, err, "failed to create request")
}
