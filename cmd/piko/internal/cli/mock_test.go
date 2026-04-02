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
	"bytes"
	"context"
	"io"

	"google.golang.org/grpc"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockConnection struct {
	health       pb.HealthServiceClient
	metrics      pb.MetricsServiceClient
	orchestrator pb.OrchestratorInspectorServiceClient
	registry     pb.RegistryInspectorServiceClient
	dispatcher   pb.DispatcherInspectorServiceClient
	rateLimiter  pb.RateLimiterInspectorServiceClient
	providerInfo pb.ProviderInfoServiceClient
}

func (m *mockConnection) HealthClient() pb.HealthServiceClient   { return m.health }
func (m *mockConnection) MetricsClient() pb.MetricsServiceClient { return m.metrics }
func (m *mockConnection) OrchestratorClient() pb.OrchestratorInspectorServiceClient {
	return m.orchestrator
}
func (m *mockConnection) RegistryClient() pb.RegistryInspectorServiceClient     { return m.registry }
func (m *mockConnection) DispatcherClient() pb.DispatcherInspectorServiceClient { return m.dispatcher }
func (m *mockConnection) RateLimiterClient() pb.RateLimiterInspectorServiceClient {
	return m.rateLimiter
}
func (m *mockConnection) ProviderInfoClient() pb.ProviderInfoServiceClient { return m.providerInfo }
func (*mockConnection) Close() error                                       { return nil }

type mockHealthClient struct {
	pb.HealthServiceClient
	GetHealthFunc   func(ctx context.Context, in *pb.GetHealthRequest, opts ...grpc.CallOption) (*pb.GetHealthResponse, error)
	WatchHealthFunc func(ctx context.Context, in *pb.WatchHealthRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.HealthUpdate], error)
}

func (m *mockHealthClient) GetHealth(ctx context.Context, in *pb.GetHealthRequest, opts ...grpc.CallOption) (*pb.GetHealthResponse, error) {
	return m.GetHealthFunc(ctx, in, opts...)
}

func (m *mockHealthClient) WatchHealth(ctx context.Context, in *pb.WatchHealthRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.HealthUpdate], error) {
	return m.WatchHealthFunc(ctx, in, opts...)
}

type mockMetricsClient struct {
	pb.MetricsServiceClient
	GetMetricsFunc         func(ctx context.Context, in *pb.GetMetricsRequest, opts ...grpc.CallOption) (*pb.GetMetricsResponse, error)
	GetTracesFunc          func(ctx context.Context, in *pb.GetTracesRequest, opts ...grpc.CallOption) (*pb.GetTracesResponse, error)
	GetSystemStatsFunc     func(ctx context.Context, in *pb.GetSystemStatsRequest, opts ...grpc.CallOption) (*pb.GetSystemStatsResponse, error)
	GetFileDescriptorsFunc func(ctx context.Context, in *pb.GetFileDescriptorsRequest, opts ...grpc.CallOption) (*pb.GetFileDescriptorsResponse, error)
	WatchMetricsFunc       func(ctx context.Context, in *pb.WatchMetricsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.MetricsUpdate], error)
}

func (m *mockMetricsClient) GetMetrics(ctx context.Context, in *pb.GetMetricsRequest, opts ...grpc.CallOption) (*pb.GetMetricsResponse, error) {
	return m.GetMetricsFunc(ctx, in, opts...)
}

func (m *mockMetricsClient) GetTraces(ctx context.Context, in *pb.GetTracesRequest, opts ...grpc.CallOption) (*pb.GetTracesResponse, error) {
	return m.GetTracesFunc(ctx, in, opts...)
}

func (m *mockMetricsClient) GetSystemStats(ctx context.Context, in *pb.GetSystemStatsRequest, opts ...grpc.CallOption) (*pb.GetSystemStatsResponse, error) {
	return m.GetSystemStatsFunc(ctx, in, opts...)
}

func (m *mockMetricsClient) GetFileDescriptors(ctx context.Context, in *pb.GetFileDescriptorsRequest, opts ...grpc.CallOption) (*pb.GetFileDescriptorsResponse, error) {
	return m.GetFileDescriptorsFunc(ctx, in, opts...)
}

func (m *mockMetricsClient) WatchMetrics(ctx context.Context, in *pb.WatchMetricsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.MetricsUpdate], error) {
	return m.WatchMetricsFunc(ctx, in, opts...)
}

