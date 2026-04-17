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
	"maps"
	"slices"
)

const (
	// defaultFieldMaxLength is the default maximum byte length for field values.
	defaultFieldMaxLength = 64 * 1024

	// defaultFieldWeight is the default scoring weight for a field.
	defaultFieldWeight = 1.0

	// maxSchemaFields is the upper bound on fields in a single schema.
	maxSchemaFields = 128

	// maxDetectorOptions is the upper bound on per-schema detector config entries.
	maxDetectorOptions = 64
)

// FieldType identifies the semantic type of a form field.
//
// Built-in types are defined as constants. Users may define additional
// types as plain FieldType strings for use with custom detectors and
// third-party providers.
type FieldType string

const (
	// FieldTypeText is the default type for generic freeform text fields.
	FieldTypeText FieldType = "text"

	// FieldTypeEmail identifies email address fields.
	FieldTypeEmail FieldType = "email"

	// FieldTypePhone identifies phone number fields.
	FieldTypePhone FieldType = "phone"

	// FieldTypeName identifies person name fields.
	FieldTypeName FieldType = "name"

	// FieldTypeURL identifies URL fields.
	FieldTypeURL FieldType = "url"
)

// Schema is an immutable description of a form's spam-checkable fields.
// Built via NewSchema and shared safely across concurrent requests.
type Schema struct {
	// detectorWeights maps detector names to their scoring weight.
	detectorWeights map[string]float64

	// detectorOptions holds per-detector configuration overrides.
	detectorOptions map[string]map[string]any

	// metadata holds static key-value pairs for provider context.
	metadata map[string]string

	// asyncHandler is called when async detectors complete.
	asyncHandler AsyncResultHandler

	// honeypotKey is the form argument key for the hidden honeypot field.
	honeypotKey string

	// timingKey is the form argument key for the timing timestamp token.
	timingKey string

	// languages holds the expected content languages (ISO 639-1).
	languages []string

	// fields holds the declared form field definitions.
	fields []Field

	// capturedHeaders lists HTTP header names to capture.
	capturedHeaders []string

	// threshold is the composite score above which a submission is spam.
	threshold float64

	// thresholdExplicit is true when the threshold was explicitly set.
	thresholdExplicit bool
}

// Field describes one form field and the spam signals that apply to it.
type Field struct {
	// Key is the form argument key for this field.
	Key string

	// Type is the semantic type of this field.
	Type FieldType

	// Signals lists the spam detection signals that apply to this field.
	Signals []Signal

	// MaxLength is the maximum byte length for this field's value.
	MaxLength int

	// Weight is the scoring weight for this field.
	Weight float64
}

// AsyncResultHandler is called when asynchronous detectors complete
// their analysis after the initial synchronous response.
type AsyncResultHandler func(submissionID string, result *AnalysisResult)

// SchemaEntry is implemented by anything that can configure a Schema
// during construction.
type SchemaEntry interface {
	// applyToSchema applies this entry's configuration to the schema.
	applyToSchema(schema *Schema)
}

// FieldEntry configures a single field. Returned by TextField and other
// typed field builders.
type FieldEntry struct {
	// field holds the field configuration.
	field Field
}

// schemaOptionFunc adapts a function into a SchemaEntry.
type schemaOptionFunc func(*Schema)

// applyToSchema calls the underlying function.
//
// Takes schema (*Schema) which receives the configuration.
func (f schemaOptionFunc) applyToSchema(schema *Schema) { f(schema) }

// fieldGroupEntry is a group of FieldEntry values composed into a single SchemaEntry.
type fieldGroupEntry []*FieldEntry

// applyToSchema applies all entries in the group to the schema.
//
// Takes schema (*Schema) which receives the grouped fields.
func (g fieldGroupEntry) applyToSchema(schema *Schema) {
	for _, entry := range g {
		entry.applyToSchema(schema)
	}
}

