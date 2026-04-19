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
	"time"

	tea "charm.land/bubbletea/v2"
)

// Provider defines the base interface for all data providers.
// Providers fetch data from external sources such as databases and APIs
// and make it available to panels for display.
type Provider interface {
	// Name returns a readable name for this provider.
	//
	// Returns string which identifies this provider in logs and UI.
	Name() string

	// Health checks whether the provider is connected and working.
	//
	// Returns error when the connection is not healthy.
	Health(ctx context.Context) error

	// Close releases any resources held by the provider.
	//
	// Returns error when the close operation fails.
	Close() error
}

// RefreshableProvider extends Provider with support for periodic data refresh.
// Most providers implement it to allow automatic data updates.
type RefreshableProvider interface {
	Provider

	// Refresh fetches the latest data from the data source.
	// Called periodically by the TUI service.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns error when the refresh operation fails.
	Refresh(ctx context.Context) error

	// RefreshInterval returns how often this provider should refresh.
	// The TUI service uses this to schedule refresh operations.
	//
	// Returns time.Duration which specifies the time between refreshes.
	RefreshInterval() time.Duration
}

// MetricsProvider retrieves metrics data from backends such as Piko's OTEL
// endpoint or Prometheus. It implements tui_domain.MetricsProvider.
type MetricsProvider interface {
	RefreshableProvider

	// ListMetrics returns available metric names.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns []string which contains all available metric names.
	// Returns error when the listing fails.
	ListMetrics(ctx context.Context) ([]string, error)

	// Query fetches a metric series for the given time range.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes metric (string) which identifies the metric to query.
	// Takes start (time.Time) which specifies the start of the range.
	// Takes end (time.Time) which specifies the end of the range.
	//
	// Returns *MetricSeries which contains the queried data.
	// Returns error when the query fails.
	Query(ctx context.Context, metric string, start, end time.Time) (*MetricSeries, error)

	// Current fetches the current value of a metric.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes metric (string) which identifies the metric to fetch.
	//
	// Returns *MetricValue which contains the current value.
	// Returns error when the fetch fails.
	Current(ctx context.Context, metric string) (*MetricValue, error)
}

// TracesProvider retrieves trace data from observability backends.
// It implements tui_domain.TracesProvider for displaying traces in the UI.
type TracesProvider interface {
	RefreshableProvider

	// Recent fetches the N most recent traces.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes limit (int) which specifies the maximum traces to return.
	//
	// Returns []Span which contains the recent trace spans.
	// Returns error when the fetch fails.
	Recent(ctx context.Context, limit int) ([]Span, error)

	// Errors fetches recent error spans.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes limit (int) which specifies the maximum spans to return.
	//
	// Returns []Span which contains the error spans.
	// Returns error when the fetch fails.
	Errors(ctx context.Context, limit int) ([]Span, error)

	// Get fetches all spans for a specific trace.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes traceID (string) which identifies the trace.
	//
	// Returns []Span which contains all spans in the trace.
	// Returns error when the trace cannot be found.
	Get(ctx context.Context, traceID string) ([]Span, error)
}

// ResourceProvider gives access to application resources from databases.
// It is the main provider for registry artefacts and orchestrator tasks,
// and implements the tui_domain.ResourceProvider interface.
type ResourceProvider interface {
	RefreshableProvider

	// List fetches all resources of a given kind.
	//
	// Takes kind (string) which identifies the resource type (e.g., "artefact",
	// "task").
	//
	// Returns []Resource which contains the resources.
	// Returns error when the list fails.
	List(ctx context.Context, kind string) ([]Resource, error)

	// Kinds returns the resource kinds that this provider supports.
	//
	// Returns []string which lists the kinds this provider can handle.
	Kinds() []string

	// Summary returns aggregate counts by status for each kind.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns map[string]map[ResourceStatus]int with counts per kind and status.
	// Returns error when the summary cannot be generated.
	Summary(ctx context.Context) (map[string]map[ResourceStatus]int, error)
}

// HealthProvider provides access to health check data from application
// endpoints. It returns liveness and readiness probe status for monitoring.
type HealthProvider interface {
	RefreshableProvider

	// Liveness fetches the current liveness status.
	// Liveness indicates if the application is running and not deadlocked.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *HealthStatus which contains the liveness check result.
	// Returns error when the check cannot be performed.
	Liveness(ctx context.Context) (*HealthStatus, error)

	// Readiness fetches the current readiness status.
	// Readiness indicates if the application is ready to serve traffic.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *HealthStatus which contains the readiness check result.
	// Returns error when the check cannot be performed.
	Readiness(ctx context.Context) (*HealthStatus, error)
}

