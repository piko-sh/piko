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

package i18n_domain

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/wdk/maths"
)

func TestFormatBuilder_F_Nil(t *testing.T) {
	assert.Equal(t, "", F(nil).String())
}

func TestFormatBuilder_F_String(t *testing.T) {
	assert.Equal(t, "hello", F("hello").String())
	assert.Equal(t, "", F("").String())
}

func TestFormatBuilder_F_Decimal(t *testing.T) {
	d := maths.NewDecimalFromString("19.99")
	assert.Equal(t, "19.99", F(d).String())
}

func TestFormatBuilder_F_DecimalPrecision(t *testing.T) {
	d := maths.NewDecimalFromString("19.9999")
	assert.Equal(t, "20", F(d).Precision(2).String())

	d2 := maths.NewDecimalFromString("19.456")
	assert.Equal(t, "19.46", F(d2).Precision(2).String())
}

func TestFormatBuilder_F_DecimalPrecisionTrailingZeros(t *testing.T) {

	d := maths.NewDecimalFromString("3.1")
	assert.Equal(t, "3.1", F(d).Precision(2).String())
}

func TestFormatBuilder_F_DecimalPointer(t *testing.T) {
	assert.Equal(t, "42.5", F(new(maths.NewDecimalFromString("42.5"))).String())

	var nilDec *maths.Decimal
	assert.Equal(t, "", F(nilDec).String())
}

func TestFormatBuilder_LF_Decimal(t *testing.T) {
	d := maths.NewDecimalFromString("1234567.89")
	assert.Equal(t, "1,234,567.89", NewLF(d, "en-GB").String())
	assert.Equal(t, "1.234.567,89", NewLF(d, "de-DE").String())
}

func TestFormatBuilder_LF_DecimalPrecision(t *testing.T) {
	d := maths.NewDecimalFromString("1234567.8999")
	assert.Equal(t, "1,234,567.9", NewLF(d, "en-GB").Precision(2).String())

	d2 := maths.NewDecimalFromString("1234567.456")
	assert.Equal(t, "1,234,567.46", NewLF(d2, "en-GB").Precision(2).String())
}

func TestFormatBuilder_F_Money(t *testing.T) {
	m := maths.NewMoneyFromString("1500.50", "USD")
	result := F(m).String()
	assert.Contains(t, result, "1500.50")
}

func TestFormatBuilder_LF_Money(t *testing.T) {
	m := maths.NewMoneyFromString("1500.50", "USD")
	result := NewLF(m, "en-US").String()
	assert.Contains(t, result, "1,500")
}

func TestFormatBuilder_F_MoneyPointer(t *testing.T) {
	assert.NotEmpty(t, F(new(maths.NewMoneyFromString("42.00", "GBP"))).String())

	var nilMoney *maths.Money
	assert.Equal(t, "", F(nilMoney).String())
}

func TestFormatBuilder_F_BigInt(t *testing.T) {
	b := maths.NewBigIntFromString("9223372036854775808")
	assert.Equal(t, "9223372036854775808", F(b).String())
}

func TestFormatBuilder_LF_BigInt(t *testing.T) {
	b := maths.NewBigIntFromString("9223372036854775808")
	result := NewLF(b, "en-GB").String()
	assert.Contains(t, result, ",")
	assert.Equal(t, "9,223,372,036,854,775,808", result)
}

func TestFormatBuilder_F_BigIntPointer(t *testing.T) {
	assert.Equal(t, "42", F(new(maths.NewBigIntFromString("42"))).String())

	var nilBig *maths.BigInt
	assert.Equal(t, "", F(nilBig).String())
}

func TestFormatBuilder_F_Float64(t *testing.T) {
	assert.Equal(t, "3.14159", F(3.14159).String())
}

func TestFormatBuilder_F_Float64Precision(t *testing.T) {
	assert.Equal(t, "3.14", F(3.14159).Precision(2).String())
	assert.Equal(t, "3", F(3.14159).Precision(0).String())
}

func TestFormatBuilder_LF_Float64(t *testing.T) {
	assert.Equal(t, "10,000.5", NewLF(10000.5, "en-GB").String())
	assert.Equal(t, "10.000,5", NewLF(10000.5, "de-DE").String())
}

