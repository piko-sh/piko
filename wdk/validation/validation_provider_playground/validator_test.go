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
	"errors"
	"reflect"
	"strings"
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

func TestValidatorFieldErrors(t *testing.T) {
	type target struct {
		Name  string      `json:"name"  validate:"required,max=5"`
		Price maths.Money `json:"price" validate:"money_not_negative"`
	}

	v := NewValidator()
	err := v.Struct(target{Name: "far too long", Price: maths.NewMoneyFromInt(-1, "GBP")})
	require.Error(t, err)

	fields := v.FieldErrors(err, target{})
	require.NotEmpty(t, fields)
	assert.Contains(t, fields, "name")
	assert.Contains(t, fields, "price")
}

func TestValidatorFieldErrorsIgnoresForeignErrors(t *testing.T) {
	v := NewValidator()

	assert.Nil(t, v.FieldErrors(errors.New("not from the validator"), struct{}{}))
}

func TestValidatorFieldErrorsNestedFormNames(t *testing.T) {
	type contact struct {
		ID string `json:"uuid" validate:"required"`
	}
	type customer struct {
		CompanyName string    `json:"company_name" validate:"required,max=75"`
		Contacts    []contact `json:"contacts"     validate:"omitempty,dive"`
	}
	type upsertInput struct {
		Customer customer `json:"customer"`
	}

	v := NewValidator()
	err := v.Struct(upsertInput{Customer: customer{
		CompanyName: strings.Repeat("X", 100),
		Contacts:    []contact{{ID: ""}},
	}})
	require.Error(t, err)

	fields := v.FieldErrors(err, upsertInput{})
	assert.Contains(t, fields, "customer.company_name")
	assert.Contains(t, fields, "customer.contacts[0].uuid")
}

func TestValidatorFieldErrorsPrefersBindTag(t *testing.T) {
	type target struct {
		Name string `bind:"user_name" json:"userName" validate:"required"`
	}

	v := NewValidator()
	err := v.Struct(target{})
	require.Error(t, err)

	fields := v.FieldErrors(err, target{})
	assert.Contains(t, fields, "user_name",
		"the key must be the name the binder accepts, so the client can attach the message")
	assert.NotContains(t, fields, "userName")
}

func TestValidatorFieldErrorsFallsBackToGoName(t *testing.T) {
	type target struct {
		Name string `validate:"required"`
	}

	v := NewValidator()
	err := v.Struct(target{})
	require.Error(t, err)

	assert.Contains(t, v.FieldErrors(err, target{}), "Name")
}

func TestValidatorFieldErrorsFlattensEmbeddedStructs(t *testing.T) {
	type address struct {
		City string `json:"city" validate:"required"`
	}
	type contact struct {
		address
		Email string `json:"email" validate:"required,email"`
	}

	v := NewValidator()
	err := v.Struct(contact{})
	require.Error(t, err)

	fields := v.FieldErrors(err, contact{})
	assert.Contains(t, fields, "city",
		"the binder promotes embedded fields, so the key must carry no embedding segment")
	assert.Contains(t, fields, "email")
	assert.NotContains(t, fields, "address.city")
}

func TestDerefType(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-pointer type unchanged", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, reflect.TypeFor[string](), derefType(reflect.TypeFor[string]()))
	})

	t.Run("unwraps nested pointers", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, reflect.TypeFor[string](), derefType(reflect.TypeFor[***string]()))
	})

	t.Run("returns nil for a nil type", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, derefType(nil))
	})

	t.Run("gives up beyond the pointer depth limit", func(t *testing.T) {
		t.Parallel()

		deep := reflect.TypeFor[string]()
		for range maxPointerDepth + 1 {
			deep = reflect.PointerTo(deep)
		}

		assert.Nil(t, derefType(deep),
			"a type nested deeper than the bound is refused rather than followed without limit")
	})

	t.Run("unwraps exactly at the pointer depth limit", func(t *testing.T) {
		t.Parallel()

		deep := reflect.TypeFor[string]()
		for range maxPointerDepth - 1 {
			deep = reflect.PointerTo(deep)
		}

		assert.Equal(t, reflect.TypeFor[string](), derefType(deep))
	})
}

func TestResolveSegmentNames(t *testing.T) {
	t.Parallel()

	type inner struct {
		City string `bind:"city"`
	}
	type embedded struct {
		Promoted string `bind:"promoted"`
	}
	type outer struct {
		embedded
		Inner    inner   `bind:"inner"`
		Contacts []inner `bind:"contacts"`
		Skipped  string  `bind:"-"`
	}

	testCases := []struct {
		name          string
		segments      []string
		expectedNames []string
		expectResolve bool
	}{
		{
			name:          "walks nested fields",
			segments:      []string{"Inner", "City"},
			expectedNames: []string{"inner", "city"},
			expectResolve: true,
		},
		{
			name:          "carries an index through",
			segments:      []string{"Contacts[2]", "City"},
			expectedNames: []string{"contacts[2]", "city"},
			expectResolve: true,
		},
		{
			name:          "contributes no name for an embedded segment",
			segments:      []string{"embedded", "Promoted"},
			expectedNames: []string{"promoted"},
			expectResolve: true,
		},
		{
			name:          "resolves to no names when every segment is embedded",
			segments:      []string{"embedded"},
			expectedNames: []string{},
			expectResolve: true,
		},
		{
			name:          "fails on an unknown field",
			segments:      []string{"Missing"},
			expectResolve: false,
		},
		{
			name:          "fails on a field excluded from binding",
			segments:      []string{"Skipped"},
			expectResolve: false,
		},
		{
			name:          "fails when a segment descends past a leaf",
			segments:      []string{"Inner", "City", "Deeper"},
			expectResolve: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			names, resolved := resolveSegmentNames(reflect.TypeFor[outer](), testCase.segments)

			assert.Equal(t, testCase.expectResolve, resolved)
			if !testCase.expectResolve {
				assert.Nil(t, names, "an unresolved path reports no names")
				return
			}
			assert.Equal(t, testCase.expectedNames, names)
		})
	}
}
