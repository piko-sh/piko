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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
)

func TestConvertHealthStatus_Simple(t *testing.T) {
	t.Parallel()

	status := monitoring_domain.HealthProbeStatus{
		Name:      "db",
		State:     "HEALTHY",
		Message:   "connected",
		Timestamp: 1234567890,
		Duration:  "2ms",
	}

	result := convertHealthStatus(status)

	assert.Equal(t, "db", result.Name)
	assert.Equal(t, "HEALTHY", result.State)
	assert.Equal(t, "connected", result.Message)
	assert.Equal(t, int64(1234567890), result.TimestampMs)
	assert.Equal(t, "2ms", result.Duration)
	assert.Empty(t, result.Dependencies)
}

func TestConvertHealthStatus_WithDependencies(t *testing.T) {
	t.Parallel()

	status := monitoring_domain.HealthProbeStatus{
		Name:      "overall",
		State:     "DEGRADED",
		Message:   "",
		Timestamp: 5000,
		Duration:  "10ms",
		Dependencies: []monitoring_domain.HealthProbeStatus{
			{
				Name:      "database",
				State:     "HEALTHY",
				Timestamp: 4000,
				Duration:  "1ms",
			},
			{
				Name:      "redis",
				State:     "UNHEALTHY",
				Message:   "connection refused",
				Timestamp: 4500,
				Duration:  "5ms",
			},
		},
	}

	result := convertHealthStatus(status)

	assert.Equal(t, "overall", result.Name)
	assert.Equal(t, "DEGRADED", result.State)
	require.Len(t, result.Dependencies, 2)
	assert.Equal(t, "database", result.Dependencies[0].Name)
	assert.Equal(t, "HEALTHY", result.Dependencies[0].State)
	assert.Equal(t, "redis", result.Dependencies[1].Name)
	assert.Equal(t, "UNHEALTHY", result.Dependencies[1].State)
	assert.Equal(t, "connection refused", result.Dependencies[1].Message)
}

func TestConvertHealthStatus_NestedDependencies(t *testing.T) {
	t.Parallel()

	status := monitoring_domain.HealthProbeStatus{
		Name:      "root",
		State:     "HEALTHY",
		Timestamp: 1000,
		Duration:  "50ms",
		Dependencies: []monitoring_domain.HealthProbeStatus{
			{
				Name:      "child",
				State:     "HEALTHY",
				Timestamp: 900,
				Duration:  "20ms",
				Dependencies: []monitoring_domain.HealthProbeStatus{
					{
						Name:      "grandchild",
						State:     "HEALTHY",
						Timestamp: 800,
						Duration:  "5ms",
					},
				},
			},
		},
	}

	result := convertHealthStatus(status)

	require.Len(t, result.Dependencies, 1)
	require.Len(t, result.Dependencies[0].Dependencies, 1)
	assert.Equal(t, "grandchild", result.Dependencies[0].Dependencies[0].Name)
}

func TestConvertHealthStatus_EmptyDependencies(t *testing.T) {
	t.Parallel()

	status := monitoring_domain.HealthProbeStatus{
		Name:         "probe",
		State:        "HEALTHY",
		Timestamp:    100,
		Duration:     "0ms",
		Dependencies: []monitoring_domain.HealthProbeStatus{},
	}

	result := convertHealthStatus(status)
	assert.Empty(t, result.Dependencies)
}

func TestEmptySystemStatsResponse(t *testing.T) {
	t.Parallel()

	response := emptySystemStatsResponse(time.Now())

	require.NotNil(t, response)
	assert.Nil(t, response.Build)
	assert.Nil(t, response.Runtime)
	assert.Nil(t, response.Gc)
	assert.Nil(t, response.Memory)
	assert.Nil(t, response.Process)
	assert.Greater(t, response.TimestampMs, int64(0))
	assert.Equal(t, int64(0), response.UptimeMs)
	assert.Equal(t, int64(0), response.NumCgoCalls)
	assert.Equal(t, float64(0), response.CpuMillicores)
	assert.Equal(t, int32(0), response.NumCpu)
	assert.Equal(t, int32(0), response.Gomaxprocs)
	assert.Equal(t, int32(0), response.NumGoroutines)
}

