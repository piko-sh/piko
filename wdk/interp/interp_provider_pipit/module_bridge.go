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
	"context"
	"errors"

	"piko.sh/piko/wdk/modules"
	"pipit.sh/pipit"
	pipitmodule "pipit.sh/pipit/sdk/module"
)

// This file converts between piko's module vocabulary and pipit's.
//
// The two are structurally identical, because pipit's copy was lifted from piko's, and it
// is tempting to make piko's an alias of pipit's and delete these functions. Do not.
// piko's wdk/modules and its three provider packages live in piko's ROOT module, so
// aliasing would put a requirement on pipit into piko.sh/piko's go.mod, and a piko user
// who never enables interpreted mode would start downloading and linking an interpreter.
// This module is the only place allowed to know both vocabularies, which is what keeps
// that requirement out.
//
// TestPikoRootDoesNotRequirePipit in the root module guards the invariant.

var (
	// moduleSentinelPairs maps each pipit module sentinel to piko's equivalent.
	//
	// The two packages declare their own sentinels, so an error raised inside the
	// interpreter carries pipit's and would not match a piko caller's errors.Is. Aliasing
	// piko's to pipit's would fix that and break strict optionality, so the mapping lives
	// here instead.
	moduleSentinelPairs = []struct {
		// pipitErr is the sentinel the interpreter raises.
		pipitErr error

		// pikoErr is the sentinel piko callers match against.
		pikoErr error
	}{
		{pipitmodule.ErrUnpinnedRef, modules.ErrUnpinnedModuleRef},
		{pipitmodule.ErrNotFound, modules.ErrModuleNotFound},
		{pipitmodule.ErrIntegrityMismatch, modules.ErrIntegrityMismatch},
		{pipitmodule.ErrCapabilityDenied, modules.ErrCapabilityDenied},
		{pipitmodule.ErrCapabilityExceedsPolicy, modules.ErrCapabilityExceedsPolicy},
		{pipitmodule.ErrStdlibIncompatible, modules.ErrStdlibIncompatible},
		{pipitmodule.ErrFrozen, modules.ErrFrozen},
	}
)

// translatedModuleError carries a module error raised by the interpreter together with
// piko's matching sentinel, so errors.Is succeeds for either vocabulary while the message
// stays the interpreter's own.
type translatedModuleError struct {
	// sentinel is piko's sentinel matching the interpreter's.
	sentinel error

	// original is the error the interpreter raised.
	original error
}

// Error returns the interpreter's message unchanged.
//
// Returns string which is the original error's message.
func (e *translatedModuleError) Error() string {
	return e.original.Error()
}

// Unwrap exposes both piko's sentinel and the original error to errors.Is and errors.As.
//
// Returns []error which holds piko's sentinel followed by the original error.
func (e *translatedModuleError) Unwrap() []error {
	return []error{e.sentinel, e.original}
}

// toPipitRef converts a piko module ref into pipit's.
//
// Takes ref (modules.ModuleRef) which identifies a module and its pin.
//
// Returns pipitmodule.Ref carrying the same identity.
func toPipitRef(ref modules.ModuleRef) pipitmodule.Ref {
	return pipitmodule.Ref{
		Path:    ref.Path,
		Version: ref.Version,
		Pin:     ref.Pin,
	}
}

// toPipitDescriptor converts a piko module descriptor into pipit's.
//
// Takes descriptor (*modules.ModuleDescriptor) which may be nil.
//
// Returns *pipitmodule.Descriptor, or nil when descriptor is nil.
func toPipitDescriptor(descriptor *modules.ModuleDescriptor) *pipitmodule.Descriptor {
	if descriptor == nil {
		return nil
	}

	return &pipitmodule.Descriptor{
		Entrypoints:     descriptor.Entrypoints,
		Annotations:     descriptor.Annotations,
		Ref:             toPipitRef(descriptor.Ref),
		StdlibVersion:   descriptor.StdlibVersion,
		Capabilities:    convertCapabilities[pipitmodule.Capability](descriptor.Capabilities),
		SymbolAllowlist: descriptor.SymbolAllowlist,
		SchemaVersion:   descriptor.SchemaVersion,
	}
}

// toPipitBundle converts a piko module bundle into pipit's.
//
// The byte slices are shared because a bundle is treated as immutable once resolved, and
// copying the bytecode of every queued module would be wasteful.
//
// Takes bundle (*modules.ModuleBundle) which may be nil.
//
// Returns *pipitmodule.Bundle, or nil when bundle is nil.
func toPipitBundle(bundle *modules.ModuleBundle) *pipitmodule.Bundle {
	if bundle == nil {
		return nil
	}

	return &pipitmodule.Bundle{
		Descriptor:  toPipitDescriptor(bundle.Descriptor),
		Bytecode:    bundle.Bytecode,
		TypesExport: bundle.TypesExport,
	}
}

