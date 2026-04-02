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

package maths

import (
	"strings"
	"testing"
)

func setupSimpleConverter(t *testing.T) *Converter {
	t.Helper()

	rates, err := NewExchangeRates("USD", map[string]Decimal{
		"EUR": NewDecimalFromString("0.92"),
		"GBP": NewDecimalFromString("0.80"),
		"JPY": NewDecimalFromInt(150),
	})
	if err != nil {
		t.Fatalf("Failed to create simple exchange rates: %v", err)
	}
	return NewConverter(rates)
}

func setupMatrixConverter(t *testing.T) *MatrixConverter {
	t.Helper()
	baseRates := map[string]Decimal{
		"EUR": NewDecimalFromString("0.92"),
		"GBP": NewDecimalFromString("0.80"),
		"JPY": NewDecimalFromInt(150),
	}

	overrideRates := map[string]map[string]Decimal{
		"GBP": {
			"JPY": NewDecimalFromString("188.25"),
		},
	}

	matrix, err := NewRateMatrix("USD", baseRates, overrideRates)
	if err != nil {
		t.Fatalf("Failed to create rate matrix: %v", err)
	}
	return NewMatrixConverter(matrix)
}

func TestSimpleConverter(t *testing.T) {
	converter := setupSimpleConverter(t)

	t.Run("Convert from base currency", func(t *testing.T) {
		mUSD := NewMoneyFromInt(100, "USD")
		mEUR := converter.Convert(mUSD, "EUR")
		checkMoney(t, mEUR, "92", "EUR", false)
	})

	t.Run("Convert to base currency", func(t *testing.T) {
		mGBP := NewMoneyFromInt(80, "GBP")
		mUSD := converter.Convert(mGBP, "USD")
		checkMoney(t, mUSD, "100", "USD", false)
	})

	t.Run("Cross-currency conversion", func(t *testing.T) {
		mGBP := NewMoneyFromInt(80, "GBP")
		mJPY := converter.Convert(mGBP, "JPY")
		checkMoney(t, mJPY, "15000", "JPY", false)
	})

	t.Run("No-op conversion returns same instance", func(t *testing.T) {
		mUSD := NewMoneyFromInt(123, "USD")
		if &mUSD == new(converter.Convert(mUSD, "USD")) {
			t.Error("expected no-op conversion to return the same instance for efficiency")
		}
	})

	t.Run("Error on unknown source currency", func(t *testing.T) {
		mCAD := NewMoneyFromInt(100, "CAD")
		mResult := converter.Convert(mCAD, "USD")
		if mResult.Err() == nil {
			t.Error("expected an error for unknown source currency")
		}
	})

	t.Run("Error on unknown target currency", func(t *testing.T) {
		mUSD := NewMoneyFromInt(100, "USD")
		mResult := converter.Convert(mUSD, "CAD")
		if mResult.Err() == nil {
			t.Error("expected an error for unknown target currency")
		}
	})

	t.Run("Error propagation from invalid source money", func(t *testing.T) {
		mInvalid := NewMoneyFromString("invalid", "USD")
		mResult := converter.Convert(mInvalid, "EUR")
		if mResult.Err() == nil {
			t.Error("expected error to be propagated from source money")
		}
	})
}

func TestNewExchangeRatesValidation(t *testing.T) {
	t.Run("Injects base currency when missing", func(t *testing.T) {
		rates, err := NewExchangeRates("USD", map[string]Decimal{"EUR": NewDecimalFromString("0.92")})
		if err != nil {
			t.Fatalf("did not expect error, but got %v", err)
		}
		if !rates.Rates["USD"].MustEquals(OneDecimal()) {
			t.Error("expected base currency to be injected with a rate of 1")
		}
	})

	t.Run("Error when provided base currency rate is not 1", func(t *testing.T) {
		_, err := NewExchangeRates("USD", map[string]Decimal{"USD": NewDecimalFromString("1.01")})
		if err == nil {
			t.Error("expected error when provided base currency rate is not 1")
		}
	})

	t.Run("Accepts provided base currency rate if it is 1", func(t *testing.T) {
		_, err := NewExchangeRates("USD", map[string]Decimal{"USD": OneDecimal()})
		if err != nil {
			t.Errorf("did not expect an error when correct base rate is provided, got %v", err)
		}
	})
}

