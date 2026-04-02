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

package templater_dto

import (
	"errors"
	"fmt"
)

// RedirectRequired signals that a redirect should happen instead of rendering.
// It implements the error interface but is not a failure; the HTTP handler
// should issue a redirect rather than return HTML.
type RedirectRequired struct {
	// Metadata holds the redirect URLs used to build the error message.
	Metadata InternalMetadata
}

// Error implements the error interface.
//
// Returns string which describes the redirect type and target URL.
func (r *RedirectRequired) Error() string {
	if r.Metadata.ServerRedirect != "" {
		return fmt.Sprintf("server redirect to: %s", r.Metadata.ServerRedirect)
	}
	if r.Metadata.ClientRedirect != "" {
		return fmt.Sprintf("client redirect to: %s", r.Metadata.ClientRedirect)
	}
	return "redirect required"
}

// IsRedirect checks whether an error is a RedirectRequired type.
// It uses errors.As to handle wrapped errors correctly.
//
// Takes err (error) which is the error to check.
//
// Returns *RedirectRequired which holds the redirect details if err is a
// redirect, or nil if it is not.
// Returns bool which is true if err is a redirect, false otherwise.
func IsRedirect(err error) (*RedirectRequired, bool) {
	if rr, ok := errors.AsType[*RedirectRequired](err); ok {
		return rr, true
	}
	return nil, false
}
