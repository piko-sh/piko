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

// Defines the Expression interface and concrete expression types for template
// interpolations and directive values. Includes identifiers, member access,
// binary operations, function calls, literals, and ternary expressions with
// cloning and transformation support.

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// OpNot is the logical NOT operator (!).
	OpNot UnaryOp = "!"

	// OpNeg is the unary negation operator that changes the sign of a number.
	OpNeg UnaryOp = "-"

	// OpAddrOf is the address-of unary operator (&).
	OpAddrOf UnaryOp = "&"

	// OpTruthy is the truthiness operator (~) that converts a value to its boolean
	// truthiness.
	OpTruthy UnaryOp = "~"

	// OpEq represents the strict equality operator (==). It uses Go-style,
	// type-safe comparison.
	OpEq BinaryOp = "=="

	// OpNe represents the strict inequality operator (!=). It is Go-style and
	// type-safe.
	OpNe BinaryOp = "!="

	// OpLooseEq is the loose equality operator (~=). JS-style, with type coercion.
	OpLooseEq BinaryOp = "~="

	// OpLooseNe represents the loose inequality operator (!~=). It uses
	// JS-style comparison with type coercion.
	OpLooseNe BinaryOp = "!~="

	// OpGt is the greater-than comparison operator (>).
	OpGt BinaryOp = ">"

	// OpLt is the less-than comparison operator (<).
	OpLt BinaryOp = "<"

	// OpGe is the greater than or equal to operator (>=).
	OpGe BinaryOp = ">="

	// OpLe represents the less than or equal operator (<=).
	OpLe BinaryOp = "<="

	// OpAnd is the logical AND operator (&&).
	OpAnd BinaryOp = "&&"

	// OpOr is the logical OR operator (||).
	OpOr BinaryOp = "||"

	// OpPlus is the addition operator (+).
	OpPlus BinaryOp = "+"

	// OpMinus is the subtraction operator for numbers and times.
	OpMinus BinaryOp = "-"

	// OpMul is the multiplication operator (*).
	OpMul BinaryOp = "*"

	// OpDiv is the division operator (/).
	OpDiv BinaryOp = "/"

	// OpMod is the modulo operator (%).
	OpMod BinaryOp = "%"

	// OpCoalesce is the nullish coalescing operator (??).
	OpCoalesce BinaryOp = "??"
)

const (
	_ int = iota

	// precLowest is the lowest operator precedence level.
	precLowest

	// precTernary is the precedence level for ternary conditional expressions.
	precTernary

	// precOr is the precedence level for logical OR expressions.
	precOr

	// precCoalesce is the precedence level for the coalesce operator.
	precCoalesce

	// precAnd is the precedence level for logical AND expressions.
	precAnd

	// precEquals is the precedence level for equality operators.
	precEquals

	// precLessGreater is the precedence level for comparison operators.
	precLessGreater

	// precSum is the precedence level for addition and
	// subtraction operators.
	precSum

	// precProduct is the precedence level for multiplication,
	// division, and modulo operators.
	precProduct

	// precPrefix is the precedence level for unary prefix
	// expressions.
	precPrefix

	// precPostfix is the precedence level for postfix operators
	// such as function calls, indexing, and member access.
	precPostfix
)

// precedences maps operator symbols to their binding power for expression parsing.
var precedences = map[string]int{
	"?":   precTernary,
	"??":  precCoalesce,
	"||":  precOr,
	"&&":  precAnd,
	"==":  precEquals,
	"!=":  precEquals,
	"~=":  precEquals,
	"!~=": precEquals,
	"<":   precLessGreater,
	">":   precLessGreater,
	"<=":  precLessGreater,
	">=":  precLessGreater,
	"+":   precSum,
	"-":   precSum,
	"*":   precProduct,
	"/":   precProduct,
	"%":   precProduct,
	"(":   precPostfix,
	"[":   precPostfix,
	".":   precPostfix,
	"?.":  precPostfix,
}

