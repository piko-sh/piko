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
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
)

func TestMathsFacadeAPI(t *testing.T) {
	surface := apitest.Surface{
		"BigInt":             BigInt{},
		"Decimal":            Decimal{},
		"Money":              Money{},
		"CurrencyDefinition": CurrencyDefinition{},
		"RateMatrix":         RateMatrix{},
		"MatrixConverter":    MatrixConverter{},
		"ExchangeRates":      ExchangeRates{},
		"Converter":          Converter{},

		"NewBigIntFromString": NewBigIntFromString,
		"NewBigIntFromApd":    NewBigIntFromApd,
		"NewBigIntFromInt":    NewBigIntFromInt,
		"ZeroBigInt":          ZeroBigInt,
		"ZeroBigIntWithError": ZeroBigIntWithError,
		"OneBigInt":           OneBigInt,
		"TenBigInt":           TenBigInt,
		"HundredBigInt":       HundredBigInt,
		"SumBigInts":          SumBigInts,
		"AverageBigInts":      AverageBigInts,
		"MinBigInt":           MinBigInt,
		"MaxBigInt":           MaxBigInt,
		"AbsSumBigInts":       AbsSumBigInts,
		"SortBigInts":         SortBigInts,
		"SortBigIntsReverse":  SortBigIntsReverse,

		"NewDecimalFromString": NewDecimalFromString,
		"NewDecimalFromApd":    NewDecimalFromApd,
		"NewDecimalFromInt":    NewDecimalFromInt,
		"NewDecimalFromFloat":  NewDecimalFromFloat,
		"ZeroDecimal":          ZeroDecimal,
		"ZeroDecimalWithError": ZeroDecimalWithError,
		"OneDecimal":           OneDecimal,
		"TenDecimal":           TenDecimal,
		"HundredDecimal":       HundredDecimal,
		"SumDecimals":          SumDecimals,
		"AverageDecimals":      AverageDecimals,
		"MinDecimal":           MinDecimal,
		"MaxDecimal":           MaxDecimal,
		"AbsSumDecimals":       AbsSumDecimals,
		"SortDecimals":         SortDecimals,
		"SortDecimalsReverse":  SortDecimalsReverse,

		"RegisterCurrency":     RegisterCurrency,
		"NewMoneyFromDecimal":  NewMoneyFromDecimal,
		"NewMoneyFromString":   NewMoneyFromString,
		"NewMoneyFromInt":      NewMoneyFromInt,
		"NewMoneyFromMinorInt": NewMoneyFromMinorInt,
		"NewMoneyFromFloat":    NewMoneyFromFloat,
		"ZeroMoney":            ZeroMoney,
		"ZeroMoneyWithError":   ZeroMoneyWithError,
		"OneMoney":             OneMoney,
		"HundredMoney":         HundredMoney,
		"AverageMoney":         AverageMoney,
		"SumMoney":             SumMoney,
		"AbsSumMoney":          AbsSumMoney,
		"MinMoney":             MinMoney,
		"MaxMoney":             MaxMoney,
		"SortMoney":            SortMoney,
		"SortMoneyReverse":     SortMoneyReverse,
		"NewRateMatrix":        NewRateMatrix,
		"NewMatrixConverter":   NewMatrixConverter,
		"NewExchangeRates":     NewExchangeRates,
		"NewConverter":         NewConverter,
		"InvertRates":          InvertRates,
	}

	apitest.Check(t, surface, filepath.Join("maths.golden.yaml"))
}
