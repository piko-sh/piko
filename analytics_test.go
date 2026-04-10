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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/analytics/analytics_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/wdk/maths"
)

type testAuth struct {
	userID        string
	authenticated bool
}

func (a *testAuth) IsAuthenticated() bool { return a.authenticated }
func (a *testAuth) UserID() string        { return a.userID }
func (a *testAuth) Get(_ string) any      { return nil }

func TestSetAnalyticsRevenue(t *testing.T) {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	revenue := maths.NewMoneyFromString("29.99", "GBP")
	SetAnalyticsRevenue(ctx, revenue)

	require.NotNil(t, pctx.AnalyticsRevenue, "AnalyticsRevenue should be set")
	assert.Equal(t, "29.99", pctx.AnalyticsRevenue.MustNumber())

	currencyCode, err := pctx.AnalyticsRevenue.CurrencyCode()
	require.NoError(t, err)
	assert.Equal(t, "GBP", currencyCode)
}

func TestSetAnalyticsRevenue_NilContext(t *testing.T) {

	SetAnalyticsRevenue(context.Background(), maths.NewMoneyFromString("10.00", "GBP"))
}

func TestAddAnalyticsProperty(t *testing.T) {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	AddAnalyticsProperty(ctx, "variant", "B")
	AddAnalyticsProperty(ctx, "category", "blog")

	require.NotNil(t, pctx.AnalyticsProperties)
	assert.Equal(t, "B", pctx.AnalyticsProperties["variant"])
	assert.Equal(t, "blog", pctx.AnalyticsProperties["category"])
}

func TestAddAnalyticsProperty_LazyMapAllocation(t *testing.T) {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)

	assert.Nil(t, pctx.AnalyticsProperties, "map should be nil before first use")

	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)
	AddAnalyticsProperty(ctx, "key", "value")

	assert.NotNil(t, pctx.AnalyticsProperties, "map should be allocated on first use")
}

func TestAddAnalyticsProperty_NilContext(t *testing.T) {
	AddAnalyticsProperty(context.Background(), "key", "value")
}

func TestSetAnalyticsEventName(t *testing.T) {
	pctx := daemon_dto.AcquirePikoRequestCtx()
	defer daemon_dto.ReleasePikoRequestCtx(pctx)
	ctx := daemon_dto.WithPikoRequestCtx(context.Background(), pctx)

	SetAnalyticsEventName(ctx, "signup")

	assert.Equal(t, "signup", pctx.AnalyticsEventName)
}

func TestSetAnalyticsEventName_NilContext(t *testing.T) {
	SetAnalyticsEventName(context.Background(), "should-not-panic")
}

func TestEnrichEventFromRequestCtx_FillsEmptyFields(t *testing.T) {
	pctx := &daemon_dto.PikoRequestCtx{
		ClientIP:       "203.0.113.50",
		Locale:         "de",
		MatchedPattern: "/users/{id}",
		Hostname:       "example.com",
		CachedAuth: &testAuth{
			authenticated: true,
			userID:        "user-42",
		},
	}

	event := &analytics_dto.Event{}
	enrichEventFromRequestCtx(event, pctx)

	assert.Equal(t, "203.0.113.50", event.ClientIP)
	assert.Equal(t, "de", event.Locale)
	assert.Equal(t, "/users/{id}", event.MatchedPattern)
	assert.Equal(t, "example.com", event.Hostname)
	assert.Equal(t, "user-42", event.UserID)
}

func TestEnrichEventFromRequestCtx_DoesNotOverwriteExistingFields(t *testing.T) {
	pctx := &daemon_dto.PikoRequestCtx{
		ClientIP:       "203.0.113.50",
		Locale:         "de",
		MatchedPattern: "/users/{id}",
		Hostname:       "example.com",
		CachedAuth: &testAuth{
			authenticated: true,
			userID:        "pctx-user",
		},
	}

	event := &analytics_dto.Event{
		ClientIP:       "10.0.0.1",
		Locale:         "en",
		MatchedPattern: "/custom",
		Hostname:       "other.com",
		UserID:         "existing-user",
	}
	enrichEventFromRequestCtx(event, pctx)

	assert.Equal(t, "10.0.0.1", event.ClientIP, "should not overwrite ClientIP")
	assert.Equal(t, "en", event.Locale, "should not overwrite Locale")
	assert.Equal(t, "/custom", event.MatchedPattern, "should not overwrite MatchedPattern")
	assert.Equal(t, "other.com", event.Hostname, "should not overwrite Hostname")
	assert.Equal(t, "existing-user", event.UserID, "should not overwrite UserID")
}

func TestEnrichEventFromRequestCtx_UnauthenticatedSkipsUserID(t *testing.T) {
	pctx := &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
		CachedAuth: &testAuth{
			authenticated: false,
			userID:        "should-not-appear",
		},
	}

	event := &analytics_dto.Event{}
	enrichEventFromRequestCtx(event, pctx)

	assert.Empty(t, event.UserID, "unauthenticated auth should not set UserID")
}

func TestEnrichEventFromRequestCtx_NilAuth(t *testing.T) {
	pctx := &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	}

	event := &analytics_dto.Event{}
	enrichEventFromRequestCtx(event, pctx)

	assert.Empty(t, event.UserID)
	assert.Equal(t, "10.0.0.1", event.ClientIP)
}
