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

package main

import (
	"reflect"

	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func init() {
	pretouchTypes := []reflect.Type{
		reflect.TypeFor[[]casefile.ClientAction](),
		reflect.TypeFor[casefile.CheckoutInput](),
		reflect.TypeFor[casefile.PartialComponentsInput](),
		reflect.TypeFor[casefile.EventsInput](),
		reflect.TypeFor[casefile.LifecycleInput](),
		reflect.TypeFor[casefile.DOMPlan](),
		reflect.TypeFor[casefile.DOMState](),
		reflect.TypeFor[casefile.FormWorkflow](),
		reflect.TypeFor[[]casefile.FormRow](),
		reflect.TypeFor[checkoutPublicError](),
	}
	for _, pretouchType := range pretouchTypes {
		_ = json.Pretouch(pretouchType)
	}
}
