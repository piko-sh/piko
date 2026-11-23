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
	"errors"
	"maps"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/fuzzkit/tape"
	core "piko.sh/piko/internal/binder"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestReservationFamilies(t *testing.T) {
	for family := range byte(4) {
		for variant := range byte(32) {
			data := make([]byte, 60)
			for index := range data {
				data[index] = variant*37 + byte(index)*13
			}
			data[0] = family
			require.NoError(t, run(t.Context(), &casefile.Case{Target: "binder", Data: data}), "family %d variant %d", family, variant)
		}
		require.NoError(t, run(t.Context(), &casefile.Case{Target: "binder-safe", Data: []byte{family}}), "short family %d", family)
	}
}

func TestReservationWireMutationsReachFields(t *testing.T) {
	model := drawReservation(tape.Replay(nil))
	mutations := map[string]string{
		"start":     "2026-12-31T23:00:00Z",
		"window":    "-61s",
		"recipient": "Other Desk <other@example.test>",
		"link":      "https://example.test/changed?id=1",
		"confirmed": "true",
		"seats":     "255",
	}
	for field, replacement := range mutations {
		fields := maps.Clone(model.fields)
		fields[field] = replacement
		for _, representation := range []string{"form", "map", "json"} {
			t.Run(field+"/"+representation, func(t *testing.T) {
				var got reservation
				require.NoError(t, bindReservation(t.Context(), core.NewASTBinder(), representation, &got, fields))
				assert.Error(t, assertReservation(model.want, got, representation), "modified field was indistinguishable from original")
			})
		}
	}
}

func TestBindReservationRejectsUnknownRepresentation(t *testing.T) {
	var got reservation
	assert.Error(t, bindReservation(t.Context(), core.NewASTBinder(), "xml", &got, nil))
}

func TestCapacityValidatorRejectsForeignValue(t *testing.T) {
	validator := &capacityValidator{maxSeats: 1}

	require.ErrorIs(t, validator.Struct("not a reservation"), errNotReservation)
	assert.Nil(t, validator.FieldErrors(errors.New("other"), nil))
	assert.Equal(t, map[string][]string{"seats": {"insufficient seats"}}, validator.FieldErrors(errUnavailable, nil))
}

func TestReservationErrorClassificationCanary(t *testing.T) {
	astBinder := core.NewASTBinder()
	err := astBinder.BindJSON(t.Context(), &reservation{}, []byte("[]"))
	require.NoError(t, assertBindError(err, "wire", http.StatusBadRequest, "BAD_REQUEST", "request body is not a valid JSON object"))
	for _, tc := range []struct {
		err           error
		code, message string
		status        int
	}{
		{err: nil, status: http.StatusBadRequest, code: "BAD_REQUEST", message: "request body is not a valid JSON object"},
		{err: errors.New("request body is not a valid JSON object"), status: http.StatusBadRequest, code: "BAD_REQUEST", message: "request body is not a valid JSON object"},
		{err: err, status: http.StatusUnprocessableEntity, code: "BAD_REQUEST", message: "request body is not a valid JSON object"},
		{err: err, status: http.StatusBadRequest, code: "VALIDATION_FAILED", message: "request body is not a valid JSON object"},
		{err: err, status: http.StatusBadRequest, code: "BAD_REQUEST", message: "wrong public message"},
	} {
		assert.Error(t, assertBindError(tc.err, "wire", tc.status, tc.code, tc.message), "wrong answer shape passed")
	}
}
