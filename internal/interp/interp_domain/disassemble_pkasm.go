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
	"strings"
)

const (
	// pkasmSeparatorWidth is the width of the separator line in
	// function headers.
	pkasmSeparatorWidth = 61

	// pkasmIndentUnit is the indentation string per nesting level.
	pkasmIndentUnit = "  "

	// pkasmConstantSeparator separates entries in constant pool
	// dumps.
	pkasmConstantSeparator = "  "
)

// pkasmWriter writes human-readable bytecode assembly to a string
// builder. It tracks indentation depth for nested function output.
type pkasmWriter struct {
	// builder accumulates the assembly output.
	builder *strings.Builder

	// indentLevel tracks the current nesting depth.
	indentLevel int
}

// DisassembleAssembly returns the complete human-readable bytecode
// assembly listing for the compiled file set. The output includes a
// file header, the root function body (if any), the variable init
// function (if any), and all child functions recursively.
//
// Returns the assembly listing as a string.
func (cfs *CompiledFileSet) DisassembleAssembly() string {
	w := &pkasmWriter{builder: &strings.Builder{}}

	w.writeLine("; pkasm - Piko Bytecode Assembly")
	w.writeLine("")

	root := cfs.root
	if root == nil {
		return w.builder.String()
	}

	if len(root.body) > 0 {
		w.writeFunction(root, "<root>")
	}

	if cfs.variableInitFunction != nil {
		w.writeFunction(cfs.variableInitFunction, "<varinit>")
	}

	for _, child := range root.functions {
		w.writeFunctionRecursive(child)
	}

	return w.builder.String()
}

// DisassembleFunctionAssembly returns the human-readable bytecode assembly listing
// for the function and all its nested children.
//
// Returns the assembly listing as a string.
func (cf *CompiledFunction) DisassembleFunctionAssembly() string {
	w := &pkasmWriter{builder: &strings.Builder{}}
	w.writeFunctionRecursive(cf)
	return w.builder.String()
}

// writeFunctionRecursive writes a function block and then recurses
// into its child functions with increased indentation.
//
// Takes cf (*CompiledFunction) which is the function to write.
func (w *pkasmWriter) writeFunctionRecursive(cf *CompiledFunction) {
	name := cf.name
	if name == "" {
		name = "<anonymous>"
	}
	w.writeFunction(cf, name)

	w.indentLevel++
	for _, child := range cf.functions {
		w.writeFunctionRecursive(child)
	}
	w.indentLevel--
}

// writeFunction writes a single function block: header, constant
// pools, and instruction listing.
//
// Takes cf (*CompiledFunction) which is the function to write.
// Takes name (string) which is the display name for the header.
func (w *pkasmWriter) writeFunction(cf *CompiledFunction, name string) {
	w.writeLine("")
	w.writeFunctionHeader(cf, name)
	w.writeConstantPools(cf)
	w.writeInstructions(cf)
}

// writeFunctionHeader writes the decorated function header showing
// name, register counts, parameter kinds, return kinds, and variadic
// flag.
//
// Takes cf (*CompiledFunction) which provides the metadata.
// Takes name (string) which is the display name for the header.
func (w *pkasmWriter) writeFunctionHeader(cf *CompiledFunction, name string) {
	separator := "; " + strings.Repeat("═", pkasmSeparatorWidth)
	w.writeLine(separator)
	w.writeLine(fmt.Sprintf("; function %s", name))

	if cf.sourceFile != "" {
		w.writeLine(fmt.Sprintf(";   source:    %s", cf.sourceFile))
	}

	regParts := formatRegisterCounts(cf)
	if len(regParts) > 0 {
		w.writeLine(fmt.Sprintf(";   registers: %s", strings.Join(regParts, " ")))
	}

	paramStr := formatKindList(cf.paramKinds)
	w.writeLine(fmt.Sprintf(";   params:    %s", paramStr))

	returnStr := formatKindList(cf.resultKinds)
	w.writeLine(fmt.Sprintf(";   returns:   %s", returnStr))

	if cf.isVariadic {
		w.writeLine(";   variadic:  true")
	}

	w.writeLine(separator)
}

// formatRegisterCounts returns a slice of "kind=N" strings for
// non-zero register banks.
//
// Takes cf (*CompiledFunction) which provides register counts.
//
// Returns []string containing one "kind=N" entry per non-zero
// register bank.
func formatRegisterCounts(cf *CompiledFunction) []string {
	var parts []string
	for i := range NumRegisterKinds {
		count := cf.numRegisters[i]
		if count > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", registerKind(i).String(), count))
		}
	}
	return parts
}

