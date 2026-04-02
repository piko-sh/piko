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
	"fmt"
	"maps"
)

// RateMatrix holds currency exchange rates relative to a base currency.
type RateMatrix struct {
	// Rates maps source currency codes to target currency codes to exchange
	// rates. Each outer key is a source currency, and each inner key is a
	// target currency with its conversion rate.
	Rates map[string]map[string]Decimal

	// BaseCurrency is the currency code used as the base for all conversions.
	BaseCurrency string
}

// MatrixConverter provides currency conversion using a rate matrix.
type MatrixConverter struct {
	// matrix holds the exchange rate data and base currency for conversions.
	matrix RateMatrix
}

// rateMatrixBuilder provides methods to build a currency rate matrix.
type rateMatrixBuilder struct {
	// matrix maps source currency to target currency to exchange rate.
	matrix map[string]map[string]Decimal
}

// ensureMap creates the inner map for a currency if it does not exist.
//
// Takes key (string) which identifies the currency to initialise.
func (b *rateMatrixBuilder) ensureMap(key string) {
	if _, ok := b.matrix[key]; !ok {
		b.matrix[key] = make(map[string]Decimal)
	}
}

// initialiseIdentityRates sets the identity rate (1) for each currency to
// itself.
//
// Takes currencies (map[string]struct{}) which specifies the set of currencies
// to initialise.
func (b *rateMatrixBuilder) initialiseIdentityRates(currencies map[string]struct{}) {
	for currency := range currencies {
		b.ensureMap(currency)
		b.matrix[currency][currency] = OneDecimal()
	}
}

// setBaseRates populates rates from the base currency to other currencies,
// including inverse rates.
//
// Takes baseCurrency (string) which specifies the source currency code.
// Takes baseRates (map[string]Decimal) which maps target currencies to their
// exchange rates from the base currency.
//
// Returns error when a rate is zero or the inverse rate calculation fails.
func (b *rateMatrixBuilder) setBaseRates(baseCurrency string, baseRates map[string]Decimal) error {
	for currency, rate := range baseRates {
		if rate.CheckIsZero() {
			return fmt.Errorf("maths: exchange rate for '%s' cannot be zero", currency)
		}
		b.matrix[baseCurrency][currency] = rate
		inverseRate := OneDecimal().Divide(rate)
		if inverseRate.Err() != nil {
			return fmt.Errorf("maths: failed to calculate inverse rate for '%s': %w", currency, inverseRate.Err())
		}
		b.matrix[currency][baseCurrency] = inverseRate
	}
	return nil
}

// setOverrideRates applies direct rate overrides between currency pairs,
// including inverse rates.
//
// Takes overrides (map[string]map[string]Decimal) which maps source currencies
// to target currencies with their exchange rates.
//
// Returns error when a rate is zero or the inverse rate cannot be calculated.
func (b *rateMatrixBuilder) setOverrideRates(overrides map[string]map[string]Decimal) error {
	for from, targets := range overrides {
		for to, rate := range targets {
			if rate.CheckIsZero() {
				return fmt.Errorf("maths: override exchange rate from '%s' to '%s' cannot be zero", from, to)
			}
			b.matrix[from][to] = rate
			inverseRate := OneDecimal().Divide(rate)
			if inverseRate.Err() != nil {
				return fmt.Errorf("maths: failed to calculate inverse override rate for '%s'->'%s': %w", from, to, inverseRate.Err())
			}
			b.matrix[to][from] = inverseRate
		}
	}
	return nil
}

// NewMatrixConverter creates a new converter using the given rate matrix.
//
// Takes matrix (RateMatrix) which provides the conversion rates to use.
//
// Returns *MatrixConverter which is ready to perform conversions.
func NewMatrixConverter(matrix RateMatrix) *MatrixConverter {
	return &MatrixConverter{matrix: matrix}
}

// Convert transforms a monetary value to a different currency.
//
// Attempts direct conversion first using the rate matrix. If no direct rate
// exists, triangulates through the base currency. When the source currency
// matches the target, returns the original value unchanged.
//
// Takes source (Money) which is the monetary value to convert.
// Takes targetCode (string) which is the ISO 4217 code of the target currency.
//
// Returns Money which contains the converted amount, or an error if the source
// has an error, no conversion path exists, or a calculation fails.
func (c *MatrixConverter) Convert(source Money, targetCode string) Money {
	if source.err != nil {
		return source
	}
	sourceCode, _ := source.CurrencyCode()
	if sourceCode == targetCode {
		return source
	}
	sourceAmount, err := source.Amount()
	if err != nil {
		return Money{err: err}
	}

	if targets, ok := c.matrix.Rates[sourceCode]; ok {
		if directRate, ok := targets[targetCode]; ok {
			convertedAmount := sourceAmount.Multiply(directRate)
			if convertedAmount.Err() != nil {
				return Money{err: fmt.Errorf("maths: direct conversion calculation failed: %w", convertedAmount.Err())}
			}
			return NewMoneyFromDecimal(convertedAmount, targetCode)
		}
	}

	base := c.matrix.BaseCurrency
	sourceToBaseRate, sOk := c.matrix.Rates[sourceCode][base]
	baseToTargetRate, tOk := c.matrix.Rates[base][targetCode]

	if sOk && tOk {
		convertedAmount := sourceAmount.Multiply(sourceToBaseRate).Multiply(baseToTargetRate)
		if convertedAmount.Err() != nil {
			return Money{err: fmt.Errorf("maths: triangulated conversion calculation failed: %w", convertedAmount.Err())}
		}
		return NewMoneyFromDecimal(convertedAmount, targetCode)
	}

	return Money{err: fmt.Errorf("maths: no conversion path found from '%s' to '%s'", sourceCode, targetCode)}
}

