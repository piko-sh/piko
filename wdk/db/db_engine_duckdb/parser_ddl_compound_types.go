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

package db_engine_duckdb

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

// tryParseCompoundType dispatches struct, map, union, list, and array type parsers when
// lower names one of those compound forms.
//
// Takes engine (typeNormaliser) which resolves nested type names.
// Takes lower (string) which is the candidate compound type keyword lower-cased.
//
// Returns querier_dto.SQLType which is the parsed compound type when matched, else the
// zero value.
// Returns bool which is true when a compound type was parsed.
func (p *parser) tryParseCompoundType(engine typeNormaliser, lower string) (querier_dto.SQLType, bool) {
	switch lower {
	case "struct":
		return p.parseStructType(engine), true
	case "map":
		return p.parseMapType(engine), true
	case "union":
		return p.parseUnionType(engine), true
	case "list", "array":
		return p.parseListType(engine), true
	default:
		return querier_dto.SQLType{}, false
	}
}

// parseNamedTypeList parses a parenthesised list of "name type" pairs.
//
// Used by struct and union type parsers, which both expect parallel slices of names and
// SQL types.
//
// Takes engine (typeNormaliser) which resolves each field's type.
//
// Returns []string which is the parsed field name list.
// Returns []querier_dto.SQLType which is the parallel field type list.
func (p *parser) parseNamedTypeList(engine typeNormaliser) ([]string, []querier_dto.SQLType) {
	p.advance()

	var names []string
	var types []querier_dto.SQLType

	for !p.atEnd() && p.current().kind != tokenRightParen {
		name, nameError := p.parseIdentifierOrKeyword()
		if nameError != nil {
			break
		}
		sqlType, _ := p.parseColumnType(engine)
		names = append(names, name)
		types = append(types, sqlType)
		if p.current().kind == tokenComma {
			p.advance()
		}
	}
	if p.current().kind == tokenRightParen {
		p.advance()
	}

	return names, types
}

// parseStructType parses a STRUCT(...) compound type into its field list.
//
// Takes engine (typeNormaliser) which resolves each field's type.
//
// Returns querier_dto.SQLType which is the struct type with its fields populated.
func (p *parser) parseStructType(engine typeNormaliser) querier_dto.SQLType {
	names, types := p.parseNamedTypeList(engine)

	fields := make([]querier_dto.StructField, len(names))
	for index := range names {
		fields[index] = querier_dto.StructField{
			Name:    names[index],
			SQLType: types[index],
		}
	}

	structType := querier_dto.NewSQLType(querier_dto.TypeCategoryStruct, "struct")
	structType.StructFields = fields
	return structType
}

// parseMapType parses a MAP(key, value) compound type.
//
// Takes engine (typeNormaliser) which resolves the key and value types.
//
// Returns querier_dto.SQLType which is the map type with both type pointers populated.
func (p *parser) parseMapType(engine typeNormaliser) querier_dto.SQLType {
	p.advance()

	keyType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenComma {
		p.advance()
	}

	valueType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	mapType := querier_dto.NewSQLType(querier_dto.TypeCategoryMap, "map")
	mapType.KeyType = &keyType
	mapType.ElementType = &valueType
	return mapType
}

// parseUnionType parses a UNION(...) compound type into its tagged members.
//
// Takes engine (typeNormaliser) which resolves each member's type.
//
// Returns querier_dto.SQLType which is the union type with members populated.
func (p *parser) parseUnionType(engine typeNormaliser) querier_dto.SQLType {
	names, types := p.parseNamedTypeList(engine)

	members := make([]querier_dto.UnionMember, len(names))
	for index := range names {
		members[index] = querier_dto.UnionMember{
			Tag:     names[index],
			SQLType: types[index],
		}
	}

	unionType := querier_dto.NewSQLType(querier_dto.TypeCategoryUnion, "union")
	unionType.UnionMembers = members
	return unionType
}

// parseListType parses a LIST(element) or ARRAY(element) compound type.
//
// Takes engine (typeNormaliser) which resolves the element type.
//
// Returns querier_dto.SQLType which is the array type with its element type populated.
func (p *parser) parseListType(engine typeNormaliser) querier_dto.SQLType {
	p.advance()

	elementType, _ := p.parseColumnType(engine)

	if p.current().kind == tokenRightParen {
		p.advance()
	}

	listType := querier_dto.NewSQLType(querier_dto.TypeCategoryArray, "list")
	listType.ElementType = &elementType
	return listType
}
