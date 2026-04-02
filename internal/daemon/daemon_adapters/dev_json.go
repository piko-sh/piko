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

package daemon_adapters

import (
	"reflect"

	pikojson "piko.sh/piko/internal/json"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

// devJSON is a JSON encoder configured to sort map keys so that SSE and REST
// responses from the dev tools have deterministic field ordering.
var devJSON = pikojson.Freeze(pikojson.Config{SortMapKeys: true})

func init() {
	pretouchTypes := []reflect.Type{
		reflect.TypeFor[monitoring_domain.SystemStats](),
		reflect.TypeFor[monitoring_domain.MemoryInfo](),
		reflect.TypeFor[monitoring_domain.GCInfo](),
		reflect.TypeFor[monitoring_domain.ProcessInfo](),
		reflect.TypeFor[monitoring_domain.RuntimeInfo](),
		reflect.TypeFor[monitoring_domain.BuildInfo](),
		reflect.TypeFor[monitoring_domain.ResourceData](),
		reflect.TypeFor[monitoring_domain.ResourceCategory](),
		reflect.TypeFor[orchestrator_domain.TaskSummary](),
		reflect.TypeFor[orchestrator_domain.TaskListItem](),
		reflect.TypeFor[provider_domain.ProviderDetail](),
		reflect.TypeFor[monitoring_domain.ProviderListResult](),
		reflect.TypeFor[DevBuildEvent](),
	}

	for _, t := range pretouchTypes {
		_ = pikojson.Pretouch(t)
	}
}
