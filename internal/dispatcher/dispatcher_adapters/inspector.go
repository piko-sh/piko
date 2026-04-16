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

package dispatcher_adapters

import (
	"context"
	"errors"
	"fmt"

	"piko.sh/piko/internal/dispatcher/dispatcher_domain"
	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/notification/notification_domain"
	"piko.sh/piko/wdk/logger"
)

const (
	// dispatcherTypeEmail identifies the email dispatcher
	// type for routing and metrics.
	dispatcherTypeEmail = "email"

	// dispatcherTypeNotification identifies the notification
	// dispatcher type for routing and metrics.
	dispatcherTypeNotification = "notification"
)

var (
	// log is the package-level logger for the dispatcher_adapters package.
	log = logger.GetLogger("piko/internal/dispatcher/dispatcher_adapters")

	// errEmailDispatcherNotConfigured is returned when an email dispatcher
	// operation is attempted but no email dispatcher has been configured.
	errEmailDispatcherNotConfigured = errors.New("email dispatcher not configured")

	// errNotificationDispatcherNotConfigured is returned when a notification
	// dispatcher operation is attempted but no notification dispatcher has been
	// configured.
	errNotificationDispatcherNotConfigured = errors.New("notification dispatcher not configured")
)

// Inspector implements dispatcher_domain.DispatcherInspector by delegating to
// email and notification dispatcher ports.
type Inspector struct {
	// emailDispatcher provides access to the email dispatcher; may be nil.
	emailDispatcher email_domain.EmailDispatcherPort

	// notificationDispatcher provides access to the notification dispatcher;
	// may be nil if not configured.
	notificationDispatcher notification_domain.NotificationDispatcherPort
}

// NewInspector creates a new Inspector with the given dispatchers.
// Either dispatcher may be nil if not configured.
//
// Takes emailDispatcher (EmailDispatcherPort) which may be nil.
// Takes notificationDispatcher (NotificationDispatcherPort) which may be nil.
//
// Returns *Inspector ready for use.
func NewInspector(
	emailDispatcher email_domain.EmailDispatcherPort,
	notificationDispatcher notification_domain.NotificationDispatcherPort,
) *Inspector {
	return &Inspector{
		emailDispatcher:        emailDispatcher,
		notificationDispatcher: notificationDispatcher,
	}
}

// GetDispatcherSummaries returns statistics for all configured dispatchers.
//
// Returns []dispatcher_domain.DispatcherSummary which contains statistics for
// each available dispatcher.
// Returns error when retrieval fails.
func (i *Inspector) GetDispatcherSummaries(ctx context.Context) ([]dispatcher_domain.DispatcherSummary, error) {
	ctx, l := logger_domain.From(ctx, log)
	var summaries []dispatcher_domain.DispatcherSummary

	if i.emailDispatcher != nil {
		summary, err := i.getEmailSummary(ctx)
		if err != nil {
			l.Warn("Failed to get email dispatcher summary", logger.Error(err))
		} else {
			summaries = append(summaries, summary)
		}
	}

	if i.notificationDispatcher != nil {
		summary, err := i.getNotificationSummary(ctx)
		if err != nil {
			l.Warn("Failed to get notification dispatcher summary", logger.Error(err))
		} else {
			summaries = append(summaries, summary)
		}
	}

	return summaries, nil
}

// GetDLQEntries returns dead letter queue entries for a specific dispatcher
// type.
//
// Takes dispatcherType (string) which specifies the dispatcher to query, such
// as "email" or "notification".
// Takes limit (int) which sets the maximum number of entries to return.
//
// Returns []dispatcher_domain.DLQEntry which contains the dead letter queue
// entries for the specified dispatcher.
// Returns error when the dispatcher type is unknown.
func (i *Inspector) GetDLQEntries(ctx context.Context, dispatcherType string, limit int) ([]dispatcher_domain.DLQEntry, error) {
	switch dispatcherType {
	case dispatcherTypeEmail:
		return i.getEmailDLQEntries(ctx, limit)
	case dispatcherTypeNotification:
		return i.getNotificationDLQEntries(ctx, limit)
	default:
		return nil, fmt.Errorf("unknown dispatcher type: %s", dispatcherType)
	}
}

// GetDLQCount returns the number of entries in a dispatcher's dead letter queue.
//
// Takes dispatcherType (string) which specifies which dispatcher to query.
//
// Returns int which is the count of entries in the dead letter queue.
// Returns error when the dispatcher type is unknown or not configured.
func (i *Inspector) GetDLQCount(ctx context.Context, dispatcherType string) (int, error) {
	switch dispatcherType {
	case dispatcherTypeEmail:
		if i.emailDispatcher == nil {
			return 0, errEmailDispatcherNotConfigured
		}
		return i.emailDispatcher.GetDeadLetterCount(ctx)
	case dispatcherTypeNotification:
		if i.notificationDispatcher == nil {
			return 0, errNotificationDispatcherNotConfigured
		}
		return i.notificationDispatcher.GetDeadLetterCount(ctx)
	default:
		return 0, fmt.Errorf("unknown dispatcher type: %s", dispatcherType)
	}
}

