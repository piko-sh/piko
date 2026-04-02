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

package ast_domain

// Provides OpenTelemetry integration for the ast_domain package with logging and metrics instrumentation.
// Initialises package-level logger and meter instances for observability and performance monitoring throughout AST operations.

import (
	"go.opentelemetry.io/otel"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	log = logger_domain.GetLogger("piko/internal/ast/ast_domain")

	// Meter provides OpenTelemetry metrics for the ast_domain package.
	Meter = otel.Meter("piko/internal/ast/ast_domain")
)
