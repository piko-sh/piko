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

package coordinator_domain

import (
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
)

// state represents the current status of the coordinator service.
// It implements fmt.Stringer for readable output.
type state int

const (
	// stateIdle indicates the coordinator has not yet performed a build or has
	// been invalidated.
	stateIdle state = iota

	// stateBuilding indicates that a build is currently in progress.
	stateBuilding

	// stateReady indicates that a successful build result is available.
	stateReady

	// stateFailed indicates that the last attempted build resulted in an error.
	stateFailed
)

// String returns the name of the state as a readable string.
//
// Returns string which is the state name (e.g. "Idle", "Building", "Ready").
func (s state) String() string {
	switch s {
	case stateIdle:
		return "Idle"
	case stateBuilding:
		return "Building"
	case stateReady:
		return "Ready"
	case stateFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// buildStatus is a snapshot of the coordinator's state at a single point in
// time. It is the primary DTO for consumers who need the project build status.
type buildStatus struct {
	// LastBuildTime is when the build status was last updated.
	LastBuildTime time.Time

	// LastBuildError is the error from the most recent build; nil if successful.
	LastBuildError error

	// Result holds the annotation output from the build; nil if the build failed.
	Result *annotator_dto.ProjectAnnotationResult

	// State indicates the current build state.
	State state
}

// BuildNotification is the event payload sent to all subscribers when a build
// completes.
type BuildNotification struct {
	// Result holds the build output; nil if the build failed.
	Result *annotator_dto.ProjectAnnotationResult

	// CausationID is the ID of the event that caused this build notification.
	CausationID string
}