func TestNewRateMatrix(t *testing.T) {
	t.Run("Success with base rates and overrides", func(t *testing.T) {
		baseRates := map[string]Decimal{"EUR": NewDecimalFromString("0.90")}
		overrides := map[string]map[string]Decimal{"EUR": {"GBP": NewDecimalFromString("0.85")}}
		matrix, err := NewRateMatrix("USD", baseRates, overrides)
		if err != nil {
			t.Fatalf("unexpected error creating matrix: %v", err)
		}

		checkDecimal(t, matrix.Rates["USD"]["EUR"], "0.9", false)
		checkDecimal(t, matrix.Rates["EUR"]["USD"], "1.111111111111111111111111111111111", false)
		checkDecimal(t, matrix.Rates["EUR"]["GBP"], "0.85", false)
		checkDecimal(t, matrix.Rates["GBP"]["EUR"], "1.176470588235294117647058823529412", false)
		checkDecimal(t, matrix.Rates["USD"]["USD"], "1", false)
		checkDecimal(t, matrix.Rates["EUR"]["EUR"], "1", false)
		checkDecimal(t, matrix.Rates["GBP"]["GBP"], "1", false)
	})

	t.Run("Success with nil overrides", func(t *testing.T) {
		_, err := NewRateMatrix("USD", map[string]Decimal{"EUR": NewDecimalFromString("0.90")}, nil)
		if err != nil {
			t.Errorf("NewRateMatrix should succeed with nil overrides, but got error: %v", err)
		}
	})

	t.Run("Error on zero base rate", func(t *testing.T) {
		_, err := NewRateMatrix("USD", map[string]Decimal{"EUR": ZeroDecimal()}, nil)
		if err == nil {
			t.Error("expected error for zero base rate")
		}
	})

	t.Run("Error on zero override rate", func(t *testing.T) {
		overrides := map[string]map[string]Decimal{"EUR": {"GBP": ZeroDecimal()}}
		_, err := NewRateMatrix("USD", nil, overrides)
		if err == nil {
			t.Error("expected error for zero override rate")
		}
	})
}

func TestMatrixConverter(t *testing.T) {
	converter := setupMatrixConverter(t)

	t.Run("Uses direct override rate when available", func(t *testing.T) {
		mGBP := NewMoneyFromInt(100, "GBP")
		mJPY := converter.Convert(mGBP, "JPY")

		checkMoney(t, mJPY, "18825", "JPY", false)
	})

	t.Run("Falls back to triangulation when no direct rate", func(t *testing.T) {
		mEUR := NewMoneyFromInt(100, "EUR")
		mJPY := converter.Convert(mEUR, "JPY")

		amount, _ := mJPY.Amount()
		checkDecimal(t, amount, "16304.34782608695652173913043478261", false)
	})

	t.Run("Error on no possible conversion path", func(t *testing.T) {
		mUSD := NewMoneyFromInt(100, "USD")

		matrix, _ := NewRateMatrix("USD", map[string]Decimal{"EUR": OneDecimal()}, nil)
		noCADConverter := NewMatrixConverter(matrix)

		result := noCADConverter.Convert(mUSD, "CAD")
		if result.Err() == nil {
			t.Error("expected error when no conversion path exists")
		}
	})
}

func TestMatrixConverterIntrospection(t *testing.T) {
	converter := setupMatrixConverter(t)

	t.Run("Supports", func(t *testing.T) {
		if !converter.Supports("USD") {
			t.Error("Supports should return true for base currency")
		}
		if !converter.Supports("GBP") {
			t.Error("Supports should return true for provided currency")
		}
		if converter.Supports("CAD") {
			t.Error("Supports should return false for unsupported currency")
		}
	})

	t.Run("CanConvert", func(t *testing.T) {
		if !converter.CanConvert("GBP", "JPY") {
			t.Error("CanConvert should be true for a direct override path")
		}
		if !converter.CanConvert("EUR", "JPY") {
			t.Error("CanConvert should be true for a triangulation path")
		}
		if !converter.CanConvert("USD", "USD") {
			t.Error("CanConvert should be true for a no-op conversion")
		}
		if converter.CanConvert("USD", "CAD") {
			t.Error("CanConvert should be false when target is unsupported")
		}
		if converter.CanConvert("CAD", "USD") {
			t.Error("CanConvert should be false when source is unsupported")
		}
	})
}

