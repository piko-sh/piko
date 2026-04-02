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
	// The primary UUID from Google, imported with its default name.
	"github.com/google/uuid"

	// A second external UUID package, aliased.
	gofrsuuid "github.com/gofrs/uuid/v5"

	// A third conflicting UUID from a transitive-like dependency, aliased.
	moderncuuid "modernc.org/libc/uuid/uuid"

	// Our own local package named "uuid", aliased.
	localuuid "testcase_37_multi_package_name_collision/pkg/uuid"
)

// Response uses all four conflicting types to test the serialiser's context-awareness.
type Response struct {
	// Should resolve to github.com/google/uuid
	RealGoogleUUID uuid.UUID

	// Should resolve to testcase_37.../pkg/uuid
	LocalFakeUUID localuuid.UUID

	// Should resolve to github.com/gofrs/uuid/v5
	GofrsUUID gofrsuuid.UUID

	// Should resolve to modernc.org/libc/uuid/uuid
	ModerncUUID moderncuuid.Uuid_t
}