// NewSchema creates an immutable Schema from the given entries.
//
// Takes entries (...SchemaEntry) which configure the schema fields and options.
//
// Returns *Schema which is the immutable schema, safe for concurrent use.
func NewSchema(entries ...SchemaEntry) *Schema {
	schema := &Schema{
		threshold: defaultScoreThreshold,
	}

	for _, entry := range entries {
		if entry == nil {
			continue
		}
		entry.applyToSchema(schema)
	}

	if len(schema.fields) > maxSchemaFields {
		schema.fields = schema.fields[:maxSchemaFields]
	}

	return schema
}

// newFieldEntry creates a FieldEntry with the given key, type, and
// signals.
//
// Takes key (string) which is the form argument key.
// Takes fieldType (FieldType) which is the semantic type.
// Takes signals ([]Signal) which are the detection signals.
//
// Returns *FieldEntry which is the configured entry.
func newFieldEntry(key string, fieldType FieldType, signals []Signal) *FieldEntry {
	return &FieldEntry{
		field: Field{
			Key:       key,
			Type:      fieldType,
			Signals:   signals,
			MaxLength: defaultFieldMaxLength,
			Weight:    defaultFieldWeight,
		},
	}
}

// TextField creates a field entry for a generic text form field.
//
// Takes key (string) which is the form argument key.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the text field.
func TextField(key string, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, FieldTypeText, signals)
}

// EmailField creates a field entry semantically typed as an email address.
//
// Takes key (string) which is the form argument key.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the email field.
func EmailField(key string, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, FieldTypeEmail, signals)
}

// PhoneField creates a field entry semantically typed as a phone number.
//
// Takes key (string) which is the form argument key.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the phone field.
func PhoneField(key string, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, FieldTypePhone, signals)
}

// NameField creates a field entry semantically typed as a person's name.
//
// Takes key (string) which is the form argument key.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the name field.
func NameField(key string, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, FieldTypeName, signals)
}

// URLField creates a field entry semantically typed as a URL.
//
// Takes key (string) which is the form argument key.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the URL field.
func URLField(key string, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, FieldTypeURL, signals)
}

// TypedField creates a field entry with a custom field type.
//
// Takes key (string) which is the form argument key.
// Takes fieldType (FieldType) which is the custom semantic type.
// Takes signals (...Signal) which are the detection signals for this field.
//
// Returns *FieldEntry which configures the typed field.
func TypedField(key string, fieldType FieldType, signals ...Signal) *FieldEntry {
	return newFieldEntry(key, fieldType, signals)
}

// Honeypot declares the form argument key for the hidden honeypot field.
//
// Takes key (string) which is the honeypot field key.
//
// Returns SchemaEntry which configures the honeypot.
func Honeypot(key string) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		schema.honeypotKey = key
	})
}

// Timing declares the form argument key for the signed form-load
// timestamp token.
//
// Takes key (string) which is the timing field key.
//
// Returns SchemaEntry which configures the timing signal.
func Timing(key string) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		schema.timingKey = key
	})
}

// Threshold sets the composite score above which a submission is rejected.
//
// Takes threshold (float64) which overrides the service-level default.
//
// Returns SchemaEntry which configures the threshold.
func Threshold(threshold float64) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		schema.threshold = threshold
		schema.thresholdExplicit = true
	})
}

// Language adds an expected content language (ISO 639-1 code).
//
// Multiple calls add multiple languages. The gibberish detector
// analyses text against all declared languages and uses the best
// score.
//
// Takes language (string) which is the ISO 639-1 language code.
//
// Returns SchemaEntry which adds the language.
func Language(language string) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		if len(schema.languages) < maxDetectorOptions {
			schema.languages = append(schema.languages, language)
		}
	})
}

// DetectorWeight sets the scoring weight for a named detector when
// computing field-level scores.
//
// Takes detectorName (string) which identifies the detector.
// Takes weight (float64) which is the scoring weight.
//
// Returns SchemaEntry which configures the detector weight.
func DetectorWeight(detectorName string, weight float64) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		if schema.detectorWeights == nil {
			schema.detectorWeights = make(map[string]float64)
		}
		if len(schema.detectorWeights) >= maxDetectorOptions {
			return
		}
		schema.detectorWeights[detectorName] = weight
	})
}

