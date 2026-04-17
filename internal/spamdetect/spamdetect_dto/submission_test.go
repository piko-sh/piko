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

package spamdetect_dto

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSubmission_Sanitise_TruncatesFields(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message").MaxLength(100))
	submission := &Submission{
		Fields: map[string]FieldValue{"message": {Value: strings.Repeat("a", 200)}},
	}
	submission.Sanitise(schema)
	assert.Equal(t, 100, len(submission.Fields["message"].Value))
	assert.True(t, submission.WasTruncated())
}

func TestSubmission_Sanitise_PreservesShortFields(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message"))
	submission := &Submission{
		Fields: map[string]FieldValue{"message": {Value: "Hello"}},
	}
	submission.Sanitise(schema)
	assert.Equal(t, "Hello", submission.Fields["message"].Value)
	assert.False(t, submission.WasTruncated())
}

func TestSubmission_Sanitise_UTF8Safety(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("name").MaxLength(10))
	submission := &Submission{
		Fields: map[string]FieldValue{"name": {Value: strings.Repeat("\U0001F600", 5)}},
	}
	submission.Sanitise(schema)
	assert.True(t, utf8.ValidString(submission.Fields["name"].Value))
	assert.LessOrEqual(t, len(submission.Fields["name"].Value), 10)
}

func TestSubmission_Sanitise_MetadataFields(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		RemoteIP:  strings.Repeat("x", 500),
		UserAgent: strings.Repeat("x", 2000),
		PageURL:   strings.Repeat("x", 5000),
	}
	submission.Sanitise(nil)
	assert.Equal(t, maxRemoteIPLength, len(submission.RemoteIP))
	assert.Equal(t, maxUserAgentLength, len(submission.UserAgent))
	assert.Equal(t, maxPageURLLength, len(submission.PageURL))
}

func TestSubmission_FieldString(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Fields: map[string]FieldValue{"message": {Value: "hello"}},
	}
	assert.Equal(t, "hello", submission.FieldString("message"))
	assert.Equal(t, "", submission.FieldString("nonexistent"))
}

func TestSubmission_FieldString_NilFields(t *testing.T) {
	t.Parallel()
	submission := &Submission{}
	assert.Equal(t, "", submission.FieldString("anything"))
}

func TestTruncateStringUTF8(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		maximum  int
	}{
		{name: "under limit", input: "hello", maximum: 10, expected: "hello"},
		{name: "at limit", input: "hello", maximum: 5, expected: "hello"},
		{name: "over limit", input: "hello world", maximum: 5, expected: "hello"},
		{name: "empty", input: "", maximum: 10, expected: ""},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := truncateStringUTF8(testCase.input, testCase.maximum)
			assert.Equal(t, testCase.expected, result)
			assert.True(t, utf8.ValidString(result))
		})
	}
}

func TestTruncateStringUTF8_MultiByteBoundary(t *testing.T) {
	t.Parallel()
	input := "\u4e16"
	result := truncateStringUTF8(input, 2)
	assert.True(t, utf8.ValidString(result))
	assert.Equal(t, "", result)
}

func TestSignal_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "gibberish", SignalGibberish.String())
	assert.Equal(t, "link_density", SignalLinkDensity.String())
}

func TestSpamDetectError(t *testing.T) {
	t.Parallel()
	underlying := assert.AnError
	spamErr := NewSpamDetectError("analyse", "test_detector", underlying)

	assert.Contains(t, spamErr.Error(), "analyse")
	assert.Contains(t, spamErr.Error(), "test_detector")
	assert.ErrorIs(t, spamErr, underlying)
}

func TestDefaultServiceConfig(t *testing.T) {
	t.Parallel()
	config := DefaultServiceConfig()
	assert.Equal(t, 0.7, config.ScoreThreshold)
	assert.Greater(t, config.Timeout.Milliseconds(), int64(0))
}

func TestSubmission_GetField_ReturnsFieldValueWithType(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Fields: map[string]FieldValue{
			"email": {Value: "test@example.com", Type: FieldTypeEmail},
			"name":  {Value: "John", Type: FieldTypeName},
		},
	}

	emailField := submission.GetField("email")
	assert.Equal(t, "test@example.com", emailField.Value)
	assert.Equal(t, FieldTypeEmail, emailField.Type)

	nameField := submission.GetField("name")
	assert.Equal(t, "John", nameField.Value)
	assert.Equal(t, FieldTypeName, nameField.Type)

	missingField := submission.GetField("nonexistent")
	assert.Equal(t, "", missingField.Value)
	assert.Equal(t, FieldType(""), missingField.Type)
}

