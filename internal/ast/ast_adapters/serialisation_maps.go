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

package ast_adapters

import (
	"fmt"
	"maps"
	"slices"

	"piko.sh/piko/internal/json"
	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
)

// buildOnEventsMap serialises an event handlers map to FlatBuffers format.
//
// Takes m (map[string][]ast_domain.Directive) which maps event names to their
// handler directives.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised vector,
// or zero if the map is empty.
// Returns error when serialising any directive fails.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (s *encoder) buildOnEventsMap(m map[string][]ast_domain.Directive) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		handlersVec, err := buildVectorOfValues(s, m[k], (*encoder).buildDirective)
		if err != nil {
			return 0, fmt.Errorf("building on-event handlers for event %q: %w", k, err)
		}

		ast_schema_gen.OnEventEntryFBStart(s.builder)
		ast_schema_gen.OnEventEntryFBAddEventName(s.builder, keyOff)
		ast_schema_gen.OnEventEntryFBAddHandlers(s.builder, handlersVec)
		offsets[i] = ast_schema_gen.OnEventEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildCustomEventsMap serialises a map of custom event directives into a
// FlatBuffers vector.
//
// Takes m (map[string][]ast_domain.Directive) which maps event names to their
// handler directives.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised vector,
// or zero if the map is empty.
// Returns error when directive serialisation fails.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (s *encoder) buildCustomEventsMap(m map[string][]ast_domain.Directive) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		handlersVec, err := buildVectorOfValues(s, m[k], (*encoder).buildDirective)
		if err != nil {
			return 0, fmt.Errorf("building custom event handlers for event %q: %w", k, err)
		}

		ast_schema_gen.CustomEventEntryFBStart(s.builder)
		ast_schema_gen.CustomEventEntryFBAddEventName(s.builder, keyOff)
		ast_schema_gen.CustomEventEntryFBAddHandlers(s.builder, handlersVec)
		offsets[i] = ast_schema_gen.CustomEventEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildBindsMap converts a map of bind directives to a FlatBuffer vector.
//
// Takes m (map[string]*ast_domain.Directive) which contains the bind entries
// keyed by attribute name.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised vector.
// Returns error when a directive fails to build.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (s *encoder) buildBindsMap(m map[string]*ast_domain.Directive) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		dirOff, err := s.buildDirective(m[k])
		if err != nil {
			return 0, fmt.Errorf("building bind directive for attribute %q: %w", k, err)
		}

		ast_schema_gen.BindEntryFBStart(s.builder)
		ast_schema_gen.BindEntryFBAddAttributeName(s.builder, keyOff)
		ast_schema_gen.BindEntryFBAddDirective(s.builder, dirOff)
		offsets[i] = ast_schema_gen.BindEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildDynamicAttributeOriginsMap converts a string map into a FlatBuffers
// vector of key-value entries.
//
// Takes m (map[string]string) which contains the attribute origin mappings.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if the map is empty.
// Returns error when vector creation fails.
func (s *encoder) buildDynamicAttributeOriginsMap(m map[string]string) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		valOff := s.builder.CreateString(m[k])

		ast_schema_gen.DynamicAttributeOriginEntryFBStart(s.builder)
		ast_schema_gen.DynamicAttributeOriginEntryFBAddKey(s.builder, keyOff)
		ast_schema_gen.DynamicAttributeOriginEntryFBAddValue(s.builder, valOff)
		offsets[i] = ast_schema_gen.DynamicAttributeOriginEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildPropValueMap serialises a map of property values to a FlatBuffer vector.
//
// Takes m (map[string]ast_domain.PropValue) which contains the property values
// to serialise, keyed by property name.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if the map is empty.
// Returns error when a property value fails to serialise.
func (s *encoder) buildPropValueMap(m map[string]ast_domain.PropValue) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		valOff, err := s.buildPropValue(new(m[k]))
		if err != nil {
			return 0, fmt.Errorf("building prop value for key %q: %w", k, err)
		}

		ast_schema_gen.PropValueEntryFBStart(s.builder)
		ast_schema_gen.PropValueEntryFBAddKey(s.builder, keyOff)
		ast_schema_gen.PropValueEntryFBAddValue(s.builder, valOff)
		offsets[i] = ast_schema_gen.PropValueEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildObjectLiteralPairs serialises a map of expressions into a FlatBuffer
