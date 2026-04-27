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

import "fmt"

// validateWatchdogConfig checks the configuration for invalid or dangerous
// values and returns an error describing the first problem found.
//
// Takes config (WatchdogConfig) which provides the configuration to validate.
//
// Returns error when the configuration contains invalid or dangerous values.
func validateWatchdogConfig(config *WatchdogConfig) error {
	if err := validateWatchdogThresholds(config); err != nil {
		return err
	}

	return validateWatchdogTimings(config)
}

// validateWatchdogThresholds validates threshold and limit fields in the
// watchdog configuration. The function delegates per-section validation to
// keep each helper short and focused.
//
// Takes config (*WatchdogConfig) which provides the configuration to validate.
//
// Returns error when a threshold or limit field contains an invalid value.
func validateWatchdogThresholds(config *WatchdogConfig) error {
	if err := validateHeapAndRSSThresholds(config); err != nil {
		return err
	}
	if err := validateBudgetAndCountThresholds(config); err != nil {
		return err
	}
	if err := validateForensicThresholds(config); err != nil {
		return err
	}
	if err := validateContinuousProfiling(config); err != nil {
		return err
	}
	return validateContentionDiagnostic(config)
}

// validateHeapAndRSSThresholds validates the heap and RSS percent thresholds
// and the goroutine threshold/ceiling pairing.
//
// Takes config (*WatchdogConfig) which holds the candidate configuration.
//
// Returns error wrapping ErrInvalidWatchdogConfig when any threshold is
// out of range or the goroutine ceiling does not exceed the threshold.
func validateHeapAndRSSThresholds(config *WatchdogConfig) error {
	if config.HeapThresholdPercent <= 0 || config.HeapThresholdPercent > 1.0 {
		return fmt.Errorf("%w: HeapThresholdPercent must be in (0.0, 1.0], got %v", ErrInvalidWatchdogConfig, config.HeapThresholdPercent)
	}
	if config.RSSThresholdPercent < 0 || config.RSSThresholdPercent > 1.0 {
		return fmt.Errorf("%w: RSSThresholdPercent must be in [0.0, 1.0], got %v", ErrInvalidWatchdogConfig, config.RSSThresholdPercent)
	}
	if config.GoroutineThreshold < 1 {
		return fmt.Errorf("%w: GoroutineThreshold must be at least 1, got %d", ErrInvalidWatchdogConfig, config.GoroutineThreshold)
	}
	if config.GoroutineSafetyCeiling <= config.GoroutineThreshold {
		return fmt.Errorf("%w: GoroutineSafetyCeiling (%d) must be greater than GoroutineThreshold (%d)",
			ErrInvalidWatchdogConfig, config.GoroutineSafetyCeiling, config.GoroutineThreshold)
	}
	if config.TrendWindowSize < 0 {
		return fmt.Errorf("%w: TrendWindowSize must be non-negative, got %d", ErrInvalidWatchdogConfig, config.TrendWindowSize)
	}
	if config.MaxProfileSizeBytes <= 0 {
		return fmt.Errorf("%w: MaxProfileSizeBytes must be positive, got %d", ErrInvalidWatchdogConfig, config.MaxProfileSizeBytes)
	}
	return nil
}

// validateBudgetAndCountThresholds validates the rate-limit budgets and
// per-type retention caps.
//
// Takes config (*WatchdogConfig) which holds the candidate configuration.
//
// Returns error wrapping ErrInvalidWatchdogConfig when any budget or
// retention cap is below 1.
func validateBudgetAndCountThresholds(config *WatchdogConfig) error {
	if config.MaxProfilesPerType < 1 {
		return fmt.Errorf("%w: MaxProfilesPerType must be at least 1, got %d", ErrInvalidWatchdogConfig, config.MaxProfilesPerType)
	}
	if config.MaxCapturesPerWindow < 1 {
		return fmt.Errorf("%w: MaxCapturesPerWindow must be at least 1, got %d", ErrInvalidWatchdogConfig, config.MaxCapturesPerWindow)
	}
	if config.MaxWarningsPerWindow < 1 {
		return fmt.Errorf("%w: MaxWarningsPerWindow must be at least 1, got %d", ErrInvalidWatchdogConfig, config.MaxWarningsPerWindow)
	}
	return nil
}

