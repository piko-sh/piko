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

package transformer_crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/storage/storage_dto"
)

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	ct := New(nil, "", 0)
	require.NotNil(t, ct)
	assert.Equal(t, "crypto-service", ct.Name())
	assert.Equal(t, 250, ct.Priority())
	assert.Equal(t, storage_dto.TransformerEncryption, ct.Type())
}

func TestNew_CustomValues(t *testing.T) {
	t.Parallel()

	ct := New(nil, "my-crypto", 100)
	require.NotNil(t, ct)
	assert.Equal(t, "my-crypto", ct.Name())
	assert.Equal(t, 100, ct.Priority())
	assert.Equal(t, storage_dto.TransformerEncryption, ct.Type())
}
