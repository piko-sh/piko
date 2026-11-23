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
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"piko.sh/fuzzkit/tape"
	"piko.sh/json"
	core "piko.sh/piko/internal/binder"
	public "piko.sh/piko/wdk/binder"
	"piko.sh/safeconv"
)

const (
	// seatsField is the wire name of the reservation seat count.
	seatsField = "seats"

	// confirmedField is the wire name of the reservation confirmation flag.
	confirmedField = "confirmed"

	// daysPerMonth is the number of drawn days valid in every month.
	daysPerMonth = 28

	// hoursPerDay is the number of hours in a day.
	hoursPerDay = 24

	// monthsPerYear is the number of months in a year.
	monthsPerYear = 12

	// baseYear is the earliest drawn reservation year.
	baseYear = 2000

	// yearRange is the number of distinct drawn reservation years.
	yearRange = 100

	// secondRange is the number of distinct drawn second offsets, including a leap second.
	secondRange = 61

	// secondsPerMinute is the number of seconds in a minute.
	secondsPerMinute = 60

	// minuteRange is the number of distinct drawn minute offsets.
	minuteRange = 60

	// zoneRange is the number of distinct zones per drawn position.
	zoneRange = 32

	// reservationFamilies is the number of reservation scenario families.
	reservationFamilies = 4

	// familyPaths selects the indexed-path family.
	familyPaths = 1

	// familyErrors selects the malformed-field family.
	familyErrors = 2

	// maxSeatCapacity bounds the drawn seat capacity below the byte limit.
	maxSeatCapacity = 253

	// maxExtraZones bounds the zones drawn beyond the first.
	maxExtraZones = 4

	// platformRange is the number of distinct drawn platform names.
	platformRange = 64

	// errorScenarioCount is the number of malformed-field scenarios.
	errorScenarioCount = 3

	// errorScenarioMalformedFlag selects a malformed confirmation flag.
	errorScenarioMalformedFlag = 1

	// errorScenarioBoth selects malformed seats and confirmation.
	errorScenarioBoth = 2

	// minJSONBodyRoom is the smallest drawn JSON body limit.
	minJSONBodyRoom = 4

	// jsonBodyRoomSpread is the number of distinct drawn JSON body limits.
	jsonBodyRoomSpread = 30
)

var (
	// errUnavailable marks a reservation above the configured capacity.
	errUnavailable = errors.New("reservation exceeds capacity")

	// errNotReservation reports a validator receiving a value other than *reservation.
	errNotReservation = errors.New("validated value is not a reservation")
)

// reservation tests binding of time, URL, address and numeric fields.
type reservation struct {
	// Start is the reservation start time.
	Start time.Time `bind:"start"`

	// Recipient is the bound reservation email address.
	Recipient mail.Address `bind:"recipient"`

	// Link is the bound reservation URL.
	Link url.URL `bind:"link"`

	// Window is the reservation duration.
	Window time.Duration `bind:"window"`

	// Seats is the reserved seat count.
	Seats uint8 `bind:"seats"`

	// Confirmed is the bound reservation confirmation flag.
	Confirmed bool `bind:"confirmed"`
}

// reservationModel pairs a reservation with its wire representation.
type reservationModel struct {
	// fields maps wire field names to encoded values.
	fields map[string]string

	// want holds the independently expected reservation.
	want reservation
}

// httpAnswer exposes an error's public HTTP status and code.
type httpAnswer interface {
	error

	// StatusCode returns the error's public HTTP status.
	//
	// Returns int which is the public HTTP response status.
	StatusCode() int

	// ErrorCode returns the error's public identifier.
	//
	// Returns string which is the public error identifier.
	ErrorCode() string

	// SafeMessage returns the error's public message.
	//
	// Returns string which is the message safe to expose to the caller.
	SafeMessage() string
}

// capacityValidator rejects reservations above a seat limit.
type capacityValidator struct {
	// seen holds the last value passed to validation.
	seen reservation

	// calls counts observed calls.
	calls int

	// maxSeats is the permitted seat count.
	maxSeats uint8
}

// reservationStop tests map fields inside a slice.
type reservationStop struct {
	// Notes maps note names to their text.
	Notes map[string]string `json:"notes"`
}

// zoneReservation holds the fields of one indexed reservation.
type zoneReservation struct {
	// Desk is the reserved desk name.
	Desk string `json:"desk"`

	// Seats is the reserved seat count.
	Seats uint8 `json:"seats"`
}

