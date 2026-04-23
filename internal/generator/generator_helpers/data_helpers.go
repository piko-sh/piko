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

package generator_helpers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"slices"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// maxPageDataJSONBytes caps the JSON round-trip that backs
	// GetData[T]. Real collection items are a few kilobytes at most; a
	// hard ceiling here prevents a single oversized item from
	// exhausting memory in an interpreted render.
	maxPageDataJSONBytes = 8 << 20

	// logAttrTargetType is the attribute key used across the
	// GetDataReflect diagnostic logs; pulled into a constant to keep
	// the log shape consistent and satisfy revive's add-constant rule.
	logAttrTargetType = "target_type"
)

// errPageDataTooLarge reports that the streamed JSON encoding of the
// page data would exceed maxPageDataJSONBytes before it finishes
// writing.
var errPageDataTooLarge = errors.New("page data exceeds size limit")

// boundedWriter accumulates up to limit bytes and returns
// errPageDataTooLarge once a write would push the total over. Used to
// abort streaming JSON encoders when input is oversized.
type boundedWriter struct {
	// buf collects the bytes accepted so far.
	buf bytes.Buffer

	// written tracks the cumulative byte count for comparison with
	// limit on each Write.
	written int64

	// limit is the maximum total byte count accepted across all
	// Write calls.
	limit int64
}

var _ io.Writer = (*boundedWriter)(nil)

// Write appends data to the buffer when the running total would still
// fit inside limit.
//
// Takes data ([]byte) which is the chunk the encoder wants to write.
//
// Returns the number of bytes accepted and errPageDataTooLarge when
// the limit would be exceeded.
func (w *boundedWriter) Write(data []byte) (int, error) {
	remaining := w.limit - w.written
	if int64(len(data)) > remaining {
		if remaining > 0 {
			if _, err := w.buf.Write(data[:remaining]); err != nil {
				return 0, err
			}
			w.written += remaining
		}
		return 0, errPageDataTooLarge
	}
	written, err := w.buf.Write(data)
	w.written += int64(written)
	return written, err
}

// GetData retrieves the page data from CollectionData and converts it to type T.
//
// This function is called at runtime by generated code when using
// piko.GetData[T](r) in the Render function of collection page templates.
// It extracts the "page" key from the root collection data map and performs
// type-safe conversion.
//
// Takes r (*templater_dto.RequestData) which contains the CollectionData map.
//
// Returns T which is the page data converted to the requested type, or the
// zero value if conversion fails.
func GetData[T any](r *templater_dto.RequestData) T {
	result, _ := GetDataReflect(r, reflect.TypeFor[T]())
	value, ok := result.Interface().(T)
	if !ok {
		_, l := logger_domain.From(r.Context(), log)
		l.Warn("GetData type assertion failed; returning zero value",
			logger_domain.String(logAttrTargetType, reflect.TypeFor[T]().String()),
			logger_domain.String("actual_type", result.Type().String()))
		var zero T
		return zero
	}
	return value
}

// GetDataReflect is the non-generic core used by both GetData and its
// //piko:link sibling GetDataLink at interpreter runtime.
//
// Takes r (*templater_dto.RequestData) which contains the
// CollectionData map.
// Takes tType (reflect.Type) which is the target instantiated type T.
//
// Returns a reflect.Value of concrete type tType, either zero or
// populated from the collection's "page" map via json round-trip, plus
// a bool reporting whether population succeeded.
func GetDataReflect(r *templater_dto.RequestData, tType reflect.Type) (reflect.Value, bool) {
	zero := reflect.New(tType).Elem()
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	pageData, ok := extractPageData(ctx, r, tType)
	if !ok {
		return zero, false
	}
	return decodePageData(ctx, pageData, tType, zero)
}

