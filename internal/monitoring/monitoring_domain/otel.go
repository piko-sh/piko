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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"piko.sh/piko/wdk/logger"
)

var (
	// log is the package-level logger for the monitoring_domain package.
	log = logger.GetLogger("piko/internal/monitoring/monitoring_domain")

	// String is a logger field constructor for string values.
	String = logger.String

	// Int is a convenience alias for logger.Int to create integer log fields.
	Int = logger.Int

	// Int64 is an alias for logger.Int64 for logging int64 values.
	Int64 = logger.Int64

	// Error is the package-level error logging function.
	Error = logger.Error
)

var (
	// watchdogMeter is the OpenTelemetry meter for watchdog diagnostic metrics.
	watchdogMeter = otel.Meter("piko/internal/monitoring/monitoring_domain/watchdog")

	// watchdogHeapCaptureCount tracks the total number of heap profile captures
	// triggered by the watchdog when heap usage exceeds the configured threshold.
	watchdogHeapCaptureCount metric.Int64Counter

	// watchdogGoroutineCaptureCount tracks the total number of goroutine profile
	// captures triggered when the goroutine count exceeds the configured threshold.
	watchdogGoroutineCaptureCount metric.Int64Counter

	// watchdogCaptureErrorCount tracks the total number of profile capture
	// failures encountered by the watchdog.
	watchdogCaptureErrorCount metric.Int64Counter

	// watchdogGCPressureWarningCount tracks the total number of GC pressure
	// warnings emitted when GCCPUFraction exceeds the configured threshold.
	watchdogGCPressureWarningCount metric.Int64Counter

	// watchdogCooldownSkipCount tracks the total number of capture attempts
	// that were skipped because the cooldown period had not yet elapsed.
	watchdogCooldownSkipCount metric.Int64Counter

	// watchdogHeapHighWaterResetCount tracks the total number of times the heap
	// high-water mark was reset back to the initial threshold after sustained
	// low memory usage.
	watchdogHeapHighWaterResetCount metric.Int64Counter

	// watchdogHeapHighWaterBytes records the current heap high-water mark in
	// bytes, representing the threshold that must be exceeded before a new
	// heap profile capture is triggered.
	watchdogHeapHighWaterBytes metric.Int64Gauge

	// watchdogGoroutineLeakDetectionCount tracks the total number of goroutine
	// leak detections via the Go 1.26 goroutine leak profile.
	watchdogGoroutineLeakDetectionCount metric.Int64Counter

	// watchdogPreDeathSnapshotCount tracks the total number of pre-death
	// diagnostic snapshots captured during shutdown.
	watchdogPreDeathSnapshotCount metric.Int64Counter

	// watchdogHeapTrendWarningCount tracks the total number of heap trend
	// warnings emitted when projected growth breaches the warning horizon.
	watchdogHeapTrendWarningCount metric.Int64Counter

	// watchdogHeapGrowthRateBytesPerSecond records the current heap growth
	// rate in bytes per second from the linear regression.
	watchdogHeapGrowthRateBytesPerSecond metric.Int64Gauge

	// watchdogRSSBytes records the current process RSS in bytes.
	watchdogRSSBytes metric.Int64Gauge

	// watchdogCgroupMemoryLimitBytes records the cgroup memory limit in bytes.
	watchdogCgroupMemoryLimitBytes metric.Int64Gauge

	// watchdogRSSCaptureCount tracks the total number of profile captures
	// triggered by RSS approaching the cgroup memory limit.
	watchdogRSSCaptureCount metric.Int64Counter

	// watchdogNotificationSentCount tracks the total number of watchdog event
	// notifications successfully delivered to external systems.
	watchdogNotificationSentCount metric.Int64Counter

	// watchdogNotificationErrorCount tracks the total number of watchdog
	// notification delivery failures.
	watchdogNotificationErrorCount metric.Int64Counter

	// watchdogProfileUploadCount tracks the total number of profiles
	// successfully uploaded to remote storage.
	watchdogProfileUploadCount metric.Int64Counter

	// watchdogProfileUploadErrorCount tracks the total number of profile
	// upload failures.
	watchdogProfileUploadErrorCount metric.Int64Counter
)

func init() {
	var err error

	watchdogHeapCaptureCount, err = watchdogMeter.Int64Counter(
		"watchdog.heap_capture_count",
		metric.WithDescription("Number of heap profile captures triggered by the watchdog"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogGoroutineCaptureCount, err = watchdogMeter.Int64Counter(
		"watchdog.goroutine_capture_count",
		metric.WithDescription("Number of goroutine profile captures triggered by the watchdog"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogCaptureErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.capture_error_count",
		metric.WithDescription("Number of profile capture failures in the watchdog"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogGCPressureWarningCount, err = watchdogMeter.Int64Counter(
		"watchdog.gc_pressure_warning_count",
		metric.WithDescription("Number of GC pressure warnings emitted by the watchdog"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogCooldownSkipCount, err = watchdogMeter.Int64Counter(
		"watchdog.cooldown_skip_count",
		metric.WithDescription("Number of capture attempts skipped due to cooldown"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogHeapHighWaterResetCount, err = watchdogMeter.Int64Counter(
		"watchdog.heap_high_water_reset_count",
		metric.WithDescription("Number of heap high-water mark resets to initial threshold"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogHeapHighWaterBytes, err = watchdogMeter.Int64Gauge(
		"watchdog.heap_high_water_bytes",
		metric.WithDescription("Current heap high-water mark threshold in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogGoroutineLeakDetectionCount, err = watchdogMeter.Int64Counter(
		"watchdog.goroutine_leak_detection_count",
		metric.WithDescription("Number of goroutine leak detections via the Go 1.26 goroutine leak profile"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogPreDeathSnapshotCount, err = watchdogMeter.Int64Counter(
		"watchdog.pre_death_snapshot_count",
		metric.WithDescription("Number of pre-death diagnostic snapshots captured during shutdown"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogHeapTrendWarningCount, err = watchdogMeter.Int64Counter(
		"watchdog.heap_trend_warning_count",
		metric.WithDescription("Number of heap trend warnings projected to breach the memory limit"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogHeapGrowthRateBytesPerSecond, err = watchdogMeter.Int64Gauge(
		"watchdog.heap_growth_rate_bytes_per_second",
		metric.WithDescription("Current heap growth rate from linear regression in bytes per second"),
		metric.WithUnit("By/s"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogRSSBytes, err = watchdogMeter.Int64Gauge(
		"watchdog.rss_bytes",
		metric.WithDescription("Current process resident set size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogCgroupMemoryLimitBytes, err = watchdogMeter.Int64Gauge(
		"watchdog.cgroup_memory_limit_bytes",
		metric.WithDescription("Container cgroup memory limit in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogRSSCaptureCount, err = watchdogMeter.Int64Counter(
		"watchdog.rss_capture_count",
		metric.WithDescription("Number of profile captures triggered by RSS approaching the cgroup memory limit"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogNotificationSentCount, err = watchdogMeter.Int64Counter(
		"watchdog.notification_sent_count",
		metric.WithDescription("Number of watchdog event notifications successfully delivered"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogNotificationErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.notification_error_count",
		metric.WithDescription("Number of watchdog notification delivery failures"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogProfileUploadCount, err = watchdogMeter.Int64Counter(
		"watchdog.profile_upload_count",
		metric.WithDescription("Number of profiles successfully uploaded to remote storage"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogProfileUploadErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.profile_upload_error_count",
		metric.WithDescription("Number of profile upload failures"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
