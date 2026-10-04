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
package db_catalogue_postgres

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

type modifierTypeNormaliser struct{}

func (modifierTypeNormaliser) NormaliseTypeName(name string, modifiers ...int) querier_dto.SQLType {
	if len(modifiers) > 0 {
		name = fmt.Sprintf("%s%v", name, modifiers)
	}
	return querier_dto.NewSQLType(querier_dto.TypeCategoryUnknown, name)
}

func newTestSchema(name string) *querier_dto.Schema {
	return &querier_dto.Schema{
		Name:           name,
		Tables:         make(map[string]*querier_dto.Table),
		Views:          make(map[string]*querier_dto.View),
		Enums:          make(map[string]*querier_dto.Enum),
		Functions:      make(map[string][]*querier_dto.FunctionSignature),
		CompositeTypes: make(map[string]*querier_dto.CompositeType),
		Sequences:      make(map[string]*querier_dto.Sequence),
	}
}
