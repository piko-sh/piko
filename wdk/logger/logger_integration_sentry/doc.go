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

// Package logger_integration_sentry provides Sentry error tracking
// integration for Piko's logging system.
//
// Importing the package registers Sentry as a logging integration so that
// errors are reported automatically, breadcrumbs are captured, and
// OpenTelemetry traces propagate via the Sentry SDK. Enable it
// programmatically with [Enable] from your func main.
//
// The package also provides OpenTelemetry span processing through Sentry's
// OTel bridge, so distributed traces and log events are correlated in the
// Sentry dashboard. [Enable] uses [sync.Once] and is safe for concurrent use.
package logger_integration_sentry
