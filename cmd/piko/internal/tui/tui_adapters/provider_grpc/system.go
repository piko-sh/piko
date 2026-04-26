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

package provider_grpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

var _ tui_domain.SystemProvider = (*SystemProvider)(nil)

// SystemProvider provides system statistics through a gRPC connection.
// It implements tui_domain.SystemProvider.
type SystemProvider struct {
	// conn holds the gRPC connection with health and metrics clients.
	conn *Connection

	// stats holds the cached system statistics; nil until first Refresh call.
	stats *tui_domain.SystemStats

	// mu guards concurrent access to stats.
	mu sync.RWMutex

	// interval is the duration between data refreshes.
	interval time.Duration
}

// NewSystemProvider creates a new SystemProvider.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which is the refresh interval.
//
// Returns *SystemProvider which is the configured provider.
func NewSystemProvider(conn *Connection, interval time.Duration) *SystemProvider {
	return &SystemProvider{
		conn:     conn,
		stats:    nil,
		mu:       sync.RWMutex{},
		interval: interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier "grpc-system".
func (*SystemProvider) Name() string {
	return "grpc-system"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check fails.
func (p *SystemProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking system provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when resources cannot be released; currently always nil.
func (*SystemProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between data refreshes.
func (p *SystemProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest system stats via gRPC.
//
// Returns error when the gRPC call fails or the connection is unavailable.
//
// Safe for concurrent use. The method locks the internal mutex when updating
// the cached stats.
func (p *SystemProvider) Refresh(ctx context.Context) error {
	return refreshProvider(ctx,
		func(ctx context.Context) (*tui_domain.SystemStats, error) {
			response, err := p.conn.metricsClient.GetSystemStats(ctx, &pb.GetSystemStatsRequest{})
			if err != nil {
				return nil, err
			}
			return convertSystemStats(response), nil
		},
		func(stats *tui_domain.SystemStats) {
			p.mu.Lock()
			p.stats = stats
			p.mu.Unlock()
		},
		"system stats",
	)
}

// GetStats returns the current system statistics.
//
// Returns *tui_domain.SystemStats which contains the current system metrics.
// Returns error when no statistics are available.
//
// Safe for concurrent use.
func (p *SystemProvider) GetStats(_ context.Context) (*tui_domain.SystemStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.stats == nil {
		return nil, errors.New("no system stats available")
	}

	return p.stats, nil
}

// convertSystemStats converts a protobuf system stats response to the domain
// model.
//
// Takes response (*pb.GetSystemStatsResponse) which is the protobuf response to
// convert.
//
// Returns *tui_domain.SystemStats which is the domain representation, or nil
// if response is nil.
func convertSystemStats(response *pb.GetSystemStatsResponse) *tui_domain.SystemStats {
	if response == nil {
		return nil
	}

	return &tui_domain.SystemStats{
		Timestamp:     time.UnixMilli(response.GetTimestampMs()),
		Uptime:        time.Duration(response.GetUptimeMs()) * time.Millisecond,
		NumCPU:        safeconv.Int32ToInt(response.GetNumCpu()),
		GOMAXPROCS:    safeconv.Int32ToInt(response.GetGomaxprocs()),
		NumGoroutines: safeconv.Int32ToInt(response.GetNumGoroutines()),
		NumCGOCalls:   response.GetNumCgoCalls(),
		CPUMillicores: response.GetCpuMillicores(),
		Memory:        convertMemoryStats(response.GetMemory()),
		GC:            convertGCStats(response.GetGc()),
		Build:         convertBuildInfo(response.GetBuild()),
		Runtime:       convertRuntimeConfig(response.GetRuntime()),
		Process:       convertProcessInfo(response.GetProcess()),
		Cache:         convertCacheStats(response.GetCache()),
	}
}

// convertMemoryStats converts protobuf memory stats to domain format.
//
// Takes mem (*pb.MemoryInfo) which contains the memory statistics from the
// monitoring API.
//
// Returns tui_domain.SystemMemoryStats which contains the converted memory
// statistics for the TUI layer.
func convertMemoryStats(mem *pb.MemoryInfo) tui_domain.SystemMemoryStats {
	return tui_domain.SystemMemoryStats{
		Alloc:        mem.GetAlloc(),
		TotalAlloc:   mem.GetTotalAlloc(),
		Sys:          mem.GetSys(),
		HeapAlloc:    mem.GetHeapAlloc(),
		HeapSys:      mem.GetHeapSys(),
		HeapIdle:     mem.GetHeapIdle(),
		HeapInuse:    mem.GetHeapInuse(),
		HeapObjects:  mem.GetHeapObjects(),
		HeapReleased: mem.GetHeapReleased(),
		StackSys:     mem.GetStackSys(),
		Mallocs:      mem.GetMallocs(),
		Frees:        mem.GetFrees(),
		LiveObjects:  mem.GetLiveObjects(),
	}
}

// convertGCStats converts protobuf GC stats to domain format.
//
// Takes gc (*pb.GCInfo) which contains the garbage collection statistics.
//
// Returns tui_domain.SystemGCStats which holds the converted GC statistics.
func convertGCStats(gc *pb.GCInfo) tui_domain.SystemGCStats {
	return tui_domain.SystemGCStats{
		NumGC:         gc.GetNumGc(),
		LastGC:        gc.GetLastGcNs(),
		PauseTotalNs:  gc.GetPauseTotalNs(),
		LastPauseNs:   gc.GetLastPauseNs(),
		NextGC:        gc.GetNextGc(),
		GCCPUFraction: gc.GetGcCpuFraction(),
		RecentPauses:  gc.GetRecentPauses(),
	}
}

// convertBuildInfo converts protobuf build info to domain format.
//
// Takes build (*pb.BuildInfo) which contains the build metadata to convert.
//
// Returns tui_domain.SystemBuildInfo which holds the converted build details.
func convertBuildInfo(build *pb.BuildInfo) tui_domain.SystemBuildInfo {
	return tui_domain.SystemBuildInfo{
		GoVersion: build.GetGoVersion(),
		Version:   build.GetVersion(),
		Commit:    build.GetCommit(),
		BuildTime: build.GetBuildTime(),
		OS:        build.GetOs(),
		Arch:      build.GetArch(),
	}
}

// convertRuntimeConfig converts protobuf runtime config to domain format.
//
// Takes runtime (*pb.RuntimeInfo) which contains the protobuf runtime data.
//
// Returns tui_domain.SystemRuntimeConfig which is the converted domain format.
func convertRuntimeConfig(runtime *pb.RuntimeInfo) tui_domain.SystemRuntimeConfig {
	return tui_domain.SystemRuntimeConfig{
		GOGC:       runtime.GetGogc(),
		GOMEMLIMIT: runtime.GetGomemlimit(),
	}
}

// convertCacheStats converts protobuf cache info to domain format.
//
// Takes cache (*pb.CacheInfo) which provides the render cache statistics.
//
// Returns tui_domain.SystemCacheStats which contains the converted cache
// statistics.
func convertCacheStats(cache *pb.CacheInfo) tui_domain.SystemCacheStats {
	if cache == nil {
		return tui_domain.SystemCacheStats{}
	}
	return tui_domain.SystemCacheStats{
		ComponentCacheSize: safeconv.Int32ToInt(cache.GetComponentCacheSize()),
		SVGCacheSize:       safeconv.Int32ToInt(cache.GetSvgCacheSize()),
	}
}

// convertProcessInfo converts protobuf process info to domain format.
//
// Takes process (*pb.ProcessInfo) which provides the protobuf process data.
//
// Returns tui_domain.SystemProcessInfo which contains the converted process
// information.
func convertProcessInfo(process *pb.ProcessInfo) tui_domain.SystemProcessInfo {
	return tui_domain.SystemProcessInfo{
		PID:         safeconv.Int32ToInt(process.GetPid()),
		ThreadCount: safeconv.Int32ToInt(process.GetThreadCount()),
		FDCount:     safeconv.Int32ToInt(process.GetFdCount()),
		RSS:         process.GetRss(),
	}
}
