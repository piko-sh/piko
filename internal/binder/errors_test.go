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

package binder

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type numericForm struct {
	Name     string
	Quantity int
	Price    float64
	Age      uint8
	Accepted bool
}

func errorOf[T any](_ T, err error) error {
	return err
}

func TestMultiError_AnswersAsAValidationFailure(t *testing.T) {
	t.Parallel()

	bindingErrors := MultiError{"Quantity": errors.New("could not convert")}

	assert.Equal(t, http.StatusUnprocessableEntity, bindingErrors.StatusCode())
	assert.Equal(t, "VALIDATION_FAILED", bindingErrors.ErrorCode())
	assert.Equal(t, (&ValidationFailedError{}).ErrorCode(), bindingErrors.ErrorCode(),
		"a value that will not convert and a broken validate tag must share one code")
	assert.Equal(t, "validation failed", bindingErrors.SafeMessage())
}

func TestFieldMessage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		err  error
		name string
		want string
	}{
		{
			name: "text in a whole number field",
			err:  errorOf(strconv.ParseInt("abc", 10, 64)),
			want: "Enter a whole number",
		},
		{
			name: "negative value in an unsigned field",
			err:  errorOf(strconv.ParseUint("-5", 10, 64)),
			want: "Enter a whole number that is 0 or more",
		},
		{
			name: "text in a decimal field",
			err:  errorOf(strconv.ParseFloat("1.2.3", 64)),
			want: "Enter a number",
		},
		{
			name: "text in a true or false field",
			err:  errorOf(strconv.ParseBool("maybe")),
			want: "Enter true or false",
		},
		{
			name: "whole number too large for its field",
			err:  errorOf(strconv.ParseInt("300", 10, 8)),
			want: "Number is too large or too small",
		},
		{
			name: "whole number too small for its field",
			err:  errorOf(strconv.ParseInt("-300", 10, 8)),
			want: "Number is too large or too small",
		},
		{
			name: "decimal too large for its field",
			err:  errorOf(strconv.ParseFloat("1e400", 64)),
			want: "Number is too large or too small",
		},
		{
			name: "number failure wrapped by the field setter",
			err: errSetField{
				path:      "Quantity",
				field:     "Quantity",
				fieldType: "int",
				err:       errorOf(strconv.ParseInt("abc", 10, 64)),
			},
			want: "Enter a whole number",
		},
		{
			name: "syntax failure from a kind of value with no message",
			err:  errorOf(strconv.ParseComplex("abc", 128)),
			want: "Enter a valid value",
		},
		{
			name: "syntax failure that is not a number",
			err:  fmt.Errorf("unquoting value: %w", strconv.ErrSyntax),
			want: "Enter a valid value",
		},
		{
			name: "value longer than allowed",
			err:  validateValueLength("Name", "too long", 3),
			want: "Enter a valid value",
		},
		{
			name: "key that is not a field path",
			err:  errInvalidPath{path: "1 + 1", err: errors.New("path cannot contain operators")},
			want: "Enter a valid value",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Error(t, testCase.err)
			assert.Equal(t, testCase.want, fieldMessage(testCase.err))
		})
	}
}

