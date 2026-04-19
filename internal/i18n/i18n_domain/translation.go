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

package i18n_domain

import (
	"fmt"
	"sync"
	"time"

	"piko.sh/piko/wdk/maths"
)

// varKind represents the type of a variable slot.
type varKind uint8

const (
	// varKindString indicates a string variable for interpolation.
	varKindString varKind = iota + 1

	// varKindInt indicates an integer variable for interpolation.
	varKindInt

	// varKindFloat indicates a floating-point variable for interpolation.
	varKindFloat

	// varKindDecimal indicates a variable holds a high-precision decimal value.
	varKindDecimal

	// varKindMoney indicates a Money variable for locale-aware currency formatting.
	varKindMoney

	// varKindBigInt indicates a variable holds a BigInt value for interpolation.
	varKindBigInt

	// varKindDateTime identifies a date and time variable for locale-aware formatting.
	varKindDateTime

	// varKindTime indicates a time.Time variable for locale-aware formatting.
	varKindTime

	// varKindAny indicates a variable slot holding any type.
	varKindAny
)

// varSlot holds a single variable with its type for fast lookup without
// memory allocation.
type varSlot struct {
	// any holds complex types such as Decimal, Money, BigInt, or DateTime.
	any any

	// name is the variable name used to match a slot.
	name string

	// text holds the string value when kind is varKindString.
	text string

	// f holds the float value for varKindFloat variables.
	f float64

	// i holds the integer value for varKindInt slots.
	i int64

	// kind indicates which type of value is stored in this slot.
	kind varKind
}

const (
	// maxInlineVars is the number of variables stored inline before overflow.
	maxInlineVars = 4

	// defaultStrBufCapacity is the default buffer size used when no pool is available.
	defaultStrBufCapacity = 256
)

var (
	// translationPool is a pool of Translation structs for reuse.
	translationPool = sync.Pool{
		New: func() any {
			return &Translation{}
		},
	}

	// scopeMapPool is a pool of scope maps for reuse.
	scopeMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, maxInlineVars)
		},
	}

	_ fmt.Stringer = (*Translation)(nil)

	_ scopeProvider = (*Translation)(nil)
)

// Translation represents a translatable string with a fluent API for setting
// variables. It implements fmt.Stringer for automatic string conversion in
// templates.
type Translation struct {
	// key is the identifier used to look up the translation.
	key string

	// entry is the translation entry from lookup; nil when in literal mode.
	entry *Entry

	// literal is the fallback text used when entry is nil.
	literal string

	// pool is the buffer pool for rendering translations; nil means a new buffer
	// is created each time.
	pool *StrBufPool

	// locale specifies the locale for plural rules; empty uses the default.
	locale string

	// scope holds template variables added while building the translation.
	scope map[string]any

	// varsExtra holds extra variables when more than the fixed slots are needed.
	varsExtra []varSlot

	// vars stores variable slots directly in the struct to avoid memory
	// allocation in the common case.
	vars [maxInlineVars]varSlot

	// count holds the plural count value used to select the correct plural form.
	count int

	// varsLen is the number of variables stored in the inline buffer.
	varsLen int

	// hasCount indicates whether a count value has been set for pluralisation.
	hasCount bool
}

// NewTranslation creates a new Translation for the given key with an entry
// that has already been looked up. Use this when you already have an Entry
// from the Store.
//
// Takes key (string) which is the translation key, used as a fallback if entry
// is nil.
// Takes entry (*Entry) which is the entry already looked up from the Store.
// Takes pool (*StrBufPool) which provides buffer pooling for string building.
//
// Returns *Translation which is a pooled translation ready for use.
func NewTranslation(key string, entry *Entry, pool *StrBufPool) *Translation {
	t, ok := translationPool.Get().(*Translation)
	if !ok {
		t = &Translation{}
	}
	t.reset()
	t.key = key
	t.entry = entry
	t.pool = pool
	if entry == nil {
		t.literal = key
	}
	return t
}

// NewTranslationFromString creates a Translation from a plain string literal.
// This is used for older or fallback translations that do not come from a
// Store.
//
// Takes key (string) which identifies the translation.
// Takes literal (string) which provides the translation text.
// Takes pool (*StrBufPool) which supplies string buffers for formatting.
//
// Returns *Translation which is ready for use from a pooled instance.
func NewTranslationFromString(key, literal string, pool *StrBufPool) *Translation {
	t, ok := translationPool.Get().(*Translation)
	if !ok {
		t = &Translation{}
	}
	t.reset()
	t.key = key
	t.literal = literal
	t.pool = pool
	return t
}

