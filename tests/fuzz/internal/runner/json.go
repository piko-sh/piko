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
	"reflect"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func init() {
	pretouchTypes := []reflect.Type{
		reflect.TypeFor[ActorRead](),
		reflect.TypeFor[CacheRead](),
		reflect.TypeFor[NoticeStatus](),
		reflect.TypeFor[UploadRead](),
		reflect.TypeFor[batchEnvelope](),
		reflect.TypeFor[checkoutLedger](),
		reflect.TypeFor[formActionRow](),
		reflect.TypeFor[noticePayload](),
		reflect.TypeFor[publicationReply](),
		reflect.TypeFor[shimRefusal](),
		reflect.TypeFor[shimStateWire](),
		reflect.TypeFor[topologyAnswer](),
		reflect.TypeFor[uploadRemoval](),
		reflect.TypeFor[casefile.ActionRefusal](),
		reflect.TypeFor[casefile.ActionValue](),
		reflect.TypeFor[casefile.AuthLedger](),
		reflect.TypeFor[casefile.ClientTransition](),
		reflect.TypeFor[casefile.DOMTrace](),
		reflect.TypeFor[casefile.FormShimWrite](),
		reflect.TypeFor[casefile.FormWorkflowObservation](),
	}

	for _, t := range pretouchTypes {
		_ = json.Pretouch(t)
	}
}
