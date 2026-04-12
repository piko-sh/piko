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
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"golang.org/x/tools/go/gcexportdata"

	"piko.sh/piko/internal/modules/modules_domain"
	"piko.sh/piko/wdk/safeconv"
)

// PackageModule compiles a multi-package Go program and packages it.
//
// Compilation reuses the path used by Service.CompileProgram; the bundle wraps the
// serialised bytecode with the descriptor.
//
// Takes ctx (context.Context) which is observed for cancellation.
// Takes descriptor (modules_domain.ModuleDescriptor) which declares the module's identity
// and capability set. Required fields: SchemaVersion, Ref.Path.
// Takes modulePath (string) which is the Go module path passed to CompileProgram.
// Takes packages (map[string]map[string]string) which maps package -> file -> source,
// same shape as CompileProgram.
// Takes bytecodePacker (func(*CompiledFileSet) []byte) which serialises the compiled
// output. Pass interp_adapters.PackCompiledFileSetToBytes from the driven_bytecode_store
// package; piko keeps the packer abstract here so this layer does not depend on the
// adapters package.
//
// Returns *modules_domain.ModuleBundle which is the packaged module.
// Returns error when compilation, packing, or fingerprinting fails.
func (s *Service) PackageModule(
	ctx context.Context,
	descriptor modules_domain.ModuleDescriptor,
	modulePath string,
	packages map[string]map[string]string,
	bytecodePacker func(*CompiledFileSet) []byte,
) (*modules_domain.ModuleBundle, error) {
	if bytecodePacker == nil {
		return nil, errors.New("interp_domain: PackageModule requires a non-nil bytecodePacker")
	}
	if err := descriptor.Validate(); err != nil {
		return nil, err
	}

	bridgesAtStart := len(s.pendingVarBridges)
	baseAtStart := s.globals.lengths()

	compiled, err := s.CompileProgram(ctx, modulePath, packages)
	if err != nil {
		return nil, fmt.Errorf("interp_domain: PackageModule compile: %w", err)
	}

	baseAtEnd := s.globals.lengths()
	if err := s.stampBundleMetadata(compiled, baseAtStart, baseAtEnd, s.pendingVarBridges[bridgesAtStart:]); err != nil {
		return nil, err
	}

	bytecode := bytecodePacker(compiled)

	typesExport, err := s.encodeTypesExportForPackages(modulePath, packages)
	if err != nil {
		return nil, fmt.Errorf("interp_domain: PackageModule type-export: %w", err)
	}

	bundle := &modules_domain.ModuleBundle{
		Descriptor:  &descriptor,
		Bytecode:    bytecode,
		TypesExport: typesExport,
	}
	fingerprint, err := bundle.Fingerprint()
	if err != nil {
		return nil, fmt.Errorf("interp_domain: PackageModule fingerprint: %w", err)
	}
	bundle.Descriptor.Ref.Pin = fingerprint
	return bundle, nil
}

// stampBundleMetadata relativises operands and records bundle metadata.
//
// Relativises the compiled bundle's global operands against baseAtStart and records slot
// allocation and exported-var metadata on the CompiledFileSet so the unpack path can
// rebuild storage and registry entries without source access.
//
// Takes compiled (*CompiledFileSet) which receives the metadata stamps.
// Takes baseAtStart (SlotAllocation) which is the slot count before compilation began.
// Takes baseAtEnd (SlotAllocation) which is the slot count after compilation completed.
// Takes newBridges ([]pendingVarBridge) which lists the var bridges produced by this
// compilation.
//
// Returns error when relativisation or metadata collection fails.
func (s *Service) stampBundleMetadata(compiled *CompiledFileSet, baseAtStart, baseAtEnd SlotAllocation, newBridges []pendingVarBridge) error {
	var slotAlloc SlotAllocation
	for k := range NumGlobalRegisterKinds {
		slotAlloc[k] = baseAtEnd[k] - baseAtStart[k]
	}
	if err := relativiseGlobalOperands(compiled.root, baseAtStart); err != nil {
		return fmt.Errorf("interp_domain: PackageModule relativise: %w", err)
	}
	if err := relativiseGlobalOperands(compiled.variableInitFunction, baseAtStart); err != nil {
		return fmt.Errorf("interp_domain: PackageModule relativise varinit: %w", err)
	}
	compiled.slotAllocation = slotAlloc
	if len(newBridges) > 0 {
		packageVars, err := s.collectPackageVariables(newBridges, baseAtStart)
		if err != nil {
			return fmt.Errorf("interp_domain: PackageModule var metadata: %w", err)
		}
		compiled.packageVariables = packageVars
	}
	return nil
}

