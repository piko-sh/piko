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
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"piko.sh/piko/wdk/maths"
)

const (
	// maxPointerDepth bounds how many pointer indirections a field name resolution follows.
	maxPointerDepth = 16

	// maxNamespaceSegments bounds how deep a validator namespace is walked when resolving a
	// form field name. It sits far above any real form's nesting.
	maxNamespaceSegments = 64
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

// WithPlaygroundOption adds raw playground validator options that are passed to
// validator.New().
//
// Takes opts (...validator.Option) which specifies the validator options to add.
//
// Returns Option which appends the playground options.
func WithPlaygroundOption(opts ...validator.Option) Option {
	return func(c *config) {
		c.options = append(c.options, opts...)
	}
}

// Validator wraps the go-playground/validator with Piko-specific custom rules for Money
// and Decimal types pre-registered.
//
// It satisfies the bootstrap.StructValidator interface.
type Validator struct {
	// v holds the underlying playground validator instance.
	v *validator.Validate
}

// NewValidator creates a playground validator with Piko's custom Money/Decimal validation
// rules pre-registered.
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

// FieldErrors translates a validation failure into per-field messages, so the HTTP layer
// can answer with a 422 and a field-keyed map without knowing anything about
// go-playground. It satisfies the framework's optional FieldErrorReporter interface.
//
// Takes err (error) which is an error previously returned by Struct.
// Takes destination (any) which is the struct that was validated, used to resolve Go
// field names to the names the payload carries.
//
// Returns map[string][]string keyed by form field name, or nil when err did not originate
// here.
func (*Validator) FieldErrors(err error, destination any) map[string][]string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return nil
	}

	destinationType := reflect.TypeOf(destination)
	fields := make(map[string][]string, len(validationErrors))
	for _, fieldErr := range validationErrors {
		name := formFieldName(destinationType, fieldErr.Namespace())
		fields[name] = append(fields[name], describeFieldError(fieldErr))
	}
	return fields
}

// formFieldName converts a validator namespace such as "UpsertInput.Customer.CompanyName"
// into the name the form input actually carries, such as "customer.company_name", by
// walking the destination's bind and json tags. It falls back to the namespace when the
// path cannot be resolved, so a message is never lost.
//
// Takes destinationType (reflect.Type) which is the validated struct's type.
// Takes namespace (string) which is the validator's dotted path to the failing field.
//
// Returns string which is the form field name.
func formFieldName(destinationType reflect.Type, namespace string) string {
	segments := strings.Split(namespace, ".")
	if len(segments) < 2 || len(segments) > maxNamespaceSegments {
		return namespace
	}

	current := derefType(destinationType)
	names := make([]string, 0, len(segments)-1)
	for _, segment := range segments[1:] {
		fieldName, index := splitIndex(segment)

		if current == nil || current.Kind() != reflect.Struct {
			return namespace
		}
		field, found := current.FieldByName(fieldName)
		if !found {
			return namespace
		}

		current = derefType(elementType(field.Type))

		if field.Anonymous && index == "" && current != nil && current.Kind() == reflect.Struct {
			continue
		}

		name := bindFieldName(field)
		if name == "-" {
			return namespace
		}
		if index != "" {
			name += "[" + index + "]"
		}
		names = append(names, name)
	}

	if len(names) == 0 {
		return namespace
	}

	return strings.Join(names, ".")
}

// bindFieldName returns the name the binder accepts for a field: its bind tag when
// present, otherwise its json tag, otherwise its Go name.
//
// The precedence mirrors the binder's own, so a field carrying both tags is reported
// under the name the client actually submitted and the message can be attached to the
// input that produced it.
//
// Takes field (reflect.StructField) which is the field to name.
//
// Returns string which is the name used in form and JSON payloads, or "-" when the field
// is excluded from binding.
func bindFieldName(field reflect.StructField) string {
	for _, tagName := range []string{"bind", "json"} {
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}
		if tag == "-" {
			return "-"
		}
		if name, _, _ := strings.Cut(tag, ","); name != "" {
			return name
		}
	}
	return field.Name
}

// splitIndex separates a namespace segment such as "Contacts[0]" into its field name and
// index.
//
// Takes segment (string) which is one namespace segment.
//
// Returns string which is the field name.
// Returns string which is the index, empty when the segment is not indexed.
func splitIndex(segment string) (string, string) {
	open := strings.IndexByte(segment, '[')
	if open < 0 || !strings.HasSuffix(segment, "]") {
		return segment, ""
	}
	return segment[:open], segment[open+1 : len(segment)-1]
}