// ConvertAll converts a slice of Money values to the target currency.
//
// Takes sources ([]Money) which contains the monetary values to convert.
// Takes targetCode (string) which specifies the target currency code.
//
// Returns []Money which contains the converted monetary values.
// Returns error when any source Money has an error or conversion fails.
func (c *MatrixConverter) ConvertAll(sources []Money, targetCode string) ([]Money, error) {
	if sources == nil {
		return nil, nil
	}
	results := make([]Money, len(sources))
	for i, source := range sources {
		if source.Err() != nil {
			return nil, fmt.Errorf("maths: invalid money object provided at index %d: %w", i, source.Err())
		}

		converted := c.Convert(source, targetCode)
		if converted.Err() != nil {
			return nil, fmt.Errorf("maths: conversion failed at index %d for '%s': %w", i, source.MustString(), converted.Err())
		}
		results[i] = converted
	}
	return results, nil
}

// Supports reports whether the given currency code exists in the exchange rate
// matrix.
//
// Takes code (string) which is the currency code to check.
//
// Returns bool which is true if the currency code is supported.
func (c *MatrixConverter) Supports(code string) bool {
	_, ok := c.matrix.Rates[code]
	return ok
}

// CanConvert reports whether a conversion between two currencies is possible.
//
// Takes from (string) which is the source currency code.
// Takes to (string) which is the target currency code.
//
// Returns bool which is true if direct or base-currency conversion exists.
func (c *MatrixConverter) CanConvert(from, to string) bool {
	if from == to {
		return true
	}

	if targets, ok := c.matrix.Rates[from]; ok {
		if _, ok := targets[to]; ok {
			return true
		}
	}

	base := c.matrix.BaseCurrency
	_, fromToBaseOk := c.matrix.Rates[from][base]
	_, baseToToOk := c.matrix.Rates[base][to]

	return fromToBaseOk && baseToToOk
}

// ExchangeRates holds currency conversion rates relative to a base currency.
type ExchangeRates struct {
	// Rates maps currency codes to their exchange rate relative to the base currency.
	Rates map[string]Decimal

	// BaseCurrency is the base currency code used for exchange rate conversions.
	BaseCurrency string
}

// Converter provides currency conversion using a set of exchange rates.
type Converter struct {
	// rates holds the exchange rates used for currency conversion.
	rates ExchangeRates
}

// NewConverter creates a currency converter with the given exchange rates.
//
// Takes rates (ExchangeRates) which provides the conversion rates to use.
//
// Returns *Converter which is ready to convert between currencies.
func NewConverter(rates ExchangeRates) *Converter {
	return &Converter{rates: rates}
}

// Convert changes a money value from one currency to another.
//
// Takes source (Money) which is the money value to convert.
// Takes targetCode (string) which is the ISO currency code for the target.
//
// Returns Money which holds the converted amount in the target currency.
// If source already has an error, it is returned unchanged. If the source
// and target currencies match, the original value is returned. Any errors
// during conversion are stored in the returned Money's error field.
func (c *Converter) Convert(source Money, targetCode string) Money {
	if source.err != nil {
		return source
	}
	sourceCode, _ := source.CurrencyCode()
	if sourceCode == targetCode {
		return source
	}
	sourceAmount, err := source.Amount()
	if err != nil {
		return Money{err: err}
	}

	sourceRate, sourceOk := c.rates.Rates[sourceCode]
	if !sourceOk {
		return Money{err: fmt.Errorf("maths: source currency '%s' not found in exchange rates", sourceCode)}
	}
	targetRate, targetOk := c.rates.Rates[targetCode]
	if !targetOk {
		return Money{err: fmt.Errorf("maths: target currency '%s' not found in exchange rates", targetCode)}
	}

	convertedAmount := sourceAmount.Divide(sourceRate).Multiply(targetRate)
	if convertedAmount.Err() != nil {
		return Money{err: fmt.Errorf("maths: conversion calculation failed: %w", convertedAmount.Err())}
	}
	return NewMoneyFromDecimal(convertedAmount, targetCode)
}

