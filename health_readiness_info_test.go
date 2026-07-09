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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/wdk/telemetry/readiness"
)

type fakeProviderInfoInspector struct {
	probeToType map[string]string
	providers   map[string][]provider_domain.ProviderListEntry
	details     map[string]*provider_domain.ProviderDetail
	describeErr error
}

func (f *fakeProviderInfoInspector) ListResourceTypes(context.Context) []string { return nil }

func (f *fakeProviderInfoInspector) ListProviders(_ context.Context, resourceType string) (*monitoring_domain.ProviderListResult, error) {
	rows, ok := f.providers[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return &monitoring_domain.ProviderListResult{Rows: rows}, nil
}

func (f *fakeProviderInfoInspector) DescribeProvider(_ context.Context, resourceType, name string) (*provider_domain.ProviderDetail, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	detail, ok := f.details[resourceType+"/"+name]
	if !ok {
		return nil, fmt.Errorf("provider %q of type %q not found", name, resourceType)
	}
	return detail, nil
}

func (f *fakeProviderInfoInspector) ListSubResources(context.Context, string, string) (*monitoring_domain.ProviderListResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProviderInfoInspector) DescribeResourceType(context.Context, string) (*provider_domain.ProviderDetail, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f *fakeProviderInfoInspector) ResourceTypeForProbe(_ context.Context, probeName string) (string, bool) {
	rt, ok := f.probeToType[probeName]
	return rt, ok
}

var (
	_ monitoring_domain.ProviderInfoInspector = (*fakeProviderInfoInspector)(nil)
	_ monitoring_domain.ProviderProbeResolver = (*fakeProviderInfoInspector)(nil)
)

func databaseInspector() *fakeProviderInfoInspector {
	return &fakeProviderInfoInspector{
		probeToType: map[string]string{"DatabaseService": "database"},
		providers: map[string][]provider_domain.ProviderListEntry{
			"database": {{Name: "primary"}},
		},
		details: map[string]*provider_domain.ProviderDetail{
			"database/primary": {
				Name: "primary",
				Sections: []provider_domain.InfoSection{
					{
						Title: "Overview",
						Entries: []provider_domain.InfoEntry{
							{Key: "Name", Value: "primary"},
							{Key: "Driver", Value: "sqlite"},
						},
					},
					{
						Title: "Configuration",
						Entries: []provider_domain.InfoEntry{
							{Key: "Host", Value: "localhost"},
							{Key: "Mode", Value: "wal"},
						},
					},
					{
						Title: "Connection Pool",
						Entries: []provider_domain.InfoEntry{
							{Key: "Open Connections", Value: "5"},
							{Key: "In Use", Value: "2"},
						},
					},
					{
						Title: "Engine Diagnostics",
						Entries: []provider_domain.InfoEntry{
							{Key: "database_size", Value: "142 MiB"},
							{Key: "replication_lag", Value: "0s"},
						},
					},
				},
			},
		},
	}
}

func dbReadinessStatus() monitoring_domain.HealthProbeStatus {
	return monitoring_domain.HealthProbeStatus{
		Name:  "readiness",
		State: "HEALTHY",
		Dependencies: []monitoring_domain.HealthProbeStatus{
			{Name: "DatabaseService", State: "HEALTHY", Message: "ok", Duration: "1.5ms"},
			{Name: "OrchestratorService", State: "HEALTHY", Message: "ok", Duration: "0.5ms"},
		},
	}
}

func findInfo(dep readiness.Dependency, section, key string) (string, bool) {
	for _, e := range dep.Info {
		if e.Section == section && e.Key == key {
			return e.Value, true
		}
	}
	return "", false
}

func depByName(snap readiness.Snapshot, name string) (readiness.Dependency, bool) {
	for _, d := range snap.Dependencies {
		if d.Name == name {
			return d, true
		}
	}
	return readiness.Dependency{}, false
}

func TestReadinessSnapshotFromInjectsWhitelistedProviderInfo(t *testing.T) {
	snap := readinessSnapshotFrom(context.Background(), dbReadinessStatus(), databaseInspector(), DefaultReadinessInfoKeyFilter)

	db, ok := depByName(snap, "DatabaseService")
	require.True(t, ok, "database dependency should be present")
	require.NotEmpty(t, db.Info, "matched dependency must carry provider info")

	size, ok := findInfo(db, "Engine Diagnostics", "database_size")
	require.True(t, ok, "database_size must be present in Engine Diagnostics")
	assert.Equal(t, "142 MiB", size)

	lag, ok := findInfo(db, "Engine Diagnostics", "replication_lag")
	require.True(t, ok)
	assert.Equal(t, "0s", lag)

	driver, ok := findInfo(db, "Overview", "Driver")
	require.True(t, ok)
	assert.Equal(t, "sqlite", driver)

	mode, ok := findInfo(db, "Configuration", "Mode")
	require.True(t, ok)
	assert.Equal(t, "wal", mode)

	_, ok = findInfo(db, "Configuration", "Host")
	assert.False(t, ok, "sensitive Configuration keys (Host) must not egress off-box")

	_, ok = findInfo(db, "Connection Pool", "Open Connections")
	assert.False(t, ok, "Connection Pool counters must not leak into info")

	_, ok = findInfo(db, "Overview", "Name")
	assert.True(t, ok, "Overview Name should be retained when it does not echo the dependency name")
}

func TestReadinessSnapshotFromLeavesUnmatchedDependencyInfoEmpty(t *testing.T) {
	snap := readinessSnapshotFrom(context.Background(), dbReadinessStatus(), databaseInspector(), DefaultReadinessInfoKeyFilter)

	orch, ok := depByName(snap, "OrchestratorService")
	require.True(t, ok)
	assert.Empty(t, orch.Info, "a dependency with no matching descriptor carries empty Info")
}

func TestReadinessSnapshotFromNilInspectorIsGraceful(t *testing.T) {
	snap := readinessSnapshotFrom(context.Background(), dbReadinessStatus(), nil, DefaultReadinessInfoKeyFilter)

	require.Len(t, snap.Dependencies, 2)
	for _, d := range snap.Dependencies {
		assert.Empty(t, d.Info, "nil inspector yields no info, dependencies render unchanged")
	}
}

func TestReadinessSnapshotFromGatesEngineDiagnosticsToDatabase(t *testing.T) {

	insp := &fakeProviderInfoInspector{
		probeToType: map[string]string{"CacheService": "cache"},
		providers: map[string][]provider_domain.ProviderListEntry{
			"cache": {{Name: "default", IsDefault: true}},
		},
		details: map[string]*provider_domain.ProviderDetail{
			"cache/default": {
				Name: "default",
				Sections: []provider_domain.InfoSection{
					{Title: "Overview", Entries: []provider_domain.InfoEntry{{Key: "Type", Value: "redis"}}},
					{Title: "Engine Diagnostics", Entries: []provider_domain.InfoEntry{{Key: "secret_leak", Value: "nope"}}},
				},
			},
		},
	}
	status := monitoring_domain.HealthProbeStatus{
		Name:  "readiness",
		State: "HEALTHY",
		Dependencies: []monitoring_domain.HealthProbeStatus{
			{Name: "CacheService", State: "HEALTHY", Duration: "1ms"},
		},
	}

	snap := readinessSnapshotFrom(context.Background(), status, insp, DefaultReadinessInfoKeyFilter)
	cache, ok := depByName(snap, "CacheService")
	require.True(t, ok)

	_, ok = findInfo(cache, "Overview", "Type")
	assert.True(t, ok, "Overview is whitelisted for cache")
	_, ok = findInfo(cache, "Engine Diagnostics", "secret_leak")
	assert.False(t, ok, "Engine Diagnostics is gated to the database resource type")
}

func TestReadinessSnapshotFromChoosesDefaultProvider(t *testing.T) {

	insp := &fakeProviderInfoInspector{
		probeToType: map[string]string{"DatabaseService": "database"},
		providers: map[string][]provider_domain.ProviderListEntry{
			"database": {{Name: "replica"}, {Name: "primary", IsDefault: true}},
		},
		details: map[string]*provider_domain.ProviderDetail{
			"database/primary": {
				Name:     "primary",
				Sections: []provider_domain.InfoSection{{Title: "Overview", Entries: []provider_domain.InfoEntry{{Key: "Role", Value: "primary"}}}},
			},
			"database/replica": {
				Name:     "replica",
				Sections: []provider_domain.InfoSection{{Title: "Overview", Entries: []provider_domain.InfoEntry{{Key: "Role", Value: "replica"}}}},
			},
		},
	}
	status := monitoring_domain.HealthProbeStatus{
		Name: "readiness", State: "HEALTHY",
		Dependencies: []monitoring_domain.HealthProbeStatus{{Name: "DatabaseService", State: "HEALTHY", Duration: "1ms"}},
	}

	snap := readinessSnapshotFrom(context.Background(), status, insp, DefaultReadinessInfoKeyFilter)
	db, _ := depByName(snap, "DatabaseService")
	role, ok := findInfo(db, "Overview", "Role")
	require.True(t, ok)
	assert.Equal(t, "primary", role, "the IsDefault connection's detail is surfaced")
}

func TestReadinessSnapshotFromDescribeErrorYieldsEmptyInfo(t *testing.T) {
	insp := databaseInspector()
	insp.describeErr = fmt.Errorf("db down")
	snap := readinessSnapshotFrom(context.Background(), dbReadinessStatus(), insp, DefaultReadinessInfoKeyFilter)
	db, _ := depByName(snap, "DatabaseService")
	assert.Empty(t, db.Info, "a DescribeProvider error leaves Info empty, dependency renders unchanged")
}

func TestSkipInfoEntryDropsReservedAndDuplicateKeys(t *testing.T) {
	assert.True(t, skipInfoEntry("DatabaseService", "status", "x", DefaultReadinessInfoKeyFilter), "reserved status key dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "icon", "x", DefaultReadinessInfoKeyFilter), "reserved icon key dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "message", "x", DefaultReadinessInfoKeyFilter), "reserved message key dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "DatabaseService", "x", DefaultReadinessInfoKeyFilter), "key echoing dep name dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "k", "DatabaseService", DefaultReadinessInfoKeyFilter), "value echoing dep name dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "", "v", DefaultReadinessInfoKeyFilter), "empty key dropped")
	assert.True(t, skipInfoEntry("DatabaseService", "k", "", DefaultReadinessInfoKeyFilter), "empty value dropped")
	assert.False(t, skipInfoEntry("DatabaseService", "Driver", "sqlite", DefaultReadinessInfoKeyFilter), "ordinary entry kept")
}

