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

// Package captcha_dto defines data transfer objects for the captcha module.
//
// It contains request/response types, configuration structs, provider type
// identifiers, and sentinel errors used across architectural boundaries in
// the captcha verification subsystem.
//
// The [VerifyResponse] type carries a normalised Score field ranging from
// 0.0 (bot) to 1.0 (human), allowing callers to apply adaptive thresholds
// regardless of the upstream provider's native scoring convention.
// Configuration types such as [ServiceConfig] and [ProviderType] are
// intended to be set once at initialisation and read thereafter.
package captcha_dto
