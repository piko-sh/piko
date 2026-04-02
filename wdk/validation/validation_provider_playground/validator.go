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
	"reflect"

	"github.com/go-playground/validator/v10"

	"piko.sh/piko/wdk/maths"
)

// Option configures the playground validator.
type Option func(*config)

// config holds settings for the playground validator.
type config struct {
	// registrations maps custom tag names to their validation functions.
	registrations map[string]validator.Func

	// options holds raw playground validator options.
	options []validator.Option
}

// WithRegistration registers a custom validation function with the given tag.
//
// Takes tag (string) which is the validation tag name.
// Takes fn (validator.Func) which is the validation logic.
//
// Returns Option which registers the custom validation.
func WithRegistration(tag string, fn validator.Func) Option {
	return func(c *config) {
		c.registrations[tag] = fn
	}
}

// WithPlaygroundOption adds raw playground validator options that are passed
// to validator.New().
//
// Takes opts (...validator.Option) which specifies the validator options to
// add.
//
// Returns Option which appends the playground options.
func WithPlaygroundOption(opts ...validator.Option) Option {
	return func(c *config) {
		c.options = append(c.options, opts...)
	}
}

// Validator wraps the go-playground/validator with Piko-specific custom rules
// for Money and Decimal types pre-registered.
//
// It satisfies the bootstrap.StructValidator interface.
type Validator struct {
	// v holds the underlying playground validator instance.
	v *validator.Validate
}

// NewValidator creates a playground validator with Piko's custom Money/Decimal
// validation rules pre-registered.
//
// Takes opts (...Option) which provides optional configuration such as custom
// registrations or playground options.
//
// Returns *Validator which is ready for struct validation.
func NewValidator(opts ...Option) *Validator {
	cfg := &config{
		registrations: make(map[string]validator.Func),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	allOpts := make([]validator.Option, 0, 1+len(cfg.options))
	allOpts = append(allOpts, validator.WithRequiredStructEnabled())
	allOpts = append(allOpts, cfg.options...)

	v := validator.New(allOpts...)

	registerMoneyValidations(v)
	registerDecimalValidations(v)

	for tag, fn := range cfg.registrations {
		_ = v.RegisterValidation(tag, fn)
	}

	return &Validator{v: v}
}

// Struct validates a struct's exposed fields based on validation tags.
//
// Takes s (any) which is the struct to validate.
//
// Returns error when any field fails its validation constraint.
func (val *Validator) Struct(s any) error {
	return val.v.Struct(s)
}

// Underlying returns the raw *validator.Validate instance for advanced use
// cases such as registering additional validations after creation.
//
// Returns *validator.Validate which is the underlying playground validator.
func (val *Validator) Underlying() *validator.Validate {
	return val.v
}

// registerMoneyValidations adds money validation rules to the validator.
//
// Takes v (*validator.Validate) which is the validator to register rules with.
func registerMoneyValidations(v *validator.Validate) {
	_ = v.RegisterValidation("money_positive", isMoneyPositive)
	_ = v.RegisterValidation("money_negative", isMoneyNegative)
	_ = v.RegisterValidation("money_not_negative", isMoneyNotNegative)
	_ = v.RegisterValidation("money_not_zero", isMoneyNotZero)
}

// registerDecimalValidations adds decimal validation rules to a validator.
//
// Takes v (*validator.Validate) which receives the custom validation rules.
func registerDecimalValidations(v *validator.Validate) {
	_ = v.RegisterValidation("decimal_positive", isDecimalPositive)
	_ = v.RegisterValidation("decimal_negative", isDecimalNegative)
	_ = v.RegisterValidation("decimal_not_negative", isDecimalNotNegative)
	_ = v.RegisterValidation("decimal_not_zero", isDecimalNotZero)
}

// isMoneyNotNegative checks that a money field is not negative.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the field value is zero or positive.
func isMoneyNotNegative(fl validator.FieldLevel) bool {
	return !isMoneyNegative(fl)
}

// isDecimalNotNegative checks if a decimal field value is zero or positive.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the value is not negative.
func isDecimalNotNegative(fl validator.FieldLevel) bool {
	return !isDecimalNegative(fl)
}

// numericValidator is satisfied by types that support error checking and
// sign/zero comparison - currently maths.Money and maths.Decimal.
type numericValidator interface {
	// Err returns any error stored in the numeric value.
	Err() error
}

// validateNumeric extracts a value of type T (or *T) from the
// field, returning nilResult when the pointer is nil.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
// Takes nilResult (bool) which is the result to return when the
// pointer is nil.
// Takes check (func(T) (bool, error)) which is the validation
// function to apply to the extracted value.
//
// Returns true when the check function passes for the extracted
// value.
func validateNumeric[T numericValidator](
	fl validator.FieldLevel,
	nilResult bool,
	check func(T) (bool, error),
) bool {
	var value T
	field := fl.Field()
	if v, ok := reflect.TypeAssert[T](field); ok {
		value = v
	} else if vp, ok := reflect.TypeAssert[*T](field); ok {
		if vp == nil {
			return nilResult
		}
		value = *vp
	} else {
		return false
	}
	if value.Err() != nil {
		return false
	}
	result, err := check(value)
	if err != nil {
		return false
	}
	return result
}

// isMoneyPositive checks that a money field value is positive.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the money value is positive.
func isMoneyPositive(fl validator.FieldLevel) bool {
	return validateNumeric(fl, false, maths.Money.IsPositive)
}

// isMoneyNegative checks that a money field value is negative.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the money value is negative.
func isMoneyNegative(fl validator.FieldLevel) bool {
	return validateNumeric(fl, true, maths.Money.IsNegative)
}

// isMoneyNotZero checks that a money field value is not zero.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the money value is not zero.
func isMoneyNotZero(fl validator.FieldLevel) bool {
	return validateNumeric[maths.Money](fl, true, func(m maths.Money) (bool, error) {
		isZero, err := m.IsZero()
		return !isZero, err
	})
}

// isDecimalPositive checks that a decimal field value is positive.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the decimal value is positive.
func isDecimalPositive(fl validator.FieldLevel) bool {
	return validateNumeric(fl, false, maths.Decimal.IsPositive)
}

// isDecimalNegative checks that a decimal field value is negative.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the decimal value is negative.
func isDecimalNegative(fl validator.FieldLevel) bool {
	return validateNumeric(fl, true, maths.Decimal.IsNegative)
}

// isDecimalNotZero checks that a decimal field value is not zero.
//
// Takes fl (validator.FieldLevel) which provides access to the
// field value.
//
// Returns bool which is true when the decimal value is not zero.
func isDecimalNotZero(fl validator.FieldLevel) bool {
	return validateNumeric[maths.Decimal](fl, true, func(d maths.Decimal) (bool, error) {
		isZero, err := d.IsZero()
		return !isZero, err
	})
}