// DetectorConfig sets per-schema configuration options for a named detector.
//
// Takes detectorName (string) which identifies the detector.
// Takes options (map[string]any) which are the configuration overrides.
//
// Returns SchemaEntry which configures the detector options.
func DetectorConfig(detectorName string, options map[string]any) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		if schema.detectorOptions == nil {
			schema.detectorOptions = make(map[string]map[string]any)
		}
		if len(schema.detectorOptions) < maxDetectorOptions {
			schema.detectorOptions[detectorName] = maps.Clone(options)
		}
	})
}

// OnAsyncResult registers a callback for asynchronous detector completion.
//
// Takes handler (AsyncResultHandler) which receives the async results.
//
// Returns SchemaEntry which configures the callback.
func OnAsyncResult(handler AsyncResultHandler) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		schema.asyncHandler = handler
	})
}

// Meta declares a static metadata key-value pair that is copied into
// every submission.
//
// Takes key (string) which is the metadata key.
// Takes value (string) which is the metadata value.
//
// Returns SchemaEntry which configures the metadata entry.
func Meta(key, value string) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		if schema.metadata == nil {
			schema.metadata = make(map[string]string)
		}
		if len(schema.metadata) < maxDetectorOptions {
			schema.metadata[key] = value
		}
	})
}

// CaptureHeader declares an HTTP request header that should be captured
// into the submission for provider use.
//
// Takes headerName (string) which is the header name to capture.
//
// Returns SchemaEntry which configures the header capture.
func CaptureHeader(headerName string) SchemaEntry {
	return schemaOptionFunc(func(schema *Schema) {
		if len(schema.capturedHeaders) < maxDetectorOptions {
			schema.capturedHeaders = append(schema.capturedHeaders, headerName)
		}
	})
}

// FieldGroup composes multiple field entries into a single SchemaEntry
// for reuse across schemas.
//
// Takes fields (...*FieldEntry) which are the entries to group.
//
// Returns SchemaEntry which applies all grouped fields.
func FieldGroup(fields ...*FieldEntry) SchemaEntry {
	copied := make([]*FieldEntry, 0, len(fields))
	for _, entry := range fields {
		if entry == nil {
			continue
		}
		entryCopy := *entry
		fieldCopy := entry.field
		signalsCopy := make([]Signal, len(entry.field.Signals))
		copy(signalsCopy, entry.field.Signals)
		fieldCopy.Signals = signalsCopy
		entryCopy.field = fieldCopy
		copied = append(copied, &entryCopy)
	}
	return fieldGroupEntry(copied)
}

// MaxLength sets the maximum byte length for this field's value during
// sanitisation.
//
// Takes length (int) which is the maximum byte length (clamped to 1 minimum).
//
// Returns *FieldEntry which is the same entry for chaining.
func (f *FieldEntry) MaxLength(length int) *FieldEntry {
	if length < 1 {
		length = 1
	}
	f.field.MaxLength = length
	return f
}

// Weight sets the scoring weight for this field when computing the
// composite score.
//
// Takes weight (float64) which is the scoring weight (default 1.0).
//
// Returns *FieldEntry which is the same entry for chaining.
func (f *FieldEntry) Weight(weight float64) *FieldEntry {
	f.field.Weight = weight
	return f
}

// applyToSchema appends this entry's field to the schema.
//
// Takes schema (*Schema) which receives the field.
func (f *FieldEntry) applyToSchema(schema *Schema) {
	schema.fields = append(schema.fields, f.field)
}

// Fields returns a copy of all fields declared in the schema.
//
// Returns []Field which contains the declared fields.
func (s *Schema) Fields() []Field {
	return slices.Clone(s.fields)
}

// FieldsWithSignal returns all fields that include the given signal.
//
// Takes signal (Signal) which is the signal to filter by.
//
// Returns []Field which contains the matching fields.
func (s *Schema) FieldsWithSignal(signal Signal) []Field {
	var matched []Field
	for _, field := range s.fields {
		if slices.Contains(field.Signals, signal) {
			matched = append(matched, field)
		}
	}
	return matched
}