// extractPageData returns the `page` entry from the request's
// CollectionData map, logging a single warn line on every failure
// path so dev-i operators can see which hop in the chain misfired.
//
// Takes r (*templater_dto.RequestData) which carries the collection
// data.
// Takes tType (reflect.Type) which is logged for diagnostic context.
//
// Returns the page payload and true when the request contained a
// well-formed CollectionData map with a `page` key; otherwise nil and
// false.
func extractPageData(ctx context.Context, r *templater_dto.RequestData, tType reflect.Type) (any, bool) {
	_, l := logger_domain.From(ctx, log)
	if r == nil {
		l.Warn("GetDataReflect received a nil RequestData",
			logger_domain.String(logAttrTargetType, tType.String()))
		return nil, false
	}

	collection := r.CollectionData()
	if collection == nil {
		l.Warn("RequestData.CollectionData() is nil; Render called without WithCollectionData",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.String("url_path", safeURLPath(r)))
		return nil, false
	}

	rootMap, ok := collection.(map[string]any)
	if !ok {
		l.Warn("CollectionData is not map[string]any",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.String("actual_type", reflect.TypeOf(collection).String()))
		return nil, false
	}

	pageData, exists := rootMap["page"]
	if !exists {
		l.Warn("CollectionData has no 'page' key",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.Strings("available_keys", collectionKeys(rootMap)))
		return nil, false
	}
	return pageData, true
}

// decodePageData converts the raw page payload into a reflect.Value of
// tType, short-circuiting when the payload already has the target type.
//
// Takes pageData (any) which is the value extracted from the "page"
// slot.
// Takes tType (reflect.Type) which is the target type.
// Takes zero (reflect.Value) which is returned on every failure path.
//
// Returns the populated reflect.Value and true on success, or zero and
// false with a diagnostic log on any failure.
func decodePageData(ctx context.Context, pageData any, tType reflect.Type, zero reflect.Value) (reflect.Value, bool) {
	if pageData != nil {
		pageValue := reflect.ValueOf(pageData)
		if pageValue.IsValid() && pageValue.Type() == tType {
			return pageValue, true
		}
	}

	_, l := logger_domain.From(ctx, log)
	pageMap, ok := pageData.(map[string]any)
	if !ok {
		l.Warn("'page' value is not map[string]any",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.String("actual_page_type", reflect.TypeOf(pageData).String()))
		return zero, false
	}

	encoded, err := marshalBounded(pageMap, maxPageDataJSONBytes)
	if err != nil {
		if errors.Is(err, errPageDataTooLarge) {
			l.Warn("Page data exceeds size limit; refusing to unmarshal",
				logger_domain.String(logAttrTargetType, tType.String()),
				logger_domain.Int("limit_bytes", maxPageDataJSONBytes))
			return zero, false
		}
		l.Warn("Marshalling page map failed",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.Error(err))
		return zero, false
	}

	result := reflect.New(tType)
	if err := json.Unmarshal(encoded, result.Interface()); err != nil {
		l.Warn("Unmarshalling page map into target type failed",
			logger_domain.String(logAttrTargetType, tType.String()),
			logger_domain.Error(err))
		return zero, false
	}

	return result.Elem(), true
}

// safeURLPath returns r.URL().Path when available, or a placeholder
// when the request has no URL (typical in unit tests that build a
// bare RequestData).
//
// Takes r (*templater_dto.RequestData) which may or may not hold a
// URL.
//
// Returns the URL path string, or "<nil-url>".
func safeURLPath(r *templater_dto.RequestData) string {
	url := r.URL()
	if url == nil {
		return "<nil-url>"
	}
	return url.Path
}

// collectionKeys extracts the top-level keys of a CollectionData map
// for inclusion in diagnostic log output when the expected "page" key
// is missing. Keys are sorted for deterministic log output.
//
// Takes rootMap (map[string]any) which is the CollectionData map.
//
// Returns the sorted list of keys.
func collectionKeys(rootMap map[string]any) []string {
	keys := make([]string, 0, len(rootMap))
	for key := range rootMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// marshalBounded encodes value as JSON into a buffer that aborts
// encoding when the byte budget is exceeded. Streaming through a
// bounded writer avoids the full-size allocation a direct
// json.Marshal would make on hostile input before size checks run.
//
// Takes value (any) which is the JSON-serialisable payload.
// Takes limitBytes (int) which is the maximum accepted encoded size.
//
// Returns the encoded bytes and nil on success; nil and
// errPageDataTooLarge when the limit is hit mid-stream; nil and the
// encoder's error otherwise.
func marshalBounded(value any, limitBytes int) ([]byte, error) {
	writer := &boundedWriter{limit: int64(limitBytes)}
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(value); err != nil {
		if errors.Is(err, errPageDataTooLarge) {
			return nil, errPageDataTooLarge
		}
		return nil, err
	}
	return bytes.TrimRight(writer.buf.Bytes(), "\n"), nil
}
