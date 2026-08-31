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
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	flatbuffers "github.com/google/flatbuffers/go"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb/telemetryfb"
)

const (
	// marshalBufferHint is the initial FlatBuffers builder capacity for a Batch.
	marshalBufferHint = 1024

	// ackBufferHint is the (much smaller) initial builder capacity for an IngestAck.
	ackBufferHint = 64

	// inlineProfileBudget caps the sum of inline pprof blobs carried by one batch.
	inlineProfileBudget = MaxMessageSize - (2 << 20)

	// maxHostnameLen caps the emitter hostname at the RFC 1035 ceiling for a fully qualified
	// name, so no real hostname is lost and the field stays tiny against the frame budget.
	maxHostnameLen = 253

	// maxInstanceIDLen caps the emitter instance identifier, which is a platform-assigned
	// replica name, the machine hostname, or a generated 36-byte UUID, so only a hostname
	// past half the RFC 1035 ceiling is shortened.
	maxInstanceIDLen = 128

	// maxServiceNameLen caps the emitting service's name, an operator-chosen label with no
	// format of its own, so it gets the same generous 128 bytes as the instance identifier.
	maxServiceNameLen = 128

	// maxServiceVersionLen caps the emitting build's version, where even a pseudo-version
	// carrying a timestamp and a commit hash stays well inside 64 bytes.
	maxServiceVersionLen = 64

	// maxEnvironmentLen caps the deployment environment, a single word such as production or
	// staging, so 64 bytes is already far more than any real value needs.
	maxEnvironmentLen = 64

	// maxRegionLen caps the service's cloud region, where the longest real region name runs
	// to about twenty characters, so 64 bytes leaves headroom for a new one.
	maxRegionLen = 64

	// blobOmittedFieldKey marks a profile whose inline blob was dropped to keep the batch
	// under the frame cap; the value records why.
	blobOmittedFieldKey = "blob_omitted"

	// blobOmittedBudget is the blob_omitted value recording that the batch budget caused the
	// inline blob to be dropped.
	blobOmittedBudget = "batch_budget"

	// pendingBlobRef is the placeholder BlobRef stamped on a profile whose inline blob was
	// dropped for the batch budget but had no out-of-band ref of its own.
	pendingBlobRef = "inline-dropped"

	// vectorPreallocCap bounds the up-front slice capacity reserved when decoding an
	// untrusted vector, so an attacker-supplied element count cannot force a large
	// allocation before any element is read. append grows the slice as the verifier-walked
	// elements are actually decoded, so real capacity never exceeds the real element count.
	vectorPreallocCap = 1024
)

var (
	// strTruncWarnOnce guards a single package-level warning when str truncates a string to
	// maxStringLen, so a log storm cannot follow from a repeatedly over-long field.
	strTruncWarnOnce sync.Once

	// identityTruncCount counts identity fields shortened to their cap.
	identityTruncCount atomic.Uint64

	// warnIdentityTruncatedOnce keeps the identity truncation warning to one line.
	warnIdentityTruncatedOnce sync.Once

	// strTruncCount counts every truncation, not just the first.
	strTruncCount atomic.Uint64
)

var (
	// ErrFrameTooLarge is returned by Marshal when a serialised frame exceeds
	// MaxMessageSize. The send path detects it to drop just the offending batch (with an
	// accurate cause) rather than handing an oversized frame to the transport, where it
	// would be rejected whole and tear down the shared stream.
	ErrFrameTooLarge = errors.New("telemetry_grpcfb: marshalled frame exceeds MaxMessageSize")
)

var (
	// kvFields is the verifier field spec for a KV table (key, value).
	kvFields = []field{{voffset: 4, kind: kString}, {voffset: 6, kind: kString}}

	// analyticsFields is the verifier field spec for an AnalyticsEvent table.
	analyticsFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kInt64},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kString},
		{voffset: 18, kind: kInt32},
		{voffset: 20, kind: kInt64},
		{voffset: 22, kind: kString},
		{voffset: 24, kind: kString},
		{voffset: 26, kind: kString},
		{voffset: 28, kind: kString},
		{voffset: 30, kind: kString},
		{voffset: 32, kind: kString},
		{voffset: 34, kind: kString},
		{voffset: 36, kind: kString},
		{voffset: 38, kind: kString},
		{voffset: 40, kind: kVectorTable, elem: kvFields},
	}

	// watchdogFields is the verifier field spec for a WatchdogEvent table.
	watchdogFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kInt32},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kInt64},
		{voffset: 12, kind: kVectorTable, elem: kvFields},
	}

	// logFields is the verifier field spec for a LogLine table.
	logFields = []field{
		{voffset: 4, kind: kInt64},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kVectorTable, elem: kvFields},
	}

	// spanFields is the verifier field spec for a Span table.
	spanFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kInt64},
		{voffset: 18, kind: kInt64},
		{voffset: 20, kind: kString},
		{voffset: 22, kind: kVectorTable, elem: kvFields},
	}

	// metricFields is the verifier field spec for a MetricPoint table.
	metricFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kInt64},
		{voffset: 10, kind: kFloat64},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kVectorTable, elem: kvFields},
	}

	// errorFields is the verifier field spec for an ErrorEvent table.
	errorFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kInt64},
		{voffset: 16, kind: kString},
		{voffset: 18, kind: kString},
		{voffset: 20, kind: kString},
		{voffset: 22, kind: kBool},
		{voffset: 24, kind: kString},
		{voffset: 26, kind: kString},
		{voffset: 28, kind: kVectorTable, elem: kvFields},
	}

	// profileFields is the verifier field spec for a ProfileMeta table.
	profileFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kInt64},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kInt64},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kVectorTable, elem: kvFields},
		{voffset: 18, kind: kVectorByte},
	}

	// workerFields is the verifier field spec for a WorkerEvent table.
	workerFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kString},
		{voffset: 18, kind: kInt32},
		{voffset: 20, kind: kInt64},
		{voffset: 22, kind: kInt64},
		{voffset: 24, kind: kString},
		{voffset: 26, kind: kVectorTable, elem: kvFields},
	}

	// queryStatFields is the verifier field spec for a QueryStat table.
	queryStatFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kInt64},
		{voffset: 16, kind: kInt64},
		{voffset: 18, kind: kInt64},
		{voffset: 20, kind: kInt64},
		{voffset: 22, kind: kVectorTable, elem: kvFields},
	}

	// emailFields is the verifier field spec for an EmailEvent table.
	emailFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kString},
		{voffset: 12, kind: kString},
		{voffset: 14, kind: kString},
		{voffset: 16, kind: kString},
		{voffset: 18, kind: kString},
		{voffset: 20, kind: kInt64},
		{voffset: 22, kind: kVectorTable, elem: kvFields},
	}

	// batchFields is the verifier field spec for a TelemetryBatch table.
	batchFields = []field{
		{voffset: 4, kind: kString},
		{voffset: 6, kind: kString},
		{voffset: 8, kind: kString},
		{voffset: 10, kind: kInt64},
		{voffset: 12, kind: kInt64},
		{voffset: 14, kind: kVectorTable, elem: analyticsFields},
		{voffset: 16, kind: kVectorTable, elem: watchdogFields},
		{voffset: 18, kind: kVectorTable, elem: logFields},
		{voffset: 20, kind: kVectorTable, elem: spanFields},
		{voffset: 22, kind: kVectorTable, elem: metricFields},
		{voffset: 24, kind: kVectorTable, elem: errorFields},
		{voffset: 26, kind: kVectorTable, elem: profileFields},
		{voffset: 28, kind: kVectorTable, elem: workerFields},
		{voffset: 30, kind: kVectorTable, elem: queryStatFields},
		{voffset: 32, kind: kVectorTable, elem: emailFields},
		{voffset: 34, kind: kString},
		{voffset: 36, kind: kString},
		{voffset: 38, kind: kString},
		{voffset: 40, kind: kString},
		{voffset: 42, kind: kString},
		{voffset: 44, kind: kString},
		{voffset: 46, kind: kInt64},
		{voffset: 48, kind: kInt32},
	}

	// ackFields is the verifier field spec for an IngestAck table.
	ackFields = []field{
		{voffset: 4, kind: kBool},
		{voffset: 6, kind: kInt64},
		{voffset: 8, kind: kInt64},
		{voffset: 10, kind: kString},
	}
)

