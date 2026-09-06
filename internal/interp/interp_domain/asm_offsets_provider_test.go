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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"piko.sh/asmgen"
	"piko.sh/piko/internal/interp/interp_domain/asm"
	interp_amd64 "piko.sh/piko/internal/interp/interp_domain/asm/asmgen_arch_amd64"
)

func TestProvidedOffsetsMatchLiveStructs(t *testing.T) {
	t.Parallel()

	headers := asm.HeaderFiles(
		ProvideDispatchContextOffsets(),
		ProvideCallFrameOffsets(),
		ProvideASMCallInfoOffsets(),
		ProvideVarLocationOffsets(),
	)

	var generated string
	for _, header := range headers {
		if header.Name != "asm_dispatch_offsets.h" {
			continue
		}
		generated = header.Emit([]asmgen.ArchitecturePort{interp_amd64.New()})
		break
	}
	if generated == "" {
		t.Fatal("HeaderFiles did not return asm_dispatch_offsets.h")
	}

	livePath := filepath.Join("asm_dispatch_offsets.h")
	liveBytes, err := os.ReadFile(livePath)
	if err != nil {
		t.Fatalf("read live header: %v", err)
	}
	if string(liveBytes) != generated {
		t.Errorf("asmgen-generated header differs from live %s.\n"+
			"Live and asmgen-derived offsets are out of sync. Either:\n"+
			"  1. A runtime struct grew but ProvideDispatchContextOffsets / "+
			"ProvideCallFrameOffsets / ProvideASMCallInfoOffsets / "+
			"ProvideVarLocationOffsets was not updated to expose the new field, OR\n"+
			"  2. asmgen has not been re-run since the struct change "+
			"(run hack/generate/asmgen.sh to refresh the live header).\n"+
			"\nFirst diverging line:\n%s", livePath, firstDiff(string(liveBytes), generated))
	}
}

func firstDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")
	limit := min(len(actualLines), len(expectedLines))
	for i := range limit {
		if expectedLines[i] != actualLines[i] {
			return formatLineDiff(i, expectedLines[i], actualLines[i])
		}
	}
	if len(expectedLines) != len(actualLines) {
		return "  expected " + itoa(len(expectedLines)) + " lines, got " + itoa(len(actualLines))
	}
	return ""
}

func formatLineDiff(index int, expectedLine, actualLine string) string {
	return "  line " + itoa(index+1) + ":\n" +
		"    expected: " + expectedLine + "\n" +
		"    actual:   " + actualLine
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var buffer [20]byte
	pos := len(buffer)
	for n > 0 {
		pos--
		buffer[pos] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		pos--
		buffer[pos] = '-'
	}
	return string(buffer[pos:])
}

func TestProvidedOffsetsAreSelfConsistent(t *testing.T) {
	t.Parallel()

	contextOffsets := ProvideDispatchContextOffsets()
	var ctx DispatchContext
	if got, want := contextOffsets.SlicesIntBase, unsafe.Offsetof(ctx.slicesIntBase); got != want {
		t.Errorf("SlicesIntBase offset mismatch: provider=%d struct=%d", got, want)
	}
	if got, want := contextOffsets.AsmCallInfoBase, unsafe.Offsetof(ctx.asmCallInfoBase); got != want {
		t.Errorf("AsmCallInfoBase offset mismatch: provider=%d struct=%d", got, want)
	}

	frameOffsets := ProvideCallFrameOffsets()
	var frame callFrame
	if got, want := frameOffsets.Size, unsafe.Sizeof(frame); got != want {
		t.Errorf("CallFrame Size mismatch: provider=%d struct=%d", got, want)
	}
}
