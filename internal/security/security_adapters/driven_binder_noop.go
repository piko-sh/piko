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

import (
	"net/http"

	"piko.sh/piko/internal/security/security_domain"
)

// noOpBinderAdapter implements RequestContextBinderAdapter with no action.
// It returns a fixed identifier, useful when CSRF protection is disabled.
type noOpBinderAdapter struct{}

var _ security_domain.RequestContextBinderAdapter = (*noOpBinderAdapter)(nil)

// GetBindingIdentifier returns a constant identifier for all requests.
//
// Returns string which is always "context_none" for this no-op adapter.
func (*noOpBinderAdapter) GetBindingIdentifier(_ *http.Request) string {
	return "context_none"
}

// NewNoOpBinderAdapter creates a new no-operation binder adapter.
//
// Returns security_domain.RequestContextBinderAdapter which performs no
// binding operations.
func NewNoOpBinderAdapter() security_domain.RequestContextBinderAdapter {
	return &noOpBinderAdapter{}
}
