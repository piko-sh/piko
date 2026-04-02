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
	"testproject_uuid_nightmare/pkg/models"

	// The real UUID from Google that we want to use, imported without an alias.
	"github.com/google/uuid"

	// A second, different external UUID package, with an alias.
	gofrsuuid "github.com/gofrs/uuid/v5"

	// The conflicting UUID from modernc, with an alias.
	moderncuuid "modernc.org/libc/uuid/uuid"

	// Our own local package named "uuid", with an alias.
	localuuid "testproject_uuid_nightmare/pkg/uuid"
)

// Response uses all four conflicting types.
type Response struct {
	// This is the one that should have the .String() method.
	// Its type comes from the un-aliased 'uuid' import.
	RealGoogleUUID uuid.UUID

	// This is our fake local type.
	LocalFakeUUID localuuid.UUID

	// This is the type from the second external dependency.
	GofrsUUID gofrsuuid.UUID

	// This is the type from the transitive dependency.
	ModerncUUID moderncuuid.Uuid_t

	ModelA models.ModelA
	ModelB models.ModelB
	ModelC models.ModelC
	ModelD models.ModelD
	ModelE models.ModelE
}

// MethodToTest forces the inspector to resolve the method call on the un-aliased import.
func (r *Response) MethodToTest() string {
	return r.RealGoogleUUID.String()
}
