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
	"context"

	flatbuffers "github.com/google/flatbuffers/go"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb/telemetryfb"
)

// batchIdentity holds the string offsets for the emitter identity fields, which must all
// be created before the table is started.
type batchIdentity struct {
	// instanceID is the offset of the encoded instance identifier, which names the emitting
	// process among sibling replicas of the same service.
	instanceID flatbuffers.UOffsetT

	// hostname is the offset of the encoded name of the machine the batch was emitted from,
	// clipped to the RFC 1035 ceiling for a fully qualified name.
	hostname flatbuffers.UOffsetT

	// serviceName is the offset of the encoded name of the deployed service the emitting
	// process belongs to.
	serviceName flatbuffers.UOffsetT

	// serviceVersion is the offset of the encoded version of the build the emitting process
	// is running.
	serviceVersion flatbuffers.UOffsetT

	// environment is the offset of the encoded deployment environment, such as "production"
	// or "staging".
	environment flatbuffers.UOffsetT

	// region is the offset of the encoded cloud region of the SERVICE, not the user's, and
	// is zero off-cloud.
	region flatbuffers.UOffsetT
}

// marshalIdentity creates the identity strings ahead of the table.
//
// Takes b (*flatbuffers.Builder) which is the builder being written to.
// Takes bt (*Batch) which carries the identity.
//
// Returns batchIdentity which holds the created string offsets.
func marshalIdentity(b *flatbuffers.Builder, bt *Batch) batchIdentity {
	return batchIdentity{
		instanceID:     str(b, clip(bt.InstanceID, maxInstanceIDLen)),
		hostname:       str(b, clip(bt.Hostname, maxHostnameLen)),
		serviceName:    str(b, clip(bt.ServiceName, maxServiceNameLen)),
		serviceVersion: str(b, clip(bt.ServiceVersion, maxServiceVersionLen)),
		environment:    str(b, clip(bt.Environment, maxEnvironmentLen)),
		region:         str(b, clip(bt.Region, maxRegionLen)),
	}
}

// clip shortens an identity field to its cap, keeping the result valid UTF-8 and counting
// the loss.
//
// Takes value (string) which is the field as the emitter supplied it.
// Takes maxBytes (int) which is the cap for that field.
//
// Returns string which is at most maxBytes bytes.
func clip(value string, maxBytes int) string {
	clipped, truncated := TruncateUTF8(value, maxBytes)
	if truncated {
		identityTruncCount.Add(1)
		stringsTruncatedCount.Add(context.Background(), 1)
		warnIdentityTruncatedOnce.Do(func() {
			_, l := logger_domain.From(context.Background(), log)
			l.Warn("telemetry identity field shortened to fit its cap; "+
				"two emitters with a long shared prefix will report as one",
				logger_domain.Int("cap_bytes", maxBytes),
			)
		})
	}

	return clipped
}

// addIdentity writes the emitter identity into the open table.
//
// Takes b (*flatbuffers.Builder) which is the builder being written to.
// Takes bt (*Batch) which carries the scalar identity fields.
// Takes identity (batchIdentity) which holds the pre-created string offsets.
func addIdentity(b *flatbuffers.Builder, bt *Batch, identity batchIdentity) {
	addOffset(b, identity.instanceID, telemetryfb.TelemetryBatchAddInstanceId)
	addOffset(b, identity.hostname, telemetryfb.TelemetryBatchAddHostname)
	addOffset(b, identity.serviceName, telemetryfb.TelemetryBatchAddServiceName)
	addOffset(b, identity.serviceVersion, telemetryfb.TelemetryBatchAddServiceVersion)
	addOffset(b, identity.environment, telemetryfb.TelemetryBatchAddEnvironment)
	addOffset(b, identity.region, telemetryfb.TelemetryBatchAddRegion)
	telemetryfb.TelemetryBatchAddStartedAtMs(b, bt.StartedAtMs)
	telemetryfb.TelemetryBatchAddPid(b, bt.PID)
}