// validateForensicThresholds validates the FD pressure, scheduler latency,
// and crash-loop thresholds.
//
// Takes config (*WatchdogConfig) which holds the candidate configuration.
//
// Returns error wrapping ErrInvalidWatchdogConfig when any threshold is
// out of range; zero is permitted to disable a rule.
func validateForensicThresholds(config *WatchdogConfig) error {
	if config.FDPressureThresholdPercent < 0 || config.FDPressureThresholdPercent > 1.0 {
		return fmt.Errorf("%w: FDPressureThresholdPercent must be in [0.0, 1.0], got %v", ErrInvalidWatchdogConfig, config.FDPressureThresholdPercent)
	}
	if config.SchedulerLatencyP99Threshold < 0 {
		return fmt.Errorf("%w: SchedulerLatencyP99Threshold must be non-negative, got %v", ErrInvalidWatchdogConfig, config.SchedulerLatencyP99Threshold)
	}
	if config.CrashLoopWindow < 0 {
		return fmt.Errorf("%w: CrashLoopWindow must be non-negative, got %v", ErrInvalidWatchdogConfig, config.CrashLoopWindow)
	}
	if config.CrashLoopThreshold < 0 {
		return fmt.Errorf("%w: CrashLoopThreshold must be non-negative, got %d", ErrInvalidWatchdogConfig, config.CrashLoopThreshold)
	}
	return nil
}

// validateContinuousProfiling validates the continuous-profiling fields when
// the feature is enabled. When disabled the fields are ignored.
//
// Takes config (*WatchdogConfig) which holds the candidate configuration.
//
// Returns error wrapping ErrInvalidWatchdogConfig when the interval,
// retention, or types are unsupported.
func validateContinuousProfiling(config *WatchdogConfig) error {
	if !config.ContinuousProfilingEnabled {
		return nil
	}
	if config.ContinuousProfilingInterval < minContinuousProfilingInterval {
		return fmt.Errorf("%w: ContinuousProfilingInterval must be at least %v, got %v",
			ErrInvalidWatchdogConfig, minContinuousProfilingInterval, config.ContinuousProfilingInterval)
	}
	if config.ContinuousProfilingRetention < 1 || config.ContinuousProfilingRetention > maxContinuousProfilingRetention {
		return fmt.Errorf("%w: ContinuousProfilingRetention must be in [1, %d], got %d",
			ErrInvalidWatchdogConfig, maxContinuousProfilingRetention, config.ContinuousProfilingRetention)
	}
	for _, profileType := range config.ContinuousProfilingTypes {
		switch profileType {
		case profileTypeHeap, profileTypeGoroutine, "allocs":

		default:
			return fmt.Errorf("%w: ContinuousProfilingTypes contains unsupported type %q (allowed: heap, goroutine, allocs)",
				ErrInvalidWatchdogConfig, profileType)
		}
	}
	return nil
}

