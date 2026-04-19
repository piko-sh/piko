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

package render_domain

import "reflect"

// cmsMediaWrapper wraps an object that has CMS media methods. Uses reflection to
// call methods, allowing duck typing without needing the type to import the render
// package.
type cmsMediaWrapper struct {
	// value holds the reflected value of the wrapped media object.
	value reflect.Value

	// urlMethod holds the reflected method used to get the media URL.
	urlMethod reflect.Value

	// widthM holds the reflected MediaWidth method, if present.
	widthM reflect.Value

	// heightM holds the reflected MediaHeight method; zero value if the
	// method is not present.
	heightM reflect.Value

	// altM holds the reflected MediaAltText method.
	altM reflect.Value

	// variantM holds the reflected MediaVariant method for fetching a single variant.
	variantM reflect.Value

	// variantsM holds the reflected MediaVariants method; zero Value if not present.
	variantsM reflect.Value
}

// MediaURL returns the primary URL for this media file.
//
// Returns string which is the URL for accessing this media.
func (w *cmsMediaWrapper) MediaURL() string {
	results := w.urlMethod.Call(nil)
	return results[0].String()
}

// MediaWidth returns the original width of the media in pixels.
//
// Returns int which is the width in pixels, or zero if the width method is
// invalid or returns no results.
func (w *cmsMediaWrapper) MediaWidth() int {
	if !w.widthM.IsValid() {
		return 0
	}
	results := w.widthM.Call(nil)
	if len(results) > 0 && results[0].CanInt() {
		return int(results[0].Int())
	}
	return 0
}

// MediaHeight returns the original height of the media in pixels.
//
// Returns int which is the height in pixels, or zero if unavailable.
func (w *cmsMediaWrapper) MediaHeight() int {
	if !w.heightM.IsValid() {
		return 0
	}
	results := w.heightM.Call(nil)
	if len(results) > 0 && results[0].CanInt() {
		return int(results[0].Int())
	}
	return 0
}

// MediaAltText returns the alt text for this media.
//
// Returns string which is the alt text, or an empty string if unavailable.
func (w *cmsMediaWrapper) MediaAltText() string {
	if !w.altM.IsValid() {
		return ""
	}
	results := w.altM.Call(nil)
	if len(results) > 0 && results[0].Kind() == reflect.String {
		return results[0].String()
	}
	return ""
}

// variantWrapper wraps a variant object to provide uniform method access.
type variantWrapper struct {
	// value holds the reflected value used to call methods at runtime.
	value reflect.Value
}

// MediaVariant returns a specific variant by name.
//
// Takes name (string) which specifies the variant to retrieve.
//
// Returns *variantWrapper which wraps the requested variant, or nil if the
// variant method is invalid or the named variant does not exist.
func (w *cmsMediaWrapper) MediaVariant(name string) *variantWrapper {
	if !w.variantM.IsValid() {
		return nil
	}
	results := w.variantM.Call([]reflect.Value{reflect.ValueOf(name)})
	if len(results) == 0 || results[0].IsNil() {
		return nil
	}
	return &variantWrapper{value: results[0]}
}

// MediaVariants returns all available variants.
//
// Returns map[string]*variantWrapper which maps variant names to their
// wrappers, or nil if no variants are available.
func (w *cmsMediaWrapper) MediaVariants() map[string]*variantWrapper {
	if !w.variantsM.IsValid() {
		return nil
	}
	results := w.variantsM.Call(nil)
	if len(results) == 0 || results[0].IsNil() {
		return nil
	}

	mapVal := results[0]
	if mapVal.Kind() != reflect.Map {
		return nil
	}

	result := make(map[string]*variantWrapper)
	iterator := mapVal.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		if key.Kind() != reflect.String {
			continue
		}
		value := iterator.Value()
		if value.IsNil() {
			continue
		}
		result[key.String()] = &variantWrapper{value: value}
	}
	return result
}

// VariantURL returns the URL for this variant.
//
// Returns string which is the variant URL, or empty if unavailable.
func (v *variantWrapper) VariantURL() string {
	m := v.value.MethodByName("VariantURL")
	if !m.IsValid() {
		return ""
	}
	results := m.Call(nil)
	if len(results) > 0 && results[0].Kind() == reflect.String {
		return results[0].String()
	}
	return ""
}

// VariantWidth returns the width of this variant.
//
// Returns int which is the variant width, or zero if unavailable.
func (v *variantWrapper) VariantWidth() int {
	m := v.value.MethodByName("VariantWidth")
	if !m.IsValid() {
		return 0
	}
	results := m.Call(nil)
	if len(results) > 0 && results[0].CanInt() {
		return int(results[0].Int())
	}
	return 0
}

// IsReady returns true if the variant is ready.
//
// Returns bool which indicates whether the underlying variant is ready.
func (v *variantWrapper) IsReady() bool {
	m := v.value.MethodByName("IsReady")
	if !m.IsValid() {
		return false
	}
	results := m.Call(nil)
	if len(results) > 0 && results[0].Kind() == reflect.Bool {
		return results[0].Bool()
	}
	return false
}

// tryCMSMediaWrapper tries to wrap an object as a CMS media source.
//
// Takes v (any) which is the object to wrap.
//
// Returns *cmsMediaWrapper which wraps the object if it has a valid MediaURL
// method, or nil if the object is nil or lacks the required method.
func tryCMSMediaWrapper(v any) *cmsMediaWrapper {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)

	urlMethod := rv.MethodByName("MediaURL")
	if !urlMethod.IsValid() {
		return nil
	}
	urlType := urlMethod.Type()
	if urlType.NumIn() != 0 || urlType.NumOut() != 1 || urlType.Out(0).Kind() != reflect.String {
		return nil
	}

	wrapper := &cmsMediaWrapper{
		value:     rv,
		urlMethod: urlMethod,
	}

	wrapper.widthM = rv.MethodByName("MediaWidth")
	wrapper.heightM = rv.MethodByName("MediaHeight")
	wrapper.altM = rv.MethodByName("MediaAltText")
	wrapper.variantM = rv.MethodByName("MediaVariant")
	wrapper.variantsM = rv.MethodByName("MediaVariants")

	return wrapper
}
