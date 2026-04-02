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

package logger_domain

const (
	// KeyTime is the key for the timestamp field in log records.
	KeyTime = "time"

	// KeyLevel is the key used to find the log level in a log record.
	KeyLevel = "level"

	// KeyMessage is the key used to identify the log message field.
	KeyMessage = "msg"

	// KeySource is the key for source location attributes in log entries.
	KeySource = "source"

	// KeyPID is the key for the process ID field in log records.
	KeyPID = "pid"

	// KeyHost is the attribute key for the hostname.
	KeyHost = "host"

	// KeyContext is the attribute key for the package name in log records.
	KeyContext = "ctx"

	// KeyReference is the standard key for a reference ID in log entries.
	KeyReference = "ref"

	// KeyMethod is the log entry key for the method or function name.
	KeyMethod = "mtd"

	// KeyTaskQueue is the key for task queue attributes in log output.
	KeyTaskQueue = "tq"

	// KeyIPAddress is the log entry key for IP address fields.
	KeyIPAddress = "ip"

	// KeyAccountID is the key for user or account ID attributes in logs.
	KeyAccountID = "uid"

	// KeyAttempt is the standard key for retry attempt attributes.
	KeyAttempt = "attempt"

	// KeyError is the standard key for error details in log entries.
	KeyError = "error"

	// KeyRuntimeEnvironment is the key for the detected runtime environment
	// (e.g. "kubernetes", "aws-lambda", "cloud-run").
	KeyRuntimeEnvironment = "runtime.environment"
)
