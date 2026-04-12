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

package interp_domain

import (
	"fmt"
	"reflect"
	"sync/atomic"
)

// fastPathDiagnostics holds per-VM counters for fast-path anomalies.
//
// Surfaces silent-failure incidents that would otherwise be lost.
//
// The counters are atomic so hot-path increments do not need a lock; hosts read them via
// FastPathStats.
type fastPathDiagnostics struct {
	// typeMismatches counts how many times a fpDispatch* function received a cached value
	// whose concrete type did not match the expected dispatcher signature. Each increment
	// corresponds to a dispatcher that falls back to reflect-based dispatch and logs the
	// event rather than returning silently.
	typeMismatches atomic.Int64

	// outOfBoundsWrites counts attempted writes to a register index outside the bank's
	// allocated range. Should always be zero in production; non-zero indicates either a
	// verifier gap or a tampered bytecode.
	outOfBoundsWrites atomic.Int64
}

// FastPathStats is the read-only view a host can fetch from the service via
// Service.FastPathStats. Hosts log these counts to surface dispatch anomalies that would
// otherwise be lost.
type FastPathStats struct {
	// TypeMismatches counts dispatcher type-assertion failures.
	TypeMismatches int64

	// OutOfBoundsWrites counts attempted bank-overrun writes.
	OutOfBoundsWrites int64
}

// FastPathStats returns a snapshot of the per-Service fast-path dispatch counters.
// Counters reset to zero when Service.Reset is called.
//
// Returns a FastPathStats snapshot.
func (s *Service) FastPathStats() FastPathStats {
	return FastPathStats{
		TypeMismatches:    s.limits.fastPathDiagnostics().typeMismatches.Load(),
		OutOfBoundsWrites: s.limits.fastPathDiagnostics().outOfBoundsWrites.Load(),
	}
}

// fastPathDiagnostics returns the per-vmLimits diagnostics block.
//
// Lazily initialises the block on first read. Stored on vmLimits (not directly on the
// Service) so spawned goroutines inherit a shared counter via vmLimits copy-by-value of
// the pointer field.
//
// Returns *fastPathDiagnostics which holds the per-VM counters.
func (l *vmLimits) fastPathDiagnostics() *fastPathDiagnostics {
	if l.diagnostics == nil {
		l.diagnostics = &fastPathDiagnostics{}
	}
	return l.diagnostics
}

// recordFastPathTypeMismatch increments the type-mismatch counter and (when configured)
// emits a structured log entry via the VM's host logger. Returns a non-nil error so
// callers can route the mismatch through the normal error path; callers that must not
// surface the mismatch wrap the returned error in a fallback-to-slow-path indicator.
//
// Takes vm (*VM) which holds the diagnostics block.
// Takes siteIndex (int) which identifies the call site for diagnostic context.
// Takes wantType (string) which is the dispatcher's expected type signature (e.g.
// "func(string) string").
// Takes gotType (string) which is the actual type the cache held.
//
// Returns a non-nil error describing the mismatch.
func recordFastPathTypeMismatch(vm *VM, siteIndex int, wantType, gotType string) error {
	if vm != nil {
		vm.limits.fastPathDiagnostics().typeMismatches.Add(1)
	}
	return fmt.Errorf("interp_domain: fast-path dispatcher at site %d expected %s, got %s",
		siteIndex, wantType, gotType)
}

// reflectTypeName returns a stable string for the dynamic type of v.
//
// Mirrors fmt.Sprintf("%T", v) but lives in this file so the fast-path mass-substitution
// does not require an extra "fmt" import in every dispatcher.
//
// Takes v (any) which is the value whose dynamic type is reported.
//
// Returns string which is the type name or "<nil>" when v is nil.
func reflectTypeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return reflect.TypeOf(v).String()
}

// recordFastPathOOBWrite increments the OOB-write counter and returns a descriptive
// error. Called by dispatchers that validate register indices before writing.
//
// Takes vm (*VM) which holds the diagnostics block.
// Takes register (registerKind) which is the bank.
// Takes index (int) which is the attempted (rejected) index.
// Takes bankSize (int) which is the allocated bank size.
//
// Returns a non-nil error describing the OOB attempt.
func recordFastPathOOBWrite(vm *VM, register registerKind, index, bankSize int) error {
	if vm != nil {
		vm.limits.fastPathDiagnostics().outOfBoundsWrites.Add(1)
	}
	return fmt.Errorf("interp_domain: fast-path attempted OOB write: bank=%v index=%d size=%d",
		register, index, bankSize)
}