// formatKindList formats a slice of register kinds as a parenthesised
// comma-separated list, e.g. "(int, string)" or "(none)".
//
// Takes kinds ([]registerKind) which lists the kinds to format.
//
// Returns string containing the formatted parenthesised list.
func formatKindList(kinds []registerKind) string {
	if len(kinds) == 0 {
		return "(none)"
	}
	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.String()
	}
	return "(" + strings.Join(names, ", ") + ")"
}

// writeConstantPools dumps non-empty constant pools compactly.
//
// Takes cf (*CompiledFunction) which provides the constant pools.
func (w *pkasmWriter) writeConstantPools(cf *CompiledFunction) {
	hasAny := len(cf.intConstants) > 0 || len(cf.floatConstants) > 0 ||
		len(cf.stringConstants) > 0 || len(cf.boolConstants) > 0 ||
		len(cf.uintConstants) > 0 || len(cf.complexConstants) > 0

	if !hasAny {
		return
	}

	w.writeLine("")
	w.writeLine("; constants:")

	if len(cf.intConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   ints:    %s", formatIntConstants(cf.intConstants)))
	}
	if len(cf.floatConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   floats:  %s", formatFloatConstants(cf.floatConstants)))
	}
	if len(cf.stringConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   strings: %s", formatStringConstants(cf.stringConstants)))
	}
	if len(cf.boolConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   bools:   %s", formatBoolConstants(cf.boolConstants)))
	}
	if len(cf.uintConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   uints:   %s", formatUintConstants(cf.uintConstants)))
	}
	if len(cf.complexConstants) > 0 {
		w.writeLine(fmt.Sprintf(";   complex: %s", formatComplexConstants(cf.complexConstants)))
	}
}

// writeInstructions writes the instruction listing with source line
// annotations and enhanced call comments.
//
// Takes cf (*CompiledFunction) which provides the instruction body.
func (w *pkasmWriter) writeInstructions(cf *CompiledFunction) {
	if len(cf.body) == 0 {
		return
	}

	w.writeLine("")

	var lastFile string
	var lastLine int

	for pc := range cf.body {
		lastFile, lastLine = w.writeSourceAnnotation(cf, pc, lastFile, lastLine)
		w.writeInstruction(cf, pc)
	}
}

// writeSourceAnnotation emits a source line comment when the source
// position changes.
//
// Takes cf (*CompiledFunction) which provides the source map.
// Takes pc (int) which is the program counter to annotate.
// Takes lastFile (string) which is the previous source file.
// Takes lastLine (int) which is the previous source line.
//
// Returns the updated file and line trackers.
func (w *pkasmWriter) writeSourceAnnotation(cf *CompiledFunction, pc int, lastFile string, lastLine int) (string, int) {
	if cf.debugSourceMap == nil {
		return lastFile, lastLine
	}
	file, line, _ := cf.debugSourceMap.SourcePosition(pc)
	if line <= 0 || (file == lastFile && line == lastLine) {
		return lastFile, lastLine
	}
	if file != lastFile {
		w.writeLineRaw(w.indent() + fmt.Sprintf("%52s; %s:%d", "", file, line))
	} else {
		w.writeLineRaw(w.indent() + fmt.Sprintf("%52s; :%d", "", line))
	}
	return file, line
}

// writeInstruction writes a single instruction line with optional
// inline comment.
//
// Takes cf (*CompiledFunction) which provides the instruction body.
// Takes pc (int) which is the program counter of the instruction.
func (w *pkasmWriter) writeInstruction(cf *CompiledFunction, pc int) {
	instr := cf.body[pc]
	comment := cf.pkasmComment(instr)
	if comment != "" {
		w.writeLine(fmt.Sprintf("%04d  %-26s %3d %3d %3d    ; %s",
			pc, instr.op, instr.a, instr.b, instr.c, comment))
	} else {
		w.writeLine(fmt.Sprintf("%04d  %-26s %3d %3d %3d",
			pc, instr.op, instr.a, instr.b, instr.c))
	}
}

// pkasmComment returns an enhanced inline comment for an instruction,
// including call target resolution.
//
// Takes instr (instruction) which is the instruction to comment.
//
// Returns string containing the comment, or empty if none applies.
func (cf *CompiledFunction) pkasmComment(instr instruction) string {
	if comment := cf.pkasmCallComment(instr); comment != "" {
		return comment
	}
	return cf.disassembleComment(instr)
}

