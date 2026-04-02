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

package validation_provider_playground

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/maths"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()

	require.NotNil(t, v)

	type MoneyStruct struct {
		Amount maths.Money `validate:"money_positive"`
	}

	zeroMoney := maths.NewMoneyFromString("0", "GBP")
	err := v.Struct(MoneyStruct{Amount: zeroMoney})
	assert.Error(t, err, "zero money should fail money_positive validation")

	positiveMoney := maths.NewMoneyFromString("100", "GBP")
	err = v.Struct(MoneyStruct{Amount: positiveMoney})
	assert.NoError(t, err, "positive money should pass money_positive validation")
}

func TestMoneyPositiveValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Amount maths.Money `validate:"money_positive"`
	}

	tests := []struct {
		name       string
		amount     string
		currency   string
		shouldPass bool
	}{
		{
			name:       "positive amount passes",
			amount:     "100.00",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "large positive amount passes",
			amount:     "999999.99",
			currency:   "USD",
			shouldPass: true,
		},
		{
			name:       "small positive amount passes",
			amount:     "0.01",
			currency:   "EUR",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			amount:     "0",
			currency:   "GBP",
			shouldPass: false,
		},
		{
			name:       "negative amount fails",
			amount:     "-50.00",
			currency:   "GBP",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			money := maths.NewMoneyFromString(tc.amount, tc.currency)
			s := TestStruct{Amount: money}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMoneyNegativeValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Amount maths.Money `validate:"money_negative"`
	}

	tests := []struct {
		name       string
		amount     string
		currency   string
		shouldPass bool
	}{
		{
			name:       "negative amount passes",
			amount:     "-100.00",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			amount:     "0",
			currency:   "GBP",
			shouldPass: false,
		},
		{
			name:       "positive amount fails",
			amount:     "50.00",
			currency:   "GBP",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			money := maths.NewMoneyFromString(tc.amount, tc.currency)
			s := TestStruct{Amount: money}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMoneyNotNegativeValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Amount maths.Money `validate:"money_not_negative"`
	}

	tests := []struct {
		name       string
		amount     string
		currency   string
		shouldPass bool
	}{
		{
			name:       "positive amount passes",
			amount:     "100.00",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "zero passes",
			amount:     "0",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "negative amount fails",
			amount:     "-50.00",
			currency:   "GBP",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			money := maths.NewMoneyFromString(tc.amount, tc.currency)
			s := TestStruct{Amount: money}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMoneyNotZeroValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Amount maths.Money `validate:"money_not_zero"`
	}

	tests := []struct {
		name       string
		amount     string
		currency   string
		shouldPass bool
	}{
		{
			name:       "positive amount passes",
			amount:     "100.00",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "negative amount passes",
			amount:     "-50.00",
			currency:   "GBP",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			amount:     "0",
			currency:   "GBP",
			shouldPass: false,
		},
		{
			name:       "zero with decimals fails",
			amount:     "0.00",
			currency:   "GBP",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			money := maths.NewMoneyFromString(tc.amount, tc.currency)
			s := TestStruct{Amount: money}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestMoneyPointerValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Amount *maths.Money `validate:"money_positive"`
	}

	t.Run("nil pointer fails", func(t *testing.T) {
		s := TestStruct{Amount: nil}
		err := v.Struct(s)
		assert.Error(t, err)
	})

	t.Run("valid pointer passes", func(t *testing.T) {
		s := TestStruct{Amount: new(maths.NewMoneyFromString("100", "GBP"))}
		err := v.Struct(s)
		assert.NoError(t, err)
	})
}

func TestDecimalPositiveValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Value maths.Decimal `validate:"decimal_positive"`
	}

	tests := []struct {
		name       string
		value      string
		shouldPass bool
	}{
		{
			name:       "positive value passes",
			value:      "100.00",
			shouldPass: true,
		},
		{
			name:       "small positive value passes",
			value:      "0.001",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			value:      "0",
			shouldPass: false,
		},
		{
			name:       "negative value fails",
			value:      "-50.00",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decimal := maths.NewDecimalFromString(tc.value)
			s := TestStruct{Value: decimal}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDecimalNegativeValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Value maths.Decimal `validate:"decimal_negative"`
	}

	tests := []struct {
		name       string
		value      string
		shouldPass bool
	}{
		{
			name:       "negative value passes",
			value:      "-100.00",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			value:      "0",
			shouldPass: false,
		},
		{
			name:       "positive value fails",
			value:      "50.00",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decimal := maths.NewDecimalFromString(tc.value)
			s := TestStruct{Value: decimal}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDecimalNotNegativeValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Value maths.Decimal `validate:"decimal_not_negative"`
	}

	tests := []struct {
		name       string
		value      string
		shouldPass bool
	}{
		{
			name:       "positive value passes",
			value:      "100.00",
			shouldPass: true,
		},
		{
			name:       "zero passes",
			value:      "0",
			shouldPass: true,
		},
		{
			name:       "negative value fails",
			value:      "-50.00",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decimal := maths.NewDecimalFromString(tc.value)
			s := TestStruct{Value: decimal}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDecimalNotZeroValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Value maths.Decimal `validate:"decimal_not_zero"`
	}

	tests := []struct {
		name       string
		value      string
		shouldPass bool
	}{
		{
			name:       "positive value passes",
			value:      "100.00",
			shouldPass: true,
		},
		{
			name:       "negative value passes",
			value:      "-50.00",
			shouldPass: true,
		},
		{
			name:       "zero fails",
			value:      "0",
			shouldPass: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decimal := maths.NewDecimalFromString(tc.value)
			s := TestStruct{Value: decimal}

			err := v.Struct(s)
			if tc.shouldPass {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestDecimalPointerValidation(t *testing.T) {
	v := NewValidator()

	type TestStruct struct {
		Value *maths.Decimal `validate:"decimal_positive"`
	}

	t.Run("nil pointer fails", func(t *testing.T) {
		s := TestStruct{Value: nil}
		err := v.Struct(s)
		assert.Error(t, err)
	})

	t.Run("valid pointer passes", func(t *testing.T) {
		s := TestStruct{Value: new(maths.NewDecimalFromString("100"))}
		err := v.Struct(s)
		assert.NoError(t, err)
	})
}

func TestCombinedValidations(t *testing.T) {
	v := NewValidator()

	type Invoice struct {
		Total    maths.Money   `validate:"money_positive"`
		Discount maths.Money   `validate:"money_not_negative"`
		TaxRate  maths.Decimal `validate:"decimal_not_negative"`
	}

	t.Run("valid invoice passes", func(t *testing.T) {
		invoice := Invoice{
			Total:    maths.NewMoneyFromString("100.00", "GBP"),
			Discount: maths.NewMoneyFromString("10.00", "GBP"),
			TaxRate:  maths.NewDecimalFromString("0.20"),
		}
		err := v.Struct(invoice)
		assert.NoError(t, err)
	})

	t.Run("zero discount is valid", func(t *testing.T) {
		invoice := Invoice{
			Total:    maths.NewMoneyFromString("100.00", "GBP"),
			Discount: maths.NewMoneyFromString("0", "GBP"),
			TaxRate:  maths.NewDecimalFromString("0.20"),
		}
		err := v.Struct(invoice)
		assert.NoError(t, err)
	})

	t.Run("zero total fails", func(t *testing.T) {
		invoice := Invoice{
			Total:    maths.NewMoneyFromString("0", "GBP"),
			Discount: maths.NewMoneyFromString("0", "GBP"),
			TaxRate:  maths.NewDecimalFromString("0.20"),
		}
		err := v.Struct(invoice)
		assert.Error(t, err)
	})

	t.Run("negative discount fails", func(t *testing.T) {
		invoice := Invoice{
			Total:    maths.NewMoneyFromString("100.00", "GBP"),
			Discount: maths.NewMoneyFromString("-10.00", "GBP"),
			TaxRate:  maths.NewDecimalFromString("0.20"),
		}
		err := v.Struct(invoice)
		assert.Error(t, err)
	})
}

func TestWithRegistration(t *testing.T) {
	customFn := func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "valid"
	}

	v := NewValidator(WithRegistration("is_valid", customFn))

	type TestStruct struct {
		Name string `validate:"is_valid"`
	}

	err := v.Struct(TestStruct{Name: "valid"})
	assert.NoError(t, err)

	err = v.Struct(TestStruct{Name: "invalid"})
	assert.Error(t, err)
}

func TestUnderlying(t *testing.T) {
	v := NewValidator()

	underlying := v.Underlying()
	require.NotNil(t, underlying)

	err := underlying.RegisterValidation("custom_test", func(_ validator.FieldLevel) bool {
		return true
	})
	assert.NoError(t, err)
}
