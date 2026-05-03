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

package inspector_domain

import (
	"strings"
	"sync"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// ImplementationIndex maps interfaces to their implementing types.
// This enables "Go to Implementation" and "Find All Implementations" features.
type ImplementationIndex struct {
	// interfaceToImplementors maps interface key
	// (packagePath#typeName) to implementing types.
	interfaceToImplementors map[string][]ImplementorInfo

	// mu guards concurrent access to the index maps.
	mu sync.RWMutex
}

// ImplementorInfo describes a type that implements an interface.
type ImplementorInfo struct {
	// TypeName is the name of the implementing type.
	TypeName string

	// PackagePath is the import path of the package that contains the type.
	PackagePath string

	// DefinitionFile is the file path where the type is defined; empty if unknown.
	DefinitionFile string

	// DefinitionLine is the 1-based line number where the type is defined.
	DefinitionLine int

	// DefinitionCol is the column number where the type is defined.
	DefinitionCol int
}

// interfaceSpec holds the method signatures needed for implementation checking.
type interfaceSpec struct {
	// methods maps method names to their signature strings.
	methods map[string]string

	// name is the interface name being documented.
	name string

	// packagePath is the import path of the package containing the interface.
	packagePath string
}

// NewImplementationIndex builds an index from the given type data.
//
// Takes typeData (*inspector_dto.TypeData) which holds all type information.
//
// Returns *ImplementationIndex which maps interfaces to their implementors.
func NewImplementationIndex(typeData *inspector_dto.TypeData) *ImplementationIndex {
	index := &ImplementationIndex{
		interfaceToImplementors: make(map[string][]ImplementorInfo),
	}
	index.buildFromTypeData(typeData)
	return index
}

// FindImplementations returns all types that implement the given interface.
//
// Takes packagePath (string) which is the package path of the interface.
// Takes interfaceName (string) which is the name of the interface.
//
// Returns []ImplementorInfo which contains all implementing types.
//
// Safe for concurrent use.
func (idx *ImplementationIndex) FindImplementations(packagePath, interfaceName string) []ImplementorInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	key := packagePath + "#" + interfaceName
	return idx.interfaceToImplementors[key]
}

// buildFromTypeData populates the index by scanning all types.
//
// Takes typeData (*inspector_dto.TypeData) which contains the package and type
// information to scan.
func (idx *ImplementationIndex) buildFromTypeData(typeData *inspector_dto.TypeData) {
	if typeData == nil || typeData.Packages == nil {
		return
	}

	interfaces := idx.collectInterfaces(typeData)

	for packagePath, pkg := range typeData.Packages {
		if pkg == nil || pkg.NamedTypes == nil {
			continue
		}
		for typeName, typeInfo := range pkg.NamedTypes {
			if typeInfo == nil || isInterfaceType(typeInfo) {
				continue
			}
			idx.checkImplementations(packagePath, typeName, typeInfo, interfaces)
		}
	}
}

// collectInterfaces gathers all interface types from TypeData.
//
// Takes typeData (*inspector_dto.TypeData) which provides the parsed type
// information to scan for interfaces.
//
// Returns map[string]*interfaceSpec which maps qualified names (package path
// plus type name) to their interface specifications.
func (*ImplementationIndex) collectInterfaces(typeData *inspector_dto.TypeData) map[string]*interfaceSpec {
	interfaces := make(map[string]*interfaceSpec)
	for packagePath, pkg := range typeData.Packages {
		if pkg == nil || pkg.NamedTypes == nil {
			continue
		}
		for typeName, typeInfo := range pkg.NamedTypes {
			if typeInfo != nil && isInterfaceType(typeInfo) {
				key := packagePath + "#" + typeName
				interfaces[key] = &interfaceSpec{
					name:        typeName,
					packagePath: packagePath,
					methods:     extractMethodSigs(typeInfo.Methods),
				}
			}
		}
	}
	return interfaces
}

// checkImplementations tests if a type implements any of the known interfaces.
//
// Takes packagePath (string) which is the package path of the type being checked.
// Takes typeName (string) which is the name of the type being checked.
// Takes typeInfo (*inspector_dto.Type) which provides the type's method set.
// Takes interfaces (map[string]*interfaceSpec) which contains the interfaces
// to check against.
func (idx *ImplementationIndex) checkImplementations(
	packagePath, typeName string,
	typeInfo *inspector_dto.Type,
	interfaces map[string]*interfaceSpec,
) {
	typeMethods := extractMethodSigs(typeInfo.Methods)

	for ifaceKey, iface := range interfaces {
		if implementsInterface(typeMethods, iface.methods) {
			idx.interfaceToImplementors[ifaceKey] = append(
				idx.interfaceToImplementors[ifaceKey],
				ImplementorInfo{
					TypeName:       typeName,
					PackagePath:    packagePath,
					DefinitionFile: typeInfo.DefinedInFilePath,
					DefinitionLine: typeInfo.DefinitionLine,
					DefinitionCol:  typeInfo.DefinitionColumn,
				},
			)
		}
	}
}

// isInterfaceType checks if a type is an interface based on its
// UnderlyingTypeString.
//
// Takes typeInfo (*inspector_dto.Type) which provides the type information to
// check.
//
// Returns bool which is true if the type is an interface.
func isInterfaceType(typeInfo *inspector_dto.Type) bool {
	return strings.HasPrefix(typeInfo.UnderlyingTypeString, "interface")
}

// extractMethodSigs creates a map of method names to their normalised
// signatures.
//
// Takes methods ([]*inspector_dto.Method) which provides the methods
// to extract signatures from.
//
// Returns map[string]string which maps each method name to its
// signature string.
func extractMethodSigs(methods []*inspector_dto.Method) map[string]string {
	result := make(map[string]string, len(methods))
	for _, m := range methods {
		if m != nil && m.Name != "" {
			result[m.Name] = m.Signature.ToSignatureString()
		}
	}
	return result
}

// implementsInterface checks if typeMethods satisfy all interface methods.
//
// Takes typeMethods (map[string]string) which maps method names to signatures
// for the type being checked.
// Takes ifaceMethods (map[string]string) which maps method names to signatures
// for the interface to match against.
//
// Returns bool which is true if all interface methods are present with matching
// signatures, or false if the interface is empty or any method is missing or
// mismatched.
func implementsInterface(typeMethods, ifaceMethods map[string]string) bool {
	if len(ifaceMethods) == 0 {
		return false
	}
	for methodName, ifaceSig := range ifaceMethods {
		typeSig, ok := typeMethods[methodName]
		if !ok || typeSig != ifaceSig {
			return false
		}
	}
	return true
}