// KV is a generic string key/value pair (event properties, span attributes, labels).
type KV struct {
	// Key is the pair's key.
	Key string

	// Value is the pair's value.
	Value string
}

// AnalyticsEvent mirrors piko's analytics_dto.Event.
type AnalyticsEvent struct {
	// Kind is the event kind (pageview, action, custom).
	Kind string

	// Hostname is the request host.
	Hostname string

	// URL is the full request URL including query parameters.
	URL string

	// Path is the URL path of the request.
	Path string

	// MatchedPattern is the route pattern that matched.
	MatchedPattern string

	// Method is the HTTP method.
	Method string

	// Referrer is the Referer header value.
	Referrer string

	// UserAgent is the User-Agent header value.
	UserAgent string

	// ClientIP is the resolved client IP.
	ClientIP string

	// Locale is the request locale.
	Locale string

	// UserID is the authenticated user's identifier, empty if anonymous.
	UserID string

	// ActionName is the name of the server action, empty for page views.
	ActionName string

	// EventName is an explicit name for custom analytics events.
	EventName string

	// RevenueAmount is the monetary amount for revenue events.
	RevenueAmount string

	// RevenueCurrency is the currency code for RevenueAmount.
	RevenueCurrency string

	// Properties holds arbitrary key/value metadata.
	Properties []KV

	// TimestampMs is the event time in epoch milliseconds.
	TimestampMs int64

	// DurationMs is the time taken to handle the request, in milliseconds.
	DurationMs int64

	// StatusCode is the HTTP response status code.
	StatusCode int32
}

// WatchdogEvent mirrors piko's monitoring_domain.WatchdogEvent.
type WatchdogEvent struct {
	// EventType names the watchdog event type.
	EventType string

	// Message is the human-readable event message.
	Message string

	// Fields holds arbitrary key/value metadata.
	Fields []KV

	// TimestampMs is the event time in epoch milliseconds.
	TimestampMs int64

	// Priority is the event priority.
	Priority int32
}

// LogLine is a single structured log record.
type LogLine struct {
	// Level is the log severity level.
	Level string

	// Logger is the name of the logger that emitted the record.
	Logger string

	// Message is the log message text.
	Message string

	// TraceID correlates the record with a distributed trace.
	TraceID string

	// SpanID correlates the record with a trace span.
	SpanID string

	// Fields holds arbitrary key/value metadata.
	Fields []KV

	// TimestampMs is the record time in epoch milliseconds.
	TimestampMs int64
}

// Span is one unit of a distributed trace.
type Span struct {
	// TraceID is the identifier of the owning trace.
	TraceID string

	// SpanID is this span's identifier.
	SpanID string

	// ParentID is the parent span's identifier, empty for a root span.
	ParentID string

	// Service is the name of the service that produced the span.
	Service string

	// Operation is the operation the span represents.
	Operation string

	// Kind is the span kind (client, server, internal, etc.).
	Kind string

	// Status is the span outcome status.
	Status string

	// Attributes holds arbitrary key/value metadata.
	Attributes []KV

	// StartMs is the span start time in epoch milliseconds.
	StartMs int64

	// DurationUs is the span duration in microseconds.
	DurationUs int64
}

// MetricPoint is one numeric sample.
type MetricPoint struct {
	// Name is the metric name.
	Name string

	// Kind is the metric kind (counter, gauge, histogram, etc.).
	Kind string

	// Unit is the unit of measurement.
	Unit string

	// Labels holds arbitrary key/value dimensions.
	Labels []KV

	// TimestampMs is the sample time in epoch milliseconds.
	TimestampMs int64

	// Value is the sampled numeric value.
	Value float64
}

// ErrorEvent is one error/exception occurrence.
type ErrorEvent struct {
	// Fingerprint groups occurrences of the same error.
	Fingerprint string

	// Type is the error or exception type name.
	Type string

	// Value is the error message or value.
	Value string

	// Culprit is the code location blamed for the error.
	Culprit string

	// Level is the error severity level.
	Level string

	// Release is the application release that produced the error.
	Release string

	// Environment is the deployment environment.
	Environment string

	// UserID is the affected user's identifier, empty if anonymous.
	UserID string

	// StackJSON is the JSON-encoded stack trace.
	StackJSON string

	// BreadcrumbsJSON is the JSON-encoded breadcrumb trail.
	BreadcrumbsJSON string

	// Context holds arbitrary key/value metadata.
	Context []KV

	// TimestampMs is the occurrence time in epoch milliseconds.
	TimestampMs int64

	// Handled reports whether the error was caught and handled.
	Handled bool
}

// ProfileMeta describes a captured diagnostic profile. The compressed pprof bytes may
// travel inline (Blob, bounded by the frame size cap) or out-of-band (BlobRef).
type ProfileMeta struct {
	// ProfileType is the kind of profile captured (cpu, heap, etc.).
	ProfileType string

	// Reason is the human-readable trigger for this capture (e.g. the watchdog threshold
	// that fired).
	//
	// The watchdog adapter derives it from the caller's "reason" metadata key; leave it
	// empty when no reason is available.
	Reason string

	// ContentEncoding is the encoding of the inline blob (e.g. gzip).
	ContentEncoding string

	// BlobRef is the out-of-band reference to the profile bytes when not carried inline.
	BlobRef string

	// Fields holds arbitrary key/value metadata.
	Fields []KV

	// Blob is the inline compressed pprof bytes, bounded by the frame size cap.
	Blob []byte

	// TimestampMs is the capture time in epoch milliseconds.
	TimestampMs int64

	// SizeBytes is the original uncapped size of the profile in bytes.
	SizeBytes int64
}