// collectPackageVariables converts pendingVarBridges to bundle metadata.
//
// Converts a slice of pendingVarBridge (the new entries produced by the current
// PackageModule's CompileProgram) into PackageVariableMetadata with slot indices made
// relative to baseAtStart. The result is stored on the produced CompiledFileSet and
// reconstructed at load time.
//
// Takes pending ([]pendingVarBridge) which is the new var bridge list.
// Takes baseAtStart (SlotAllocation) which is the slot count before this compilation
// began.
//
// Returns []PackageVariableMetadata which is the per-var metadata.
// Returns error when a kind is out of range or a slot precedes baseAtStart.
func (*Service) collectPackageVariables(pending []pendingVarBridge, baseAtStart SlotAllocation) ([]PackageVariableMetadata, error) {
	count := 0
	for _, p := range pending {
		count += len(p.vars)
	}
	out := make([]PackageVariableMetadata, 0, count)
	for _, p := range pending {
		for _, v := range p.vars {
			kind := uint8(v.slot.kind)
			if int(kind) >= NumGlobalRegisterKinds {
				return nil, fmt.Errorf("var %s.%s has out-of-range kind %d", p.importPath, v.name, kind)
			}
			absoluteSlot := safeconv.MustIntToUint16(v.slot.index)
			if absoluteSlot < baseAtStart[kind] {
				return nil, fmt.Errorf("var %s.%s slot %d precedes baseAtStart %d (kind %d)", p.importPath, v.name, absoluteSlot, baseAtStart[kind], kind)
			}
			out = append(out, PackageVariableMetadata{
				Name:         v.name,
				PackagePath:  p.importPath,
				Type:         new(exportTypeDescriptor(reflectTypeToDescriptor(v.reflectType))),
				RegisterKind: kind,
				RelativeSlot: absoluteSlot - baseAtStart[kind],
			})
		}
	}
	return out, nil
}

// encodeTypesExportForPackages serialises packages as gcexportdata.
//
// Walks the packages map provided to PackageModule, looks up each sub-package's
// [go/types.Package] from the registry (where it was just registered by the trailing
// CompileProgram call), and serialises each as gcexportdata.
//
// Takes modulePath (string) which is the Go module path.
// Takes packages (map[string]map[string]string) which maps package -> file -> source.
//
// Returns []byte which is the TLV-encoded TypesExport payload, or nil bytes when no
// non-main packages are present.
// Returns error when a gcexportdata serialisation step fails.
func (s *Service) encodeTypesExportForPackages(modulePath string, packages map[string]map[string]string) ([]byte, error) {
	relPaths := make([]string, 0, len(packages))
	for relPath := range packages {
		relPaths = append(relPaths, relPath)
	}
	slices.Sort(relPaths)

	entries := make([]modules_domain.TypesExportEntry, 0, len(relPaths))
	for _, relPath := range relPaths {
		importPath := modulePath
		if relPath != "" {
			importPath = modulePath + "/" + relPath
		}
		pkg, err := s.symbols.Import(importPath)
		if err != nil {
			continue
		}
		if pkg == nil || pkg.Name() == "main" {
			continue
		}
		var buf bytes.Buffer
		if err := gcexportdata.Write(&buf, s.fileSet, pkg); err != nil {
			return nil, fmt.Errorf("gcexportdata.Write %q: %w", importPath, err)
		}
		funcTable := s.deriveFuncTableForPackage(importPath)
		entries = append(entries, modules_domain.TypesExportEntry{
			ImportPath: importPath,
			Data:       buf.Bytes(),
			FuncTable:  funcTable,
		})
	}
	return modules_domain.EncodeTypesExport(entries)
}

// deriveFuncTableForPackage recovers function indices for a package.
//
// Walks the symbol registry's reflect.Value exports for importPath and recovers each
// function's index in the shared root function by pointer comparison. Needed because
// CompileProgram's returned CompiledFileSet only preserves one package's funcTable (the
// main package's, or the last compiled one); to bridge a loaded bytecode bundle's other
// sub-packages we must ship the per-package index map alongside the types data.
//
// Takes importPath (string) which is the package import path.
//
// Returns map[string]uint16 which maps exported function name to its index in the shared
// root function, or nil when no exports are registered for importPath.
func (s *Service) deriveFuncTableForPackage(importPath string) map[string]uint16 {
	exports, ok := s.symbols.LookupPackage(importPath)
	if !ok {
		return nil
	}
	out := make(map[string]uint16, len(exports))
	for name, value := range exports {
		if !value.IsValid() || value.Kind() != reflect.Pointer {
			continue
		}
		closure, ok := reflect.TypeAssert[*runtimeClosure](value)
		if !ok || closure == nil || closure.function == nil || closure.rootFunction == nil {
			continue
		}
		for i, fn := range closure.rootFunction.functions {
			if fn == closure.function {
				out[name] = uint16(i)
				break
			}
		}
	}
	return out
}

