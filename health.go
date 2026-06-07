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

package piko

import (
	"context"
	"strings"
	"unicode"

	"piko.sh/piko/internal/healthprobe/healthprobe_domain"
	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/wdk/telemetry/readiness"
)

//
// Piko provides a health check system with two endpoints:
//   - /live (Liveness): Returns 200 if the application is running (not deadlocked)
//   - /ready (Readiness): Returns 200 if the application is ready to serve traffic
//
// The health check server runs on a separate port (default: 9090) and binds to localhost
// by default for security. It can be configured via the HealthProbe section in your Piko
// configuration.
//
// Built-in Health Checks: Piko automatically monitors the health of its internal
// services:
//   - RegistryService: Artefact storage and metadata
//   - OrchestratorService: Task queue and workers
//   - CollectionService: Content providers
//   - RenderService: Template rendering pipeline
//   - StorageService: File/blob storage
//   - EmailService: Email delivery
//   - CryptoService: Encryption operations
//   - CacheService: Cache backends
//   - And more...
//
// Custom Health Checks: You can extend Piko's health monitoring with your own
// application-specific checks by implementing the HealthProbe interface and registering
// it with WithCustomHealthProbe.
//
// Example configuration:
//
// 	piko.New(
// 	    piko.WithHealthEnabled(true),
// 	    piko.WithHealthProbePort(9090),
// 	    piko.WithHealthBindAddress("127.0.0.1"),  // localhost only (secure)
// 	    piko.WithHealthLivePath("/live"),
// 	    piko.WithHealthReadyPath("/ready"),
// 	    piko.WithHealthCheckTimeout(5*time.Second),
// 	)
//
// To expose health checks externally (e.g. for Docker health checks), use
// WithHealthBindAddress("0.0.0.0") - WARNING: this exposes internal health data so do not
// use on a public interface.

const (
	// HealthStateHealthy indicates the component is working normally.
	HealthStateHealthy HealthState = healthprobe_dto.StateHealthy

	// HealthStateDegraded indicates the component is working but with reduced performance or
	// limited features.
	HealthStateDegraded HealthState = healthprobe_dto.StateDegraded

	// HealthStateUnhealthy indicates the component is not working.
	HealthStateUnhealthy HealthState = healthprobe_dto.StateUnhealthy

	// HealthCheckLiveness checks if the application is running and not stuck. If this check
	// fails, the application is usually restarted.
	HealthCheckLiveness HealthCheckType = healthprobe_dto.CheckTypeLiveness

	// HealthCheckReadiness determines if the application is ready to serve traffic. Failing
	// this check typically results in traffic being withheld from the application.
	HealthCheckReadiness HealthCheckType = healthprobe_dto.CheckTypeReadiness

	// databaseDiagnosticsResourceType is the only resource type for which the "Engine
	// Diagnostics" section (database_size, replication_lag, ...) is surfaced as readiness
	// info. Other resource types that happen to name a section "Engine Diagnostics" do not
	// carry it through.
	databaseDiagnosticsResourceType = "database"

	// engineDiagnosticsSection is the title of the database section that carries provider
	// diagnostics; gated to databaseDiagnosticsResourceType.
	engineDiagnosticsSection = "Engine Diagnostics"

	// maxInfoEntriesPerDependency caps how many InfoEntry values the readiness adapter
	// attaches to a single dependency, a defence-in-depth bound matched by the collector's
	// own cap so a pathological descriptor cannot make one dependency carry unbounded info.
	maxInfoEntriesPerDependency = 32
)

var (
	// infoSectionWhitelist is the allow-list of provider-detail section titles a readiness
	// dependency may surface off-box, so a new secret-bearing section cannot leak by
	// default. sectionAllowed further gates "Engine Diagnostics" to the database resource
	// type.
	infoSectionWhitelist = map[string]bool{
		"Overview":           true,
		"Configuration":      true,
		"Engine Diagnostics": true,
	}

	// sensitiveInfoKeyTokens are key word-tokens that must never egress off-box: locators
	// (host, path, endpoint) and credentials (password, token).
	// DefaultReadinessInfoKeyFilter matches a key against this set.
	sensitiveInfoKeyTokens = map[string]bool{
		"host": true, "hostname": true, "addr": true, "address": true, "ip": true,
		"port": true, "path": true, "dir": true, "directory": true, "outbox": true,
		"dsn": true, "conn": true, "connection": true, "url": true, "uri": true,
		"endpoint": true, "bucket": true, "repository": true, "repo": true, "region": true,
		"password": true, "passwd": true, "secret": true, "token": true, "key": true,
		"apikey": true, "credential": true, "credentials": true, "user": true,
		"username": true, "account": true, "email": true,
	}
)

