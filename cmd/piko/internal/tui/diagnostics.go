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

package tui

import (
	"context"

	"google.golang.org/grpc/credentials"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
)

// Re-export diagnostics types from domain.
type (
	// DiagnosticsResult holds the results of running diagnostics.
	DiagnosticsResult = tui_domain.DiagnosticsResult

	// ServiceResult holds the result for a single service test.
	ServiceResult = tui_domain.ServiceResult
)

// RunDiagnostics tests connectivity to a gRPC monitoring endpoint and
// all services. It returns a structured result that can be formatted
// for output.
//
// Example:
//
//	result := tui.RunDiagnostics(ctx, "127.0.0.1:9091")
//	result.Print(os.Stdout)
//	if !result.AllPassed() {
//	    os.Exit(1)
//	}
//
// Takes endpoint (string) which is the gRPC server address
// (e.g., "127.0.0.1:9091"). If empty, defaults to
// "127.0.0.1:9091".
// Takes creds (credentials.TransportCredentials) which is optional TLS
// credentials; nil uses insecure credentials.
//
// Returns *DiagnosticsResult with the test results.
func RunDiagnostics(ctx context.Context, endpoint string, creds credentials.TransportCredentials) *DiagnosticsResult {
	return tui_domain.RunDiagnostics(ctx, endpoint, creds)
}