// appendLoadedVarBridges rebuilds pendingVarBridge entries at load.
//
// Rebuilds entries from the bundle's per-package var metadata and the load-time slot
// bases. Each var gets a freshly-allocated settable reflect.Value (the storage advertised
// in the symbol registry by bridgeBytecodeModule); after Service.ExecuteInits runs the
// bundle's variableInitFunction, finalisePendingVarBridges snapshots the populated
// globalStore values into these storages.
//
// Takes vars ([]PackageVariableMetadata) which is the bundle's per-package var metadata.
// Takes bases (SlotAllocation) which is the load-time slot base table.
//
// Returns error when a type descriptor decode fails or a kind is out of range.
func (s *Service) appendLoadedVarBridges(vars []PackageVariableMetadata, bases SlotAllocation) error {
	if len(vars) == 0 {
		return nil
	}
	byPackage := make(map[string][]packageVarExport)
	for _, v := range vars {
		if v.Type == nil {
			return fmt.Errorf("var %s.%s missing type descriptor", v.PackagePath, v.Name)
		}
		if int(v.RegisterKind) >= NumGlobalRegisterKinds {
			return fmt.Errorf("var %s.%s register-kind %d out of range", v.PackagePath, v.Name, v.RegisterKind)
		}
		reflectType, err := s.importTypeDescriptorAsReflectType(*v.Type)
		if err != nil {
			return fmt.Errorf("var %s.%s reflect type: %w", v.PackagePath, v.Name, err)
		}
		storage := reflect.New(reflectType).Elem()
		absoluteSlot := int(bases[v.RegisterKind]) + int(v.RelativeSlot)
		byPackage[v.PackagePath] = append(byPackage[v.PackagePath], packageVarExport{
			name:        v.Name,
			reflectType: reflectType,
			storage:     storage,
			slot:        globalVariableInfo{index: absoluteSlot, kind: registerKind(v.RegisterKind)},
		})
	}
	for importPath, vars := range byPackage {
		s.pendingVarBridges = append(s.pendingVarBridges, pendingVarBridge{
			importPath: importPath,
			vars:       vars,
		})
	}
	return nil
}

// runLoadedBundleInits executes the bundle's variable initialisers.
//
// Executes both the top-level variableInitFunction (when present) and each function
// indexed by initFunctionIndices. Mirrors what Service.ExecuteInits does minus the
// finalisePendingVarBridges step so that only this load's bridges are drained (via
// snapshotLoadedVarBridges); any host-level later compile and ExecuteInits still
// finalises its own bridges.
//
// The per-package variableInitFunction synthesised by compileSinglePackage lives at one
// of the indices in initFunctionIndices, not at cfs.variableInitFunction (which is only
// populated by CompileFileSet, not CompileProgram). The init-functions loop therefore
// populates package vars for bundles produced by PackageModule.
//
// Takes ctx (context.Context) which is observed for cancellation.
// Takes cfs (*CompiledFileSet) which is the loaded bundle.
//
// Returns error when an init function execution fails.
func (s *Service) runLoadedBundleInits(ctx context.Context, cfs *CompiledFileSet) error {
	if cfs == nil || cfs.root == nil {
		return nil
	}
	if cfs.variableInitFunction != nil {
		if err := s.runVariableInits(ctx, cfs); err != nil {
			return err
		}
	}
	for _, initIndex := range cfs.initFunctionIndices {
		if int(initIndex) >= len(cfs.root.functions) {
			continue
		}
		function := cfs.root.functions[initIndex]
		if err := s.executeInitFunc(ctx, cfs.root, function); err != nil {
			return fmt.Errorf("executing loaded-bundle init function: %w", err)
		}
	}
	return nil
}