// HealthProbe is the interface that custom application health checks must implement.
// Implementing HealthProbe allows your application to take part in Piko's health check
// system, which is exposed via the /live and /ready endpoints.
//
// Interface definition:
//
//	type HealthProbe interface {
//	    Name() string
//	    Check(ctx context.Context, checkType HealthCheckType) HealthStatus
//	}
//
// The Check method receives the checkType parameter, which allows your probe to return
// different results for liveness vs readiness checks:
//   - Liveness: Quick check - is the service initialised and not deadlocked?
//   - Readiness: Thorough check - is the service ready to handle requests?
type HealthProbe = healthprobe_domain.Probe

// HealthStatus represents the result of a health check operation. It includes the
// component name, state, optional message, timestamp, and duration.
type HealthStatus = healthprobe_dto.Status

// HealthState represents the health status of a component.
type HealthState = healthprobe_dto.State

// HealthCheckType indicates whether this is a liveness or readiness check.
type HealthCheckType = healthprobe_dto.CheckType

// WithCustomHealthProbe registers a custom health probe with the Piko framework. The
// probe will be included in the /live and /ready health check endpoints.
//
// Use it to extend Piko's built-in health monitoring with application-specific checks
// (e.g., database connectivity, external API availability).
//
// Takes probe (HealthProbe) which is the health probe to register.
//
// Returns Option which configures the container with the custom probe.
func WithCustomHealthProbe(probe HealthProbe) Option {
	return func(c *Container) {
		if probe == nil {
			return
		}
		c.AddCustomHealthProbe(probe)
	}
}

// HealthProbe returns a readiness.Probe exposing piko's readiness tree, so an embedding
// app can sample it (for example to stream it to an external monitor). It is a thin
// public seam over the internal monitoring service and leaks no internal types.
//
// Returns readiness.Probe which samples the current readiness Snapshot.
func (s *SSRServer) HealthProbe() readiness.Probe {
	return &ssrReadinessProbe{server: s}
}

// ReadinessInfoKeyFilter reports whether a provider info key is sensitive and must be
// dropped before readiness info egresses off-box. It returns true to drop the entry.
type ReadinessInfoKeyFilter = func(key string) bool

// WithReadinessInfoKeyFilter overrides the predicate deciding which provider info keys
// are too sensitive to egress off-box via readiness telemetry. A nil filter keeps the
// default.
//
// Takes fn (ReadinessInfoKeyFilter) which reports true for a key to drop off-box.
//
// Returns Option which configures the container with the readiness info key filter.
func WithReadinessInfoKeyFilter(fn ReadinessInfoKeyFilter) Option {
	return func(c *Container) {
		c.SetReadinessInfoKeyFilter(fn)
	}
}

// DefaultReadinessInfoKeyFilter reports whether key names an infrastructure locator or
// credential that must not egress off-box. It matches any lower-cased word-token of key
// (split on non-alphanumeric boundaries) against sensitiveInfoKeyTokens.
//
// Takes key (string) which is the provider info key to classify.
//
// Returns bool which is true when the key is sensitive and must be dropped.
func DefaultReadinessInfoKeyFilter(key string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(key), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	for _, token := range tokens {
		if sensitiveInfoKeyTokens[token] {
			return true
		}
	}
	return false
}

// ssrReadinessProbe adapts piko's internal monitoring HealthProbeService to the public
// readiness.Probe seam, resolving the live service from the server's container on every
// sample (the container is built during Run, after the app obtains this probe).
type ssrReadinessProbe struct {
	// server is the owning SSRServer; its Container is consulted lazily at sample time.
	server *SSRServer
}

// CheckReadiness resolves the monitoring health probe and returns its readiness tree as a
// flat Snapshot, enriched per dependency with whitelisted provider info. When monitoring
// or its health probe is unavailable it returns an UNHEALTHY snapshot describing the gap.
//
// Takes ctx (context.Context) for cancellation of the underlying checks.
//
// Returns readiness.Snapshot which is the flattened readiness tree.
func (p *ssrReadinessProbe) CheckReadiness(ctx context.Context) readiness.Snapshot {
	probe, inspector := p.resolve()
	if probe == nil {
		return readiness.Snapshot{
			Name:    "readiness",
			State:   readiness.StateUnhealthy,
			Message: "monitoring health probe not configured",
		}
	}
	return readinessSnapshotFrom(ctx, probe.CheckReadiness(ctx), inspector, p.keyFilter())
}