// indexedReservations tests maps and slices in reservation input.
type indexedReservations struct {
	// ByZone maps numeric zone IDs to reservations.
	ByZone map[int]zoneReservation `json:"by_zone"`

	// Stops holds reservation stops.
	Stops []reservationStop `json:"stops"`
}

// Struct validates the fixture reservation's seat count.
//
// Takes value (any) which must be a non-nil *reservation to validate.
//
// Returns error which reports a seat count above the configured capacity, or nil on
// success.
func (v *capacityValidator) Struct(value any) error {
	v.calls++
	booking, ok := value.(*reservation)
	if !ok || booking == nil {
		return fmt.Errorf("%w: got %T", errNotReservation, value)
	}
	v.seen = *booking
	if booking.Seats > v.maxSeats {
		return errUnavailable
	}
	return nil
}

// FieldErrors maps the fixture capacity error to its public field message.
//
// Takes err (error) which is the operation failure to inspect.
//
// Returns map[string][]string which maps a capacity refusal to field-validation messages,
// or is nil for other errors.
func (*capacityValidator) FieldErrors(err error, _ any) map[string][]string {
	if errors.Is(err, errUnavailable) {
		return map[string][]string{seatsField: {"insufficient seats"}}
	}
	return nil
}

// drawReservation draws a reservation and its independent wire representation.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns reservationModel which holds the drawn reservation and its encoded wire fields.
func drawReservation(choices *tape.Tape) reservationModel {
	day := 1 + drawInt(choices, daysPerMonth)
	hour := drawInt(choices, hoursPerDay)
	month := 1 + drawInt(choices, monthsPerYear)
	year := baseYear + drawInt(choices, yearRange)
	durationSeconds := drawInt(choices, secondRange)
	durationSeconds += secondsPerMinute * drawInt(choices, minuteRange)
	if choices.Bool(1, 2) {
		durationSeconds = -durationSeconds
	}
	zone := drawInt(choices, zoneRange)
	id := drawInt(choices, byteRange)
	recipient := fmt.Sprintf("desk%d@example.test", id)
	want := reservation{
		Start:     time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.UTC),
		Window:    time.Duration(durationSeconds) * time.Second,
		Link:      url.URL{Scheme: "https", Host: "example.test", Path: fmt.Sprintf("/zone/%d", zone), RawQuery: fmt.Sprintf("id=%d", id)},
		Recipient: mail.Address{Name: fmt.Sprintf("Desk %d", zone), Address: recipient},
		Seats:     drawByte(choices),
		Confirmed: choices.Bool(1, 2),
	}
	fields := map[string]string{
		"start":        fmt.Sprintf("%04d-%02d-%02dT%02d:00:00Z", year, month, day, hour),
		"window":       strconv.Itoa(durationSeconds) + "s",
		"link":         fmt.Sprintf("https://example.test/zone/%d?id=%d", zone, id),
		"recipient":    fmt.Sprintf("Desk %d <%s>", zone, recipient),
		seatsField:     strconv.Itoa(int(want.Seats)),
		confirmedField: strconv.FormatBool(want.Confirmed),
	}
	return reservationModel{want: want, fields: fields}
}

// bindReservation binds the selected wire representation into a reservation.
//
// Takes astBinder (*core.ASTBinder) which is the production binder under test.
// Takes representation (string) which selects form, map or JSON binding.
// Takes got (*reservation) which receives the bound reservation fields.
// Takes fields (map[string]string) which maps wire field names to their values.
// Takes opts (...core.Option) which selects binder limits and validation.
//
// Returns error which reports the production binder's refusal of the selected
// representation, or nil on success.
func bindReservation(ctx context.Context, astBinder *core.ASTBinder, representation string, got *reservation, fields map[string]string, opts ...core.Option) error {
	switch representation {
	case "form":
		form := make(map[string][]string, len(fields))
		for name, value := range fields {
			form[name] = []string{value}
		}
		return astBinder.Bind(ctx, got, form, opts...)
	case "map":
		object := make(map[string]any, len(fields))
		for name, value := range fields {
			object[name] = value
		}
		return astBinder.BindMap(ctx, got, object, opts...)
	case "json":
		wire, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("encoding reservation JSON: %w", err)
		}
		return astBinder.BindJSON(ctx, got, wire, opts...)
	default:
		return fmt.Errorf("unknown representation %q", representation)
	}
}

// assertReservation compares every bound reservation field with the model.
//
// Takes want (reservation) which holds the independently expected result.
// Takes got (reservation) which holds the observed result.
// Takes representation (string) which identifies the wire representation in failure
// messages.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func assertReservation(want, got reservation, representation string) error {
	if !reflect.DeepEqual(want, got) {
		return property("reservation-"+representation+"-value", "got %+v, want %+v", got, want)
	}
	return nil
}