// snapshotLoadedVarBridges copies post-init values into bridge storages.
//
// Copies the post-init globalStore values into the storages for the pendingVarBridge
// entries in the half-open range [startIndex, len). Entries stay in the queue so
// bridgeBytecodeModule (which runs later in LoadModule) can still iterate them when
// merging var storages into the exports map. The next host-level Service.ExecuteInits
// call re-snapshots harmlessly because storage.Set is idempotent for the same source
// value.
//
// Takes startIndex (int) which is the first pendingVarBridge index to snapshot.
func (s *Service) snapshotLoadedVarBridges(startIndex int) {
	if startIndex >= len(s.pendingVarBridges) {
		return
	}
	for _, pending := range s.pendingVarBridges[startIndex:] {
		for _, v := range pending.vars {
			value, ok := s.globals.snapshotVarAs(v.slot, v.reflectType)
			if !ok {
				continue
			}
			v.storage.Set(value)
		}
	}
}

// importTypeDescriptorAsReflectType decodes a TypeDescriptorData.
//
// Decodes a serialised TypeDescriptorData into a reflect.Type via the same import path
// the bytecode unpacker uses to reconstruct type-table entries.
//
// Takes data (TypeDescriptorData) which is the serialised descriptor.
//
// Returns reflect.Type which is the resolved native type.
// Returns error when the descriptor cannot be resolved.
func (s *Service) importTypeDescriptorAsReflectType(data TypeDescriptorData) (reflect.Type, error) {
	descriptor := ImportTypeDescriptor(data)
	return descriptorToReflectType(descriptor, s.symbols)
}

// LoadModule verifies, unpacks, and registers a module bundle.
//
// Verifies a bundle, checks its capability set against the host policy (via the
// configured CapabilityHook), unpacks the bytecode, and registers the module so
// subsequent calls in this Service can resolve symbols within it. Verification proceeds
// in the audit-friendly order of bundle structural validation, pin verification against
// the supplied ref, CapabilityHook consultation per declared capability claim, and symbol
// registration scoped to the descriptor's allowlist.
//
// Takes ctx (context.Context) which is observed for cancellation; loading itself is
// synchronous.
// Takes bundle (*modules_domain.ModuleBundle) which must be non-nil and validation-clean.
// Takes ref (modules_domain.ModuleRef) which carries the expected pin; pass the same ref
// the host's ModuleProvider was called with so the integrity check sees the
// operator-declared pin.
// Takes _ (modules_domain.ModuleProvider) which is reserved for resolving transitive
// module dependencies during load. May be nil.
// Takes bytecodeUnpacker (func([]byte, *SymbolRegistry) (*CompiledFileSet, error)) which
// deserialises the bytecode. Hosts pass the LoadCompiledFromBytes function exposed by
// driven_bytecode_store; the dependency is injected so interp_domain does not need to
// import interp_adapters.
//
// Returns *modules_domain.LoadedModule which is the handle the host can use to inspect or
// unload the module. The handle's Fingerprint is the content hash captured at load time;
// hosts may use it as a cache key.
// Returns error when verification, unpacking, or capability checks fail.
func (s *Service) LoadModule(
	ctx context.Context,
	bundle *modules_domain.ModuleBundle,
	ref modules_domain.ModuleRef,
	_ modules_domain.ModuleProvider,
	bytecodeUnpacker func([]byte, *SymbolRegistry) (*CompiledFileSet, error),
) (*modules_domain.LoadedModule, error) {
	if err := s.validateLoadModuleInputs(ctx, bundle, ref, bytecodeUnpacker); err != nil {
		return nil, err
	}

	cfs, err := bytecodeUnpacker(bundle.Bytecode, s.symbols)
	if err != nil {
		return nil, fmt.Errorf("interp_domain: LoadModule unpack: %w", err)
	}

	if err := s.installLoadedBundle(ctx, cfs, bundle); err != nil {
		return nil, err
	}

	fingerprint, err := bundle.Fingerprint()
	if err != nil {
		return nil, err
	}
	return &modules_domain.LoadedModule{
		Descriptor:  bundle.Descriptor,
		Fingerprint: fingerprint,
	}, nil
}

