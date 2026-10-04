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

package monitoring_domain

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// GetStartupHistory returns the most recent startup-history entries in chronological
// order (oldest first). A missing or empty history file, or a watchdog without a profile
// store, yields a nil slice.
//
// Returns []WatchdogStartupHistoryEntry which is the parsed history.
// Returns error when an existing file cannot be read or parsed; an oversized file is also
// counted in OTel. A missing file is not an error.
func (w *Watchdog) GetStartupHistory(ctx context.Context) ([]WatchdogStartupHistoryEntry, error) {
	if w.profileStore == nil {
		return nil, nil
	}
	file, _, err := w.profileStore.readHistory()
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		if errors.Is(err, safedisk.ErrFileExceedsLimit) {
			watchdogProfileFileOversizeCount.Add(ctx, 1)
		}
		return nil, fmt.Errorf("reading watchdog startup history: %w", err)
	}
	if len(file.Entries) == 0 {
		return nil, nil
	}
	result := make([]WatchdogStartupHistoryEntry, len(file.Entries))
	for index, entry := range file.Entries {
		result[index] = WatchdogStartupHistoryEntry(entry)
	}
	return result, nil
}

// processStartupHistory drives the startup-history flow at watchdog Start.
//
// It inspects the on-disk ring, classifies the previous run (clean or unclean), detects
// crash loops, then appends a new entry for the current process. Failures are logged and
// counted but never block startup, since the watchdog must come up even when the profile
// directory is unwritable. Set CrashLoopWindow to zero to disable history processing
// entirely.
func (w *Watchdog) processStartupHistory(ctx context.Context) {
	if w.config.CrashLoopWindow == 0 {
		return
	}

	ctx, l := logger_domain.From(ctx, log)

	file, existed, err := w.profileStore.readHistory()
	if err != nil && existed {
		watchdogStartupHistoryReadErrorCount.Add(ctx, 1)
		l.Warn("Failed to read startup history; continuing with empty history",
			logger_domain.Error(err),
		)
	}

	w.classifyPreviousEntry(ctx, &file)
	w.detectCrashLoop(ctx, file)

	now := w.clock.Now()
	entry := startupHistoryEntry{
		StartedAt:       now,
		PID:             os.Getpid(),
		Hostname:        w.hostname,
		Version:         buildVersionString(),
		GomemlimitBytes: w.gomemlimit,
		StoppedAt:       time.Time{},
		Reason:          "",
	}
	file.Entries = append(file.Entries, entry)
	if len(file.Entries) > maxStartupHistoryEntries {
		file.Entries = file.Entries[len(file.Entries)-maxStartupHistoryEntries:]
	}

	if writeErr := w.profileStore.writeHistory(file); writeErr != nil {
		watchdogStartupHistoryWriteErrorCount.Add(ctx, 1)
		l.Warn("Failed to write startup history", logger_domain.Error(writeErr))
	}
}

// classifyPreviousEntry checks whether the most recent history entry exited uncleanly (no
// StoppedAt). When it did, the entry is patched with Reason="unclean" and a
// PreviousCrashClassified event is emitted so the operator can investigate.
//
// Takes file (*startupHistoryFile) which is mutated in place when the last entry needs
// reclassification.
func (w *Watchdog) classifyPreviousEntry(ctx context.Context, file *startupHistoryFile) {
	if len(file.Entries) == 0 {
		return
	}
	last := &file.Entries[len(file.Entries)-1]
	if !last.StoppedAt.IsZero() {
		return
	}

	last.Reason = "unclean"
	watchdogUncleanShutdownCount.Add(ctx, 1)

	_, l := logger_domain.From(ctx, log)
	l.Notice("Previous run did not exit cleanly; classifying as unclean shutdown",
		logger_domain.Int("prev_pid", last.PID),
	)

	w.sendNotification(ctx, NewPreviousCrashClassifiedEvent(*last))
}

// detectCrashLoop counts unclean entries within the CrashLoopWindow.
//
// Entries whose StoppedAt is zero count as unclean. When the count reaches
// CrashLoopThreshold a CrashLoopDetected event is emitted.
//
// Takes file (startupHistoryFile) which is the history snapshot to scan.
func (w *Watchdog) detectCrashLoop(ctx context.Context, file startupHistoryFile) {
	if w.config.CrashLoopThreshold <= 0 {
		return
	}

	cutoff := w.clock.Now().Add(-w.config.CrashLoopWindow)
	unclean := 0
	for _, entry := range file.Entries {
		if !entry.StartedAt.After(cutoff) {
			continue
		}
		if !entry.StoppedAt.IsZero() {
			continue
		}
		unclean++
	}

	if unclean < w.config.CrashLoopThreshold {
		return
	}

	watchdogCrashLoopDetectionCount.Add(ctx, 1)

	_, l := logger_domain.From(ctx, log)
	l.Error("Crash loop detected from startup history",
		logger_domain.Int("unclean_in_window", unclean),
		logger_domain.Int("window_seconds", int(w.config.CrashLoopWindow.Seconds())),
	)

	w.sendNotification(ctx, NewCrashLoopDetectedEvent(unclean, int(w.config.CrashLoopWindow.Seconds())))
}

// markStartupHistoryStopped patches the most recent entry's StoppedAt field so the next
// start can classify the previous run as clean.
//
// Called from Stop. Failure is logged but never propagated.
func (w *Watchdog) markStartupHistoryStopped(ctx context.Context) {
	if w.config.CrashLoopWindow == 0 {
		return
	}

	file, _, err := w.profileStore.readHistory()
	if err != nil {
		watchdogStartupHistoryReadErrorCount.Add(ctx, 1)
		return
	}
	if len(file.Entries) == 0 {
		return
	}

	file.Entries[len(file.Entries)-1].StoppedAt = w.clock.Now()
	if file.Entries[len(file.Entries)-1].Reason == "" {
		file.Entries[len(file.Entries)-1].Reason = "clean"
	}

	if writeErr := w.profileStore.writeHistory(file); writeErr != nil {
		watchdogStartupHistoryWriteErrorCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Warn("Failed to mark startup history as cleanly stopped", logger_domain.Error(writeErr))
	}
}
