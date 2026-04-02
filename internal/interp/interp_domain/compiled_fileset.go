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

// CompiledFileSet holds the compiled output of one or more Go source
// files that belong to the same package. Functions from all files share
// a unified function table and can call each other.
type CompiledFileSet struct {
	// root is a synthetic container whose Functions slice holds all
	// compiled functions from every source file.
	root *CompiledFunction

	// variableInitFunction holds bytecode for package-level variable
	// initialisers. Executed before init() functions.
	variableInitFunction *CompiledFunction

	// entrypoints maps function names to their indices in
	// root.functions. All non-init functions are included.
	entrypoints map[string]uint16

	// initFunctionIndices holds the indices of init() functions in
	// root.functions, in source order.
	initFunctionIndices []uint16
}

// FindFunction looks up a function by name in the compiled file set.
//
// Takes name (string) which is the function name to find.
//
// Returns *CompiledFunction and nil error if found, or nil and an
// error if not found.
func (cfs *CompiledFileSet) FindFunction(name string) (*CompiledFunction, error) {
	index, ok := cfs.entrypoints[name]
	if !ok {
		return nil, errEntrypointNotFound
	}
	return cfs.root.functions[index], nil
}
