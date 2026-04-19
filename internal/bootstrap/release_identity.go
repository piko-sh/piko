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

package bootstrap

import (
	"cmp"
	"runtime/debug"
	"strings"
)

const (
	// unversionedReleaseID is the release identifier stamped on build variants when a build
	// carries neither an explicit WithReleaseID override nor a VCS revision.
	unversionedReleaseID = "unversioned"
)

// buildReleaseIdentity returns the release identifier and full build hash.
//
// Takes override (string) which is the explicit release identifier, or empty for none.
//
// Returns release which is the resolved release identifier.
// Returns hash which is the full build hash.
func buildReleaseIdentity(override string) (release, hash string) {
	if info, ok := debug.ReadBuildInfo(); ok {
		hash = appBuildHashFromBuildInfo(info)
	}
	revision := hash
	if index := strings.IndexByte(revision, ':'); index >= 0 {
		revision = revision[:index]
	}
	return cmp.Or(override, revision, unversionedReleaseID), hash
}

// appBuildHashFromBuildInfo returns "<vcs.revision>:<vcs.modified>" from the build
// settings, or "" when no VCS stamp is available (e.g. a bare `go build` of an untracked
// tree).
//
// Equal hashes mean an identical build.
//
// Takes info (*debug.BuildInfo) which is the build info to inspect.
//
// Returns string which is the build hash, or empty when no VCS stamp is present.
func appBuildHashFromBuildInfo(info *debug.BuildInfo) string {
	var revision, modified string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}
	if revision == "" {
		return ""
	}
	return revision + ":" + modified
}
