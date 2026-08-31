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
	"strconv"
	"strings"
)

const (
	// configFieldCount sizes the config event's field map. Keep it equal to the number of
	// keys the add*Fields helpers write.
	configFieldCount = 31
)

// NewWatchdogConfigEvent reports the watchdog's effective configuration once, at Start.
//
// Takes status (*WatchdogStatusInfo) which is the resolved configuration snapshot.
//
// Returns WatchdogEvent which carries the effective configuration in Fields.
func NewWatchdogConfigEvent(status *WatchdogStatusInfo) WatchdogEvent {
	if status == nil {
		return WatchdogEvent{
			EventType: WatchdogEventConfig,
			Priority:  WatchdogPriorityNormal,
			Message:   "Watchdog configuration unavailable",
			Fields:    nil,
		}
	}

	fields := make(map[string]string, configFieldCount)
	addIdentityFields(fields, status)
	addThresholdFields(fields, status)
	addBudgetFields(fields, status)
	addProfilingFields(fields, status)

	return WatchdogEvent{
		EventType: WatchdogEventConfig,
		Priority:  WatchdogPriorityNormal,
		Message:   "Watchdog started with the reported configuration",
		Fields:    fields,
	}
}

// addIdentityFields records which process and build the configuration belongs to, so a
// sink receiving configs from several replicas can tell them apart.
//
// Takes fields (map[string]string) which is mutated in place.
// Takes status (*WatchdogStatusInfo) which is the resolved configuration snapshot.
func addIdentityFields(fields map[string]string, status *WatchdogStatusInfo) {
	fields["enabled"] = strconv.FormatBool(status.Enabled)
	fields["hostname"] = status.Hostname
	fields["version"] = status.Version
	fields["pid"] = strconv.Itoa(status.PID)
	fields["gomemlimit_bytes"] = strconv.FormatInt(status.GomemlimitBytes, 10)
	fields["profile_directory"] = status.ProfileDirectory
}

// addThresholdFields records the levels that trigger a capture or a warning.
//
// Takes fields (map[string]string) which is mutated in place.
// Takes status (*WatchdogStatusInfo) which is the resolved configuration snapshot.
func addThresholdFields(fields map[string]string, status *WatchdogStatusInfo) {
	fields["heap_threshold_bytes"] = strconv.FormatUint(status.HeapThresholdBytes, 10)
	fields["goroutine_threshold"] = strconv.Itoa(status.GoroutineThreshold)
	fields["goroutine_safety_ceiling"] = strconv.Itoa(status.GoroutineSafetyCeiling)
	fields["fd_pressure_threshold_percent"] = strconv.FormatFloat(
		status.FDPressureThresholdPercent, 'f', -1, 64)
	fields["scheduler_latency_p99_threshold"] = status.SchedulerLatencyP99Threshold.String()
	fields["crash_loop_threshold"] = strconv.Itoa(status.CrashLoopThreshold)
	fields["crash_loop_window"] = status.CrashLoopWindow.String()

	fields["gc_pressure_threshold"] = strconv.FormatFloat(status.GCPressureThreshold, 'f', -1, 64)
	fields["rss_threshold_percent"] = strconv.FormatFloat(status.RSSThresholdPercent, 'f', -1, 64)
	fields["trend_window_size"] = strconv.Itoa(status.TrendWindowSize)
	fields["trend_warning_horizon"] = status.TrendWarningHorizon.String()
}

// addBudgetFields records the rate limits, which decide how much of what the thresholds
// detect actually reaches the operator.
//
// Takes fields (map[string]string) which is mutated in place.
// Takes status (*WatchdogStatusInfo) which is the resolved configuration snapshot.
func addBudgetFields(fields map[string]string, status *WatchdogStatusInfo) {
	fields["check_interval"] = status.CheckInterval.String()
	fields["cooldown"] = status.Cooldown.String()
	fields["warm_up_duration"] = status.WarmUpDuration.String()
	fields["capture_window"] = status.CaptureWindow.String()
	fields["max_captures_per_window"] = strconv.Itoa(status.MaxCapturesPerWindow)
	fields["max_warnings_per_window"] = strconv.Itoa(status.MaxWarningsPerWindow)
	fields["max_profiles_per_type"] = strconv.Itoa(status.MaxProfilesPerType)
}

// addProfilingFields records the routine and on-demand profiling configuration.
//
// Takes fields (map[string]string) which is mutated in place.
// Takes status (*WatchdogStatusInfo) which is the resolved configuration snapshot.
func addProfilingFields(fields map[string]string, status *WatchdogStatusInfo) {
	fields["continuous_profiling_enabled"] = strconv.FormatBool(status.ContinuousProfilingEnabled)
	fields["continuous_profiling_interval"] = status.ContinuousProfilingInterval.String()
	fields["continuous_profiling_types"] = strings.Join(status.ContinuousProfilingTypes, ",")
	fields["continuous_profiling_retention"] = strconv.Itoa(status.ContinuousProfilingRetention)
	fields["contention_diagnostic_window"] = status.ContentionDiagnosticWindow.String()
	fields["contention_diagnostic_cooldown"] = status.ContentionDiagnosticCooldown.String()
	fields["contention_diagnostic_auto_fire"] = strconv.FormatBool(status.ContentionDiagnosticAutoFire)
}
