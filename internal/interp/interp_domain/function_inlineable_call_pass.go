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

// rewriteInlineableMethodCalls scans cf.body for opCallMethod instructions and promotes
// those whose call-site signature matches a candidate fused-dispatch shape to
// opCallMethodInlineable. The promoted opcode is bytecode-equivalent to opCallMethod but
// routes through handleCallMethodInlineable's per-callSite inline-descriptor cache, which
// classifies known fused shapes for inline-capable dispatch.
//
// The heuristic is intentionally conservative: only call sites whose signature matches a
// shape we have a known fused-dispatch path for are promoted. Sites that don't match keep
// the standard opCallMethod so they pay no extra cache overhead. As new fused shapes are
// added (int / float / multiple returns / getter chains), isInlineableShape is extended
// accordingly.
//
// Called from runPostPurityPeepholePass alongside the other per-function post-purity
// passes.
func (cf *CompiledFunction) rewriteInlineableMethodCalls() {
	for index := range cf.body {
		instr := cf.body[index]
		if instr.op != opCallMethod {
			continue
		}
		siteIndex := instr.wideIndex()
		if int(siteIndex) >= len(cf.callSites) {
			continue
		}
		site := &cf.callSites[siteIndex]
		if !isInlineableShape(site) {
			continue
		}
		cf.body[index] = makeInstruction(opCallMethodInlineable, instr.a, instr.b, instr.c)
	}
}

// isInlineableShape reports whether a call-site matches a fused-dispatch shape.
//
// The runtime cache machinery knows how to potentially classify and inline matching
// sites. The current allowlist accepts exactly two arguments with the first on the
// general bank (interface receiver) and exactly one return value on the uint bank,
// matching polyast's `Eval(env []uintN) uintN` Node shape (the concrete target for
// inlineShapeBinopUint). Sites that do not match keep opCallMethod and avoid the cache
// slot memory cost.
//
// Takes site (*callSite) which is the call-site descriptor to check.
//
// Returns true when the site's signature matches a candidate shape.
func isInlineableShape(site *callSite) bool {
	if len(site.arguments) != 2 || len(site.returns) != 1 {
		return false
	}
	if site.arguments[0].kind != registerGeneral {
		return false
	}
	if site.returns[0].kind != registerUint {
		return false
	}
	return true
}
