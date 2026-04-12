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

package maths_test

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/maths"
)

func TestMathsFacadeAPI(t *testing.T) {
	surface := apitest.Surface{
		"BigInt":             maths.BigInt{},
		"Decimal":            maths.Decimal{},
		"Money":              maths.Money{},
		"CurrencyDefinition": maths.CurrencyDefinition{},
		"RateMatrix":         maths.RateMatrix{},
		"MatrixConverter":    maths.MatrixConverter{},
		"ExchangeRates":      maths.ExchangeRates{},
		"Converter":          maths.Converter{},

		"NewBigIntFromString": maths.NewBigIntFromString,
		"NewBigIntFromApd":    maths.NewBigIntFromApd,
		"NewBigIntFromInt":    maths.NewBigIntFromInt,
		"ZeroBigInt":          maths.ZeroBigInt,
		"ZeroBigIntWithError": maths.ZeroBigIntWithError,
		"OneBigInt":           maths.OneBigInt,
		"TenBigInt":           maths.TenBigInt,
		"HundredBigInt":       maths.HundredBigInt,
		"SumBigInts":          maths.SumBigInts,
		"AverageBigInts":      maths.AverageBigInts,
		"MinBigInt":           maths.MinBigInt,
		"MaxBigInt":           maths.MaxBigInt,
		"AbsSumBigInts":       maths.AbsSumBigInts,
		"SortBigInts":         maths.SortBigInts,
		"SortBigIntsReverse":  maths.SortBigIntsReverse,

		"NewDecimalFromString": maths.NewDecimalFromString,
		"NewDecimalFromApd":    maths.NewDecimalFromApd,
		"NewDecimalFromInt":    maths.NewDecimalFromInt,
		"NewDecimalFromFloat":  maths.NewDecimalFromFloat,
		"ZeroDecimal":          maths.ZeroDecimal,
		"ZeroDecimalWithError": maths.ZeroDecimalWithError,
		"OneDecimal":           maths.OneDecimal,
		"TenDecimal":           maths.TenDecimal,
		"HundredDecimal":       maths.HundredDecimal,
		"SumDecimals":          maths.SumDecimals,
		"AverageDecimals":      maths.AverageDecimals,
		"MinDecimal":           maths.MinDecimal,
		"MaxDecimal":           maths.MaxDecimal,
		"AbsSumDecimals":       maths.AbsSumDecimals,
		"SortDecimals":         maths.SortDecimals,
		"SortDecimalsReverse":  maths.SortDecimalsReverse,

		"RegisterCurrency":     maths.RegisterCurrency,
		"NewMoneyFromDecimal":  maths.NewMoneyFromDecimal,
		"NewMoneyFromString":   maths.NewMoneyFromString,
		"NewMoneyFromInt":      maths.NewMoneyFromInt,
		"NewMoneyFromMinorInt": maths.NewMoneyFromMinorInt,
		"NewMoneyFromFloat":    maths.NewMoneyFromFloat,
		"ZeroMoney":            maths.ZeroMoney,
		"ZeroMoneyWithError":   maths.ZeroMoneyWithError,
		"OneMoney":             maths.OneMoney,
		"HundredMoney":         maths.HundredMoney,
		"AverageMoney":         maths.AverageMoney,
		"SumMoney":             maths.SumMoney,
		"AbsSumMoney":          maths.AbsSumMoney,
		"MinMoney":             maths.MinMoney,
		"MaxMoney":             maths.MaxMoney,
		"SortMoney":            maths.SortMoney,
		"SortMoneyReverse":     maths.SortMoneyReverse,
		"NewRateMatrix":        maths.NewRateMatrix,
		"NewMatrixConverter":   maths.NewMatrixConverter,
		"NewExchangeRates":     maths.NewExchangeRates,
		"NewConverter":         maths.NewConverter,
		"InvertRates":          maths.InvertRates,
	}

	apitest.Check(t, surface, filepath.Join("maths.golden.yaml"))
}
