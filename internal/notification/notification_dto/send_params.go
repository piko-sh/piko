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

package notification_dto

// SendParams holds the data needed to send a notification to one or more
// providers.
type SendParams struct {
	// Content holds the message data including title, message text, and fields.
	Content NotificationContent `json:"content"`

	// ProviderOptions contains provider-specific configuration
	// (e.g. Slack channel, Discord webhook URL).
	ProviderOptions map[string]any `json:"provider_options,omitempty"`

	// Context holds metadata about the notification source, such as priority,
	// environment, service name, and trace ID.
	Context NotificationContext `json:"context"`

	// Providers lists the provider names for sending to multiple providers.
	// If empty, the default provider is used.
	Providers []string `json:"providers,omitempty"`
}
