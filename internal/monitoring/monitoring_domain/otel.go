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

package monitoring_domain

import (
	"piko.sh/piko/wdk/logger"
)

var (
	log = logger.GetLogger("piko/internal/monitoring/monitoring_domain")

	// String is a logger field constructor for string values.
	String = logger.String

	// Int is a convenience alias for logger.Int to create integer log fields.
	Int = logger.Int

	// Int64 is an alias for logger.Int64 for logging int64 values.
	Int64 = logger.Int64

	// Error is the package-level error logging function.
	Error = logger.Error
)
