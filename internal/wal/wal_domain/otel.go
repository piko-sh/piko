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

package wal_domain

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/internal/logger/logger_domain"
)

var (
	// Log is the logger for the WAL domain.
	Log = logger_domain.GetLogger("piko/internal/wal/wal_domain")

	// Meter is the OpenTelemetry meter for WAL metrics.
	Meter = otel.Meter("piko/internal/wal/wal_domain")

	// AppendTotal counts the total number of WAL append operations.
	AppendTotal metric.Int64Counter

	// AppendDuration measures the duration of WAL append operations.
	AppendDuration metric.Float64Histogram

	// AppendBytesTotal counts the total bytes written to the WAL.
	AppendBytesTotal metric.Int64Counter

	// SyncTotal counts the total number of fsync operations.
	SyncTotal metric.Int64Counter

	// SyncDuration measures the duration of fsync operations.
	SyncDuration metric.Float64Histogram

	// RecoveryDuration measures the duration of WAL recovery operations.
	RecoveryDuration metric.Float64Histogram

	// RecoveryEntriesTotal counts the total entries recovered.
	RecoveryEntriesTotal metric.Int64Counter

	// TruncationsTotal counts the number of WAL truncations due to corruption.
	TruncationsTotal metric.Int64Counter

	// TruncatedBytesTotal counts the bytes lost due to truncation.
	TruncatedBytesTotal metric.Int64Counter

	// SnapshotSaveTotal counts the total number of snapshot save operations.
	SnapshotSaveTotal metric.Int64Counter

	// SnapshotSaveDuration measures the duration of snapshot save operations.
	SnapshotSaveDuration metric.Float64Histogram

	// SnapshotLoadDuration measures the duration of snapshot load operations.
	SnapshotLoadDuration metric.Float64Histogram

	// SnapshotSizeBytes records the size of the last snapshot in bytes.
	SnapshotSizeBytes metric.Int64Gauge

	// WALSizeBytes records the current size of the WAL file in bytes.
	WALSizeBytes metric.Int64Gauge

	// WALEntryCount records the number of entries in the WAL.
	WALEntryCount metric.Int64Gauge
)

func init() {
	var err error

	AppendTotal, err = Meter.Int64Counter(
		"piko.wal.append.total",
		metric.WithDescription("Total number of WAL append operations."),
	)
	if err != nil {
		otel.Handle(err)
	}

	AppendDuration, err = Meter.Float64Histogram(
		"piko.wal.append.duration",
		metric.WithDescription("Duration of WAL append operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	AppendBytesTotal, err = Meter.Int64Counter(
		"piko.wal.append.bytes.total",
		metric.WithDescription("Total bytes written to the WAL."),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SyncTotal, err = Meter.Int64Counter(
		"piko.wal.sync.total",
		metric.WithDescription("Total number of fsync operations."),
	)
	if err != nil {
		otel.Handle(err)
	}

	SyncDuration, err = Meter.Float64Histogram(
		"piko.wal.sync.duration",
		metric.WithDescription("Duration of fsync operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	RecoveryDuration, err = Meter.Float64Histogram(
		"piko.wal.recovery.duration",
		metric.WithDescription("Duration of WAL recovery operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	RecoveryEntriesTotal, err = Meter.Int64Counter(
		"piko.wal.recovery.entries.total",
		metric.WithDescription("Total entries recovered from WAL."),
	)
	if err != nil {
		otel.Handle(err)
	}

	TruncationsTotal, err = Meter.Int64Counter(
		"piko.wal.truncations.total",
		metric.WithDescription("Number of WAL truncations due to corruption."),
	)
	if err != nil {
		otel.Handle(err)
	}

	TruncatedBytesTotal, err = Meter.Int64Counter(
		"piko.wal.truncated.bytes.total",
		metric.WithDescription("Bytes lost due to WAL truncation."),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SnapshotSaveTotal, err = Meter.Int64Counter(
		"piko.wal.snapshot.save.total",
		metric.WithDescription("Total number of snapshot save operations."),
	)
	if err != nil {
		otel.Handle(err)
	}

	SnapshotSaveDuration, err = Meter.Float64Histogram(
		"piko.wal.snapshot.save.duration",
		metric.WithDescription("Duration of snapshot save operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SnapshotLoadDuration, err = Meter.Float64Histogram(
		"piko.wal.snapshot.load.duration",
		metric.WithDescription("Duration of snapshot load operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		otel.Handle(err)
	}

	SnapshotSizeBytes, err = Meter.Int64Gauge(
		"piko.wal.snapshot.size.bytes",
		metric.WithDescription("Size of the last snapshot in bytes."),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WALSizeBytes, err = Meter.Int64Gauge(
		"piko.wal.size.bytes",
		metric.WithDescription("Current size of the WAL file in bytes."),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	WALEntryCount, err = Meter.Int64Gauge(
		"piko.wal.entry.count",
		metric.WithDescription("Number of entries currently in the WAL."),
	)
	if err != nil {
		otel.Handle(err)
	}
}