// WorkerEvent is one run-telemetry record from a piko site's worker/job subsystem,
// modelled generically: Category names which lifecycle shape this record is (job_run |
// attempt | transition | chain | batch | dlq | worker). RunID correlates the records of
// one run; ParentID links a run to its chain/batch parent.
type WorkerEvent struct {
	// EventID is this record's unique identifier.
	EventID string

	// RunID correlates the records of one run.
	RunID string

	// ParentID links a run to its chain or batch parent.
	ParentID string

	// Category names the lifecycle shape of this record.
	Category string

	// Queue is the queue the work was drawn from.
	Queue string

	// Worker is the name of the worker that handled the run.
	Worker string

	// Status is the run outcome status.
	Status string

	// Error is the failure message, empty on success.
	Error string

	// Attrs holds arbitrary key/value metadata.
	Attrs []KV

	// TsMs is the record time in epoch milliseconds.
	TsMs int64

	// DurationMs is the run duration in milliseconds.
	DurationMs int64

	// Attempt is the retry attempt number.
	Attempt int32
}

// QueryStat is one database query observation from a piko site's data layer.
//
// The shape is generic so a single record covers reads and writes. Statement carries the
// SQL or a normalised fingerprint; Calls is how many executions this record aggregates (1
// = a single statement), with DurationMs and Rows summed over those calls.
type QueryStat struct {
	// Connection is the name of the connection that ran the query.
	Connection string

	// Statement is the SQL or a normalised fingerprint of it.
	Statement string

	// Operation is the query operation (select, insert, etc.).
	Operation string

	// Status is the query outcome status.
	Status string

	// Error is the failure message, empty on success.
	Error string

	// Attrs holds arbitrary key/value metadata.
	Attrs []KV

	// TsMs is the observation time in epoch milliseconds.
	TsMs int64

	// DurationMs is the total duration over Calls executions, in milliseconds.
	DurationMs int64

	// Rows is the total row count over Calls executions.
	Rows int64

	// Calls is how many executions this record aggregates.
	Calls int64
}

// EmailEvent is one email lifecycle observation from a piko site's mail subsystem.
// MessageID correlates the records of one message; Event names the lifecycle transition
// (queued | sent | delivered | bounced | opened | failed).
type EmailEvent struct {
	// MessageID correlates the records of one message.
	MessageID string

	// Provider is the name of the mail provider.
	Provider string

	// Template is the template the message was rendered from.
	Template string

	// Recipient is the message recipient address.
	Recipient string

	// Subject is the message subject line.
	Subject string

	// Event names the lifecycle transition.
	Event string

	// Status is the lifecycle outcome status.
	Status string

	// Error is the failure message, empty on success.
	Error string

	// Attrs holds arbitrary key/value metadata.
	Attrs []KV

	// TsMs is the observation time in epoch milliseconds.
	TsMs int64
}

// Batch is one streamed telemetry frame (stream envelope + parallel typed vectors).
type Batch struct {
	// SiteID identifies the originating piko site.
	SiteID string

	// APIKey authenticates the batch.
	APIKey string

	// Source names the producing subsystem.
	Source string

	// InstanceID identifies the emitting process among sibling replicas of the same service,
	// stable for the process lifetime.
	InstanceID string

	// Hostname is the emitting machine's hostname, empty when it could not be resolved.
	Hostname string

	// ServiceName is the emitting service's name.
	ServiceName string

	// ServiceVersion is the emitting build's version.
	ServiceVersion string

	// Environment is the deployment environment ("production", "staging"), empty when unset.
	Environment string

	// Region is the SERVICE's cloud region, empty off-cloud. It is not the user's region:
	// that needs licensed GeoIP and is not derivable here.
	Region string

	// Analytics holds the analytics events in the frame.
	Analytics []AnalyticsEvent

	// Watchdog holds the watchdog events in the frame.
	Watchdog []WatchdogEvent

	// Logs holds the log lines in the frame.
	Logs []LogLine

	// Spans holds the trace spans in the frame.
	Spans []Span

	// Metrics holds the metric points in the frame.
	Metrics []MetricPoint

	// Errors holds the error events in the frame.
	Errors []ErrorEvent

	// Profiles holds the captured profiles in the frame.
	Profiles []ProfileMeta

	// Workers holds the worker events in the frame.
	Workers []WorkerEvent

	// QueryStats holds the query observations in the frame.
	QueryStats []QueryStat

	// Emails holds the email events in the frame.
	Emails []EmailEvent

	// SentAtMs is the send time in epoch milliseconds.
	SentAtMs int64

	// Seq is the per-stream sequence number of the frame.
	Seq int64

	// StartedAtMs is when the emitting process started, in epoch milliseconds.
	StartedAtMs int64

	// PID is the emitting process's operating-system identifier.
	PID int32
}

// EventCount is the total number of events carried by the batch (all frame types).
//
// Returns int which is the summed length of every typed vector in the batch.
func (bt *Batch) EventCount() int {
	return len(bt.Analytics) + len(bt.Watchdog) + len(bt.Logs) + len(bt.Spans) +
		len(bt.Metrics) + len(bt.Errors) + len(bt.Profiles) + len(bt.Workers) +
		len(bt.QueryStats) + len(bt.Emails)
}

// IngestAck is the server's single response after the client half-closes the stream.
type IngestAck struct {
	// Message is an optional human-readable status message.
	Message string

	// Frames is the number of frames the server accepted.
	Frames int64

	// Events is the number of events the server accepted.
	Events int64

	// OK reports whether ingestion succeeded.
	OK bool
}

// Marshal serialises the batch to a FlatBuffer.
//
// Returns []byte which is the finished FlatBuffer frame.
// Returns error which is ErrFrameTooLarge when the frame exceeds MaxMessageSize.
func (bt *Batch) Marshal() ([]byte, error) {
	b := flatbuffers.NewBuilder(marshalBufferHint)

	aVec := marshalVector(b, bt.Analytics, buildAnalytics)
	wVec := marshalVector(b, bt.Watchdog, buildWatchdog)
	lVec := marshalVector(b, bt.Logs, buildLog)
	sVec := marshalVector(b, bt.Spans, buildSpan)
	mVec := marshalVector(b, bt.Metrics, buildMetric)
	eVec := marshalVector(b, bt.Errors, buildError)
	pVec := marshalVector(b, bt.Profiles, buildProfile)
	wkVec := marshalVector(b, bt.Workers, buildWorker)
	qVec := marshalVector(b, bt.QueryStats, buildQueryStat)
	emVec := marshalVector(b, bt.Emails, buildEmail)

	site := str(b, bt.SiteID)
	key := str(b, bt.APIKey)
	src := str(b, bt.Source)
	identity := marshalIdentity(b, bt)

	telemetryfb.TelemetryBatchStart(b)
	addOffset(b, site, telemetryfb.TelemetryBatchAddSiteId)
	addOffset(b, key, telemetryfb.TelemetryBatchAddApiKey)
	addOffset(b, src, telemetryfb.TelemetryBatchAddSource)
	telemetryfb.TelemetryBatchAddSentAtMs(b, bt.SentAtMs)
	telemetryfb.TelemetryBatchAddSeq(b, bt.Seq)
	addOffset(b, aVec, telemetryfb.TelemetryBatchAddAnalytics)
	addOffset(b, wVec, telemetryfb.TelemetryBatchAddWatchdog)
	addOffset(b, lVec, telemetryfb.TelemetryBatchAddLogs)
	addOffset(b, sVec, telemetryfb.TelemetryBatchAddSpans)
	addOffset(b, mVec, telemetryfb.TelemetryBatchAddMetrics)
	addOffset(b, eVec, telemetryfb.TelemetryBatchAddErrors)
	addOffset(b, pVec, telemetryfb.TelemetryBatchAddProfiles)
	addOffset(b, wkVec, telemetryfb.TelemetryBatchAddWorkers)
	addOffset(b, qVec, telemetryfb.TelemetryBatchAddQueryStats)
	addOffset(b, emVec, telemetryfb.TelemetryBatchAddEmailEvents)
	addIdentity(b, bt, identity)
	b.Finish(telemetryfb.TelemetryBatchEnd(b))
	out := b.FinishedBytes()

	if len(out) > MaxMessageSize {
		return nil, fmt.Errorf("%w: %d > %d", ErrFrameTooLarge, len(out), MaxMessageSize)
	}
	return out, nil
}

