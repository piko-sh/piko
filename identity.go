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

package piko

import (
	"piko.sh/piko/internal/bootstrap"
)

// DeploymentIdentity describes the running process: which build it is, where it runs, and
// how to distinguish it from a sibling replica.
type DeploymentIdentity = bootstrap.DeploymentIdentity

// Identity returns this process's DeploymentIdentity, resolved once on first call from
// the process environment (the Kubernetes Downward API, AWS Lambda, Cloud Run and Azure
// Container Apps, plus the PIKO_* overrides) and the Go build info.
//
// Returns DeploymentIdentity which describes the running process.
func Identity() DeploymentIdentity { return bootstrap.Identity() }
