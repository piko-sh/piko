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

package engine_shared

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// Arg names a single function argument and its SQL type.
//
// It is the typed unit accepted by Args, so a malformed argument is a compile error
// rather than a value silently dropped at registration time. Engines pass their own
// dialect SQLType values, keeping their type vocabulary engine-local while the
// registration mechanism is shared.
type Arg struct {
	// Name is the argument name.
	Name string

	// Type is the argument SQL type.
	Type querier_dto.SQLType
}

// CatalogueBuilder accumulates FunctionSignatures into a FunctionCatalogue.
//
// It carries no dialect-specific type vocabulary: engines embed it and pass
// querier_dto.SQLType values, so one builder serves every dialect whose named type fields
// differ. An engine embeds it to add its own type fields and registration groupings.
type CatalogueBuilder struct {
	// Catalogue is the function catalogue being assembled.
	Catalogue *querier_dto.FunctionCatalogue
}

// NewCatalogueBuilder returns a CatalogueBuilder with an empty catalogue ready to
// register function signatures into.
//
// Returns *CatalogueBuilder which is ready for registration calls.
func NewCatalogueBuilder() *CatalogueBuilder {
	return &CatalogueBuilder{
		Catalogue: &querier_dto.FunctionCatalogue{
			Functions: make(map[string][]*querier_dto.FunctionSignature),
		},
	}
}

// Args builds a slice of FunctionArgument from the supplied typed arguments, preserving
// declaration order.
//
// Takes arguments (...Arg) which names each positional argument together with its SQL
// type.
//
// Returns []querier_dto.FunctionArgument which lists the arguments in order.
func (*CatalogueBuilder) Args(arguments ...Arg) []querier_dto.FunctionArgument {
	result := make([]querier_dto.FunctionArgument, 0, len(arguments))
	for index := range arguments {
		result = append(result, querier_dto.FunctionArgument{Name: arguments[index].Name, Type: arguments[index].Type})
	}
	return result
}

// Add registers a function signature under the given name and returns it so the caller
// can refine fields (for example MinArguments) on the registered value. The signature is
// marked read-only, the canonical data access of every built-in function.
//
// Takes name (string) which is the function name to register under.
// Takes signature (*querier_dto.FunctionSignature) which describes the function.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func (b *CatalogueBuilder) Add(name string, signature *querier_dto.FunctionSignature) *querier_dto.FunctionSignature {
	signature.Name = name
	signature.DataAccess = querier_dto.DataAccessReadOnly
	b.Catalogue.Functions[name] = append(b.Catalogue.Functions[name], signature)
	return signature
}

// NullOnNull registers a function that returns NULL when any argument is NULL.
//
// Takes name (string) which is the function name to register.
// Takes arguments ([]querier_dto.FunctionArgument) which describes the positional
// arguments.
// Takes returnType (querier_dto.SQLType) which is the result type when all arguments are
// non-null.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func (b *CatalogueBuilder) NullOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) *querier_dto.FunctionSignature {
	return b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableReturnsNullOnNull,
	})
}

// NeverNull registers a function that never returns NULL.
//
// Takes name (string) which is the function name to register.
// Takes arguments ([]querier_dto.FunctionArgument) which describes the positional
// arguments.
// Takes returnType (querier_dto.SQLType) which is the result type.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func (b *CatalogueBuilder) NeverNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) *querier_dto.FunctionSignature {
	return b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
}

// CalledOnNull registers a function that is invoked even with NULL arguments; the result
// may or may not be NULL depending on the function.
//
// Takes name (string) which is the function name to register.
// Takes arguments ([]querier_dto.FunctionArgument) which describes the positional
// arguments.
// Takes returnType (querier_dto.SQLType) which is the result type.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func (b *CatalogueBuilder) CalledOnNull(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) *querier_dto.FunctionSignature {
	return b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableCalledOnNull,
	})
}

// ModifiesData registers a built-in that may modify database state.
//
// Such functions include a sequence advance like nextval or setval. It is registered as
// never-null, and its data access is set to DataAccessModifiesData, overriding the
// read-only default that Add applies, so a query projecting it is classified as
// data-modifying and routed to the writer connection rather than a read replica.
//
// Takes name (string) which is the function name to register.
// Takes arguments ([]querier_dto.FunctionArgument) which describes the positional
// arguments.
// Takes returnType (querier_dto.SQLType) which is the result type.
//
// Returns *querier_dto.FunctionSignature which is the registered signature.
func (b *CatalogueBuilder) ModifiesData(name string, arguments []querier_dto.FunctionArgument, returnType querier_dto.SQLType) *querier_dto.FunctionSignature {
	signature := b.Add(name, &querier_dto.FunctionSignature{
		Arguments:         arguments,
		ReturnType:        returnType,
		NullableBehaviour: querier_dto.FunctionNullableNeverNull,
	})
	signature.DataAccess = querier_dto.DataAccessModifiesData
	return signature
}
