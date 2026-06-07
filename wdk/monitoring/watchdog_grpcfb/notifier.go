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

package watchdog_grpcfb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// maxInlineProfile bounds an inline pprof blob, leaving headroom under the frame size
	// cap. Larger profiles are reported meta-only (the bytes are dropped and noted) rather
	// than producing an unsendable oversized frame.
	maxInlineProfile = 12 << 20

	// maxMessageLen bounds the caller-influenced event message so a pathological breach
	// event cannot bloat a telemetry frame. Generously high.
	maxMessageLen = 4096

	// maxFieldValueLen bounds each event field value so a pathological breach event cannot
	// bloat a telemetry frame. Generously high.
	maxFieldValueLen = 1024

	// maxFields bounds the event field count so a pathological breach event cannot bloat a
	// telemetry frame. Generously high.
	maxFields = 128
)

var (
	// ErrEmptyProfileType is returned by Upload when called without a profile type.
	ErrEmptyProfileType = errors.New("watchdog_grpcfb: empty profile type")

	_ monitoring_domain.WatchdogNotifier = (*Notifier)(nil)

	_ monitoring_domain.WatchdogProfileUploader = (*Notifier)(nil)
)

// Notifier streams watchdog events and profiles to a telemetry sink.
type Notifier struct {
	// client is the telemetry transport that events and profiles are streamed onto.
	client *telemetry_grpcfb.Client

	// ownsClient reports whether Close should drain and close client rather than flush.
	ownsClient bool
}

// Start begins streaming (idempotent; safe when sharing a client).
//
// It is a no-op when the client is nil (e.g. the app's Dial failed) so a down sink never
// crashes the host, matching the nil-safety of the sibling analytics collector.
func (n *Notifier) Start(ctx context.Context) {
	if n == nil || n.client == nil {
		return
	}
	n.client.Start(ctx)
}

// Notify translates a watchdog threshold-breach event into a telemetry frame and enqueues
// it (non-blocking). It is a no-op when the client is nil.
//
// Takes event (monitoring_domain.WatchdogEvent) which is the threshold-breach event.
//
// Returns error which is always nil, present to satisfy the notifier interface.
func (n *Notifier) Notify(ctx context.Context, event monitoring_domain.WatchdogEvent) error {
	if n == nil || n.client == nil {
		return nil
	}
	message, _ := telemetry_grpcfb.TruncateUTF8(event.Message, maxMessageLen)
	ev := telemetry_grpcfb.WatchdogEvent{
		EventType:   string(event.EventType),
		Priority:    safeconv.IntToInt32(int(event.Priority)),
		Message:     message,
		TimestampMs: time.Now().UnixMilli(),
	}
	ev.Fields = cappedFields(event.Fields)
	n.client.AddWatchdog(ctx, ev)
	return nil
}

// cappedFields copies at most maxFields entries, each value UTF-8-truncated to
// maxFieldValueLen, so a high-cardinality or oversized field set cannot blow the frame
// budget. Map order is non-deterministic, acceptable for a lossy best-effort overflow
// cap.
//
// Takes fields (map[string]string) which is the field set to cap and truncate.
//
// Returns []telemetry_grpcfb.KV which is the capped, truncated field set, or nil.
func cappedFields(fields map[string]string) []telemetry_grpcfb.KV {
	if len(fields) == 0 {
		return nil
	}
	out := make([]telemetry_grpcfb.KV, 0, min(len(fields), maxFields))
	for k, v := range fields {
		if len(out) >= maxFields {
			break
		}
		value, _ := telemetry_grpcfb.TruncateUTF8(v, maxFieldValueLen)
		out = append(out, telemetry_grpcfb.KV{Key: k, Value: value})
	}
	return out
}

// Upload streams a captured diagnostic profile and never blocks.
//
// The gzip-compressed pprof bytes ride inline when they fit under the frame cap;
// otherwise the profile is reported meta-only.
//
// Takes profileType (string) which names the pprof profile and must be non-empty.
// Takes data ([]byte) which is the gzip-compressed pprof blob.
// Takes metadata (map[string]string) which is the profile's annotating fields.
//
// Returns error which is ErrEmptyProfileType when profileType is empty, else nil.
func (n *Notifier) Upload(ctx context.Context, profileType string, data []byte, metadata map[string]string) error {
	if profileType == "" {
		return fmt.Errorf("watchdog_grpcfb: upload: %w", ErrEmptyProfileType)
	}
	if n == nil || n.client == nil {
		return nil
	}
	pm := telemetry_grpcfb.ProfileMeta{
		ProfileType:     profileType,
		TimestampMs:     time.Now().UnixMilli(),
		SizeBytes:       int64(len(data)),
		ContentEncoding: "gzip",
		Reason:          metadata["reason"],
	}
	pm.Fields = cappedFields(metadata)
	switch {
	case len(data) == 0:

	case len(data) <= maxInlineProfile:

		pm.Blob = make([]byte, len(data))
		copy(pm.Blob, data)
	default:
		pm.Fields = append(pm.Fields, telemetry_grpcfb.KV{Key: "blob_omitted", Value: "oversize"})
	}
	n.client.AddProfile(ctx, pm)
	return nil
}

// Flush queues the current partial batch (no-op when the client is nil).
//
// Returns error which is non-nil when the client flush fails.
func (n *Notifier) Flush(ctx context.Context) error {
	if n == nil || n.client == nil {
		return nil
	}
	if err := n.client.Flush(ctx); err != nil {
		return fmt.Errorf("watchdog_grpcfb: flush: %w", err)
	}
	return nil
}

// Close drains and closes an owned client; for a shared client it only flushes. It is a
// no-op when the client is nil.
//
// Returns error which is non-nil when closing or flushing the client fails.
func (n *Notifier) Close(ctx context.Context) error {
	if n == nil || n.client == nil {
		return nil
	}
	if n.ownsClient {
		if err := n.client.Close(ctx); err != nil {
			return fmt.Errorf("watchdog_grpcfb: close: %w", err)
		}
		return nil
	}
	if err := n.client.Flush(ctx); err != nil {
		return fmt.Errorf("watchdog_grpcfb: flush on close: %w", err)
	}
	return nil
}

// New wraps an existing, shared telemetry client (its lifecycle is driven by the owner;
// this Notifier's Close only flushes).
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
//
// Returns *Notifier which streams events and profiles over the shared client.
func New(client *telemetry_grpcfb.Client) *Notifier {
	return &Notifier{client: client}
}

// Dial creates a Notifier backed by its own telemetry client and connection. Its Close
// drains and closes that client.
//
// Takes target (string) which is the telemetry endpoint address to dial.
// Takes config (telemetry_grpcfb.Config) which configures the new telemetry client.
// Takes dialOpts (...grpc.DialOption) which tune the gRPC connection.
//
// Returns *Notifier which owns and streams over the dialled client.
// Returns error which is non-nil when dialling the telemetry endpoint fails.
func Dial(target string, config telemetry_grpcfb.Config, dialOpts ...grpc.DialOption) (*Notifier, error) {
	client, err := telemetry_grpcfb.Dial(target, config, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("watchdog_grpcfb: dial %q: %w", target, err)
	}
	return &Notifier{client: client, ownsClient: true}, nil
}