// NewRateMatrix creates a currency exchange rate matrix from base rates and
// optional overrides.
//
// The matrix allows conversion between any pair of currencies. Inverse rates
// are calculated automatically for each defined rate.
//
// Takes baseCurrency (string) which is the reference currency for base rates.
// Takes baseRates (map[string]Decimal) which maps currencies to their exchange
// rates relative to the base currency.
// Takes overrides (map[string]map[string]Decimal) which provides direct rates
// between specific currency pairs, bypassing base rate calculation.
//
// Returns RateMatrix which contains the complete conversion matrix.
// Returns error when any rate is zero or inverse calculation fails.
func NewRateMatrix(baseCurrency string, baseRates map[string]Decimal, overrides map[string]map[string]Decimal) (RateMatrix, error) {
	builder := newRateMatrixBuilder()

	allCurrencies := collectAllCurrencies(baseCurrency, baseRates, overrides)
	builder.initialiseIdentityRates(allCurrencies)

	if err := builder.setBaseRates(baseCurrency, baseRates); err != nil {
		return RateMatrix{}, err
	}
	if err := builder.setOverrideRates(overrides); err != nil {
		return RateMatrix{}, err
	}

	return RateMatrix{
		BaseCurrency: baseCurrency,
		Rates:        builder.matrix,
	}, nil
}

// NewExchangeRates creates an ExchangeRates with the given base currency and
// rate mappings.
//
// The base currency rate is set to 1 if not in the rates map. If present, it
// must equal 1.
//
// Takes baseCurrency (string) which is the currency code for the base rate.
// Takes rates (map[string]Decimal) which maps currency codes to their rates.
//
// Returns ExchangeRates which contains the checked rate mappings.
// Returns error when the base currency rate is present but not equal to 1.
func NewExchangeRates(baseCurrency string, rates map[string]Decimal) (ExchangeRates, error) {
	validatedRates := make(map[string]Decimal, len(rates)+1)
	maps.Copy(validatedRates, rates)

	rate, ok := validatedRates[baseCurrency]
	if ok {
		isOne, err := rate.Equals(OneDecimal())
		if err != nil {
			return ExchangeRates{}, fmt.Errorf("maths: error checking base currency rate: %w", err)
		}
		if !isOne {
			return ExchangeRates{}, fmt.Errorf("maths: rate for provided base currency '%s' must be 1, but got %s", baseCurrency, rate.MustString())
		}
	} else {
		validatedRates[baseCurrency] = OneDecimal()
	}

	return ExchangeRates{
		BaseCurrency: baseCurrency,
		Rates:        validatedRates,
	}, nil
}

// InvertRates converts exchange rates to use a different base currency.
//
// Takes original (ExchangeRates) which contains the rates to convert.
// Takes newBaseCurrency (string) which specifies the new base currency code.
//
// Returns ExchangeRates which contains the recalculated rates relative to the
// new base.
// Returns error when the new base currency is not found in the original rates
// or when a rate calculation fails.
func InvertRates(original ExchangeRates, newBaseCurrency string) (ExchangeRates, error) {
	oldBaseToNewBaseRate, ok := original.Rates[newBaseCurrency]
	if !ok {
		return ExchangeRates{}, fmt.Errorf("maths: new base currency '%s' not found in original rate set", newBaseCurrency)
	}

	newRates := make(map[string]Decimal, len(original.Rates))
	for currency, oldRate := range original.Rates {
		newRate := oldRate.Divide(oldBaseToNewBaseRate)
		if newRate.Err() != nil {
			return ExchangeRates{}, fmt.Errorf("maths: failed to calculate new rate for '%s': %w", currency, newRate.Err())
		}
		newRates[currency] = newRate
	}

	return NewExchangeRates(newBaseCurrency, newRates)
}

// newRateMatrixBuilder creates a new builder for rate matrix construction.
//
// Returns *rateMatrixBuilder which is an empty builder ready for use.
func newRateMatrixBuilder() *rateMatrixBuilder {
	return &rateMatrixBuilder{
		matrix: make(map[string]map[string]Decimal),
	}
}

// collectAllCurrencies gathers all currency codes from base rates and
// overrides.
//
// Takes baseCurrency (string) which is the primary currency to include.
// Takes baseRates (map[string]Decimal) which contains the standard rates.
// Takes overrides (map[string]map[string]Decimal) which contains custom rates.
//
// Returns map[string]struct{} which is a set of all unique currency codes.
func collectAllCurrencies(baseCurrency string, baseRates map[string]Decimal, overrides map[string]map[string]Decimal) map[string]struct{} {
	allCurrencies := map[string]struct{}{baseCurrency: {}}
	for currency := range baseRates {
		allCurrencies[currency] = struct{}{}
	}
	for from, targets := range overrides {
		allCurrencies[from] = struct{}{}
		for to := range targets {
			allCurrencies[to] = struct{}{}
		}
	}
	return allCurrencies
}
