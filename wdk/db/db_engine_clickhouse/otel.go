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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// log is the package-level logger for the ClickHouse engine. Used to surface a recovered
	// parser panic at warn level (with the full stack) while the returned error stays free
	// of internal symbol paths that have no value to the webdev consuming the diagnostic.
	log = logger_domain.GetLogger("piko/wdk/db/db_engine_clickhouse")
)
