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
	"context"
	"fmt"
	"time"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/clock"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

// MetricsServiceOption configures a MetricsService.
type MetricsServiceOption func(*MetricsService)

// MetricsService implements the gRPC MetricsServiceServer interface.
type MetricsService struct {
	pb.UnimplementedMetricsServiceServer

	// telemetry provides access to metrics and trace data; nil disables telemetry.
	telemetry TelemetryProvider

	// system provides system and runtime statistics; nil disables stats collection.
	system SystemStatsProvider

	// fds provides file descriptor information; nil disables the feature.
	fds ResourceProvider

	// cacheStats provides render cache statistics; nil when not available.
	cacheStats RenderCacheStatsProvider

	// clock provides time operations for timestamps and tickers.
	clock clock.Clock
}

// TelemetryProvider is a type alias for monitoring_domain.TelemetryProvider.
type TelemetryProvider = monitoring_domain.TelemetryProvider

// SystemStatsProvider is an alias for monitoring_domain.SystemStatsProvider.
type SystemStatsProvider = monitoring_domain.SystemStatsProvider

// ResourceProvider is an alias for monitoring_domain.ResourceProvider.
type ResourceProvider = monitoring_domain.ResourceProvider

// RenderCacheStatsProvider is an alias for
// monitoring_domain.RenderCacheStatsProvider.
type RenderCacheStatsProvider = monitoring_domain.RenderCacheStatsProvider

