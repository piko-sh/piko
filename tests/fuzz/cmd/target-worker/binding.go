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

package main

import (
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"piko.sh/fuzzkit/tape"
	"piko.sh/json"
	core "piko.sh/piko/internal/binder"
	"piko.sh/piko/internal/daemon/daemon_adapters"
	"piko.sh/piko/internal/daemon/daemon_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/binder"
	"piko.sh/safeconv"
)

const (
	// maxExtraContacts bounds the contacts drawn beyond the first.
	maxExtraContacts = 4

	// maxExtraTags bounds the drawn repeated-value limit beyond two.
	maxExtraTags = 4

	// doublePrecisionLimit is the first integer above which doubles skip integers.
	doublePrecisionLimit = 1 << 53

	// precisionOffsetRange is the number of distinct odd offsets above the limit.
	precisionOffsetRange = 2048

	// maxActionCount bounds the drawn action contact counter.
	maxActionCount = 1000
)

// contact supplies a nested action argument.
type contact struct {
	// Name is the bound contact name.
	Name string

	// Count is the bound contact counter.
	Count int16
}

// order supplies the action binder's input fields.
type order struct {
	// Optional is an optional label pointer.
	Optional *string

	// Label is the displayed label.
	Label string `json:"title"`

	// Unused is ignored by the binder through bind:"-".
	Unused string `bind:"-"`

	// Contacts holds nested contact values.
	Contacts []contact

	// Enabled is the action's enabled flag.
	Enabled bool `bind:"active" json:"is_enabled"`
}

// StreamProgress writes the contact's progress event to the action stream.
//
// Takes stream (*daemon_domain.SSEStream) which receives the progress event.
//
// Returns error which reports a failed progress-event write, or nil on success.
func (c *contact) StreamProgress(stream *daemon_domain.SSEStream) error {
	return stream.Send("fixture", c)
}

// label draws a bounded string for binding checks.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns string which is the drawn bounded label.
func label(choices *tape.Tape) string {
	return pick(choices, []string{"alpha", "deux mots", "år", "七", "x+y & z", "don't", "🙂"})
}

// checkBinder checks form and JSON binding against independently drawn values.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
// Takes savedCase (*casefile.Case) which holds the saved sources, choices and expected
// results.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkBinder(ctx context.Context, choices *tape.Tape, savedCase *casefile.Case) error {
	want := order{Label: label(choices), Enabled: choices.Bool(1, 2), Unused: "", Optional: nil, Contacts: nil}
	form := map[string][]string{
		"title":  {want.Label},
		"active": {strconv.FormatBool(want.Enabled)},
		"Unused": {"discard"},
	}
	object := map[string]any{"title": want.Label, "active": want.Enabled, "Unused": "discard"}
	if choices.Bool(1, 2) {
		value := label(choices)
		want.Optional = &value
		form["Optional"] = []string{value}
		object["Optional"] = value
	}
	contactCount := 1 + drawInt(choices, maxExtraContacts)
	contacts := make([]any, 0, contactCount)
	for index := range contactCount {
		entry := contact{Name: label(choices), Count: pick(choices, []int16{math.MinInt16, -1, 0, 1, math.MaxInt16})}
		want.Contacts = append(want.Contacts, entry)
		form[fmt.Sprintf("Contacts[%d].Name", index)] = []string{entry.Name}
		form[fmt.Sprintf("Contacts[%d].Count", index)] = []string{strconv.Itoa(int(entry.Count))}
		contacts = append(contacts, map[string]any{"Name": entry.Name, "Count": entry.Count})
	}
	object["Contacts"] = contacts
	wire, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("encoding binder JSON fixture: %w", err)
	}
	for _, representation := range []string{"form", "map", "json"} {
		var got order
		switch representation {
		case "form":
			err = binder.Bind(ctx, &got, form)
		case "map":
			err = binder.BindMap(ctx, &got, object)
		case "json":
			err = binder.BindJSON(ctx, &got, wire)
		default:
			return fmt.Errorf("unknown representation %q", representation)
		}
		if err != nil {
			return property(representation+"-acceptance", "%s bind: %w", representation, err)
		}
		if !reflect.DeepEqual(got, want) {
			return property(representation+"-value", "%s bind got %+v, want %+v", representation, got, want)
		}
	}
	if err := checkFormSlice(ctx, choices); err != nil {
		return err
	}
	if err := checkJSONNumber(ctx, choices); err != nil {
		return err
	}
	if err := checkInvoice(ctx, choices); err != nil {
		return err
	}
	if err := checkBinderLimits(ctx, choices); err != nil {
		return err
	}
	return checkReservation(ctx, savedCase.Data)
}