// Expression defines the interface that all expression nodes implement.
type Expression interface {
	// String returns the string form of this expression.
	String() string

	// TransformIdentifiers walks the expression tree and applies a transformation
	// function to the name of every Identifier node.
	//
	// Takes func(string) string which transforms each identifier name.
	//
	// Returns Expression which is the transformed expression tree.
	TransformIdentifiers(func(string) string) Expression

	// Clone creates a deep copy of the expression.
	//
	// Returns Expression which is an independent copy of the original.
	Clone() Expression

	// GetRelativeLocation returns the location relative to the current context.
	//
	// Returns Location which represents the relative position.
	GetRelativeLocation() Location

	// SetLocation sets the source location and length for this node.
	//
	// Takes location (Location) which specifies the position in the source.
	// Takes length (int) which specifies the span in bytes.
	SetLocation(location Location, length int)

	// GetGoAnnotation returns the Go generator annotation for the node.
	//
	// Returns *GoGeneratorAnnotation which contains Go-specific code generation
	// settings, or nil if no annotation is set.
	GetGoAnnotation() *GoGeneratorAnnotation

	// SetGoAnnotation assigns the Go generator annotation for this element.
	//
	// Takes *GoGeneratorAnnotation which specifies the Go-specific code generation
	// settings.
	SetGoAnnotation(*GoGeneratorAnnotation)

	// GetSourceLength returns the length of the source content in bytes.
	GetSourceLength() int
}

// Identifier represents a variable or property name (e.g. "user", "name").
type Identifier struct {
	// GoAnnotations holds hints for code generation from Go comments.
	GoAnnotations *GoGeneratorAnnotation

	// Name is the identifier string.
	Name string

	// RelativeLocation is the source position of this identifier in the original
	// source file.
	RelativeLocation Location

	// SourceLength is the byte length of the identifier in the source code.
	SourceLength int
}

// String returns the name of this identifier.
//
// Returns string which is the identifier name.
func (id *Identifier) String() string { return id.Name }

// GetSourceLength returns the byte length of this identifier in the original
// source.
//
// Returns int which is the length in bytes.
func (id *Identifier) GetSourceLength() int { return id.SourceLength }

// TransformIdentifiers applies the transformation function to the
// identifier's name.
//
// Takes f (func(string) string) which transforms the identifier name.
//
// Returns Expression which is a new Identifier with the transformed name.
func (id *Identifier) TransformIdentifiers(f func(string) string) Expression {
	return &Identifier{Name: f(id.Name), GoAnnotations: id.GoAnnotations, RelativeLocation: id.RelativeLocation, SourceLength: id.SourceLength}
}

