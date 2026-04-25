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

package tui_domain

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
	"piko.sh/piko/cmd/piko/internal/tui/tui_dto"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

const (
	// serviceHealth is the key for health check results in diagnostics output.
	serviceHealth = "HealthService"

	// serviceMetrics is the name for the metrics service used in diagnostics.
	serviceMetrics = "MetricsService"

	// serviceOrchestrator is the service name for orchestrator diagnostics.
	serviceOrchestrator = "OrchestratorInspectorService"

	// serviceRegistry is the name of the registry inspection service.
	serviceRegistry = "RegistryInspectorService"

	// serviceProfiling is the name of the profiling service.
	serviceProfiling = "ProfilingService"

	// serviceWatchdog is the name of the watchdog inspector service.
	serviceWatchdog = "WatchdogInspectorService"

	// resultFailed is the status text shown when a diagnostic check fails.
	resultFailed = "failed"

	// resultSkipped is the status text shown when an opt-in service is not
	// registered on the server.
	resultSkipped = "not registered (opt-in)"

	// diagnosticsTimeout is the time allowed for diagnostics checks to complete.
	diagnosticsTimeout = 15 * time.Second

	// diagnosticsLimit is the maximum number of items to fetch from each list query
	// when running diagnostics.
	diagnosticsLimit = 5
)

// DiagnosticsResult holds the results of running checks against services.
type DiagnosticsResult struct {
	// ConnectionError holds any error that occurred when connecting to the endpoint.
	ConnectionError error

	// Endpoint is the gRPC address that was tested.
	Endpoint string

	// Services holds the results of each service check.
	Services []ServiceResult

	// Passed is the number of tests that passed.
	Passed int

	// Failed is the count of tests that did not pass.
	Failed int

	// Skipped is the number of tests skipped because the service is not
	// registered (opt-in services).
	Skipped int

	// Connected indicates whether a connection to the server was established.
	Connected bool
}

// ServiceResult holds the outcome of a single service test.
type ServiceResult struct {
	// Error holds any error from the service check; nil means success.
	Error error

	// Name is the service identifier used for grouping results.
	Name string

	// Method is the name of the RPC method that was tested.
	Method string

	// Details is extra information about the result, shown when the check passes.
	Details string

	// OK indicates whether the service check passed.
	OK bool

	// IsSkipped indicates the check was skipped because the service is not
	// registered on the server (opt-in services).
	IsSkipped bool
}

// AllPassed returns true if all tests passed.
//
// Returns bool which is true when the connection succeeded and no tests failed.
func (r *DiagnosticsResult) AllPassed() bool {
	return r.Connected && r.Failed == 0
}

// Print writes a formatted diagnostics report to the given writer.
//
// Takes w (io.Writer) which receives the formatted output.
func (r *DiagnosticsResult) Print(w io.Writer) {
	_, _ = fmt.Fprint(w, "=== TUI Diagnostics ===\n")
	_, _ = fmt.Fprintf(w, "Monitoring Endpoint: %s\n\n", r.Endpoint)

	_, _ = fmt.Fprint(w, "--- Connection ---\n")
	if !r.Connected {
		r.printConnectionFailure(w)
		return
	}

	_, _ = fmt.Fprint(w, "  Status: Connected\n\n")
	r.printServiceResults(w)
	r.printSummary(w)
}

// printConnectionFailure writes a connection failure message to the given
// writer.
//
// Takes w (io.Writer) which receives the formatted failure output.
func (r *DiagnosticsResult) printConnectionFailure(w io.Writer) {
	_, _ = fmt.Fprint(w, "  Status: FAILED\n")
	_, _ = fmt.Fprintf(w, "  Error: %v\n\n", r.ConnectionError)
	_, _ = fmt.Fprint(w, "=== Summary ===\n")
	_, _ = fmt.Fprint(w, "Connection failed. Ensure your Piko app is running with:\n")
	_, _ = fmt.Fprint(w, "  piko.WithMonitoring()\n")
}

