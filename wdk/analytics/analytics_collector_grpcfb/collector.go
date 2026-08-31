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

package analytics_collector_grpcfb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
	"piko.sh/piko/wdk/useragent"
)

const (
	// collectorName identifies the collector in piko's analytics fan-out diagnostics.
	collectorName = "grpcfb"

	// maxProperties bounds how many custom event properties travel per event, so attacker or
	// app influenced cardinality cannot exhaust the frame budget.
	maxProperties = 128

	// maxPropertyValueLen bounds each custom property value, so an oversized value cannot
	// exhaust the frame budget.
	maxPropertyValueLen = 1024

	// clientClassPrefix namespaces the properties derived from the User-Agent.
	clientClassPrefix = "client."

	// clientBrowserKey carries the browser family, e.g. "Chrome".
	clientBrowserKey = clientClassPrefix + "browser"

	// clientBrowserMajorKey carries the browser major version as a decimal string.
	clientBrowserMajorKey = clientClassPrefix + "browser_major"

	// clientOSKey carries the operating-system family, e.g. "macOS".
	clientOSKey = clientClassPrefix + "os"

	// clientDeviceKey carries the form factor: desktop, mobile, tablet or bot.
	clientDeviceKey = clientClassPrefix + "device"

	// propertiesDroppedKey marks an event whose properties hit the count cap; its value is
	// the number that were not sent.
	propertiesDroppedKey = clientClassPrefix + "properties_dropped"

	// propertiesTruncatedKey marks an event carrying at least one property value clipped to
	// maxPropertyValueLen; its value is the number of values shortened.
	propertiesTruncatedKey = clientClassPrefix + "properties_truncated"

	// propertiesReservedKey reports how many caller properties were dropped for naming the
	// framework's reserved prefix.
	propertiesReservedKey = clientClassPrefix + "properties_reserved"

	// maxPropertyMarkers is how many marker entries cappedProperties may append, and so how
	// many slots it holds back from the cap.
	maxPropertyMarkers = 3

	// clientBotKey marks a self-identified crawler.
	clientBotKey = clientClassPrefix + "bot"

	// maxClientClassProps is the largest number of properties userAgentClass can emit. It
	// sizes the slice; the caller's budget is reduced by the number actually derived.
	maxClientClassProps = 5
)

// privacyPolicy controls how client-identifying fields are anonymised before streaming
// off-box. The default hashes the client IP and user agent, strips URL query strings, and
// omits the raw user id, mirroring the framework's GA4 collector.
type privacyPolicy struct {
	// hashClientIP reports whether the client IP is SHA-256 hashed before streaming.
	hashClientIP bool

	// hashUserAgent reports whether the user agent is SHA-256 hashed before streaming.
	hashUserAgent bool

	// stripURLQuery reports whether URL and referrer query strings are stripped off.
	stripURLQuery bool

	// sendUserID reports whether the raw user id is streamed rather than omitted.
	sendUserID bool

	// classifyUserAgent reports whether coarse client families are derived from the
	// User-Agent and shipped as reserved properties.
	classifyUserAgent bool
}

// Collector implements analytics.Collector by translating analytics events into telemetry
// frames and handing them to a telemetry_grpcfb.Client.
type Collector struct {
	// client is the telemetry transport that streams frames off-box.
	client *telemetry_grpcfb.Client

	// privacy is the anonymisation policy applied to each event before streaming.
	privacy privacyPolicy

	// ownsClient reports whether Close should drain and close client rather than flush.
	ownsClient bool
}

// Option configures a Collector's privacy policy.
type Option func(*Collector)

var (
	_ analytics.Collector = (*Collector)(nil)
)

// WithRawClientIP streams the real client IP instead of a SHA-256 hash.
//
// Returns Option which configures the collector to stream the raw client IP.
func WithRawClientIP() Option { return func(c *Collector) { c.privacy.hashClientIP = false } }

// WithRawUserAgent streams the real user agent instead of a SHA-256 hash.
//
// Returns Option which configures the collector to stream the raw user agent.
func WithRawUserAgent() Option { return func(c *Collector) { c.privacy.hashUserAgent = false } }

// WithURLQuery keeps the URL and referrer query string instead of stripping it. Query
// strings routinely carry auth or reset tokens and search PII.
//
// Returns Option which configures the collector to keep query strings.
func WithURLQuery() Option { return func(c *Collector) { c.privacy.stripURLQuery = false } }

// WithUserID streams the raw user id, which is omitted by default.
//
// Returns Option which configures the collector to stream the raw user id.
func WithUserID() Option { return func(c *Collector) { c.privacy.sendUserID = true } }

