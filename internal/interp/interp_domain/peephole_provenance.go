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
)

// peepholeRewriteKind classifies a single peephole rewrite for the disassembler's per-PC
// annotation lookup. The kind determines the human-readable phrase prefixed to the inline
// comment ("CSE'd from", "LICM hoist of", etc.).
type peepholeRewriteKind uint8

const (
	// peepholeRewriteCseTier0 marks a tier-0 read that was rewritten to a same-bank MOVE
	// because a prior matching read still held the cached value.
	peepholeRewriteCseTier0 peepholeRewriteKind = iota + 1

	// peepholeRewriteCseTier1Umbrella marks the umbrella word of a tier-1 read that was
	// rewritten to a same-bank tier-1 MOVE.
	peepholeRewriteCseTier1Umbrella

	// peepholeRewriteCseTier1Ext marks the EXT word of a CSE'd tier-1 read that was nopped
	// because the umbrella was rewritten to a move and no longer consumes its layout
	// extension.
	peepholeRewriteCseTier1Ext

	// peepholeRewriteCsePostSet marks a GET that was rewritten to a MOVE because a preceding
	// SET to the same field carried the stored value in a register still live at the read
	// site.
	peepholeRewriteCsePostSet

	// peepholeRewriteLicmHoist marks an instruction inserted at a loop pre-header by the
	// loop-invariant code motion pass.
	peepholeRewriteLicmHoist

	// peepholeRewriteLicmOpNop marks the (now-shifted) original position of a read that LICM
	// lifted out of the loop body.
	peepholeRewriteLicmOpNop

	// peepholeRewriteGvn marks an instruction that GVN rewrote to a same-bank MOVE from an
	// earlier dominator-validated equivalent computation.
	peepholeRewriteGvn

	// peepholeRewriteBce marks a slice / string index access whose runtime bounds check was
	// elided in favour of the unchecked opcode variant. origin records the PC of the proof
	// source (the range loop header, or the guarding LtInt comparison).
	peepholeRewriteBce

	// peepholeRewriteUnroll marks the leading instruction of an inlined copy of a
	// self-recursive callee body. origin records the PC of the original opCall site that the
	// inliner replaced.
	peepholeRewriteUnroll
)

// peepholeAnnotation records a single rewrite event for a compiled instruction.
//
// The origin field carries the PC the rewritten instruction derives from (the source for
// CSE MOVE rewrites; the original in-loop read PC for LICM hoists). The disassembler
// interprets origin only when the kind implies an origin (CSE moves and LICM hoists).
type peepholeAnnotation struct {
	// kind classifies the rewrite so callers can pick the appropriate human-readable phrase.
	kind peepholeRewriteKind

	// origin is the PC the rewritten instruction derives from, or zero when the kind does
	// not require an origin.
	origin int
}

// recordPeepholeRewrite stores an annotation describing the rewrite applied at pc. Lazily
// allocates the side table on first use; cheap because the table is touched only at
// compile-time peephole rewrites.
//
// Takes pc (int) which is the PC of the rewritten instruction, kind (peepholeRewriteKind)
// which classifies the rewrite, and origin (int) which is the PC the rewrite derives from
// (or any non-negative value when origin is not meaningful for the kind).
func (cf *CompiledFunction) recordPeepholeRewrite(pc int, kind peepholeRewriteKind, origin int) {
	if cf.peepholeProvenance == nil {
		cf.peepholeProvenance = make(map[int]peepholeAnnotation)
	}
	cf.peepholeProvenance[pc] = peepholeAnnotation{kind: kind, origin: origin}
}

// shiftPeepholeProvenanceAfterInsert reindexes the provenance map after an instruction
// insertion at insertPC.
//
// Every entry whose PC is at or after insertPC moves up by one slot, and origins are
// updated symmetrically so the post-insert PC continues to point at the same logical
// instruction.
//
// Takes insertPC (int) which is the index that received a new instruction; entries at
// insertPC and later shift by one.
func (cf *CompiledFunction) shiftPeepholeProvenanceAfterInsert(insertPC int) {
	if cf.peepholeProvenance == nil {
		return
	}
	rebuilt := make(map[int]peepholeAnnotation, len(cf.peepholeProvenance))
	for pc, ann := range cf.peepholeProvenance {
		newPC := pc
		if pc >= insertPC {
			newPC = pc + 1
		}
		newOrigin := ann.origin
		if ann.origin >= insertPC {
			newOrigin = ann.origin + 1
		}
		rebuilt[newPC] = peepholeAnnotation{kind: ann.kind, origin: newOrigin}
	}
	cf.peepholeProvenance = rebuilt
}

// peepholeAnnotationAt returns the annotation recorded for pc.
//
// Takes pc (int) which is the program counter to look up.
//
// Returns the recorded annotation, or the zero value when no annotation exists for that
// PC.
func (cf *CompiledFunction) peepholeAnnotationAt(pc int) peepholeAnnotation {
	if cf.peepholeProvenance == nil {
		return peepholeAnnotation{}
	}
	return cf.peepholeProvenance[pc]
}

// formatPeepholeAnnotation renders an annotation as the trailing comment text appended to
// a disassembled instruction line.
//
// Takes ann (peepholeAnnotation) which is the annotation recorded by the peephole pass.
//
// Returns the trailing comment text, or an empty string for the zero annotation so
// callers can fall through to other comment producers.
func formatPeepholeAnnotation(ann peepholeAnnotation) string {
	switch ann.kind {
	case peepholeRewriteCseTier0:
		return fmt.Sprintf("CSE'd from PC %d", ann.origin)
	case peepholeRewriteCseTier1Umbrella:
		return fmt.Sprintf("CSE'd from PC %d (tier-1 read)", ann.origin)
	case peepholeRewriteCseTier1Ext:
		return "CSE'd EXT word"
	case peepholeRewriteCsePostSet:
		return fmt.Sprintf("CSE'd from SET at PC %d", ann.origin)
	case peepholeRewriteLicmHoist:
		return fmt.Sprintf("LICM hoist of PC %d", ann.origin)
	case peepholeRewriteLicmOpNop:
		return "LICM-hoisted; original site"
	case peepholeRewriteGvn:
		return fmt.Sprintf("GVN'd from PC %d", ann.origin)
	case peepholeRewriteBce:
		return fmt.Sprintf("BCE: bounds-check elided (proof from PC %d)", ann.origin)
	case peepholeRewriteUnroll:
		return fmt.Sprintf("inlined self-recursive body of call at PC %d", ann.origin)
	}
	return ""
}
