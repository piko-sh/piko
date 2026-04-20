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

package linguistics_stemmer_hebrew

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

func TestNew(t *testing.T) {
	t.Parallel()

	stemmer, err := New()
	require.NoError(t, err)
	require.NotNil(t, stemmer)
}

func TestFactory(t *testing.T) {
	t.Parallel()

	stemmer, err := Factory()
	require.NoError(t, err)
	require.NotNil(t, stemmer)

	var _ linguistics_domain.StemmerPort = stemmer
}

func TestStemmer_GetLanguage(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	assert.Equal(t, Language, stemmer.GetLanguage())
	assert.Equal(t, "hebrew", stemmer.GetLanguage())
}

func TestStemmer_Stem(t *testing.T) {
	t.Parallel()

	stemmer := &Stemmer{}
	assert.Empty(t, stemmer.Stem(""))
	assert.Equal(t, "בנק", stemmer.Stem("הבנק"))
	assert.Equal(t, "ילד", stemmer.Stem("שהילדים"))
}

func TestStemmer_RegisteredViaInit(t *testing.T) {
	t.Parallel()

	port := linguistics_domain.CreateStemmer(Language)
	require.NotNil(t, port)
	assert.Equal(t, "hebrew", port.GetLanguage())
}