// SystemProvider provides access to runtime system statistics from the server.
// It extends RefreshableProvider to add CPU, memory, goroutine, and GC stats.
type SystemProvider interface {
	RefreshableProvider

	// GetStats returns the current system statistics.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *SystemStats which contains the current stats.
	// Returns error when the stats cannot be fetched.
	GetStats(ctx context.Context) (*SystemStats, error)
}

// SystemStats holds runtime statistics collected from the server.
type SystemStats struct {
	// Timestamp is when the statistics were collected.
	Timestamp time.Time

	// Build holds version and compile-time details such as Go version, commit
	// hash, and build timestamp.
	Build SystemBuildInfo

	// Runtime holds Go runtime settings such as GOGC and GOMEMLIMIT.
	Runtime SystemRuntimeConfig

	// GC holds garbage collection statistics.
	GC SystemGCStats

	// Memory holds Go runtime memory statistics.
	Memory SystemMemoryStats

	// Process holds metrics about the current process such as PID, thread count,
	// file descriptor count, and memory usage.
	Process SystemProcessInfo

	// Uptime is how long the application has been running.
	Uptime time.Duration

	// NumCGOCalls is the total number of CGO calls made by the process.
	NumCGOCalls int64

	// CPUMillicores is the CPU usage in millicores; 1000 equals one full core.
	CPUMillicores float64

	// NumCPU is the number of logical CPUs available to this process.
	NumCPU int

	// GOMAXPROCS is the number of CPUs that can run goroutines at the same time.
	GOMAXPROCS int

	// NumGoroutines is the current number of goroutines.
	NumGoroutines int

	// Cache holds render cache statistics from the server.
	Cache SystemCacheStats
}

// SystemCacheStats holds render cache statistics from the server.
type SystemCacheStats struct {
	// ComponentCacheSize is the number of entries in the component metadata cache.
	ComponentCacheSize int

	// SVGCacheSize is the number of entries in the SVG asset cache.
	SVGCacheSize int
}

// SystemMemoryStats holds memory usage data from the server.
type SystemMemoryStats struct {
	// Alloc is the bytes of heap memory currently in use.
	Alloc uint64

	// TotalAlloc is the total bytes allocated since the process started.
	TotalAlloc uint64

	// Sys is the total bytes of memory obtained from the OS by the Go runtime.
	Sys uint64

	// HeapAlloc is the number of bytes of allocated heap objects.
	HeapAlloc uint64

	// HeapSys is the total heap memory in bytes that the runtime has got from the OS.
	HeapSys uint64

	// HeapIdle is the number of bytes in heap spans that are not in use.
	HeapIdle uint64

	// HeapInuse is the number of bytes in heap spans currently in use.
	HeapInuse uint64

	// HeapObjects is the number of allocated heap objects.
	HeapObjects uint64

	// HeapReleased is the number of bytes of heap memory returned to the OS.
	HeapReleased uint64

	// StackSys is the total stack memory in bytes obtained from the OS.
	StackSys uint64

	// Mallocs is the total number of heap objects allocated over time.
	Mallocs uint64

	// Frees is the total number of heap objects freed since the program started.
	Frees uint64

	// LiveObjects is the count of heap objects currently in use.
	LiveObjects uint64
}

// SystemGCStats holds garbage collection statistics from the server.
type SystemGCStats struct {
	// RecentPauses holds the most recent GC pause times in nanoseconds.
	RecentPauses []uint64

	// LastGC is the Unix timestamp in nanoseconds of the last garbage collection.
	LastGC int64

	// PauseTotalNs is the total time spent in GC pauses, in nanoseconds.
	PauseTotalNs uint64

	// LastPauseNs is the duration of the last garbage collection pause in nanoseconds.
	LastPauseNs uint64

	// GCCPUFraction is the share of CPU time used by the garbage collector.
	GCCPUFraction float64

	// NextGC is the heap size target in bytes for the next garbage collection.
	NextGC uint64

	// NumGC is the total number of completed garbage collection cycles.
	NumGC uint32
}

// SystemBuildInfo holds build details from the server.
type SystemBuildInfo struct {
	// GoVersion is the version of Go used to build the application.
	GoVersion string

	// Version is the application version string.
	Version string

	// Commit is the Git commit hash for this build.
	Commit string

	// BuildTime is when the binary was built, in RFC3339 format.
	BuildTime string

	// OS is the operating system the binary was built for.
	OS string

	// Arch is the target architecture (e.g. amd64, arm64).
	Arch string
}

