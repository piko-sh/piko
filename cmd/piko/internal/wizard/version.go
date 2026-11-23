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
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"piko.sh/json"
)

const (
	// releasesURL is the GitHub API endpoint for fetching Piko releases.
	releasesURL = "https://api.github.com/repos/piko-sh/piko/releases?per_page=100"

	// versionRequestTimeout is the maximum time allowed for the version lookup.
	versionRequestTimeout = 10 * time.Second

	// fallbackVersion is used when the latest version cannot be resolved.
	fallbackVersion = "v0.0.0"

	// maxReleasesResponseBytes limits the size of the GitHub release list.
	maxReleasesResponseBytes = 4 << 20
)

var (
	// errReleasesResponseTooLarge reports a release list exceeding the size limit.
	errReleasesResponseTooLarge = fmt.Errorf("github releases response exceeds %d bytes", maxReleasesResponseBytes)
)

// githubRelease holds the fields we need from the GitHub releases API.
type githubRelease struct {
	// TagName is the Git tag for the release (e.g. "v0.1.0").
	TagName string `json:"tag_name"`

	// Draft indicates whether the release is still a draft.
	Draft bool `json:"draft"`

	// Prerelease indicates whether the release is marked as a prerelease.
	Prerelease bool `json:"prerelease"`
}

// resolveLatestVersion selects the latest available Piko release from GitHub.
//
// Returns string which is the stable release tag, or a prerelease tag if necessary.
// Returns error when the release list cannot be fetched or has no usable release.
func resolveLatestVersion() (string, error) {
	return fetchLatestVersion(http.DefaultClient, releasesURL)
}

// fetchLatestVersion finds the first stable release from the endpoint.
//
// If the list has no stable release, it uses the first tagged prerelease.
//
// Takes client (*http.Client) which sends the request.
// Takes url (string) which is the releases endpoint to query.
//
// Returns string which is the resolved version tag.
// Returns error when the request fails, the response is unsuccessful, oversized or
// malformed, or the list has no usable release.
func fetchLatestVersion(client *http.Client, url string) (string, error) {
	ctx, cancel := context.WithTimeoutCause(
		context.Background(),
		versionRequestTimeout,
		errors.New("github releases lookup timed out"),
	)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxReleasesResponseBytes))
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxReleasesResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read releases: %w", err)
	}
	if len(body) > maxReleasesResponseBytes {
		return "", errReleasesResponseTooLarge
	}

	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("failed to decode releases: %w", err)
	}

	return selectVersion(releases)
}

// selectVersion picks the best version from a list of GitHub releases. It returns the
// first stable (non-prerelease, non-draft) release, or falls back to the first prerelease
// if no stable release exists.
//
// Takes releases ([]githubRelease) which is the list of releases to search.
//
// Returns string which is the selected version tag.
// Returns error when no suitable releases are found.
func selectVersion(releases []githubRelease) (string, error) {
	var latestPrerelease string

	for _, release := range releases {
		if release.Draft || release.TagName == "" {
			continue
		}

		if !release.Prerelease {
			return release.TagName, nil
		}

		if latestPrerelease == "" {
			latestPrerelease = release.TagName
		}
	}

	if latestPrerelease != "" {
		return latestPrerelease, nil
	}

	return "", errors.New("no releases found")
}