// assertBindError checks a binding failure's public HTTP representation.
//
// Takes err (error) which is the operation failure to inspect.
// Takes propertyID (string) which identifies the checked property.
// Takes status (int) which is the expected HTTP status.
// Takes code (string) which is the expected public error code.
// Takes safe (string) which is the expected public error message.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func assertBindError(err error, propertyID string, status int, code, safe string) error {
	answer, ok := errors.AsType[httpAnswer](err)
	if !ok || answer.StatusCode() != status || answer.ErrorCode() != code || answer.SafeMessage() != safe {
		return property(propertyID, "got %T (%v), want HTTP %d %s %q", err, err, status, code, safe)
	}
	return nil
}

// checkReservation dispatches a saved reservation binding scenario.
//
// Takes data ([]byte) which holds the replayable choice bytes.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkReservation(ctx context.Context, data []byte) error {
	choices := tape.Replay(&tape.Recording{Data: data})
	switch family := choices.Uint(reservationFamilies); family {
	case 0:
		return checkReservationValidation(ctx, choices)
	case familyPaths:
		return checkReservationPaths(ctx, choices)
	case familyErrors:
		return checkReservationErrors(ctx, choices)
	default:
		return checkReservationWire(ctx, choices)
	}
}

// checkReservationValidation checks optional validation after reservation binding.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkReservationValidation(ctx context.Context, choices *tape.Tape) error {
	model := drawReservation(choices)
	maxSeats := safeconv.Uint64ToUint8(1 + choices.Uint(maxSeatCapacity))

	for _, seats := range []uint8{maxSeats - 1, maxSeats, maxSeats + 1} {
		model.want.Seats = seats
		model.fields[seatsField] = strconv.Itoa(int(seats))
		for _, representation := range []string{"form", "map", "json"} {
			if err := checkValidatedReservation(ctx, &model, representation, maxSeats); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkValidatedReservation binds one representation with opt-in validation.
//
// Takes model (*reservationModel) which holds the expected reservation and its wire
// fields.
// Takes representation (string) which selects form, map or JSON binding.
// Takes maxSeats (uint8) which is the validator's seat capacity.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkValidatedReservation(ctx context.Context, model *reservationModel, representation string, maxSeats uint8) error {
	validator := &capacityValidator{seen: reservation{}, calls: 0, maxSeats: maxSeats}
	astBinder := core.NewASTBinder()
	astBinder.SetStructValidator(validator)
	var got reservation
	err := bindReservation(ctx, astBinder, representation, &got, model.fields, core.WithValidation(true))
	if mismatch := assertReservation(model.want, got, representation); mismatch != nil {
		return mismatch
	}
	if validator.calls != 1 || !reflect.DeepEqual(validator.seen, model.want) {
		return property("validator-input", "%s: calls %d, observed %+v", representation, validator.calls, validator.seen)
	}
	seats := model.want.Seats
	if seats <= maxSeats {
		if err != nil {
			return property("validator-acceptance", "%s rejected %d seats at capacity %d: %w", representation, seats, maxSeats, err)
		}
		return nil
	}
	if detail := assertBindError(err, "validator-status", http.StatusUnprocessableEntity, "VALIDATION_FAILED", "validation failed"); detail != nil {
		return detail
	}
	failure, ok := errors.AsType[*core.ValidationFailedError](err)
	if !ok || !errors.Is(err, errUnavailable) || !errors.Is(err, core.ErrValidationFailed) ||
		!reflect.DeepEqual(failure.Fields, map[string][]string{seatsField: {"insufficient seats"}}) {
		return property("validator-fields", "%s incorrect validation fields or sentinel: %+v", representation, err)
	}

	if err := bindReservation(ctx, astBinder, representation, &got, model.fields, core.WithValidation(false)); err != nil || validator.calls != 1 {
		return property("validation-opt-out", "%s unexpectedly validated twice or rejected input: %v", representation, err)
	}
	return nil
}

// checkReservationPaths checks numeric map keys and nested field paths.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkReservationPaths(ctx context.Context, choices *tape.Tape) error {
	var got indexedReservations
	want := indexedReservations{ByZone: make(map[int]zoneReservation), Stops: nil}
	fields := map[string][]string{}
	count := 1 + drawInt(choices, maxExtraZones)
	last := 0
	for position := range count {
		zone := drawInt(choices, zoneRange) + position*zoneRange
		seats := drawByte(choices)
		desk := label(choices)
		want.ByZone[zone] = zoneReservation{Seats: seats, Desk: desk}
		fields[fmt.Sprintf("by_zone[%d].desk", zone)] = []string{desk}
		fields[fmt.Sprintf("by_zone[%d].seats", zone)] = []string{strconv.Itoa(int(seats))}
		index := last + drawInt(choices, 2)
		for len(want.Stops) <= index {
			want.Stops = append(want.Stops, reservationStop{})
		}
		key := pick(choices, []string{"colour", "a.b", "space key", "名"})
		text := label(choices)
		platform := fmt.Sprintf("platform-%d", choices.Uint(platformRange))
		want.Stops[index].Notes = map[string]string{key: text, "platform": platform}
		fields[fmt.Sprintf("stops[%d].notes['platform']", index)] = []string{platform}
		quote := pick(choices, []string{"'", `"`})
		fields[fmt.Sprintf("stops[%d].notes[%s%s%s]", index, quote, key, quote)] = []string{text}
		last = index + 1
	}
	if err := core.NewASTBinder().Bind(ctx, &got, fields); err != nil {
		return property("indexed-path-acceptance", "valid map/slice paths %v: %w", fields, err)
	}
	if !reflect.DeepEqual(want, got) {
		return property("indexed-path-value", "got %+v, want %+v for %v", got, want, fields)
	}
	return nil
}

