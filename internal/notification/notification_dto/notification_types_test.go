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

import "testing"

func TestNotificationType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want string
		typ  NotificationType
	}{
		{name: "plain", typ: NotificationTypePlain, want: "plain"},
		{name: "rich", typ: NotificationTypeRich, want: "rich"},
		{name: "templated", typ: NotificationTypeTemplated, want: "templated"},
		{name: "unknown", typ: NotificationType(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("NotificationType(%d).String() = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestNotificationPriority_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		want     string
		priority NotificationPriority
	}{
		{name: "low", priority: PriorityLow, want: "low"},
		{name: "normal", priority: PriorityNormal, want: "normal"},
		{name: "high", priority: PriorityHigh, want: "high"},
		{name: "critical", priority: PriorityCritical, want: "critical"},
		{name: "unknown", priority: NotificationPriority(99), want: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.priority.String(); got != tt.want {
				t.Errorf("NotificationPriority(%d).String() = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}
