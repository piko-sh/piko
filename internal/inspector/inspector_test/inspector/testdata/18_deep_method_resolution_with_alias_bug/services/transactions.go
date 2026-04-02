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

package services

// This is the critical file.
// It imports the 'dtos' package using an alias.
import dto_alias "testcase_18_deep_method_resolution_with_alias_bug/dtos"

// This struct uses the aliased import. The inspector must correctly
// handle this when resolving fields on this type from another package's context.
type TransactionServiceResponse struct {
	Transaction dto_alias.TransactionDto
}