func TestMultiError_FieldMessages(t *testing.T) {
	t.Parallel()

	t.Run("names every field the binder could not fill", func(t *testing.T) {
		t.Parallel()

		var destination numericForm
		err := NewASTBinder().BindMap(t.Context(), &destination, map[string]any{
			"Quantity": "lots",
			"Price":    "cheap",
			"Age":      "300",
			"Accepted": "maybe",
			"Name":     "Ada",
		})
		require.Error(t, err)

		bindingErrors, ok := errors.AsType[MultiError](err)
		require.True(t, ok, "expected a MultiError, got %T", err)

		assert.Equal(t, map[string]string{
			"Quantity": "Enter a whole number",
			"Price":    "Enter a number",
			"Age":      "Number is too large or too small",
			"Accepted": "Enter true or false",
		}, bindingErrors.FieldMessages())
	})

	t.Run("does not repeat the value or the Go type", func(t *testing.T) {
		t.Parallel()

		var destination numericForm
		err := NewASTBinder().BindMap(t.Context(), &destination, map[string]any{
			"Quantity": "<script>",
		})
		bindingErrors, ok := errors.AsType[MultiError](err)
		require.True(t, ok, "expected a MultiError, got %T", err)

		message := bindingErrors.FieldMessages()["Quantity"]
		assert.NotContains(t, message, "<script>")
		assert.NotContains(t, message, "int")
		assert.NotContains(t, message, "strconv")
	})

	t.Run("returns nil when nothing failed", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, MultiError{}.FieldMessages())
		assert.Nil(t, MultiError(nil).FieldMessages())
	})

	t.Run("reports every field at the limit", func(t *testing.T) {
		t.Parallel()

		bindingErrors := multiErrorWithFields(t, int(DefaultMaxFieldErrors))

		messages := bindingErrors.FieldMessages()

		assert.Len(t, messages, int(DefaultMaxFieldErrors))
		assert.NotContains(t, messages, FieldErrorsTruncatedKey)
	})

	t.Run("trims fields past the limit and says so", func(t *testing.T) {
		t.Parallel()

		total := int(DefaultMaxFieldErrors) + 5
		bindingErrors := multiErrorWithFields(t, total)

		messages := bindingErrors.FieldMessages()

		assert.Len(t, messages, int(DefaultMaxFieldErrors)+1)
		assert.Equal(t,
			fmt.Sprintf("reporting %d of %d failed fields", DefaultMaxFieldErrors, total),
			messages[FieldErrorsTruncatedKey])
	})
}

func multiErrorWithFields(t *testing.T, count int) MultiError {
	t.Helper()

	bindingErrors := make(MultiError, count)
	for index := range count {
		bindingErrors["Field"+strconv.Itoa(index)] = errorOf(strconv.ParseInt("x", 10, 64))
	}
	return bindingErrors
}

func TestMultiError_ErrorIsCapped(t *testing.T) {
	t.Parallel()

	total := maxMultiErrorTextFields + 7
	bindingErrors := multiErrorWithFields(t, total)

	message := bindingErrors.Error()

	assert.Equal(t, maxMultiErrorTextFields, strings.Count(message, "  - field '"))
	assert.Contains(t, message, fmt.Sprintf("%d validation errors:", total))
	assert.Contains(t, message, "and 7 more")
	assert.Equal(t, message, bindingErrors.Error(), "the text must not depend on map order")
}

func TestErrSetField_ErrorTruncatesTheValue(t *testing.T) {
	t.Parallel()

	value := strings.Repeat("é", 500)
	fieldErr := errSetField{
		path:      "Quantity",
		field:     "Quantity",
		fieldType: "int",
		err:       errorOf(strconv.ParseInt(value, 10, 64)),
	}

	message := fieldErr.Error()

	assert.True(t, utf8.ValidString(message), "truncation must not cut a multi-byte rune")
	assert.Contains(t, message, strings.Repeat("é", maxQuotedValueRunes))
	assert.NotContains(t, message, strings.Repeat("é", maxQuotedValueRunes+1))
	assert.Contains(t, message, "invalid syntax", "the reason must survive the truncation")
	assert.ErrorIs(t, fieldErr, strconv.ErrSyntax)
}

func TestMultiError_FieldMessagesShortenLongKeys(t *testing.T) {
	t.Parallel()

	longKey := strings.Repeat("é", 4096)
	bindingErrors := MultiError{
		longKey: errors.New("could not convert"),
		"short": errors.New("could not convert"),
	}

	messages := bindingErrors.FieldMessages()

	require.Len(t, messages, 2)
	assert.Contains(t, messages, "short")
	for key := range messages {
		assert.LessOrEqual(t, utf8.RuneCountInString(key), maxFieldMessageKeyRunes,
			"a client chosen key must not be echoed back at full length")
		assert.True(t, utf8.ValidString(key))
	}
	assert.Contains(t, messages, strings.Repeat("é", maxFieldMessageKeyRunes))
}
