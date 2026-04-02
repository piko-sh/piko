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

package security_adapters

import "piko.sh/piko/internal/security/security_dto"

// Type aliases for backward compatibility with consumers that already
// reference these types by their security_adapters package path.
type (
	// SecurityHeadersValues is an alias for security_dto.SecurityHeadersValues.
	SecurityHeadersValues = security_dto.SecurityHeadersValues

	// CookieSecurityValues is an alias for security_dto.CookieSecurityValues.
	CookieSecurityValues = security_dto.CookieSecurityValues

	// ReportingValues is an alias for security_dto.ReportingValues.
	ReportingValues = security_dto.ReportingValues

	// RateLimitValues is an alias for security_dto.RateLimitValues.
	RateLimitValues = security_dto.RateLimitValues

	// RateLimitTierValues is an alias for security_dto.RateLimitTierValues.
	RateLimitTierValues = security_dto.RateLimitTierValues
)
