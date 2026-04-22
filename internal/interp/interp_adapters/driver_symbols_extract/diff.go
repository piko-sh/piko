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

package driver_symbols_extract

import "slices"

// ManifestDiff summarises the difference between what a manifest
// declares and what a project actually requires. A green check run
// should report empty Missing; Unused entries are informational.
type ManifestDiff struct {
	// Missing lists import paths that discovery found but the
	// manifest does not declare. Non-empty means the interpreter
	// will fail to resolve these packages.
	Missing []string

	// Unused lists import paths that the manifest declares but
	// discovery did not find in the project. Keeping them is
	// harmless but they add unnecessary registration overhead.
	Unused []string
}

// Diff computes the symmetric difference between a loaded manifest
// and a discovery result. Both input slices are treated as sets;
// output slices are sorted for determinism.
//
// Takes manifest (*Manifest) which supplies the declared packages.
// Takes discovered (DiscoverResult) which supplies the required set.
//
// Returns a ManifestDiff describing missing and unused entries.
func Diff(manifest *Manifest, discovered DiscoverResult) ManifestDiff {
	declared := make(map[string]struct{})
	if manifest != nil {
		for _, config := range manifest.Packages {
			if config.ImportPath != "" {
				declared[config.ImportPath] = struct{}{}
			}
		}
	}

	required := make(map[string]struct{}, len(discovered.RequiredImports))
	for _, path := range discovered.RequiredImports {
		required[path] = struct{}{}
	}

	var missing []string
	for path := range required {
		if _, ok := declared[path]; !ok {
			missing = append(missing, path)
		}
	}
	var unused []string
	for path := range declared {
		if _, ok := required[path]; !ok {
			unused = append(unused, path)
		}
	}

	slices.Sort(missing)
	slices.Sort(unused)

	return ManifestDiff{
		Missing: missing,
		Unused:  unused,
	}
}
