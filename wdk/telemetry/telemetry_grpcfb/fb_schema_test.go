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

package telemetry_grpcfb

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	flatbuffers "github.com/google/flatbuffers/go"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb/telemetryfb"
)

var (
	elementSpecs = map[string]string{
		"AnalyticsEvent": fmt.Sprintf("%p", analyticsFields),
		"WatchdogEvent":  fmt.Sprintf("%p", watchdogFields),
		"LogLine":        fmt.Sprintf("%p", logFields),
		"Span":           fmt.Sprintf("%p", spanFields),
		"MetricPoint":    fmt.Sprintf("%p", metricFields),
		"ErrorEvent":     fmt.Sprintf("%p", errorFields),
		"ProfileMeta":    fmt.Sprintf("%p", profileFields),
		"WorkerEvent":    fmt.Sprintf("%p", workerFields),
		"QueryStat":      fmt.Sprintf("%p", queryStatFields),
		"EmailEvent":     fmt.Sprintf("%p", emailFields),
	}
)

const (
	schemaPath = "telemetry.fbs"
	firstVOffset = 4
	voffsetStride = 2
)

func isTableName(element string) bool {
	return element != "" && element[0] >= 'A' && element[0] <= 'Z'
}

var (
	batchFieldPattern = regexp.MustCompile(`^\s*[a-z_0-9]+:([^;(]+?)\s*\(\s*id:\s*(\d+)\s*\)\s*;`)
)

func schemaFieldTypes(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	table := regexp.MustCompile(`(?s)table TelemetryBatch \{(.*?)\n\}`).FindStringSubmatch(string(raw))
	require.Len(t, table, 2, "TelemetryBatch table not found in %s", schemaPath)

	declaration := regexp.MustCompile(`^\s*[a-z_0-9]+:[^;]+;`)

	var types []string
	for line := range strings.SplitSeq(table[1], "\n") {
		if !declaration.MatchString(line) {
			continue
		}

		match := batchFieldPattern.FindStringSubmatch(line)
		require.NotNil(t, match,
			"every TelemetryBatch field needs an explicit id; %q has none", strings.TrimSpace(line))
		require.Equal(t, strconv.Itoa(len(types)), match[2],
			"field ids must run in declaration order from zero, or the wire layout shifts")

		types = append(types, strings.TrimSpace(match[1]))
	}

	return types
}

func schemaKind(t *testing.T, fbsType string) fieldKind {
	t.Helper()

	if fbsType == "[ubyte]" || fbsType == "[byte]" {
		return kVectorByte
	}
	if inner, isVector := strings.CutPrefix(fbsType, "["); isVector {
		element := strings.TrimSuffix(inner, "]")
		if !isTableName(element) {
			require.Failf(t, "unmapped .fbs vector",
				"%q holds scalars; extend schemaKind before adding it to a table", fbsType)
			return kBool
		}
		return kVectorTable
	}
	switch fbsType {
	case "string":
		return kString
	case "long":
		return kInt64
	case "int":
		return kInt32
	case "bool":
		return kBool
	case "double":
		return kFloat64
	default:
		require.Failf(t, "unmapped .fbs type",
			"%q is not mapped; extend schemaKind before adding it to a table", fbsType)
		return kBool
	}
}

func TestBatchFieldsMatchSchema(t *testing.T) {
	types := schemaFieldTypes(t)

	require.Len(t, batchFields, len(types),
		"telemetry.fbs declares %d TelemetryBatch fields and batchFields lists %d",
		len(types), len(batchFields))

	for i, fbsType := range types {
		want := uint16(firstVOffset + i*voffsetStride)
		assert.Equal(t, want, batchFields[i].voffset,
			"field %d (%s): voffsets must run 4, 6, 8 ... with no gaps", i, fbsType)
		assert.Equal(t, schemaKind(t, fbsType), batchFields[i].kind,
			"field %d (%s): verifier kind must match the schema type", i, fbsType)

		if batchFields[i].kind == kVectorTable {
			element := strings.TrimSuffix(strings.TrimPrefix(fbsType, "["), "]")
			assert.Equal(t, elementSpecs[element], fmt.Sprintf("%p", batchFields[i].elem),
				"field %d (%s): verifier element spec must be the one for %s", i, fbsType, element)
		}
	}

	assert.EqualValues(t, 48, batchFields[len(batchFields)-1].voffset)
	assert.Equal(t, kInt32, batchFields[len(batchFields)-1].kind)
}

func TestOldVtableDecodesAsZero(t *testing.T) {
	frame := marshalPreIdentityBatch(t)

	var got Batch
	require.NoError(t, got.Unmarshal(frame),
		"a frame from an emitter that predates the identity fields must still verify")

	assert.Equal(t, "site-1", got.SiteID, "the fields it did write are unaffected")
	assert.EqualValues(t, 7, got.Seq)

	assert.Empty(t, got.InstanceID, "an unwritten field reads as its zero value")
	assert.Empty(t, got.Hostname)
	assert.Empty(t, got.ServiceName)
	assert.Empty(t, got.ServiceVersion)
	assert.Empty(t, got.Environment)
	assert.Empty(t, got.Region)
	assert.Zero(t, got.StartedAtMs)
	assert.Zero(t, got.PID)
}

func marshalPreIdentityBatch(t *testing.T) []byte {
	t.Helper()

	b := flatbuffers.NewBuilder(0)
	site := b.CreateString("site-1")

	telemetryfb.TelemetryBatchStart(b)
	telemetryfb.TelemetryBatchAddSiteId(b, site)
	telemetryfb.TelemetryBatchAddSeq(b, 7)
	b.Finish(telemetryfb.TelemetryBatchEnd(b))

	return b.FinishedBytes()
}