// keyFilter returns the container's configured readiness info key filter, falling back to
// DefaultReadinessInfoKeyFilter when none was registered.
//
// Returns ReadinessInfoKeyFilter which decides the keys dropped before egress.
func (p *ssrReadinessProbe) keyFilter() ReadinessInfoKeyFilter {
	if p.server != nil && p.server.Container != nil {
		if fn := p.server.Container.GetReadinessInfoKeyFilter(); fn != nil {
			return fn
		}
	}
	return DefaultReadinessInfoKeyFilter
}

// resolve fetches the configured HealthProbeService and ProviderInfoInspector from the
// live container. Both are nil when monitoring is disabled or the container is not built
// yet, so missing provider info never blocks a readiness sample.
//
// Returns monitoring_domain.HealthProbeService which runs the readiness checks.
// Returns monitoring_domain.ProviderInfoInspector which supplies per-dependency info.
func (p *ssrReadinessProbe) resolve() (monitoring_domain.HealthProbeService, monitoring_domain.ProviderInfoInspector) {
	if p.server == nil || p.server.Container == nil {
		return nil, nil
	}
	monitoringService := p.server.Container.GetMonitoringService()
	if monitoringService == nil {
		return nil, nil
	}
	return monitoringService.HealthProbe(), monitoringService.ProviderInfoInspector()
}

// readinessSnapshotFrom flattens the internal HealthProbeStatus tree into a public
// readiness.Snapshot: the root plus its immediate dependency children. Each dependency
// that mapped to a descriptor is enriched with its whitelisted info, gated by sensitive.
//
// Takes ctx (context.Context) for cancellation of the per-dependency describe work.
// Takes status (monitoring_domain.HealthProbeStatus) which is the internal readiness
// tree.
// Takes inspector (monitoring_domain.ProviderInfoInspector) which supplies provider info.
// Takes sensitive (ReadinessInfoKeyFilter) which drops sensitive info keys off-box.
//
// Returns readiness.Snapshot which is the flattened, enriched tree.
func readinessSnapshotFrom(ctx context.Context, status monitoring_domain.HealthProbeStatus, inspector monitoring_domain.ProviderInfoInspector, sensitive ReadinessInfoKeyFilter) readiness.Snapshot {
	var resolver monitoring_domain.ProviderProbeResolver
	if r, ok := inspector.(monitoring_domain.ProviderProbeResolver); ok {
		resolver = r
	}
	deps := make([]readiness.Dependency, 0, len(status.Dependencies))
	for _, dep := range status.Dependencies {
		deps = append(deps, readiness.Dependency{
			Name:     dep.Name,
			State:    readiness.State(dep.State),
			Message:  dep.Message,
			Duration: dep.Duration,
			Info:     providerInfoForDependency(ctx, inspector, resolver, dep.Name, sensitive),
		})
	}
	return readiness.Snapshot{
		Name:         status.Name,
		State:        readiness.State(status.State),
		Message:      status.Message,
		Duration:     status.Duration,
		Dependencies: deps,
	}
}

// providerInfoForDependency resolves the provider detail for the dependency depName and
// flattens its whitelisted, non-sensitive sections into readiness.InfoEntry values. It
// returns nil when no descriptor maps to it, the resource lists no providers, or describe
// fails.
//
// Takes ctx (context.Context) for cancellation of the describe work.
// Takes inspector (monitoring_domain.ProviderInfoInspector) which describes providers.
// Takes resolver (monitoring_domain.ProviderProbeResolver) which maps a probe to a
// resource.
// Takes depName (string) which is the readiness dependency name.
// Takes sensitive (ReadinessInfoKeyFilter) which drops sensitive info keys off-box.
//
// Returns []readiness.InfoEntry which is the whitelisted provider info, or nil.
func providerInfoForDependency(
	ctx context.Context,
	inspector monitoring_domain.ProviderInfoInspector,
	resolver monitoring_domain.ProviderProbeResolver,
	depName string,
	sensitive ReadinessInfoKeyFilter,
) []readiness.InfoEntry {
	if inspector == nil || resolver == nil {
		return nil
	}
	resourceType, ok := resolver.ResourceTypeForProbe(ctx, depName)
	if !ok {
		return nil
	}

	providerName, ok := defaultProviderName(ctx, inspector, resourceType)
	if !ok {
		return nil
	}

	detail, err := inspector.DescribeProvider(ctx, resourceType, providerName)
	if err != nil || detail == nil {
		return nil
	}

	return whitelistedInfoEntries(resourceType, depName, detail, sensitive)
}