// vector of key-value pairs.
//
// Takes m (map[string]ast_domain.Expression) which contains the key-value
// pairs to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised vector,
// or zero if the map is empty.
// Returns error when serialising any expression value fails.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (s *encoder) buildObjectLiteralPairs(m map[string]ast_domain.Expression) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)
		valOff, err := s.buildExpressionNode(m[k])
		if err != nil {
			return 0, fmt.Errorf("building object literal value for key %q: %w", k, err)
		}

		ast_schema_gen.ObjectPairFBStart(s.builder)
		ast_schema_gen.ObjectPairFBAddKey(s.builder, keyOff)
		ast_schema_gen.ObjectPairFBAddValue(s.builder, valOff)
		offsets[i] = ast_schema_gen.ObjectPairFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// buildDiagnosticDataMap converts a map of diagnostic data into a FlatBuffers
// vector of key-value entries.
//
// Takes m (map[string]any) which contains the diagnostic data to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if the map is empty.
// Returns error when a map value cannot be marshalled to JSON.
func (s *encoder) buildDiagnosticDataMap(m map[string]any) (flatbuffers.UOffsetT, error) {
	if len(m) == 0 {
		return 0, nil
	}
	keys := getSortedKeys(m)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		keyOff := s.builder.CreateString(k)

		valueJSON, err := json.Marshal(m[k])
		if err != nil {
			return 0, fmt.Errorf("failed to marshal diagnostic data value for key '%s': %w", k, err)
		}
		valOff := s.builder.CreateString(string(valueJSON))

		ast_schema_gen.DiagnosticDataEntryFBStart(s.builder)
		ast_schema_gen.DiagnosticDataEntryFBAddKey(s.builder, keyOff)
		ast_schema_gen.DiagnosticDataEntryFBAddValue(s.builder, valOff)
		offsets[i] = ast_schema_gen.DiagnosticDataEntryFBEnd(s.builder)
	}
	return createVector(s, offsets), nil
}

// unpackOnEventsMap extracts the on-events map from a template node buffer.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer node to
// unpack.
//
// Returns map[string][]ast_domain.Directive which maps event names to their
// handler directives.
// Returns error when handler unpacking fails.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (d *decoder) unpackOnEventsMap(fb *ast_schema_gen.TemplateNodeFB) (map[string][]ast_domain.Directive, error) {
	length := fb.OnEventsLength()
	if length == 0 {
		return nil, nil
	}

	result := make(map[string][]ast_domain.Directive, length)
	var entryFB ast_schema_gen.OnEventEntryFB
	for i := range length {
		if fb.OnEvents(&entryFB, i) {
			key := string(entryFB.EventName())
			handlers, err := unpackVector(d, entryFB.HandlersLength(), entryFB.Handlers, (*decoder).unpackDirectiveValue)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack handlers for event '%s': %w", key, err)
			}
			result[key] = handlers
		}
	}
	return result, nil
}

// unpackCustomEventsMap extracts custom event handlers from a template node.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer node to
// extract custom events from.
//
// Returns map[string][]ast_domain.Directive which maps event names to their
// handler directives, or nil if no custom events exist.
// Returns error when unpacking any event's handlers fails.
//
//nolint:dupl // type-specific FlatBuffer serialisation
func (d *decoder) unpackCustomEventsMap(fb *ast_schema_gen.TemplateNodeFB) (map[string][]ast_domain.Directive, error) {
	numEvents := fb.CustomEventsLength()
	if numEvents == 0 {
		return nil, nil
	}

	result := make(map[string][]ast_domain.Directive, numEvents)
	var entFB ast_schema_gen.CustomEventEntryFB
	for i := range numEvents {
		if fb.CustomEvents(&entFB, i) {
			evtName := string(entFB.EventName())
			hdlrs, err := unpackVector(d, entFB.HandlersLength(), entFB.Handlers, (*decoder).unpackDirectiveValue)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack handlers for custom event '%s': %w", evtName, err)
			}
			result[evtName] = hdlrs
		}
	}
	return result, nil
}

// unpackBindsMap extracts bind entries from a FlatBuffer template node into a
// map of directives keyed by attribute name.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer node
// containing bind entries.
//
// Returns map[string]*ast_domain.Directive which maps attribute names to their
// directives, or nil if there are no binds.
// Returns error when a directive cannot be unpacked.
func (d *decoder) unpackBindsMap(fb *ast_schema_gen.TemplateNodeFB) (map[string]*ast_domain.Directive, error) {
	length := fb.BindsLength()
	if length == 0 {
		return nil, nil
	}

	result := make(map[string]*ast_domain.Directive, length)
	var entryFB ast_schema_gen.BindEntryFB
	for i := range length {
		if fb.Binds(&entryFB, i) {
			key := string(entryFB.AttributeName())
			directive, err := d.unpackDirective(entryFB.Directive(&d.dirFB))
			if err != nil {
				return nil, fmt.Errorf("failed to unpack directive for bind '%s': %w", key, err)
			}
			result[key] = directive
		}
	}
	return result, nil
}

