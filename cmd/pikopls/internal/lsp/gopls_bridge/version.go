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

package gopls_bridge

import (
	"strconv"
	"strings"
)

const (
	// minGoplsMajor is the lowest gopls major version the bridge drives.
	minGoplsMajor = 0

	// minGoplsMinor is the lowest gopls minor version the bridge drives, paired with
	// minGoplsMajor.
	minGoplsMinor = 12

	// versionFieldParts is the major.minor.patch split width used when parsing a gopls
	// version string; only the leading major and minor are compared.
	versionFieldParts = 3
)

// goplsVersionSupported reports whether a gopls serverInfo version string meets the
// bridge's floor. An unparseable version is treated as supported, so a locally built or
// unusually formatted gopls is never wrongly refused.
//
// Takes version (string) which is the gopls serverInfo version to check.
//
// Returns bool which is true when the version meets the floor or cannot be parsed.
func goplsVersionSupported(version string) bool {
	major, minor, ok := parseGoplsVersion(version)
	if !ok {
		return true
	}
	if major != minGoplsMajor {
		return major > minGoplsMajor
	}
	return minor >= minGoplsMinor
}

// parseGoplsVersion extracts the leading major.minor from a gopls version string such as
// "v0.21.0", "0.21.0" or "golang.org/x/tools/gopls v0.21.0".
//
// Takes version (string) which is the gopls version string to parse.
//
// Returns major (int) which is the parsed major component.
// Returns minor (int) which is the parsed minor component.
// Returns ok (bool) which is false when no numeric version can be found.
func parseGoplsVersion(version string) (major, minor int, ok bool) {
	for field := range strings.FieldsSeq(version) {
		candidate := strings.TrimPrefix(field, "v")
		parts := strings.SplitN(candidate, ".", versionFieldParts)
		if len(parts) < 2 {
			continue
		}
		parsedMajor, majorErr := strconv.Atoi(parts[0])
		parsedMinor, minorErr := strconv.Atoi(parts[1])

		if majorErr != nil || minorErr != nil || parsedMajor < 0 || parsedMinor < 0 {
			continue
		}
		return parsedMajor, parsedMinor, true
	}
	return 0, 0, false
}
