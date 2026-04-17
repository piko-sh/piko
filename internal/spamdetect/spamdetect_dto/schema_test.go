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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSchema_BasicFields(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		TextField("message", SignalGibberish, SignalLinkDensity),
		TextField("name", SignalGibberish),
	)

	assert.Len(t, schema.Fields(), 2)
	assert.Equal(t, "message", schema.Fields()[0].Key)
	assert.Equal(t, "name", schema.Fields()[1].Key)
}

func TestNewSchema_Honeypot(t *testing.T) {
	t.Parallel()
	schema := NewSchema(Honeypot("_hp"))
	assert.Equal(t, "_hp", schema.HoneypotKey())
	assert.True(t, schema.HasSignal(SignalHoneypot))
}

func TestNewSchema_Timing(t *testing.T) {
	t.Parallel()
	schema := NewSchema(Timing("_ts"))
	assert.Equal(t, "_ts", schema.TimingKey())
	assert.True(t, schema.HasSignal(SignalTiming))
}

func TestNewSchema_Threshold(t *testing.T) {
	t.Parallel()
	schema := NewSchema(Threshold(0.5))
	assert.InDelta(t, 0.5, schema.ScoreThreshold(), 0.01)
}

func TestNewSchema_DefaultThreshold(t *testing.T) {
	t.Parallel()
	schema := NewSchema()

	assert.InDelta(t, 0.0, schema.ScoreThreshold(), 0.01)
}

func TestFieldEntry_Weight(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message").Weight(3.0))
	assert.InDelta(t, 3.0, schema.Fields()[0].Weight, 0.01)
}

func TestFieldEntry_MaxLength(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message").MaxLength(100))
	assert.Equal(t, 100, schema.Fields()[0].MaxLength)
}

func TestFieldGroup_Composability(t *testing.T) {
	t.Parallel()
	personalFields := FieldGroup(
		TextField("firstname", SignalGibberish),
		TextField("lastname", SignalGibberish),
	)

	schemaA := NewSchema(personalFields, TextField("message", SignalBlocklist))
	schemaB := NewSchema(personalFields, TextField("notes", SignalLinkDensity))

	assert.Len(t, schemaA.Fields(), 3)
	assert.Len(t, schemaB.Fields(), 3)
	assert.Equal(t, "message", schemaA.Fields()[2].Key)
	assert.Equal(t, "notes", schemaB.Fields()[2].Key)
}

func TestFieldGroup_IsolatesMutations(t *testing.T) {
	t.Parallel()
	group := FieldGroup(TextField("name", SignalGibberish))
	schemaA := NewSchema(group)
	schemaB := NewSchema(group)

	assert.Len(t, schemaA.Fields(), 1)
	assert.Len(t, schemaB.Fields(), 1)
}

func TestSchema_FieldsWithSignal(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		TextField("message", SignalGibberish, SignalLinkDensity),
		TextField("name", SignalGibberish),
		TextField("email", SignalBlocklist),
	)

	gibberishFields := schema.FieldsWithSignal(SignalGibberish)
	assert.Len(t, gibberishFields, 2)

	blocklistFields := schema.FieldsWithSignal(SignalBlocklist)
	assert.Len(t, blocklistFields, 1)
	assert.Equal(t, "email", blocklistFields[0].Key)

	linkFields := schema.FieldsWithSignal(SignalLinkDensity)
	assert.Len(t, linkFields, 1)
	assert.Equal(t, "message", linkFields[0].Key)
}

func TestSchema_HasSignal(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		TextField("message", SignalGibberish),
		Honeypot("_hp"),
	)

	assert.True(t, schema.HasSignal(SignalGibberish))
	assert.True(t, schema.HasSignal(SignalHoneypot))
	assert.False(t, schema.HasSignal(SignalLinkDensity))
	assert.False(t, schema.HasSignal(SignalTiming))
}

func TestSchema_AllSignals(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		TextField("message", SignalGibberish, SignalLinkDensity),
		TextField("name", SignalGibberish),
		Honeypot("_hp"),
		Timing("_ts"),
	)

	signals := schema.AllSignals()
	assert.Contains(t, signals, SignalHoneypot)
	assert.Contains(t, signals, SignalTiming)
	assert.Contains(t, signals, SignalGibberish)
	assert.Contains(t, signals, SignalLinkDensity)
	assert.Len(t, signals, 4)
}

func TestSchema_NilEntries(t *testing.T) {
	t.Parallel()
	schema := NewSchema(nil, TextField("message"), nil)
	assert.Len(t, schema.Fields(), 1)
}

