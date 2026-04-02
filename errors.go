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

package piko

import (
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/safeerror"
)

// Error is an error that carries a user-safe message separate from
// its internal cause.
//
// In production, only SafeMessage() reaches the user; the full error
// detail is logged server-side. In development mode (dev or dev-i),
// the full error string is shown for easier debugging.
//
// Any error in the chain can implement this interface; the error
// boundary will discover it via errors.As. Existing sentinels and
// errors.Is chains are preserved through Unwrap().
//
// Example:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    user, err := loadUser(r.Context(), id)
//	    if err != nil {
//	        return Response{}, piko.Metadata{},
//	            piko.NewError("could not load user profile", err)
//	    }
//	    return Response{User: user}, piko.Metadata{}, nil
//	}
type Error = safeerror.Error

var (
	// NewError wraps a cause error with a user-safe message. The cause's
	// Error() string is used for internal logging, while safeMessage is
	// the string shown to users in production.
	//
	// The returned error implements Unwrap(), so errors.Is and errors.As
	// continue to work through the chain.
	//
	// Example:
	//
	//	return piko.NewError("something went wrong", err)
	NewError = safeerror.NewError

	// Errorf creates an error with a user-safe message and a formatted
	// internal cause (using fmt.Errorf semantics for the internal part).
	//
	// Example:
	//
	//	return piko.Errorf("could not process order",
	//	    "loading order %s from database: %w", orderID, err)
	Errorf = safeerror.Errorf
)

// IsDevelopmentMode reports whether the current request is being
// served in development mode (dev or dev-i).
//
// Use this in error page Render functions to decide whether to show
// internal error details. Returns false when r is nil or the request
// context does not carry development mode information.
//
// Takes r (*RequestData) which provides the request context to check.
//
// Returns bool which is true when the request is being served
// in development mode.
//
// Example:
//
//	func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
//	    errCtx := piko.GetErrorContext(r)
//	    if errCtx == nil {
//	        return Response{Message: "Unknown error"}, piko.Metadata{}, nil
//	    }
//
//	    message := errCtx.Message
//	    if piko.IsDevelopmentMode(r) && errCtx.InternalMessage != "" {
//	        message = errCtx.InternalMessage
//	    }
//
//	    return Response{Code: errCtx.StatusCode, Message: message}, piko.Metadata{}, nil
//	}
func IsDevelopmentMode(r *RequestData) bool {
	if r == nil {
		return false
	}
	ctx := r.Context()
	if ctx == nil {
		return false
	}
	pctx := daemon_dto.PikoRequestCtxFromContext(ctx)
	if pctx == nil {
		return false
	}
	return pctx.DevelopmentMode
}
