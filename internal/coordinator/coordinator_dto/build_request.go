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

package coordinator_dto

import (
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// BuildRequest encapsulates all the inputs for a full, unique, and
// deterministic project build. It includes the source file entry points
// and any external context that could influence the final compiled output.
//
// Actions are auto-discovered from the actions/ directory during annotation.
type BuildRequest struct {
	// Resolver provides path resolution for this build. When set, it overrides the
	// coordinator's default resolver, enabling per-module resolution in LSP
	// contexts where files from different Go modules may be analysed.
	Resolver resolver_domain.ResolverPort

	// CausationID is the identifier used to trace the cause of status updates.
	CausationID string

	// EntryPoints lists the files where the build should start.
	EntryPoints []annotator_dto.EntryPoint

	// FaultTolerant enables processing to continue when errors occur.
	FaultTolerant bool
}