func TestSchema_MaxFields(t *testing.T) {
	t.Parallel()
	entries := make([]SchemaEntry, maxSchemaFields+10)
	for index := range entries {
		entries[index] = TextField("field_" + string(rune('a'+index%26)))
	}
	schema := NewSchema(entries...)
	assert.LessOrEqual(t, len(schema.Fields()), maxSchemaFields)
}

func TestSchema_CustomSignal(t *testing.T) {
	t.Parallel()
	customSignal := Signal("profanity")
	schema := NewSchema(TextField("message", customSignal))

	assert.True(t, schema.HasSignal(customSignal))
	fields := schema.FieldsWithSignal(customSignal)
	assert.Len(t, fields, 1)
}

func TestSchema_Meta_Single(t *testing.T) {
	t.Parallel()
	schema := NewSchema(Meta("site", "example.com"))

	metadata := schema.GetMetadata()
	assert.Len(t, metadata, 1)
	assert.Equal(t, "example.com", metadata["site"])
}

func TestSchema_Meta_Multiple(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		Meta("site", "example.com"),
		Meta("form_type", "contact"),
		Meta("region", "eu"),
	)

	metadata := schema.GetMetadata()
	assert.Len(t, metadata, 3)
	assert.Equal(t, "example.com", metadata["site"])
	assert.Equal(t, "contact", metadata["form_type"])
	assert.Equal(t, "eu", metadata["region"])
}

func TestSchema_Meta_CapsAtMaxDetectorOptions(t *testing.T) {
	t.Parallel()
	entries := make([]SchemaEntry, 0, maxDetectorOptions+10)
	for index := range maxDetectorOptions + 10 {
		entries = append(entries, Meta("key_"+string(rune('a'+index%26))+string(rune('0'+index/26)), "value"))
	}
	schema := NewSchema(entries...)

	metadata := schema.GetMetadata()
	assert.LessOrEqual(t, len(metadata), maxDetectorOptions)
}

func TestSchema_CaptureHeader_Single(t *testing.T) {
	t.Parallel()
	schema := NewSchema(CaptureHeader("X-Forwarded-For"))

	headers := schema.CapturedHeaders()
	assert.Equal(t, []string{"X-Forwarded-For"}, headers)
}

func TestSchema_CaptureHeader_Multiple(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		CaptureHeader("X-Forwarded-For"),
		CaptureHeader("Accept-Language"),
	)

	headers := schema.CapturedHeaders()
	assert.Equal(t, []string{"X-Forwarded-For", "Accept-Language"}, headers)
}

func TestSchema_CaptureHeader_CapsAtMaxDetectorOptions(t *testing.T) {
	t.Parallel()
	entries := make([]SchemaEntry, 0, maxDetectorOptions+5)
	for index := range maxDetectorOptions + 5 {
		entries = append(entries, CaptureHeader("Header-"+string(rune('A'+index%26))))
	}
	schema := NewSchema(entries...)

	headers := schema.CapturedHeaders()
	assert.LessOrEqual(t, len(headers), maxDetectorOptions)
}

func TestSchema_Language_Single(t *testing.T) {
	t.Parallel()
	schema := NewSchema(Language("en"))
	assert.Equal(t, []string{"en"}, schema.GetLanguages())
}

func TestSchema_Language_Multiple(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		Language("en"),
		Language("fr"),
		Language("de"),
	)
	assert.Equal(t, []string{"en", "fr", "de"}, schema.GetLanguages())
}

func TestSchema_Language_CapsAtMaxDetectorOptions(t *testing.T) {
	t.Parallel()
	entries := make([]SchemaEntry, 0, maxDetectorOptions+5)
	for index := range maxDetectorOptions + 5 {
		entries = append(entries, Language("lang_"+string(rune('a'+index%26))))
	}
	schema := NewSchema(entries...)

	languages := schema.GetLanguages()
	assert.LessOrEqual(t, len(languages), maxDetectorOptions)
}

func TestSchema_DetectorWeight(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		DetectorWeight("gibberish", 2.0),
		DetectorWeight("blocklist", 0.5),
	)

	assert.InDelta(t, 2.0, schema.GetDetectorWeight("gibberish"), 0.01)
	assert.InDelta(t, 0.5, schema.GetDetectorWeight("blocklist"), 0.01)
	assert.InDelta(t, 0.0, schema.GetDetectorWeight("nonexistent"), 0.01)
}

func TestSchema_DetectorWeight_NilMap(t *testing.T) {
	t.Parallel()
	schema := NewSchema()
	assert.InDelta(t, 0.0, schema.GetDetectorWeight("anything"), 0.01)
}