func TestSkipInfoEntryFiltersSensitiveKeys(t *testing.T) {
	sensitive := []string{
		"host", "Hostname", "addr", "Address", "IP", "port", "path", "Base Directory",
		"outbox_path", "dsn", "connection_string", "url", "Script URL", "endpoint",
		"bucket", "repository", "region", "password", "secret", "token", "api_key",
		"Site Key", "credential", "username", "account", "from_email", "email",
	}
	for _, key := range sensitive {
		assert.Truef(t, skipInfoEntry("Svc", key, "v", DefaultReadinessInfoKeyFilter), "sensitive key %q must be dropped off-box", key)
	}

	benign := []string{
		"Driver", "Backend", "Version", "Replicas", "Model", "Database Count",
		"Open Connections", "Default", "Registered", "Type", "Resource Type",
	}
	for _, key := range benign {
		assert.Falsef(t, skipInfoEntry("Svc", key, "v", DefaultReadinessInfoKeyFilter), "benign key %q must be kept", key)
	}
}

func TestWhitelistedInfoEntriesDropsSensitiveConfiguration(t *testing.T) {
	detail := &provider_domain.ProviderDetail{
		Name: "smtp",
		Sections: []provider_domain.InfoSection{
			{
				Title: "Configuration",
				Entries: []provider_domain.InfoEntry{
					{Key: "host", Value: "mail.internal.example"},
					{Key: "from_email", Value: "noreply@example.com"},
					{Key: "Backend", Value: "smtp"},
				},
			},
		},
	}
	entries := whitelistedInfoEntries("email", "EmailService", detail, DefaultReadinessInfoKeyFilter)
	require.Len(t, entries, 1, "only the benign entry should survive")
	assert.Equal(t, "Backend", entries[0].Key)
	assert.Equal(t, "smtp", entries[0].Value)
}