// Unmarshal validates the untrusted buffer, then reads via generated accessors.
//
// Takes data ([]byte) which is the untrusted FlatBuffer frame to decode.
//
// Returns error which is non-nil when verification or decoding fails.
func (bt *Batch) Unmarshal(data []byte) error {
	if err := verifyMessage(data, batchFields); err != nil {
		return fmt.Errorf("telemetry_grpcfb: verify batch: %w", err)
	}
	root := telemetryfb.GetRootAsTelemetryBatch(data, 0)
	bt.SiteID = string(root.SiteId())
	bt.APIKey = string(root.ApiKey())
	bt.Source = string(root.Source())
	bt.SentAtMs = root.SentAtMs()
	bt.Seq = root.Seq()

	bt.Analytics = unmarshalVector(root.AnalyticsLength(), root.Analytics, readAnalytics)
	bt.Watchdog = unmarshalVector(root.WatchdogLength(), root.Watchdog, readWatchdog)
	bt.Logs = unmarshalVector(root.LogsLength(), root.Logs, readLog)
	bt.Spans = unmarshalVector(root.SpansLength(), root.Spans, readSpan)
	bt.Metrics = unmarshalVector(root.MetricsLength(), root.Metrics, readMetric)
	bt.Errors = unmarshalVector(root.ErrorsLength(), root.Errors, readError)
	bt.Profiles = unmarshalVector(root.ProfilesLength(), root.Profiles, readProfile)
	bt.Workers = unmarshalVector(root.WorkersLength(), root.Workers, readWorker)
	bt.QueryStats = unmarshalVector(root.QueryStatsLength(), root.QueryStats, readQueryStat)
	bt.Emails = unmarshalVector(root.EmailEventsLength(), root.EmailEvents, readEmail)

	bt.InstanceID = clip(string(root.InstanceId()), maxInstanceIDLen)
	bt.Hostname = clip(string(root.Hostname()), maxHostnameLen)
	bt.ServiceName = clip(string(root.ServiceName()), maxServiceNameLen)
	bt.ServiceVersion = clip(string(root.ServiceVersion()), maxServiceVersionLen)
	bt.Environment = clip(string(root.Environment()), maxEnvironmentLen)
	bt.Region = clip(string(root.Region()), maxRegionLen)

	if startedAt := root.StartedAtMs(); startedAt >= 0 {
		bt.StartedAtMs = startedAt
	}
	if pid := root.Pid(); pid >= 0 {
		bt.PID = pid
	}
	return nil
}

// capInlineProfiles keeps the sum of inline profile blobs within inlineProfileBudget.
//
// Walking the profiles largest-first, once the running total would breach the budget the
// offending blob is dropped (its bytes cleared) and the profile is downgraded to a
// blob_ref placeholder with a blob_omitted marker, so the marshalled frame can never
// exceed the frame cap and be silently dropped. SizeBytes is preserved so the sink still
// knows the original size.
//
// Returns int which is the number of profiles whose inline blob was dropped.
func (bt *Batch) capInlineProfiles() int {
	var total int
	for i := range bt.Profiles {
		total += len(bt.Profiles[i].Blob)
	}
	if total <= inlineProfileBudget {
		return 0
	}

	order := make([]int, 0, len(bt.Profiles))
	for i := range bt.Profiles {
		if len(bt.Profiles[i].Blob) > 0 {
			order = append(order, i)
		}
	}
	slices.SortFunc(order, func(a, b int) int {
		if d := cmp.Compare(len(bt.Profiles[b].Blob), len(bt.Profiles[a].Blob)); d != 0 {
			return d
		}
		return cmp.Compare(b, a)
	})
	dropped := 0
	for _, i := range order {
		if total <= inlineProfileBudget {
			break
		}
		total -= len(bt.Profiles[i].Blob)
		bt.Profiles[i].Blob = nil

		bt.Profiles[i].Fields = append(bt.Profiles[i].Fields, KV{Key: blobOmittedFieldKey, Value: blobOmittedBudget})
		if bt.Profiles[i].BlobRef == "" {
			bt.Profiles[i].BlobRef = pendingBlobRef
		}
		dropped++
	}
	return dropped
}

// marshalVector builds each element of src into a table and packs the resulting offsets
// into one FlatBuffers vector, returning 0 for an empty slice.
//
// Elements are built before the vector is opened (FlatBuffers builds bottom-up).
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes src ([]T) which is the slice of elements to encode.
// Takes build (func(*flatbuffers.Builder, *T) flatbuffers.UOffsetT) encoding one element.
//
// Returns flatbuffers.UOffsetT which is the vector offset, or 0 when src is empty.
func marshalVector[T any](b *flatbuffers.Builder, src []T, build func(*flatbuffers.Builder, *T) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(src) == 0 {
		return 0
	}
	offs := make([]flatbuffers.UOffsetT, len(src))
	for i := range src {
		offs[i] = build(b, &src[i])
	}
	return offsetVector(b, offs)
}

// unmarshalVector reads a FlatBuffers vector of length n, decoding each present element
// through read into the returned slice.
//
// Takes n (int) which is the vector length.
// Takes fill (func(*F, int) bool) which is the generated indexed accessor returning ok.
// Takes read (func(*F) T) which decodes one filled element.
//
// Returns []T which holds the decoded elements, or nil when n is zero.
func unmarshalVector[F any, T any](n int, fill func(*F, int) bool, read func(*F) T) []T {
	if n <= 0 {
		return nil
	}
	out := make([]T, 0, min(n, vectorPreallocCap))
	var e F
	for i := range n {
		if fill(&e, i) {
			out = append(out, read(&e))
		}
	}
	return out
}