// Clone creates a deep copy of the Identifier.
//
// Returns Expression which is a new Identifier with copied values, or nil if
// the receiver is nil.
func (id *Identifier) Clone() Expression {
	if id == nil {
		return nil
	}
	return &Identifier{
		Name:             id.Name,
		GoAnnotations:    id.GoAnnotations.Clone(),
		RelativeLocation: id.RelativeLocation,
		SourceLength:     id.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing file.
func (id *Identifier) GetRelativeLocation() Location { return id.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span of the source text.
func (id *Identifier) SetLocation(location Location, length int) {
	id.RelativeLocation, id.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation, or nil if none is
// set.
func (id *Identifier) GetGoAnnotation() *GoGeneratorAnnotation {
	return id.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (id *Identifier) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	id.GoAnnotations = ann
}

// MemberExpression represents a property access such as user.name or user?.name.
type MemberExpression struct {
	// Base is the expression for the struct or pointer being accessed.
	Base Expression

	// Property is the expression that identifies the member being accessed.
	Property Expression

	// GoAnnotations holds metadata used by the Go code generator.
	GoAnnotations *GoGeneratorAnnotation

	// Optional indicates whether this is optional chaining access (?.).
	Optional bool

	// Computed indicates whether the member access uses bracket notation.
	Computed bool

	// RelativeLocation is the source position relative to the containing element.
	RelativeLocation Location

	// SourceLength is the number of bytes this expression spans in the source
	// text.
	SourceLength int
}

// String returns the text form of the member expression.
//
// Returns string which contains the base, an optional chaining operator if
// present, and the property name joined together.
func (me *MemberExpression) String() string {
	var b strings.Builder
	b.WriteString(me.Base.String())
	if me.Optional {
		b.WriteString("?.")
	} else {
		_, _ = b.WriteRune('.')
	}
	b.WriteString(me.Property.String())
	return b.String()
}

// GetSourceLength returns the byte length of this expression in the source.
//
// Returns int which is the number of bytes the expression spans.
func (me *MemberExpression) GetSourceLength() int { return me.SourceLength }

// TransformIdentifiers applies a transform function to all identifiers in the
// base and property expressions.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new MemberExpression with transformed identifiers.
func (me *MemberExpression) TransformIdentifiers(f func(string) string) Expression {
	return &MemberExpression{
		Base:             me.Base.TransformIdentifiers(f),
		Property:         me.Property.TransformIdentifiers(f),
		Optional:         me.Optional,
		Computed:         me.Computed,
		GoAnnotations:    me.GoAnnotations,
		RelativeLocation: me.RelativeLocation,
		SourceLength:     me.SourceLength,
	}
}

// Clone creates a deep copy of the MemberExpression.
//
// Returns Expression which is the cloned member expression, or nil if the
// receiver is nil.
func (me *MemberExpression) Clone() Expression {
	if me == nil {
		return nil
	}
	return &MemberExpression{
		Base:             me.Base.Clone(),
		Property:         me.Property.Clone(),
		Optional:         me.Optional,
		Computed:         me.Computed,
		GoAnnotations:    me.GoAnnotations.Clone(),
		RelativeLocation: me.RelativeLocation,
		SourceLength:     me.SourceLength,
	}
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing element.
func (me *MemberExpression) GetRelativeLocation() Location { return me.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span in the source text.
func (me *MemberExpression) SetLocation(location Location, length int) {
	me.RelativeLocation, me.SourceLength = location, length
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (me *MemberExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return me.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (me *MemberExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	me.GoAnnotations = ann
}

// IndexExpression represents an index access such as items[0] or items?.[0].
type IndexExpression struct {
	// Base is the expression being indexed.
	Base Expression

	// Index is the type argument expression inside the brackets.
	Index Expression

	// GoAnnotations holds code generation hints for this expression.
	GoAnnotations *GoGeneratorAnnotation

	// Optional indicates whether this uses optional chaining (?.[]).
	Optional bool

	// RelativeLocation is the source location of this expression in the original
	// source.
	RelativeLocation Location

	// SourceLength is the byte length of this expression in the source code.
	SourceLength int
}

// String returns the text form of the index expression.
//
// Returns string which is the formatted expression. When Optional is true,
// the result uses optional chaining syntax (?.[) instead of standard brackets.
func (ie *IndexExpression) String() string {
	var builder strings.Builder
	builder.Grow(len(ie.Base.String()) + len(ie.Index.String()) + 3)
	builder.WriteString(ie.Base.String())
	if ie.Optional {
		builder.WriteString("?.[")
	} else {
		_ = builder.WriteByte('[')
	}
	builder.WriteString(ie.Index.String())
	_ = builder.WriteByte(']')
	return builder.String()
}

// GetSourceLength returns the length of this expression in bytes as it
// appears in the original source code.
//
// Returns int which is the byte length of the expression.
func (ie *IndexExpression) GetSourceLength() int { return ie.SourceLength }

// TransformIdentifiers recursively transforms identifiers in the base and
// index expressions.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new IndexExpression with transformed identifiers.
func (ie *IndexExpression) TransformIdentifiers(f func(string) string) Expression {
	return &IndexExpression{
		Base:             ie.Base.TransformIdentifiers(f),
		Index:            ie.Index.TransformIdentifiers(f),
		Optional:         ie.Optional,
		GoAnnotations:    ie.GoAnnotations,
		RelativeLocation: ie.RelativeLocation,
		SourceLength:     ie.SourceLength,
	}
}

// Clone creates a deep copy of the IndexExpression.
//
// Returns Expression which is the cloned expression, or nil if the receiver
// is nil.
func (ie *IndexExpression) Clone() Expression {
	if ie == nil {
		return nil
	}
	return &IndexExpression{
		Base:             ie.Base.Clone(),
		Index:            ie.Index.Clone(),
		Optional:         ie.Optional,
		GoAnnotations:    ie.GoAnnotations.Clone(),
		RelativeLocation: ie.RelativeLocation,
		SourceLength:     ie.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for this node.
func (ie *IndexExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return ie.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to set.
func (ie *IndexExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	ie.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing element.
func (ie *IndexExpression) GetRelativeLocation() Location { return ie.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the length of the expression in bytes.
func (ie *IndexExpression) SetLocation(location Location, length int) {
	ie.RelativeLocation, ie.SourceLength = location, length
}

// UnaryOp represents a unary operator such as ! or -.
type UnaryOp string

// UnaryExpression represents a unary expression such as negation or logical not.
// It implements the Expression interface.
type UnaryExpression struct {
	// Right is the expression to which the unary operator applies.
	Right Expression

	// GoAnnotations holds the code generation annotation for this expression.
	GoAnnotations *GoGeneratorAnnotation

	// Operator specifies which unary operation to apply to the expression.
	Operator UnaryOp

	// RelativeLocation is the source position of this expression, stored relative
	// to the parent node.
	RelativeLocation Location

	// SourceLength is the byte length of this expression in the source code.
	SourceLength int
}

// String returns the text form of the unary expression.
//
// Returns string which is the operator followed by its operand.
func (u *UnaryExpression) String() string {
	rs := u.Right.String()
	switch u.Right.(type) {
	case *BinaryExpression, *UnaryExpression:
		rs = "(" + rs + ")"
	}
	return string(u.Operator) + rs
}

// GetSourceLength returns the byte length of this expression in the original
// source.
//
// Returns int which is the byte length of the expression.
func (u *UnaryExpression) GetSourceLength() int { return u.SourceLength }

// TransformIdentifiers applies a transform function to all identifiers in
// the right-hand expression.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new UnaryExpression with transformed identifiers.
func (u *UnaryExpression) TransformIdentifiers(f func(string) string) Expression {
	return &UnaryExpression{
		Operator:         u.Operator,
		Right:            u.Right.TransformIdentifiers(f),
		GoAnnotations:    u.GoAnnotations,
		RelativeLocation: u.RelativeLocation,
		SourceLength:     u.SourceLength,
	}
}

// Clone creates a deep copy of the UnaryExpression.
//
// Returns Expression which is the cloned expression, or nil if the receiver
// is nil.
func (u *UnaryExpression) Clone() Expression {
	if u == nil {
		return nil
	}
	return &UnaryExpression{
		Operator:         u.Operator,
		Right:            u.Right.Clone(),
		GoAnnotations:    u.GoAnnotations.Clone(),
		RelativeLocation: u.RelativeLocation,
		SourceLength:     u.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for this node, or nil
// if none is set.
func (u *UnaryExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return u.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to attach.
func (u *UnaryExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	u.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the parent node.
func (u *UnaryExpression) GetRelativeLocation() Location { return u.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the span of the expression in bytes.
func (u *UnaryExpression) SetLocation(location Location, length int) {
	u.RelativeLocation, u.SourceLength = location, length
}

// BinaryOp represents a binary operator such as +, ==, or &&.
type BinaryOp string

// BinaryExpression represents a binary operation such as a + b.
// It implements the Expression interface.
type BinaryExpression struct {
	// Left is the left-hand operand of the binary expression.
	Left Expression

	// Right is the right-hand side of the binary expression.
	Right Expression

	// GoAnnotations holds the Go code generator annotation for this expression.
	GoAnnotations *GoGeneratorAnnotation

	// Operator specifies the binary operation to apply between Left and Right.
	Operator BinaryOp

	// RelativeLocation is the position of this expression in the original source.
	RelativeLocation Location

	// SourceLength is the length of this expression in bytes within the source
	// code.
	SourceLength int
}

// GetSourceLength returns the length of this expression in bytes as it
// appears in the original source.
//
// Returns int which is the byte length of the expression.
func (b *BinaryExpression) GetSourceLength() int { return b.SourceLength }

// String returns the binary expression as text.
//
// Returns string which contains the expression in the form "(left op right)".
func (b *BinaryExpression) String() string {
	leftString := b.Left.String()
	rightString := b.Right.String()
	var builder strings.Builder
	builder.Grow(len(leftString) + len(rightString) + len(b.Operator) + 4)
	_ = builder.WriteByte('(')
	builder.WriteString(leftString)
	_ = builder.WriteByte(' ')
	builder.WriteString(string(b.Operator))
	_ = builder.WriteByte(' ')
	builder.WriteString(rightString)
	_ = builder.WriteByte(')')
	return builder.String()
}

// TransformIdentifiers applies a function to all identifiers in the left and
// right expressions of this binary expression.
//
// Takes f (func(string) string) which changes each identifier name.
//
// Returns Expression which is a new BinaryExpression with changed identifiers.
func (b *BinaryExpression) TransformIdentifiers(f func(string) string) Expression {
	return &BinaryExpression{
		Left:             b.Left.TransformIdentifiers(f),
		Operator:         b.Operator,
		Right:            b.Right.TransformIdentifiers(f),
		GoAnnotations:    b.GoAnnotations,
		RelativeLocation: b.RelativeLocation,
		SourceLength:     b.SourceLength,
	}
}

// Clone creates a deep copy of the BinaryExpression.
//
// Returns Expression which is a new BinaryExpression with all fields
// copied, or nil
// if the receiver is nil.
func (b *BinaryExpression) Clone() Expression {
	if b == nil {
		return nil
	}
	return &BinaryExpression{
		Left:             b.Left.Clone(),
		Operator:         b.Operator,
		Right:            b.Right.Clone(),
		GoAnnotations:    b.GoAnnotations.Clone(),
		RelativeLocation: b.RelativeLocation,
		SourceLength:     b.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if no annotation is set.
func (b *BinaryExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return b.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to set.
func (b *BinaryExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	b.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this node relative to its parent.
func (b *BinaryExpression) GetRelativeLocation() Location { return b.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span in characters.
func (b *BinaryExpression) SetLocation(location Location, length int) {
	b.RelativeLocation, b.SourceLength = location, length
}

// ForInExpression represents a for-in loop expression that iterates over a
// collection. It implements the Expression interface.
type ForInExpression struct {
	// IndexVariable is the optional loop index variable; nil when not used.
	IndexVariable *Identifier

	// ItemVariable is the identifier for the current item in each loop cycle.
	ItemVariable *Identifier

	// Collection is the expression that yields the items to iterate over.
	Collection Expression

	// GoAnnotations holds the Go code generation settings for this expression.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the source location of this expression in the original
	// source.
	RelativeLocation Location

	// SourceLength is the byte length of this expression in the source code.
	SourceLength int
}

// GetSourceLength returns the length in bytes of this expression in the
// original source.
//
// Returns int which is the byte length of the expression.
func (f *ForInExpression) GetSourceLength() int { return f.SourceLength }

// String returns the text form of the for-in expression.
//
// Returns string which contains the formatted expression.
func (f *ForInExpression) String() string {
	if f.IndexVariable == nil {
		return fmt.Sprintf("%s in %s", f.ItemVariable.String(), f.Collection.String())
	}
	return fmt.Sprintf("(%s, %s) in %s",
		f.IndexVariable.String(), f.ItemVariable.String(), f.Collection.String())
}

// TransformIdentifiers recursively transforms identifiers in the collection
// expression.
//
// Takes x (func(string) string) which transforms each identifier
// name.
//
// Returns Expression which is a new ForInExpression with transformed
// identifiers.
func (f *ForInExpression) TransformIdentifiers(x func(string) string) Expression {
	return &ForInExpression{
		IndexVariable:    f.IndexVariable,
		ItemVariable:     f.ItemVariable,
		Collection:       f.Collection.TransformIdentifiers(x),
		GoAnnotations:    f.GoAnnotations,
		RelativeLocation: f.RelativeLocation,
		SourceLength:     f.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation.
func (f *ForInExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return f.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (f *ForInExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	f.GoAnnotations = ann
}

// Clone creates a deep copy of the ForInExpression.
//
// Returns Expression which is the cloned expression, or nil if the receiver
// is nil.
func (f *ForInExpression) Clone() Expression {
	if f == nil {
		return nil
	}

	clone := &ForInExpression{
		IndexVariable:    nil,
		ItemVariable:     nil,
		Collection:       f.Collection.Clone(),
		GoAnnotations:    f.GoAnnotations.Clone(),
		RelativeLocation: f.RelativeLocation,
		SourceLength:     f.SourceLength,
	}

	if f.ItemVariable != nil {
		if clonedItem, ok := f.ItemVariable.Clone().(*Identifier); ok {
			clone.ItemVariable = clonedItem
		}
	}

	if f.IndexVariable != nil {
		if clonedIndex, ok := f.IndexVariable.Clone().(*Identifier); ok {
			clone.IndexVariable = clonedIndex
		}
	}

	return clone
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this node relative to its parent.
func (f *ForInExpression) GetRelativeLocation() Location { return f.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source.
// Takes length (int) which specifies the number of characters.
func (f *ForInExpression) SetLocation(location Location, length int) {
	f.RelativeLocation, f.SourceLength = location, length
}

// CallExpression represents a function or method call expression in the AST.
// It implements the Expression interface and stores the callee, arguments,
// and source location information for the call.
type CallExpression struct {
	// Callee is the expression being called.
	Callee Expression

	// GoAnnotations holds the Go code generator annotation for this call
	// expression.
	GoAnnotations *GoGeneratorAnnotation

	// Args holds the function arguments in the order they appear.
	Args []Expression

	// RelativeLocation is the source location of this expression in the original
	// source.
	RelativeLocation Location

	// LparenLocation is the position of the opening parenthesis.
	LparenLocation Location

	// RparenLocation is the position of the closing parenthesis.
	RparenLocation Location

	// SourceLength is the byte length of this expression in the source code.
	SourceLength int
}

// String returns the call expression as text.
//
// Returns string which holds the callee name followed by its arguments in
// brackets, such as "foo(x, y)".
func (c *CallExpression) String() string {
	var b strings.Builder
	b.WriteString(c.Callee.String())
	_, _ = b.WriteRune('(')
	for i, argument := range c.Args {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(argument.String())
	}
	_, _ = b.WriteRune(')')
	return b.String()
}

// GetSourceLength returns the byte length of this expression in the source.
//
// Returns int which is the byte length of the expression.
func (c *CallExpression) GetSourceLength() int { return c.SourceLength }

// TransformIdentifiers recursively transforms identifiers in the callee and
// arguments.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new CallExpression with all
// identifiers transformed.
func (c *CallExpression) TransformIdentifiers(f func(string) string) Expression {
	newCallee := c.Callee.TransformIdentifiers(f)
	newArgs := make([]Expression, len(c.Args))
	for i, a := range c.Args {
		newArgs[i] = a.TransformIdentifiers(f)
	}
	return &CallExpression{
		Callee:           newCallee,
		Args:             newArgs,
		GoAnnotations:    c.GoAnnotations,
		RelativeLocation: c.RelativeLocation,
		LparenLocation:   c.LparenLocation,
		RparenLocation:   c.RparenLocation,
		SourceLength:     c.SourceLength,
	}
}

// Clone creates a deep copy of the CallExpression.
//
// Returns Expression which is a new CallExpression with all fields copied, or nil
// if the receiver is nil.
func (c *CallExpression) Clone() Expression {
	if c == nil {
		return nil
	}
	newArgs := make([]Expression, len(c.Args))
	for i, argument := range c.Args {
		newArgs[i] = argument.Clone()
	}
	return &CallExpression{
		Callee:           c.Callee.Clone(),
		Args:             newArgs,
		GoAnnotations:    c.GoAnnotations.Clone(),
		RelativeLocation: c.RelativeLocation,
		LparenLocation:   c.LparenLocation,
		RparenLocation:   c.RparenLocation,
		SourceLength:     c.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation, or nil if none.
func (c *CallExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return c.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to assign.
func (c *CallExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	c.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position relative to the containing file.
func (c *CallExpression) GetRelativeLocation() Location { return c.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the length in bytes.
func (c *CallExpression) SetLocation(location Location, length int) {
	c.RelativeLocation, c.SourceLength = location, length
}

// TemplateLiteralPart represents a single segment of a template literal.
// Each part is either a plain string or a parsed expression.
type TemplateLiteralPart struct {
	// Expression is the parsed expression for this part; nil when IsLiteral is
	// true.
	Expression Expression

	// Literal holds the raw string content of this template part.
	Literal string

	// IsLiteral indicates whether this part is a literal string segment.
	IsLiteral bool

	// RelativeLocation is the position of this part within the template literal.
	RelativeLocation Location
}

// Clone creates a deep copy of the TemplateLiteralPart.
//
// Returns TemplateLiteralPart which is a copy with cloned expression.
func (tlp *TemplateLiteralPart) Clone() TemplateLiteralPart {
	clone := TemplateLiteralPart{
		Expression:       nil,
		Literal:          tlp.Literal,
		IsLiteral:        tlp.IsLiteral,
		RelativeLocation: tlp.RelativeLocation,
	}

	if !tlp.IsLiteral && tlp.Expression != nil {
		clone.Expression = tlp.Expression.Clone()
	}

	return clone
}

// GetRelativeLocation returns the source location of this part.
//
// Returns Location which is the position relative to the template literal.
func (tlp *TemplateLiteralPart) GetRelativeLocation() Location { return tlp.RelativeLocation }

// TemplateLiteral represents a template string such as `Hello, ${user.name}`.
// It implements the Expression interface.
type TemplateLiteral struct {
	// GoAnnotations holds the Go code generator annotation for this template
	// literal.
	GoAnnotations *GoGeneratorAnnotation

	// Parts holds the template literal segments in order.
	Parts []TemplateLiteralPart

	// RelativeLocation is the position of this literal in the source file.
	RelativeLocation Location

	// SourceLength is the byte length of this literal in the source code.
	SourceLength int
}

// GetSourceLength returns the length of this expression in bytes as it
// appears in the original source.
//
// Returns int which is the byte length of the expression.
func (tl *TemplateLiteral) GetSourceLength() int { return tl.SourceLength }

// String returns the template literal as a string.
//
// Returns string which is the template literal text with backticks and
// interpolation markers (${}) escaped.
func (tl *TemplateLiteral) String() string {
	var b strings.Builder
	_, _ = b.WriteRune('`')
	for _, part := range tl.Parts {
		if part.IsLiteral {
			s := strings.ReplaceAll(part.Literal, "`", "\\`")
			s = strings.ReplaceAll(s, "${", "\\${")
			b.WriteString(s)
		} else {
			b.WriteString("${")
			innerString := part.Expression.String()

			if _, ok := part.Expression.(*StringLiteral); ok {
				if unquoted, err := strconv.Unquote(innerString); err == nil {
					innerString = unquoted
				}
			}
			b.WriteString(innerString)
			b.WriteString("}")
		}
	}
	_, _ = b.WriteRune('`')
	return b.String()
}

// TransformIdentifiers applies a transform function to all identifiers within
// the expression parts of this template literal.
//
// Takes f (func(string) string) which transforms each identifier name.
//
// Returns Expression which is a new TemplateLiteral with transformed parts.
func (tl *TemplateLiteral) TransformIdentifiers(f func(string) string) Expression {
	newParts := make([]TemplateLiteralPart, len(tl.Parts))
	for i, part := range tl.Parts {
		if part.IsLiteral {
			newParts[i] = part
		} else {
			newParts[i] = TemplateLiteralPart{
				Expression:       part.Expression.TransformIdentifiers(f),
				Literal:          "",
				IsLiteral:        false,
				RelativeLocation: part.RelativeLocation,
			}
		}
	}
	return &TemplateLiteral{Parts: newParts, GoAnnotations: tl.GoAnnotations, RelativeLocation: tl.RelativeLocation, SourceLength: tl.SourceLength}
}

// Clone creates a deep copy of the TemplateLiteral.
//
// Returns Expression which is the cloned template literal, or nil if the
// receiver is nil.
func (tl *TemplateLiteral) Clone() Expression {
	if tl == nil {
		return nil
	}

	newParts := make([]TemplateLiteralPart, len(tl.Parts))
	for i, part := range tl.Parts {
		newParts[i] = part.Clone()
	}

	return &TemplateLiteral{
		Parts:            newParts,
		GoAnnotations:    tl.GoAnnotations.Clone(),
		RelativeLocation: tl.RelativeLocation,
		SourceLength:     tl.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation,
// or nil if none is set.
func (tl *TemplateLiteral) GetGoAnnotation() *GoGeneratorAnnotation {
	return tl.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to set.
func (tl *TemplateLiteral) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	tl.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which specifies the position within the source file.
func (tl *TemplateLiteral) GetRelativeLocation() Location { return tl.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the position in the source code.
// Takes length (int) which specifies the number of characters in the source.
func (tl *TemplateLiteral) SetLocation(location Location, length int) {
	tl.RelativeLocation, tl.SourceLength = location, length
}

// LinkedMessageExpression represents a linked message reference in i18n templates
// (e.g., @common.greeting). The @ operator references other translation keys.
type LinkedMessageExpression struct {
	// Path is the path expression, either an Identifier or a MemberExpression chain.
	Path Expression

	// GoAnnotations holds the Go code generation settings.
	GoAnnotations *GoGeneratorAnnotation

	// RelativeLocation is the source location of this expression in the original
	// source.
	RelativeLocation Location

	// SourceLength is the number of bytes this expression spans in the source.
	SourceLength int
}

// String returns the string form of the linked message expression.
//
// Returns string which is the path with an "@" prefix.
func (lm *LinkedMessageExpression) String() string {
	return "@" + lm.Path.String()
}

// GetSourceLength returns the byte length of this expression in the original
// source.
//
// Returns int which is the number of bytes the expression spans.
func (lm *LinkedMessageExpression) GetSourceLength() int { return lm.SourceLength }

// TransformIdentifiers recursively transforms identifiers in the path
// expression.
//
// Takes f (func(string) string) which transforms each identifier string.
//
// Returns Expression which is a new LinkedMessageExpression with transformed
// identifiers.
func (lm *LinkedMessageExpression) TransformIdentifiers(f func(string) string) Expression {
	return &LinkedMessageExpression{
		Path:             lm.Path.TransformIdentifiers(f),
		GoAnnotations:    lm.GoAnnotations,
		RelativeLocation: lm.RelativeLocation,
		SourceLength:     lm.SourceLength,
	}
}

// Clone creates a deep copy of the LinkedMessageExpression.
//
// Returns Expression which is a new LinkedMessageExpression with all
// fields copied,
// or nil if the receiver is nil.
func (lm *LinkedMessageExpression) Clone() Expression {
	if lm == nil {
		return nil
	}
	return &LinkedMessageExpression{
		Path:             lm.Path.Clone(),
		GoAnnotations:    lm.GoAnnotations.Clone(),
		RelativeLocation: lm.RelativeLocation,
		SourceLength:     lm.SourceLength,
	}
}

// GetGoAnnotation returns the code generation annotation for this node.
//
// Returns *GoGeneratorAnnotation which is the annotation for code generation.
func (lm *LinkedMessageExpression) GetGoAnnotation() *GoGeneratorAnnotation {
	return lm.GoAnnotations
}

// SetGoAnnotation sets the code generation annotation for this node.
//
// Takes ann (*GoGeneratorAnnotation) which specifies the annotation to apply.
func (lm *LinkedMessageExpression) SetGoAnnotation(ann *GoGeneratorAnnotation) {
	lm.GoAnnotations = ann
}

// GetRelativeLocation returns the source location of this expression node.
//
// Returns Location which is the position of this node relative to its parent.
func (lm *LinkedMessageExpression) GetRelativeLocation() Location { return lm.RelativeLocation }

// SetLocation sets the source location and length for this expression node.
//
// Takes location (Location) which specifies the source position.
// Takes length (int) which specifies the span of the source text.
func (lm *LinkedMessageExpression) SetLocation(location Location, length int) {
	lm.RelativeLocation, lm.SourceLength = location, length
}

var (
	_ Expression = (*Identifier)(nil)

	_ Expression = (*MemberExpression)(nil)

	_ Expression = (*IndexExpression)(nil)

	_ Expression = (*StringLiteral)(nil)

	_ Expression = (*IntegerLiteral)(nil)

	_ Expression = (*FloatLiteral)(nil)

	_ Expression = (*BooleanLiteral)(nil)

	_ Expression = (*NilLiteral)(nil)

	_ Expression = (*UnaryExpression)(nil)

	_ Expression = (*BinaryExpression)(nil)

	_ Expression = (*ForInExpression)(nil)

	_ Expression = (*CallExpression)(nil)

	_ Expression = (*TemplateLiteral)(nil)

	_ Expression = (*ObjectLiteral)(nil)

	_ Expression = (*TernaryExpression)(nil)

	_ Expression = (*ArrayLiteral)(nil)

	_ Expression = (*DecimalLiteral)(nil)

	_ Expression = (*BigIntLiteral)(nil)

	_ Expression = (*DateTimeLiteral)(nil)

	_ Expression = (*DateLiteral)(nil)

	_ Expression = (*TimeLiteral)(nil)

	_ Expression = (*DurationLiteral)(nil)

	_ Expression = (*RuneLiteral)(nil)

	_ Expression = (*LinkedMessageExpression)(nil)
)

// getPrecedence returns the precedence level for the given operator.
//
// Takes op (string) which is the operator to look up.
//
// Returns int which is the precedence level, or precLowest if the operator
// is not found.
func getPrecedence(op string) int {
	if p, ok := precedences[op]; ok {
		return p
	}
	return precLowest
}