func TestConvertSystemStatsToPB(t *testing.T) {
	t.Parallel()

	stats := &monitoring_domain.SystemStats{
		TimestampMs:          1234567890,
		UptimeMs:             3600000,
		NumCPU:               8,
		GOMAXPROCS:           8,
		NumGoroutines:        100,
		NumCGOCalls:          42,
		CPUMillicores:        1500.5,
		SystemUptimeMs:       86400000,
		CgroupPath:           "/sys/fs/cgroup",
		MonitoringListenAddr: ":9090",
		Build: monitoring_domain.BuildInfo{
			GoVersion:     "go1.23.0",
			Version:       "1.0.0",
			Commit:        "abc123",
			BuildTime:     "2026-01-01T00:00:00Z",
			OS:            "linux",
			Arch:          "amd64",
			ModulePath:    "piko.sh/piko",
			ModuleVersion: "v1.0.0",
			VCSModified:   false,
			VCSTime:       "2026-01-01T00:00:00Z",
		},
		Runtime: monitoring_domain.RuntimeInfo{
			GOGC:       "100",
			GOMEMLIMIT: "1GiB",
			Compiler:   "gc",
		},
		GC: monitoring_domain.GCInfo{
			RecentPauses:  []uint64{100, 200, 300},
			LastGC:        999,
			PauseTotalNs:  600,
			LastPauseNs:   300,
			GCCPUFraction: 0.01,
			NextGC:        1048576,
			NumGC:         15,
			NumForcedGC:   2,
		},
		Memory: monitoring_domain.MemoryInfo{
			Alloc:        1024,
			TotalAlloc:   2048,
			Sys:          4096,
			HeapAlloc:    512,
			HeapSys:      2048,
			HeapIdle:     1024,
			HeapInuse:    1024,
			HeapObjects:  100,
			HeapReleased: 256,
			StackSys:     128,
			Mallocs:      500,
			Frees:        400,
			LiveObjects:  100,
			StackInuse:   64,
			MSpanInuse:   32,
			MSpanSys:     64,
			MCacheInuse:  16,
			MCacheSys:    32,
			GCSys:        256,
			OtherSys:     128,
			BuckHashSys:  64,
			Lookups:      10,
		},
		Process: monitoring_domain.ProcessInfo{
			PID:              1234,
			ThreadCount:      10,
			FDCount:          50,
			RSS:              1048576,
			Hostname:         "server-1",
			Executable:       "/usr/bin/piko",
			CWD:              "/app",
			UID:              1000,
			GID:              1000,
			MaxOpenFilesSoft: 1024,
			MaxOpenFilesHard: 65536,
			IoReadBytes:      5000,
			IoWriteBytes:     3000,
			IoRchar:          10000,
			IoWchar:          8000,
			PPID:             1,
		},
	}

	result := convertSystemStatsToPB(stats)

	require.NotNil(t, result)

	assert.Equal(t, int64(1234567890), result.TimestampMs)
	assert.Equal(t, int64(3600000), result.UptimeMs)
	assert.Equal(t, int32(8), result.NumCpu)
	assert.Equal(t, int32(8), result.Gomaxprocs)
	assert.Equal(t, int32(100), result.NumGoroutines)
	assert.Equal(t, int64(42), result.NumCgoCalls)
	assert.InDelta(t, 1500.5, result.CpuMillicores, 1e-9)
	assert.Equal(t, int64(86400000), result.SystemUptimeMs)
	assert.Equal(t, "/sys/fs/cgroup", result.CgroupPath)
	assert.Equal(t, ":9090", result.MonitoringListenAddr)

	require.NotNil(t, result.Build)
	assert.Equal(t, "go1.23.0", result.Build.GoVersion)
	assert.Equal(t, "1.0.0", result.Build.Version)
	assert.Equal(t, "abc123", result.Build.Commit)
	assert.Equal(t, "linux", result.Build.Os)
	assert.Equal(t, "amd64", result.Build.Arch)
	assert.Equal(t, "piko.sh/piko", result.Build.ModulePath)
	assert.Equal(t, "v1.0.0", result.Build.ModuleVersion)
	assert.False(t, result.Build.VcsModified)

	require.NotNil(t, result.Runtime)
	assert.Equal(t, "100", result.Runtime.Gogc)
	assert.Equal(t, "1GiB", result.Runtime.Gomemlimit)
	assert.Equal(t, "gc", result.Runtime.Compiler)

	require.NotNil(t, result.Gc)
	assert.Equal(t, []uint64{100, 200, 300}, result.Gc.RecentPauses)
	assert.Equal(t, int64(999), result.Gc.LastGcNs)
	assert.Equal(t, uint64(600), result.Gc.PauseTotalNs)
	assert.Equal(t, uint64(300), result.Gc.LastPauseNs)
	assert.InDelta(t, 0.01, result.Gc.GcCpuFraction, 1e-9)
	assert.Equal(t, uint64(1048576), result.Gc.NextGc)
	assert.Equal(t, uint32(15), result.Gc.NumGc)
	assert.Equal(t, uint32(2), result.Gc.NumForcedGc)

	require.NotNil(t, result.Memory)
	assert.Equal(t, uint64(1024), result.Memory.Alloc)
	assert.Equal(t, uint64(2048), result.Memory.TotalAlloc)
	assert.Equal(t, uint64(4096), result.Memory.Sys)
	assert.Equal(t, uint64(512), result.Memory.HeapAlloc)
	assert.Equal(t, uint64(2048), result.Memory.HeapSys)
	assert.Equal(t, uint64(1024), result.Memory.HeapIdle)
	assert.Equal(t, uint64(1024), result.Memory.HeapInuse)
	assert.Equal(t, uint64(100), result.Memory.HeapObjects)
	assert.Equal(t, uint64(256), result.Memory.HeapReleased)
	assert.Equal(t, uint64(128), result.Memory.StackSys)
	assert.Equal(t, uint64(500), result.Memory.Mallocs)
	assert.Equal(t, uint64(400), result.Memory.Frees)
	assert.Equal(t, uint64(100), result.Memory.LiveObjects)
	assert.Equal(t, uint64(64), result.Memory.StackInuse)
	assert.Equal(t, uint64(32), result.Memory.MspanInuse)
	assert.Equal(t, uint64(64), result.Memory.MspanSys)
	assert.Equal(t, uint64(16), result.Memory.McacheInuse)
	assert.Equal(t, uint64(32), result.Memory.McacheSys)
	assert.Equal(t, uint64(256), result.Memory.GcSys)
	assert.Equal(t, uint64(128), result.Memory.OtherSys)
	assert.Equal(t, uint64(64), result.Memory.BuckhashSys)
	assert.Equal(t, uint64(10), result.Memory.Lookups)

	require.NotNil(t, result.Process)
	assert.Equal(t, int32(1234), result.Process.Pid)
	assert.Equal(t, int32(10), result.Process.ThreadCount)
	assert.Equal(t, int32(50), result.Process.FdCount)
	assert.Equal(t, uint64(1048576), result.Process.Rss)
	assert.Equal(t, "server-1", result.Process.Hostname)
	assert.Equal(t, "/usr/bin/piko", result.Process.Executable)
	assert.Equal(t, "/app", result.Process.Cwd)
	assert.Equal(t, int32(1000), result.Process.Uid)
	assert.Equal(t, int32(1000), result.Process.Gid)
	assert.Equal(t, int64(1024), result.Process.MaxOpenFilesSoft)
	assert.Equal(t, int64(65536), result.Process.MaxOpenFilesHard)
	assert.Equal(t, uint64(5000), result.Process.IoReadBytes)
	assert.Equal(t, uint64(3000), result.Process.IoWriteBytes)
	assert.Equal(t, uint64(10000), result.Process.IoRchar)
	assert.Equal(t, uint64(8000), result.Process.IoWchar)
	assert.Equal(t, int32(1), result.Process.Ppid)
}