// fromPipitBundle converts a pipit module bundle back into piko's.
//
// The interpreter produces a bundle when it packages a module, and piko's own registry
// and CLI speak piko's vocabulary, so the value has to come back across.
//
// Takes bundle (*pipitmodule.Bundle) which may be nil.
//
// Returns *modules.ModuleBundle, or nil when bundle is nil.
func fromPipitBundle(bundle *pipitmodule.Bundle) *modules.ModuleBundle {
	if bundle == nil {
		return nil
	}

	return &modules.ModuleBundle{
		Descriptor:  fromPipitDescriptor(bundle.Descriptor),
		Bytecode:    bundle.Bytecode,
		TypesExport: bundle.TypesExport,
	}
}

// fromPipitDescriptor converts a pipit module descriptor back into piko's.
//
// Takes descriptor (*pipitmodule.Descriptor) which may be nil.
//
// Returns *modules.ModuleDescriptor, or nil when descriptor is nil.
func fromPipitDescriptor(descriptor *pipitmodule.Descriptor) *modules.ModuleDescriptor {
	if descriptor == nil {
		return nil
	}

	return &modules.ModuleDescriptor{
		Entrypoints:     descriptor.Entrypoints,
		Annotations:     descriptor.Annotations,
		Ref:             fromPipitRef(descriptor.Ref),
		StdlibVersion:   descriptor.StdlibVersion,
		Capabilities:    convertCapabilities[modules.Capability](descriptor.Capabilities),
		SymbolAllowlist: descriptor.SymbolAllowlist,
		SchemaVersion:   descriptor.SchemaVersion,
	}
}

// convertCapabilities copies a capability set between piko's and pipit's capability
// types, which share one shape.
//
// Takes capabilities ([]From) which is the set to copy.
//
// Returns []To which holds the same claims in the target type.
func convertCapabilities[To, From ~struct{ Axis, Scope string }](capabilities []From) []To {
	converted := make([]To, 0, len(capabilities))
	for _, capability := range capabilities {
		converted = append(converted, To(capability))
	}
	return converted
}

// fromPipitRef converts a pipit module ref back into piko's.
//
// Takes ref (pipitmodule.Ref) which identifies a module and its pin.
//
// Returns modules.ModuleRef carrying the same identity.
func fromPipitRef(ref pipitmodule.Ref) modules.ModuleRef {
	return modules.ModuleRef{
		Path:    ref.Path,
		Version: ref.Version,
		Pin:     ref.Pin,
	}
}

// PackageModuleForPiko packages a module and hands the result back in piko's vocabulary.
//
// Exported so piko-side tests and callers can package a module without learning pipit's
// module types; the bridge converts between the two representations.
//
// Takes interpreter (*pipit.Interpreter) which compiles and packages the module.
// Takes descriptor (modules.ModuleDescriptor) which declares the module's identity.
// Takes modulePath (string) which is the module's import path.
// Takes packages (map[string]map[string]string) which are its sources by package path.
// Takes pack (func(*pipit.CompiledFileSet) []byte) which serialises the compiled output.
//
// Returns *modules.ModuleBundle which is the packaged module.
// Returns error when compilation or packaging fails, matching piko's module sentinels
// where one applies, or when the interpreter panics.
func PackageModuleForPiko(
	ctx context.Context,
	interpreter *pipit.Interpreter,
	descriptor modules.ModuleDescriptor,
	modulePath string,
	packages map[string]map[string]string,
	pack func(*pipit.CompiledFileSet) []byte,
) (result *modules.ModuleBundle, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = interpreterPanicError(ctx, "packaging a module", recovered)
		}
	}()
	converted := toPipitDescriptor(&descriptor)

	bundle, err := interpreter.PackageModule(ctx, *converted, modulePath, packages, pack)
	if err != nil {
		return nil, translateModuleError(err)
	}

	return fromPipitBundle(bundle), nil
}

// translateModuleError re-raises a module error so it also matches piko's sentinels.
//
// The returned error wraps both the original and piko's sentinel, so errors.Is succeeds
// for either vocabulary, and its message is the original's alone, so the sentinel text is
// not repeated.
//
// Takes err (error) which may be nil or carry no module sentinel.
//
// Returns error matching piko's sentinel where one applies, otherwise err unchanged.
func translateModuleError(err error) error {
	if err == nil {
		return nil
	}

	for _, pair := range moduleSentinelPairs {
		if errors.Is(err, pair.pipitErr) {
			return &translatedModuleError{sentinel: pair.pikoErr, original: err}
		}
	}

	return err
}