// derefType unwraps pointer types.
//
// Takes t (reflect.Type) which may be a pointer.
//
// Returns reflect.Type which is the pointed-to type, or t unchanged.
func derefType(t reflect.Type) reflect.Type {
	for range maxPointerDepth {
		if t == nil || t.Kind() != reflect.Ptr {
			return t
		}
		t = t.Elem()
	}
	return nil
}

// elementType unwraps slice, array and map types to their element type so a namespace can
// descend through collections.
//
// Takes t (reflect.Type) which may be a collection.
//
// Returns reflect.Type which is the element type, or t unchanged.
func elementType(t reflect.Type) reflect.Type {
	t = derefType(t)
	if t == nil {
		return nil
	}
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return t.Elem()
	default:
		return t
	}
}

// describeFieldError renders a message for one failed constraint.
//
// The message is rendered next to the input that produced it, so it is phrased for the
// person filling in the form rather than for the developer who wrote the tag. Tags with
// no phrasing of their own, including any the application registered, fall back to naming
// the constraint, which is still more use than nothing.
//
// Takes fieldErr (validator.FieldError) which describes the failure.
//
// Returns string which is the message for that field.
func describeFieldError(fieldErr validator.FieldError) string {
	param := fieldErr.Param()

	switch fieldErr.Tag() {
	case "required", "required_if", "required_with", "required_without":
		return "This field is required"
	case "email":
		return "Enter a valid email address"
	case "url", "uri":
		return "Enter a valid URL"
	case "min":
		return fmt.Sprintf("Must be at least %s", param)
	case "max":
		return fmt.Sprintf("Must be no more than %s", param)
	case "len":
		return fmt.Sprintf("Must be exactly %s", param)
	case "gte":
		return fmt.Sprintf("Must be %s or more", param)
	case "lte":
		return fmt.Sprintf("Must be %s or less", param)
	case "gt":
		return fmt.Sprintf("Must be more than %s", param)
	case "lt":
		return fmt.Sprintf("Must be less than %s", param)
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", strings.ReplaceAll(param, " ", ", "))
	case "eqfield", "eqcsfield":
		return "This field does not match"
	case "numeric":
		return "Enter a number"
	case "alpha":
		return "Use letters only"
	case "alphanum":
		return "Use letters and numbers only"
	}

	if param != "" {
		return fmt.Sprintf("Failed the %s check (%s)", fieldErr.Tag(), param)
	}
	return fmt.Sprintf("Failed the %s check", fieldErr.Tag())
}

// Underlying returns the raw *validator.Validate instance for advanced use cases such as
// registering additional validations after creation.
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

// numericValidator is satisfied by types that support error checking and sign/zero
// comparison - currently maths.Money and maths.Decimal.
type numericValidator interface {
	// Err returns any error stored in the numeric value.
	Err() error
}

// validateNumeric extracts a value of type T (or *T) from the field, returning nilResult
// when the pointer is nil.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
// Takes nilResult (bool) which is the result to return when the pointer is nil.
// Takes check (func(T) (bool, error)) which is the validation function to apply to the
// extracted value.
//
// Returns true when the check function passes for the extracted value.
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
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the money value is positive.
func isMoneyPositive(fl validator.FieldLevel) bool {
	return validateNumeric(fl, false, maths.Money.IsPositive)
}

// isMoneyNegative checks that a money field value is negative.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the money value is negative.
func isMoneyNegative(fl validator.FieldLevel) bool {
	return validateNumeric(fl, true, maths.Money.IsNegative)
}

// isMoneyNotZero checks that a money field value is not zero.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
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
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the decimal value is positive.
func isDecimalPositive(fl validator.FieldLevel) bool {
	return validateNumeric(fl, false, maths.Decimal.IsPositive)
}

// isDecimalNegative checks that a decimal field value is negative.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the decimal value is negative.
func isDecimalNegative(fl validator.FieldLevel) bool {
	return validateNumeric(fl, true, maths.Decimal.IsNegative)
}

// isDecimalNotZero checks that a decimal field value is not zero.
//
// Takes fl (validator.FieldLevel) which provides access to the field value.
//
// Returns bool which is true when the decimal value is not zero.
func isDecimalNotZero(fl validator.FieldLevel) bool {
	return validateNumeric[maths.Decimal](fl, true, func(d maths.Decimal) (bool, error) {
		isZero, err := d.IsZero()
		return !isZero, err
	})
}
