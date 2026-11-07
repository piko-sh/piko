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
	"runtime/debug"
)

const (
	// develVersion is the version string stamped on an unversioned build. The release
	// pipeline overrides it through ldflags.
	develVersion = "dev"

	// moduleDevelVersion is the main-module version the Go toolchain reports for a build
	// from a local checkout rather than a tagged module download.
	moduleDevelVersion = "(devel)"

	// shortRevisionLength is how many characters of the VCS revision are kept in a dev
	// version string.
	shortRevisionLength = 12
)

var (
	// Version holds the current version of the Piko language server. Override at build
	// time using: go build -ldflags "-X main.Version=1.0.0" ./cmd/pikopls.
	Version = develVersion
)

// resolveVersion returns the version reported by --version and in the LSP serverInfo.
//
// Returns string which is the resolved version.
func resolveVersion() string {
	if Version != develVersion {
		return Version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}
	return versionFromBuildInfo(info)
}

// versionFromBuildInfo derives a version string from the embedded build information.
//
// Takes info (*debug.BuildInfo) which is the build information to inspect, or nil.
//
// Returns string which is the module version, a dev version tagged with the VCS
// revision, or the bare dev version when neither is available.
func versionFromBuildInfo(info *debug.BuildInfo) string {
	if info == nil {
		return develVersion
	}
	if info.Main.Version != "" && info.Main.Version != moduleDevelVersion {
		return info.Main.Version
	}
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
		return develVersion
	}
	if len(revision) > shortRevisionLength {
		revision = revision[:shortRevisionLength]
	}
	if modified == "true" {
		return develVersion + "+" + revision + "-dirty"
	}
	return develVersion + "+" + revision
}