// defaultProviderName returns the provider name to describe for a resource type: the
// entry flagged IsDefault, else the first listed entry. A dependency with several
// connections surfaces only this single default connection's detail, keeping the info
// bounded.
//
// Takes ctx (context.Context) for cancellation of the provider listing.
// Takes inspector (monitoring_domain.ProviderInfoInspector) which lists providers.
// Takes resourceType (string) which is the resource whose default provider is sought.
//
// Returns string which is the default provider name.
// Returns bool which is false when the resource lists no providers.
func defaultProviderName(ctx context.Context, inspector monitoring_domain.ProviderInfoInspector, resourceType string) (string, bool) {
	list, err := inspector.ListProviders(ctx, resourceType)
	if err != nil || list == nil || len(list.Rows) == 0 {
		return "", false
	}
	for i := range list.Rows {
		if list.Rows[i].IsDefault {
			return list.Rows[i].Name, true
		}
	}
	return list.Rows[0].Name, true
}

// whitelistedInfoEntries flattens a ProviderDetail into readiness.InfoEntry values,
// keeping only sections in infoSectionWhitelist (with "Engine Diagnostics" gated to the
// database type) and entries passing skipInfoEntry, bounded to
// maxInfoEntriesPerDependency.
//
// Takes resourceType (string) which is the resource being described.
// Takes depName (string) which is the readiness dependency name.
// Takes detail (*provider_domain.ProviderDetail) which is the provider's full detail.
// Takes sensitive (ReadinessInfoKeyFilter) which drops sensitive info keys off-box.
//
// Returns []readiness.InfoEntry which is the whitelisted info, or nil when none
// qualifies.
func whitelistedInfoEntries(resourceType, depName string, detail *provider_domain.ProviderDetail, sensitive ReadinessInfoKeyFilter) []readiness.InfoEntry {
	entries := make([]readiness.InfoEntry, 0, maxInfoEntriesPerDependency)
	for i := range detail.Sections {
		section := &detail.Sections[i]
		if !sectionAllowed(resourceType, section.Title) {
			continue
		}
		for j := range section.Entries {
			entry := &section.Entries[j]
			if skipInfoEntry(depName, entry.Key, entry.Value, sensitive) {
				continue
			}
			entries = append(entries, readiness.InfoEntry{
				Section: section.Title,
				Key:     entry.Key,
				Value:   entry.Value,
			})
			if len(entries) >= maxInfoEntriesPerDependency {
				return entries
			}
		}
	}
	if len(entries) == 0 {
		return nil
	}
	return entries
}

// sectionAllowed reports whether a provider detail section may be surfaced off-box.
// "Engine Diagnostics" is allowed only for the database resource type, so a like-named
// section on another resource cannot smuggle unexpected fields through.
//
// Takes resourceType (string) which is the resource being described.
// Takes title (string) which is the section title to check.
//
// Returns bool which is true when the section may be surfaced.
func sectionAllowed(resourceType, title string) bool {
	if !infoSectionWhitelist[title] {
		return false
	}
	if title == engineDiagnosticsSection {
		return resourceType == databaseDiagnosticsResourceType
	}
	return true
}

// skipInfoEntry reports whether an info entry must be dropped before off-box egress:
// empty, dependency-name-echoing, reserved-label, or sensitive-key entries. The local CLI
// describe view bypasses this filter; only the readiness telemetry path is gated.
//
// Takes depName (string) which is the owning dependency name.
// Takes key (string) which is the info entry key.
// Takes value (string) which is the info entry value.
// Takes sensitive (ReadinessInfoKeyFilter) which rejects keys that must not egress
// off-box.
//
// Returns bool which is true when the entry must be dropped.
func skipInfoEntry(depName, key, value string, sensitive ReadinessInfoKeyFilter) bool {
	if key == "" || value == "" {
		return true
	}
	if strings.EqualFold(key, depName) || strings.EqualFold(value, depName) {
		return true
	}
	switch strings.ToLower(key) {
	case "status", "icon", "message":
		return true
	}
	return sensitive(key)
}