// NewTranslationWithLocale creates a Translation with a specific locale for
// plural rules. Use this when you need to set the locale for plural forms.
//
// Takes key (string) which identifies the translation message.
// Takes entry (*Entry) which provides the translation data.
// Takes pool (*StrBufPool) which supplies reusable string buffers.
// Takes locale (string) which sets the locale for plural rules.
//
// Returns *Translation which is set up with the given locale.
func NewTranslationWithLocale(key string, entry *Entry, pool *StrBufPool, locale string) *Translation {
	t := NewTranslation(key, entry, pool)
	t.locale = locale
	return t
}

// Release returns the Translation to the pool for reuse. Do not use the Translation
// after calling Release.
func (t *Translation) Release() {
	translationPool.Put(t)
}

// WithLocale sets the locale for plural rules.
//
// Takes locale (string) which specifies the locale code for plural rules.
//
// Returns *Translation which is the same instance for method chaining.
func (t *Translation) WithLocale(locale string) *Translation {
	t.locale = locale
	return t
}

// StringVar sets a string variable for interpolation.
//
// Takes name (string) which identifies the variable in the translation.
// Takes value (string) which provides the string to substitute.
//
// Returns *Translation which allows method chaining.
func (t *Translation) StringVar(name, value string) *Translation {
	t.addVar(varSlot{any: nil, name: name, text: value, f: 0, i: 0, kind: varKindString})
	t.addToScope(name, value)
	return t
}

// IntVar sets an integer variable for interpolation.
//
// Takes name (string) which specifies the variable name.
// Takes value (int) which provides the integer value to assign.
//
// Returns *Translation which allows method chaining.
func (t *Translation) IntVar(name string, value int) *Translation {
	t.addVar(varSlot{any: nil, name: name, text: "", f: 0, i: int64(value), kind: varKindInt})
	t.addToScope(name, int64(value))
	return t
}

// FloatVar sets a float variable for interpolation.
//
// Takes name (string) which specifies the variable name.
// Takes value (float64) which provides the numeric value.
//
// Returns *Translation which allows method chaining.
func (t *Translation) FloatVar(name string, value float64) *Translation {
	t.addVar(varSlot{any: nil, name: name, text: "", f: value, i: 0, kind: varKindFloat})
	t.addToScope(name, value)
	return t
}

// Var sets a variable of any type for interpolation.
//
// Takes name (string) which specifies the variable name for interpolation.
// Takes value (any) which provides the value to substitute.
//
// Returns *Translation which allows method chaining.
func (t *Translation) Var(name string, value any) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindAny})
	t.addToScope(name, value)
	return t
}

// DecimalVar sets a Decimal variable for interpolation.
//
// Takes name (string) which identifies the variable in the template.
// Takes value (maths.Decimal) which provides the decimal value to substitute.
//
// Returns *Translation which allows method chaining.
func (t *Translation) DecimalVar(name string, value maths.Decimal) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindDecimal})
	t.addToScope(name, value)
	return t
}

// MoneyVar sets a Money variable for interpolation.
//
// Takes name (string) which identifies the variable in the template.
// Takes value (maths.Money) which provides the monetary value to insert.
//
// Returns *Translation which allows method chaining.
func (t *Translation) MoneyVar(name string, value maths.Money) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindMoney})
	t.addToScope(name, value)
	return t
}

// BigIntVar sets a BigInt variable for interpolation.
//
// Takes name (string) which identifies the variable in the translation.
// Takes value (maths.BigInt) which provides the BigInt value to interpolate.
//
// Returns *Translation which allows method chaining.
func (t *Translation) BigIntVar(name string, value maths.BigInt) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindBigInt})
	t.addToScope(name, value)
	return t
}

// TimeVar sets a time.Time variable for interpolation.
// The time will be formatted according to the locale with medium style.
//
// Takes name (string) which identifies the variable for substitution.
// Takes value (time.Time) which provides the time value to format.
//
// Returns *Translation which allows method chaining.
func (t *Translation) TimeVar(name string, value time.Time) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindTime})
	t.addToScope(name, value)
	return t
}

// DateTimeVar sets a DateTime variable for interpolation with custom
// formatting options. Use this when you need to specify the formatting style
// (short, medium, long, full).
//
// Takes name (string) which identifies the variable in the translation.
// Takes value (DateTime) which provides the date/time to format.
//
// Returns *Translation which allows method chaining.
func (t *Translation) DateTimeVar(name string, value DateTime) *Translation {
	t.addVar(varSlot{any: value, name: name, text: "", f: 0, i: 0, kind: varKindDateTime})
	t.addToScope(name, value)
	return t
}

// Count sets the plural count for selecting the appropriate plural form.
//
// Takes n (int) which specifies the count for plural form selection.
//
// Returns *Translation which allows method chaining.
func (t *Translation) Count(n int) *Translation {
	t.count = n
	t.hasCount = true
	return t
}

