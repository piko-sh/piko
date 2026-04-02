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

package linguistics_domain

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	stemmerTestSetup   sync.Once
	stemmerSuccessMock *MockStemmer
)

func setupStemmerTests() {
	stemmerTestSetup.Do(func() {
		RegisterStemmerFactory("stemmer-test-error-lang", func() (StemmerPort, error) {
			return nil, errors.New("factory failed")
		})

		stemmerSuccessMock = &MockStemmer{FixedLanguage: "stemmer-test-success-lang"}
		RegisterStemmerFactory("stemmer-test-success-lang", func() (StemmerPort, error) {
			return stemmerSuccessMock, nil
		})

		RegisterStemmerFactory("stemmer-test-list-lang", func() (StemmerPort, error) {
			return NewNoOpStemmer("stemmer-test-list-lang"), nil
		})
	})
}

func TestCreateStemmer_UnregisteredReturnsNoOp(t *testing.T) {
	stemmer := CreateStemmer("unregistered-stemmer-lang-xyz")

	_, isNoOp := stemmer.(*NoOpStemmer)
	assert.True(t, isNoOp, "should return NoOpStemmer for unregistered language")
	assert.Equal(t, "unregistered-stemmer-lang-xyz", stemmer.GetLanguage())
}

func TestCreateStemmer_FactoryErrorReturnsNoOp(t *testing.T) {
	setupStemmerTests()

	stemmer := CreateStemmer("stemmer-test-error-lang")

	_, isNoOp := stemmer.(*NoOpStemmer)
	assert.True(t, isNoOp, "should return NoOpStemmer when factory errors")
}

func TestCreateStemmer_FactorySuccess(t *testing.T) {
	setupStemmerTests()

	stemmer := CreateStemmer("stemmer-test-success-lang")
	assert.Equal(t, stemmerSuccessMock, stemmer)
}

func TestRegisteredStemmerFactories_IncludesRegistered(t *testing.T) {
	setupStemmerTests()

	names := RegisteredStemmerFactories()
	assert.Contains(t, names, "stemmer-test-list-lang")
}

func TestNoOpStemmer_StemReturnsUnchanged(t *testing.T) {
	s := NewNoOpStemmer("english")

	assert.Equal(t, "running", s.Stem("running"))
	assert.Equal(t, "cats", s.Stem("cats"))
	assert.Equal(t, "", s.Stem(""))
}

func TestNoOpStemmer_GetLanguage(t *testing.T) {
	s := NewNoOpStemmer("French")
	assert.Equal(t, "french", s.GetLanguage())
}

func TestNoOpStemmer_EmptyLanguageDefaultsToEnglish(t *testing.T) {
	s := NewNoOpStemmer("")
	assert.Equal(t, LanguageEnglish, s.GetLanguage())
}

func TestNoOpStemmer_ImplementsStemmerPort(t *testing.T) {
	var _ StemmerPort = (*NoOpStemmer)(nil)

	s := NewNoOpStemmer("english")
	require.NotNil(t, s)
}