func TestSubmission_GetField_NilFields(t *testing.T) {
	t.Parallel()
	submission := &Submission{}

	result := submission.GetField("anything")
	assert.Equal(t, FieldValue{}, result)
}

func TestSubmission_MetadataValue_Found(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Metadata: map[string]string{
			"site":      "example.com",
			"form_type": "contact",
		},
	}

	assert.Equal(t, "example.com", submission.MetadataValue("site"))
	assert.Equal(t, "contact", submission.MetadataValue("form_type"))
}

func TestSubmission_MetadataValue_NotFound(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Metadata: map[string]string{"site": "example.com"},
	}
	assert.Equal(t, "", submission.MetadataValue("missing_key"))
}

func TestSubmission_MetadataValue_NilMetadata(t *testing.T) {
	t.Parallel()
	submission := &Submission{}
	assert.Equal(t, "", submission.MetadataValue("anything"))
}

func TestSubmission_HeaderValue_Found(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Headers: map[string]string{
			"Accept-Language": "en-GB",
			"X-Forwarded-For": "10.0.0.1",
		},
	}

	assert.Equal(t, "en-GB", submission.HeaderValue("Accept-Language"))
	assert.Equal(t, "10.0.0.1", submission.HeaderValue("X-Forwarded-For"))
}

func TestSubmission_HeaderValue_NotFound(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Headers: map[string]string{"Accept-Language": "en-GB"},
	}
	assert.Equal(t, "", submission.HeaderValue("X-Missing"))
}

func TestSubmission_HeaderValue_NilHeaders(t *testing.T) {
	t.Parallel()
	submission := &Submission{}
	assert.Equal(t, "", submission.HeaderValue("anything"))
}

func TestSubmission_SanitiseMap_OversizedEvictsAlphabetically(t *testing.T) {
	t.Parallel()

	oversizedMap := make(map[string]string, maxMetadataEntries+10)
	for index := range maxMetadataEntries + 10 {

		key := fmt.Sprintf("key_%03d", index)
		oversizedMap[key] = "value"
	}

	submission := &Submission{Metadata: oversizedMap}
	submission.Sanitise(nil)

	assert.LessOrEqual(t, len(submission.Metadata), maxMetadataEntries)

	for index := range maxMetadataEntries {
		key := fmt.Sprintf("key_%03d", index)
		assert.Contains(t, submission.Metadata, key)
	}

	for index := maxMetadataEntries; index < maxMetadataEntries+10; index++ {
		key := fmt.Sprintf("key_%03d", index)
		assert.NotContains(t, submission.Metadata, key)
	}
}

func TestSubmission_SanitiseMap_KeyTruncation(t *testing.T) {
	t.Parallel()

	longKey := strings.Repeat("k", maxMetadataValueLen+1)
	submission := &Submission{
		Metadata: map[string]string{
			longKey:  "value",
			"normal": "ok",
		},
	}
	submission.Sanitise(nil)

	assert.NotContains(t, submission.Metadata, longKey)
	assert.Contains(t, submission.Metadata, "normal")
}

func TestSubmission_SanitiseMap_ValueTruncationSetsTruncated(t *testing.T) {
	t.Parallel()
	longValue := strings.Repeat("v", maxMetadataValueLen+100)
	submission := &Submission{
		Metadata: map[string]string{
			"key": longValue,
		},
	}
	submission.Sanitise(nil)

	assert.LessOrEqual(t, len(submission.Metadata["key"]), maxMetadataValueLen)
	assert.True(t, submission.WasTruncated())
}

func TestSubmission_Sanitise_WithMetadataAndHeaders(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message").MaxLength(50))
	submission := &Submission{
		Fields: map[string]FieldValue{
			"message": {Value: strings.Repeat("m", 100), Type: FieldTypeText},
		},
		Metadata: map[string]string{
			"site": strings.Repeat("s", maxMetadataValueLen+1),
		},
		Headers: map[string]string{
			"Accept-Language": strings.Repeat("h", maxHeaderValueLen+1),
		},
	}

	submission.Sanitise(schema)

	assert.LessOrEqual(t, len(submission.Fields["message"].Value), 50)
	assert.LessOrEqual(t, len(submission.Metadata["site"]), maxMetadataValueLen)
	assert.LessOrEqual(t, len(submission.Headers["Accept-Language"]), maxHeaderValueLen)
	assert.True(t, submission.WasTruncated())
}

