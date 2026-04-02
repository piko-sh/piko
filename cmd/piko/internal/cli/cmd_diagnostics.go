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

package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/credentials"
	"piko.sh/piko/cmd/piko/internal/tui"
)

// diagnosticServiceResult represents a single service check result for JSON
// output.
type diagnosticServiceResult struct {
	// Name is the identifier of the diagnostic.
	Name string `json:"name"`

	// Method is the HTTP method used for the request.
	Method string `json:"method"`

	// Details contains extra information about the diagnostic result.
	Details string `json:"details,omitempty"`

	// Error contains the error message if this result failed; empty on success.
	Error string `json:"error,omitempty"`

	// OK indicates whether the diagnostic check passed.
	OK bool `json:"ok"`
}

// diagnosticJSONResult is the top-level JSON structure for diagnostics output.
type diagnosticJSONResult struct {
	// Endpoint is the API endpoint path associated with the diagnostic.
	Endpoint string `json:"endpoint"`

	// Error contains the error message if the diagnostic failed; empty on success.
	Error string `json:"error,omitempty"`

	// Services contains the diagnostic results for each analysed service.
	Services []diagnosticServiceResult `json:"services"`

	// Passed is the number of diagnostics that passed without error.
	Passed int `json:"passed"`

	// Failed is the count of diagnostics that could not be processed.
	Failed int `json:"failed"`

	// Connected indicates whether a connection to the target is active.
	Connected bool `json:"connected"`
}

// runDiagnosticsCmd tests connectivity to the gRPC monitoring server.
//
// Takes ctx (context.Context) which controls the deadline for the
// diagnostic checks.
// Takes cc (*CommandContext) which provides output settings and the
// target endpoint.
//
// Returns error when diagnostics fail or output formatting fails.
func runDiagnosticsCmd(ctx context.Context, cc *CommandContext, _ []string) error {
	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, false)

	var creds credentials.TransportCredentials
	if cc.Opts.CertsDir != "" {
		var err error
		creds, err = loadTLSCredentials(cc.Factory, cc.Opts.CertsDir)
		if err != nil {
			return fmt.Errorf("loading TLS credentials from %s: %w", cc.Opts.CertsDir, err)
		}
	}

	result := tui.RunDiagnostics(ctx, cc.Opts.Endpoint, creds)
	return formatDiagnosticResult(result, p, cc.Stdout)
}

// formatDiagnosticResult formats a diagnostic result as JSON or text.
//
// Takes result (*tui.DiagnosticsResult) which contains the
// diagnostic checks.
// Takes p (*Printer) which controls the output format.
// Takes stdout (io.Writer) which receives text output.
//
// Returns error when JSON marshalling fails or diagnostics did not all pass.
func formatDiagnosticResult(result *tui.DiagnosticsResult, p *Printer, stdout io.Writer) error {
	if p.IsJSON() {
		jr := diagnosticJSONResult{
			Endpoint:  result.Endpoint,
			Connected: result.Connected,
			Passed:    result.Passed,
			Failed:    result.Failed,
		}
		if result.ConnectionError != nil {
			jr.Error = result.ConnectionError.Error()
		}
		for _, s := range result.Services {
			service := diagnosticServiceResult{
				Name:    s.Name,
				Method:  s.Method,
				OK:      s.OK,
				Details: s.Details,
			}
			if s.Error != nil {
				service.Error = s.Error.Error()
			}
			jr.Services = append(jr.Services, service)
		}
		return p.PrintJSON(jr)
	}

	result.Print(stdout)

	if !result.AllPassed() {
		return errors.New("diagnostics failed")
	}
	return nil
}
