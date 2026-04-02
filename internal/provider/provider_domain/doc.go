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

// Package provider_domain manages named provider instances (e.g. email
// senders, storage backends) through a type-safe, generic registry.
//
// The [ProviderRegistry] port and its production implementation
// [StandardRegistry] support registration, default selection, discovery,
// and graceful shutdown.
//
// # Usage
//
//	registry := provider_domain.NewStandardRegistry[EmailPort]("email")
//	_ = registry.RegisterProvider(ctx, "ses", sesProvider)
//	_ = registry.SetDefaultProvider(ctx, "ses")
//
//	provider, err := registry.GetProvider(ctx, "ses")
//
// # Thread safety
//
// [StandardRegistry] is safe for concurrent use. All read operations
// use a read lock and all write operations use an exclusive lock.
package provider_domain