func TestIdentityRoundTripsAndIsNotAnEvent(t *testing.T) {
	sent := Batch{
		SiteID: "site-1", InstanceID: "pod-7", Hostname: "node-a",
		ServiceName: "piko-site", ServiceVersion: "v1.2.3",
		Environment: "production", Region: "europe-west2",
		StartedAtMs: 1_700_000_000_000, PID: 4242,
		Logs: []LogLine{{Message: "hello"}},
	}
	frame, err := sent.Marshal()
	require.NoError(t, err)

	var got Batch
	require.NoError(t, got.Unmarshal(frame))

	assert.Equal(t, "pod-7", got.InstanceID)
	assert.Equal(t, "node-a", got.Hostname)
	assert.Equal(t, "piko-site", got.ServiceName)
	assert.Equal(t, "v1.2.3", got.ServiceVersion)
	assert.Equal(t, "production", got.Environment)
	assert.Equal(t, "europe-west2", got.Region)
	assert.EqualValues(t, 1_700_000_000_000, got.StartedAtMs)
	assert.EqualValues(t, 4242, got.PID)

	assert.Equal(t, 1, got.EventCount(),
		"identity describes the sender; counting it would make ack reconciliation short")
}

func TestClientStampsIdentityOnEveryFrame(t *testing.T) {
	config := Config{
		SiteID: "site-1", APIKey: "key-1", Source: "test",
		Identity: Identity{
			InstanceID: "pod-7", Hostname: "node-a", ServiceName: "piko-site",
			ServiceVersion: "v1.2.3", Environment: "production", Region: "europe-west2",
			StartedAtMs: 1_700_000_000_000, PID: 4242,
		},
		FlushSize: 1, FlushInterval: time.Hour, MaxQueuedBatches: 0, Breaker: nil,
	}

	for frame := range 3 {
		batch := newBatch(&config)
		assert.Equal(t, "pod-7", batch.InstanceID, "frame %d", frame)
		assert.Equal(t, "node-a", batch.Hostname, "frame %d", frame)
		assert.Equal(t, "piko-site", batch.ServiceName, "frame %d", frame)
		assert.Equal(t, "v1.2.3", batch.ServiceVersion, "frame %d", frame)
		assert.Equal(t, "production", batch.Environment, "frame %d", frame)
		assert.Equal(t, "europe-west2", batch.Region, "frame %d", frame)
		assert.EqualValues(t, 1_700_000_000_000, batch.StartedAtMs, "frame %d", frame)
		assert.EqualValues(t, 4242, batch.PID, "frame %d", frame)
	}
}

func TestClientWithoutIdentityStampsNothing(t *testing.T) {
	config := Config{
		SiteID: "site-1", APIKey: "", Source: "", Identity: Identity{},
		FlushSize: 0, FlushInterval: 0, MaxQueuedBatches: 0, Breaker: nil,
	}

	batch := newBatch(&config)

	assert.Equal(t, "site-1", batch.SiteID)
	assert.Empty(t, batch.InstanceID)
	assert.Zero(t, batch.PID)
	assert.Zero(t, batch.StartedAtMs)
}

func TestIdentityFieldsAreCapped(t *testing.T) {
	huge := strings.Repeat("h", 4<<20)
	sent := Batch{
		SiteID:      "site-1",
		InstanceID:  huge,
		Hostname:    huge,
		ServiceName: huge, ServiceVersion: huge,
		Environment: huge, Region: huge,
		Logs: []LogLine{{Message: "hello"}},
	}

	frame, err := sent.Marshal()
	require.NoError(t, err, "an over-long identity must not push the frame over the cap")
	assert.Less(t, len(frame), 1<<20, "identity contributes kilobytes, not megabytes")

	var got Batch
	require.NoError(t, got.Unmarshal(frame))

	assert.Len(t, got.Hostname, maxHostnameLen)
	assert.Len(t, got.InstanceID, maxInstanceIDLen)
	assert.Len(t, got.ServiceName, maxServiceNameLen)
	assert.Len(t, got.ServiceVersion, maxServiceVersionLen)
	assert.Len(t, got.Environment, maxEnvironmentLen)
	assert.Len(t, got.Region, maxRegionLen)
}

func TestIdentityIsCappedOnDecode(t *testing.T) {
	oversized := strings.Repeat("x", 4096)

	builder := flatbuffers.NewBuilder(0)
	siteID := builder.CreateString("site-1")
	hostname := builder.CreateString(oversized)
	telemetryfb.TelemetryBatchStart(builder)
	telemetryfb.TelemetryBatchAddSiteId(builder, siteID)
	telemetryfb.TelemetryBatchAddHostname(builder, hostname)
	builder.Finish(telemetryfb.TelemetryBatchEnd(builder))
	frame := builder.FinishedBytes()

	var got Batch
	require.NoError(t, got.Unmarshal(frame))
	assert.Len(t, got.Hostname, maxHostnameLen, "the receiver applies its own cap")
}

func TestSchemaKind_DistinguishesScalarVectorsFromTableVectors(t *testing.T) {
	assert.Equal(t, kVectorByte, schemaKind(t, "[ubyte]"))
	assert.Equal(t, kVectorByte, schemaKind(t, "[byte]"))
	assert.Equal(t, kVectorTable, schemaKind(t, "[Span]"))
	assert.Equal(t, kVectorTable, schemaKind(t, "[KV]"))
}

func TestIsTableName(t *testing.T) {
	for _, element := range []string{"Span", "KV", "LogLine", "ProfileMeta"} {
		assert.True(t, isTableName(element), "%s names a table", element)
	}
	for _, element := range []string{"long", "int", "ubyte", "double", "bool", ""} {
		assert.False(t, isTableName(element), "%q names a scalar", element)
	}
}
