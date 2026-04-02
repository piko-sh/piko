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

// Package driver_providers implements notification delivery adapters for
// third-party messaging services and generic endpoints.
//
// Each provider implements [notification_domain.NotificationProviderPort],
// translating the unified SendParams DTO into the target platform's wire
// format (Slack Block Kit, Discord embeds, Teams MessageCard, PagerDuty
// Events API v2, etc.) and delivering it via HTTP webhooks or APIs.
//
// # Integration
//
// Providers are registered with the notification domain service via
// RegisterProvider. Each constructor (e.g. NewSlackProvider) accepts a
// webhook URL (or routing key) and an optional *http.Client, returning
// a ready-to-use NotificationProviderPort.
package driver_providers
