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

package db_emitter_pgx

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// EmitPrepared returns an empty generated file. pgx's connection pool handles
// implicit prepared statement caching per connection, so no explicit
// PreparedDBTX wrapper is needed.
//
// Takes packageName (string) which is the target package name.
// Takes queries ([]*querier_dto.AnalysedQuery) which are ignored.
//
// Returns querier_dto.GeneratedFile with empty content.
// Returns error which is always nil.
func (*PgxEmitter) EmitPrepared(
	_ string,
	_ []*querier_dto.AnalysedQuery,
) (querier_dto.GeneratedFile, error) {
	return querier_dto.GeneratedFile{}, nil
}