// FieldsWithType returns all fields that have the given semantic type.
//
// Takes fieldType (FieldType) which is the type to filter by.
//
// Returns []Field which contains the matching fields.
func (s *Schema) FieldsWithType(fieldType FieldType) []Field {
	var matched []Field
	for _, field := range s.fields {
		if field.Type == fieldType {
			matched = append(matched, field)
		}
	}
	return matched
}

// HoneypotKey returns the form argument key for the hidden honeypot
// field, or an empty string if not configured.
//
// Returns string which is the honeypot key.
func (s *Schema) HoneypotKey() string {
	return s.honeypotKey
}

// TimingKey returns the form argument key for the form-load timestamp
// token, or an empty string if not configured.
//
// Returns string which is the timing key.
func (s *Schema) TimingKey() string {
	return s.timingKey
}

// ScoreThreshold returns the composite score threshold for this schema,
// or 0 if no explicit threshold was set.
//
// Returns float64 which is the threshold, or 0 for the service default.
func (s *Schema) ScoreThreshold() float64 {
	if !s.thresholdExplicit {
		return 0
	}
	return s.threshold
}

// GetLanguages returns the expected content languages (ISO 639-1).
//
// Returns []string which contains the declared language codes.
func (s *Schema) GetLanguages() []string {
	return slices.Clone(s.languages)
}

// GetDetectorWeight returns the scoring weight for the named detector,
// or 0 if not set.
//
// Takes name (string) which identifies the detector.
//
// Returns float64 which is the weight, or 0 for the default.
func (s *Schema) GetDetectorWeight(name string) float64 {
	if s.detectorWeights == nil {
		return 0
	}
	return s.detectorWeights[name]
}

// DetectorOptions returns a copy of the per-schema configuration for
// the named detector, or nil if not set.
//
// Takes detectorName (string) which identifies the detector.
//
// Returns map[string]any which is the configuration copy, or nil.
func (s *Schema) DetectorOptions(detectorName string) map[string]any {
	if s.detectorOptions == nil {
		return nil
	}
	return maps.Clone(s.detectorOptions[detectorName])
}

// AsyncHandler returns the registered async result handler, or nil.
//
// Returns AsyncResultHandler which is the callback, or nil.
func (s *Schema) AsyncHandler() AsyncResultHandler {
	return s.asyncHandler
}

// GetMetadata returns a copy of the static metadata map declared by
// Meta() entries.
//
// Returns map[string]string which is the metadata copy.
func (s *Schema) GetMetadata() map[string]string {
	return maps.Clone(s.metadata)
}

// CapturedHeaders returns a copy of the HTTP header names to capture.
//
// Returns []string which contains the header names.
func (s *Schema) CapturedHeaders() []string {
	return slices.Clone(s.capturedHeaders)
}

// HasSignal reports whether any field in the schema uses the given signal.
//
// Takes signal (Signal) which is the signal to check.
//
// Returns bool which is true when the signal is present.
func (s *Schema) HasSignal(signal Signal) bool {
	if signal == SignalHoneypot {
		return s.honeypotKey != ""
	}
	if signal == SignalTiming {
		return s.timingKey != ""
	}
	for _, field := range s.fields {
		if slices.Contains(field.Signals, signal) {
			return true
		}
	}
	return false
}

// AllSignals returns the deduplicated set of signals used across all
// fields and schema-level declarations (honeypot, timing).
//
// Returns []Signal which contains the deduplicated signals.
func (s *Schema) AllSignals() []Signal {
	seen := make(map[Signal]struct{})
	var signals []Signal

	if s.honeypotKey != "" {
		seen[SignalHoneypot] = struct{}{}
		signals = append(signals, SignalHoneypot)
	}
	if s.timingKey != "" {
		seen[SignalTiming] = struct{}{}
		signals = append(signals, SignalTiming)
	}

	for _, field := range s.fields {
		for _, signal := range field.Signals {
			if _, exists := seen[signal]; !exists {
				seen[signal] = struct{}{}
				signals = append(signals, signal)
			}
		}
	}

	return signals
}
