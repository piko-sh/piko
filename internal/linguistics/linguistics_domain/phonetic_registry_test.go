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
	phoneticTestSetup   sync.Once
	phoneticSuccessMock *MockPhoneticEncoder
)

func setupPhoneticTests() {
	phoneticTestSetup.Do(func() {
		RegisterPhoneticEncoderFactory("phonetic-test-error-lang", func() (PhoneticEncoderPort, error) {
			return nil, errors.New("factory failed")
		})

		phoneticSuccessMock = &MockPhoneticEncoder{FixedLanguage: "phonetic-test-success-lang"}
		RegisterPhoneticEncoderFactory("phonetic-test-success-lang", func() (PhoneticEncoderPort, error) {
			return phoneticSuccessMock, nil
		})

		RegisterPhoneticEncoderFactory("phonetic-test-list-lang", func() (PhoneticEncoderPort, error) {
			return NewNoOpPhoneticEncoder("phonetic-test-list-lang"), nil
		})
	})
}

func TestCreatePhoneticEncoder_UnregisteredReturnsNoOp(t *testing.T) {
	encoder := CreatePhoneticEncoder("unregistered-phonetic-lang-xyz")

	_, isNoOp := encoder.(*NoOpPhoneticEncoder)
	assert.True(t, isNoOp, "should return NoOpPhoneticEncoder for unregistered language")
	assert.Equal(t, "unregistered-phonetic-lang-xyz", encoder.GetLanguage())
}

func TestCreatePhoneticEncoder_FactoryErrorReturnsNoOp(t *testing.T) {
	setupPhoneticTests()

	encoder := CreatePhoneticEncoder("phonetic-test-error-lang")

	_, isNoOp := encoder.(*NoOpPhoneticEncoder)
	assert.True(t, isNoOp, "should return NoOpPhoneticEncoder when factory errors")
}

func TestCreatePhoneticEncoder_FactorySuccess(t *testing.T) {
	setupPhoneticTests()

	encoder := CreatePhoneticEncoder("phonetic-test-success-lang")
	assert.Equal(t, phoneticSuccessMock, encoder)
}

func TestRegisteredPhoneticEncoderFactories_IncludesRegistered(t *testing.T) {
	setupPhoneticTests()

	names := RegisteredPhoneticEncoderFactories()
	assert.Contains(t, names, "phonetic-test-list-lang")
}

func TestNoOpPhoneticEncoder_EncodeReturnsEmpty(t *testing.T) {
	e := NewNoOpPhoneticEncoder("english")

	assert.Equal(t, "", e.Encode("hello"))
	assert.Equal(t, "", e.Encode("world"))
	assert.Equal(t, "", e.Encode(""))
}

func TestNoOpPhoneticEncoder_GetLanguage(t *testing.T) {
	e := NewNoOpPhoneticEncoder("German")
	assert.Equal(t, "german", e.GetLanguage())
}

func TestNoOpPhoneticEncoder_EmptyLanguageDefaultsToEnglish(t *testing.T) {
	e := NewNoOpPhoneticEncoder("")
	assert.Equal(t, LanguageEnglish, e.GetLanguage())
}

func TestNoOpPhoneticEncoder_ImplementsPhoneticEncoderPort(t *testing.T) {
	var _ PhoneticEncoderPort = (*NoOpPhoneticEncoder)(nil)

	e := NewNoOpPhoneticEncoder("english")
	require.NotNil(t, e)
}
