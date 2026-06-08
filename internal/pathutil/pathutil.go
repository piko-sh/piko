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

package pathutil

import (
	"path/filepath"
	"strings"
)

// RelWithin reports whether target is dir itself or nested beneath it.
//
// When contained, it returns the slash-separated path of target relative to dir. Both
// inputs are cleaned, so a trailing separator or unclean segment does not misclassify,
// and comparison is on path segment boundaries, so a sibling whose name only shares a
// prefix with dir (for example "/x/polite" versus "/x/politeperch") is correctly treated
// as outside dir.
//
// Takes dir (string) which is the candidate parent directory.
// Takes target (string) which is the path being tested.
//
// Returns rel (string) which is the slash-separated relative path when contained, else
// "".
// Returns within (bool) which is true when target is dir or nested beneath it.
func RelWithin(dir, target string) (rel string, within bool) {
	if dir == "" {
		return "", false
	}

	relativePath, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(target))
	if err != nil {
		return "", false
	}

	relativePath = filepath.ToSlash(relativePath)
	if relativePath == ".." || strings.HasPrefix(relativePath, "../") {
		return "", false
	}
	return relativePath, true
}

// Contains reports whether target is dir itself or nested beneath it, using the same
// boundary-aware comparison as RelWithin.
//
// Takes dir (string) which is the candidate parent directory.
// Takes target (string) which is the path being tested.
//
// Returns bool which is true when target is dir or nested beneath it.
func Contains(dir, target string) bool {
	_, within := RelWithin(dir, target)
	return within
}
