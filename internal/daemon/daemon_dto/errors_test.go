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

package daemon_dto

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/safeerror"
)

func TestNewValidationError(t *testing.T) {
	t.Parallel()

	fields := map[string]string{"email": "invalid", "name": "required"}
	err := NewValidationError(fields)

	assert.Equal(t, "validation failed", err.Error())
	assert.Equal(t, http.StatusUnprocessableEntity, err.StatusCode())
	assert.Equal(t, "VALIDATION_FAILED", err.ErrorCode())
	assert.Equal(t, fields, err.Fields)
}

func TestValidationField(t *testing.T) {
	t.Parallel()

	err := ValidationField("email", "invalid format")

	assert.Equal(t, "validation failed", err.Error())
	assert.Equal(t, map[string]string{"email": "invalid format"}, err.Fields)
}

func TestNotFound(t *testing.T) {
	t.Parallel()

	err := NotFound("user", "abc-123")

	assert.Equal(t, "user not found: abc-123", err.Error())
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
	assert.Equal(t, "NOT_FOUND", err.ErrorCode())
}

func TestNotFoundResource(t *testing.T) {
	t.Parallel()

	err := NotFoundResource("configuration")

	assert.Equal(t, "configuration not found", err.Error())
	assert.Equal(t, http.StatusNotFound, err.StatusCode())
}

func TestConflict(t *testing.T) {
	t.Parallel()

	err := Conflict("email already exists")

	assert.Equal(t, "email already exists", err.Error())
	assert.Equal(t, http.StatusConflict, err.StatusCode())
	assert.Equal(t, "CONFLICT", err.ErrorCode())
}

func TestConflictWithCode(t *testing.T) {
	t.Parallel()

	err := ConflictWithCode("email taken", "EMAIL_EXISTS")

	assert.Equal(t, "email taken", err.Error())
	assert.Equal(t, http.StatusConflict, err.StatusCode())
	assert.Equal(t, "EMAIL_EXISTS", err.ErrorCode())
}

func TestConflictError_DefaultCode(t *testing.T) {
	t.Parallel()

	err := &ConflictError{Message: "conflict"}
	assert.Equal(t, "CONFLICT", err.ErrorCode())
}

func TestForbidden(t *testing.T) {
	t.Parallel()

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		err := Forbidden("not allowed")
		assert.Equal(t, "not allowed", err.Error())
		assert.Equal(t, http.StatusForbidden, err.StatusCode())
		assert.Equal(t, "FORBIDDEN", err.ErrorCode())
	})

	t.Run("empty message", func(t *testing.T) {
		t.Parallel()

		err := &ForbiddenError{}
		assert.Equal(t, "access denied", err.Error())
	})
}

func TestUnauthorised(t *testing.T) {
	t.Parallel()

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		err := Unauthorised("session expired")
		assert.Equal(t, "session expired", err.Error())
		assert.Equal(t, http.StatusUnauthorized, err.StatusCode())
		assert.Equal(t, "UNAUTHORISED", err.ErrorCode())
	})

	t.Run("empty message", func(t *testing.T) {
		t.Parallel()

		err := &UnauthorisedError{}
		assert.Equal(t, "authentication required", err.Error())
	})
}

func TestBadRequest(t *testing.T) {
	t.Parallel()

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		err := BadRequest("missing header")
		assert.Equal(t, "missing header", err.Error())
		assert.Equal(t, http.StatusBadRequest, err.StatusCode())
		assert.Equal(t, "BAD_REQUEST", err.ErrorCode())
	})

	t.Run("empty message", func(t *testing.T) {
		t.Parallel()

		err := &BadRequestError{}
		assert.Equal(t, "bad request", err.Error())
	})
}