// pkasmCallComment resolves call instructions to their target names.
//
// Takes instr (instruction) which is the instruction to inspect.
//
// Returns string containing the resolved call comment, or empty if
// the instruction is not a call.
func (cf *CompiledFunction) pkasmCallComment(instr instruction) string {
	switch instr.op {
	case opCall:
		return cf.resolveCallTarget(instr, "call")
	case opTailCall:
		return cf.resolveCallTarget(instr, "tail call")
	case opCallIIFE:
		return cf.resolveCallTarget(instr, "iife")
	case opCallNative:
		siteIndex := int(instr.wideIndex())
		if siteIndex < len(cf.callSites) {
			site := cf.callSites[siteIndex]
			if site.isClosure {
				return fmt.Sprintf("call closure general[%d] (site %d)", site.closureRegister, siteIndex)
			}
			if site.isMethod {
				return fmt.Sprintf("call native method general[%d] (site %d)", site.nativeRegister, siteIndex)
			}
			return fmt.Sprintf("call native general[%d] (site %d)", site.nativeRegister, siteIndex)
		}
		return "call native"
	case opCallMethod:
		siteIndex := int(instr.wideIndex())
		return fmt.Sprintf("call method (site %d)", siteIndex)
	case opCallBuiltin:
		return "call builtin"
	case opMakeClosure:
		funcIndex := int(instr.wideIndex())
		if funcIndex < len(cf.functions) {
			name := cf.functions[funcIndex].name
			if name == "" {
				name = "<anonymous>"
			}
			return fmt.Sprintf("closure %s (func %d)", name, funcIndex)
		}
		return fmt.Sprintf("closure (func %d)", funcIndex)
	}
	return ""
}

// resolveCallTarget resolves a CALL/TAIL_CALL/CALL_IIFE instruction
// to the target function name via callSites.
//
// Takes instr (instruction) which is the call instruction.
// Takes label (string) which is the call type label for output.
//
// Returns string containing the resolved target description.
func (cf *CompiledFunction) resolveCallTarget(instr instruction, label string) string {
	siteIndex := int(instr.wideIndex())
	if siteIndex >= len(cf.callSites) {
		return fmt.Sprintf("%s (site %d)", label, siteIndex)
	}
	site := cf.callSites[siteIndex]
	if site.isClosure {
		return fmt.Sprintf("%s closure general[%d] (site %d)", label, site.closureRegister, siteIndex)
	}
	funcIndex := int(site.funcIndex)
	if funcIndex < len(cf.functions) {
		name := cf.functions[funcIndex].name
		if name == "" {
			name = "<anonymous>"
		}
		return fmt.Sprintf("%s %s (site %d)", label, name, siteIndex)
	}
	return fmt.Sprintf("%s (site %d)", label, siteIndex)
}

// writeLine writes an indented line to the builder.
//
// Takes line (string) which is the content to write.
func (w *pkasmWriter) writeLine(line string) {
	w.builder.WriteString(w.indent())
	w.builder.WriteString(line)
	w.builder.WriteByte('\n')
}

// writeLineRaw writes a line to the builder without adding
// indentation (caller is responsible for any prefix).
//
// Takes line (string) which is the content to write.
func (w *pkasmWriter) writeLineRaw(line string) {
	w.builder.WriteString(line)
	w.builder.WriteByte('\n')
}

// indent returns the indentation prefix for the current level
// (2 spaces per level).
//
// Returns string containing the indentation whitespace.
func (w *pkasmWriter) indent() string {
	if w.indentLevel <= 0 {
		return ""
	}
	return strings.Repeat(pkasmIndentUnit, w.indentLevel)
}

// formatIntConstants formats an int constant pool compactly.
//
// Takes constants ([]int64) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatIntConstants(constants []int64) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		parts[i] = fmt.Sprintf("[%d]=%d", i, v)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}

// formatFloatConstants formats a float constant pool compactly.
//
// Takes constants ([]float64) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatFloatConstants(constants []float64) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		parts[i] = fmt.Sprintf("[%d]=%g", i, v)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}

// formatStringConstants formats a string constant pool compactly.
//
// Takes constants ([]string) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatStringConstants(constants []string) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		s := v
		if len(s) > maxDisassembleStringLen {
			s = s[:truncatedDisassembleStringLen] + "..."
		}
		parts[i] = fmt.Sprintf("[%d]=%q", i, s)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}

// formatBoolConstants formats a bool constant pool compactly.
//
// Takes constants ([]bool) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatBoolConstants(constants []bool) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		parts[i] = fmt.Sprintf("[%d]=%v", i, v)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}

// formatUintConstants formats a uint constant pool compactly.
//
// Takes constants ([]uint64) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatUintConstants(constants []uint64) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		parts[i] = fmt.Sprintf("[%d]=%d", i, v)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}

// formatComplexConstants formats a complex constant pool compactly.
//
// Takes constants ([]complex128) which is the pool to format.
//
// Returns string containing the formatted constant entries.
func formatComplexConstants(constants []complex128) string {
	parts := make([]string, len(constants))
	for i, v := range constants {
		parts[i] = fmt.Sprintf("[%d]=%v", i, v)
	}
	return strings.Join(parts, pkasmConstantSeparator)
}