func TestMatrixConverterBulkConversion(t *testing.T) {
	converter := setupMatrixConverter(t)

	t.Run("Success with valid inputs", func(t *testing.T) {
		sources := []Money{
			NewMoneyFromInt(100, "USD"),
			NewMoneyFromInt(92, "EUR"),
			NewMoneyFromInt(100, "GBP"),
		}
		results, err := converter.ConvertAll(sources, "GBP")
		if err != nil {
			t.Fatalf("ConvertAll failed unexpectedly: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		checkMoney(t, results[0], "80", "GBP", false)
		checkMoney(t, results[1], "80", "GBP", false)
		checkMoney(t, results[2], "100", "GBP", false)
	})

	t.Run("Fails atomically on invalid money in slice", func(t *testing.T) {
		sources := []Money{
			NewMoneyFromInt(100, "USD"),
			NewMoneyFromString("invalid", "USD"),
			NewMoneyFromInt(100, "GBP"),
		}
		_, err := converter.ConvertAll(sources, "GBP")
		if err == nil {
			t.Fatal("expected error but got nil")
		}
		if !contains(err.Error(), "index 1") {
			t.Errorf("error message should specify the failing index: got %q", err.Error())
		}
	})

	t.Run("Fails atomically on unsupported currency in slice", func(t *testing.T) {
		sources := []Money{NewMoneyFromInt(100, "USD"), NewMoneyFromInt(100, "CAD")}
		_, err := converter.ConvertAll(sources, "GBP")
		if err == nil {
			t.Fatal("expected error but got nil")
		}
	})

	t.Run("Handles empty slice", func(t *testing.T) {
		results, err := converter.ConvertAll([]Money{}, "GBP")
		if err != nil {
			t.Fatalf("expected no error for empty slice, got %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results for empty slice, got %d", len(results))
		}
	})

	t.Run("Handles nil slice", func(t *testing.T) {
		results, err := converter.ConvertAll(nil, "GBP")
		if err != nil {
			t.Fatalf("expected no error for nil slice, got %v", err)
		}
		if results != nil {
			t.Errorf("expected nil result for nil slice, got %v", results)
		}
	})
}

func TestInvertRates(t *testing.T) {
	t.Run("Success re-basing from USD to EUR", func(t *testing.T) {
		original, _ := NewExchangeRates("USD", map[string]Decimal{
			"EUR": NewDecimalFromString("0.92"),
			"GBP": NewDecimalFromString("0.80"),
		})

		inverted, err := InvertRates(original, "EUR")
		if err != nil {
			t.Fatalf("InvertRates failed unexpectedly: %v", err)
		}

		if inverted.BaseCurrency != "EUR" {
			t.Errorf("expected new base currency to be EUR, got %s", inverted.BaseCurrency)
		}

		checkDecimal(t, inverted.Rates["EUR"], "1", false)

		checkDecimal(t, inverted.Rates["USD"], "1.086956521739130434782608695652174", false)

		checkDecimal(t, inverted.Rates["GBP"], "0.8695652173913043478260869565217391", false)
	})

	t.Run("Error when new base currency does not exist", func(t *testing.T) {
		original, _ := NewExchangeRates("USD", map[string]Decimal{"EUR": OneDecimal()})
		_, err := InvertRates(original, "CAD")
		if err == nil {
			t.Error("expected error when inverting to a non-existent base currency")
		}
	})
}

func TestCurrencyOddities(t *testing.T) {
	t.Run("Conversion with zero-decimal currency (JPY)", func(t *testing.T) {

		rates, _ := NewRateMatrix("USD", map[string]Decimal{"JPY": NewDecimalFromString("150.55")}, nil)
		converter := NewMatrixConverter(rates)

		mUSD := NewMoneyFromInt(1, "USD")
		mJPY := converter.Convert(mUSD, "JPY")

		checkMoney(t, mJPY, "150.55", "JPY", false)

		rounded := mJPY.RoundToStandard()
		checkMoney(t, rounded, "151", "JPY", false)
	})

	t.Run("Conversion of very large numbers", func(t *testing.T) {
		trillion := "1000000000000"
		rates, _ := NewRateMatrix("USD", map[string]Decimal{"EUR": NewDecimalFromString("0.92")}, nil)
		converter := NewMatrixConverter(rates)

		mTrillionUSD := NewMoneyFromString(trillion, "USD")
		mEUR := converter.Convert(mTrillionUSD, "EUR")
		checkMoney(t, mEUR, "920000000000", "EUR", false)
	})

	t.Run("Conversion of very small numbers (sub-minor unit)", func(t *testing.T) {
		rates, _ := NewRateMatrix("USD", map[string]Decimal{"EUR": NewDecimalFromString("0.5")}, nil)
		converter := NewMatrixConverter(rates)

		mSmallUSD := NewMoneyFromString("0.0001", "USD")
		mEUR := converter.Convert(mSmallUSD, "EUR")
		checkMoney(t, mEUR, "0.00005", "EUR", false)
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