func TestPageError(t *testing.T) {
	t.Parallel()

	err := PageError(http.StatusTooManyRequests, "rate limited")

	assert.Equal(t, "rate limited", err.Error())
	assert.Equal(t, http.StatusTooManyRequests, err.StatusCode())
	assert.Equal(t, "PAGE_ERROR", err.ErrorCode())
	assert.Equal(t, http.StatusTooManyRequests, err.Status)
	assert.Equal(t, "rate limited", err.Message)
}

func TestGenericPageError(t *testing.T) {
	t.Parallel()

	t.Run("returns message as error string", func(t *testing.T) {
		t.Parallel()

		err := &GenericPageError{Message: "something went wrong", Status: 503}
		assert.Equal(t, "something went wrong", err.Error())
	})

	t.Run("returns configured status code", func(t *testing.T) {
		t.Parallel()

		err := &GenericPageError{Status: http.StatusServiceUnavailable}
		assert.Equal(t, http.StatusServiceUnavailable, err.StatusCode())
	})

	t.Run("returns PAGE_ERROR code", func(t *testing.T) {
		t.Parallel()

		err := &GenericPageError{}
		assert.Equal(t, "PAGE_ERROR", err.ErrorCode())
	})
}

func TestTeapot(t *testing.T) {
	t.Parallel()

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		err := Teapot("short and stout")
		assert.Equal(t, "short and stout", err.Error())
		assert.Equal(t, http.StatusTeapot, err.StatusCode())
		assert.Equal(t, "TEAPOT", err.ErrorCode())
	})

	t.Run("empty message uses default", func(t *testing.T) {
		t.Parallel()

		err := Teapot("")
		assert.Equal(t, "I'm a teapot", err.Error())
		assert.Equal(t, http.StatusTeapot, err.StatusCode())
		assert.Equal(t, "TEAPOT", err.ErrorCode())
	})
}

func TestTeapotError_DirectConstruction(t *testing.T) {
	t.Parallel()

	t.Run("with message", func(t *testing.T) {
		t.Parallel()

		err := &TeapotError{Message: "not a coffee machine"}
		assert.Equal(t, "not a coffee machine", err.Error())
	})

	t.Run("empty message", func(t *testing.T) {
		t.Parallel()

		err := &TeapotError{}
		assert.Equal(t, "I'm a teapot", err.Error())
	})
}

func TestErrorPageContext_RoundTrip(t *testing.T) {
	t.Parallel()

	epc := ErrorPageContext{
		StatusCode:   404,
		Message:      "page not found",
		OriginalPath: "/missing",
	}

	ctx := WithErrorPageContext(t.Context(), epc)
	got, ok := GetErrorPageContext(ctx)

	assert.True(t, ok)
	assert.Equal(t, 404, got.StatusCode)
	assert.Equal(t, "page not found", got.Message)
	assert.Equal(t, "/missing", got.OriginalPath)
}

func TestGetErrorPageContext_MissingContext(t *testing.T) {
	t.Parallel()

	got, ok := GetErrorPageContext(t.Context())

	assert.False(t, ok)
	assert.Equal(t, ErrorPageContext{}, got)
}

func TestActionError_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ ActionError = (*ValidationError)(nil)
	var _ ActionError = (*NotFoundError)(nil)
	var _ ActionError = (*ConflictError)(nil)
	var _ ActionError = (*ForbiddenError)(nil)
	var _ ActionError = (*UnauthorisedError)(nil)
	var _ ActionError = (*BadRequestError)(nil)
	var _ ActionError = (*GenericPageError)(nil)
	var _ ActionError = (*TeapotError)(nil)
}

func TestSafeError_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	var _ safeerror.Error = (*ValidationError)(nil)
	var _ safeerror.Error = (*NotFoundError)(nil)
	var _ safeerror.Error = (*ConflictError)(nil)
	var _ safeerror.Error = (*ForbiddenError)(nil)
	var _ safeerror.Error = (*UnauthorisedError)(nil)
	var _ safeerror.Error = (*BadRequestError)(nil)
	var _ safeerror.Error = (*GenericPageError)(nil)
	var _ safeerror.Error = (*TeapotError)(nil)
}