// checkReservationErrors checks malformed reservation fields and public errors.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkReservationErrors(ctx context.Context, choices *tape.Tape) error {
	model := drawReservation(choices)
	wantMessages := map[string]string{}
	switch choices.Uint(errorScenarioCount) {
	case 0:
		model.fields[seatsField] = strconv.Itoa(byteRange + drawInt(choices, byteRange))
		wantMessages[seatsField] = "Number is too large or too small"
	case errorScenarioMalformedFlag:
		model.fields[confirmedField] = "perhaps"
		wantMessages[confirmedField] = "Enter true or false"
	case errorScenarioBoth:
		model.fields[seatsField] = "half"
		model.fields[confirmedField] = "sometimes"
		wantMessages[seatsField] = "Enter a whole number that is 0 or more"
		wantMessages[confirmedField] = "Enter true or false"
	}
	for _, representation := range []string{"form", "map", "json"} {
		astBinder := core.NewASTBinder()
		validator := &capacityValidator{seen: reservation{}, calls: 0, maxSeats: math.MaxUint8}
		astBinder.SetStructValidator(validator)
		err := bindReservation(ctx, astBinder, representation, &reservation{}, model.fields, core.WithValidation(true))
		if detail := assertBindError(err, "field-status", http.StatusUnprocessableEntity, "VALIDATION_FAILED", "validation failed"); detail != nil {
			return detail
		}
		fields, ok := errors.AsType[core.MultiError](err)
		if !ok || !maps.Equal(fields.FieldMessages(), wantMessages) || validator.calls != 0 {
			return property("conversion-field-set", "%s got %v; want %v and no validator calls (got %d)", representation, err, wantMessages, validator.calls)
		}
	}
	return nil
}

// checkReservationWire checks invalid top-level JSON and trailing input.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkReservationWire(ctx context.Context, choices *tape.Tape) error {
	token := pick(choices, []string{`["seat"]`, `true`, `"seat"`, `{"seats":`})
	astBinder := core.NewASTBinder()
	err := astBinder.BindJSON(ctx, &reservation{}, []byte(token))
	if detail := assertBindError(err, "json-shape", http.StatusBadRequest, "BAD_REQUEST", "request body is not a valid JSON object"); detail != nil {
		return detail
	}
	room := minJSONBodyRoom + drawInt64(choices, jsonBodyRoomSpread)
	astBinder.SetMaxBindJSONBytes(room)
	for _, size := range []int64{room - 1, room, room + 1} {
		body := []byte("{}" + strings.Repeat(" ", int(size)-2))
		err := astBinder.BindJSON(ctx, &reservation{}, body)
		if size <= room {
			if err != nil {
				return property("json-length-acceptance", "%d bytes at limit %d: %w", size, room, err)
			}
		} else if !errors.Is(err, public.ErrBindJSONTooLarge) {
			return property("json-length-sentinel", "%d bytes at limit %d: %v", size, room, err)
		} else if detail := assertBindError(err, "json-length-status", http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request has too many fields or elements"); detail != nil {
			return detail
		}
	}
	return nil
}
