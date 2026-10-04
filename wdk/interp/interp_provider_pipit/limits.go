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

package interp_provider_pipit

import (
	"time"

	"pipit.sh/pipit"
)

const (
	// restrictedMaxExecutionTime caps the wall-clock time of one evaluation in restricted
	// mode when no explicit limit is configured.
	restrictedMaxExecutionTime = 30 * time.Second

	// restrictedMaxAllocSize caps the element count of a single allocation in restricted
	// mode when no explicit limit is configured.
	restrictedMaxAllocSize = 16 << 20

	// restrictedMaxOutputSize caps the bytes print and println may write in restricted mode
	// when no explicit limit is configured.
	restrictedMaxOutputSize = 16 << 20

	// restrictedMaxSourceSize caps the total source accepted for one compilation in
	// restricted mode when no explicit limit is configured.
	restrictedMaxSourceSize = 16 << 20

	// restrictedMaxStringSize caps the length one string concatenation may produce in
	// restricted mode when no explicit limit is configured.
	restrictedMaxStringSize = 64 << 20

	// restrictedMaxGoroutines caps the goroutines interpreted code may run at once in
	// restricted mode when no explicit limit is configured.
	restrictedMaxGoroutines int32 = 1024
)

// limitValue is the set of numeric types the interpreter's resource limits use.
type limitValue interface {
	~int | ~int32 | ~int64
}

// limitSetting records one resource limit and whether the host configured it explicitly,
// so an explicit zero (unlimited) is told apart from "not configured".
type limitSetting[T limitValue] struct {
	// value is the configured limit; zero means unlimited.
	value T

	// configured is true once an option has set the limit.
	configured bool
}

// resourceLimits holds the interpreter resource limits a Provider applies to every
// interpreter it builds.
type resourceLimits struct {
	// maxExecutionTime caps the wall-clock time of one evaluation.
	maxExecutionTime limitSetting[time.Duration]

	// costBudget caps the metered computation cost of one execution.
	costBudget limitSetting[int64]

	// maxCallDepth caps the interpreted call stack depth.
	maxCallDepth limitSetting[int]

	// maxAllocSize caps the element count of a single allocation.
	maxAllocSize limitSetting[int]

	// maxOutputSize caps the bytes print and println may write.
	maxOutputSize limitSetting[int]

	// maxSourceSize caps the total source size accepted for one compilation.
	maxSourceSize limitSetting[int]

	// maxStringSize caps the length one string concatenation may produce.
	maxStringSize limitSetting[int]

	// maxGoroutines caps the goroutines interpreted code may run at once.
	maxGoroutines limitSetting[int32]
}

// resolve returns the configured limit when set, the restricted-mode default when
// restricted, or zero for unlimited execution otherwise.
//
// Takes restricted (bool) which reports whether the provider runs untrusted scripts.
// Takes restrictedDefault (T) which is the limit applied in restricted mode when none was
// configured.
//
// Returns T which is the effective limit; zero means unlimited.
func (s limitSetting[T]) resolve(restricted bool, restrictedDefault T) T {
	if s.configured {
		return s.value
	}
	if restricted {
		return restrictedDefault
	}
	return 0
}

// pipitOptions converts the effective limits into interpreter options, leaving out every
// limit that resolves to unlimited.
//
// Takes restricted (bool) which selects the restricted-mode defaults for limits the host
// did not configure.
//
// Returns []pipit.Option which applies the effective limits.
func (l *resourceLimits) pipitOptions(restricted bool) []pipit.Option {
	var options []pipit.Option
	if limit := l.maxExecutionTime.resolve(restricted, restrictedMaxExecutionTime); limit > 0 {
		options = append(options, pipit.WithMaxExecutionTime(limit))
	}
	if limit := l.costBudget.resolve(restricted, 0); limit > 0 {
		options = append(options, pipit.WithCostBudget(limit))
	}
	if limit := l.maxCallDepth.resolve(restricted, 0); limit > 0 {
		options = append(options, pipit.WithMaxCallDepth(limit))
	}
	if limit := l.maxAllocSize.resolve(restricted, restrictedMaxAllocSize); limit > 0 {
		options = append(options, pipit.WithMaxAllocSize(limit))
	}
	if limit := l.maxOutputSize.resolve(restricted, restrictedMaxOutputSize); limit > 0 {
		options = append(options, pipit.WithMaxOutputSize(limit))
	}
	if limit := l.maxSourceSize.resolve(restricted, restrictedMaxSourceSize); limit > 0 {
		options = append(options, pipit.WithMaxSourceSize(limit))
	}
	if limit := l.maxStringSize.resolve(restricted, restrictedMaxStringSize); limit > 0 {
		options = append(options, pipit.WithMaxStringSize(limit))
	}
	if limit := l.maxGoroutines.resolve(restricted, restrictedMaxGoroutines); limit > 0 {
		options = append(options, pipit.WithMaxGoroutines(limit))
	}
	return options
}