// checkFormSlice checks repeated form-field binding.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFormSlice(ctx context.Context, choices *tape.Tape) error {
	limit := 2 + drawInt(choices, maxExtraTags)
	values := make([]string, limit+1)
	for index := range values {
		values[index] = label(choices)
	}
	for _, count := range []int{limit - 1, limit, limit + 1} {
		var got struct {
			Tags []string
		}
		err := binder.Bind(ctx, &got, map[string][]string{"Tags": values[:count]}, binder.WithMaxSliceSize(limit))
		if count > limit {
			if err == nil {
				return property("slice-cap", "accepted %d values with limit %d", count, limit)
			}
		} else if err != nil || !reflect.DeepEqual(got.Tags, values[:count]) {
			return property("repeated-values", "limit %d count %d: got %q (%v), want %q", limit, count, got.Tags, err, values[:count])
		}
	}

	var limited order
	err := binder.Bind(ctx, &limited, map[string][]string{"title": {"a"}, "active": {"true"}}, binder.WithMaxFieldCount(1))
	if !errors.Is(err, core.ErrTooManyFields) {
		return property("field-cap", "expected ErrTooManyFields, got %v", err)
	}
	return assertBindError(err, "field-cap-status", http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request has too many fields or elements")
}

// checkJSONNumber checks integer binding around the double-precision boundary.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkJSONNumber(ctx context.Context, choices *tape.Tape) error {
	var got struct {
		Optional *string

		Raw stdjson.RawMessage

		ID int64
	}

	id := int64(doublePrecisionLimit) + 1 + 2*drawInt64(choices, precisionOffsetRange)
	if choices.Bool(1, 2) {
		id = -id
	}
	decimal := strconv.FormatInt(id, decimalBase)
	raw := `{"id":` + decimal + `}`
	input := `{"ID":` + decimal + `,"Optional":null,"Raw":` + raw + `}`
	if err := binder.BindJSON(ctx, &got, []byte(input)); err != nil {
		return property("json-subtree-acceptance", "bind precision case: %w", err)
	}
	if got.ID != id || string(got.Raw) != raw || got.Optional != nil {
		return property("json-precision", "wanted ID %s, raw %s and nil optional; got %+v raw %s", decimal, raw, got, got.Raw)
	}
	return nil
}

// checkActions checks action argument binding and result delivery.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkActions(ctx context.Context, choices *tape.Tape, _ *casefile.Case) error {
	want := contact{Name: label(choices), Count: safeconv.Int64ToInt16(drawInt64(choices, maxActionCount))}
	var got contact
	handler := daemon_adapters.NewActionHandler(nil, 1<<16, nil, security_dto.RateLimitValues{}, false, nil, nil)
	handler.Register(daemon_adapters.ActionHandlerEntry{
		Middlewares: nil,
		Name:        "fixture.Contact",
		Method:      http.MethodPost,
		HasSSE:      true,
		SSEGetAlias: true,
		Create:      func() any { return &got },
		Invoke:      func(context.Context, any, map[string]any) (any, error) { return nil, nil },
		Bind: func(ctx context.Context, _ any, args map[string]any) error {
			return binder.BindMap(ctx, &got, binder.ActionInputSource(args, "input"))
		},
	})
	router := chi.NewRouter()
	handler.Mount(router, "/_piko/actions")
	query := url.Values{"Name": {want.Name}, "Count": {strconv.Itoa(int(want.Count))}}
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/_piko/actions/fixture.Contact?"+query.Encode(), nil)
	request.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || got != want {
		return property("binding", "action bound %+v, want %+v (HTTP %d)", got, want, recorder.Code)
	}
	event, data, ok := strings.Cut(strings.TrimSpace(recorder.Body.String()), "\ndata: ")
	if !ok || event != "event: fixture" {
		return property("event-data", "unexpected event stream %q", recorder.Body.String())
	}
	var delivered contact
	if err := json.Unmarshal([]byte(data), &delivered); err != nil {
		return property("event-data", "undecodable event data %q: %w", data, err)
	}
	if delivered != want {
		return property("event-data", "delivered %+v, want %+v", delivered, want)
	}
	return nil
}