// unpackDynamicAttributeOriginsMap extracts the dynamic attribute origins from
// a FlatBuffer annotation into a map.
//
// Takes fb (*ast_schema_gen.GoGeneratorAnnotationFB) which contains the
// serialised annotation data.
//
// Returns map[string]string which maps attribute keys to their origin values,
// or nil if no entries exist.
// Returns error when deserialisation fails.
func (*decoder) unpackDynamicAttributeOriginsMap(fb *ast_schema_gen.GoGeneratorAnnotationFB) (map[string]string, error) {
	length := fb.DynamicAttributeOriginsLength()
	if length == 0 {
		return nil, nil
	}

	result := make(map[string]string, length)
	var entryFB ast_schema_gen.DynamicAttributeOriginEntryFB
	for i := range length {
		if fb.DynamicAttributeOrigins(&entryFB, i) {
			key := string(entryFB.Key())
			value := string(entryFB.Value())
			result[key] = value
		}
	}
	return result, nil
}

// unpackPropValueMap converts FlatBuffer property entries into a domain map.
//
// Takes length (int) which specifies the number of entries to unpack.
// Takes getter (func(...)) which retrieves each FlatBuffer entry by index.
//
// Returns map[string]ast_domain.PropValue which contains the unpacked
// property values keyed by name, or nil if length is zero.
// Returns error when a property value fails to unpack.
func (d *decoder) unpackPropValueMap(length int, getter func(entry *ast_schema_gen.PropValueEntryFB, j int) bool) (map[string]ast_domain.PropValue, error) {
	if length == 0 {
		return nil, nil
	}

	result := make(map[string]ast_domain.PropValue, length)
	var entryFB ast_schema_gen.PropValueEntryFB
	for i := range length {
		if getter(&entryFB, i) {
			key := string(entryFB.Key())
			value, err := d.unpackPropValue(entryFB.Value(&d.propValFB))
			if err != nil {
				return nil, fmt.Errorf("failed to unpack prop value for key '%s': %w", key, err)
			}
			result[key] = value
		}
	}
	return result, nil
}

// unpackObjectLiteralPairs extracts key-value pairs from an object literal.
//
// Takes fb (*ast_schema_gen.ObjectLiteralFB) which contains the serialised
// object literal pairs.
//
// Returns map[string]ast_domain.Expression which maps keys to their values.
// Returns error when a value expression cannot be unpacked.
func (d *decoder) unpackObjectLiteralPairs(fb *ast_schema_gen.ObjectLiteralFB) (map[string]ast_domain.Expression, error) {
	length := fb.PairsLength()
	if length == 0 {
		return nil, nil
	}

	result := make(map[string]ast_domain.Expression, length)
	var entryFB ast_schema_gen.ObjectPairFB
	for i := range length {
		if fb.Pairs(&entryFB, i) {
			key := string(entryFB.Key())
			value, err := d.unpackExpressionNode(entryFB.Value(&d.expressionNodeFB))
			if err != nil {
				return nil, fmt.Errorf("failed to unpack object literal value for key '%s': %w", key, err)
			}
			result[key] = value
		}
	}
	return result, nil
}

// unpackDiagnosticDataMap extracts the data map from a diagnostic flatbuffer.
//
// Takes fb (*ast_schema_gen.DiagnosticFB) which contains the serialised
// diagnostic data entries.
//
// Returns map[string]any which contains the deserialised key-value pairs.
// Returns error when JSON unmarshalling fails for any data value.
func (*decoder) unpackDiagnosticDataMap(fb *ast_schema_gen.DiagnosticFB) (map[string]any, error) {
	length := fb.DataLength()
	if length == 0 {
		return nil, nil
	}

	result := make(map[string]any, length)
	var entryFB ast_schema_gen.DiagnosticDataEntryFB
	for i := range length {
		if fb.Data(&entryFB, i) {
			key := string(entryFB.Key())
			valueJSON := string(entryFB.Value())

			var value any
			if err := json.UnmarshalString(valueJSON, &value); err != nil {
				return nil, fmt.Errorf("failed to unmarshal diagnostic data value for key '%s': %w", key, err)
			}
			result[key] = value
		}
	}
	return result, nil
}

// getSortedKeys returns the keys of a map in sorted order.
//
// Takes m (map[string]V) which is the map to extract keys from.
//
// Returns []string which contains all keys sorted alphabetically.
func getSortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
