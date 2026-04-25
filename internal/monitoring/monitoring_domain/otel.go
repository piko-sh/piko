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

	// watchdogLoopIterationsCount tracks the total number of evaluation loop
	// iterations completed by the watchdog. Used as a self-heartbeat signal:
	// absence of recent increments indicates the loop has stopped.
	watchdogLoopIterationsCount metric.Int64Counter

	// watchdogLoopLastTickEpochSeconds records the unix-seconds timestamp of
	// the most recent evaluation tick. External monitoring can alert on
	// staleness (now - last_tick > 2 * CheckInterval).
	watchdogLoopLastTickEpochSeconds metric.Int64Gauge

	// watchdogLoopPanicCount tracks the total number of unrecovered panics
	// in the watchdog evaluation loop. Increments at most once per process
	// because the loop does not auto-restart -- combined with a stale
	// heartbeat this signals "watchdog has stopped working".
	watchdogLoopPanicCount metric.Int64Counter

	// watchdogFDPressureWarningCount tracks the total number of FD-pressure
	// warnings emitted when the open FD count approaches the soft
	// RLIMIT_NOFILE.
	watchdogFDPressureWarningCount metric.Int64Counter

	// watchdogFDCount records the current open FD count, sampled each tick
	// when the soft limit is known.
	watchdogFDCount metric.Int64Gauge

	// watchdogFDLimitSoft records the process soft FD limit; recorded once
	// at Start because the rlimit is stable for the process.
	watchdogFDLimitSoft metric.Int64Gauge

	// watchdogSchedulerLatencyEventCount tracks the total number of
	// scheduler-latency warnings emitted.
	watchdogSchedulerLatencyEventCount metric.Int64Counter

	// watchdogSchedulerLatencyP99Nanos records the latest scheduler latency
	// p99 in nanoseconds, sampled each tick from runtime/metrics.
	watchdogSchedulerLatencyP99Nanos metric.Int64Gauge

	// watchdogStartupHistoryReadErrorCount tracks the total number of
	// startup-history read failures during Start. Read errors are
	// best-effort; they do not block startup.
	watchdogStartupHistoryReadErrorCount metric.Int64Counter

	// watchdogStartupHistoryWriteErrorCount tracks the total number of
	// startup-history write failures during Start or Stop.
	watchdogStartupHistoryWriteErrorCount metric.Int64Counter

	// watchdogCrashLoopDetectionCount tracks the total number of crash-loop
	// detections at startup.
	watchdogCrashLoopDetectionCount metric.Int64Counter

	// watchdogUncleanShutdownCount tracks the total number of previous
	// process runs classified as unclean exits.
	watchdogUncleanShutdownCount metric.Int64Counter

	// watchdogRoutineProfileCaptureCount tracks the total number of
	// continuous-profiling routine captures completed.
	watchdogRoutineProfileCaptureCount metric.Int64Counter

	// watchdogContentionDiagnosticCount tracks the total number of
	// completed contention diagnostics.
	watchdogContentionDiagnosticCount metric.Int64Counter

	// watchdogContentionDiagnosticErrorCount tracks the total number of
	// contention diagnostics that failed to start (cooldown,
	// already-running, stopped, missing controller) or errored mid-run.
	watchdogContentionDiagnosticErrorCount metric.Int64Counter

	// watchdogEventEmittedCount tracks every WatchdogEvent the watchdog
	// emits, attributed by event_type so dashboards can break the rate
	// down per rule fired.
	watchdogEventEmittedCount metric.Int64Counter

	// watchdogEventSubscriberCount is the live count of streaming event
	// subscribers attached to the watchdog so operators can alert on drift
	// such as orphaned dashboards keeping channels open.
	watchdogEventSubscriberCount metric.Int64UpDownCounter

	// watchdogEventSubscriberDropCount tracks how many events were
	// silently dropped because a subscriber's bounded channel was full.
	// Sustained drops imply a slow or hung consumer.
	watchdogEventSubscriberDropCount metric.Int64Counter

	// watchdogSidecarDownloadCount tracks every successful sidecar fetch
	// served by the inspector RPC. Useful for distinguishing operator
	// activity from automated polling.
	watchdogSidecarDownloadCount metric.Int64Counter

	// watchdogSidecarDownloadErrorCount tracks failed sidecar fetches
	// (oversize, missing, read error) so dashboards can flag corrupt or
	// stale on-disk metadata.
	watchdogSidecarDownloadErrorCount metric.Int64Counter

	// watchdogProfileFileOversizeCount tracks how many sandboxed reads
	// (sidecar / history / profile) refused to load because the file
	// exceeded its configured cap. Sustained increases imply disk
	// corruption or a malicious tenant.
	watchdogProfileFileOversizeCount metric.Int64Counter
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

	watchdogLoopIterationsCount, err = watchdogMeter.Int64Counter(
		"watchdog.loop_iterations_count",
		metric.WithDescription("Number of evaluation loop iterations completed by the watchdog (self-heartbeat counter)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogLoopLastTickEpochSeconds, err = watchdogMeter.Int64Gauge(
		"watchdog.loop_last_tick_epoch_seconds",
		metric.WithDescription("Unix-seconds timestamp of the most recent watchdog evaluation tick"),
		metric.WithUnit("s"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogLoopPanicCount, err = watchdogMeter.Int64Counter(
		"watchdog.loop_panic_count",
		metric.WithDescription("Number of unrecovered panics in the watchdog evaluation loop"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogFDPressureWarningCount, err = watchdogMeter.Int64Counter(
		"watchdog.fd_pressure_warning_count",
		metric.WithDescription("Number of FD pressure warnings emitted when open file descriptor count approached the soft limit"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogFDCount, err = watchdogMeter.Int64Gauge(
		"watchdog.fd_count",
		metric.WithDescription("Current number of open file descriptors observed by the watchdog"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogFDLimitSoft, err = watchdogMeter.Int64Gauge(
		"watchdog.fd_limit_soft",
		metric.WithDescription("Soft RLIMIT_NOFILE for the process; stable for the process lifetime"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogSchedulerLatencyEventCount, err = watchdogMeter.Int64Counter(
		"watchdog.scheduler_latency_event_count",
		metric.WithDescription("Number of scheduler-latency warnings emitted from runtime/metrics observations"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogSchedulerLatencyP99Nanos, err = watchdogMeter.Int64Gauge(
		"watchdog.scheduler_latency_p99_nanos",
		metric.WithDescription("Latest scheduler-latency p99 in nanoseconds, sampled from runtime/metrics each watchdog tick"),
		metric.WithUnit("ns"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogStartupHistoryReadErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.startup_history_read_error_count",
		metric.WithDescription("Number of failures while reading the startup history file at watchdog Start"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogStartupHistoryWriteErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.startup_history_write_error_count",
		metric.WithDescription("Number of failures while writing the startup history file at watchdog Start or Stop"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogCrashLoopDetectionCount, err = watchdogMeter.Int64Counter(
		"watchdog.crash_loop_detection_count",
		metric.WithDescription("Number of crash-loop detections at watchdog Start"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogUncleanShutdownCount, err = watchdogMeter.Int64Counter(
		"watchdog.unclean_shutdown_count",
		metric.WithDescription("Number of previous process runs classified as unclean exits"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogRoutineProfileCaptureCount, err = watchdogMeter.Int64Counter(
		"watchdog.routine_profile_capture_count",
		metric.WithDescription("Number of routine profile captures completed by the continuous-profiling loop"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogContentionDiagnosticCount, err = watchdogMeter.Int64Counter(
		"watchdog.contention_diagnostic_count",
		metric.WithDescription("Number of completed contention diagnostics (block + mutex profile pairs)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogContentionDiagnosticErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.contention_diagnostic_error_count",
		metric.WithDescription("Number of contention diagnostics that failed to start or errored mid-run"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogEventEmittedCount, err = watchdogMeter.Int64Counter(
		"watchdog.event_emitted_count",
		metric.WithDescription("Number of watchdog events emitted, attributed by event_type"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogEventSubscriberCount, err = watchdogMeter.Int64UpDownCounter(
		"watchdog.event_subscriber_count",
		metric.WithDescription("Live count of streaming watchdog event subscribers"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogEventSubscriberDropCount, err = watchdogMeter.Int64Counter(
		"watchdog.event_subscriber_drop_count",
		metric.WithDescription("Number of events dropped from a subscriber channel because it was full"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogSidecarDownloadCount, err = watchdogMeter.Int64Counter(
		"watchdog.sidecar_download_count",
		metric.WithDescription("Number of sidecar JSON fetches served by the inspector"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogSidecarDownloadErrorCount, err = watchdogMeter.Int64Counter(
		"watchdog.sidecar_download_error_count",
		metric.WithDescription("Number of sidecar JSON fetches that failed (oversize / missing / read error)"),
	)
	if err != nil {
		otel.Handle(err)
	}

	watchdogProfileFileOversizeCount, err = watchdogMeter.Int64Counter(
		"watchdog.profile_file_oversize_count",
		metric.WithDescription("Number of sandboxed reads refused because the file exceeded its size cap"),
	)
	if err != nil {
		otel.Handle(err)
	}
}