// String renders the translation with all set variables and implements
// fmt.Stringer for automatic conversion in templates.
//
// Returns string which is the rendered translation text. After calling String,
// the Translation is returned to the pool for reuse.
func (t *Translation) String() string {
	if t.entry == nil {
		result := t.literal
		t.Release()
		return result
	}

	var buffer *StrBuf
	if t.pool != nil {
		buffer = t.pool.Get()
		defer t.pool.Put(buffer)
	} else {
		buffer = NewStrBuf(defaultStrBufCapacity)
	}

	var countPtr *int
	if t.hasCount {
		countPtr = &t.count
	}
	result := renderWithVars(t.entry, t, countPtr, t.locale, buffer)
	t.Release()
	return result
}

// LookupVar finds a variable by name and writes it to the buffer.
//
// Takes name (string) which specifies the variable name to find.
// Takes buffer (*StrBuf) which receives the variable value if found.
//
// Returns bool which is true if the variable was found and written.
func (t *Translation) LookupVar(name string, buffer *StrBuf) bool {
	for i := range t.varsLen {
		if t.vars[i].name == name {
			t.writeVar(&t.vars[i], buffer)
			return true
		}
	}
	for i := range t.varsExtra {
		if t.varsExtra[i].name == name {
			t.writeVar(&t.varsExtra[i], buffer)
			return true
		}
	}
	return false
}

// GetScope returns all variables as a map[string]any for expression evaluation.
//
// Returns map[string]any which contains scope variables for V2 rendering with
// AST expressions. The scope is built incrementally as vars are added, so this
// is O(1).
func (t *Translation) GetScope() map[string]any {
	if t.hasCount {
		if t.scope == nil {
			t.scope = getScopeMap()
		}
		t.scope["count"] = t.count
	}
	return t.scope
}

// reset clears all fields and returns pooled resources to prepare for reuse.
func (t *Translation) reset() {
	t.key = ""
	t.entry = nil
	t.literal = ""
	t.pool = nil
	t.locale = ""
	if t.scope != nil {
		clear(t.scope)
		scopeMapPool.Put(t.scope)
		t.scope = nil
	}
	t.varsExtra = nil
	for i := range t.vars {
		t.vars[i] = varSlot{}
	}
	t.count = 0
	t.varsLen = 0
	t.hasCount = false
}

// addToScope adds a value to the scope map, getting one from the pool if
// needed.
//
// Takes name (string) which is the key to store the value under.
// Takes value (any) which is the value to store in the scope.
func (t *Translation) addToScope(name string, value any) {
	if t.scope == nil {
		t.scope = getScopeMap()
	}
	t.scope[name] = value
}

// addVar adds a variable slot to the translation's storage.
//
// Takes slot (varSlot) which specifies the variable slot to add.
func (t *Translation) addVar(slot varSlot) {
	if t.varsLen < maxInlineVars {
		t.vars[t.varsLen] = slot
		t.varsLen++
		return
	}
	t.varsExtra = append(t.varsExtra, slot)
}

// writeVar writes a variable value to the buffer based on its type.
//
// Takes slot (*varSlot) which holds the variable value and type details.
// Takes buffer (*StrBuf) which receives the formatted output.
func (t *Translation) writeVar(slot *varSlot, buffer *StrBuf) {
	switch slot.kind {
	case varKindString:
		buffer.WriteString(slot.text)
	case varKindInt:
		buffer.WriteInt64(slot.i)
	case varKindFloat:
		buffer.WriteFloat(slot.f)
	case varKindDecimal:
		buffer.WriteDecimal(slot.any.(maths.Decimal))
	case varKindMoney:
		buffer.WriteMoneyWithLocale(slot.any.(maths.Money), t.locale)
	case varKindBigInt:
		buffer.WriteBigInt(slot.any.(maths.BigInt))
	case varKindDateTime:
		if dt, ok := slot.any.(DateTime); ok {
			buffer.WriteString(dt.Format(t.locale))
		}
	case varKindTime:
		if tt, ok := slot.any.(time.Time); ok {
			buffer.WriteString(FormatDateTime(tt, t.locale, DateTimeStyleMedium, false, false))
		}
	case varKindAny:
		buffer.WriteAny(slot.any)
	default:
	}
}

// getScopeMap gets a scope map from the pool with safe type assertion.
//
// Returns map[string]any which is either a reused map from the pool or a new
// map with space for inline variables.
func getScopeMap() map[string]any {
	if m, ok := scopeMapPool.Get().(map[string]any); ok {
		return m
	}
	return make(map[string]any, maxInlineVars)
}
