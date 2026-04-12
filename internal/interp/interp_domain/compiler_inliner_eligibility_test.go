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
	"context"
	"testing"
)

func TestScanCalleeForRefusal_EmptyBodyRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: nil,
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalNoBody {
		t.Fatalf("empty body must refuse with noBody, got %v", got)
	}
}

func TestScanCalleeForRefusal_UpvaluesRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body:               []instruction{makeInstruction(opNop, 0, 0, 0)},
		upvalueDescriptors: []UpvalueDescriptor{{}},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalUpvalues {
		t.Fatalf("upvalues must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_DeferRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			makeInstruction(opNop, 0, 0, 0),
			makeInstruction(opDefer, 0, 0, 0),
		},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalDefer {
		t.Fatalf("opDefer must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_GoRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{makeInstruction(opGo, 0, 0, 0)},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalGo {
		t.Fatalf("opGo must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_MethodCallRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{makeInstruction(opCallMethod, 0, 0, 0)},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalMethodCall {
		t.Fatalf("opCallMethod must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_NativeCallRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{makeInstruction(opCallNative, 0, 0, 0)},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalNativeCall {
		t.Fatalf("opCallNative must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_TailCallRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{makeInstruction(opTailCall, 0, 0, 0)},
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalTailCall {
		t.Fatalf("opTailCall must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_ClosureOpsRefused(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		op   opcode
	}{
		{name: "makeClosure", op: opMakeClosure},
		{name: "getUpvalue", op: opGetUpvalue},
		{name: "setUpvalue", op: opSetUpvalue},
		{name: "syncClosureUpvalues", op: opSyncClosureUpvalues},
		{name: "callIIFE", op: opCallIIFE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{
				body: []instruction{makeInstruction(tc.op, 0, 0, 0)},
			}
			if got := scanCalleeForRefusal(cf); got != inlineRefusalClosureOps {
				t.Fatalf("%s must refuse with closureOps, got %v", tc.name, got)
			}
		})
	}
}

func TestScanCalleeForRefusal_ChannelOpsRefused(t *testing.T) {
	t.Parallel()
	cases := []opcode{opSelect, opChannelSend}
	for _, op := range cases {
		t.Run(op.String(), func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{
				body: []instruction{makeInstruction(op, 0, 0, 0)},
			}
			if got := scanCalleeForRefusal(cf); got != inlineRefusalChannelOps {
				t.Fatalf("%v must refuse with channelOps, got %v", op, got)
			}
		})
	}
}

func TestScanCalleeForRefusal_VariadicAcceptedAtCalleeLevel(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body:       []instruction{makeInstruction(opNop, 0, 0, 0)},
		isVariadic: true,
	}
	if got := scanCalleeForRefusal(cf); got != inlineEligible {
		t.Fatalf("variadic alone must be eligible at callee level, got %v", got)
	}
}

func TestScanCalleeForRefusal_GenericPlaceholderRefused(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body:                 []instruction{makeInstruction(opNop, 0, 0, 0)},
		isGenericFunc:        true,
		specialisationOrigin: nil,
	}
	if got := scanCalleeForRefusal(cf); got != inlineRefusalGenericPlaceholder {
		t.Fatalf("generic placeholder must refuse, got %v", got)
	}
}

func TestScanCalleeForRefusal_GenericSpecialisationEligible(t *testing.T) {
	t.Parallel()
	origin := &CompiledFunction{isGenericFunc: true}
	cf := &CompiledFunction{
		body:                 []instruction{makeInstruction(opNop, 0, 0, 0)},
		isGenericFunc:        true,
		specialisationOrigin: origin,
	}
	if got := scanCalleeForRefusal(cf); got != inlineEligible {
		t.Fatalf("generic specialisation must be eligible, got %v", got)
	}
}

func TestScanCalleeForRefusal_SimpleBodyEligible(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 0, 1, 2),
			makeInstruction(opAddInt, 1, 2, 3),
		},
	}
	if got := scanCalleeForRefusal(cf); got != inlineEligible {
		t.Fatalf("simple body must be eligible, got %v", got)
	}
}

func TestCalleeInlineRefusal_Caches(t *testing.T) {
	t.Parallel()
	cf := &CompiledFunction{
		body: []instruction{makeInstruction(opDefer, 0, 0, 0)},
	}
	if cf.cachedInlineRefusal != inlineRefusalUnknown {
		t.Fatalf("pre-probe cache must be Unknown, got %v", cf.cachedInlineRefusal)
	}
	first := calleeInlineRefusal(cf)
	if first != inlineRefusalDefer {
		t.Fatalf("first probe returned %v want defer", first)
	}
	if cf.cachedInlineRefusal != inlineRefusalDefer {
		t.Fatalf("cache must be populated after first probe, got %v", cf.cachedInlineRefusal)
	}

	cf.body = []instruction{makeInstruction(opAddInt, 0, 0, 0)}
	second := calleeInlineRefusal(cf)
	if second != inlineRefusalDefer {
		t.Fatalf("cached probe must remain Defer despite body change, got %v", second)
	}
}

func TestCanInline_RefusesNativeSite(t *testing.T) {
	t.Parallel()
	site := &callSite{isNative: true}
	if got := canInline(nil, site, 0, false); got != inlineRefusalSiteIndirect {
		t.Fatalf("native site must refuse, got %v", got)
	}
}