// validateContentionDiagnostic validates the contention-diagnostic
// configuration. These fields are always validated because the diagnostic
// can be invoked manually even when AutoFire is disabled.
//
// Takes config (*WatchdogConfig) which holds the candidate configuration.
//
// Returns error wrapping ErrInvalidWatchdogConfig when any field is out
// of range.
func validateContentionDiagnostic(config *WatchdogConfig) error {
	if config.ContentionDiagnosticWindowDuration < minContentionDiagnosticWindow ||
		config.ContentionDiagnosticWindowDuration > maxContentionDiagnosticWindow {
		return fmt.Errorf("%w: ContentionDiagnosticWindowDuration must be in [%v, %v], got %v",
			ErrInvalidWatchdogConfig, minContentionDiagnosticWindow, maxContentionDiagnosticWindow, config.ContentionDiagnosticWindowDuration)
	}
	if config.ContentionDiagnosticBlockProfileRate < 0 {
		return fmt.Errorf("%w: ContentionDiagnosticBlockProfileRate must be non-negative, got %d",
			ErrInvalidWatchdogConfig, config.ContentionDiagnosticBlockProfileRate)
	}
	if config.ContentionDiagnosticMutexProfileFraction < 0 {
		return fmt.Errorf("%w: ContentionDiagnosticMutexProfileFraction must be non-negative, got %d",
			ErrInvalidWatchdogConfig, config.ContentionDiagnosticMutexProfileFraction)
	}
	if config.ContentionDiagnosticConsecutiveTrigger < 1 {
		return fmt.Errorf("%w: ContentionDiagnosticConsecutiveTrigger must be at least 1, got %d",
			ErrInvalidWatchdogConfig, config.ContentionDiagnosticConsecutiveTrigger)
	}
	if config.ContentionDiagnosticTriggerWindow <= 0 {
		return fmt.Errorf("%w: ContentionDiagnosticTriggerWindow must be positive, got %v",
			ErrInvalidWatchdogConfig, config.ContentionDiagnosticTriggerWindow)
	}
	if config.ContentionDiagnosticCooldown <= 0 {
		return fmt.Errorf("%w: ContentionDiagnosticCooldown must be positive, got %v",
			ErrInvalidWatchdogConfig, config.ContentionDiagnosticCooldown)
	}
	return nil
}

// validateWatchdogTimings validates interval and duration fields in the
// watchdog configuration.
//
// The helper signature uses *WatchdogConfig to avoid passing the large
// config struct by value.
//
// Takes config (*WatchdogConfig) which provides the configuration to
// validate.
//
// Returns error wrapping ErrInvalidWatchdogConfig when an interval or
// duration field contains an invalid value.
func validateWatchdogTimings(config *WatchdogConfig) error {
	if config.CheckInterval <= 0 {
		return fmt.Errorf("%w: CheckInterval must be positive, got %v", ErrInvalidWatchdogConfig, config.CheckInterval)
	}

	if config.Cooldown <= 0 {
		return fmt.Errorf("%w: Cooldown must be positive, got %v", ErrInvalidWatchdogConfig, config.Cooldown)
	}

	if config.CaptureWindow <= 0 {
		return fmt.Errorf("%w: CaptureWindow must be positive, got %v", ErrInvalidWatchdogConfig, config.CaptureWindow)
	}

	if config.WarmUpDuration < 0 {
		return fmt.Errorf("%w: WarmUpDuration must be non-negative, got %v", ErrInvalidWatchdogConfig, config.WarmUpDuration)
	}

	if config.HighWaterResetCooldown <= 0 {
		return fmt.Errorf("%w: HighWaterResetCooldown must be positive, got %v", ErrInvalidWatchdogConfig, config.HighWaterResetCooldown)
	}

	if config.GoroutineLeakCheckInterval <= 0 {
		return fmt.Errorf("%w: GoroutineLeakCheckInterval must be positive, got %v", ErrInvalidWatchdogConfig, config.GoroutineLeakCheckInterval)
	}

	if config.TrendWindowSize > 0 {
		if config.TrendEvaluationInterval < 0 {
			return fmt.Errorf("%w: TrendEvaluationInterval must be non-negative when trend detection is enabled, got %v", ErrInvalidWatchdogConfig, config.TrendEvaluationInterval)
		}

		if config.TrendWarningHorizon <= 0 {
			return fmt.Errorf("%w: TrendWarningHorizon must be positive when trend detection is enabled, got %v", ErrInvalidWatchdogConfig, config.TrendWarningHorizon)
		}
	}

	return nil
}
