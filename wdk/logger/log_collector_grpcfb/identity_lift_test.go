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

package log_collector_grpcfb

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

func liftedFrom(h *Handler, attrs ...slog.Attr) telemetry_grpcfb.ErrorEvent {
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	r.AddAttrs(attrs...)
	line := telemetry_grpcfb.LogLine{Message: r.Message}
	fields, extracted := h.collectFields(&r, &line)
	return h.toError(&r, line, fields, extracted, "")
}

func TestErrorEventCarriesTheDetectedDeploymentIdentity(t *testing.T) {

	ev := liftedFrom(New(nil),
		slog.String("service.version", "v1.4.0"),
		slog.String("deployment.environment.name", "production"),
	)

	assert.Equal(t, "v1.4.0", ev.Release)
	assert.Equal(t, "production", ev.Environment)
}

func TestErrorEventIdentitySurvivesWithGroup(t *testing.T) {

	handler, ok := New(nil).WithGroup("app").(*Handler)
	require.True(t, ok, "WithGroup returns a *Handler")
	ev := liftedFrom(handler,
		slog.String("service.version", "v1.4.0"),
		slog.String("deployment.environment.name", "production"),
	)

	assert.Equal(t, "v1.4.0", ev.Release)
	assert.Equal(t, "production", ev.Environment)
}

func TestErrorEventAcceptsThePlainAliases(t *testing.T) {
	ev := liftedFrom(New(nil), slog.String("release", "v2"), slog.String("env", "staging"))

	assert.Equal(t, "v2", ev.Release)
	assert.Equal(t, "staging", ev.Environment)
}

func TestExplicitReleaseAndEnvironmentWin(t *testing.T) {
	handler := New(nil).WithRelease("v9.9.9").WithEnvironment("canary")
	ev := liftedFrom(handler,
		slog.String("service.version", "v1.4.0"),
		slog.String("deployment.environment.name", "production"),
	)

	assert.Equal(t, "v9.9.9", ev.Release, "an operator stating the release knows more than the process does")
	assert.Equal(t, "canary", ev.Environment)
}

func TestUserIDIsNotLiftedByDefault(t *testing.T) {

	ev := liftedFrom(New(nil), slog.String("user_id", "u-123"))

	assert.Empty(t, ev.UserID)
}

func TestUserIDIsAbsentFromContextWhenNotOptedIn(t *testing.T) {
	for _, key := range []string{"user_id", "userID", "userId", "user.id"} {
		t.Run(key, func(t *testing.T) {
			ev := liftedFrom(New(nil), slog.String(key, "u-123"), slog.String("order", "o-9"))

			assert.Empty(t, ev.UserID)
			for _, field := range ev.Context {
				assert.NotEqual(t, "u-123", field.Value, "no context entry carries the user id")
				assert.NotEqual(t, key, field.Key, "the user id key is not emitted at all")
			}
			assert.Contains(t, contextKeys(ev), "order",
				"withholding the user id does not drop the record's other fields")
		})
	}
}

func TestUserIDReachesContextWhenOptedIn(t *testing.T) {
	ev := liftedFrom(New(nil).WithUserID(true), slog.String("user_id", "u-123"))

	assert.Equal(t, "u-123", ev.UserID)
	assert.Contains(t, contextKeys(ev), "user_id")
}

func contextKeys(ev telemetry_grpcfb.ErrorEvent) []string {
	keys := make([]string, 0, len(ev.Context))
	for _, field := range ev.Context {
		keys = append(keys, field.Key)
	}
	return keys
}

func TestUserIDIsLiftedWhenOptedIn(t *testing.T) {
	for _, key := range []string{"user_id", "userID", "userId", "user.id"} {
		t.Run(key, func(t *testing.T) {
			ev := liftedFrom(New(nil).WithUserID(true), slog.String(key, "u-123"))
			assert.Equal(t, "u-123", ev.UserID)
		})
	}
}

func TestUnsetIdentityStaysEmpty(t *testing.T) {
	ev := liftedFrom(New(nil))

	assert.Empty(t, ev.Release, "an unreported release is absent, not an empty-valued claim")
	assert.Empty(t, ev.Environment)
	assert.Empty(t, ev.UserID)
}

type redactedAccount struct {
	token string
}

func (redactedAccount) LogValue() slog.Value {
	return slog.StringValue("[REDACTED]")
}

func contextValues(t *testing.T, ev telemetry_grpcfb.ErrorEvent) []string {
	t.Helper()

	values := make([]string, 0, len(ev.Context))
	for _, kv := range ev.Context {
		values = append(values, kv.Value)
	}

	return values
}

func TestUserIDIsWithheldWhenNestedInAGroup(t *testing.T) {
	ev := liftedFrom(New(nil), slog.Group("user",
		slog.String("id", "u-123"),
		slog.String("plan", "pro"),
	))

	assert.Empty(t, ev.UserID)
	assert.NotContains(t, contextValues(t, ev), "u-123",
		"a grouped user id must not reach the wire by another route")
	assert.Contains(t, contextValues(t, ev), "pro",
		"the rest of the group is ordinary context and still travels")
}

func TestUserIDIsLiftedFromAGroupWhenEnabled(t *testing.T) {
	ev := liftedFrom(New(nil).WithUserID(true), slog.Group("user", slog.String("id", "u-123")))

	assert.Equal(t, "u-123", ev.UserID)
}

func TestLogValuerIsResolvedBeforeAnythingIsRecorded(t *testing.T) {
	ev := liftedFrom(New(nil), slog.Any("account", redactedAccount{token: "sk-live-abc123"}))

	assert.Contains(t, contextValues(t, ev), "[REDACTED]")
	assert.NotContains(t, contextValues(t, ev), "sk-live-abc123",
		"the raw value must never reach the wire")
}

func TestClientIPIsWithheldByDefault(t *testing.T) {
	ev := liftedFrom(New(nil), slog.String("client_ip", "203.0.113.7"))

	assert.NotContains(t, contextValues(t, ev), "203.0.113.7")
}

func TestClientIPTravelsWhenEnabled(t *testing.T) {
	ev := liftedFrom(New(nil).WithClientIP(true), slog.String("client_ip", "203.0.113.7"))

	assert.Contains(t, contextValues(t, ev), "203.0.113.7")
}
