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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/wdk/maths"
)

func TestInvoiceSchemasAndLimits(t *testing.T) {
	for seed := range uint64(64) {
		choices := tape.New(seed)
		require.NoError(t, checkInvoice(t.Context(), choices), "seed %d", seed)
		require.NoError(t, checkBinderLimits(t.Context(), choices), "seed %d", seed)
	}
}

func TestInvoiceOracle(t *testing.T) {
	good := priceLine{
		Code:   stockCode{ID: 7},
		Amount: maths.NewDecimalFromString("3.14"),
		Total:  maths.NewMoneyFromString("3.14", "GBP"),
	}
	require.NoError(t, checkInvoiceValue(&good, 7, "3.14"))
	for _, change := range []func(*priceLine){
		func(line *priceLine) { line.Code.ID = 8 },
		func(line *priceLine) { line.Amount = maths.NewDecimalFromString("3.15") },
		func(line *priceLine) { line.Total = maths.NewMoneyFromString("3.15", "GBP") },
		func(line *priceLine) { line.Total = maths.NewMoneyFromString("3.14", "USD") },
	} {
		changed := good
		change(&changed)
		assert.Error(t, checkInvoiceValue(&changed, 7, "3.14"), "incorrect invoice passed: %+v", changed)
	}
	assert.Error(t, checkInvoiceValue(nil, 7, "3.14"))
}