// printServiceResults prints the results grouped by service.
//
// Takes w (io.Writer) which receives the formatted output.
func (r *DiagnosticsResult) printServiceResults(w io.Writer) {
	byService := make(map[string][]ServiceResult)
	for _, sr := range r.Services {
		byService[sr.Name] = append(byService[sr.Name], sr)
	}

	serviceOrder := []string{
		serviceHealth,
		serviceMetrics,
		serviceOrchestrator,
		serviceRegistry,
		serviceProfiling,
		serviceWatchdog,
	}

	for _, service := range serviceOrder {
		results, ok := byService[service]
		if !ok {
			continue
		}

		_, _ = fmt.Fprintf(w, "--- %s ---\n", service)
		for _, sr := range results {
			switch {
			case sr.IsSkipped:
				_, _ = fmt.Fprintf(w, "  %s: SKIPPED - %s\n", sr.Method, sr.Details)
			case sr.OK:
				_, _ = fmt.Fprintf(w, "  %s: OK - %s\n", sr.Method, sr.Details)
			default:
				_, _ = fmt.Fprintf(w, "  %s: FAILED - %v\n", sr.Method, sr.Error)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

// printSummary writes the test summary to the given writer.
//
// Takes w (io.Writer) which receives the summary output.
func (r *DiagnosticsResult) printSummary(w io.Writer) {
	_, _ = fmt.Fprint(w, "=== Summary ===\n")
	if r.Skipped > 0 {
		_, _ = fmt.Fprintf(w, "Tests: %d passed, %d failed, %d skipped\n", r.Passed, r.Failed, r.Skipped)
	} else {
		_, _ = fmt.Fprintf(w, "Tests: %d passed, %d failed\n", r.Passed, r.Failed)
	}

	if r.Failed > 0 {
		_, _ = fmt.Fprint(w, "\nSome services failed. This may indicate:\n")
		_, _ = fmt.Fprint(w, "  - Orchestrator/Registry inspectors not wired up\n")
		_, _ = fmt.Fprint(w, "  - Database not initialised (run generator first)\n")
	} else {
		_, _ = fmt.Fprint(w, "\nAll services operational.\n")
	}
}

// addResult adds a service test result.
//
// Takes service (string) which identifies the service being tested.
// Takes method (string) which specifies the test method name.
// Takes err (error) which indicates the test outcome; nil means success.
// Takes details (string) which provides additional information for passing
// tests.
func (r *DiagnosticsResult) addResult(service, method string, err error, details string) {
	sr := ServiceResult{
		Error:     err,
		Name:      service,
		Method:    method,
		Details:   "",
		OK:        err == nil,
		IsSkipped: false,
	}
	if err == nil {
		sr.Details = details
		r.Passed++
	} else {
		r.Failed++
	}
	r.Services = append(r.Services, sr)
}

// addSkippedResult records a service check as skipped. This is used for
// opt-in services that are not registered on the server, to avoid false
// failure alarms.
//
// Takes service (string) which identifies the service being tested.
// Takes method (string) which specifies the test method name.
// Takes details (string) which provides additional information about why the
// check was skipped.
func (r *DiagnosticsResult) addSkippedResult(service, method, details string) {
	r.Services = append(r.Services, ServiceResult{
		Name:      service,
		Method:    method,
		Details:   details,
		IsSkipped: true,
	})
	r.Skipped++
}

// RunDiagnostics tests connectivity to a gRPC monitoring endpoint and runs
// tests for all services.
//
// Takes endpoint (string) which is the gRPC server address
// (e.g. "127.0.0.1:9091"). If empty, uses the default monitoring endpoint.
// Takes creds (credentials.TransportCredentials) which is optional TLS
// credentials; nil uses insecure credentials.
//
// Returns *DiagnosticsResult which contains the test results for the
// connection and each service.
func RunDiagnostics(ctx context.Context, endpoint string, creds credentials.TransportCredentials) *DiagnosticsResult {
	if endpoint == "" {
		endpoint = tui_dto.DefaultMonitoringEndpoint
	}

	result := &DiagnosticsResult{
		ConnectionError: nil,
		Endpoint:        endpoint,
		Services:        nil,
		Passed:          0,
		Failed:          0,
		Connected:       false,
	}

	ctx, cancel := context.WithTimeoutCause(ctx, diagnosticsTimeout,
		fmt.Errorf("diagnostics collection exceeded %s timeout", diagnosticsTimeout))
	defer cancel()

	conn, err := dialWithTimeout(ctx, endpoint, creds)
	if err != nil {
		result.ConnectionError = err
		return result
	}
	defer func() { _ = conn.Close() }()

	result.Connected = true
	runServiceTests(ctx, conn, result)

	return result
}

// dialWithTimeout creates a gRPC connection to the given endpoint.
//
// Takes endpoint (string) which specifies the gRPC server address.
//
// Returns *grpc.ClientConn which is the connection ready for use.
// Returns error when the client cannot be created or the server cannot be
// reached.
func dialWithTimeout(ctx context.Context, endpoint string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	transportCreds := insecure.NewCredentials()
	if creds != nil {
		transportCreds = creds
	}

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(transportCreds),
	)
	if err != nil {
		return nil, fmt.Errorf("creating gRPC client: %w", err)
	}

	healthClient := pb.NewHealthServiceClient(conn)
	if _, err := healthClient.GetHealth(ctx, &pb.GetHealthRequest{}); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connecting to gRPC server: %w", err)
	}

	return conn, nil
}

// runServiceTests runs diagnostic tests for all services over the given
// connection.
//
// Takes conn (*grpc.ClientConn) which provides the gRPC connection to the
// server.
// Takes result (*DiagnosticsResult) which stores the test results.
func runServiceTests(ctx context.Context, conn *grpc.ClientConn, result *DiagnosticsResult) {
	testHealthService(ctx, pb.NewHealthServiceClient(conn), result)
	testMetricsService(ctx, pb.NewMetricsServiceClient(conn), result)
	testOrchestratorService(ctx, pb.NewOrchestratorInspectorServiceClient(conn), result)
	testRegistryService(ctx, pb.NewRegistryInspectorServiceClient(conn), result)
	testProfilingService(ctx, pb.NewProfilingServiceClient(conn), result)
	testWatchdogService(ctx, pb.NewWatchdogInspectorServiceClient(conn), result)
}

// testHealthService checks the health service endpoints and records the
// liveness and readiness status.
//
// Takes client (pb.HealthServiceClient) which provides access to the health
// service API.
// Takes result (*DiagnosticsResult) which collects the test results.
func testHealthService(ctx context.Context, client pb.HealthServiceClient, result *DiagnosticsResult) {
	healthResp, err := client.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		result.addResult(serviceHealth, "GetHealth", err, resultFailed)
		return
	}
	livenessState := "nil"
	readinessState := "nil"
	if healthResp.GetLiveness() != nil {
		livenessState = healthResp.GetLiveness().GetState()
	}
	if healthResp.GetReadiness() != nil {
		readinessState = healthResp.GetReadiness().GetState()
	}
	result.addResult(serviceHealth, "GetHealth", nil,
		fmt.Sprintf("liveness=%s, readiness=%s, timestamp=%d", livenessState, readinessState, healthResp.GetTimestampMs()))
}

// testMetricsService checks the gRPC endpoints of the metrics service.
//
// Takes client (pb.MetricsServiceClient) which provides the gRPC client for
// calling metrics service methods.
// Takes result (*DiagnosticsResult) which collects the test outcomes.
func testMetricsService(ctx context.Context, client pb.MetricsServiceClient, result *DiagnosticsResult) {
	metricsResp, err := client.GetMetrics(ctx, &pb.GetMetricsRequest{})
	if err != nil {
		result.addResult(serviceMetrics, "GetMetrics", err, resultFailed)
	} else {
		result.addResult(serviceMetrics, "GetMetrics", nil, fmt.Sprintf("%d metrics", len(metricsResp.GetMetrics())))
	}

	tracesResp, err := client.GetTraces(ctx, &pb.GetTracesRequest{})
	if err != nil {
		result.addResult(serviceMetrics, "GetTraces", err, resultFailed)
	} else {
		result.addResult(serviceMetrics, "GetTraces", nil, fmt.Sprintf("%d spans", len(tracesResp.GetSpans())))
	}

	systemResp, err := client.GetSystemStats(ctx, &pb.GetSystemStatsRequest{})
	if err != nil {
		result.addResult(serviceMetrics, "GetSystemStats", err, resultFailed)
	} else {
		result.addResult(serviceMetrics, "GetSystemStats", nil,
			fmt.Sprintf("goroutines=%d, uptime=%dms", systemResp.GetNumGoroutines(), systemResp.GetUptimeMs()))
	}

	fdResp, err := client.GetFileDescriptors(ctx, &pb.GetFileDescriptorsRequest{})
	if err != nil {
		result.addResult(serviceMetrics, "GetFileDescriptors", err, resultFailed)
	} else {
		result.addResult(serviceMetrics, "GetFileDescriptors", nil, fmt.Sprintf("total=%d fds", fdResp.GetTotal()))
	}
}

// testOrchestratorService tests the orchestrator inspector service endpoints.
//
// Takes client (pb.OrchestratorInspectorServiceClient) which provides access
// to the orchestrator inspector service.
// Takes result (*DiagnosticsResult) which collects the test results.
func testOrchestratorService(ctx context.Context, client pb.OrchestratorInspectorServiceClient, result *DiagnosticsResult) {
	taskSummaryResp, err := client.GetTaskSummary(ctx, &pb.GetTaskSummaryRequest{})
	if err != nil {
		result.addResult(serviceOrchestrator, "GetTaskSummary", err, resultFailed)
	} else {
		result.addResult(serviceOrchestrator, "GetTaskSummary", nil,
			fmt.Sprintf("%d status groups", len(taskSummaryResp.GetSummaries())))
	}

	recentTasksResp, err := client.ListRecentTasks(ctx, &pb.ListRecentTasksRequest{Limit: diagnosticsLimit})
	if err != nil {
		result.addResult(serviceOrchestrator, "ListRecentTasks", err, resultFailed)
	} else {
		result.addResult(serviceOrchestrator, "ListRecentTasks", nil, fmt.Sprintf("%d tasks", len(recentTasksResp.GetTasks())))
	}

	workflowResp, err := client.ListWorkflowSummary(ctx, &pb.ListWorkflowSummaryRequest{Limit: diagnosticsLimit})
	if err != nil {
		result.addResult(serviceOrchestrator, "ListWorkflowSummary", err, resultFailed)
	} else {
		result.addResult(serviceOrchestrator, "ListWorkflowSummary", nil,
			fmt.Sprintf("%d workflows", len(workflowResp.GetSummaries())))
	}
}

// testRegistryService tests the registry inspector service endpoints.
//
// Takes client (pb.RegistryInspectorServiceClient) which provides access to
// the registry inspector service.
// Takes result (*DiagnosticsResult) which collects the test outcomes.
func testRegistryService(ctx context.Context, client pb.RegistryInspectorServiceClient, result *DiagnosticsResult) {
	artefactSummaryResp, err := client.GetArtefactSummary(ctx, &pb.GetArtefactSummaryRequest{})
	if err != nil {
		result.addResult(serviceRegistry, "GetArtefactSummary", err, resultFailed)
	} else {
		result.addResult(serviceRegistry, "GetArtefactSummary", nil,
			fmt.Sprintf("%d status groups", len(artefactSummaryResp.GetSummaries())))
	}

	recentArtefactsResp, err := client.ListRecentArtefacts(ctx, &pb.ListRecentArtefactsRequest{Limit: diagnosticsLimit})
	if err != nil {
		result.addResult(serviceRegistry, "ListRecentArtefacts", err, resultFailed)
	} else {
		result.addResult(serviceRegistry, "ListRecentArtefacts", nil,
			fmt.Sprintf("%d artefacts", len(recentArtefactsResp.GetArtefacts())))
	}

	variantSummaryResp, err := client.GetVariantSummary(ctx, &pb.GetVariantSummaryRequest{})
	if err != nil {
		result.addResult(serviceRegistry, "GetVariantSummary", err, resultFailed)
	} else {
		result.addResult(serviceRegistry, "GetVariantSummary", nil,
			fmt.Sprintf("%d status groups", len(variantSummaryResp.GetSummaries())))
	}
}

// testProfilingService tests the profiling service endpoint.
//
// Takes client (pb.ProfilingServiceClient) which provides access to the
// profiling service API.
// Takes result (*DiagnosticsResult) which collects the test results.
func testProfilingService(ctx context.Context, client pb.ProfilingServiceClient, result *DiagnosticsResult) {
	statusResp, err := client.GetProfilingStatus(ctx, &pb.GetProfilingStatusRequest{})
	testOptInEnabledService(result, serviceProfiling, "GetProfilingStatus", err, statusResp.GetEnabled())
}

// testWatchdogService tests the watchdog inspector service endpoint.
//
// Takes client (pb.WatchdogInspectorServiceClient) which provides access to
// the watchdog inspector service API.
// Takes result (*DiagnosticsResult) which collects the test results.
func testWatchdogService(ctx context.Context, client pb.WatchdogInspectorServiceClient, result *DiagnosticsResult) {
	statusResp, err := client.GetWatchdogStatus(ctx, &pb.GetWatchdogStatusRequest{})
	testOptInEnabledService(result, serviceWatchdog, "GetWatchdogStatus", err, statusResp.GetEnabled())
}

// testOptInEnabledService records the result for an opt-in service that
// returns an enabled/disabled status. If the RPC returns Unimplemented, the
// check is recorded as skipped rather than failed.
//
// Takes result (*DiagnosticsResult) which collects the test results.
// Takes service (string) which identifies the service being tested.
// Takes method (string) which specifies the RPC method name.
// Takes err (error) which is the RPC error, or nil on success.
// Takes enabled (bool) which indicates whether the service reported itself
// as enabled.
func testOptInEnabledService(result *DiagnosticsResult, service, method string, err error, enabled bool) {
	if err != nil {
		if s, ok := grpcstatus.FromError(err); ok && s.Code() == codes.Unimplemented {
			result.addSkippedResult(service, method, resultSkipped)
			return
		}
		result.addResult(service, method, err, resultFailed)
		return
	}

	status := "disabled"
	if enabled {
		status = "enabled"
	}
	result.addResult(service, method, nil, fmt.Sprintf("status=%s", status))
}
