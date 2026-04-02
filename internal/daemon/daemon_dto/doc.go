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

// Package daemon_dto contains request/response types, typed action
// errors, and shared dependencies used by HTTP handlers in the daemon
// module. It includes action metadata, cookie helpers, file upload
// handling, and batch action support.
//
// # Cookie management
//
// The package provides secure-by-default cookie helpers:
//
//	// Create a secure session cookie
//	cookie := daemon_dto.SessionCookie("session", token, expiresAt)
//
//	// Create a customised cookie with options
//	cookie := daemon_dto.Cookie("prefs", value, 24*time.Hour,
//	    daemon_dto.WithPath("/settings"),
//	    daemon_dto.WithSameSiteStrict(),
//	)
//
//	// Clear a cookie on logout
//	cookie := daemon_dto.ClearCookie("session")
//
// All cookies default to HttpOnly, Secure, and SameSite=Lax for
// security. Use the Insecure variants only for local development
// without HTTPS.
//
// # Action errors
//
// Typed error constructors map to standard HTTP status codes:
//
//	return nil, daemon_dto.NotFound("user", userID)       // 404
//	return nil, daemon_dto.Forbidden("not your resource") // 403
//	return nil, daemon_dto.Unauthorised("session expired") // 401
//	return nil, daemon_dto.BadRequest("missing header")   // 400
//	return nil, daemon_dto.Conflict("already exists")     // 409
//	return nil, daemon_dto.NewValidationError(fields)     // 422
package daemon_dto