func TestCanInline_RefusesClosureSite(t *testing.T) {
	t.Parallel()
	site := &callSite{isClosure: true}
	if got := canInline(nil, site, 0, false); got != inlineRefusalSiteIndirect {
		t.Fatalf("closure site must refuse, got %v", got)
	}
}

func TestCanInline_RefusesMethodSite(t *testing.T) {
	t.Parallel()
	site := &callSite{isMethod: true}
	if got := canInline(nil, site, 0, false); got != inlineRefusalSiteIndirect {
		t.Fatalf("method site must refuse, got %v", got)
	}
}

func TestCanInline_RefusesNilSite(t *testing.T) {
	t.Parallel()
	if got := canInline(nil, nil, 0, false); got != inlineRefusalUnknown {
		t.Fatalf("nil site must refuse with Unknown, got %v", got)
	}
}

func TestCanInline_RefusesNoCachedCallee(t *testing.T) {
	t.Parallel()
	site := &callSite{cachedCallee: nil}
	if got := canInline(nil, site, 0, false); got != inlineRefusalNoBody {
		t.Fatalf("missing cachedCallee must refuse with NoBody, got %v", got)
	}
}

func TestCanInline_RespectsBudget(t *testing.T) {
	t.Parallel()

	body := make([]instruction, defaultInlineBudget+1)
	for i := range body {
		body[i] = makeInstruction(opAddInt, 0, 0, 0)
	}
	callee := &CompiledFunction{body: body}
	caller := &CompiledFunction{body: []instruction{makeInstruction(opNop, 0, 0, 0)}}
	site := &callSite{cachedCallee: callee}
	if got := canInline(caller, site, 0, false); got != inlineRefusalOversize {
		t.Fatalf("oversize callee must refuse, got %v", got)
	}

	if got := canInline(caller, site, 0, true); got == inlineRefusalOversize {
		t.Fatalf("loop-site budget should allow %d-instr callee", len(body))
	}
}

func TestCalleeHairyness_OpcodeWeights(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body []instruction
		want int
	}{
		{
			name: "ten_simple_ops",
			body: func() []instruction {
				b := make([]instruction, 10)
				for i := range b {
					b[i] = makeInstruction(opAddInt, 0, 0, 0)
				}
				return b
			}(),
			want: 10,
		},
		{
			name: "one_make_slice",
			body: []instruction{makeInstruction(opMakeSlice, 0, 0, 0)},
			want: 2,
		},
		{
			name: "two_map_index",
			body: []instruction{
				makeInstruction(opMapIndex, 0, 0, 0),
				makeInstruction(opMapIndex, 0, 0, 0),
			},
			want: 2,
		},
		{
			name: "five_nops",
			body: []instruction{
				makeInstruction(opNop, 0, 0, 0),
				makeInstruction(opNop, 0, 0, 0),
				makeInstruction(opNop, 0, 0, 0),
				makeInstruction(opNop, 0, 0, 0),
				makeInstruction(opNop, 0, 0, 0),
			},
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cf := &CompiledFunction{body: tc.body}
			if got := calleeHairyness(cf); got != tc.want {
				t.Fatalf("hairyness=%d want %d", got, tc.want)
			}
		})
	}
}

func TestCanInline_HappyPath(t *testing.T) {
	t.Parallel()
	callee := &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 0, 1, 2),
		},
		parameterKinds: []registerKind{registerInt, registerInt},
		numRegisters:   [NumRegisterKinds]uint32{registerInt: 3},
	}
	caller := &CompiledFunction{
		body:         []instruction{makeInstruction(opNop, 0, 0, 0)},
		numRegisters: [NumRegisterKinds]uint32{registerInt: 5},
	}
	site := &callSite{cachedCallee: callee}
	if got := canInline(caller, site, 0, false); got != inlineEligible {
		t.Fatalf("happy path must be eligible, got %v", got)
	}
}

func TestComputeInLoopMask_Simple(t *testing.T) {
	t.Parallel()

	offsetBack := -3
	jumpInstr := makeInstruction(
		opDrillTier1,
		byte(subOpJump),
		byte(uint16(int16(offsetBack))),
		byte(uint16(int16(offsetBack))>>8),
	)
	caller := &CompiledFunction{
		body: []instruction{
			makeInstruction(opAddInt, 0, 0, 0),
			makeInstruction(opAddInt, 0, 0, 0),
			jumpInstr,
			makeInstruction(opNop, 0, 0, 0),
		},
	}
	mask := computeInLoopMask(caller)
	if len(mask) != 4 {
		t.Fatalf("mask length %d, want 4", len(mask))
	}
	for pc := 0; pc <= 2; pc++ {
		if !mask[pc] {
			t.Errorf("pc %d should be in-loop", pc)
		}
	}
	if mask[3] {
		t.Errorf("pc 3 should not be in-loop")
	}
}

func TestRunBytecodeInliner_SingleFunctionNoOp(t *testing.T) {
	t.Parallel()
	root := &CompiledFunction{body: []instruction{makeInstruction(opNop, 0, 0, 0)}}
	origLen := len(root.body)
	if err := runBytecodeInliner(context.Background(), root); err != nil {
		t.Fatalf("runBytecodeInliner unexpected error: %v", err)
	}
	if len(root.body) != origLen {
		t.Fatalf("body length changed from %d to %d", origLen, len(root.body))
	}
}

func TestRunBytecodeInliner_NilRoot(t *testing.T) {
	t.Parallel()
	if err := runBytecodeInliner(context.Background(), nil); err != nil {
		t.Fatalf("nil root must return nil, got %v", err)
	}
}