func TestSubmission_Sanitise_NilSchema(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		Fields: map[string]FieldValue{
			"message": {Value: "hello", Type: FieldTypeText},
		},
		RemoteIP:  strings.Repeat("x", 500),
		UserAgent: strings.Repeat("x", 2000),
	}

	submission.Sanitise(nil)

	assert.Equal(t, maxRemoteIPLength, len(submission.RemoteIP))
	assert.Equal(t, maxUserAgentLength, len(submission.UserAgent))

	assert.Equal(t, "hello", submission.Fields["message"].Value)
}

func TestSubmission_Sanitise_RemovesFieldsNotInSchema(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message"))
	submission := &Submission{
		Fields: map[string]FieldValue{
			"message": {Value: "hello", Type: FieldTypeText},
			"extra":   {Value: "should be removed", Type: FieldTypeText},
		},
	}

	submission.Sanitise(schema)

	assert.Contains(t, submission.Fields, "message")
	assert.NotContains(t, submission.Fields, "extra")
}

func TestSubmission_SanitiseHeaders_OversizedEvicts(t *testing.T) {
	t.Parallel()
	oversizedHeaders := make(map[string]string, maxHeaderEntries+5)
	for index := range maxHeaderEntries + 5 {
		key := fmt.Sprintf("Header-%03d", index)
		oversizedHeaders[key] = "value"
	}

	submission := &Submission{Headers: oversizedHeaders}
	submission.Sanitise(nil)

	assert.LessOrEqual(t, len(submission.Headers), maxHeaderEntries)
}

func TestMaxDetectorCount(t *testing.T) {
	t.Parallel()
	assert.Greater(t, MaxDetectorCount(), 0)
	assert.Equal(t, 64, MaxDetectorCount())
}

func TestFieldValue_ZeroValue(t *testing.T) {
	t.Parallel()
	var fieldValue FieldValue
	assert.Equal(t, "", fieldValue.Value)
	assert.Equal(t, FieldType(""), fieldValue.Type)
}

func TestSpamDetectError_Unwrap(t *testing.T) {
	t.Parallel()
	underlying := assert.AnError
	spamErr := NewSpamDetectError("analyse", "test_detector", underlying)
	assert.Equal(t, underlying, spamErr.Unwrap())
}

func TestSignal_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "gibberish", SignalGibberish.String())
	assert.Equal(t, "link_density", SignalLinkDensity.String())
	assert.Equal(t, "blocklist", SignalBlocklist.String())
	assert.Equal(t, "honeypot", SignalHoneypot.String())
	assert.Equal(t, "timing", SignalTiming.String())
	assert.Equal(t, "repetition", SignalRepetition.String())
}

func TestDetectorPriority_Values(t *testing.T) {
	t.Parallel()
	assert.Less(t, int(PriorityCritical), int(PriorityHigh))
	assert.Less(t, int(PriorityHigh), int(PriorityNormal))
}

func TestDetectorMode_Values(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DetectorMode(0), DetectorModeSync)
	assert.Equal(t, DetectorMode(1), DetectorModeAsync)
}

func TestErrSentinels(t *testing.T) {
	t.Parallel()
	assert.Error(t, ErrSpamDetectDisabled)
	assert.Error(t, ErrSpamDetected)
	assert.Error(t, ErrAllDetectorsFailed)
	assert.Error(t, ErrNoMatchingDetectors)
	assert.Error(t, ErrDetectorUnavailable)
}

func TestSubmission_Sanitise_ActionNameTruncated(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		ActionName: strings.Repeat("a", maxActionNameLength+100),
	}
	submission.Sanitise(nil)
	assert.Equal(t, maxActionNameLength, len(submission.ActionName))
	assert.True(t, submission.WasTruncated())
}

func TestSubmission_Sanitise_HoneypotValueTruncated(t *testing.T) {
	t.Parallel()
	submission := &Submission{
		HoneypotValue: strings.Repeat("h", maxHoneypotLength+100),
	}
	submission.Sanitise(nil)
	assert.Equal(t, maxHoneypotLength, len(submission.HoneypotValue))
	assert.True(t, submission.WasTruncated())
}