// SystemProcessInfo holds process details from the operating system.
type SystemProcessInfo struct {
	// PID is the process identifier of the running application.
	PID int

	// ThreadCount is the number of threads the process is using.
	ThreadCount int

	// FDCount is the number of open file descriptors.
	FDCount int

	// RSS is the resident set size in bytes.
	RSS uint64
}

// SystemRuntimeConfig holds runtime configuration values from the server.
type SystemRuntimeConfig struct {
	// GOGC is the current value of the GOGC environment variable.
	GOGC string

	// GOMEMLIMIT is the memory limit for the Go runtime.
	GOMEMLIMIT string
}

// FDsProvider fetches file descriptor data from the Piko server. It gives
// visibility into open files, sockets, and other OS resources.
type FDsProvider interface {
	RefreshableProvider

	// GetFDs returns the current file descriptor information.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *FDsData which contains the current FD info.
	// Returns error when the FDs cannot be fetched.
	GetFDs(ctx context.Context) (*FDsData, error)
}

// FDsData holds file descriptor data returned from the server.
type FDsData struct {
	// Categories holds the file descriptor data grouped by type.
	Categories []FDCategory

	// Total is the count of file descriptors in use.
	Total int

	// Timestamp is the Unix time in seconds when the data was created.
	Timestamp int64
}

// FDCategory represents a group of file descriptors that share a common type.
type FDCategory struct {
	// Category is the name that groups related file descriptors together.
	Category string

	// FDs holds the file descriptors that belong to this category.
	FDs []FDInfo

	// Count is the number of file descriptors in this category.
	Count int
}

// FDInfo represents details about a single file descriptor.
type FDInfo struct {
	// Category classifies the file descriptor type.
	Category string

	// Target is the path or address the file descriptor points to.
	Target string

	// FirstSeen is when this file descriptor was first observed, as Unix
	// milliseconds.
	FirstSeen int64

	// AgeMs is how long the file descriptor has been open, in milliseconds.
	AgeMs int64

	// FD is the file descriptor number.
	FD int
}

// Panel represents a discrete UI section that can be focused and rendered.
// Panels follow the bubbletea model pattern for state management.
type Panel interface {
	// ID returns a unique identifier for this panel.
	// Used for panel switching and configuration.
	//
	// Returns string which uniquely identifies this panel.
	ID() string

	// Title returns the display title shown in the UI.
	//
	// Returns string which is the text shown in the panel header.
	Title() string

	// Init initialises the panel and returns any initial commands.
	// Called once when the panel is first created.
	//
	// Returns tea.Cmd which may contain async initialisation work.
	Init() tea.Cmd

	// Update handles messages and returns updated panel and commands.
	// Only called when this panel is focused.
	//
	// Takes message (tea.Msg) which contains the message to handle.
	//
	// Returns Panel which is the updated panel state.
	// Returns tea.Cmd which contains any commands to execute.
	Update(message tea.Msg) (Panel, tea.Cmd)

	// View renders the panel within the given dimensions.
	//
	// Takes width (int) which specifies the available width in characters.
	// Takes height (int) which specifies the available height in lines.
	//
	// Returns string which contains the rendered view.
	View(width, height int) string

	// Focused reports whether this panel has focus.
	//
	// Returns bool which is true if the panel has focus.
	Focused() bool

	// SetFocused sets the focus state of the panel.
	// Called by the TUI service when focus changes.
	//
	// Takes focused (bool) which specifies the new focus state.
	SetFocused(focused bool)

	// KeyMap returns panel-specific keybindings for the help view.
	//
	// Returns []KeyBinding which contains the panel's keybindings.
	KeyMap() []KeyBinding

	// DetailView renders the right-hand detail body for the panel.
	//
	// Panels with no per-row detail return the empty string; the
	// composer falls back to a placeholder hint in that case. The
	// returned string must be sized to (width, height).
	//
	// Takes width (int) and height (int) for the inner content area.
	//
	// Returns string with the rendered detail body, or "" to opt out.
	DetailView(width, height int) string

	// Selection returns what is currently selected in the panel.
	//
	// The composer hands this to other panels for cross-panel
	// coordination (e.g. trace -> route navigation). Panels with no
	// selectable rows return Selection{}.
	//
	// Returns Selection describing the focused row, or empty.
	Selection() Selection
}

// KeyBinding describes a keyboard shortcut and its action for display in help.
type KeyBinding struct {
	// Key is the key or key combination, such as "j", "Ctrl+C", or "Enter".
	Key string

	// Description is the text shown to explain what the key does.
	Description string
}