// Marshal serialises the ack to a FlatBuffer.
//
// Returns []byte which is the finished FlatBuffer frame.
// Returns error which is always nil (the signature satisfies fbMessage).
func (a *IngestAck) Marshal() ([]byte, error) {
	b := flatbuffers.NewBuilder(ackBufferHint)
	msg := str(b, a.Message)
	telemetryfb.IngestAckStart(b)
	telemetryfb.IngestAckAddOk(b, a.OK)
	telemetryfb.IngestAckAddFrames(b, a.Frames)
	telemetryfb.IngestAckAddEvents(b, a.Events)
	addOffset(b, msg, telemetryfb.IngestAckAddMessage)
	b.Finish(telemetryfb.IngestAckEnd(b))
	return b.FinishedBytes(), nil
}

// Unmarshal validates the untrusted buffer, then reads via generated accessors.
//
// Takes data ([]byte) which is the untrusted FlatBuffer frame to decode.
//
// Returns error which is non-nil when verification or decoding fails.
func (a *IngestAck) Unmarshal(data []byte) error {
	if err := verifyMessage(data, ackFields); err != nil {
		return fmt.Errorf("telemetry_grpcfb: verify ack: %w", err)
	}
	root := telemetryfb.GetRootAsIngestAck(data, 0)
	a.OK = root.Ok()
	a.Frames = root.Frames()
	a.Events = root.Events()
	a.Message = string(root.Message())
	return nil
}

// str creates a FlatBuffers string, returning 0 (absent) for the empty string so unset
// fields stay out of the buffer.
//
// As a defence-in-depth backstop it truncates any string longer than maxStringLen (UTF-8
// safe) so the producer can never emit a string the structural verifier would reject on
// the receive side; collectors apply their own tighter, marked field caps upstream, so
// this only fires on a pathological value.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes s (string) which is the string to encode.
//
// Returns flatbuffers.UOffsetT which is the string offset, or 0 when s is empty.
func str(b *flatbuffers.Builder, s string) flatbuffers.UOffsetT {
	if s == "" {
		return 0
	}
	if len(s) > maxStringLen {
		var truncated bool
		s, truncated = TruncateUTF8(s, maxStringLen)
		if truncated {
			strTruncCount.Add(1)
			stringsTruncatedCount.Add(context.Background(), 1)
			strTruncWarnOnce.Do(func() {
				log.Warn("telemetry string truncated to maxStringLen before encoding; "+
					"read TruncatedStrings for the running total",
					logger_domain.Int("max_bytes", maxStringLen))
			})
		}
	}
	return b.CreateString(s)
}

// TruncateUTF8 shortens s to at most maxBytes bytes without splitting a multi-byte rune.
//
// A naive s[:maxBytes] can sever a rune mid-sequence and emit invalid UTF-8; this backs
// up to the preceding rune boundary instead. It underpins every telemetry field length
// cap so off-box strings stay valid UTF-8 and within the wire frame's per-string limit.
//
// Takes s (string) which is the string to shorten.
// Takes maxBytes (int) which is the maximum byte length (treated as 0 when negative).
//
// Returns string which is s truncated to a rune boundary.
// Returns bool which is true when truncation occurred.
func TruncateUTF8(s string, maxBytes int) (string, bool) {
	maxBytes = max(maxBytes, 0)
	if len(s) <= maxBytes {
		return s, false
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// addOffset applies add only when off is non-zero, so an absent string or empty vector
// (offset 0) stays out of the buffer rather than being written as a present-but-empty
// field.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes off (flatbuffers.UOffsetT) which is the field offset (0 means absent).
// Takes add (func(*flatbuffers.Builder, flatbuffers.UOffsetT)) which writes the field.
func addOffset(b *flatbuffers.Builder, off flatbuffers.UOffsetT, add func(*flatbuffers.Builder, flatbuffers.UOffsetT)) {
	if off != 0 {
		add(b, off)
	}
}

// offsetVector packs a slice of table offsets into a FlatBuffers vector.
//
// Elements must already be built (FlatBuffers builds bottom-up; nothing else may be under
// construction while the vector is open).
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes offs ([]flatbuffers.UOffsetT) which are the already-built element offsets.
//
// Returns flatbuffers.UOffsetT which is the vector offset.
func offsetVector(b *flatbuffers.Builder, offs []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	b.StartVector(sizeU32, len(offs), sizeU32)
	for _, off := range slices.Backward(offs) {
		b.PrependUOffsetT(off)
	}
	return b.EndVector(len(offs))
}

// buildKV encodes one KV pair into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes kv (KV) which is the key/value pair to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildKV(b *flatbuffers.Builder, kv KV) flatbuffers.UOffsetT {
	k := str(b, kv.Key)
	v := str(b, kv.Value)
	telemetryfb.KVStart(b)
	addOffset(b, k, telemetryfb.KVAddKey)
	addOffset(b, v, telemetryfb.KVAddValue)
	return telemetryfb.KVEnd(b)
}

// buildKVVector encodes a slice of KV pairs into one FlatBuffers vector.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes kvs ([]KV) which are the pairs to encode.
//
// Returns flatbuffers.UOffsetT which is the vector offset, or 0 when kvs is empty.
func buildKVVector(b *flatbuffers.Builder, kvs []KV) flatbuffers.UOffsetT {
	if len(kvs) == 0 {
		return 0
	}
	offs := make([]flatbuffers.UOffsetT, len(kvs))
	for i := range kvs {
		offs[i] = buildKV(b, kvs[i])
	}
	return offsetVector(b, offs)
}

// buildAnalytics encodes one AnalyticsEvent into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*AnalyticsEvent) which is the event to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildAnalytics(b *flatbuffers.Builder, e *AnalyticsEvent) flatbuffers.UOffsetT {
	props := buildKVVector(b, e.Properties)
	kind := str(b, e.Kind)
	host := str(b, e.Hostname)
	url := str(b, e.URL)
	path := str(b, e.Path)
	mp := str(b, e.MatchedPattern)
	method := str(b, e.Method)
	ref := str(b, e.Referrer)
	ua := str(b, e.UserAgent)
	ip := str(b, e.ClientIP)
	loc := str(b, e.Locale)
	uid := str(b, e.UserID)
	an := str(b, e.ActionName)
	en := str(b, e.EventName)
	rev := str(b, e.RevenueAmount)
	cur := str(b, e.RevenueCurrency)
	telemetryfb.AnalyticsEventStart(b)
	addOffset(b, kind, telemetryfb.AnalyticsEventAddKind)
	telemetryfb.AnalyticsEventAddTimestampMs(b, e.TimestampMs)
	addOffset(b, host, telemetryfb.AnalyticsEventAddHostname)
	addOffset(b, url, telemetryfb.AnalyticsEventAddUrl)
	addOffset(b, path, telemetryfb.AnalyticsEventAddPath)
	addOffset(b, mp, telemetryfb.AnalyticsEventAddMatchedPattern)
	addOffset(b, method, telemetryfb.AnalyticsEventAddMethod)
	telemetryfb.AnalyticsEventAddStatusCode(b, e.StatusCode)
	telemetryfb.AnalyticsEventAddDurationMs(b, e.DurationMs)
	addOffset(b, ref, telemetryfb.AnalyticsEventAddReferrer)
	addOffset(b, ua, telemetryfb.AnalyticsEventAddUserAgent)
	addOffset(b, ip, telemetryfb.AnalyticsEventAddClientIp)
	addOffset(b, loc, telemetryfb.AnalyticsEventAddLocale)
	addOffset(b, uid, telemetryfb.AnalyticsEventAddUserId)
	addOffset(b, an, telemetryfb.AnalyticsEventAddActionName)
	addOffset(b, en, telemetryfb.AnalyticsEventAddEventName)
	addOffset(b, rev, telemetryfb.AnalyticsEventAddRevenueAmount)
	addOffset(b, cur, telemetryfb.AnalyticsEventAddRevenueCurrency)
	if props != 0 {
		telemetryfb.AnalyticsEventAddProperties(b, props)
	}
	return telemetryfb.AnalyticsEventEnd(b)
}