func TestFormatBuilder_LF_Float64Precision(t *testing.T) {
	assert.Equal(t, "10,000.54", NewLF(10000.5403, "en-GB").Precision(2).String())
}

func TestFormatBuilder_F_Float32(t *testing.T) {
	assert.Equal(t, "3.14", F(float32(3.14)).Precision(2).String())
}

func TestFormatBuilder_F_Int(t *testing.T) {
	assert.Equal(t, "42", F(42).String())
	assert.Equal(t, "-100", F(-100).String())
}

func TestFormatBuilder_LF_Int(t *testing.T) {
	assert.Equal(t, "10,000", NewLF(10000, "en-GB").String())
	assert.Equal(t, "10.000", NewLF(10000, "de-DE").String())
}

func TestFormatBuilder_F_Int64(t *testing.T) {
	assert.Equal(t, "9223372036854775807", F(int64(9223372036854775807)).String())
}

func TestFormatBuilder_F_Uint(t *testing.T) {
	assert.Equal(t, "42", F(uint(42)).String())
}

func TestFormatBuilder_LF_Uint64(t *testing.T) {
	assert.Equal(t, "1,000,000", NewLF(uint64(1000000), "en-GB").String())
}

func TestFormatBuilder_F_Bool(t *testing.T) {
	assert.Equal(t, "true", F(true).String())
	assert.Equal(t, "false", F(false).String())
}

func TestFormatBuilder_F_Time(t *testing.T) {
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	result := F(ts).String()
	assert.Contains(t, result, "2026")
	assert.Contains(t, result, "15")
}

func TestFormatBuilder_LF_TimeShort(t *testing.T) {
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	result := NewLF(ts, "en-GB").Short().String()
	assert.Contains(t, result, "15/01/2026")
}

func TestFormatBuilder_LF_TimeLongDateOnly(t *testing.T) {
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	result := NewLF(ts, "en-GB").Long().DateOnly().String()
	assert.Contains(t, result, "January")
	assert.Contains(t, result, "2026")
	assert.NotContains(t, result, "14:30")
}

func TestFormatBuilder_LF_TimeTimeOnly(t *testing.T) {
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	result := NewLF(ts, "en-GB").Short().TimeOnly().String()
	assert.NotContains(t, result, "2026")
}

func TestFormatBuilder_F_TimePointer(t *testing.T) {
	assert.Contains(t, F(new(time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC))).String(), "2026")

	var nilTime *time.Time
	assert.Equal(t, "", F(nilTime).String())
}

func TestFormatBuilder_F_TimeUTC(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	ts := time.Date(2026, 1, 15, 10, 0, 0, 0, loc)
	result := F(ts).UTC().Short().String()
	assert.Contains(t, result, "15")
}

func TestFormatBuilder_F_DateTime(t *testing.T) {
	dt := NewDateTime(time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC))
	result := F(dt).String()
	assert.Contains(t, result, "2026")
}

func TestFormatBuilder_LF_DateTimeShort(t *testing.T) {
	dt := NewDateTime(time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC))
	result := NewLF(dt, "en-GB").Short().String()
	assert.Contains(t, result, "15/01/2026")
}

func TestFormatBuilder_F_DateTimePointer(t *testing.T) {
	assert.Contains(t, F(new(NewDateTime(time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)))).String(), "2026")

	var nilDT *DateTime
	assert.Equal(t, "", F(nilDT).String())
}

func TestFormatBuilder_LocaleOverride(t *testing.T) {
	d := maths.NewDecimalFromString("1234.56")
	result := F(d).Locale("de-DE").String()
	assert.Equal(t, "1.234,56", result)
}

func TestFormatBuilder_LocaleOverrideOnLF(t *testing.T) {
	d := maths.NewDecimalFromString("1234.56")
	result := NewLF(d, "en-GB").Locale("de-DE").String()
	assert.Equal(t, "1.234,56", result)
}

func TestFormatBuilder_PoolReuse(t *testing.T) {
	for range 100 {
		result := F(42).String()
		assert.Equal(t, "42", result)
	}
}

func TestFormatBuilder_PoolNoDataLeakage(t *testing.T) {
	_ = NewLF(maths.NewDecimalFromString("999.99"), "de-DE").Precision(1).String()

	result := F(42).String()
	assert.Equal(t, "42", result)
}