func TestSchema_DetectorWeight_CapsAtMaxDetectorOptions(t *testing.T) {
	t.Parallel()
	entries := make([]SchemaEntry, 0, maxDetectorOptions+10)
	for index := range maxDetectorOptions + 10 {
		entries = append(entries, DetectorWeight("detector_"+string(rune('a'+index%26))+string(rune('0'+index/26)), 1.0))
	}
	schema := NewSchema(entries...)

	presentCount := 0
	for index := range maxDetectorOptions + 10 {
		name := "detector_" + string(rune('a'+index%26)) + string(rune('0'+index/26))
		if schema.GetDetectorWeight(name) > 0 {
			presentCount++
		}
	}
	assert.LessOrEqual(t, presentCount, maxDetectorOptions)
}

func TestSchema_DetectorConfig_ReturnsClone(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		DetectorConfig("gibberish", map[string]any{"threshold": 0.8}),
	)

	options := schema.DetectorOptions("gibberish")
	assert.NotNil(t, options)
	assert.Equal(t, 0.8, options["threshold"])

	options["threshold"] = 0.1
	assert.Equal(t, 0.8, schema.DetectorOptions("gibberish")["threshold"])

	assert.Nil(t, schema.DetectorOptions("unknown"))
}

func TestSchema_DetectorConfig_NilMap(t *testing.T) {
	t.Parallel()
	schema := NewSchema()
	assert.Nil(t, schema.DetectorOptions("anything"))
}

func TestSchema_OnAsyncResult(t *testing.T) {
	t.Parallel()

	var calledID string
	var calledResult *AnalysisResult
	handler := func(submissionID string, result *AnalysisResult) {
		calledID = submissionID
		calledResult = result
	}

	schema := NewSchema(OnAsyncResult(handler))

	assert.NotNil(t, schema.AsyncHandler())

	expectedResult := &AnalysisResult{Score: 0.9, IsSpam: true}
	schema.AsyncHandler()("test-id-123", expectedResult)

	assert.Equal(t, "test-id-123", calledID)
	assert.Equal(t, expectedResult, calledResult)
}

func TestSchema_AsyncHandler_Nil(t *testing.T) {
	t.Parallel()
	schema := NewSchema()
	assert.Nil(t, schema.AsyncHandler())
}

func TestSchema_FieldsWithType(t *testing.T) {
	t.Parallel()
	schema := NewSchema(
		EmailField("email", SignalBlocklist),
		PhoneField("phone"),
		NameField("name", SignalGibberish),
		URLField("website", SignalBlocklist),
		TextField("message", SignalGibberish),
	)

	emailFields := schema.FieldsWithType(FieldTypeEmail)
	assert.Len(t, emailFields, 1)
	assert.Equal(t, "email", emailFields[0].Key)

	phoneFields := schema.FieldsWithType(FieldTypePhone)
	assert.Len(t, phoneFields, 1)
	assert.Equal(t, "phone", phoneFields[0].Key)

	nameFields := schema.FieldsWithType(FieldTypeName)
	assert.Len(t, nameFields, 1)
	assert.Equal(t, "name", nameFields[0].Key)

	urlFields := schema.FieldsWithType(FieldTypeURL)
	assert.Len(t, urlFields, 1)
	assert.Equal(t, "website", urlFields[0].Key)

	textFields := schema.FieldsWithType(FieldTypeText)
	assert.Len(t, textFields, 1)
	assert.Equal(t, "message", textFields[0].Key)

	customFields := schema.FieldsWithType("nonexistent")
	assert.Empty(t, customFields)
}

func TestSchema_MaxLength_ClampedToOne(t *testing.T) {
	t.Parallel()

	schema := NewSchema(TextField("msg").MaxLength(0))
	assert.Equal(t, 1, schema.Fields()[0].MaxLength)

	schema2 := NewSchema(TextField("msg").MaxLength(-100))
	assert.Equal(t, 1, schema2.Fields()[0].MaxLength)
}

func TestSchema_GetMetadata_NilWhenNoMeta(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message"))
	assert.Nil(t, schema.GetMetadata())
}

func TestSchema_CapturedHeaders_NilWhenNone(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message"))
	assert.Nil(t, schema.CapturedHeaders())
}

func TestSchema_GetLanguages_NilWhenNone(t *testing.T) {
	t.Parallel()
	schema := NewSchema(TextField("message"))
	assert.Nil(t, schema.GetLanguages())
}
