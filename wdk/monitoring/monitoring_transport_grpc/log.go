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

package monitoring_transport_grpc

import (
	"piko.sh/piko/wdk/logger"
)

const (
	// defaultListLimit is the default number of items returned by list queries.
	defaultListLimit = 10

	// defaultSpanLimit is the default number of spans returned in trace queries.
	defaultSpanLimit = 100

	// minWatchIntervalMs is the smallest allowed interval for watch streams.
	minWatchIntervalMs = 100
)

var (
	// log is the package-level logger for the monitoring_transport_grpc package.
	log = logger.GetLogger("piko/wdk/monitoring/monitoring_transport_grpc")

	// String is an alias for logger.String that creates a string field for logging.
	String = logger.String

	// Int is a field constructor for integer values in structured log entries.
	Int = logger.Int

	// Int64 is an alias for logger.Int64 that creates a log field from an int64
	// value.
	Int64 = logger.Int64

	// Error logs a message at error level using the default logger.
	Error = logger.Error
)