// ClearDLQ removes all entries from a dispatcher's dead letter queue.
//
// Takes dispatcherType (string) which specifies the dispatcher to clear
// ("email" or "notification").
//
// Returns error when the dispatcher type is unknown or not configured.
func (i *Inspector) ClearDLQ(ctx context.Context, dispatcherType string) error {
	switch dispatcherType {
	case dispatcherTypeEmail:
		if i.emailDispatcher == nil {
			return errEmailDispatcherNotConfigured
		}
		return i.emailDispatcher.ClearDeadLetterQueue(ctx)
	case dispatcherTypeNotification:
		if i.notificationDispatcher == nil {
			return errNotificationDispatcherNotConfigured
		}
		return i.notificationDispatcher.ClearDeadLetterQueue(ctx)
	default:
		return fmt.Errorf("unknown dispatcher type: %s", dispatcherType)
	}
}

// getEmailSummary fetches statistics from the email dispatcher.
//
// Returns dispatcher_domain.DispatcherSummary which contains the current
// processing statistics for the email dispatcher.
// Returns error when the processing statistics cannot be retrieved.
func (i *Inspector) getEmailSummary(ctx context.Context) (dispatcher_domain.DispatcherSummary, error) {
	stats, err := i.emailDispatcher.GetProcessingStats(ctx)
	if err != nil {
		return dispatcher_domain.DispatcherSummary{}, fmt.Errorf("getting email processing stats: %w", err)
	}

	return dispatcher_domain.DispatcherSummary{
		Type:            dispatcherTypeEmail,
		QueuedItems:     stats.QueuedEmails,
		RetryQueueSize:  stats.RetryQueueSize,
		DeadLetterCount: stats.DeadLetterCount,
		TotalProcessed:  stats.TotalProcessed,
		TotalSuccessful: stats.TotalSuccessful,
		TotalFailed:     stats.TotalFailed,
		TotalRetries:    stats.TotalRetries,
		Uptime:          stats.Uptime,
	}, nil
}

// getNotificationSummary fetches statistics from the notification dispatcher.
//
// Returns dispatcher_domain.DispatcherSummary which contains the collected
// notification processing statistics.
// Returns error when fetching the processing stats fails.
func (i *Inspector) getNotificationSummary(ctx context.Context) (dispatcher_domain.DispatcherSummary, error) {
	stats, err := i.notificationDispatcher.GetProcessingStats(ctx)
	if err != nil {
		return dispatcher_domain.DispatcherSummary{}, fmt.Errorf("getting notification processing stats: %w", err)
	}

	return dispatcher_domain.DispatcherSummary{
		Type:            dispatcherTypeNotification,
		QueuedItems:     stats.QueuedNotifications,
		RetryQueueSize:  stats.RetryQueueSize,
		DeadLetterCount: stats.DeadLetterCount,
		TotalProcessed:  stats.TotalProcessed,
		TotalSuccessful: stats.TotalSuccessful,
		TotalFailed:     stats.TotalFailed,
		TotalRetries:    stats.TotalRetries,
		Uptime:          stats.Uptime,
	}, nil
}

// getEmailDLQEntries returns dead letter entries from the email dispatcher.
//
// Takes limit (int) which specifies the maximum number of entries to return.
//
// Returns []dispatcher_domain.DLQEntry which contains the dead letter entries.
// Returns error when the email dispatcher is not configured or retrieval fails.
func (i *Inspector) getEmailDLQEntries(ctx context.Context, limit int) ([]dispatcher_domain.DLQEntry, error) {
	if i.emailDispatcher == nil {
		return nil, errEmailDispatcherNotConfigured
	}

	dlq := i.emailDispatcher.GetDeadLetterQueue()
	if dlq == nil {
		return nil, nil
	}

	entries, err := dlq.Get(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("getting email DLQ entries: %w", err)
	}

	result := make([]dispatcher_domain.DLQEntry, len(entries))
	for index, e := range entries {
		result[index] = dispatcher_domain.DLQEntry{
			ID:            e.ID,
			Type:          dispatcherTypeEmail,
			OriginalError: e.OriginalError,
			TotalAttempts: e.TotalAttempts,
			FirstAttempt:  e.FirstAttempt,
			LastAttempt:   e.LastAttempt,
			AddedAt:       e.AddedToDeadLetter,
		}
	}
	return result, nil
}

// getNotificationDLQEntries returns dead letter entries from the notification
// dispatcher.
//
// Takes limit (int) which specifies the maximum number of entries to return.
//
// Returns []dispatcher_domain.DLQEntry which contains the dead letter entries.
// Returns error when the notification dispatcher is not configured or when
// fetching entries fails.
func (i *Inspector) getNotificationDLQEntries(ctx context.Context, limit int) ([]dispatcher_domain.DLQEntry, error) {
	if i.notificationDispatcher == nil {
		return nil, errNotificationDispatcherNotConfigured
	}

	dlq := i.notificationDispatcher.GetDeadLetterQueue()
	if dlq == nil {
		return nil, nil
	}

	entries, err := dlq.Get(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("getting notification DLQ entries: %w", err)
	}

	result := make([]dispatcher_domain.DLQEntry, len(entries))
	for index, e := range entries {
		result[index] = dispatcher_domain.DLQEntry{
			Type:          dispatcherTypeNotification,
			OriginalError: e.OriginalError,
			TotalAttempts: e.TotalAttempts,
			FirstAttempt:  e.FirstAttempt,
			LastAttempt:   e.LastAttempt,
		}
	}
	return result, nil
}

var _ dispatcher_domain.DispatcherInspector = (*Inspector)(nil)