func TestConvertBuildInfoToPB(t *testing.T) {
	t.Parallel()

	build := monitoring_domain.BuildInfo{
		GoVersion:     "go1.22.5",
		Version:       "2.0.0",
		Commit:        "def456",
		BuildTime:     "2026-02-01",
		OS:            "darwin",
		Arch:          "arm64",
		ModulePath:    "example.com/mod",
		ModuleVersion: "v2.0.0",
		VCSModified:   true,
		VCSTime:       "2026-02-01T10:00:00Z",
	}

	result := convertBuildInfoToPB(build)

	require.NotNil(t, result)
	assert.Equal(t, "go1.22.5", result.GoVersion)
	assert.Equal(t, "2.0.0", result.Version)
	assert.Equal(t, "def456", result.Commit)
	assert.Equal(t, "2026-02-01", result.BuildTime)
	assert.Equal(t, "darwin", result.Os)
	assert.Equal(t, "arm64", result.Arch)
	assert.Equal(t, "example.com/mod", result.ModulePath)
	assert.Equal(t, "v2.0.0", result.ModuleVersion)
	assert.True(t, result.VcsModified)
	assert.Equal(t, "2026-02-01T10:00:00Z", result.VcsTime)
}

func TestConvertRuntimeInfoToPB(t *testing.T) {
	t.Parallel()

	rt := monitoring_domain.RuntimeInfo{
		GOGC:       "off",
		GOMEMLIMIT: "512MiB",
		Compiler:   "gc",
	}

	result := convertRuntimeInfoToPB(rt)

	require.NotNil(t, result)
	assert.Equal(t, "off", result.Gogc)
	assert.Equal(t, "512MiB", result.Gomemlimit)
	assert.Equal(t, "gc", result.Compiler)
}

