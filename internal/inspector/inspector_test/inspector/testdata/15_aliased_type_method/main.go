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

import "github.com/google/uuid"

// This is a TYPE DEFINITION.
// MyUUID is a new, distinct type. It does NOT inherit methods from uuid.UUID.
// The inspector should NOT find the .String() method on this type.
type MyUUID uuid.UUID

// This is a TYPE ALIAS.
// MyUUIDAlias is just another name for uuid.UUID. It shares the same method set.
// The inspector SHOULD find the .String() method on this type.
type MyUUIDAlias = uuid.UUID

// Response uses both the defined type and the aliased type.
type Response struct {
	// A field using the new type definition.
	DefinedUUID MyUUID

	// A field using the type alias.
	AliasedUUID MyUUIDAlias
}
