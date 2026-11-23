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

package process

import (
	"os"
	"path/filepath"

	"piko.sh/piko/internal/testutil/pipitpending"
)

const (
	// overlayWorkspace is the untracked workspace that adds the local Pipit checkout to
	// the committed repository workspace.
	overlayWorkspace = "go.fuzz.work"

	// repositoryWorkspace is the committed repository workspace.
	repositoryWorkspace = "go.work"
)

// Workspace resolves the workspace file that staged applications inherit.
//
// An absolute GOWORK takes precedence because workers run in separate directories.
// Without one, PIKO_PIPIT_LOCAL=1 selects the local Pipit overlay and the default is
// the committed go.work.
//
// Takes repo (string) which is the Piko checkout root.
//
// Returns string which is the workspace file path.
func Workspace(repo string) string {
	if work := os.Getenv("GOWORK"); filepath.IsAbs(work) {
		return work
	}
	if pipitpending.Local() {
		return filepath.Join(repo, overlayWorkspace)
	}
	return filepath.Join(repo, repositoryWorkspace)
}
