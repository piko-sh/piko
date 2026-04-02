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

package integrations

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/internal/notification/notification_dto"
)

// NotificationServiceAdapter implements logger_domain.NotificationPort by
// wrapping the notification service. It converts grouped errors from the
// logger into notification send parameters.
type NotificationServiceAdapter struct {
	// service handles sending notifications.
	service notification_domain.Service
}

var _ logger_domain.NotificationPort = (*NotificationServiceAdapter)(nil)

// SendGroupedErrors converts grouped errors to notifications and sends them.
//
// Takes batch (map[string]*logger_domain.GroupedError) which contains the
// grouped errors to send.
//
// Returns error when the notification service fails to send.
func (a *NotificationServiceAdapter) SendGroupedErrors(ctx context.Context, batch map[string]*logger_domain.GroupedError) error {
	if len(batch) == 0 {
		return nil
	}

	notifications := make([]*notification_dto.SendParams, 0, len(batch))

	for _, errInfo := range batch {
		params := a.convertGroupedErrorToNotification(errInfo)
		notifications = append(notifications, params)
	}

	return a.service.SendBulk(ctx, notifications)
}

// convertGroupedErrorToNotification converts a grouped error to notification
// parameters.
//
// Takes errInfo (*logger_domain.GroupedError) which contains the error details
// and occurrence count.
//
// Returns *notification_dto.SendParams which contains the formatted
// notification ready for sending.
func (*NotificationServiceAdapter) convertGroupedErrorToNotification(errInfo *logger_domain.GroupedError) *notification_dto.SendParams {
	r := errInfo.LogRecord

	fields := make(map[string]string)
	r.Attrs(func(attr slog.Attr) bool {
		fields[attr.Key] = fmt.Sprintf("%v", attr.Value.Any())
		return true
	})

	if errInfo.Count > 1 {
		fields["occurrences"] = fmt.Sprintf("%d", errInfo.Count)
	}

	if errInfo.SourceFile != "" {
		fields["source"] = fmt.Sprintf("%s:%d", errInfo.SourceFile, errInfo.SourceLine)
	}

	fields["first_seen"] = errInfo.FirstSeen.Format(time.RFC3339)
	if errInfo.Count > 1 {
		fields["last_seen"] = errInfo.LastSeen.Format(time.RFC3339)
	}

	priority := mapLogLevelToPriority(r.Level)

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "unknown"
	}

	return &notification_dto.SendParams{
		Context: notification_dto.NotificationContext{
			Source:      "logger",
			Environment: env,
			Priority:    priority,
			Timestamp:   errInfo.FirstSeen,
		},
		Content: notification_dto.NotificationContent{
			Type:    notification_dto.NotificationTypeRich,
			Title:   fmt.Sprintf("%s: Application Error", r.Level.String()),
			Message: r.Message,
			Fields:  fields,
		},
	}
}

// NewNotificationServiceAdapter creates an adapter that bridges the
// logger to the notification service.
//
// Takes service (notification_domain.Service) which is the notification
// service to wrap.
//
// Returns logger_domain.NotificationPort which can be used by the logger.
func NewNotificationServiceAdapter(service notification_domain.Service) logger_domain.NotificationPort {
	return &NotificationServiceAdapter{
		service: service,
	}
}

// mapLogLevelToPriority maps slog levels to notification priorities.
//
// Takes level (slog.Level) which is the log level to convert.
//
// Returns notification_dto.NotificationPriority which is the matching
// priority, ranging from PriorityLow for debug levels to PriorityCritical
// for error levels and above.
func mapLogLevelToPriority(level slog.Level) notification_dto.NotificationPriority {
	switch {
	case level >= slog.LevelError:
		return notification_dto.PriorityCritical
	case level >= slog.LevelWarn:
		return notification_dto.PriorityHigh
	case level >= slog.LevelInfo:
		return notification_dto.PriorityNormal
	default:
		return notification_dto.PriorityLow
	}
}