// buildWatchdog encodes one WatchdogEvent into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*WatchdogEvent) which is the event to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildWatchdog(b *flatbuffers.Builder, e *WatchdogEvent) flatbuffers.UOffsetT {
	fields := buildKVVector(b, e.Fields)
	et := str(b, e.EventType)
	msg := str(b, e.Message)
	telemetryfb.WatchdogEventStart(b)
	addOffset(b, et, telemetryfb.WatchdogEventAddEventType)
	telemetryfb.WatchdogEventAddPriority(b, e.Priority)
	addOffset(b, msg, telemetryfb.WatchdogEventAddMessage)
	telemetryfb.WatchdogEventAddTimestampMs(b, e.TimestampMs)
	if fields != 0 {
		telemetryfb.WatchdogEventAddFields(b, fields)
	}
	return telemetryfb.WatchdogEventEnd(b)
}

// buildLog encodes one LogLine into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*LogLine) which is the log line to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildLog(b *flatbuffers.Builder, e *LogLine) flatbuffers.UOffsetT {
	fields := buildKVVector(b, e.Fields)
	level := str(b, e.Level)
	logger := str(b, e.Logger)
	msg := str(b, e.Message)
	tid := str(b, e.TraceID)
	sid := str(b, e.SpanID)
	telemetryfb.LogLineStart(b)
	telemetryfb.LogLineAddTimestampMs(b, e.TimestampMs)
	addOffset(b, level, telemetryfb.LogLineAddLevel)
	addOffset(b, logger, telemetryfb.LogLineAddLogger)
	addOffset(b, msg, telemetryfb.LogLineAddMessage)
	addOffset(b, tid, telemetryfb.LogLineAddTraceId)
	addOffset(b, sid, telemetryfb.LogLineAddSpanId)
	if fields != 0 {
		telemetryfb.LogLineAddFields(b, fields)
	}
	return telemetryfb.LogLineEnd(b)
}

// buildSpan encodes one Span into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*Span) which is the span to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildSpan(b *flatbuffers.Builder, e *Span) flatbuffers.UOffsetT {
	attrs := buildKVVector(b, e.Attributes)
	tid := str(b, e.TraceID)
	sid := str(b, e.SpanID)
	pid := str(b, e.ParentID)
	service := str(b, e.Service)
	op := str(b, e.Operation)
	kind := str(b, e.Kind)
	status := str(b, e.Status)
	telemetryfb.SpanStart(b)
	addOffset(b, tid, telemetryfb.SpanAddTraceId)
	addOffset(b, sid, telemetryfb.SpanAddSpanId)
	addOffset(b, pid, telemetryfb.SpanAddParentId)
	addOffset(b, service, telemetryfb.SpanAddService)
	addOffset(b, op, telemetryfb.SpanAddOperation)
	addOffset(b, kind, telemetryfb.SpanAddKind)
	telemetryfb.SpanAddStartMs(b, e.StartMs)
	telemetryfb.SpanAddDurationUs(b, e.DurationUs)
	addOffset(b, status, telemetryfb.SpanAddStatus)
	if attrs != 0 {
		telemetryfb.SpanAddAttributes(b, attrs)
	}
	return telemetryfb.SpanEnd(b)
}

// buildMetric encodes one MetricPoint into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*MetricPoint) which is the metric point to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildMetric(b *flatbuffers.Builder, e *MetricPoint) flatbuffers.UOffsetT {
	labels := buildKVVector(b, e.Labels)
	name := str(b, e.Name)
	kind := str(b, e.Kind)
	unit := str(b, e.Unit)
	telemetryfb.MetricPointStart(b)
	addOffset(b, name, telemetryfb.MetricPointAddName)
	addOffset(b, kind, telemetryfb.MetricPointAddKind)
	telemetryfb.MetricPointAddTimestampMs(b, e.TimestampMs)
	telemetryfb.MetricPointAddValue(b, e.Value)
	addOffset(b, unit, telemetryfb.MetricPointAddUnit)
	if labels != 0 {
		telemetryfb.MetricPointAddLabels(b, labels)
	}
	return telemetryfb.MetricPointEnd(b)
}

// buildError encodes one ErrorEvent into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*ErrorEvent) which is the error event to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildError(b *flatbuffers.Builder, e *ErrorEvent) flatbuffers.UOffsetT {
	ctx := buildKVVector(b, e.Context)
	fp := str(b, e.Fingerprint)
	typ := str(b, e.Type)
	val := str(b, e.Value)
	culprit := str(b, e.Culprit)
	level := str(b, e.Level)
	rel := str(b, e.Release)
	env := str(b, e.Environment)
	uid := str(b, e.UserID)
	stack := str(b, e.StackJSON)
	bc := str(b, e.BreadcrumbsJSON)
	telemetryfb.ErrorEventStart(b)
	addOffset(b, fp, telemetryfb.ErrorEventAddFingerprint)
	addOffset(b, typ, telemetryfb.ErrorEventAddType)
	addOffset(b, val, telemetryfb.ErrorEventAddValue)
	addOffset(b, culprit, telemetryfb.ErrorEventAddCulprit)
	addOffset(b, level, telemetryfb.ErrorEventAddLevel)
	telemetryfb.ErrorEventAddTimestampMs(b, e.TimestampMs)
	addOffset(b, rel, telemetryfb.ErrorEventAddRelease)
	addOffset(b, env, telemetryfb.ErrorEventAddEnvironment)
	addOffset(b, uid, telemetryfb.ErrorEventAddUserId)
	telemetryfb.ErrorEventAddHandled(b, e.Handled)
	addOffset(b, stack, telemetryfb.ErrorEventAddStackJson)
	addOffset(b, bc, telemetryfb.ErrorEventAddBreadcrumbsJson)
	if ctx != 0 {
		telemetryfb.ErrorEventAddContext(b, ctx)
	}
	return telemetryfb.ErrorEventEnd(b)
}

