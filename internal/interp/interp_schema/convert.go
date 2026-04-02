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

package interp_schema

import (
	"errors"

	"piko.sh/piko/internal/interp/interp_schema/interp_schema_gen"
)

var registerBankNames = [...]string{
	"int", "float", "string", "general", "bool", "uint", "complex",
}

// BytecodeInspection is a JSON-serialisable summary of a compiled
// bytecode file set.
type BytecodeInspection struct {
	Root *FunctionInspection `json:"root"`

	VarInit *FunctionInspection `json:"var_init,omitempty"`

	Entrypoints map[string]uint16 `json:"entrypoints"`

	InitFunctions []uint16 `json:"init_functions,omitempty"`
}

// FunctionInspection is a JSON-serialisable summary of a single
// compiled function.
type FunctionInspection struct {
	NumRegisters map[string]uint32 `json:"num_registers"`

	Constants map[string]int `json:"constants"`

	Name string `json:"name"`

	SourceFile string `json:"source_file,omitempty"`

	Functions []*FunctionInspection `json:"functions,omitempty"`

	Instructions int `json:"instructions"`

	CallSites int `json:"call_sites"`

	Upvalues int `json:"upvalues"`

	IsVariadic bool `json:"is_variadic,omitempty"`
}

// ConvertBytecode reads a raw FlatBuffer payload and returns a
// JSON-serialisable inspection summary.
//
// Takes payload ([]byte) which is the raw FlatBuffer bytes (after
// version header has been stripped by Unpack).
//
// Returns *BytecodeInspection which holds the structural metadata.
// Returns error when the payload is empty or malformed.
func ConvertBytecode(payload []byte) (*BytecodeInspection, error) {
	if len(payload) == 0 {
		return nil, errors.New("empty bytecode payload")
	}

	fileSet := interp_schema_gen.GetRootAsCompiledFileSet(payload, 0)

	inspection := &BytecodeInspection{
		Entrypoints: make(map[string]uint16),
	}

	var rootFunction interp_schema_gen.CompiledFunction
	if fileSet.Root(&rootFunction) != nil {
		inspection.Root = convertFunction(&rootFunction)
	}

	var varInitFunction interp_schema_gen.CompiledFunction
	if fileSet.VariableInitFunction(&varInitFunction) != nil {
		inspection.VarInit = convertFunction(&varInitFunction)
	}

	var entrypoint interp_schema_gen.EntrypointEntry
	for i := range fileSet.EntrypointsLength() {
		if fileSet.Entrypoints(&entrypoint, i) {
			inspection.Entrypoints[string(entrypoint.Name())] = entrypoint.FunctionIndex()
		}
	}

	for i := range fileSet.InitialisationFunctionsLength() {
		inspection.InitFunctions = append(inspection.InitFunctions, fileSet.InitialisationFunctions(i))
	}

	return inspection, nil
}

func convertFunction(function *interp_schema_gen.CompiledFunction) *FunctionInspection {
	inspection := &FunctionInspection{
		Name:         string(function.Name()),
		SourceFile:   string(function.SourceFile()),
		NumRegisters: make(map[string]uint32),
		Instructions: function.BodyLength(),
		Constants:    make(map[string]int),
		CallSites:    function.CallSitesLength(),
		Upvalues:     function.UpvalueDescriptorsLength(),
		IsVariadic:   function.IsVariadic(),
	}

	for i := range function.RegisterCountsLength() {
		count := function.RegisterCounts(i)
		if count > 0 && i < len(registerBankNames) {
			inspection.NumRegisters[registerBankNames[i]] = count
		}
	}

	addConstantCount(inspection.Constants, "int", function.IntConstantsLength())
	addConstantCount(inspection.Constants, "float", function.FloatConstantsLength())
	addConstantCount(inspection.Constants, "string", function.StringConstantsLength())
	addConstantCount(inspection.Constants, "bool", function.BoolConstantsLength())
	addConstantCount(inspection.Constants, "uint", function.UintConstantsLength())
	addConstantCount(inspection.Constants, "complex", function.ComplexConstantsLength())
	addConstantCount(inspection.Constants, "general", function.GeneralConstantDescriptorsLength())

	var childFunction interp_schema_gen.CompiledFunction
	for i := range function.FunctionsLength() {
		if function.Functions(&childFunction, i) {
			inspection.Functions = append(inspection.Functions, convertFunction(&childFunction))
		}
	}

	return inspection
}

func addConstantCount(constants map[string]int, name string, count int) {
	if count > 0 {
		constants[name] = count
	}
}
