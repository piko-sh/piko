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

package cache_domain

import (
	"reflect"

	"piko.sh/piko/internal/json"
)

// CacheAPI is a JSON configuration for long-lived decoded objects that will
// be stored and reused. It copies strings instead of referencing the source
// buffer, preventing memory issues when objects outlive the original JSON data.
var CacheAPI = json.Freeze(json.Config{
	CopyString: true,
	UseInt64:   true,
	EscapeHTML: false,
})

func init() {
	pretouchTypes := []reflect.Type{
		reflect.TypeFor[TransformedValue](),
	}

	for _, t := range pretouchTypes {
		_ = json.Pretouch(t)
	}
}
