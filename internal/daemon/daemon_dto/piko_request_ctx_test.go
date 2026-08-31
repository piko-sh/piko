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

package daemon_dto

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/maths"
)

func TestReleasePikoRequestCtx_ClearsRequestScopedFields(t *testing.T) {
	pctx := AcquirePikoRequestCtx()
	pctx.AnalyticsActionName = "docsearch.ask"
	pctx.AnalyticsEventName = "checkout.completed"
	pctx.UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"

	ReleasePikoRequestCtx(pctx)

	assert.Empty(t, pctx.AnalyticsActionName)
	assert.Empty(t, pctx.AnalyticsEventName)
	assert.Empty(t, pctx.UserAgent)
}

func TestAcquirePikoRequestCtx_StartsFromZero(t *testing.T) {
	first := AcquirePikoRequestCtx()
	first.AnalyticsActionName = "docsearch.ask"
	first.UserAgent = "TestAgent/1.0"
	ReleasePikoRequestCtx(first)

	second := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(second)

	require.NotNil(t, second)
	assert.Empty(t, second.AnalyticsActionName)
	assert.Empty(t, second.UserAgent)
}

func TestPikoRequestCtx_ResetClearsEveryField(t *testing.T) {
	pctx := &PikoRequestCtx{}
	fillExportedFields(t, pctx)
	pctx.userAgentClassified = true

	pctx.reset()

	value := reflect.ValueOf(pctx).Elem()
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if !field.IsExported() {
			continue
		}
		assert.True(t, value.Field(index).IsZero(),
			"reset left %s populated, so the next request would inherit it", field.Name)
	}
	assert.False(t, pctx.userAgentClassified,
		"a stale classification would describe the previous request's client")
}

func fillExportedFields(t *testing.T, pctx *PikoRequestCtx) {
	t.Helper()

	value := reflect.ValueOf(pctx).Elem()
	for index := range value.NumField() {
		field := value.Type().Field(index)
		if !field.IsExported() {
			continue
		}

		target := value.Field(index)
		switch field.Type.Kind() {
		case reflect.String:
			target.SetString("set")
		case reflect.Bool:
			target.SetBool(true)
		case reflect.Int:
			target.SetInt(1)
		case reflect.Uint64:
			target.SetUint(1)
		case reflect.Map:
			target.Set(reflect.ValueOf(map[string]string{"key": "value"}))
		case reflect.Interface:

			if field.Type.NumMethod() == 0 {
				target.Set(reflect.ValueOf(any("set")))

				continue
			}
			recorder := reflect.ValueOf(httptest.NewRecorder())
			require.Truef(t, recorder.Type().Implements(field.Type),
				"field %s is an interface this check cannot populate", field.Name)
			target.Set(recorder)
		case reflect.Pointer:
			target.Set(reflect.New(field.Type.Elem()))
		default:
			require.Failf(t, "unhandled field kind",
				"field %s has kind %s, which this check cannot populate", field.Name, field.Type.Kind())
		}
	}
}

func TestPikoRequestCtx_AnalyticsWritesAreConcurrencySafe(t *testing.T) {
	pctx := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(pctx)
	pctx.UserAgent = "Mozilla/5.0 (X11; Linux x86_64) Chrome/131.0.0.0"

	const writers = 8
	started := make(chan struct{})
	done := make(chan struct{}, writers)

	for writer := range writers {
		go func() {
			defer func() { done <- struct{}{} }()
			<-started
			pctx.SetAnalyticsProperty("key", "value", 64)
			pctx.SetAnalyticsAction("action")
			pctx.SetAnalyticsEvent("event")
			pctx.SetAnalyticsRevenue(new(maths.NewMoneyFromString("1.00", "GBP")))
			_ = pctx.UserAgentClass()
			_ = writer
		}()
	}

	close(started)
	for range writers {
		<-done
	}

	assert.Equal(t, "value", pctx.AnalyticsProperties["key"])
	assert.Equal(t, "Chrome", pctx.UserAgentClass().Browser)
}

func TestPikoRequestCtx_SetAnalyticsPropertyBoundsDistinctKeys(t *testing.T) {
	pctx := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(pctx)

	require.True(t, pctx.SetAnalyticsProperty("first", "1", 2))
	require.True(t, pctx.SetAnalyticsProperty("second", "2", 2))

	assert.False(t, pctx.SetAnalyticsProperty("third", "3", 2),
		"a new key past the limit is refused")
	assert.True(t, pctx.SetAnalyticsProperty("first", "updated", 2),
		"an existing key stays writable once the limit is reached")
	assert.Equal(t, "updated", pctx.AnalyticsProperties["first"])
}

func TestPikoRequestCtx_UserAgentClassIsDerivedOnce(t *testing.T) {
	pctx := AcquirePikoRequestCtx()
	defer ReleasePikoRequestCtx(pctx)
	pctx.UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Version/17.0 Safari/605.1"

	first := pctx.UserAgentClass()

	pctx.UserAgent = "curl/8.0"

	assert.Equal(t, first, pctx.UserAgentClass())
}
