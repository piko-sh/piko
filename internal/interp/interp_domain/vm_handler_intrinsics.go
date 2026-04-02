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

package interp_domain

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"piko.sh/piko/wdk/safeconv"
)

// maxUTF8RuneBytes is the maximum number of bytes a single UTF-8 rune can occupy.
const maxUTF8RuneBytes = 4

var stringSliceType = reflect.TypeFor[[]string]()

// handleStrContainsRune handles the opStrContainsRune instruction by
// checking if a string contains the specified rune.
//
// Takes registers (*Registers) which holds the string and rune operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrContainsRune(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = strings.ContainsRune(registers.strings[instruction.b], safeconv.Int64ToInt32(registers.ints[instruction.c]))
	return opContinue
}

// handleStrContains handles the opStrContains instruction by checking
// if a string contains the specified substring.
//
// Takes registers (*Registers) which holds the string operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrContains(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = strings.Contains(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrHasPrefix handles the opStrHasPrefix instruction by checking
// if a string starts with the specified prefix.
//
// Takes registers (*Registers) which holds the string and prefix operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrHasPrefix(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = strings.HasPrefix(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrHasSuffix handles the opStrHasSuffix instruction by checking
// if a string ends with the specified suffix.
//
// Takes registers (*Registers) which holds the string and suffix operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrHasSuffix(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = strings.HasSuffix(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrEqualFold handles the opStrEqualFold instruction by
// performing a case-insensitive string comparison.
//
// Takes registers (*Registers) which holds the two string operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrEqualFold(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.bools[instruction.a] = strings.EqualFold(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrIndex handles the opStrIndex instruction by finding the
// index of the first occurrence of a substring within a string.
//
// Takes registers (*Registers) which holds the string and substring operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrIndex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(strings.Index(registers.strings[instruction.b], registers.strings[instruction.c]))
	return opContinue
}

// handleStrCount handles the opStrCount instruction by counting the
// non-overlapping occurrences of a substring within a string.
//
// Takes registers (*Registers) which holds the string and substring operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrCount(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(strings.Count(registers.strings[instruction.b], registers.strings[instruction.c]))
	return opContinue
}

// handleStrToUpper handles the opStrToUpper instruction by converting
// all characters in a string to uppercase.
//
// Takes registers (*Registers) which holds the source string and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrToUpper(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.ToUpper(registers.strings[instruction.b])
	return opContinue
}

// handleStrToLower handles the opStrToLower instruction by converting
// all characters in a string to lowercase.
//
// Takes registers (*Registers) which holds the source string and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrToLower(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.ToLower(registers.strings[instruction.b])
	return opContinue
}

// handleStrTrimSpace handles the opStrTrimSpace instruction by
// removing leading and trailing whitespace from a string.
//
// Takes registers (*Registers) which holds the source string and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrTrimSpace(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.TrimSpace(registers.strings[instruction.b])
	return opContinue
}

// handleStrTrimPrefix handles the opStrTrimPrefix instruction by
// removing the specified prefix from a string if present.
//
// Takes registers (*Registers) which holds the string and prefix operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrTrimPrefix(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.TrimPrefix(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrTrimSuffix handles the opStrTrimSuffix instruction by
// removing the specified suffix from a string if present.
//
// Takes registers (*Registers) which holds the string and suffix operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrTrimSuffix(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.TrimSuffix(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrTrim handles the opStrTrim instruction by removing leading
// and trailing characters in a cutset from a string.
//
// Takes registers (*Registers) which holds the string and cutset operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrTrim(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.Trim(registers.strings[instruction.b], registers.strings[instruction.c])
	return opContinue
}

// handleStrIndexRune handles the opStrIndexRune instruction by finding
// the index of the first occurrence of a rune within a string.
//
// Takes registers (*Registers) which holds the string and rune operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrIndexRune(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(strings.IndexRune(registers.strings[instruction.b], safeconv.Int64ToInt32(registers.ints[instruction.c])))
	return opContinue
}

// handleMathAbs handles the opMathAbs instruction by computing the
// absolute value of a float register.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathAbs(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Abs(registers.floats[instruction.b])
	return opContinue
}

// handleMathSqrt handles the opMathSqrt instruction by computing the
// square root of a float register.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathSqrt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Sqrt(registers.floats[instruction.b])
	return opContinue
}

// handleMathFloor handles the opMathFloor instruction by computing
// the floor of a float register value.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathFloor(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Floor(registers.floats[instruction.b])
	return opContinue
}

// handleMathCeil handles the opMathCeil instruction by computing the
// ceiling of a float register value.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathCeil(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Ceil(registers.floats[instruction.b])
	return opContinue
}

// handleMathRound handles the opMathRound instruction by rounding a
// float register value to the nearest integer.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathRound(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Round(registers.floats[instruction.b])
	return opContinue
}

// handleStrconvItoa handles the opStrconvItoa instruction by converting
// an integer to its decimal string representation.
//
// Takes registers (*Registers) which holds the integer source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrconvItoa(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strconv.Itoa(int(registers.ints[instruction.b]))
	return opContinue
}

// handleMathPow handles the opMathPow instruction by computing base
// raised to the exponent power from two float registers.
//
// Takes registers (*Registers) which holds the base, exponent and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathPow(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Pow(registers.floats[instruction.b], registers.floats[instruction.c])
	return opContinue
}