// buildProfile encodes one ProfileMeta into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*ProfileMeta) which is the profile to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildProfile(b *flatbuffers.Builder, e *ProfileMeta) flatbuffers.UOffsetT {
	fields := buildKVVector(b, e.Fields)
	var blob flatbuffers.UOffsetT
	if len(e.Blob) > 0 {
		blob = b.CreateByteVector(e.Blob)
	}
	pt := str(b, e.ProfileType)
	reason := str(b, e.Reason)
	enc := str(b, e.ContentEncoding)
	ref := str(b, e.BlobRef)
	telemetryfb.ProfileMetaStart(b)
	addOffset(b, pt, telemetryfb.ProfileMetaAddProfileType)
	telemetryfb.ProfileMetaAddTimestampMs(b, e.TimestampMs)
	addOffset(b, reason, telemetryfb.ProfileMetaAddReason)
	telemetryfb.ProfileMetaAddSizeBytes(b, e.SizeBytes)
	addOffset(b, enc, telemetryfb.ProfileMetaAddContentEncoding)
	addOffset(b, ref, telemetryfb.ProfileMetaAddBlobRef)
	if fields != 0 {
		telemetryfb.ProfileMetaAddFields(b, fields)
	}
	if blob != 0 {
		telemetryfb.ProfileMetaAddBlob(b, blob)
	}
	return telemetryfb.ProfileMetaEnd(b)
}

// buildWorker encodes one WorkerEvent into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*WorkerEvent) which is the worker event to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildWorker(b *flatbuffers.Builder, e *WorkerEvent) flatbuffers.UOffsetT {
	attrs := buildKVVector(b, e.Attrs)
	eid := str(b, e.EventID)
	rid := str(b, e.RunID)
	pid := str(b, e.ParentID)
	cat := str(b, e.Category)
	queue := str(b, e.Queue)
	worker := str(b, e.Worker)
	status := str(b, e.Status)
	errStr := str(b, e.Error)
	telemetryfb.WorkerEventStart(b)
	addOffset(b, eid, telemetryfb.WorkerEventAddEventId)
	addOffset(b, rid, telemetryfb.WorkerEventAddRunId)
	addOffset(b, pid, telemetryfb.WorkerEventAddParentId)
	addOffset(b, cat, telemetryfb.WorkerEventAddCategory)
	addOffset(b, queue, telemetryfb.WorkerEventAddQueue)
	addOffset(b, worker, telemetryfb.WorkerEventAddWorker)
	addOffset(b, status, telemetryfb.WorkerEventAddStatus)
	telemetryfb.WorkerEventAddAttempt(b, e.Attempt)
	telemetryfb.WorkerEventAddTsMs(b, e.TsMs)
	telemetryfb.WorkerEventAddDurationMs(b, e.DurationMs)
	addOffset(b, errStr, telemetryfb.WorkerEventAddError)
	if attrs != 0 {
		telemetryfb.WorkerEventAddAttrs(b, attrs)
	}
	return telemetryfb.WorkerEventEnd(b)
}

// buildQueryStat encodes one QueryStat into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*QueryStat) which is the query observation to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildQueryStat(b *flatbuffers.Builder, e *QueryStat) flatbuffers.UOffsetT {
	attrs := buildKVVector(b, e.Attrs)
	conn := str(b, e.Connection)
	stmt := str(b, e.Statement)
	op := str(b, e.Operation)
	status := str(b, e.Status)
	errStr := str(b, e.Error)
	telemetryfb.QueryStatStart(b)
	addOffset(b, conn, telemetryfb.QueryStatAddConnection)
	addOffset(b, stmt, telemetryfb.QueryStatAddStatement)
	addOffset(b, op, telemetryfb.QueryStatAddOperation)
	addOffset(b, status, telemetryfb.QueryStatAddStatus)
	addOffset(b, errStr, telemetryfb.QueryStatAddError)
	telemetryfb.QueryStatAddTsMs(b, e.TsMs)
	telemetryfb.QueryStatAddDurationMs(b, e.DurationMs)
	telemetryfb.QueryStatAddRows(b, e.Rows)
	telemetryfb.QueryStatAddCalls(b, e.Calls)
	if attrs != 0 {
		telemetryfb.QueryStatAddAttrs(b, attrs)
	}
	return telemetryfb.QueryStatEnd(b)
}

// buildEmail encodes one EmailEvent into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes e (*EmailEvent) which is the email event to encode.
//
// Returns flatbuffers.UOffsetT which is the table offset.
func buildEmail(b *flatbuffers.Builder, e *EmailEvent) flatbuffers.UOffsetT {
	attrs := buildKVVector(b, e.Attrs)
	mid := str(b, e.MessageID)
	provider := str(b, e.Provider)
	tmpl := str(b, e.Template)
	rcpt := str(b, e.Recipient)
	subj := str(b, e.Subject)
	event := str(b, e.Event)
	status := str(b, e.Status)
	errStr := str(b, e.Error)
	telemetryfb.EmailEventStart(b)
	addOffset(b, mid, telemetryfb.EmailEventAddMessageId)
	addOffset(b, provider, telemetryfb.EmailEventAddProvider)
	addOffset(b, tmpl, telemetryfb.EmailEventAddTemplate)
	addOffset(b, rcpt, telemetryfb.EmailEventAddRecipient)
	addOffset(b, subj, telemetryfb.EmailEventAddSubject)
	addOffset(b, event, telemetryfb.EmailEventAddEvent)
	addOffset(b, status, telemetryfb.EmailEventAddStatus)
	addOffset(b, errStr, telemetryfb.EmailEventAddError)
	telemetryfb.EmailEventAddTsMs(b, e.TsMs)
	if attrs != 0 {
		telemetryfb.EmailEventAddAttrs(b, attrs)
	}
	return telemetryfb.EmailEventEnd(b)
}

// readKVs decodes a FlatBuffers vector of KV tables into a slice.
//
// Takes length (int) which is the vector length.
// Takes at (func(obj *telemetryfb.KV, j int) bool) which is the generated indexed
// accessor.
//
// Returns []KV which holds the decoded pairs, or nil when length is zero.
func readKVs(length int, at func(obj *telemetryfb.KV, j int) bool) []KV {
	if length == 0 {
		return nil
	}
	out := make([]KV, 0, min(length, vectorPreallocCap))
	var kv telemetryfb.KV
	for i := range length {
		if at(&kv, i) {
			out = append(out, KV{Key: string(kv.Key()), Value: string(kv.Value())})
		}
	}
	return out
}

// readAnalytics decodes a generated AnalyticsEvent table into the plain-Go struct.
//
// Takes e (*telemetryfb.AnalyticsEvent) which is the generated accessor to read.
//
// Returns AnalyticsEvent which is the decoded event.
func readAnalytics(e *telemetryfb.AnalyticsEvent) AnalyticsEvent {
	return AnalyticsEvent{
		Kind:            string(e.Kind()),
		TimestampMs:     e.TimestampMs(),
		Hostname:        string(e.Hostname()),
		URL:             string(e.Url()),
		Path:            string(e.Path()),
		MatchedPattern:  string(e.MatchedPattern()),
		Method:          string(e.Method()),
		StatusCode:      e.StatusCode(),
		DurationMs:      e.DurationMs(),
		Referrer:        string(e.Referrer()),
		UserAgent:       string(e.UserAgent()),
		ClientIP:        string(e.ClientIp()),
		Locale:          string(e.Locale()),
		UserID:          string(e.UserId()),
		ActionName:      string(e.ActionName()),
		EventName:       string(e.EventName()),
		RevenueAmount:   string(e.RevenueAmount()),
		RevenueCurrency: string(e.RevenueCurrency()),
		Properties:      readKVs(e.PropertiesLength(), e.Properties),
	}
}

