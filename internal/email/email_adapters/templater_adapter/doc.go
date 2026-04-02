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

// Package templater_adapter implements the email domain's template
// rendering port by delegating to the templater domain. It translates
// templater domain types to email domain types, keeping both domains
// decoupled so the email hexagon does not depend directly on the
// templater hexagon.
//
// The adapter itself is stateless and safe for concurrent use. Thread
// safety depends on the underlying EmailTemplateService implementation.
package templater_adapter
