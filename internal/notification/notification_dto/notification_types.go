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

import "time"

// NotificationType defines the format and structure of notification content.
// It implements fmt.Stringer.
type NotificationType int

const (
	// NotificationTypePlain is a plain text notification with no formatting.
	NotificationTypePlain NotificationType = iota

	// NotificationTypeRich represents notifications with markdown, HTML, or basic
	// formatting.
	NotificationTypeRich

	// NotificationTypeTemplated represents notifications with provider-specific
	// rich formatting (e.g. Slack blocks, Discord embeds).
	NotificationTypeTemplated
)

// NotificationPriority defines how urgent a notification is.
type NotificationPriority int

const (
	// PriorityLow is for general information that does not need quick attention.
	PriorityLow NotificationPriority = iota

	// PriorityNormal represents normal-priority notifications (default).
	PriorityNormal

	// PriorityHigh represents high-priority notifications (important).
	PriorityHigh

	// PriorityCritical represents critical-priority notifications (urgent).
	PriorityCritical
)

// String returns the string representation of the notification type.
//
// Returns string which is the human-readable name of the type.
func (t NotificationType) String() string {
	switch t {
	case NotificationTypePlain:
		return "plain"
	case NotificationTypeRich:
		return "rich"
	case NotificationTypeTemplated:
		return "templated"
	default:
		return "unknown"
	}
}

// String returns the string representation of the notification priority.
//
// Returns string which is the priority level name such as "low", "normal",
// "high", "critical", or "unknown" for undefined values.
func (p NotificationPriority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// NotificationContext provides metadata about the source and context of a
// notification.
type NotificationContext struct {
	// Timestamp is when the notification was created.
	Timestamp time.Time `json:"timestamp"`

	// Source identifies where the notification originated (e.g. "logger",
	// "webhook", "sms").
	Source string `json:"source"`

	// Environment identifies the environment where the notification was created
	// (e.g. "dev", "staging", "prod").
	Environment string `json:"environment"`

	// Service identifies the service that generated the notification
	// in microservice architectures.
	Service string `json:"service,omitempty"`

	// TraceID is the OpenTelemetry trace ID for request correlation.
	TraceID string `json:"trace_id,omitempty"`

	// Priority is the urgency level of the notification.
	Priority NotificationPriority `json:"priority"`
}

// NotificationContent holds the message data to send in a notification.
type NotificationContent struct {
	// Fields holds key-value pairs for extra data.
	Fields map[string]string `json:"fields,omitempty"`

	// TemplateData contains provider-specific template data for templated
	// notifications.
	TemplateData map[string]any `json:"template_data,omitempty"`

	// Title is the notification heading or subject line.
	Title string `json:"title"`

	// Message is the main body text of the notification.
	Message string `json:"message"`

	// ImageURL is the URL of an inline image for providers that support it.
	ImageURL string `json:"image_url,omitempty"`

	// Type specifies how the notification is formatted and structured.
	Type NotificationType `json:"type"`
}
