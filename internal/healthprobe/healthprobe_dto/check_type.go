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

package healthprobe_dto

// CheckType indicates the kind of health check being performed.
type CheckType string

const (
	// CheckTypeLiveness indicates a liveness check, which determines if the
	// application is running and not deadlocked. Failing this check typically
	// results in the application being restarted.
	CheckTypeLiveness CheckType = "liveness"

	// CheckTypeReadiness indicates a readiness check for health probes.
	//
	// A readiness check determines if the application is ready to serve traffic.
	// Failing this check results in traffic being withheld from the application.
	CheckTypeReadiness CheckType = "readiness"
)
