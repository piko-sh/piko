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

	"piko.sh/piko/internal/json"
)

const (
	// releasesURL is the GitHub API endpoint for fetching Piko releases.
	releasesURL = "https://api.github.com/repos/piko-sh/piko/releases?per_page=100"

	// versionRequestTimeout is the maximum time allowed for the version lookup.
	versionRequestTimeout = 10 * time.Second

	// fallbackVersion is used when the latest version cannot be resolved.
	fallbackVersion = "v0.0.0"
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

// resolveLatestVersion queries the GitHub releases API to find the latest
// published version of piko.sh/piko. It prefers stable releases over
// prereleases, falling back to the latest prerelease if no stable release
// exists.
//
// Returns string which is the resolved version tag (e.g. "v0.1.0").
// Returns error when the request fails or no releases are found.
func resolveLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionRequestTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", response.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to decode releases: %w", err)
	}

	return selectVersion(releases)
}

// selectVersion picks the best version from a list of GitHub releases. It
// returns the first stable (non-prerelease, non-draft) release, or falls back
// to the first prerelease if no stable release exists.
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
