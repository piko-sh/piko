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

package daemon_domain

import (
	"context"
	"time"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// mainListenerProbeName is the component name the probe reports under.
	mainListenerProbeName = "MainHTTPListener"
)

// MainListenerProbe reports whether the main HTTP listener is accepting connections.
type MainListenerProbe struct {
	// bound closes once the main listener is accepting connections, and never closes when
	// the bind failed.
	bound <-chan struct{}

	// clock supplies the timestamp on each status.
	clock clock.Clock
}

// NewMainListenerProbe creates a probe reporting on the main HTTP listener.
//
// Takes bound (<-chan struct{}) which closes when the listener binds.
// Takes serviceClock (clock.Clock) which stamps each status.
//
// Returns *MainListenerProbe which is ready to register.
func NewMainListenerProbe(bound <-chan struct{}, serviceClock clock.Clock) *MainListenerProbe {
	return &MainListenerProbe{bound: bound, clock: serviceClock}
}

// Name returns the component name this probe reports under.
//
// Returns string which names the component.
func (*MainListenerProbe) Name() string {
	return mainListenerProbeName
}

// Check reports the listener's state.
//
// Takes checkType (healthprobe_dto.CheckType) which selects liveness or readiness.
//
// Returns healthprobe_dto.Status which describes the listener.
func (p *MainListenerProbe) Check(
	_ context.Context,
	checkType healthprobe_dto.CheckType,
) healthprobe_dto.Status {
	status := healthprobe_dto.Status{
		Name:      mainListenerProbeName,
		State:     healthprobe_dto.StateHealthy,
		Timestamp: p.now(),
	}

	if checkType != healthprobe_dto.CheckTypeReadiness {
		return status
	}

	select {
	case <-p.bound:
		status.Message = "the main HTTP listener is accepting connections"
	default:
		status.State = healthprobe_dto.StateUnhealthy
		status.Message = "the main HTTP listener has not bound yet"
	}

	return status
}

// now reads the current time, falling back to the wall clock when none was supplied.
//
// Returns time.Time which stamps the status.
func (p *MainListenerProbe) now() time.Time {
	if p.clock == nil {
		return time.Now()
	}

	return p.clock.Now()
}