// WithoutUserAgentClass stops the collector deriving coarse client families from the
// User-Agent, so no clientClassPrefix property is emitted.
//
// Returns Option which configures the collector to emit no derived client classes.
func WithoutUserAgentClass() Option {
	return func(c *Collector) { c.privacy.classifyUserAgent = false }
}

// defaultPrivacy returns the default policy that hashes PII and strips query strings.
//
// Returns privacyPolicy which is the privacy-preserving default policy.
func defaultPrivacy() privacyPolicy {
	return privacyPolicy{
		hashClientIP:      true,
		hashUserAgent:     true,
		stripURLQuery:     true,
		sendUserID:        false,
		classifyUserAgent: true,
	}
}

// hashPII returns a hex SHA-256 of s, an opaque stable identifier, or "" for "". It lets
// the sink count unique clients without ever receiving the raw IP or user agent.
//
// Takes s (string) which is the raw PII value to hash.
//
// Returns string which is the hex SHA-256 digest, or "" when s is empty.
func hashPII(s string) string {
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// stripQuery drops the query string and fragment from a URL, keeping the path.
//
// Takes u (string) which is the URL to strip.
//
// Returns string which is the URL with any query string and fragment removed.
func stripQuery(u string) string {
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		return u[:i]
	}
	return u
}

// Name reports the collector's identity.
//
// Returns string which is the collector name used in fan-out diagnostics.
func (*Collector) Name() string { return collectorName }

// Start begins streaming events to the sink.
//
// Repeated calls when sharing a client have no extra effect, and the call is a no-op when
// the client is nil (for example the app's Dial failed) so a down sink never crashes the
// host, matching the nil-safety of the other grpcfb collectors.
func (c *Collector) Start(ctx context.Context) {
	if c == nil || c.client == nil {
		return
	}
	c.client.Start(ctx)
}

// Collect translates an analytics event into a telemetry frame and enqueues it. It copies
// everything it needs (the Event must not be retained per the port contract) and never
// blocks.
//
// Takes e (*analytics.Event) which is the event to translate and enqueue.
//
// Returns error which is always nil; enqueue is best-effort and never fails inline.
func (c *Collector) Collect(ctx context.Context, e *analytics.Event) error {
	if c == nil || c.client == nil || e == nil {
		return nil
	}
	url, referrer := e.URL, e.Referrer
	if c.privacy.stripURLQuery {
		url, referrer = stripQuery(url), stripQuery(referrer)
	}
	userAgent := e.UserAgent
	if c.privacy.hashUserAgent {
		userAgent = hashPII(userAgent)
	}
	clientIP := e.ClientIP
	if c.privacy.hashClientIP {
		clientIP = hashPII(clientIP)
	}
	userID := ""
	if c.privacy.sendUserID {
		userID = e.UserID
	}
	ev := telemetry_grpcfb.AnalyticsEvent{
		Kind:           e.Type.String(),
		TimestampMs:    e.Timestamp.UnixMilli(),
		Hostname:       e.Hostname,
		URL:            url,
		Path:           e.Path,
		MatchedPattern: e.MatchedPattern,
		Method:         e.Method,
		StatusCode:     safeconv.IntToInt32(e.StatusCode),
		DurationMs:     e.Duration.Milliseconds(),
		Referrer:       referrer,
		UserAgent:      userAgent,
		ClientIP:       clientIP,
		Locale:         e.Locale,
		UserID:         userID,
		ActionName:     e.ActionName,
		EventName:      e.EventName,
	}
	if e.Revenue != nil {
		if code, err := e.Revenue.CurrencyCode(); err == nil {
			if num, err := e.Revenue.Number(); err == nil {
				ev.RevenueAmount = num
				ev.RevenueCurrency = code
			}
		}
	}

	derived := c.userAgentClass(e.UserAgent)
	ev.Properties = append(cappedProperties(e.Properties, maxProperties-len(derived)), derived...)
	c.client.AddAnalytics(ctx, ev)
	return nil
}