// handleMathExp handles the opMathExp instruction by computing the
// natural exponential of a float register value.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathExp(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Exp(registers.floats[instruction.b])
	return opContinue
}

// handleMathSin handles the opMathSin instruction by computing the
// sine of a float register value in radians.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathSin(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Sin(registers.floats[instruction.b])
	return opContinue
}

// handleMathCos handles the opMathCos instruction by computing the
// cosine of a float register value in radians.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathCos(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Cos(registers.floats[instruction.b])
	return opContinue
}

// handleMathTan handles the opMathTan instruction by computing the
// tangent of a float register value in radians.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathTan(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Tan(registers.floats[instruction.b])
	return opContinue
}

// handleMathMod handles the opMathMod instruction by computing the
// floating-point remainder of two float register values.
//
// Takes registers (*Registers) which holds the two float operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathMod(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Mod(registers.floats[instruction.b], registers.floats[instruction.c])
	return opContinue
}

// handleMathTrunc handles the opMathTrunc instruction by truncating a
// float register value toward zero.
//
// Takes registers (*Registers) which holds the source float and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleMathTrunc(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.floats[instruction.a] = math.Trunc(registers.floats[instruction.b])
	return opContinue
}

// handleStrconvFormatBool handles the opStrconvFormatBool instruction
// by converting a boolean to its string representation.
//
// Takes registers (*Registers) which holds the boolean source and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrconvFormatBool(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strconv.FormatBool(registers.bools[instruction.b])
	return opContinue
}

// handleStrconvFormatInt handles the opStrconvFormatInt instruction by
// formatting an integer in the specified base as a string.
//
// Takes registers (*Registers) which holds the integer, base and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrconvFormatInt(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strconv.FormatInt(registers.ints[instruction.b], int(registers.ints[instruction.c]))
	return opContinue
}

// handleStrRepeat handles the opStrRepeat instruction by repeating a
// string the specified number of times.
//
// Takes registers (*Registers) which holds the string, count and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrRepeat(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.strings[instruction.a] = strings.Repeat(registers.strings[instruction.b], int(registers.ints[instruction.c]))
	return opContinue
}

// handleStrLastIndex handles the opStrLastIndex instruction by finding
// the index of the last occurrence of a substring within a string.
//
// Takes registers (*Registers) which holds the string and substring operands.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrLastIndex(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	registers.ints[instruction.a] = int64(strings.LastIndex(registers.strings[instruction.b], registers.strings[instruction.c]))
	return opContinue
}

// handleConcatRuneString handles the opConcatRuneString instruction by
// concatenating a string with a rune using arena-based allocation.
//
// Takes vm (*VM) which provides the arena for string allocation.
// Takes registers (*Registers) which holds the string, rune and destination.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleConcatRuneString(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	if vm.limits.maxStringSize > 0 && len(registers.strings[instruction.b])+maxUTF8RuneBytes > vm.limits.maxStringSize {
		vm.evalError = fmt.Errorf("%w: concat result exceeds limit %d",
			errStringLimit, vm.limits.maxStringSize)
		return opPanicError
	}
	registers.strings[instruction.a] = arenaConcatRuneString(
		vm.arena, registers.strings[instruction.b], safeconv.Int64ToInt32(registers.ints[instruction.c]))
	return opContinue
}

// handleStrJoin handles the opStrJoin instruction by joining a slice
// of strings with the specified separator.
//
// Takes vm (*VM) which provides the arena for string materialisation.
// Takes registers (*Registers) which holds the slice and separator.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrJoin(vm *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	slice := registers.general[instruction.b]
	sep := materialiseString(vm.arena, registers.strings[instruction.c])
	n := slice.Len()
	parts := make([]string, n)
	for i := range parts {
		parts[i] = slice.Index(i).String()
	}
	registers.strings[instruction.a] = strings.Join(parts, sep)
	return opContinue
}

// handleStrSplit handles the opStrSplit instruction by splitting a
// string around each occurrence of the specified separator.
//
// Takes registers (*Registers) which holds the string and separator.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrSplit(_ *VM, _ *callFrame, registers *Registers, instruction instruction) opResult {
	parts := strings.Split(registers.strings[instruction.b], registers.strings[instruction.c])
	result := reflect.MakeSlice(stringSliceType, len(parts), len(parts))
	for i, p := range parts {
		result.Index(i).SetString(p)
	}
	registers.general[instruction.a] = result
	return opContinue
}

// handleStrReplaceAll handles the opStrReplaceAll instruction by
// replacing all occurrences of a substring with a replacement string.
//
// Takes frame (*callFrame) which provides the replacement extension word.
// Takes registers (*Registers) which holds the string and replacement values.
// Takes instruction (instruction) which encodes the register indices.
//
// Returns opResult indicating the next execution step.
func handleStrReplaceAll(_ *VM, frame *callFrame, registers *Registers, instruction instruction) opResult {
	extensionWord := frame.function.body[frame.programCounter]
	frame.programCounter++
	registers.strings[instruction.a] = strings.ReplaceAll(
		registers.strings[instruction.b], registers.strings[instruction.c],
		registers.strings[extensionWord.a])
	return opContinue
}
