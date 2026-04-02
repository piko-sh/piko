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

package ast_domain

// Defines literal expression types for constant values including strings, numbers, booleans, dates, times, and durations.
// Provides implementations for all literal types with cloning, transformation, and string representation methods for template expressions.

import (
	"maps"
	"slices"
	"strconv"
	"strings"
)

// literalQuote is the single quote character used to wrap literal values.
const literalQuote = "'"

// StringLiteral represents a string constant in an expression.
type StringLiteral struct {
	// GoAnnotations holds the Go code generator annotation for this string literal.
	GoAnnotations *GoGeneratorAnnotation

	// Value holds the string content of this literal.
	Value string

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// String returns a quoted form of the literal value.
//
// Returns string which is the value wrapped in double quotes with escaping.
func (sl *StringLiteral) String() string {
	return strconv.Quote(sl.Value)
}

// GetSourceLength returns the byte length of this expression in the original
// source.
//
// Returns int which is the number of bytes the expression takes in source.
func (sl *StringLiteral) GetSourceLength() int { return sl.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers
// transformed. For literals this is a no-op as they contain no identifiers,
// but the annotation is kept.
//
// Takes f (func(...)) which transforms identifier names.
//
// Returns Expression which is a copy of the literal with annotations intact.
func (sl *StringLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &StringLiteral{
		Value:            sl.Value,
		RelativeLocation: sl.RelativeLocation,
		GoAnnotations:    sl.GoAnnotations,
		SourceLength:     sl.SourceLength,
	}
}

// Clone creates a deep copy of the StringLiteral.
//
// Returns Expression which is the cloned literal, or nil if the receiver is
// nil.
func (sl *StringLiteral) Clone() Expression {
	if sl == nil {
		return nil
	}

	return &StringLiteral{
		Value:            sl.Value,
		RelativeLocation: sl.RelativeLocation,
		GoAnnotations:    sl.GoAnnotations.Clone(),
		SourceLength:     sl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this literal in the source.
func (sl *StringLiteral) GetRelativeLocation() Location { return sl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span in characters.
func (sl *StringLiteral) SetLocation(location Location, length int) {
	sl.RelativeLocation, sl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation or nil if none set.
func (sl *StringLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return sl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to attach.
func (sl *StringLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	sl.GoAnnotations = ann
}

// IntegerLiteral represents an integer constant in an expression.
type IntegerLiteral struct {
	// GoAnnotations holds metadata for the Go code generator.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// Value holds the parsed integer as a 64-bit signed number.
	Value int64

	// SourceLength is the number of bytes this literal takes up in the source code.
	SourceLength int
}

// String returns the integer literal as text.
//
// Returns string which is the value as a base-10 number.
func (il *IntegerLiteral) String() string {
	return strconv.FormatInt(il.Value, 10)
}

// GetSourceLength returns the byte length of this expression in the original
// source.
//
// Returns int which is the number of bytes the expression takes up.
func (il *IntegerLiteral) GetSourceLength() int { return il.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers
// transformed. This is a no-op for integer literals as they contain no
// identifiers.
//
// Takes f (func(...)) which transforms identifier names. This is ignored for
// literals.
//
// Returns Expression which is a copy of this literal.
func (il *IntegerLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &IntegerLiteral{
		Value:            il.Value,
		RelativeLocation: il.RelativeLocation,
		GoAnnotations:    il.GoAnnotations,
		SourceLength:     il.SourceLength,
	}
}

// Clone creates a deep copy of the IntegerLiteral.
//
// Returns Expression which is the cloned literal, or nil if the receiver is
// nil.
func (il *IntegerLiteral) Clone() Expression {
	if il == nil {
		return nil
	}

	return &IntegerLiteral{
		Value:            il.Value,
		RelativeLocation: il.RelativeLocation,
		GoAnnotations:    il.GoAnnotations.Clone(),
		SourceLength:     il.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this literal in the source.
func (il *IntegerLiteral) GetRelativeLocation() Location { return il.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the length of the expression in bytes.
func (il *IntegerLiteral) SetLocation(location Location, length int) {
	il.RelativeLocation, il.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which contains the generation metadata.
func (il *IntegerLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return il.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (il *IntegerLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	il.GoAnnotations = ann
}

// FloatLiteral represents a floating-point number in an expression.
type FloatLiteral struct {
	// GoAnnotations holds Go code generator annotations for this float literal.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// Value is the floating-point number parsed from the source.
	Value float64

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// GetSourceLength returns the length in bytes of this expression in the
// original source.
//
// Returns int which is the byte length of the expression.
func (fl *FloatLiteral) GetSourceLength() int { return fl.SourceLength }

// String returns the text form of the float literal.
//
// Returns string which is the decimal form of the float value.
func (fl *FloatLiteral) String() string {
	return strconv.FormatFloat(fl.Value, 'f', -1, 64)
}

// TransformIdentifiers returns a copy of this literal with identifiers
// changed. For float literals, this has no effect as they contain no
// identifiers.
//
// Takes f (func(...)) which changes identifier names.
//
// Returns Expression which is a copy of this literal, unchanged.
func (fl *FloatLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &FloatLiteral{
		Value:            fl.Value,
		RelativeLocation: fl.RelativeLocation,
		GoAnnotations:    fl.GoAnnotations,
		SourceLength:     fl.SourceLength,
	}
}

// Clone creates a deep copy of the FloatLiteral.
//
// Returns Expression which is a new FloatLiteral with copied values, or nil
// if the receiver is nil.
func (fl *FloatLiteral) Clone() Expression {
	if fl == nil {
		return nil
	}

	return &FloatLiteral{
		Value:            fl.Value,
		RelativeLocation: fl.RelativeLocation,
		GoAnnotations:    fl.GoAnnotations.Clone(),
		SourceLength:     fl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this literal in the source.
func (fl *FloatLiteral) GetRelativeLocation() Location { return fl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the span of the expression in bytes.
func (fl *FloatLiteral) SetLocation(location Location, length int) {
	fl.RelativeLocation, fl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (fl *FloatLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return fl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (fl *FloatLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	fl.GoAnnotations = ann
}

// DecimalLiteral for high-precision numbers.
// The value is stored as a string to preserve the exact precision from the source.
type DecimalLiteral struct {
	// GoAnnotations holds Go-specific generator settings; nil uses defaults.
	GoAnnotations *GoGeneratorAnnotation

	// Value holds the decimal number as it appears in the source code.
	Value string

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the length in bytes of this literal in the source code.
	SourceLength int
}

// String returns the string representation of the decimal literal with suffix.
//
// Returns string which is the value followed by the decimal suffix.
func (dl *DecimalLiteral) String() string {
	return dl.Value + decimalSuffix
}

// GetSourceLength returns the byte length of this decimal literal as it
// appeared in the original source.
//
// Returns int which is the number of bytes.
func (dl *DecimalLiteral) GetSourceLength() int { return dl.SourceLength }

// TransformIdentifiers returns a copy of this literal with identifiers
// transformed. This has no effect for decimal literals as they contain no
// identifiers.
//
// Takes f (func(...)) which transforms identifier names.
//
// Returns Expression which is a copy of this literal unchanged.
func (dl *DecimalLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &DecimalLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations,
		SourceLength:     dl.SourceLength,
	}
}

// Clone creates a deep copy of the DecimalLiteral.
//
// Returns Expression which is a new DecimalLiteral with copied values, or nil
// if the receiver is nil.
func (dl *DecimalLiteral) Clone() Expression {
	if dl == nil {
		return nil
	}

	return &DecimalLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations.Clone(),
		SourceLength:     dl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this decimal literal in the source.
func (dl *DecimalLiteral) GetRelativeLocation() Location { return dl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the span of the expression in bytes.
func (dl *DecimalLiteral) SetLocation(location Location, length int) {
	dl.RelativeLocation, dl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (dl *DecimalLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return dl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (dl *DecimalLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	dl.GoAnnotations = ann
}

// BigIntLiteral for arbitrary-precision integers.
// The value is stored as a string to preserve the exact precision from the source.
type BigIntLiteral struct {
	// GoAnnotations holds settings for Go code generation.
	GoAnnotations *GoGeneratorAnnotation

	// Value holds the string form of the big integer literal as written in
	// source.
	Value string

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// String returns the string representation of the big integer literal with
// suffix.
//
// Returns string which is the value followed by the big integer suffix.
func (bil *BigIntLiteral) String() string {
	return bil.Value + bigIntSuffix
}

// GetSourceLength returns the length of this expression in bytes as it
// appears in the original source code.
//
// Returns int which is the byte length of the expression.
func (bil *BigIntLiteral) GetSourceLength() int { return bil.SourceLength }

// TransformIdentifiers returns a copy of this literal with identifiers
// changed. This has no effect for literals as they contain no identifiers.
//
// Takes f (func(...)) which changes identifier names.
//
// Returns Expression which is a copy of this literal, unchanged.
func (bil *BigIntLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &BigIntLiteral{
		Value:            bil.Value,
		RelativeLocation: bil.RelativeLocation,
		GoAnnotations:    bil.GoAnnotations,
		SourceLength:     bil.SourceLength,
	}
}

// Clone creates a deep copy of the BigIntLiteral.
//
// Returns Expression which is a copy of the receiver, or nil if the receiver
// is nil.
func (bil *BigIntLiteral) Clone() Expression {
	if bil == nil {
		return nil
	}
	return &BigIntLiteral{
		Value:            bil.Value,
		RelativeLocation: bil.RelativeLocation,
		GoAnnotations:    bil.GoAnnotations.Clone(),
		SourceLength:     bil.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing element.
func (bil *BigIntLiteral) GetRelativeLocation() Location { return bil.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the length of the source text.
func (bil *BigIntLiteral) SetLocation(location Location, length int) {
	bil.RelativeLocation, bil.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (bil *BigIntLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return bil.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (bil *BigIntLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	bil.GoAnnotations = ann
}

// DateTimeLiteral holds an absolute date and time value.
// The value is stored as an ISO 8601 formatted string.
type DateTimeLiteral struct {
	// GoAnnotations holds settings for Go code generation for this literal.
	GoAnnotations *GoGeneratorAnnotation

	// Value is the datetime string in RFC3339 format.
	Value string

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// String returns the string form of the datetime literal.
//
// Returns string which is the datetime value with "dt" prefix and quotes.
func (dtl *DateTimeLiteral) String() string {
	return "dt" + literalQuote + dtl.Value + literalQuote
}

// GetSourceLength returns the length of this expression in bytes as it
// appears in the original source code.
//
// Returns int which is the byte length of the expression.
func (dtl *DateTimeLiteral) GetSourceLength() int { return dtl.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers
// transformed. This is a no-op for date-time literals as they contain no
// identifiers.
//
// Takes f (func(...)) which transforms identifier strings. This parameter is
// unused for literals.
//
// Returns Expression which is a new DateTimeLiteral with the same values.
func (dtl *DateTimeLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &DateTimeLiteral{
		Value:            dtl.Value,
		RelativeLocation: dtl.RelativeLocation,
		GoAnnotations:    dtl.GoAnnotations,
		SourceLength:     dtl.SourceLength,
	}
}

// Clone creates a deep copy of the DateTimeLiteral.
//
// Returns Expression which is a new DateTimeLiteral with copied values, or nil
// if the receiver is nil.
func (dtl *DateTimeLiteral) Clone() Expression {
	if dtl == nil {
		return nil
	}

	return &DateTimeLiteral{
		Value:            dtl.Value,
		RelativeLocation: dtl.RelativeLocation,
		GoAnnotations:    dtl.GoAnnotations.Clone(),
		SourceLength:     dtl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this literal in the source.
func (dtl *DateTimeLiteral) GetRelativeLocation() Location { return dtl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span of this node in the source.
func (dtl *DateTimeLiteral) SetLocation(location Location, length int) {
	dtl.RelativeLocation, dtl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (dtl *DateTimeLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return dtl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (dtl *DateTimeLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	dtl.GoAnnotations = ann
}

// DurationLiteral for time duration values.
// The value is stored as a string parsable by time.ParseDuration.
type DurationLiteral struct {
	// GoAnnotations holds settings for Go code generation for this literal.
	GoAnnotations *GoGeneratorAnnotation

	// Value is the duration string as written in the source code.
	Value string

	// RelativeLocation is the position of this node relative to its parent.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source text.
	SourceLength int
}

// String returns the string representation of the duration literal.
//
// Returns string which is the formatted duration with prefix and quotes.
func (dl *DurationLiteral) String() string {
	return "du" + literalQuote + dl.Value + literalQuote
}

// GetSourceLength returns the byte length of this expression in the source.
//
// Returns int which is the byte length of the source text.
func (dl *DurationLiteral) GetSourceLength() int { return dl.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers changed.
// This is a no-op for literals as they contain no identifiers to transform.
//
// Takes f (func(...)) which would change identifier names (unused for literals).
//
// Returns Expression which is a new DurationLiteral copy with the same values.
func (dl *DurationLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &DurationLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations,
		SourceLength:     dl.SourceLength,
	}
}

// Clone creates a deep copy of the DurationLiteral.
//
// Returns Expression which is the cloned literal, or nil if the receiver is
// nil.
func (dl *DurationLiteral) Clone() Expression {
	if dl == nil {
		return nil
	}

	return &DurationLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations.Clone(),
		SourceLength:     dl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this node relative to its parent.
func (dl *DurationLiteral) GetRelativeLocation() Location { return dl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the length of the source text.
func (dl *DurationLiteral) SetLocation(location Location, length int) {
	dl.RelativeLocation, dl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for this node, or nil
// if none is set.
func (dl *DurationLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return dl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (dl *DurationLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	dl.GoAnnotations = ann
}

// DateLiteral represents an absolute date value without a time component.
// The value is stored as a string in YYYY-MM-DD format.
type DateLiteral struct {
	// GoAnnotations holds settings for Go code generation.
	GoAnnotations *GoGeneratorAnnotation

	// Value is the date string in YYYY-MM-DD format.
	Value string

	// RelativeLocation is the position of this literal in the source.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// String returns the string representation of the date literal.
//
// Returns string which is the date literal formatted with a "d" prefix and
// quotes.
func (dl *DateLiteral) String() string {
	return "d" + literalQuote + dl.Value + literalQuote
}

// GetSourceLength returns the length in bytes of this expression in the
// original source.
//
// Returns int which is the byte length of the expression.
func (dl *DateLiteral) GetSourceLength() int { return dl.SourceLength }

// TransformIdentifiers returns a copy of this date literal unchanged.
// This is a no-op for literals as they contain no identifiers.
//
// Takes f (func(...)) which transforms identifier names.
//
// Returns Expression which is a copy of this date literal.
func (dl *DateLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &DateLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations,
		SourceLength:     dl.SourceLength,
	}
}

// Clone creates a deep copy of the DateLiteral.
//
// Returns Expression which is a new DateLiteral with copied values, or nil if
// the receiver is nil.
func (dl *DateLiteral) Clone() Expression {
	if dl == nil {
		return nil
	}

	return &DateLiteral{
		Value:            dl.Value,
		RelativeLocation: dl.RelativeLocation,
		GoAnnotations:    dl.GoAnnotations.Clone(),
		SourceLength:     dl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this date literal in the source.
func (dl *DateLiteral) GetRelativeLocation() Location { return dl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the span of this node in the source.
func (dl *DateLiteral) SetLocation(location Location, length int) {
	dl.RelativeLocation, dl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (dl *DateLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return dl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (dl *DateLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	dl.GoAnnotations = ann
}

// RuneLiteral represents a single character literal in the AST.
// It implements the Expression interface and stores the value as a rune.
type RuneLiteral struct {
	// GoAnnotations holds the Go code generator annotation for this rune literal.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// Value is the rune character that this literal represents.
	Value rune

	// SourceLength is the number of bytes this literal takes in the source code.
	SourceLength int
}

// String returns the string form of the rune literal.
//
// Returns string which is the rune value with an "r" prefix.
func (rl *RuneLiteral) String() string {
	return "r" + strconv.QuoteRune(rl.Value)
}

// GetSourceLength returns the byte length of this rune literal in the source.
//
// Returns int which is the number of bytes this literal takes in the source.
func (rl *RuneLiteral) GetSourceLength() int { return rl.SourceLength }

// TransformIdentifiers returns a copy of this rune literal unchanged.
// Rune literals do not contain identifiers, so the transform function is
// not applied.
//
// Takes f (func(...)) which would transform identifier names if any existed.
//
// Returns Expression which is a copy of this rune literal.
func (rl *RuneLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &RuneLiteral{
		Value:            rl.Value,
		RelativeLocation: rl.RelativeLocation,
		GoAnnotations:    rl.GoAnnotations,
		SourceLength:     rl.SourceLength,
	}
}

// Clone creates a deep copy of the RuneLiteral.
//
// Returns Expression which is the cloned copy, or nil if the receiver is nil.
func (rl *RuneLiteral) Clone() Expression {
	if rl == nil {
		return nil
	}

	return &RuneLiteral{
		Value:            rl.Value,
		RelativeLocation: rl.RelativeLocation,
		GoAnnotations:    rl.GoAnnotations.Clone(),
		SourceLength:     rl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing block.
func (rl *RuneLiteral) GetRelativeLocation() Location { return rl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the number of bytes in the source.
func (rl *RuneLiteral) SetLocation(location Location, length int) {
	rl.RelativeLocation, rl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation, or nil if none set.
func (rl *RuneLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return rl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (rl *RuneLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	rl.GoAnnotations = ann
}

// TimeLiteral represents an absolute time-only value.
// The value is stored as an HH:mm:ss formatted string.
type TimeLiteral struct {
	// GoAnnotations holds the code generation annotation for this literal.
	GoAnnotations *GoGeneratorAnnotation

	// Value is the time in HH:MM:SS format as a string.
	Value string

	// RelativeLocation is the position in the source code for this time literal.
	RelativeLocation Location

	// SourceLength is the byte length of this time literal in the source code.
	SourceLength int
}

// String returns the text form of the time literal.
//
// Returns string which is the time value wrapped in quotes with a "t" prefix.
func (tl *TimeLiteral) String() string {
	return "t" + literalQuote + tl.Value + literalQuote
}

// GetSourceLength returns the length of this expression in bytes as it appears
// in the original source code.
//
// Returns int which is the byte length of the expression.
func (tl *TimeLiteral) GetSourceLength() int { return tl.SourceLength }

// TransformIdentifiers returns a copy of this time literal unchanged.
// This is a no-op for literals as they contain no identifiers.
//
// Takes f (func(...)) which is ignored for literals.
//
// Returns Expression which is a copy of this time literal.
func (tl *TimeLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &TimeLiteral{
		Value:            tl.Value,
		RelativeLocation: tl.RelativeLocation,
		GoAnnotations:    tl.GoAnnotations,
		SourceLength:     tl.SourceLength,
	}
}

// Clone creates a deep copy of the TimeLiteral.
//
// Returns Expression which is the cloned literal, or nil if the receiver is
// nil.
func (tl *TimeLiteral) Clone() Expression {
	if tl == nil {
		return nil
	}

	return &TimeLiteral{
		Value:            tl.Value,
		RelativeLocation: tl.RelativeLocation,
		GoAnnotations:    tl.GoAnnotations.Clone(),
		SourceLength:     tl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position in the source code.
func (tl *TimeLiteral) GetRelativeLocation() Location { return tl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the length of the source text.
func (tl *TimeLiteral) SetLocation(location Location, length int) {
	tl.RelativeLocation, tl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation.
func (tl *TimeLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return tl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to attach.
func (tl *TimeLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	tl.GoAnnotations = ann
}

// BooleanLiteral represents a boolean value (true or false) in an expression.
type BooleanLiteral struct {
	// GoAnnotations holds settings for Go code generation; nil means use defaults.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// Value holds the boolean value from the source: true or false.
	Value bool

	// SourceLength is the number of bytes this literal spans in the source code.
	SourceLength int
}

// String returns the string representation of the boolean literal.
//
// Returns string which is "true" or "false" based on the literal's value.
func (bl *BooleanLiteral) String() string {
	return strconv.FormatBool(bl.Value)
}

// GetSourceLength returns the length of this expression in the source code.
//
// Returns int which is the number of bytes this expression spans.
func (bl *BooleanLiteral) GetSourceLength() int { return bl.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers
// changed. Boolean literals have no identifiers, so the copy is the same.
//
// Takes f (func(...)) which changes identifier names.
//
// Returns Expression which is a copy of this literal with no changes.
func (bl *BooleanLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &BooleanLiteral{
		Value:            bl.Value,
		RelativeLocation: bl.RelativeLocation,
		GoAnnotations:    bl.GoAnnotations,
		SourceLength:     bl.SourceLength,
	}
}

// Clone creates a deep copy of the BooleanLiteral.
//
// Returns Expression which is the cloned literal, or nil if the receiver is
// nil.
func (bl *BooleanLiteral) Clone() Expression {
	if bl == nil {
		return nil
	}

	return &BooleanLiteral{
		Value:            bl.Value,
		RelativeLocation: bl.RelativeLocation,
		GoAnnotations:    bl.GoAnnotations.Clone(),
		SourceLength:     bl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing element.
func (bl *BooleanLiteral) GetRelativeLocation() Location { return bl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the length in bytes.
func (bl *BooleanLiteral) SetLocation(location Location, length int) {
	bl.RelativeLocation, bl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (bl *BooleanLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return bl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to assign.
func (bl *BooleanLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	bl.GoAnnotations = ann
}

// NilLiteral represents a nil value in an expression.
// It implements the Expression interface.
type NilLiteral struct {
	// GoAnnotations holds Go-specific generator annotations; nil if none is set.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// String returns the string representation of the nil literal.
//
// Returns string which is always "nil".
func (*NilLiteral) String() string {
	return "nil"
}

// GetSourceLength returns the byte length of this expression in the source.
//
// Returns int which is the byte length of the expression in the source code.
func (nl *NilLiteral) GetSourceLength() int { return nl.SourceLength }

// TransformIdentifiers returns a copy of the literal with identifiers
// transformed. For nil literals, this returns an unchanged copy since there
// are no identifiers to transform.
//
// Takes f (func(...)) which transforms identifier names.
//
// Returns Expression which is a copy of this literal unchanged.
func (nl *NilLiteral) TransformIdentifiers(_ func(string) string) Expression {
	return &NilLiteral{
		RelativeLocation: nl.RelativeLocation,
		GoAnnotations:    nl.GoAnnotations,
		SourceLength:     nl.SourceLength,
	}
}

// Clone creates a deep copy of the NilLiteral.
//
// Returns Expression which is the cloned nil literal, or nil if the receiver
// is nil.
func (nl *NilLiteral) Clone() Expression {
	if nl == nil {
		return nil
	}

	return &NilLiteral{
		RelativeLocation: nl.RelativeLocation,
		GoAnnotations:    nl.GoAnnotations.Clone(),
		SourceLength:     nl.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing element.
func (nl *NilLiteral) GetRelativeLocation() Location { return nl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the number of source characters.
func (nl *NilLiteral) SetLocation(location Location, length int) {
	nl.RelativeLocation, nl.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (nl *NilLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return nl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (nl *NilLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	nl.GoAnnotations = ann
}

// ObjectLiteral represents a map of key-value pairs in an expression.
type ObjectLiteral struct {
	// Pairs maps property names to their expression values.
	Pairs map[string]Expression

	// GoAnnotations holds the Go code generator annotation for this
	// object literal.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the source position of this literal in
	// the original code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source
	// code.
	SourceLength int
}

// GetSourceLength returns the length of this expression in bytes as it appears
// in the original source code.
//
// Returns int which is the byte length of the expression.
func (ol *ObjectLiteral) GetSourceLength() int { return ol.SourceLength }

// String returns the text form of the object literal.
//
// Returns string which is the literal in a JSON-like format with keys in
// sorted order.
func (ol *ObjectLiteral) String() string {
	keys := slices.Sorted(maps.Keys(ol.Pairs))

	var b strings.Builder
	_, _ = b.WriteRune('{')

	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}

		b.WriteString(strconv.Quote(k))
		_, _ = b.WriteRune(':')
		_, _ = b.WriteRune(' ')
		b.WriteString(ol.Pairs[k].String())
	}

	_, _ = b.WriteRune('}')
	return b.String()
}

// TransformIdentifiers applies a function to all identifiers in the object's
// values, working through nested expressions.
//
// Takes f (func(string) string) which transforms each identifier string.
//
// Returns Expression which is a new ObjectLiteral with transformed values.
func (ol *ObjectLiteral) TransformIdentifiers(f func(string) string) Expression {
	newPairs := make(map[string]Expression, len(ol.Pairs))
	for k, v := range ol.Pairs {
		newPairs[k] = v.TransformIdentifiers(f)
	}
	return &ObjectLiteral{
		Pairs:            newPairs,
		RelativeLocation: ol.RelativeLocation,
		GoAnnotations:    ol.GoAnnotations,
		SourceLength:     ol.SourceLength,
	}
}

// Clone creates a deep copy of the ObjectLiteral.
//
// Returns Expression which is the cloned object, or nil if the receiver is
// nil.
func (ol *ObjectLiteral) Clone() Expression {
	if ol == nil {
		return nil
	}
	newPairs := make(map[string]Expression, len(ol.Pairs))
	for i, pair := range ol.Pairs {
		newPairs[i] = pair.Clone()
	}
	return &ObjectLiteral{
		Pairs:            newPairs,
		RelativeLocation: ol.RelativeLocation,
		GoAnnotations:    ol.GoAnnotations.Clone(),
		SourceLength:     ol.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the parent node.
func (ol *ObjectLiteral) GetRelativeLocation() Location { return ol.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span in characters.
func (ol *ObjectLiteral) SetLocation(location Location, length int) {
	ol.RelativeLocation, ol.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation, or nil if none is
// set.
func (ol *ObjectLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return ol.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (ol *ObjectLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	ol.GoAnnotations = ann
}

// TernaryExpression represents a conditional expression that evaluates one of two
// branches based on a condition. It implements the Expression interface.
type TernaryExpression struct {
	// Condition is the expression that decides which branch to evaluate.
	Condition Expression

	// Consequent is the expression to return when the condition is true.
	Consequent Expression

	// Alternate is the expression returned when the condition is false.
	Alternate Expression

	// GoAnnotations holds generator annotations from Go code comments.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the source location of this expression in
	// the original source.
	RelativeLocation Location

	// SourceLength is the length in bytes of this expression in the
	// source code.
	SourceLength int
}

// String returns the text form of the ternary expression.
//
// Returns string which is the expression formatted as
// "(condition ? consequent : alternate)".
func (te *TernaryExpression) String() string {
	condString := te.Condition.String()
	consString := te.Consequent.String()
	altString := te.Alternate.String()

	var builder strings.Builder
	builder.Grow(len(condString) + len(consString) + len(altString) + 7)

	_ = builder.WriteByte('(')
	builder.WriteString(condString)
	builder.WriteString(" ? ")
	builder.WriteString(consString)
	builder.WriteString(" : ")
	builder.WriteString(altString)
	_ = builder.WriteByte(')')

	return builder.String()
}

// GetSourceLength returns the byte length of this expression in the source
// code.
//
// Returns int which is the length in bytes.
func (te *TernaryExpression) GetSourceLength() int { return te.SourceLength }

// TransformIdentifiers recursively transforms all identifiers in the ternary
// expression.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new TernaryExpression with all identifiers
// transformed.
func (te *TernaryExpression) TransformIdentifiers(f func(string) string) Expression {
	return &TernaryExpression{
		Condition:        te.Condition.TransformIdentifiers(f),
		Consequent:       te.Consequent.TransformIdentifiers(f),
		Alternate:        te.Alternate.TransformIdentifiers(f),
		GoAnnotations:    te.GoAnnotations,
		RelativeLocation: te.RelativeLocation,
		SourceLength:     te.SourceLength,
	}
}

// Clone creates a deep copy of the TernaryExpression.
//
// Returns Expression which is a new TernaryExpression with all
// fields copied, or nil
// if the receiver is nil.
func (te *TernaryExpression) Clone() Expression {
	if te == nil {
		return nil
	}

	return &TernaryExpression{
		Condition:        te.Condition.Clone(),
		Consequent:       te.Consequent.Clone(),
		Alternate:        te.Alternate.Clone(),
		GoAnnotations:    te.GoAnnotations.Clone(),
		RelativeLocation: te.RelativeLocation,
		SourceLength:     te.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this node relative to its parent.
func (te *TernaryExpression) GetRelativeLocation() Location { return te.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span in characters.
func (te *TernaryExpression) SetLocation(location Location, length int) {
	te.RelativeLocation, te.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation, or nil if not set.
func (te *TernaryExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return te.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (te *TernaryExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	te.GoAnnotations = ann
}

// ArrayLiteral represents an array of expressions in the abstract syntax tree.
// It implements the Expression interface.
type ArrayLiteral struct {
	// GoAnnotations holds settings for Go code generation for this array literal.
	GoAnnotations *GoGeneratorAnnotation

	// Elements holds the expressions within the array literal.
	Elements []Expression

	// RelativeLocation is the position of this literal in the source code.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the original source.
	SourceLength int
}

// String returns the string representation of the array literal.
//
// Returns string which is the formatted array with elements separated by
// commas and enclosed in square brackets.
func (al *ArrayLiteral) String() string {
	var b strings.Builder
	_, _ = b.WriteRune('[')

	for i, element := range al.Elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(element.String())
	}

	_, _ = b.WriteRune(']')
	return b.String()
}

// GetSourceLength returns the length of the array literal in the source code.
//
// Returns int which is the byte length of this expression in the original
// source.
func (al *ArrayLiteral) GetSourceLength() int { return al.SourceLength }

// TransformIdentifiers recursively transforms all identifiers in the array's
// elements.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new ArrayLiteral with transformed elements.
func (al *ArrayLiteral) TransformIdentifiers(f func(string) string) Expression {
	newElements := make([]Expression, len(al.Elements))
	for i, element := range al.Elements {
		newElements[i] = element.TransformIdentifiers(f)
	}
	return &ArrayLiteral{
		Elements:         newElements,
		RelativeLocation: al.RelativeLocation,
		GoAnnotations:    al.GoAnnotations,
		SourceLength:     al.SourceLength,
	}
}

// Clone returns a deep copy of the array literal.
//
// Returns Expression which is the cloned array literal, or nil if the
// receiver is nil.
func (al *ArrayLiteral) Clone() Expression {
	if al == nil {
		return nil
	}
	newElements := make([]Expression, len(al.Elements))
	for i, element := range al.Elements {
		newElements[i] = element.Clone()
	}
	return &ArrayLiteral{
		Elements:         newElements,
		RelativeLocation: al.RelativeLocation,
		GoAnnotations:    al.GoAnnotations.Clone(),
		SourceLength:     al.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this array literal in the source.
func (al *ArrayLiteral) GetRelativeLocation() Location { return al.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span of this node in the source.
func (al *ArrayLiteral) SetLocation(location Location, length int) {
	al.RelativeLocation, al.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation or nil if none set.
func (al *ArrayLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return al.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to set.
func (al *ArrayLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	al.GoAnnotations = ann
}