type mockOrchestratorClient struct {
	pb.OrchestratorInspectorServiceClient
	GetTaskSummaryFunc      func(ctx context.Context, in *pb.GetTaskSummaryRequest, opts ...grpc.CallOption) (*pb.GetTaskSummaryResponse, error)
	ListRecentTasksFunc     func(ctx context.Context, in *pb.ListRecentTasksRequest, opts ...grpc.CallOption) (*pb.ListRecentTasksResponse, error)
	ListWorkflowSummaryFunc func(ctx context.Context, in *pb.ListWorkflowSummaryRequest, opts ...grpc.CallOption) (*pb.ListWorkflowSummaryResponse, error)
	WatchTasksFunc          func(ctx context.Context, in *pb.WatchTasksRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.TasksUpdate], error)
}

func (m *mockOrchestratorClient) GetTaskSummary(ctx context.Context, in *pb.GetTaskSummaryRequest, opts ...grpc.CallOption) (*pb.GetTaskSummaryResponse, error) {
	return m.GetTaskSummaryFunc(ctx, in, opts...)
}

func (m *mockOrchestratorClient) ListRecentTasks(ctx context.Context, in *pb.ListRecentTasksRequest, opts ...grpc.CallOption) (*pb.ListRecentTasksResponse, error) {
	return m.ListRecentTasksFunc(ctx, in, opts...)
}

func (m *mockOrchestratorClient) ListWorkflowSummary(ctx context.Context, in *pb.ListWorkflowSummaryRequest, opts ...grpc.CallOption) (*pb.ListWorkflowSummaryResponse, error) {
	return m.ListWorkflowSummaryFunc(ctx, in, opts...)
}

func (m *mockOrchestratorClient) WatchTasks(ctx context.Context, in *pb.WatchTasksRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.TasksUpdate], error) {
	return m.WatchTasksFunc(ctx, in, opts...)
}

type mockRegistryClient struct {
	pb.RegistryInspectorServiceClient
	GetArtefactSummaryFunc  func(ctx context.Context, in *pb.GetArtefactSummaryRequest, opts ...grpc.CallOption) (*pb.GetArtefactSummaryResponse, error)
	GetVariantSummaryFunc   func(ctx context.Context, in *pb.GetVariantSummaryRequest, opts ...grpc.CallOption) (*pb.GetVariantSummaryResponse, error)
	ListRecentArtefactsFunc func(ctx context.Context, in *pb.ListRecentArtefactsRequest, opts ...grpc.CallOption) (*pb.ListRecentArtefactsResponse, error)
	WatchArtefactsFunc      func(ctx context.Context, in *pb.WatchArtefactsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ArtefactsUpdate], error)
}

func (m *mockRegistryClient) GetArtefactSummary(ctx context.Context, in *pb.GetArtefactSummaryRequest, opts ...grpc.CallOption) (*pb.GetArtefactSummaryResponse, error) {
	return m.GetArtefactSummaryFunc(ctx, in, opts...)
}

func (m *mockRegistryClient) GetVariantSummary(ctx context.Context, in *pb.GetVariantSummaryRequest, opts ...grpc.CallOption) (*pb.GetVariantSummaryResponse, error) {
	return m.GetVariantSummaryFunc(ctx, in, opts...)
}

func (m *mockRegistryClient) ListRecentArtefacts(ctx context.Context, in *pb.ListRecentArtefactsRequest, opts ...grpc.CallOption) (*pb.ListRecentArtefactsResponse, error) {
	return m.ListRecentArtefactsFunc(ctx, in, opts...)
}

func (m *mockRegistryClient) WatchArtefacts(ctx context.Context, in *pb.WatchArtefactsRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[pb.ArtefactsUpdate], error) {
	return m.WatchArtefactsFunc(ctx, in, opts...)
}

type mockDispatcherClient struct {
	pb.DispatcherInspectorServiceClient
	GetDispatcherSummaryFunc func(ctx context.Context, in *pb.GetDispatcherSummaryRequest, opts ...grpc.CallOption) (*pb.GetDispatcherSummaryResponse, error)
	ListDLQEntriesFunc       func(ctx context.Context, in *pb.ListDLQEntriesRequest, opts ...grpc.CallOption) (*pb.ListDLQEntriesResponse, error)
	GetDLQCountFunc          func(ctx context.Context, in *pb.GetDLQCountRequest, opts ...grpc.CallOption) (*pb.GetDLQCountResponse, error)
}

func (m *mockDispatcherClient) GetDispatcherSummary(ctx context.Context, in *pb.GetDispatcherSummaryRequest, opts ...grpc.CallOption) (*pb.GetDispatcherSummaryResponse, error) {
	return m.GetDispatcherSummaryFunc(ctx, in, opts...)
}

