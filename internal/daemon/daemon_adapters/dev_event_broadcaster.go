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

package daemon_adapters

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

const (
	// devSSEClientBuffer is the per-client channel buffer size. Messages are
	// silently dropped when a client's buffer is full (the client is likely
	// dead and will be cleaned up when its context is cancelled).
	devSSEClientBuffer = 16

	// devSSEKeepaliveInterval is the interval between keepalive pings sent
	// to each SSE client to prevent proxy/load-balancer timeouts.
	devSSEKeepaliveInterval = 15 * time.Second

	// devSSEStatsInterval is how often system-stats events are pushed to
	// connected dev tools clients.
	devSSEStatsInterval = 5 * time.Second

	// devSSEFullStateInterval is how often health, resources, providers, and
	// memory-detail events are pushed. These change less frequently than
	// system-stats so a longer interval avoids unnecessary traffic.
	devSSEFullStateInterval = 10 * time.Second

	// devSSERecentTaskLimit is the maximum number of recent tasks to include in
	// build update events.
	devSSERecentTaskLimit = 10

	// devSSEWorkflowSummaryLimit is the maximum number of workflow summaries to
	// include in build update events.
	devSSEWorkflowSummaryLimit = 5
)

// DevBuildEvent is the payload sent to connected browsers when a dev build
// completes.
type DevBuildEvent struct {
	// Type is the event type, e.g. "rebuild-complete".
	Type string `json:"type"`

	// AffectedRoutes holds the URL route patterns that were
	// rebuilt, such as "/login" or "/dashboard"; an entry of
	// "*" means all routes.
	AffectedRoutes []string `json:"affectedRoutes"`

	// TimestampMs is the Unix timestamp in milliseconds when the event was
	// created.
	TimestampMs int64 `json:"timestampMs"`
}

// DevEventBroadcaster manages SSE connections for dev-mode build events.
// It implements http.Handler (for the /_piko/dev/events endpoint) and
// lifecycle_domain.DevEventNotifier (for receiving build-complete signals).
type DevEventBroadcaster struct {
	// systemStats provides system metrics for periodic SSE push. Nil disables
	// the system-stats event.
	systemStats monitoring_domain.SystemStatsProvider

	// healthProbe provides liveness/readiness probe results for SSE push.
	healthProbe monitoring_domain.HealthProbeService

	// resources provides file descriptor / resource data for SSE push.
	resources monitoring_domain.ResourceProvider

	// providerInfo provides resource type / provider discovery for SSE push.
	providerInfo monitoring_domain.ProviderInfoInspector

	// orchestrator provides build pipeline state for SSE push.
	orchestrator orchestrator_domain.OrchestratorInspector

	// statsCancel stops the periodic system stats goroutine.
	statsCancel context.CancelFunc

	// clients holds the set of connected SSE client channels.
	clients map[chan []byte]struct{}

	// mu guards clients, closed, and statsRunning fields.
	mu sync.RWMutex

	// closed indicates whether the broadcaster has been shut down.
	closed bool

	// statsRunning tracks whether the periodic stats goroutine is active.
	statsRunning bool
}

// NewDevEventBroadcaster creates a new broadcaster ready to accept SSE
// clients.
//
// Returns *DevEventBroadcaster which is the initialised broadcaster.
func NewDevEventBroadcaster() *DevEventBroadcaster {
	return &DevEventBroadcaster{
		clients: make(map[chan []byte]struct{}),
	}
}

// SetSystemStatsProvider configures the provider used to push periodic
// system-stats SSE events. Must be called before the first client connects.
//
// Takes p (monitoring_domain.SystemStatsProvider) which supplies system metrics.
//
// Safe for concurrent use; guarded by mu.
func (b *DevEventBroadcaster) SetSystemStatsProvider(p monitoring_domain.SystemStatsProvider) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.systemStats = p
}

// SetHealthProbeService configures the health probe provider for SSE push.
//
// Takes p (monitoring_domain.HealthProbeService) which
// supplies liveness and readiness probes.
//
// Safe for concurrent use; guarded by mu.
func (b *DevEventBroadcaster) SetHealthProbeService(p monitoring_domain.HealthProbeService) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.healthProbe = p
}

