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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithStemmer_SetsOption(t *testing.T) {
	stemmer := NewNoOpStemmer("english")
	opt := WithStemmer(stemmer)

	var opts analyserOptions
	opt(&opts)

	assert.Equal(t, stemmer, opts.stemmer)
}

func TestWithTokeniser_SetsOption(t *testing.T) {
	tokeniser := &mockTokeniser{}
	opt := WithTokeniser(tokeniser)

	var opts analyserOptions
	opt(&opts)

	assert.Equal(t, tokeniser, opts.tokeniser)
}

func TestWithPhoneticEncoder_SetsOption(t *testing.T) {
	encoder := NewNoOpPhoneticEncoder("english")
	opt := WithPhoneticEncoder(encoder)

	var opts analyserOptions
	opt(&opts)

	assert.Equal(t, encoder, opts.phonetic)
}

func TestWithStopWordsProvider_SetsOption(t *testing.T) {
	provider := NewNoOpStopWordsProvider()
	opt := WithStopWordsProvider(provider)

	var opts analyserOptions
	opt(&opts)

	assert.Equal(t, provider, opts.stopWordsProvider)
}

func TestWithLanguage_SetsAllComponents(t *testing.T) {
	opt := WithLanguage("english")

	var opts analyserOptions
	opt(&opts)

	assert.NotNil(t, opts.stemmer, "stemmer should be set")
	assert.NotNil(t, opts.phonetic, "phonetic encoder should be set")
	assert.NotNil(t, opts.stopWordsProvider, "stop words provider should be set")
}

func TestWithLanguage_UsesNoOpForUnregistered(t *testing.T) {
	opt := WithLanguage("option-test-unregistered-xyz")

	var opts analyserOptions
	opt(&opts)

	_, isStemmerNoOp := opts.stemmer.(*NoOpStemmer)
	assert.True(t, isStemmerNoOp, "should use NoOp stemmer for unregistered language")

	_, isPhoneticNoOp := opts.phonetic.(*NoOpPhoneticEncoder)
	assert.True(t, isPhoneticNoOp, "should use NoOp phonetic encoder for unregistered language")

	_, isStopWordsNoOp := opts.stopWordsProvider.(*NoOpStopWordsProvider)
	assert.True(t, isStopWordsNoOp, "should use NoOp stop words provider for unregistered language")
}