func TestConvertGCInfoToPB(t *testing.T) {
	t.Parallel()

	gc := monitoring_domain.GCInfo{
		RecentPauses:  []uint64{50, 100},
		LastGC:        500,
		PauseTotalNs:  150,
		LastPauseNs:   100,
		GCCPUFraction: 0.02,
		NextGC:        2097152,
		NumGC:         25,
		NumForcedGC:   5,
	}

	result := convertGCInfoToPB(gc)

	require.NotNil(t, result)
	assert.Equal(t, []uint64{50, 100}, result.RecentPauses)
	assert.Equal(t, int64(500), result.LastGcNs)
	assert.Equal(t, uint64(150), result.PauseTotalNs)
	assert.Equal(t, uint64(100), result.LastPauseNs)
	assert.InDelta(t, 0.02, result.GcCpuFraction, 1e-9)
	assert.Equal(t, uint64(2097152), result.NextGc)
	assert.Equal(t, uint32(25), result.NumGc)
	assert.Equal(t, uint32(5), result.NumForcedGc)
}

func TestConvertMemoryInfoToPB(t *testing.T) {
	t.Parallel()

	mem := monitoring_domain.MemoryInfo{
		Alloc:   100,
		Sys:     200,
		Mallocs: 50,
		Frees:   30,
	}

	result := convertMemoryInfoToPB(mem)

	require.NotNil(t, result)
	assert.Equal(t, uint64(100), result.Alloc)
	assert.Equal(t, uint64(200), result.Sys)
	assert.Equal(t, uint64(50), result.Mallocs)
	assert.Equal(t, uint64(30), result.Frees)
}

func TestConvertProcessInfoToPB(t *testing.T) {
	t.Parallel()

	proc := monitoring_domain.ProcessInfo{
		PID:              999,
		ThreadCount:      5,
		FDCount:          20,
		RSS:              524288,
		Hostname:         "worker",
		Executable:       "/bin/app",
		CWD:              "/home/app",
		UID:              500,
		GID:              500,
		MaxOpenFilesSoft: 1024,
		MaxOpenFilesHard: 4096,
		IoReadBytes:      1000,
		IoWriteBytes:     2000,
		IoRchar:          3000,
		IoWchar:          4000,
		PPID:             42,
	}

	result := convertProcessInfoToPB(proc)

	require.NotNil(t, result)
	assert.Equal(t, int32(999), result.Pid)
	assert.Equal(t, int32(5), result.ThreadCount)
	assert.Equal(t, int32(20), result.FdCount)
	assert.Equal(t, uint64(524288), result.Rss)
	assert.Equal(t, "worker", result.Hostname)
	assert.Equal(t, "/bin/app", result.Executable)
	assert.Equal(t, "/home/app", result.Cwd)
	assert.Equal(t, int32(500), result.Uid)
	assert.Equal(t, int32(500), result.Gid)
	assert.Equal(t, int64(1024), result.MaxOpenFilesSoft)
	assert.Equal(t, int64(4096), result.MaxOpenFilesHard)
	assert.Equal(t, uint64(1000), result.IoReadBytes)
	assert.Equal(t, uint64(2000), result.IoWriteBytes)
	assert.Equal(t, uint64(3000), result.IoRchar)
	assert.Equal(t, uint64(4000), result.IoWchar)
	assert.Equal(t, int32(42), result.Ppid)
}

func TestConvertSystemStatsToPB_ZeroValues(t *testing.T) {
	t.Parallel()

	stats := &monitoring_domain.SystemStats{}
	result := convertSystemStatsToPB(stats)

	require.NotNil(t, result)
	require.NotNil(t, result.Build)
	require.NotNil(t, result.Runtime)
	require.NotNil(t, result.Gc)
	require.NotNil(t, result.Memory)
	require.NotNil(t, result.Process)
	assert.Equal(t, int64(0), result.TimestampMs)
	assert.Equal(t, int64(0), result.UptimeMs)
	assert.Equal(t, int32(0), result.NumCpu)
}
