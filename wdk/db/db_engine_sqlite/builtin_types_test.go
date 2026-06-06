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

package db_engine_sqlite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestNormaliseByAffinityRetainsTextLengthModifier(t *testing.T) {
	t.Parallel()

	result := normaliseTypeName("VARCHAR2", 50)
	assert.Equal(t, querier_dto.TypeCategoryText, result.Category)
	require.NotNil(t, result.Length, "affinity-resolved text type dropped its length modifier")
	assert.Equal(t, 50, *result.Length)
}

func TestNormaliseByAffinityDecimalKeepsPrecisionAndScale(t *testing.T) {
	t.Parallel()

	result := normaliseTypeName("DECIMAL_LIKE", 10, 2)
	assert.Equal(t, querier_dto.TypeCategoryDecimal, result.Category)
	require.NotNil(t, result.Precision)
	assert.Equal(t, 10, *result.Precision)
	require.NotNil(t, result.Scale)
	assert.Equal(t, 2, *result.Scale)
}