// SetResourceProvider configures the resource/FD provider for SSE push.
//
// Takes p (monitoring_domain.ResourceProvider) which supplies
// file descriptor and resource data.
//
// Safe for concurrent use; guarded by mu.
func (b *DevEventBroadcaster) SetResourceProvider(p monitoring_domain.ResourceProvider) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.resources = p
}

// SetProviderInfoInspector configures the provider info inspector for SSE push.
//
// Takes p (monitoring_domain.ProviderInfoInspector) which
// supplies resource type and provider discovery.
//
// Safe for concurrent use; guarded by mu.
func (b *DevEventBroadcaster) SetProviderInfoInspector(p monitoring_domain.ProviderInfoInspector) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.providerInfo = p
}

// SetOrchestratorInspector configures the orchestrator inspector for SSE push.
//
// Takes p (orchestrator_domain.OrchestratorInspector)
// which supplies build pipeline state.
//
// Safe for concurrent use; guarded by mu.
func (b *DevEventBroadcaster) SetOrchestratorInspector(p orchestrator_domain.OrchestratorInspector) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.orchestrator = p
}

// ServeHTTP implements http.Handler. It upgrades the connection to an SSE
// stream, sends an initial heartbeat, and blocks until the client disconnects
// or the broadcaster is closed.
//
// Takes w (http.ResponseWriter) which is the response writer for the SSE stream.
// Takes r (*http.Request) which is the incoming HTTP request.
func (b *DevEventBroadcaster) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprint(w, "retry: 2000\nevent: connected\ndata: {}\n\n")
	flusher.Flush()

	b.writeInitialState(w, flusher)

	ch := make(chan []byte, devSSEClientBuffer)
	b.addClient(ch)
	defer b.removeClient(ch)

	keepalive := time.NewTicker(devSSEKeepaliveInterval)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		case <-keepalive.C:
			_, _ = fmt.Fprint(w, "event: ping\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}

// Broadcast sends an event to all connected SSE clients. Clients whose
// buffers are full have the message silently dropped.
//
// Takes event (DevBuildEvent) which holds the event payload to broadcast.
//
// Safe for concurrent use; reads the client set under mu.RLock.
func (b *DevEventBroadcaster) Broadcast(event DevBuildEvent) {
	if b == nil {
		return
	}

	data, err := devJSON.Marshal(event)
	if err != nil {
		return
	}

	msg := fmt.Appendf(nil, "event: %s\ndata: %s\n\n", event.Type, data)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// NotifyRebuildComplete implements lifecycle_domain.DevEventNotifier. It
// converts component-relative paths to URL route patterns and broadcasts
// a "rebuild-complete" event to all connected browsers.
//
// Takes affectedPaths ([]string) which lists the component-relative paths that changed.
func (b *DevEventBroadcaster) NotifyRebuildComplete(_ context.Context, affectedPaths []string) {
	routes := make([]string, 0, len(affectedPaths))
	for _, p := range affectedPaths {
		if route, ok := pathToRoute(p); ok {
			routes = append(routes, route)
		}
	}

	if len(routes) == 0 {
		routes = []string{"*"}
	}

	b.Broadcast(DevBuildEvent{
		Type:           "rebuild-complete",
		AffectedRoutes: routes,
		TimestampMs:    time.Now().UnixMilli(),
	})

	b.broadcastEvent("template-changed", DevTemplateChangedEvent{
		AffectedPaths: affectedPaths,
		TimestampMs:   time.Now().UnixMilli(),
	})
}

// DevTemplateChangedEvent is broadcast alongside rebuild-complete to notify
// preview tabs about non-page template changes (emails, PDFs, partials).
type DevTemplateChangedEvent struct {
	// AffectedPaths lists the raw source paths that changed (e.g.,
	// "emails/welcome.pk", "pdfs/invoice.pk").
	AffectedPaths []string `json:"affectedPaths"`

	// TimestampMs is when the rebuild completed.
	TimestampMs int64 `json:"timestampMs"`
}

// DevBuildSummary is pushed via SSE after each build completes, alongside
// the existing rebuild-complete event.
type DevBuildSummary struct {
	// ComponentCount is the number of components built.
	ComponentCount int `json:"componentCount"`

	// DurationMs is the build duration in milliseconds.
	DurationMs int64 `json:"durationMs"`

	// ErrorCount is the number of build errors.
	ErrorCount int `json:"errorCount"`

	// TimestampMs is when the build completed.
	TimestampMs int64 `json:"timestampMs"`
}

// BroadcastBuildSummary sends a build-summary event to all connected SSE
// clients.
//
// Takes summary (DevBuildSummary) which holds the build summary payload.
//
// Safe for concurrent use; reads the client set under mu.RLock.
func (b *DevEventBroadcaster) BroadcastBuildSummary(summary DevBuildSummary) {
	if b == nil {
		return
	}

	data, err := devJSON.Marshal(summary)
	if err != nil {
		return
	}
	msg := fmt.Appendf(nil, "event: build-summary\ndata: %s\n\n", data)

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Close shuts down the broadcaster and closes all client channels.
//
// Safe for concurrent use; acquires mu.Lock to mark closed and drain clients.
func (b *DevEventBroadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}
	b.closed = true

	if b.statsCancel != nil {
		b.statsCancel()
	}

	for ch := range b.clients {
		close(ch)
		delete(b.clients, ch)
	}
}

// ClientCount returns the number of currently connected SSE clients.
// Primarily useful for testing.
//
// Returns int which is the number of active client connections.
//
// Safe for concurrent use; reads under mu.RLock.
func (b *DevEventBroadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// addClient registers a new SSE client channel and starts the periodic
// stats goroutine if this is the first client.
//
// Takes ch (chan []byte) which is the client's message channel.
//
// Safe for concurrent use; acquires mu.Lock to insert the client.
func (b *DevEventBroadcaster) addClient(ch chan []byte) {
	b.mu.Lock()
	if !b.closed {
		b.clients[ch] = struct{}{}
	}
	if !b.statsRunning && b.systemStats != nil && len(b.clients) > 0 {
		b.startPeriodicStatsLocked()
	}
	b.mu.Unlock()
}

// removeClient unregisters an SSE client channel and stops the periodic
// stats goroutine when no clients remain.
//
// Takes ch (chan []byte) which is the client's message channel to remove.
//
// Safe for concurrent use; acquires mu.Lock to delete the client.
func (b *DevEventBroadcaster) removeClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	if b.statsRunning && len(b.clients) == 0 {
		b.stopPeriodicStatsLocked()
	}
	b.mu.Unlock()
}

// startPeriodicStatsLocked launches a goroutine that pushes system-stats
// events every 5 seconds and full-state events (health, resources, providers,
// memory) every 10 seconds while clients are connected.
//
// Must be called with b.mu held.
func (b *DevEventBroadcaster) startPeriodicStatsLocked() {
	if b.statsRunning {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.statsCancel = cancel
	b.statsRunning = true
	provider := b.systemStats

	go func() {
		statsTicker := time.NewTicker(devSSEStatsInterval)
		fullStateTicker := time.NewTicker(devSSEFullStateInterval)
		defer statsTicker.Stop()
		defer fullStateTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-statsTicker.C:
				b.broadcastSystemStats(provider)
			case <-fullStateTicker.C:
				b.broadcastFullState(ctx, provider)
			}
		}
	}()
}

// stopPeriodicStatsLocked stops the periodic system stats goroutine.
//
// Must be called with b.mu held.
func (b *DevEventBroadcaster) stopPeriodicStatsLocked() {
	if b.statsCancel != nil {
		b.statsCancel()
		b.statsCancel = nil
	}
	b.statsRunning = false
}

// broadcastSystemStats sends a system-stats SSE event with current metrics.
//
// Takes provider (monitoring_domain.SystemStatsProvider)
// which supplies the current system metrics.
//
// Safe for concurrent use; reads the client set under
// mu.RLock.
func (b *DevEventBroadcaster) broadcastSystemStats(provider monitoring_domain.SystemStatsProvider) {
	stats := provider.GetStats()

	trimmed := map[string]any{
		"heapAlloc":     stats.Memory.HeapAlloc,
		"heapSys":       stats.Memory.HeapSys,
		"goroutines":    stats.NumGoroutines,
		"cpuMillicores": stats.CPUMillicores,
		"uptimeMs":      stats.UptimeMs,
		"gcPauseNs":     stats.GC.LastPauseNs,
		"numGC":         stats.GC.NumGC,
		"timestampMs":   stats.TimestampMs,
	}
	data, err := devJSON.Marshal(trimmed)
	if err != nil {
		return
	}
	msg := fmt.Appendf(nil, "event: system-stats\ndata: %s\n\n", data)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// broadcastFullState pushes build, health, resources, providers, and
// memory-detail SSE events to all connected clients. Each event is
// independent so a nil provider simply skips that event.
//
// Takes statsProvider
// (monitoring_domain.SystemStatsProvider) which supplies
// system metrics for memory-detail events.
//
// Safe for concurrent use; reads provider fields under mu.RLock.
func (b *DevEventBroadcaster) broadcastFullState(ctx context.Context, statsProvider monitoring_domain.SystemStatsProvider) {
	b.mu.RLock()
	hp := b.healthProbe
	rp := b.resources
	pi := b.providerInfo
	orch := b.orchestrator
	b.mu.RUnlock()

	if orch != nil {
		b.broadcastBuildUpdate(ctx, orch)
	}
	if hp != nil {
		b.broadcastHealthUpdate(ctx, hp)
	}
	if rp != nil {
		b.broadcastResourcesUpdate(rp)
	}
	if pi != nil {
		b.broadcastProvidersUpdate(ctx, pi)
	}
	if statsProvider != nil {
		b.broadcastMemoryDetail(statsProvider)
	}
}

// writeInitialState sends the current full state directly to a newly connected
// client's response writer so all tabs are populated immediately.
//
// Takes w (http.ResponseWriter) which is the client's response writer.
// Takes flusher (http.Flusher) which flushes buffered data to the client.
//
// Safe for concurrent use; reads provider fields under mu.RLock.
func (b *DevEventBroadcaster) writeInitialState(w http.ResponseWriter, flusher http.Flusher) {
	b.mu.RLock()
	provider := b.systemStats
	hp := b.healthProbe
	rp := b.resources
	pi := b.providerInfo
	orch := b.orchestrator
	b.mu.RUnlock()

	ctx := context.Background()

	writeInitialSystemStats(w, provider)
	writeInitialBuildUpdate(ctx, w, orch)

	if hp != nil {
		if msg := formatSSEMessage("health-update", map[string]any{
			"liveness":  hp.CheckLiveness(ctx),
			"readiness": hp.CheckReadiness(ctx),
		}); msg != nil {
			_, _ = w.Write(msg)
		}
	}

	if rp != nil {
		if msg := formatSSEMessage("resources-update", rp.GetResources()); msg != nil {
			_, _ = w.Write(msg)
		}
	}

	if pi != nil {
		if msg := formatSSEMessage("providers-update", gatherProviderPayload(ctx, pi)); msg != nil {
			_, _ = w.Write(msg)
		}
	}

	flusher.Flush()
}

// writeInitialSystemStats writes system-stats and memory-detail SSE messages
// to the response writer for a newly connected client.
//
// Takes w (http.ResponseWriter) which is the client's response writer.
// Takes provider (monitoring_domain.SystemStatsProvider)
// which supplies the current system metrics.
func writeInitialSystemStats(w http.ResponseWriter, provider monitoring_domain.SystemStatsProvider) {
	if provider == nil {
		return
	}
	stats := provider.GetStats()
	if msg := formatSSEMessage("system-stats", map[string]any{
		"heapAlloc":     stats.Memory.HeapAlloc,
		"heapSys":       stats.Memory.HeapSys,
		"goroutines":    stats.NumGoroutines,
		"cpuMillicores": stats.CPUMillicores,
		"uptimeMs":      stats.UptimeMs,
		"gcPauseNs":     stats.GC.LastPauseNs,
		"numGC":         stats.GC.NumGC,
		"timestampMs":   stats.TimestampMs,
	}); msg != nil {
		_, _ = w.Write(msg)
	}
	if msg := formatSSEMessage("memory-detail", map[string]any{
		"memory":        stats.Memory,
		"process":       stats.Process,
		"gc":            stats.GC,
		"runtime":       stats.Runtime,
		"build":         stats.Build,
		"NumGoroutines": stats.NumGoroutines,
		"CPUMillicores": stats.CPUMillicores,
		"NumCPU":        stats.NumCPU,
		"GOMAXPROCS":    stats.GOMAXPROCS,
	}); msg != nil {
		_, _ = w.Write(msg)
	}
}

// writeInitialBuildUpdate writes a build-update SSE message to the response
// writer for a newly connected client.
//
// Takes w (http.ResponseWriter) which is the client's response writer.
// Takes orch (orchestrator_domain.OrchestratorInspector)
// which supplies build pipeline state.
func writeInitialBuildUpdate(ctx context.Context, w http.ResponseWriter, orch orchestrator_domain.OrchestratorInspector) {
	if orch == nil {
		return
	}
	payload := map[string]any{}
	if summary, err := orch.ListTaskSummary(ctx); err == nil {
		payload["taskSummary"] = summary
	}
	if recentTasks, err := orch.ListRecentTasks(ctx, devSSERecentTaskLimit); err == nil {
		payload["recentTasks"] = recentTasks
	}
	if workflows, err := orch.ListWorkflowSummary(ctx, devSSEWorkflowSummaryLimit); err == nil {
		payload["workflowSummary"] = workflows
	}
	if len(payload) > 0 {
		if msg := formatSSEMessage("build-update", payload); msg != nil {
			_, _ = w.Write(msg)
		}
	}
}

// broadcastBuildUpdate pushes a build-update SSE event with task summary,
// recent tasks, and workflow summary from the orchestrator.
//
// Takes orch (orchestrator_domain.OrchestratorInspector)
// which supplies build pipeline state.
func (b *DevEventBroadcaster) broadcastBuildUpdate(ctx context.Context, orch orchestrator_domain.OrchestratorInspector) {
	payload := map[string]any{}

	summary, err := orch.ListTaskSummary(ctx)
	if err != nil {
		return
	}
	payload["taskSummary"] = summary

	recentTasks, _ := orch.ListRecentTasks(ctx, devSSERecentTaskLimit)
	payload["recentTasks"] = recentTasks

	workflows, _ := orch.ListWorkflowSummary(ctx, devSSEWorkflowSummaryLimit)
	payload["workflowSummary"] = workflows

	b.broadcastEvent("build-update", payload)
}

// broadcastHealthUpdate pushes a health-update SSE event.
//
// Takes hp (monitoring_domain.HealthProbeService) which
// supplies liveness and readiness probe results.
func (b *DevEventBroadcaster) broadcastHealthUpdate(ctx context.Context, hp monitoring_domain.HealthProbeService) {
	payload := map[string]any{
		"liveness":  hp.CheckLiveness(ctx),
		"readiness": hp.CheckReadiness(ctx),
	}
	b.broadcastEvent("health-update", payload)
}

// broadcastResourcesUpdate pushes a resources-update SSE event.
//
// Takes rp (monitoring_domain.ResourceProvider) which
// supplies resource and file descriptor data.
func (b *DevEventBroadcaster) broadcastResourcesUpdate(rp monitoring_domain.ResourceProvider) {
	b.broadcastEvent("resources-update", rp.GetResources())
}

// broadcastProvidersUpdate pushes a providers-update SSE event. For each
// resource type, the payload includes the provider list, per-provider detail
// sections (from DescribeProvider), and sub-resources where available.
//
// Takes pi (monitoring_domain.ProviderInfoInspector) which
// supplies provider discovery data.
func (b *DevEventBroadcaster) broadcastProvidersUpdate(ctx context.Context, pi monitoring_domain.ProviderInfoInspector) {
	b.broadcastEvent("providers-update", gatherProviderPayload(ctx, pi))
}

// gatherProviderPayload collects provider details and sub-resources for all
// resource types into a payload map suitable for SSE serialisation.
//
// Takes pi (monitoring_domain.ProviderInfoInspector) which
// supplies provider discovery data.
//
// Returns map[string]any which holds the aggregated provider
// payload.
func gatherProviderPayload(ctx context.Context, pi monitoring_domain.ProviderInfoInspector) map[string]any {
	types := pi.ListResourceTypes(ctx)
	providers := make(map[string]any, len(types))
	for _, rt := range types {
		list, err := pi.ListProviders(ctx, rt)
		if err != nil {
			continue
		}

		details := make(map[string]any, len(list.Rows))
		subResources := make(map[string]any)
		for _, row := range list.Rows {
			if detail, dErr := pi.DescribeProvider(ctx, rt, row.Name); dErr == nil && detail != nil {
				details[row.Name] = detail
			}
			if sub, sErr := pi.ListSubResources(ctx, rt, row.Name); sErr == nil && sub != nil && len(sub.Rows) > 0 {
				subResources[row.Name] = sub
			}
		}

		providers[rt] = map[string]any{
			"Columns":      list.Columns,
			"Rows":         list.Rows,
			"details":      details,
			"subResources": subResources,
		}
	}
	return map[string]any{
		"resourceTypes": types,
		"providers":     providers,
	}
}

// broadcastMemoryDetail pushes a memory-detail SSE event with full memory,
// GC, process, runtime, and build info. Top-level SystemStats fields that
// live outside the nested structs (goroutines, CPU, NumCPU, GOMAXPROCS) are
// included explicitly so the widget can display them.
//
// Takes provider (monitoring_domain.SystemStatsProvider)
// which supplies the current system metrics.
func (b *DevEventBroadcaster) broadcastMemoryDetail(provider monitoring_domain.SystemStatsProvider) {
	stats := provider.GetStats()
	b.broadcastEvent("memory-detail", map[string]any{
		"memory":        stats.Memory,
		"process":       stats.Process,
		"gc":            stats.GC,
		"runtime":       stats.Runtime,
		"build":         stats.Build,
		"NumGoroutines": stats.NumGoroutines,
		"CPUMillicores": stats.CPUMillicores,
		"NumCPU":        stats.NumCPU,
		"GOMAXPROCS":    stats.GOMAXPROCS,
	})
}

// broadcastEvent marshals a payload and sends it as a named SSE event to all
// connected clients.
//
// Takes eventName (string) which identifies the SSE event type.
// Takes payload (any) which is the data to marshal and broadcast.
//
// Safe for concurrent use; reads the client set under mu.RLock.
func (b *DevEventBroadcaster) broadcastEvent(eventName string, payload any) {
	msg := formatSSEMessage(eventName, payload)
	if msg == nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

// formatSSEMessage marshals a payload into an SSE-formatted byte slice.
//
// Takes eventName (string) which identifies the SSE event type.
// Takes payload (any) which is the data to marshal.
//
// Returns []byte which holds the SSE-formatted message, or nil if marshalling fails.
func formatSSEMessage(eventName string, payload any) []byte {
	data, err := devJSON.Marshal(payload)
	if err != nil {
		return nil
	}
	return fmt.Appendf(nil, "event: %s\ndata: %s\n\n", eventName, data)
}

// pathToRoute converts a component-relative path to a URL route pattern.
// Only page paths are converted; partials, emails, and other paths return false.
//
// Takes relPath (string) which is the component-relative path, e.g. "pages/login.pk".
//
// Returns string which is the URL route pattern, e.g. "/login".
// Returns bool which indicates whether the path was a valid page path.
func pathToRoute(relPath string) (string, bool) {
	if !strings.HasPrefix(relPath, "pages/") {
		return "", false
	}

	route := strings.TrimPrefix(relPath, "pages/")
	route = strings.TrimSuffix(route, ".pk")
	route = "/" + route

	route = strings.TrimSuffix(route, "/index")
	if route == "" {
		route = "/"
	}

	return route, true
}
