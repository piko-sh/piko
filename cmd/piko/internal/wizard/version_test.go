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
