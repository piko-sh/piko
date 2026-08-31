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

package analytics_collector_grpcfb_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/analytics"
	"piko.sh/piko/wdk/analytics/analytics_collector_grpcfb"
	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	chromeUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

func propMap(kvs []telemetry_grpcfb.KV) map[string]string {
	out := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value
	}
	return out
}

func TestUserAgentClassIsDerivedAlongsideTheHash(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: chromeUA,
	})

	assert.Equal(t, hashPII(chromeUA), got.UserAgent, "the raw header must never reach the wire")

	props := propMap(got.Properties)
	assert.Equal(t, "Chrome", props["client.browser"])
	assert.Equal(t, "131", props["client.browser_major"])
	assert.Equal(t, "macOS", props["client.os"])
	assert.Equal(t, "desktop", props["client.device"])
	assert.NotContains(t, props, "client.bot", "a non-bot emits no bot marker")
}

func TestUserAgentClassSurvivesWithTheRawHeaderStreamed(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client, analytics_collector_grpcfb.WithRawUserAgent())

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: chromeUA,
	})

	assert.Equal(t, chromeUA, got.UserAgent)
	assert.Equal(t, "Chrome", propMap(got.Properties)["client.browser"])
}

func TestUserAgentClassMarksBots(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	})

	props := propMap(got.Properties)
	assert.Equal(t, "true", props["client.bot"])
	assert.Equal(t, "bot", props["client.device"])
}

func TestWithoutUserAgentClassEmitsNothingDerived(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client, analytics_collector_grpcfb.WithoutUserAgentClass())

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: chromeUA,
	})

	for _, kv := range got.Properties {
		assert.NotContains(t, kv.Key, "client.")
	}
	assert.Equal(t, hashPII(chromeUA), got.UserAgent, "opting out of classes keeps the hash")
}

func TestUserAgentClassOmitsUndeterminedFields(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: "something-nobody-has-ever-seen/1",
	})

	props := propMap(got.Properties)
	assert.NotContains(t, props, "client.browser", "absence means undetermined, not a blank measurement")
	assert.NotContains(t, props, "client.os")
}

func TestNoUserAgentDerivesNothing(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	got := streamOne(t, h, col, &analytics.Event{
		Type:       analytics.EventPageView,
		Timestamp:  time.Unix(1_700_000_000, 0),
		Properties: map[string]string{"plan": "pro"},
	})

	assert.Equal(t, []telemetry_grpcfb.KV{{Key: "plan", Value: "pro"}}, got.Properties)
}

func TestCallerCannotForgeAReservedProperty(t *testing.T) {
	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	got := streamOne(t, h, col, &analytics.Event{
		Type:      analytics.EventPageView,
		Timestamp: time.Unix(1_700_000_000, 0),
		UserAgent: chromeUA,
		Properties: map[string]string{
			"client.browser": "Netscape",
			"client.made_up": "x",
			"plan":           "pro",
		},
	})

	props := propMap(got.Properties)
	assert.Equal(t, "Chrome", props["client.browser"])
	assert.NotContains(t, props, "client.made_up")
	assert.Equal(t, "pro", props["plan"], "properties outside the namespace are untouched")
}

func TestUserAgentClassStaysWithinThePropertyCap(t *testing.T) {
	const maxProperties = 128

	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	props := make(map[string]string, maxProperties*2)
	for i := range maxProperties * 2 {
		props["k"+strconv.Itoa(i)] = "v"
	}

	got := streamOne(t, h, col, &analytics.Event{
		Type:       analytics.EventCustom,
		Timestamp:  time.Unix(1_700_000_000, 0),
		UserAgent:  chromeUA,
		Properties: props,
	})

	require.LessOrEqual(t, len(got.Properties), maxProperties,
		"the derived classes and markers must fit inside the cap, not push past it")
	assert.Equal(t, "Chrome", propMap(got.Properties)["client.browser"])
}

func TestPropertiesStayWithinTheCapWhenEveryMarkerFires(t *testing.T) {
	const maxProperties = 128

	h := newHarness(t)
	col := analytics_collector_grpcfb.New(h.client)

	props := map[string]string{"client.plan": "pro"}
	for i := range maxProperties * 2 {
		props["k"+strconv.Itoa(i)] = strings.Repeat("v", 2048)
	}

	got := streamOne(t, h, col, &analytics.Event{
		Type:       analytics.EventCustom,
		Timestamp:  time.Unix(1_700_000_000, 0),
		UserAgent:  chromeUA,
		Properties: props,
	})

	require.LessOrEqual(t, len(got.Properties), maxProperties)

	byKey := propMap(got.Properties)
	assert.NotEmpty(t, byKey["client.properties_dropped"])
	assert.NotEmpty(t, byKey["client.properties_truncated"])
	assert.Equal(t, "1", byKey["client.properties_reserved"],
		"a caller property under the reserved prefix is reported, not silently discarded")
}
