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

// CompiledFileSet holds the compiled output of one or more Go source files that belong to
// the same package. Functions from all files share a unified function table and can call
// each other.
type CompiledFileSet struct {
	// root is a synthetic container whose Functions slice holds all compiled functions from
	// every source file.
	root *CompiledFunction

	// variableInitFunction holds bytecode for package-level variable initialisers. Executed
	// before init() functions.
	variableInitFunction *CompiledFunction

	// entrypoints maps function names to their indices in root.functions. All non-init
	// functions are included.
	entrypoints map[string]uint16

	// initFunctionIndices holds the indices of init() functions in root.functions, in source
	// order.
	initFunctionIndices []uint16

	// packageVariables describes the bundle's exported package-level variables.
	//
	// Carries relative slot coordinates (the absolute index inside the originating Service
	// minus the per-kind base at compile start). At load time, Service.LoadModule
	// reconstructs settable reflect.Values backed by the recorded type and registers them in
	// the symbol registry; the post-ExecuteInits finaliser then snapshots the initialised
	// globalStore value into each storage.
	packageVariables []PackageVariableMetadata

	// slotAllocation records how many global-store slots this bundle reserved during
	// compilation, per register-kind bank. Populated by Service.PackageModule before the
	// bundle is serialised; reused by Service.LoadModule to reserve the matching counts in
	// the target Service's globalStore.
	slotAllocation SlotAllocation
}

const (
	// NumGlobalRegisterKinds is the number of register-kind banks the globalStore allocates
	// per package-level var.
	//
	// Covers int, float, string, general, bool, uint, complex (ordinals 0..6). The remaining
	// NumRegisterKinds entries cover typed slice register variants that never back package
	// globals.
	NumGlobalRegisterKinds = 7
)

// SlotAllocation records per-kind global-store reservations for a single CompiledFileSet.
// Index by register-kind ordinal (one of the first NumGlobalRegisterKinds register
// kinds).
type SlotAllocation [NumGlobalRegisterKinds]uint16

// PackageVariableMetadata describes one exported package-level var captured during
// Service.PackageModule. Type is preserved as a TypeDescriptor so the unpack path can
// rebuild the reflect.Type without source.
type PackageVariableMetadata struct {
	// Type is the bytecode-portable description of the var's Go type, used by the load path
	// to call ImportTypeDescriptor and allocate a settable storage reflect.Value. Mirrors
	// the existing serialisation pattern used for general constants and type-table entries.
	Type *TypeDescriptorData

	// Name is the exported identifier (no package qualifier).
	Name string

	// PackagePath is the full import path of the owning package. For a multi-package module
	// bundle, several PackageVariableMetadata entries share the bundle's bytecode but each
	// carries its own PackagePath.
	PackagePath string

	// RelativeSlot is the slot's index inside this bundle's allocation - always less than
	// slotAllocation[RegisterKind]. The absolute load-time slot is RelativeSlot + the
	// per-kind base assigned at load.
	RelativeSlot uint16

	// RegisterKind selects which globalStore bank the slot lives in (ints, floats, strings,
	// bools, uints, complexes, general).
	RegisterKind uint8
}

// SlotAllocation returns the bundle's per-kind slot reservations.
//
// Returns SlotAllocation which holds per-kind slot counts.
func (cfs *CompiledFileSet) SlotAllocation() SlotAllocation {
	if cfs == nil {
		return SlotAllocation{}
	}
	return cfs.slotAllocation
}

// PackageVariables returns the bundle's exported-var metadata.
//
// Entries appear in capture order. No caller-owned copy is made; do not mutate the
// returned slice.
//
// Returns []PackageVariableMetadata which is the captured metadata.
func (cfs *CompiledFileSet) PackageVariables() []PackageVariableMetadata {
	if cfs == nil {
		return nil
	}
	return cfs.packageVariables
}

// FindFunction looks up a function by name in the compiled file set.
//
// Takes name (string) which is the function name to find.
//
// Returns *CompiledFunction and nil error if found, or nil and an error if not found.
func (cfs *CompiledFileSet) FindFunction(name string) (*CompiledFunction, error) {
	index, ok := cfs.entrypoints[name]
	if !ok {
		return nil, errEntrypointNotFound
	}
	return cfs.root.functions[index], nil
}

// FunctionNames returns the entrypoint names registered in the compiled file set, in
// unspecified order. Used by external tooling (the pipit CLI) to enumerate functions for
// disassembly and inspection.
//
// Returns []string which contains the registered entrypoint names.
func (cfs *CompiledFileSet) FunctionNames() []string {
	if cfs == nil {
		return nil
	}
	names := make([]string, 0, len(cfs.entrypoints))
	for name := range cfs.entrypoints {
		names = append(names, name)
	}
	return names
}
