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

package runner

import (
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// compareReload identifies the first differing reload-state field.
//
// Takes got (*casefile.PartialReloadState) which holds the observed result.
// Takes want (*casefile.PartialReloadState) which holds the independently expected
// result.
//
// Returns string which identifies the first differing reload field, or is empty when all
// fields agree.
func compareReload(got, want *casefile.PartialReloadState) string {
	if got == nil || want == nil {
		if got != want {
			return "reload-observation"
		}
		return ""
	}
	switch {
	case got.Result != want.Result:
		return "reload-result"
	case got.RootStage != want.RootStage:
		return "root-attribute"
	case got.ChildStage != want.ChildStage || got.Default != want.Default:
		return "child-content"
	case got.LiveOnly != want.LiveOnly || got.Preserved != want.Preserved:
		return "root-ownership"
	case !got.RootRetained || !got.InputRetained || !got.Slotted:
		return "owner-identity"
	case got.Draft != want.Draft || !got.Focused || got.Selection != want.Selection:
		return "focused-draft"
	case !got.SiblingRetained || !got.SiblingUntouched:
		return "sibling"
	default:
		return ""
	}
}