// validateLoadModuleInputs performs LoadModule pre-flight checks.
//
// Checks context liveness, a non-nil unpacker, bundle self-validation, ref verification,
// and capability consultation.
//
// Takes ctx (context.Context) which is observed for cancellation.
// Takes bundle (*modules_domain.ModuleBundle) which must self-validate.
// Takes ref (modules_domain.ModuleRef) which must match the bundle pin.
// Takes bytecodeUnpacker (func([]byte, *SymbolRegistry) (*CompiledFileSet, error)) which
// must be non-nil.
//
// Returns error when any pre-flight check fails.
func (s *Service) validateLoadModuleInputs(
	ctx context.Context,
	bundle *modules_domain.ModuleBundle,
	ref modules_domain.ModuleRef,
	bytecodeUnpacker func([]byte, *SymbolRegistry) (*CompiledFileSet, error),
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if bytecodeUnpacker == nil {
		return errors.New("interp_domain: LoadModule requires a non-nil bytecodeUnpacker")
	}
	if err := bundle.Validate(); err != nil {
		return err
	}
	if err := bundle.VerifyAgainstRef(ref); err != nil {
		return err
	}
	return s.consultModuleCapabilities(ctx, bundle.Descriptor)
}

// installLoadedBundle reserves slots and installs an unpacked bundle.
//
// Reserves global-store slots for the unpacked bundle and stamps per-kind bases onto its
// functions, rebuilds the pendingVarBridge entries, bridges the bundle's TypesExport
// packages into the registry, runs its variable initialisers, and snapshots the
// initialised values into the registered storages. Each loaded bundle owns its own
// variableInitFunction and is installed independently.
//
// Takes ctx (context.Context) which is observed for cancellation.
// Takes cfs (*CompiledFileSet) which is the unpacked bundle bytecode.
// Takes bundle (*modules_domain.ModuleBundle) which carries the TypesExport payload to
// bridge.
//
// Returns error when any installation step fails.
func (s *Service) installLoadedBundle(ctx context.Context, cfs *CompiledFileSet, bundle *modules_domain.ModuleBundle) error {
	loadBases := s.globals.reserveSlots(cfs.slotAllocation)
	setGlobalBases(cfs.root, &loadBases)
	setGlobalBases(cfs.variableInitFunction, &loadBases)

	bridgesAtLoad := len(s.pendingVarBridges)
	if err := s.appendLoadedVarBridges(cfs.packageVariables, loadBases); err != nil {
		return fmt.Errorf("interp_domain: LoadModule var bridges: %w", err)
	}
	if err := s.bridgeLoadedTypesExport(bundle, cfs); err != nil {
		return err
	}
	if err := s.runLoadedBundleInits(ctx, cfs); err != nil {
		return fmt.Errorf("interp_domain: LoadModule inits: %w", err)
	}
	if len(s.pendingVarBridges) > bridgesAtLoad {
		s.snapshotLoadedVarBridges(bridgesAtLoad)
	}
	return nil
}

// bridgeLoadedTypesExport bridges the bundle's TypesExport into registry.
//
// Decodes the bundle's TypesExport and bridges each go/types.Package into the symbol
// registry so a downstream importer can resolve this module's packages. No-op when the
// bundle carries no TypesExport (function-only module).
//
// Takes bundle (*modules_domain.ModuleBundle) which carries the TypesExport payload.
// Takes cfs (*CompiledFileSet) which is the unpacked bundle bytecode.
//
// Returns error when decode or bridging fails.
func (s *Service) bridgeLoadedTypesExport(bundle *modules_domain.ModuleBundle, cfs *CompiledFileSet) error {
	if len(bundle.TypesExport) == 0 {
		return nil
	}
	entries, err := modules_domain.DecodeTypesExport(bundle.TypesExport)
	if err != nil {
		return fmt.Errorf("interp_domain: LoadModule TypesExport decode: %w", err)
	}
	for _, entry := range entries {
		if err := s.bridgeBytecodeModule(entry, cfs); err != nil {
			return fmt.Errorf("interp_domain: LoadModule bridge %q: %w", entry.ImportPath, err)
		}
	}
	return nil
}

// consultModuleCapabilities walks the descriptor's capability set and asks the installed
// CapabilityHook to approve each claim.
// Returns the first denial; nil when the hook is unset or every claim passes.
//
// Takes ctx (context.Context) which is forwarded to the hook.
// Takes descriptor (*modules_domain.ModuleDescriptor) which supplies the claims.
//
// Returns the first denial error or nil.
func (s *Service) consultModuleCapabilities(ctx context.Context, descriptor *modules_domain.ModuleDescriptor) error {
	hook := s.limits.capabilityHook
	if hook == nil {
		return nil
	}
	for _, capability := range descriptor.Capabilities {
		path := "module/load:" + capability.Axis
		if err := hook.CheckFunctionCall(ctx, descriptor.Ref.Path, path, nil); err != nil {
			return fmt.Errorf("%w: %s: %w", modules_domain.ErrCapabilityDenied, capability.String(), err)
		}
	}
	return nil
}
