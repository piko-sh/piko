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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullBatch() *Batch {
	return &Batch{
		SiteID:   "site-42",
		APIKey:   "delivery-key",
		Source:   "pod-7",
		SentAtMs: 1_700_000_000_000,
		Seq:      9,
		Analytics: []AnalyticsEvent{{
			Kind:            "pageview",
			TimestampMs:     1_700_000_000_111,
			Hostname:        "example.com",
			URL:             "https://example.com/blog/post?utm=x",
			Path:            "/blog/post",
			MatchedPattern:  "/blog/{slug}",
			Method:          "GET",
			StatusCode:      200,
			DurationMs:      37,
			Referrer:        "https://news.example",
			UserAgent:       "Mozilla/5.0",
			ClientIP:        "203.0.113.9",
			Locale:          "en-GB",
			UserID:          "u-1",
			ActionName:      "view",
			EventName:       "page_view",
			RevenueAmount:   "29.99",
			RevenueCurrency: "GBP",
			Properties:      []KV{{Key: "plan", Value: "pro"}, {Key: "ref", Value: "twitter"}},
		}},
		Watchdog: []WatchdogEvent{{
			EventType:   "heap_threshold_exceeded",
			Priority:    3,
			Message:     "heap above threshold",
			TimestampMs: 1_700_000_000_222,
			Fields:      []KV{{Key: "heap_bytes", Value: "1048576"}},
		}},
		Logs: []LogLine{{
			TimestampMs: 1_700_000_000_333,
			Level:       "error",
			Logger:      "http",
			Message:     "request failed",
			TraceID:     "trace-1",
			SpanID:      "span-1",
			Fields:      []KV{{Key: "status", Value: "500"}},
		}},
		Spans: []Span{{
			TraceID:    "trace-1",
			SpanID:     "span-2",
			ParentID:   "span-1",
			Service:    "api",
			Operation:  "GET /blog",
			Kind:       "server",
			StartMs:    1_700_000_000_444,
			DurationUs: 12_345,
			Status:     "ok",
			Attributes: []KV{{Key: "http.method", Value: "GET"}},
		}},
		Metrics: []MetricPoint{{
			Name:        "request.duration",
			Kind:        "histogram",
			TimestampMs: 1_700_000_000_555,
			Value:       42.5,
			Unit:        "ms",
			Labels:      []KV{{Key: "route", Value: "/blog"}},
		}},
		Errors: []ErrorEvent{{
			Fingerprint:     "fp-abc",
			Type:            "panic",
			Value:           "nil pointer dereference",
			Culprit:         "handlers.Blog",
			Level:           "fatal",
			TimestampMs:     1_700_000_000_666,
			Release:         "v1.2.3",
			Environment:     "production",
			UserID:          "u-1",
			Handled:         true,
			StackJSON:       `[{"fn":"main"}]`,
			BreadcrumbsJSON: `[{"msg":"clicked"}]`,
			Context:         []KV{{Key: "feature", Value: "blog"}},
		}},
		Profiles: []ProfileMeta{{
			ProfileType:     "heap",
			TimestampMs:     1_700_000_000_777,
			Reason:          "threshold",
			SizeBytes:       6,
			ContentEncoding: "gzip",
			BlobRef:         "s3://bucket/profiles/heap-1",
			Fields:          []KV{{Key: "goroutines", Value: "120"}},
			Blob:            []byte{0x1f, 0x8b, 0x08, 0x00, 0x01, 0x02},
		}},
		Workers: []WorkerEvent{{
			EventID:    "we-1",
			RunID:      "run-9",
			ParentID:   "chain-3",
			Category:   "attempt",
			Queue:      "emails",
			Worker:     "worker-2",
			Status:     "failed",
			Attempt:    2,
			TsMs:       1_700_000_000_888,
			DurationMs: 1_234,
			Error:      "smtp: connection refused",
			Attrs:      []KV{{Key: "job", Value: "send_welcome"}, {Key: "payload_bytes", Value: "512"}},
		}},
		QueryStats: []QueryStat{{
			Connection: "primary",
			Statement:  "SELECT * FROM users WHERE id = ?",
			Operation:  "select",
			Status:     "error",
			Error:      "context deadline exceeded",
			TsMs:       1_700_000_000_999,
			DurationMs: 87,
			Rows:       3,
			Calls:      5,
			Attrs:      []KV{{Key: "table", Value: "users"}, {Key: "index", Value: "users_pkey"}},
		}},
		Emails: []EmailEvent{{
			MessageID: "msg-77",
			Provider:  "ses",
			Template:  "welcome",
			Recipient: "user@example.com",
			Subject:   "Welcome aboard",
			Event:     "bounced",
			Status:    "error",
			Error:     "550 mailbox unavailable",
			TsMs:      1_700_000_001_000,
			Attrs:     []KV{{Key: "bounce_type", Value: "hard"}, {Key: "smtp_code", Value: "550"}},
		}},
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	orig := fullBatch()
	data, err := orig.Marshal()
	require.NoError(t, err)
	var got Batch
	require.NoError(t, got.Unmarshal(data))
	assert.Equal(t, orig, &got)
}

func TestIngestAckRoundTrip(t *testing.T) {
	orig := &IngestAck{OK: true, Frames: 12, Events: 345, Message: "accepted"}
	data, err := orig.Marshal()
	require.NoError(t, err)
	var got IngestAck
	require.NoError(t, got.Unmarshal(data))
	assert.Equal(t, orig, &got)
}

func TestVerifyRejectsObviouslyMalformed(t *testing.T) {
	cases := map[string][]byte{
		"empty":    {},
		"tooShort": {1, 2, 3},
		"oobRoot":  {0xff, 0xff, 0xff, 0xff},
		"zeroRoot": {0x00, 0x00, 0x00, 0x00},
		"garbage":  {0x10, 0x00, 0x00, 0x00, 0xaa, 0xbb, 0xcc, 0xdd},
	}
	for name, buf := range cases {
		t.Run(name, func(t *testing.T) {
			var b Batch
			assert.Error(t, b.Unmarshal(buf))
		})
	}
}

func TestVerifyRejectsOversized(t *testing.T) {
	big := make([]byte, MaxMessageSize+1)
	var b Batch
	assert.Error(t, b.Unmarshal(big))
}

func TestUnmarshalNeverPanics(t *testing.T) {
	data, err := fullBatch().Marshal()
	require.NoError(t, err)
	patterns := []byte{0x01, 0x7f, 0x80, 0xff}
	for i := range data {
		for _, xor := range patterns {
			mut := make([]byte, len(data))
			copy(mut, data)
			mut[i] ^= xor
			assert.NotPanics(t, func() {
				var b Batch
				_ = b.Unmarshal(mut)
			}, "mutation at byte %d xor %#x", i, xor)
		}
	}
}

func TestVerifierTableDepthBoundary(t *testing.T) {
	t.Run("atMaxDepthRejected", func(t *testing.T) {
		v := &verifier{}
		assert.ErrorIs(t, v.table(0, nil, maxDepth), errTooDeep)
	})
	t.Run("belowMaxDepthPassesDepthGuard", func(t *testing.T) {
		v := &verifier{}
		assert.ErrorIs(t, v.table(0, nil, maxDepth-1), errOOB)
	})
}

func TestVerifyByteVectorOverrunIsOOB(t *testing.T) {
	v := &verifier{buf: lenPrefixed(8, 0)}
	assert.ErrorIs(t, v.verifyByteVector(0), errOOB)
}

func TestVerifyStringOverrunIsOOB(t *testing.T) {
	v := &verifier{buf: lenPrefixed(8, 0)}
	assert.ErrorIs(t, v.verifyString(0), errOOB)
}

func TestCodecRecoversFromGarbage(t *testing.T) {
	c := Codec{}
	garbage := []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01, 0x02, 0x03}
	var b Batch
	assert.Error(t, c.Unmarshal(garbage, &b))
}