// readWatchdog decodes a generated WatchdogEvent table into the plain-Go struct.
//
// Takes e (*telemetryfb.WatchdogEvent) which is the generated accessor to read.
//
// Returns WatchdogEvent which is the decoded event.
func readWatchdog(e *telemetryfb.WatchdogEvent) WatchdogEvent {
	return WatchdogEvent{
		EventType:   string(e.EventType()),
		Priority:    e.Priority(),
		Message:     string(e.Message()),
		TimestampMs: e.TimestampMs(),
		Fields:      readKVs(e.FieldsLength(), e.Fields),
	}
}

// readLog decodes a generated LogLine table into the plain-Go struct.
//
// Takes e (*telemetryfb.LogLine) which is the generated accessor to read.
//
// Returns LogLine which is the decoded log line.
func readLog(e *telemetryfb.LogLine) LogLine {
	return LogLine{
		TimestampMs: e.TimestampMs(),
		Level:       string(e.Level()),
		Logger:      string(e.Logger()),
		Message:     string(e.Message()),
		TraceID:     string(e.TraceId()),
		SpanID:      string(e.SpanId()),
		Fields:      readKVs(e.FieldsLength(), e.Fields),
	}
}

// readSpan decodes a generated Span table into the plain-Go struct.
//
// Takes e (*telemetryfb.Span) which is the generated accessor to read.
//
// Returns Span which is the decoded span.
func readSpan(e *telemetryfb.Span) Span {
	return Span{
		TraceID:    string(e.TraceId()),
		SpanID:     string(e.SpanId()),
		ParentID:   string(e.ParentId()),
		Service:    string(e.Service()),
		Operation:  string(e.Operation()),
		Kind:       string(e.Kind()),
		StartMs:    e.StartMs(),
		DurationUs: e.DurationUs(),
		Status:     string(e.Status()),
		Attributes: readKVs(e.AttributesLength(), e.Attributes),
	}
}

// readMetric decodes a generated MetricPoint table into the plain-Go struct.
//
// Takes e (*telemetryfb.MetricPoint) which is the generated accessor to read.
//
// Returns MetricPoint which is the decoded metric point.
func readMetric(e *telemetryfb.MetricPoint) MetricPoint {
	return MetricPoint{
		Name:        string(e.Name()),
		Kind:        string(e.Kind()),
		TimestampMs: e.TimestampMs(),
		Value:       e.Value(),
		Unit:        string(e.Unit()),
		Labels:      readKVs(e.LabelsLength(), e.Labels),
	}
}

// readError decodes a generated ErrorEvent table into the plain-Go struct.
//
// Takes e (*telemetryfb.ErrorEvent) which is the generated accessor to read.
//
// Returns ErrorEvent which is the decoded error event.
func readError(e *telemetryfb.ErrorEvent) ErrorEvent {
	return ErrorEvent{
		Fingerprint:     string(e.Fingerprint()),
		Type:            string(e.Type()),
		Value:           string(e.Value()),
		Culprit:         string(e.Culprit()),
		Level:           string(e.Level()),
		TimestampMs:     e.TimestampMs(),
		Release:         string(e.Release()),
		Environment:     string(e.Environment()),
		UserID:          string(e.UserId()),
		Handled:         e.Handled(),
		StackJSON:       string(e.StackJson()),
		BreadcrumbsJSON: string(e.BreadcrumbsJson()),
		Context:         readKVs(e.ContextLength(), e.Context),
	}
}

// readProfile decodes a generated ProfileMeta table into the plain-Go struct, copying any
// inline blob bytes out of the buffer.
//
// Takes e (*telemetryfb.ProfileMeta) which is the generated accessor to read.
//
// Returns ProfileMeta which is the decoded profile.
func readProfile(e *telemetryfb.ProfileMeta) ProfileMeta {
	pm := ProfileMeta{
		ProfileType:     string(e.ProfileType()),
		TimestampMs:     e.TimestampMs(),
		Reason:          string(e.Reason()),
		SizeBytes:       e.SizeBytes(),
		ContentEncoding: string(e.ContentEncoding()),
		BlobRef:         string(e.BlobRef()),
		Fields:          readKVs(e.FieldsLength(), e.Fields),
	}
	if n := e.BlobLength(); n > 0 {
		src := e.BlobBytes()
		pm.Blob = make([]byte, len(src))
		copy(pm.Blob, src)
	}
	return pm
}

// readWorker decodes a generated WorkerEvent table into the plain-Go struct.
//
// Takes e (*telemetryfb.WorkerEvent) which is the generated accessor to read.
//
// Returns WorkerEvent which is the decoded worker event.
func readWorker(e *telemetryfb.WorkerEvent) WorkerEvent {
	return WorkerEvent{
		EventID:    string(e.EventId()),
		RunID:      string(e.RunId()),
		ParentID:   string(e.ParentId()),
		Category:   string(e.Category()),
		Queue:      string(e.Queue()),
		Worker:     string(e.Worker()),
		Status:     string(e.Status()),
		Attempt:    e.Attempt(),
		TsMs:       e.TsMs(),
		DurationMs: e.DurationMs(),
		Error:      string(e.Error()),
		Attrs:      readKVs(e.AttrsLength(), e.Attrs),
	}
}

// readQueryStat decodes a generated QueryStat table into the plain-Go struct.
//
// Takes e (*telemetryfb.QueryStat) which is the generated accessor to read.
//
// Returns QueryStat which is the decoded query observation.
func readQueryStat(e *telemetryfb.QueryStat) QueryStat {
	return QueryStat{
		Connection: string(e.Connection()),
		Statement:  string(e.Statement()),
		Operation:  string(e.Operation()),
		Status:     string(e.Status()),
		Error:      string(e.Error()),
		TsMs:       e.TsMs(),
		DurationMs: e.DurationMs(),
		Rows:       e.Rows(),
		Calls:      e.Calls(),
		Attrs:      readKVs(e.AttrsLength(), e.Attrs),
	}
}

// readEmail decodes a generated EmailEvent table into the plain-Go struct.
//
// Takes e (*telemetryfb.EmailEvent) which is the generated accessor to read.
//
// Returns EmailEvent which is the decoded email event.
func readEmail(e *telemetryfb.EmailEvent) EmailEvent {
	return EmailEvent{
		MessageID: string(e.MessageId()),
		Provider:  string(e.Provider()),
		Template:  string(e.Template()),
		Recipient: string(e.Recipient()),
		Subject:   string(e.Subject()),
		Event:     string(e.Event()),
		Status:    string(e.Status()),
		Error:     string(e.Error()),
		TsMs:      e.TsMs(),
		Attrs:     readKVs(e.AttrsLength(), e.Attrs),
	}
}

// TruncatedStrings reports how many telemetry strings have been shortened to fit
// maxStringLen since the process started.
//
// Returns uint64 which is the cumulative truncation count.
func TruncatedStrings() uint64 {
	return strTruncCount.Load() + identityTruncCount.Load()
}