// NewMetricsService creates a new MetricsService.
//
// Takes telemetry (TelemetryProvider) which provides telemetry data access.
// Takes system (SystemStatsProvider) which provides system statistics.
// Takes fds (ResourceProvider) which provides file descriptor metrics.
// Takes cacheStats (RenderCacheStatsProvider) which provides render cache
// statistics; may be nil.
//
// Returns *MetricsService which is the configured service ready for use.
func NewMetricsService(telemetry TelemetryProvider, system SystemStatsProvider, fds ResourceProvider, cacheStats RenderCacheStatsProvider, opts ...MetricsServiceOption) *MetricsService {
	s := &MetricsService{
		UnimplementedMetricsServiceServer: pb.UnimplementedMetricsServiceServer{},
		telemetry:                         telemetry,
		system:                            system,
		fds:                               fds,
		cacheStats:                        cacheStats,
		clock:                             clock.RealClock(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetMetrics returns all current metrics.
//
// Returns *pb.GetMetricsResponse which contains the collected metrics with a
// timestamp.
// Returns error when the metrics cannot be retrieved.
func (s *MetricsService) GetMetrics(_ context.Context, _ *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	if s.telemetry == nil {
		return &pb.GetMetricsResponse{
			Metrics:     nil,
			TimestampMs: s.clock.Now().UnixMilli(),
		}, nil
	}

	metrics := s.telemetry.GetMetrics()
	pbMetrics := make([]*pb.Metric, len(metrics))

	for i, m := range metrics {
		dataPoints := make([]*pb.MetricDataPoint, len(m.DataPoints))
		for j, dp := range m.DataPoints {
			dataPoints[j] = &pb.MetricDataPoint{
				TimestampMs: dp.TimestampMs,
				Attributes:  dp.Attributes,
				Value:       dp.Value,
			}
		}

		pbMetrics[i] = &pb.Metric{
			Name:        m.Name,
			Description: m.Description,
			Unit:        m.Unit,
			Type:        m.Type,
			DataPoints:  dataPoints,
		}
	}

	return &pb.GetMetricsResponse{
		Metrics:     pbMetrics,
		TimestampMs: s.clock.Now().UnixMilli(),
	}, nil
}

// GetTraces returns recent trace spans.
//
// Takes request (*pb.GetTracesRequest) which specifies the query parameters
// including optional trace ID, limit, and error filtering.
//
// Returns *pb.GetTracesResponse which contains the matching spans with
// metadata.
// Returns error when the request cannot be processed.
func (s *MetricsService) GetTraces(_ context.Context, request *pb.GetTracesRequest) (*pb.GetTracesResponse, error) {
	if s.telemetry == nil {
		return &pb.GetTracesResponse{
			Spans:       nil,
			TimestampMs: s.clock.Now().UnixMilli(),
			Count:       0,
		}, nil
	}

	var spans []monitoring_domain.SpanData
	if request.GetTraceId() != "" {
		spans = s.telemetry.GetSpanByTraceID(request.GetTraceId())
	} else {
		limit := int(request.GetLimit())
		if limit <= 0 {
			limit = defaultSpanLimit
		}
		spans = s.telemetry.GetSpans(limit, request.GetErrorsOnly())
	}

	pbSpans := make([]*pb.Span, len(spans))
	for i := range spans {
		span := &spans[i]
		events := make([]*pb.SpanEvent, len(span.Events))
		for j, e := range span.Events {
			events[j] = &pb.SpanEvent{
				Name:        e.Name,
				TimestampMs: e.TimestampMs,
				Attributes:  e.Attributes,
			}
		}

		pbSpans[i] = &pb.Span{
			TraceId:       span.TraceID,
			SpanId:        span.SpanID,
			ParentSpanId:  span.ParentSpanID,
			Name:          span.Name,
			Kind:          span.Kind,
			Status:        span.Status,
			StatusMessage: span.StatusMessage,
			ServiceName:   span.ServiceName,
			StartTimeMs:   span.StartTimeMs,
			EndTimeMs:     span.EndTimeMs,
			DurationNs:    span.DurationNs,
			Attributes:    span.Attributes,
			Events:        events,
		}
	}

	return &pb.GetTracesResponse{
		Spans:       pbSpans,
		TimestampMs: s.clock.Now().UnixMilli(),
		Count:       safeconv.IntToInt32(len(pbSpans)),
	}, nil
}

// GetSystemStats returns system and runtime statistics.
//
// Returns *pb.GetSystemStatsResponse which contains the current system stats,
// or an empty response if the system collector is not available.
// Returns error when the stats cannot be retrieved.
func (s *MetricsService) GetSystemStats(_ context.Context, _ *pb.GetSystemStatsRequest) (*pb.GetSystemStatsResponse, error) {
	if s.system == nil {
		return emptySystemStatsResponse(s.clock.Now()), nil
	}
	response := convertSystemStatsToPB(new(s.system.GetStats()))

	if s.cacheStats != nil {
		response.Cache = &pb.CacheInfo{
			ComponentCacheSize: safeconv.IntToInt32(s.cacheStats.GetComponentCacheSize()),
			SvgCacheSize:       safeconv.IntToInt32(s.cacheStats.GetSVGCacheSize()),
		}
	}

	return response, nil
}

// GetFileDescriptors returns open file descriptor information.
//
// Returns *pb.GetFileDescriptorsResponse which contains categorised file
// descriptor data with counts and timestamps.
// Returns error when the request cannot be processed.
func (s *MetricsService) GetFileDescriptors(_ context.Context, _ *pb.GetFileDescriptorsRequest) (*pb.GetFileDescriptorsResponse, error) {
	if s.fds == nil {
		return &pb.GetFileDescriptorsResponse{
			Categories:  nil,
			Total:       0,
			TimestampMs: s.clock.Now().UnixMilli(),
		}, nil
	}

	data := s.fds.GetResources()

	categories := make([]*pb.FileDescriptorCategory, len(data.Categories))
	for i, cat := range data.Categories {
		fds := make([]*pb.FileDescriptorInfo, len(cat.Resources))
		for j, fd := range cat.Resources {
			fds[j] = &pb.FileDescriptorInfo{
				Fd:          fd.FD,
				Category:    fd.Category,
				Target:      fd.Target,
				FirstSeenMs: fd.FirstSeenMs,
				AgeMs:       fd.AgeMs,
			}
		}
		categories[i] = &pb.FileDescriptorCategory{
			Category: cat.Category,
			Fds:      fds,
			Count:    cat.Count,
		}
	}

	return &pb.GetFileDescriptorsResponse{
		Categories:  categories,
		Total:       data.Total,
		TimestampMs: data.TimestampMs,
	}, nil
}

// WatchMetrics streams metric updates at the requested interval.
//
// Takes request (*pb.WatchMetricsRequest) which specifies the update interval.
// Takes stream (pb.MetricsService_WatchMetricsServer) which receives metric
// updates.
//
// Returns error when the stream context is cancelled or sending fails.
func (s *MetricsService) WatchMetrics(request *pb.WatchMetricsRequest, stream pb.MetricsService_WatchMetricsServer) error {
	if s.telemetry == nil {
		return nil
	}

	interval := time.Duration(request.GetIntervalMs()) * time.Millisecond
	if interval < minWatchIntervalMs*time.Millisecond {
		interval = 1 * time.Second
	}

	ticker := s.clock.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C():
			metrics := s.telemetry.GetMetrics()
			pbMetrics := make([]*pb.Metric, len(metrics))

			for i, m := range metrics {
				dataPoints := make([]*pb.MetricDataPoint, len(m.DataPoints))
				for j, dp := range m.DataPoints {
					dataPoints[j] = &pb.MetricDataPoint{
						TimestampMs: dp.TimestampMs,
						Attributes:  dp.Attributes,
						Value:       dp.Value,
					}
				}

				pbMetrics[i] = &pb.Metric{
					Name:        m.Name,
					Description: m.Description,
					Unit:        m.Unit,
					Type:        m.Type,
					DataPoints:  dataPoints,
				}
			}

			if err := stream.Send(&pb.MetricsUpdate{
				Metrics:     pbMetrics,
				TimestampMs: s.clock.Now().UnixMilli(),
			}); err != nil {
				return fmt.Errorf("sending metrics update: %w", err)
			}
		}
	}
}

// WithMetricsServiceClock sets the clock used for timestamp generation. If not
// provided, the real system clock is used.
//
// Takes clk (clock.Clock) which provides time operations.
//
// Returns MetricsServiceOption which configures the service's clock.
func WithMetricsServiceClock(clk clock.Clock) MetricsServiceOption {
	return func(s *MetricsService) {
		if clk != nil {
			s.clock = clk
		}
	}
}

// emptySystemStatsResponse returns a response with only a timestamp when stats
// are unavailable.
//
// Takes now (time.Time) which provides the current time for the timestamp.
//
// Returns *pb.GetSystemStatsResponse which contains only the given timestamp
// with all other fields set to nil or zero values.
func emptySystemStatsResponse(now time.Time) *pb.GetSystemStatsResponse {
	return &pb.GetSystemStatsResponse{
		Build:         nil,
		Runtime:       nil,
		Gc:            nil,
		Memory:        nil,
		Process:       nil,
		TimestampMs:   now.UnixMilli(),
		UptimeMs:      0,
		NumCgoCalls:   0,
		CpuMillicores: 0,
		NumCpu:        0,
		Gomaxprocs:    0,
		NumGoroutines: 0,
	}
}

// convertSystemStatsToPB converts domain SystemStats to protobuf format.
//
// Takes stats (*monitoring_domain.SystemStats) which contains the system
// statistics to convert.
//
// Returns *pb.GetSystemStatsResponse which contains the converted statistics.
func convertSystemStatsToPB(stats *monitoring_domain.SystemStats) *pb.GetSystemStatsResponse {
	return &pb.GetSystemStatsResponse{
		Build:                convertBuildInfoToPB(stats.Build),
		Runtime:              convertRuntimeInfoToPB(stats.Runtime),
		Gc:                   convertGCInfoToPB(stats.GC),
		Memory:               convertMemoryInfoToPB(stats.Memory),
		Process:              convertProcessInfoToPB(stats.Process),
		Schedule:             convertSchedulerInfoToPB(stats.Schedule),
		Sync:                 convertSyncInfoToPB(stats.Sync),
		TimestampMs:          stats.TimestampMs,
		UptimeMs:             stats.UptimeMs,
		NumCgoCalls:          stats.NumCGOCalls,
		CpuMillicores:        stats.CPUMillicores,
		NumCpu:               stats.NumCPU,
		Gomaxprocs:           stats.GOMAXPROCS,
		NumGoroutines:        stats.NumGoroutines,
		SystemUptimeMs:       stats.SystemUptimeMs,
		CgroupPath:           stats.CgroupPath,
		MonitoringListenAddr: stats.MonitoringListenAddr,
	}
}

// convertBuildInfoToPB converts a domain BuildInfo to its protobuf representation.
//
// Takes build (monitoring_domain.BuildInfo) which contains the build metadata.
//
// Returns *pb.BuildInfo which is the protobuf message for gRPC transmission.
func convertBuildInfoToPB(build monitoring_domain.BuildInfo) *pb.BuildInfo {
	return &pb.BuildInfo{
		GoVersion:     build.GoVersion,
		Version:       build.Version,
		Commit:        build.Commit,
		BuildTime:     build.BuildTime,
		Os:            build.OS,
		Arch:          build.Arch,
		ModulePath:    build.ModulePath,
		ModuleVersion: build.ModuleVersion,
		VcsModified:   build.VCSModified,
		VcsTime:       build.VCSTime,
	}
}

// convertRuntimeInfoToPB converts a domain RuntimeInfo to its protobuf form.
//
// Takes rt (monitoring_domain.RuntimeInfo) which holds Go runtime settings.
//
// Returns *pb.RuntimeInfo which contains the GOGC, GOMEMLIMIT, and Compiler
// values.
func convertRuntimeInfoToPB(rt monitoring_domain.RuntimeInfo) *pb.RuntimeInfo {
	return &pb.RuntimeInfo{
		Gogc:       rt.GOGC,
		Gomemlimit: rt.GOMEMLIMIT,
		Compiler:   rt.Compiler,
	}
}

// convertGCInfoToPB converts a domain GCInfo to its protobuf representation.
//
// Takes gc (monitoring_domain.GCInfo) which contains garbage collection stats.
//
// Returns *pb.GCInfo which is the protobuf message for the GC information.
func convertGCInfoToPB(gc monitoring_domain.GCInfo) *pb.GCInfo {
	return &pb.GCInfo{
		RecentPauses:  gc.RecentPauses,
		LastGcNs:      gc.LastGC,
		PauseTotalNs:  gc.PauseTotalNs,
		LastPauseNs:   gc.LastPauseNs,
		GcCpuFraction: gc.GCCPUFraction,
		NextGc:        gc.NextGC,
		NumGc:         gc.NumGC,
		NumForcedGc:   gc.NumForcedGC,
		PauseP50Ns:    gc.PauseP50.Nanoseconds(),
		PauseP95Ns:    gc.PauseP95.Nanoseconds(),
		PauseP99Ns:    gc.PauseP99.Nanoseconds(),
	}
}

// convertMemoryInfoToPB converts a domain memory info struct to its protobuf
// representation.
//
// Takes mem (monitoring_domain.MemoryInfo) which contains the memory statistics
// to convert.
//
// Returns *pb.MemoryInfo which is the protobuf message with all memory fields
// mapped.
func convertMemoryInfoToPB(mem monitoring_domain.MemoryInfo) *pb.MemoryInfo {
	return &pb.MemoryInfo{
		Alloc:             mem.Alloc,
		TotalAlloc:        mem.TotalAlloc,
		Sys:               mem.Sys,
		HeapAlloc:         mem.HeapAlloc,
		HeapSys:           mem.HeapSys,
		HeapIdle:          mem.HeapIdle,
		HeapInuse:         mem.HeapInuse,
		HeapObjects:       mem.HeapObjects,
		HeapReleased:      mem.HeapReleased,
		StackSys:          mem.StackSys,
		Mallocs:           mem.Mallocs,
		Frees:             mem.Frees,
		LiveObjects:       mem.LiveObjects,
		StackInuse:        mem.StackInuse,
		MspanInuse:        mem.MSpanInuse,
		MspanSys:          mem.MSpanSys,
		McacheInuse:       mem.MCacheInuse,
		McacheSys:         mem.MCacheSys,
		GcSys:             mem.GCSys,
		OtherSys:          mem.OtherSys,
		BuckhashSys:       mem.BuckHashSys,
		Lookups:           mem.Lookups,
		HeapObjectsBytes:  mem.HeapObjectsBytes,
		HeapFreeBytes:     mem.HeapFreeBytes,
		HeapReleasedBytes: mem.HeapReleasedBytes,
		HeapStacksBytes:   mem.HeapStacksBytes,
		HeapUnusedBytes:   mem.HeapUnusedBytes,
		TotalBytes:        mem.TotalBytes,
	}
}

// convertSchedulerInfoToPB converts a domain SchedulerInfo to its protobuf
// representation.
//
// Takes sched (monitoring_domain.SchedulerInfo) which contains scheduler
// latency percentiles, goroutine count, and GOMAXPROCS.
//
// Returns *pb.SchedulerInfo which is the protobuf message for the scheduler
// information.
func convertSchedulerInfoToPB(sched monitoring_domain.SchedulerInfo) *pb.SchedulerInfo {
	return &pb.SchedulerInfo{
		LatencyP50Ns:   sched.LatencyP50.Nanoseconds(),
		LatencyP99Ns:   sched.LatencyP99.Nanoseconds(),
		GoroutineCount: sched.GoroutineCount,
		Gomaxprocs:     sched.GoMaxProcs,
	}
}

// convertSyncInfoToPB converts a domain SyncInfo to its protobuf
// representation.
//
// Takes sync (monitoring_domain.SyncInfo) which contains synchronisation
// contention metrics.
//
// Returns *pb.SyncInfo which is the protobuf message for the sync information.
func convertSyncInfoToPB(sync monitoring_domain.SyncInfo) *pb.SyncInfo {
	return &pb.SyncInfo{
		MutexWaitTotalSeconds: sync.MutexWaitTotalSeconds,
	}
}

// convertProcessInfoToPB converts a domain process info to its protobuf form.
//
// Takes proc (monitoring_domain.ProcessInfo) which is the domain model to
// convert.
//
// Returns *pb.ProcessInfo which contains the converted process information.
func convertProcessInfoToPB(proc monitoring_domain.ProcessInfo) *pb.ProcessInfo {
	return &pb.ProcessInfo{
		Pid:              proc.PID,
		ThreadCount:      proc.ThreadCount,
		FdCount:          proc.FDCount,
		Rss:              proc.RSS,
		Hostname:         proc.Hostname,
		Executable:       proc.Executable,
		Cwd:              proc.CWD,
		Uid:              proc.UID,
		Gid:              proc.GID,
		MaxOpenFilesSoft: proc.MaxOpenFilesSoft,
		MaxOpenFilesHard: proc.MaxOpenFilesHard,
		IoReadBytes:      proc.IoReadBytes,
		IoWriteBytes:     proc.IoWriteBytes,
		IoRchar:          proc.IoRchar,
		IoWchar:          proc.IoWchar,
		Ppid:             proc.PPID,
	}
}
