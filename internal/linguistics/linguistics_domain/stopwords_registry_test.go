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
)

type mockStopWordsProvider struct {
	stopWords          map[string]bool
	supportedLanguages []string
}

func (m *mockStopWordsProvider) GetStopWords(_ string) map[string]bool {
	return m.stopWords
}

func (m *mockStopWordsProvider) SupportedLanguages() []string {
	return m.supportedLanguages
}

var (
	stopwordsTestSetup   sync.Once
	stopwordsSuccessMock *mockStopWordsProvider
)

func setupStopwordsTests() {
	stopwordsTestSetup.Do(func() {
		RegisterStopWordsProviderFactory("stopwords-test-error-lang", func() (StopWordsProviderPort, error) {
			return nil, errors.New("factory failed")
		})

		stopwordsSuccessMock = &mockStopWordsProvider{
			stopWords:          map[string]bool{"the": true, "a": true},
			supportedLanguages: []string{"stopwords-test-success-lang"},
		}
		RegisterStopWordsProviderFactory("stopwords-test-success-lang", func() (StopWordsProviderPort, error) {
			return stopwordsSuccessMock, nil
		})

		RegisterStopWordsProviderFactory("stopwords-test-list-lang", func() (StopWordsProviderPort, error) {
			return &mockStopWordsProvider{}, nil
		})
	})
}

func TestCreateStopWordsProvider_UnregisteredReturnsNoOp(t *testing.T) {
	provider := CreateStopWordsProvider("unregistered-stopwords-lang-xyz")

	_, isNoOp := provider.(*NoOpStopWordsProvider)
	assert.True(t, isNoOp, "should return NoOpStopWordsProvider for unregistered language")
}

func TestCreateStopWordsProvider_FactoryErrorReturnsNoOp(t *testing.T) {
	setupStopwordsTests()

	provider := CreateStopWordsProvider("stopwords-test-error-lang")

	_, isNoOp := provider.(*NoOpStopWordsProvider)
	assert.True(t, isNoOp, "should return NoOpStopWordsProvider when factory errors")
}

func TestCreateStopWordsProvider_FactorySuccess(t *testing.T) {
	setupStopwordsTests()

	provider := CreateStopWordsProvider("stopwords-test-success-lang")
	assert.Equal(t, stopwordsSuccessMock, provider)
}

func TestRegisteredStopWordsProviderFactories_IncludesRegistered(t *testing.T) {
	setupStopwordsTests()

	names := RegisteredStopWordsProviderFactories()
	assert.Contains(t, names, "stopwords-test-list-lang")
}

func TestNoOpStopWordsProvider_GetStopWordsReturnsEmpty(t *testing.T) {
	p := NewNoOpStopWordsProvider()

	result := p.GetStopWords("english")
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestNoOpStopWordsProvider_SupportedLanguagesReturnsEmpty(t *testing.T) {
	p := NewNoOpStopWordsProvider()

	langs := p.SupportedLanguages()
	assert.NotNil(t, langs)
	assert.Empty(t, langs)
}

func TestNoOpStopWordsProvider_ImplementsStopWordsProviderPort(t *testing.T) {
	var _ StopWordsProviderPort = (*NoOpStopWordsProvider)(nil)
}
