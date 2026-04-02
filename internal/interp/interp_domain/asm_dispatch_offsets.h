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

// dispatch_offsets.h -- architecture-independent constants shared by both
// vm_dispatch_*_amd64.s and vm_dispatch_*_arm64.s.
//
// All offsets are verified at test time by vm_dispatch_offsets_test.go.

// DispatchContext field offsets (verified by TestDispatchContextOffsets):
//   codeBase      =   0    uintptr  pointer to Body[0]
//   codeLen       =   8    int64    number of instructions
//   pc            =  16    int64    current program counter
//   intsBase      =  24    uintptr  pointer to regs.Ints[0]
//   intsLen       =  32    int64    number of int registers
//   floatsBase    =  40    uintptr  pointer to regs.Floats[0]
//   floatsLen     =  48    int64    number of float registers
//   intConstsBase =  56    uintptr  pointer to IntConstants[0]
//   intConstsLen  =  64    int64    number of int constants
//   fltConstsBase =  72    uintptr  pointer to FloatConstants[0]
//   fltConstsLen  =  80    int64    number of float constants
//   jumpTable     =  88    uintptr  pointer to jump table[0]
//   exitReason    =  96    int64    exit reason code
//   exitPC        = 104    int64    PC at exit point

// Instruction encoding: {Op uint8, A uint8, B uint8, C uint8} = 4 bytes

// Exit reason constants:
#define EXIT_END_OF_CODE    0
#define EXIT_TIER2          1
#define EXIT_DIV_BY_ZERO    2
#define EXIT_CALL           3
#define EXIT_RETURN         4
#define EXIT_RETURN_VOID    5
#define EXIT_TAIL_CALL      6
#define EXIT_CALL_OVERFLOW  7

// callFrame size in bytes (verified by TestCallFrameOffsets).
#define CALLFRAME_SIZE 320

// callFrame field offsets:
#define CF_REGS_INTS_PTR    0
#define CF_REGS_INTS_LEN    8
#define CF_REGS_FLOATS_PTR  24
#define CF_REGS_FLOATS_LEN  32
#define CF_REGS_STRINGS_PTR 48
#define CF_REGS_STRINGS_LEN 56
#define CF_REGS_BOOLS_PTR   96
#define CF_REGS_BOOLS_LEN   104
#define CF_REGS_UINTS_PTR   120
#define CF_REGS_UINTS_LEN   128
#define CF_FUNCTION         168
#define CF_SHARED_CELLS     176
#define CF_UPVALUES_PTR     184
#define CF_RETURNDEST_PTR   208
#define CF_RETURNDEST_LEN   216
#define CF_RETURNDEST_CAP   224
#define CF_PROGRAM_COUNTER  232
#define CF_DEFERBASE        240
#define CF_ARENA_SAVE       248

// DispatchContext extended field offsets (inline call/return):
#define CTX_ASM_CALL_INFO_BASE  112
#define CTX_CSTACK_BASE         120
#define CTX_CSTACK_LEN          128
#define CTX_FRAME_POINTER       136
#define CTX_BASE_FRAME_POINTER  144
#define CTX_DEPTH_LIMIT         152
#define CTX_ARENA_INT_SLAB      160
#define CTX_ARENA_INT_CAP       168
#define CTX_ARENA_INT_IDX       176
#define CTX_ARENA_FLT_SLAB      184
#define CTX_ARENA_FLT_CAP       192
#define CTX_ARENA_FLT_IDX       200
#define CTX_ARENA_STR_IDX       208
#define CTX_ARENA_GEN_IDX       216
#define CTX_ARENA_BOOL_IDX      224
#define CTX_ARENA_UINT_IDX      232
#define CTX_ARENA_CPLX_IDX      240
#define CTX_DEFER_STACK_LEN     248
#define CTX_ASM_CI_PTRS         256
#define CTX_DISPATCH_SAVES      264
#define CTX_STRINGS_BASE        272
#define CTX_UINTS_BASE          280
#define CTX_BOOLS_BASE          288
#define CTX_ARENA_STR_SLAB      296
#define CTX_ARENA_STR_CAP       304
#define CTX_ARENA_BOOL_SLAB     312
#define CTX_ARENA_BOOL_CAP      320
#define CTX_ARENA_UINT_SLAB     328
#define CTX_ARENA_UINT_CAP      336

// asmCallInfo field offsets (verified by TestASMCallInfoOffsets):
#define ACI_CALLEE_FUNCTION     0
#define ACI_CALLEE_BODY         8
#define ACI_CALLEE_BODY_LEN     16
#define ACI_CALLEE_INT_CONSTS   24
#define ACI_CALLEE_FLT_CONSTS   32
#define ACI_CALLEE_NUM_INTS     40
#define ACI_CALLEE_NUM_FLOATS   48
#define ACI_NUM_INT_ARGS        56
#define ACI_INT_ARG_SRCS        64
#define ACI_NUM_FLOAT_ARGS      128
#define ACI_FLOAT_ARG_SRCS      136
#define ACI_NUM_RETURNS         200
#define ACI_RET_DEST_KIND       208
#define ACI_RET_DEST_REG        216
#define ACI_RET_DEST_PTR        224
#define ACI_RET_DEST_LEN        232
#define ACI_CALLEE_CALL_INFO    240
#define ACI_IS_FAST_PATH        248
#define ACI_CALLEE_NUM_STRINGS  256
#define ACI_CALLEE_NUM_BOOLS    264
#define ACI_CALLEE_NUM_UINTS    272
#define ACI_NUM_STRING_ARGS     280
#define ACI_STRING_ARG_SRCS     288
#define ACI_NUM_BOOL_ARGS       352
#define ACI_BOOL_ARG_SRCS       360
#define ACI_NUM_UINT_ARGS       424
#define ACI_UINT_ARG_SRCS       432

// sizeof(asmCallInfo) = 512 (power of 2, use shift).
#define ACI_SIZE_SHIFT 9

// varLocation field offsets:
#define VL_UPVALUE_INDEX  0
#define VL_REGISTER       8
#define VL_KIND           9
#define VL_IS_UPVALUE     10
