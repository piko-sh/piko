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

package db_engine_mysql

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// tableValuedFunctionColumns maps MySQL table-valued function names to their
// output column definitions. MySQL's primary table-valued function is
// JSON_TABLE (MySQL 8.0+), whose columns are defined by the user's COLUMNS
// clause rather than being fixed - the DML analyser handles those
// user-specified column definitions via the AS clause. This map is
// intentionally empty for now.
var tableValuedFunctionColumns = map[string][]querier_dto.ScopedColumn{}