// cappedProperties copies at most limit entries, each value UTF-8 truncated to
// maxPropertyValueLen, so a high-cardinality or oversized property set cannot blow the
// frame budget. Map iteration order is non-deterministic, acceptable for a lossy,
// best-effort overflow cap.
//
// Takes props (map[string]string) which are the custom event properties to cap.
// Takes limit (int) which is the most entries to copy, leaving room for derived ones.
//
// Returns []telemetry_grpcfb.KV which is the capped, truncated property list, or nil.
func cappedProperties(props map[string]string, limit int) []telemetry_grpcfb.KV {
	if len(props) == 0 || limit <= 0 {
		return nil
	}
	out := make([]telemetry_grpcfb.KV, 0, min(len(props), limit))
	dropped, truncated, reserved := 0, 0, 0

	for key, value := range props {
		if strings.HasPrefix(key, clientClassPrefix) {
			reserved++

			continue
		}

		if len(out) >= limit-maxPropertyMarkers {
			dropped++

			continue
		}

		clipped, wasTruncated := telemetry_grpcfb.TruncateUTF8(value, maxPropertyValueLen)
		if wasTruncated {
			truncated++
		}
		out = append(out, telemetry_grpcfb.KV{Key: key, Value: clipped})
	}

	if dropped > 0 {
		out = append(out, telemetry_grpcfb.KV{
			Key: propertiesDroppedKey, Value: strconv.Itoa(dropped),
		})
	}
	if truncated > 0 {
		out = append(out, telemetry_grpcfb.KV{
			Key: propertiesTruncatedKey, Value: strconv.Itoa(truncated),
		})
	}

	if reserved > 0 {
		out = append(out, telemetry_grpcfb.KV{
			Key: propertiesReservedKey, Value: strconv.Itoa(reserved),
		})
	}

	return out
}

// Flush queues the current partial batch.
//
// Returns error which wraps any failure from the underlying client flush.
func (c *Collector) Flush(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Flush(ctx); err != nil {
		return fmt.Errorf("analytics_collector_grpcfb: flush: %w", err)
	}
	return nil
}

// Close drains and closes an owned client; for a shared client it only flushes.
//
// Returns error which wraps any failure from the underlying client close or flush.
func (c *Collector) Close(ctx context.Context) error {
	if c == nil || c.client == nil {
		return nil
	}
	if c.ownsClient {
		if err := c.client.Close(ctx); err != nil {
			return fmt.Errorf("analytics_collector_grpcfb: close: %w", err)
		}
		return nil
	}
	if err := c.client.Flush(ctx); err != nil {
		return fmt.Errorf("analytics_collector_grpcfb: flush on close: %w", err)
	}
	return nil
}

// New wraps an existing, shared telemetry client. The caller owns the client's lifecycle
// (Start and Close are driven by the client owner); the collector's Close only flushes,
// so it can safely share one client with the watchdog adapter.
//
// Takes client (*telemetry_grpcfb.Client) which is the shared telemetry transport.
// Takes opts (...Option) which override the default privacy policy.
//
// Returns *Collector which is the configured collector wrapping the shared client.
func New(client *telemetry_grpcfb.Client, opts ...Option) *Collector {
	c := &Collector{client: client, privacy: defaultPrivacy()}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// Dial creates a collector backed by its own telemetry client and connection. Its Close
// drains and closes that client, and the default privacy policy is applied (use New with
// a pre-built client to customise the policy).
//
// Takes target (string) which is the telemetry sink address to dial.
// Takes config (telemetry_grpcfb.Config) which configures the new telemetry client.
// Takes dialOpts (...grpc.DialOption) which are passed to the underlying gRPC dial.
//
// Returns *Collector which owns the dialled telemetry client.
// Returns error which wraps any failure to dial the sink.
func Dial(target string, config telemetry_grpcfb.Config, dialOpts ...grpc.DialOption) (*Collector, error) {
	client, err := telemetry_grpcfb.Dial(target, config, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("analytics_collector_grpcfb: dial %q: %w", target, err)
	}
	return &Collector{client: client, privacy: defaultPrivacy(), ownsClient: true}, nil
}

// userAgentClass derives the coarse client families from the raw User-Agent header.
//
// Takes rawUserAgent (string) which is the unhashed User-Agent header from the request.
//
// Returns []telemetry_grpcfb.KV which are the derived client properties, or nil.
func (c *Collector) userAgentClass(rawUserAgent string) []telemetry_grpcfb.KV {
	if !c.privacy.classifyUserAgent || rawUserAgent == "" {
		return nil
	}
	class := useragent.Classify(rawUserAgent)
	out := make([]telemetry_grpcfb.KV, 0, maxClientClassProps)

	for _, candidate := range [...]telemetry_grpcfb.KV{
		{Key: clientBrowserKey, Value: class.Browser},
		{Key: clientBrowserMajorKey, Value: class.BrowserMajor},
		{Key: clientOSKey, Value: class.OS},
		{Key: clientDeviceKey, Value: class.Device},
	} {
		if candidate.Value == "" {
			continue
		}
		value, _ := telemetry_grpcfb.TruncateUTF8(candidate.Value, maxPropertyValueLen)
		out = append(out, telemetry_grpcfb.KV{Key: candidate.Key, Value: value})
	}
	if class.Bot {
		out = append(out, telemetry_grpcfb.KV{Key: clientBotKey, Value: "true"})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