// WithMaxExecutionTime caps the wall-clock time of one evaluation, including the init
// functions run by CompileAndExecute.
//
// Restricted mode applies a 30 second cap unless this option sets another; compiled and
// development use is unlimited unless configured. Zero means unlimited.
//
// Takes maximum (time.Duration) which is the time limit.
//
// Returns ProviderOption which configures the provider.
func WithMaxExecutionTime(maximum time.Duration) ProviderOption {
	return func(p *Provider) {
		p.limits.maxExecutionTime = limitSetting[time.Duration]{value: max(maximum, 0), configured: true}
	}
}

// WithCostBudget caps the metered computation cost of one execution.
//
// Unlimited unless configured, in every mode. Zero means unlimited.
//
// Takes budget (int64) which is the cost budget.
//
// Returns ProviderOption which configures the provider.
func WithCostBudget(budget int64) ProviderOption {
	return func(p *Provider) {
		p.limits.costBudget = limitSetting[int64]{value: max(budget, 0), configured: true}
	}
}

// WithMaxCallDepth caps the interpreted call stack depth before a stack overflow is
// reported.
//
// The interpreter's built-in depth limit applies unless configured. Zero keeps that
// built-in limit.
//
// Takes maximum (int) which is the call depth limit.
//
// Returns ProviderOption which configures the provider.
func WithMaxCallDepth(maximum int) ProviderOption {
	return func(p *Provider) {
		p.limits.maxCallDepth = limitSetting[int]{value: max(maximum, 0), configured: true}
	}
}

// WithMaxAllocSize caps the element count of a single allocation (make of a slice or
// channel).
//
// Restricted mode applies a cap of 16Mi elements unless this option sets another. Zero
// means unlimited.
//
// Takes maximum (int) which is the element limit.
//
// Returns ProviderOption which configures the provider.
func WithMaxAllocSize(maximum int) ProviderOption {
	return func(p *Provider) {
		p.limits.maxAllocSize = limitSetting[int]{value: max(maximum, 0), configured: true}
	}
}

// WithMaxOutputSize caps the bytes print and println may write.
//
// Restricted mode applies a 16 MiB cap unless this option sets another. Zero means
// unlimited.
//
// Takes maximum (int) which is the output limit in bytes.
//
// Returns ProviderOption which configures the provider.
func WithMaxOutputSize(maximum int) ProviderOption {
	return func(p *Provider) {
		p.limits.maxOutputSize = limitSetting[int]{value: max(maximum, 0), configured: true}
	}
}

// WithMaxSourceSize caps the total source size accepted for one compilation.
//
// Restricted mode applies a 16 MiB cap unless this option sets another. Zero means
// unlimited.
//
// Takes maximum (int) which is the source limit in bytes.
//
// Returns ProviderOption which configures the provider.
func WithMaxSourceSize(maximum int) ProviderOption {
	return func(p *Provider) {
		p.limits.maxSourceSize = limitSetting[int]{value: max(maximum, 0), configured: true}
	}
}

// WithMaxStringSize caps the length one string concatenation may produce.
//
// Restricted mode applies a 64 MiB cap unless this option sets another. Zero means
// unlimited.
//
// Takes maximum (int) which is the string limit in bytes.
//
// Returns ProviderOption which configures the provider.
func WithMaxStringSize(maximum int) ProviderOption {
	return func(p *Provider) {
		p.limits.maxStringSize = limitSetting[int]{value: max(maximum, 0), configured: true}
	}
}

// WithMaxGoroutines caps the goroutines interpreted code may run at once.
//
// Restricted mode applies a cap of 1024 unless this option sets another. Zero means
// unlimited.
//
// Takes maximum (int32) which is the goroutine limit.
//
// Returns ProviderOption which configures the provider.
func WithMaxGoroutines(maximum int32) ProviderOption {
	return func(p *Provider) {
		p.limits.maxGoroutines = limitSetting[int32]{value: max(maximum, 0), configured: true}
	}
}