func TestFormatBuilder_ImplementsStringer(t *testing.T) {
	var s fmt.Stringer = F(42)
	assert.Equal(t, "42", s.String())
}

func TestFormatBuilder_PrecisionOnDateIsNoOp(t *testing.T) {
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)
	withPrecision := F(ts).Precision(2).String()
	without := F(ts).String()
	assert.Equal(t, without, withPrecision)
}

func TestFormatBuilder_ShortOnDecimalIsNoOp(t *testing.T) {
	d := maths.NewDecimalFromString("42.5")
	withShort := F(d).Short().String()
	without := F(d).String()
	assert.Equal(t, without, withShort)
}

func TestFormatBuilder_F_DefaultValue(t *testing.T) {
	type custom struct{ X int }
	result := F(custom{X: 42}).String()
	assert.Equal(t, "{42}", result)
}

func TestFormatBuilder_SmallInts(t *testing.T) {
	assert.Equal(t, "42", F(int8(42)).String())
	assert.Equal(t, "42", F(int16(42)).String())
	assert.Equal(t, "42", F(int32(42)).String())
	assert.Equal(t, "42", F(uint8(42)).String())
	assert.Equal(t, "42", F(uint16(42)).String())
	assert.Equal(t, "42", F(uint32(42)).String())
}

func TestFormatBuilder_F_ZeroValues(t *testing.T) {
	assert.Equal(t, "0", F(0).String())
	assert.Equal(t, "0", F(int64(0)).String())
	assert.Equal(t, "0", F(uint(0)).String())
	assert.Equal(t, "0", F(0.0).String())
	assert.Equal(t, "0", F(float32(0.0)).String())
	assert.Equal(t, "0", F(maths.NewDecimalFromString("0")).String())
	assert.Equal(t, "0", F(maths.NewBigIntFromString("0")).String())
	assert.Contains(t, F(maths.NewMoneyFromString("0", "USD")).String(), "0")
}

func TestFormatBuilder_F_NegativeValues(t *testing.T) {
	assert.Equal(t, "-42.5", F(-42.5).String())
	assert.Equal(t, "-19.99", F(maths.NewDecimalFromString("-19.99")).String())
	assert.Equal(t, "-20", F(maths.NewDecimalFromString("-19.99")).Precision(0).String())
	assert.Equal(t, "-999", F(maths.NewBigIntFromString("-999")).String())
}

func TestFormatBuilder_LF_NegativeDecimal(t *testing.T) {
	d := maths.NewDecimalFromString("-1234.56")
	result := NewLF(d, "en-GB").String()
	assert.Contains(t, result, "-")
	assert.Contains(t, result, "1,234")
}

func TestFormatBuilder_F_Duration(t *testing.T) {
	assert.Equal(t, "1m0s", F(time.Minute).String())
	assert.Equal(t, "1h30m0s", F(90*time.Minute).String())
	assert.Equal(t, "0s", F(time.Duration(0)).String())
}

func TestFormatBuilder_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			d := maths.NewDecimalFromString("19.99")
			result := F(d).Precision(2).String()
			assert.Equal(t, "19.99", result)
			result = NewLF(d, "en-GB").String()
			assert.Equal(t, "19.99", result)
		})
	}
	wg.Wait()
}

func TestFormatBuilder_ErrorStateDecimal(t *testing.T) {
	errDec := maths.ZeroDecimalWithError(errors.New("test error"))
	assert.Equal(t, "", F(errDec).String())
	assert.Equal(t, "", NewLF(errDec, "en-GB").String())
}

func TestFormatBuilder_ErrorStateBigInt(t *testing.T) {
	errBig := maths.ZeroBigIntWithError(errors.New("test error"))
	assert.Equal(t, "", F(errBig).String())
	assert.Equal(t, "", NewLF(errBig, "en-GB").String())
}

func TestFormatBuilder_ErrorStateMoney(t *testing.T) {
	errMoney := maths.ZeroMoneyWithError("USD", errors.New("test error"))
	assert.Equal(t, "", F(errMoney).String())
	assert.Equal(t, "", NewLF(errMoney, "en-US").String())
}