func (m *mockDispatcherClient) ListDLQEntries(ctx context.Context, in *pb.ListDLQEntriesRequest, opts ...grpc.CallOption) (*pb.ListDLQEntriesResponse, error) {
	return m.ListDLQEntriesFunc(ctx, in, opts...)
}

func (m *mockDispatcherClient) GetDLQCount(ctx context.Context, in *pb.GetDLQCountRequest, opts ...grpc.CallOption) (*pb.GetDLQCountResponse, error) {
	return m.GetDLQCountFunc(ctx, in, opts...)
}

type mockRateLimiterClient struct {
	pb.RateLimiterInspectorServiceClient
	GetRateLimiterStatusFunc func(ctx context.Context, in *pb.GetRateLimiterStatusRequest, opts ...grpc.CallOption) (*pb.GetRateLimiterStatusResponse, error)
}

func (m *mockRateLimiterClient) GetRateLimiterStatus(ctx context.Context, in *pb.GetRateLimiterStatusRequest, opts ...grpc.CallOption) (*pb.GetRateLimiterStatusResponse, error) {
	return m.GetRateLimiterStatusFunc(ctx, in, opts...)
}

type mockProviderInfoClient struct {
	pb.ProviderInfoServiceClient
	ListResourceTypesFunc    func(ctx context.Context, in *pb.ListResourceTypesRequest, opts ...grpc.CallOption) (*pb.ListResourceTypesResponse, error)
	ListProvidersFunc        func(ctx context.Context, in *pb.ListProvidersRequest, opts ...grpc.CallOption) (*pb.ListProvidersResponse, error)
	DescribeProviderFunc     func(ctx context.Context, in *pb.DescribeProviderRequest, opts ...grpc.CallOption) (*pb.DescribeProviderResponse, error)
	ListSubResourcesFunc     func(ctx context.Context, in *pb.ListSubResourcesRequest, opts ...grpc.CallOption) (*pb.ListSubResourcesResponse, error)
	DescribeResourceTypeFunc func(ctx context.Context, in *pb.DescribeResourceTypeRequest, opts ...grpc.CallOption) (*pb.DescribeProviderResponse, error)
}

func (m *mockProviderInfoClient) ListResourceTypes(ctx context.Context, in *pb.ListResourceTypesRequest, opts ...grpc.CallOption) (*pb.ListResourceTypesResponse, error) {
	return m.ListResourceTypesFunc(ctx, in, opts...)
}

func (m *mockProviderInfoClient) ListProviders(ctx context.Context, in *pb.ListProvidersRequest, opts ...grpc.CallOption) (*pb.ListProvidersResponse, error) {
	return m.ListProvidersFunc(ctx, in, opts...)
}

func (m *mockProviderInfoClient) DescribeProvider(ctx context.Context, in *pb.DescribeProviderRequest, opts ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
	return m.DescribeProviderFunc(ctx, in, opts...)
}

func (m *mockProviderInfoClient) ListSubResources(ctx context.Context, in *pb.ListSubResourcesRequest, opts ...grpc.CallOption) (*pb.ListSubResourcesResponse, error) {
	return m.ListSubResourcesFunc(ctx, in, opts...)
}

func (m *mockProviderInfoClient) DescribeResourceType(ctx context.Context, in *pb.DescribeResourceTypeRequest, opts ...grpc.CallOption) (*pb.DescribeProviderResponse, error) {
	return m.DescribeResourceTypeFunc(ctx, in, opts...)
}

type mockStream[T any] struct {
	grpc.ServerStreamingClient[T]
	items []*T
	index int
}

func (m *mockStream[T]) Recv() (*T, error) {
	if m.index >= len(m.items) {
		return nil, io.EOF
	}
	item := m.items[m.index]
	m.index++
	return item, nil
}

type mockFileFormatter struct {
	FormatFunc func(ctx context.Context, source []byte) ([]byte, error)
}

func (m *mockFileFormatter) Format(ctx context.Context, source []byte) ([]byte, error) {
	return m.FormatFunc(ctx, source)
}

func newTestCC(conn monitoringConnection) (*CommandContext, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &CommandContext{
		Conn:   conn,
		Opts:   &GlobalOptions{Output: defaultOutputFormat, Endpoint: defaultEndpoint},
		Stdout: stdout,
		Stderr: stderr,
	}, stdout, stderr
}
