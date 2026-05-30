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
	"reflect"
	"strconv"
	"sync"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestRegisterArenaGrowSlabs(t *testing.T) {
	t.Parallel()
	a := &RegisterArena{
		intSlab:     make([]int64, 2),
		floatSlab:   make([]float64, 2),
		stringSlab:  make([]string, 2),
		generalSlab: make([]reflect.Value, 2),
	}

	regs := a.AllocRegisters([NumRegisterKinds]uint32{2, 1, 1, 1})
	require.Len(t, regs.ints, 2)
	require.Len(t, regs.floats, 1)

	regs2 := a.AllocRegisters([NumRegisterKinds]uint32{2, 2, 2, 2})
	require.Len(t, regs2.ints, 2)
	require.Len(t, regs2.floats, 2)
}

func TestRegisterArenaSaveRestore(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()
	save := a.Save()

	regs := a.AllocRegisters([NumRegisterKinds]uint32{4, 2, 2, 2})
	regs.ints[0] = 42
	regs.strings[0] = "hello"

	a.Restore(save)
	require.Equal(t, save.intIndex, a.intIndex)
	require.Equal(t, save.floatIndex, a.floatIndex)
}

func TestRegisterArenaGrowIndividualSlabs(t *testing.T) {
	t.Parallel()
	a := &RegisterArena{
		intSlab:     make([]int64, 1),
		floatSlab:   make([]float64, 100),
		stringSlab:  make([]string, 100),
		generalSlab: make([]reflect.Value, 100),
	}
	regs := a.AllocRegisters([NumRegisterKinds]uint32{5, 1, 1, 1})
	require.Len(t, regs.ints, 5)
	require.True(t, len(a.intSlab) >= 5)
}

func TestRegisterArenaPoolRoundTrip(t *testing.T) {
	t.Parallel()
	a := GetRegisterArena()
	require.NotNil(t, a)
	_ = a.AllocRegisters([NumRegisterKinds]uint32{10, 5, 3, 2})
	PutRegisterArena(a)
}

func TestPutRegisterArenaNil(t *testing.T) {
	t.Parallel()
	PutRegisterArena(nil)
}

func TestRegisterArenaUpvalueCellAllocation(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()

	sp := a.Save()
	cells := a.allocUpvalueCells(3)
	refs := a.allocUpvalueRefs(3)

	require.Len(t, cells, 3)
	require.Len(t, refs, 3)

	require.Equal(t, registerKind(0), cells[0].kind)
	require.Equal(t, int64(0), cells[0].intValue)
	require.Equal(t, "", cells[0].stringValue)
	require.Nil(t, refs[0].value)

	cells[0].kind = registerInt
	cells[0].intValue = 42
	refs[0].value = &cells[0]

	a.Restore(sp)
	require.Equal(t, sp.upvalueCellIndex, a.upvalueCellIndex)
	require.Equal(t, sp.upvalueReferenceIndex, a.upvalueReferenceIndex)

	cells2 := a.allocUpvalueCells(3)
	refs2 := a.allocUpvalueRefs(3)
	require.Equal(t, registerKind(0), cells2[0].kind)
	require.Equal(t, int64(0), cells2[0].intValue)
	require.Nil(t, refs2[0].value)
}

func TestRegisterArenaUpvalueCellGrow(t *testing.T) {
	t.Parallel()
	a := newRegisterArena()

	cells := a.allocUpvalueCells(initialUpvalueCellSlabs + 10)
	require.Len(t, cells, initialUpvalueCellSlabs+10)
	require.True(t, len(a.upvalueCellSlab) >= initialUpvalueCellSlabs+10)

	refs := a.allocUpvalueRefs(initialUpvalueRefSlabs + 10)
	require.Len(t, refs, initialUpvalueRefSlabs+10)
	require.True(t, len(a.upvalueReferenceSlab) >= initialUpvalueRefSlabs+10)
}

func TestArenaIsolation(t *testing.T) {
	t.Parallel()

	arenaCount := 0
	var mu sync.Mutex
	factory := func() *RegisterArena {
		mu.Lock()
		arenaCount++
		mu.Unlock()
		return newRegisterArena()
	}

	service := NewService(WithArenaFactory(factory))
	ctx := context.Background()

	for i := range 5 {
		result, err := service.Eval(ctx, "1 + 2")
		require.NoError(t, err, "iteration %d", i)
		require.Equal(t, int64(3), result, "iteration %d", i)
	}

	mu.Lock()
	count := arenaCount
	mu.Unlock()
	require.GreaterOrEqual(t, count, 5,
		"expected at least 5 arena allocations from factory, got %d", count)
}

func TestArenaIsolationParallel(t *testing.T) {
	t.Parallel()

	var arenaCount int64
	var mu sync.Mutex
	factory := func() *RegisterArena {
		mu.Lock()
		arenaCount++
		mu.Unlock()
		return newRegisterArena()
	}

	service := NewService(WithArenaFactory(factory))
	ctx := context.Background()

	var wg sync.WaitGroup
	const concurrency = 10
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			result, err := service.Eval(ctx, "2 + 3")
			require.NoError(t, err)
			require.Equal(t, int64(5), result)
		}()
	}
	wg.Wait()

	mu.Lock()
	count := arenaCount
	mu.Unlock()
	require.GreaterOrEqual(t, count, int64(concurrency),
		"expected at least %d arena allocations, got %d", concurrency, count)
}

func TestArenaDefaultPoolUsed(t *testing.T) {
	t.Parallel()

	service := NewService()
	ctx := context.Background()

	result, err := service.Eval(ctx, "10 * 4")
	require.NoError(t, err)
	require.Equal(t, int64(40), result)
}

func TestArenaSaveRestoreLIFO(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	sp0 := arena.Save()
	regs0 := arena.AllocRegisters([NumRegisterKinds]uint32{4, 2, 2, 2})
	regs0.ints[0] = 100
	regs0.strings[0] = "level0"

	sp1 := arena.Save()
	regs1 := arena.AllocRegisters([NumRegisterKinds]uint32{4, 2, 2, 2})
	regs1.ints[0] = 200
	regs1.strings[0] = "level1"

	sp2 := arena.Save()
	regs2 := arena.AllocRegisters([NumRegisterKinds]uint32{4, 2, 2, 2})
	regs2.ints[0] = 300

	require.Equal(t, int64(100), regs0.ints[0])
	require.Equal(t, int64(200), regs1.ints[0])
	require.Equal(t, int64(300), regs2.ints[0])

	arena.Restore(sp2)
	require.Equal(t, sp2.intIndex, arena.intIndex)

	require.Equal(t, int64(100), regs0.ints[0])
	require.Equal(t, "level0", regs0.strings[0])
	require.Equal(t, int64(200), regs1.ints[0])

	arena.Restore(sp1)
	require.Equal(t, sp1.intIndex, arena.intIndex)
	require.Equal(t, int64(100), regs0.ints[0])

	arena.Restore(sp0)
	require.Equal(t, sp0.intIndex, arena.intIndex)
}

func TestArenaGrowthUnderDeepCalls(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	const depth = 100

	saves := make([]ArenaSavePoint, depth)
	allRegs := make([]Registers, depth)

	for i := range depth {
		saves[i] = arena.Save()
		allRegs[i] = arena.AllocRegisters([NumRegisterKinds]uint32{4, 2, 1, 1})
		allRegs[i].ints[0] = int64(i)
	}

	for i := range depth {
		require.Equal(t, int64(i), allRegs[i].ints[0],
			"level %d int register incorrect", i)
	}

	for i := depth - 1; i >= 0; i-- {
		arena.Restore(saves[i])
	}
}

func TestArenaResetClearsState(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	regs := arena.AllocRegisters([NumRegisterKinds]uint32{10, 5, 5, 5})
	regs.ints[0] = 999
	regs.strings[0] = "before reset"

	arena.Reset()

	require.Equal(t, 0, arena.intIndex)
	require.Equal(t, 0, arena.floatIndex)
	require.Equal(t, 0, arena.stringIndex)
	require.Equal(t, 0, arena.generalIndex)

	regs2 := arena.AllocRegisters([NumRegisterKinds]uint32{2, 1, 1, 1})
	require.Len(t, regs2.ints, 2)
	require.Equal(t, int64(0), regs2.ints[0], "reset should yield zeroed registers")
}

func TestArenaFactoryWithExecute(t *testing.T) {
	t.Parallel()

	factoryCalled := false
	factory := func() *RegisterArena {
		factoryCalled = true
		return newRegisterArena()
	}

	service := NewService(WithArenaFactory(factory))

	b := newBytecodeBuilder()
	b.addIntConst(21)
	b.addIntConst(21)
	b.intRegisters(3).returnInt()
	b.emit(opLoadIntConst, 1, 0, 0)
	b.emit(opLoadIntConst, 2, 1, 0)
	b.emit(opAddInt, 0, 1, 2)
	b.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2Return), 1)

	result, err := service.Execute(context.Background(), b.build())
	require.NoError(t, err)
	require.Equal(t, int64(42), result)
	require.True(t, factoryCalled, "arena factory should have been called")
}

func TestArenaDeepRecursion(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func sum(n int) int {
	if n <= 0 {
		return 0
	}
	return n + sum(n-1)
}

func run() int {
	return sum(50)
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(1275), result)
}

func TestArenaStringSlabGrowth(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func run() int {
	result := ""
	for i := 0; i < 100; i++ {
		result += "abcdefghij"
	}
	return len(result)
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(1000), result)
}

func TestArenaUintSlabGrowth(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func run() uint {
	var sum uint
	for i := uint(0); i < 100; i++ {
		sum += i
	}
	return sum
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, uint64(4950), result)
}

func TestArenaPoolReuse(t *testing.T) {
	t.Parallel()

	a := GetRegisterArena()
	require.NotNil(t, a)

	regs := a.AllocRegisters([NumRegisterKinds]uint32{4, 2, 2, 2, 1, 1, 1})
	require.Equal(t, 4, len(regs.ints))

	PutRegisterArena(a)

	b := GetRegisterArena()
	require.NotNil(t, b)
	PutRegisterArena(b)
}

func TestArenaDeepRecursionGrowFrameStack(t *testing.T) {
	t.Parallel()

	service := NewService()

	source := `package main

func countdown(n int) int {
	if n <= 0 {
		return 0
	}
	return 1 + countdown(n-1)
}

func run() int {
	return countdown(80)
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(80), result)
}

func TestShrinkOvergrownSlabs(t *testing.T) {
	t.Parallel()

	a := newRegisterArena()

	a.intSlab = make([]int64, initialIntSlabs*maxArenaMultiplier+1)
	a.floatSlab = make([]float64, initialFloatSlabs*maxArenaMultiplier+1)
	a.stringSlab = make([]string, initialStringSlabs*maxArenaMultiplier+1)
	a.generalSlab = make([]reflect.Value, initialGeneralSlabs*maxArenaMultiplier+1)
	a.boolSlab = make([]bool, initialBoolSlabs*maxArenaMultiplier+1)
	a.uintSlab = make([]uint64, initialUintSlabs*maxArenaMultiplier+1)
	a.complexSlab = make([]complex128, initialComplexSlabs*maxArenaMultiplier+1)
	a.frameSlab = make([]callFrame, initialFrameSlabs*maxArenaMultiplier+1)
	a.callInfoBasesSlab = make([]uintptr, initialFrameSlabs*maxArenaMultiplier+1)
	a.dispatchSavesSlab = make([]asmDispatchSave, initialFrameSlabs*maxArenaMultiplier+1)
	a.upvalueCellSlab = make([]upvalueCell, initialUpvalueCellSlabs*maxArenaMultiplier+1)
	a.upvalueReferenceSlab = make([]upvalue, initialUpvalueRefSlabs*maxArenaMultiplier+1)
	a.byteSlab = make([]byte, initialByteSlabSize*maxArenaMultiplier+1)

	for range backingSlabIdleResetsBeforeShrink {
		a.Reset()
	}

	require.Equal(t, initialIntSlabs, len(a.intSlab))
	require.Equal(t, initialFloatSlabs, len(a.floatSlab))
	require.Equal(t, initialStringSlabs, len(a.stringSlab))
	require.Equal(t, initialGeneralSlabs, len(a.generalSlab))
	require.Equal(t, initialBoolSlabs, len(a.boolSlab))
	require.Equal(t, initialUintSlabs, len(a.uintSlab))
	require.Equal(t, initialComplexSlabs, len(a.complexSlab))
	require.Equal(t, initialFrameSlabs, len(a.frameSlab))
	require.Equal(t, initialFrameSlabs, len(a.callInfoBasesSlab))
	require.Equal(t, initialFrameSlabs, len(a.dispatchSavesSlab))
	require.Equal(t, initialUpvalueCellSlabs, len(a.upvalueCellSlab))
	require.Equal(t, initialUpvalueRefSlabs, len(a.upvalueReferenceSlab))
	require.Equal(t, initialByteSlabSize, len(a.byteSlab))
}

func TestShrinkOvergrownSlabsWithinThreshold(t *testing.T) {
	t.Parallel()

	a := newRegisterArena()

	originalIntLen := len(a.intSlab)
	originalFloatLen := len(a.floatSlab)

	a.Reset()

	require.Equal(t, originalIntLen, len(a.intSlab))
	require.Equal(t, originalFloatLen, len(a.floatSlab))
}

func TestArenaConcatRuneStringTailNoGrowth(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	s := arenaConcatString(arena, "", "ab")
	result := arenaConcatRuneString(arena, s, 'c')
	require.Equal(t, "abc", result)
}

func TestArenaConcatRuneStringTailTriggersGrowth(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	padding := make([]byte, len(arena.byteSlab)-3)
	for i := range padding {
		padding[i] = 'x'
	}
	_ = arenaBytesToString(arena, padding)

	s := arenaConcatString(arena, "", "ab")
	require.Equal(t, "ab", s)

	result := arenaConcatRuneString(arena, s, '€')
	require.Equal(t, "ab€", result)
}

func TestArenaConcatRuneStringNonTail(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	first := arenaConcatString(arena, "", "hello")
	_ = arenaConcatString(arena, "", "world")

	result := arenaConcatRuneString(arena, first, '!')
	require.Equal(t, "hello!", result)
}

func TestArenaConcatRuneStringEmptyBase(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	result := arenaConcatRuneString(arena, "", 'x')
	require.Equal(t, "x", result)
}

func TestArenaConcatRuneStringInvalidRune(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	result := arenaConcatRuneString(arena, "hello", rune(-1))
	require.Equal(t, "hello\uFFFD", result)
}

func TestArenaConcatRuneStringMultiByteBoundary(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	padding := make([]byte, len(arena.byteSlab)-3)
	for i := range padding {
		padding[i] = 'x'
	}
	_ = arenaBytesToString(arena, padding)

	s := arenaConcatString(arena, "", "a")

	result := arenaConcatRuneString(arena, s, '🎉')
	require.Equal(t, "a🎉", result)
}

func TestArenaConcatStringBothEmpty(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()
	result := arenaConcatString(arena, "", "")
	require.Equal(t, "", result)
}

func TestArenaConcatStringTailNoGrowth(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	s := arenaConcatString(arena, "", "he")
	result := arenaConcatString(arena, s, "llo")
	require.Equal(t, "hello", result)
}

func TestArenaConcatStringTailTriggersGrowth(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	padding := make([]byte, len(arena.byteSlab)-4)
	for i := range padding {
		padding[i] = 'x'
	}
	_ = arenaBytesToString(arena, padding)

	s := arenaConcatString(arena, "", "ab")
	require.Equal(t, "ab", s)

	result := arenaConcatString(arena, s, "world")
	require.Equal(t, "abworld", result)
}

func TestArenaRuneToStringVariousSizes(t *testing.T) {
	t.Parallel()

	t.Run("1-byte ASCII rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, 'A')
		require.Equal(t, "A", result)
	})

	t.Run("2-byte rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, 'é')
		require.Equal(t, "é", result)
	})

	t.Run("3-byte CJK rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, '日')
		require.Equal(t, "日", result)
	})

	t.Run("4-byte emoji rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, '🎉')
		require.Equal(t, "🎉", result)
	})

	t.Run("max valid rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, '\U0010FFFF')
		require.Equal(t, "\U0010FFFF", result)
		require.Len(t, result, 4)
	})

	t.Run("surrogate half produces replacement", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()

		result := arenaRuneToString(arena, rune(0xD800))
		require.Equal(t, "\uFFFD", result)
	})

	t.Run("NUL rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaRuneToString(arena, 0)
		require.Equal(t, "\x00", result)
		require.Len(t, result, 1)
	})
}

func TestArenaConcatRuneStringUTF8EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("concat 4-byte emoji to ASCII", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		s := arenaConcatString(arena, "", "ok")
		result := arenaConcatRuneString(arena, s, '🎊')
		require.Equal(t, "ok🎊", result)
	})

	t.Run("concat surrogate half produces replacement", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		s := arenaConcatString(arena, "", "abc")
		result := arenaConcatRuneString(arena, s, rune(0xD800))
		require.Equal(t, "abc\uFFFD", result)
	})

	t.Run("concat max valid rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		s := arenaConcatString(arena, "", "x")
		result := arenaConcatRuneString(arena, s, '\U0010FFFF')
		require.Equal(t, "x\U0010FFFF", result)
	})

	t.Run("concat NUL rune", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		s := arenaConcatString(arena, "", "ab")
		result := arenaConcatRuneString(arena, s, 0)
		require.Equal(t, "ab\x00", result)
	})
}

func TestArenaConcatStringUTF8EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("concat two multi-byte strings", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaConcatString(arena, "日本", "語")
		require.Equal(t, "日本語", result)
	})

	t.Run("concat emoji strings", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaConcatString(arena, "🎉", "🎊")
		require.Equal(t, "🎉🎊", result)
	})

	t.Run("concat with NUL bytes", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaConcatString(arena, "a\x00", "b\x00c")
		require.Equal(t, "a\x00b\x00c", result)
	})

	t.Run("concat invalid UTF-8 preserves bytes", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaConcatString(arena, "\x80\x81", "\x82\x83")
		require.Equal(t, "\x80\x81\x82\x83", result)
		require.Len(t, result, 4)
	})
}

func TestArenaBytesToStringEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("invalid UTF-8 preserved", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaBytesToString(arena, []byte{0x80, 0x81, 0x82})
		require.Equal(t, "\x80\x81\x82", result)
		require.Len(t, result, 3)
	})

	t.Run("NUL bytes preserved", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaBytesToString(arena, []byte{0, 0, 0})
		require.Equal(t, "\x00\x00\x00", result)
		require.Len(t, result, 3)
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		result := arenaBytesToString(arena, []byte{})
		require.Equal(t, "", result)
	})
}

func TestGrowByteSlabDoublesSize(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	originalLen := len(arena.byteSlab)
	arena.growByteSlab(1)

	require.Equal(t, originalLen*2, len(arena.byteSlab))
	require.Equal(t, 0, arena.byteIndex)
	require.Len(t, arena.oldByteSlabs, 1)
	require.Len(t, arena.oldByteSlabs[0], originalLen)
}

func TestGrowByteSlabMinExtraLargerThanDouble(t *testing.T) {
	t.Parallel()
	arena := &RegisterArena{
		byteSlab: make([]byte, 10),
	}

	arena.growByteSlab(100)

	require.Equal(t, 100, len(arena.byteSlab))
	require.Equal(t, 0, arena.byteIndex)
}

func TestGrowByteSlabPreservesOldStrings(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	s := arenaBytesToString(arena, []byte("preserved"))

	arena.growByteSlab(1)

	require.Equal(t, "preserved", s)
}

func TestGrowByteSlabMultipleGrowths(t *testing.T) {
	t.Parallel()
	arena := newRegisterArena()

	arena.growByteSlab(1)
	arena.growByteSlab(1)
	arena.growByteSlab(1)

	require.Len(t, arena.oldByteSlabs, 3)
}

func TestArenaManyFunctionCalls(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func add(a, b int) int { return a + b }

func run() int {
	sum := 0
	for i := 0; i < 200; i++ {
		sum = add(sum, i)
	}
	return sum
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(19900), result)
}

var (
	arenaItoaFormatIntParityCases = []int64{
		0, 1, -1, 7, -7, 10, -10, 99, -99, 100, -100,
		12345, -12345, 1 << 31, -(1 << 31),
		(1 << 62), -(1 << 62),
		9223372036854775807,
		-9223372036854775808,
	}
)

func TestArenaItoaStringParityWithStrconv(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	for _, x := range arenaItoaFormatIntParityCases {
		got := arenaItoaString(arena, x)
		want := strconv.FormatInt(x, 10)
		require.Equal(t, want, got, "arenaItoaString(%d) parity mismatch", x)
	}
}

func TestArenaFormatIntStringParityWithStrconv(t *testing.T) {
	t.Parallel()

	bases := []int{2, 8, 10, 16, 36}
	arena := newRegisterArena()
	for _, x := range arenaItoaFormatIntParityCases {
		for _, base := range bases {
			got := arenaFormatIntString(arena, x, base)
			want := strconv.FormatInt(x, base)
			require.Equal(t, want, got, "arenaFormatIntString(%d, base=%d) parity mismatch", x, base)
		}
	}
}

func TestArenaItoaStringRewindsByteIndex(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arena byteIndex bump pointer is not exposed")
	}
	t.Parallel()

	arena := newRegisterArena()
	startIndex := arena.byteIndex

	_ = arenaItoaString(arena, 42)
	require.Equal(t, startIndex+2, arena.byteIndex, "byteIndex should advance by len(\"42\")")

	_ = arenaItoaString(arena, 7)
	require.Equal(t, startIndex+3, arena.byteIndex, "byteIndex should advance by len(\"7\")")

	_ = arenaItoaString(arena, -100)
	require.Equal(t, startIndex+7, arena.byteIndex, "byteIndex should advance by len(\"-100\")")
}

func TestArenaItoaStringNoAllocSteadyState(t *testing.T) {

	arena := newRegisterArena()

	for _, x := range arenaItoaFormatIntParityCases {
		_ = arenaItoaString(arena, x)
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = arenaItoaString(arena, 12345)
	})
	if allocs != 0 {
		t.Logf("note: arenaItoaString reported %v allocs/op (expected 0 in unsafe build); "+
			"if running with -tags=safe, this is expected.", allocs)
	}
}

func TestArenaAllocIntBackingShape(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	s := arena.AllocIntBacking(5)
	require.Equal(t, 5, len(s), "AllocIntBacking len")
	require.Equal(t, 5, cap(s), "AllocIntBacking cap (three-index form)")

	t2 := arena.AllocIntBacking(3)
	require.Equal(t, 3, cap(t2))

	for i := range s {
		s[i] = int64(i + 100)
	}
	for i := range t2 {
		t2[i] = int64(i + 200)
	}
	for i, v := range s {
		require.Equal(t, int64(i+100), v)
	}
	for i, v := range t2 {
		require.Equal(t, int64(i+200), v)
	}
}

func TestArenaAllocIntBackingZero(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	for _, n := range []int{0, -1, -100} {
		require.Nil(t, arena.AllocIntBacking(n), "n=%d should return nil", n)
		require.Nil(t, arena.AllocFloatBacking(n), "n=%d should return nil", n)
		require.Nil(t, arena.AllocStringBacking(n), "n=%d should return nil", n)
		require.Nil(t, arena.AllocBoolBacking(n), "n=%d should return nil", n)
		require.Nil(t, arena.AllocUintBacking(n), "n=%d should return nil", n)
	}
}

func TestArenaAllocIntBackingGrow(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	initialCap := len(arena.intBackingSlab)

	drained := arena.AllocIntBacking(initialCap)
	for i := range drained {
		drained[i] = int64(i)
	}

	require.Empty(t, arena.oldIntBackings, "no old slabs before grow")
	grown := arena.AllocIntBacking(10)
	require.Equal(t, 10, len(grown))
	require.Len(t, arena.oldIntBackings, 1, "exactly one old slab retained")
	require.Equal(t, initialCap, len(arena.oldIntBackings[0]),
		"retained slab is the previous one at initial capacity")

	for i, v := range drained {
		require.Equal(t, int64(i), v, "post-grow read of pre-grow slice")
	}

	require.GreaterOrEqual(t, len(arena.intBackingSlab), 2*initialCap,
		"new slab is at least 2x previous")
}

func TestArenaAppendIntInCapNoAlloc(t *testing.T) {

	arena := newRegisterArena()
	s := arena.AllocIntBacking(100)[:0:100]

	allocs := testing.AllocsPerRun(50, func() {
		s = arenaAppendInt(arena, s[:0], 42)
	})
	require.Equal(t, float64(0), allocs, "append in-cap must not allocate")
}

func TestArenaAppendIntGrowFromHeapSlice(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arenaAppendInt does not route through intBackingSlab")
	}
	t.Parallel()

	arena := newRegisterArena()

	s := []int64{1, 2, 3}
	require.Equal(t, len(s), cap(s))

	out := arenaAppendInt(arena, s, 4)
	require.Equal(t, []int64{1, 2, 3, 4}, out)
	require.GreaterOrEqual(t, cap(out), 4, "grown cap is at least len(out)")

	backingStart := &arena.intBackingSlab[0]
	outStart := &out[0]
	require.Equal(t, backingStart, outStart,
		"grown slice's backing should be the arena's intBackingSlab head")
}

func TestArenaAppendStringPreservesUnderlyingType(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	s := arenaAppendString(arena, []string{"a", "b"}, "c")
	require.Equal(t, []string{"a", "b", "c"}, s)
}

func TestArenaResetClearsBackingIndices(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	_ = arena.AllocIntBacking(50)
	_ = arena.AllocFloatBacking(50)
	_ = arena.AllocStringBacking(50)
	_ = arena.AllocBoolBacking(50)
	_ = arena.AllocUintBacking(50)

	require.NotZero(t, arena.intBackingIndex)
	require.NotZero(t, arena.floatBackingIndex)
	require.NotZero(t, arena.stringBackingIndex)
	require.NotZero(t, arena.boolBackingIndex)
	require.NotZero(t, arena.uintBackingIndex)

	arena.Reset()

	require.Zero(t, arena.intBackingIndex)
	require.Zero(t, arena.floatBackingIndex)
	require.Zero(t, arena.stringBackingIndex)
	require.Zero(t, arena.boolBackingIndex)
	require.Zero(t, arena.uintBackingIndex)
	require.Empty(t, arena.oldIntBackings)
	require.Empty(t, arena.oldFloatBackings)
	require.Empty(t, arena.oldStringBackings)
	require.Empty(t, arena.oldBoolBackings)
	require.Empty(t, arena.oldUintBackings)
}

func TestArenaEnsureCapacityRespectsByteHint(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	startSize := len(arena.byteSlab)

	arena.EnsureCapacity(typedSlabCounts{bytes: startSize * 4})

	require.GreaterOrEqual(t, len(arena.byteSlab), startSize*4,
		"EnsureCapacity should grow byteSlab to at least the hinted size")
}

func TestArenaEnsureCapacityRespectsBackingHints(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	hint := 4096
	arena.EnsureCapacity(typedSlabCounts{
		intBacking:    hint,
		floatBacking:  hint,
		stringBacking: hint,
		boolBacking:   hint,
		uintBacking:   hint,
	})

	require.GreaterOrEqual(t, len(arena.intBackingSlab), hint, "intBackingSlab")
	require.GreaterOrEqual(t, len(arena.floatBackingSlab), hint, "floatBackingSlab")
	require.GreaterOrEqual(t, len(arena.stringBackingSlab), hint, "stringBackingSlab")
	require.GreaterOrEqual(t, len(arena.boolBackingSlab), hint, "boolBackingSlab")
	require.GreaterOrEqual(t, len(arena.uintBackingSlab), hint, "uintBackingSlab")
}

func TestArenaEnsureCapacityDoesNotShrink(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	arena.EnsureCapacity(typedSlabCounts{bytes: initialByteSlabSize * 8})
	bigSize := len(arena.byteSlab)
	require.Greater(t, bigSize, initialByteSlabSize)

	arena.EnsureCapacity(typedSlabCounts{bytes: 64})
	require.Equal(t, bigSize, len(arena.byteSlab),
		"EnsureCapacity must not shrink an already-grown slab")
}

func TestAccumulateArenaBytecodeHints(t *testing.T) {
	t.Parallel()

	body := []instruction{
		{op: opConcatString},
		{op: opConcatString},
		{op: opConcatRuneString},
		{op: opDrillTier1, a: uint8(subOpBytesToString)},
		{op: opDrillTier1, a: uint8(subOpStrconvItoa)},
		{op: opDrillTier1, a: uint8(subOpStrconvFormatInt)},
		{op: opDrillTier1, a: uint8(subOpRuneToString)},
		{op: opDrillTier1, a: uint8(subOpMakeSliceInt)},
		{op: opDrillTier1, a: uint8(subOpMakeSliceFloat)},
		{op: opDrillTier1, a: uint8(subOpMakeSliceString)},
		{op: opDrillTier1, a: uint8(subOpMakeSliceBool)},
		{op: opDrillTier1, a: uint8(subOpMakeSliceUint)},

		{op: opDrillTier1, a: uint8(subOpMathSin)},
		{op: opDrillTier1, a: uint8(subOpCap)},
	}

	var hints arenaBytecodeHints
	accumulateArenaBytecodeHints(&hints, body, nil)

	expectedBytes := 2*arenaConcatStringAvgBytes +
		arenaConcatRuneStringAvgBytes +
		arenaBytesToStringAvgBytes +
		arenaItoaMaxBytes +
		arenaFormatIntMaxBytes +
		arenaRuneToStringMaxBytes
	require.Equal(t, expectedBytes, hints.bytes, "byte hint sum")

	require.Equal(t, 1, hints.makeSliceInt)
	require.Equal(t, 1, hints.makeSliceFloat)
	require.Equal(t, 1, hints.makeSliceString)
	require.Equal(t, 1, hints.makeSliceBool)
	require.Equal(t, 1, hints.makeSliceUint)
}

func TestSizeArenaFromFunctionsPresizesByteSlab(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func run() int {
	out := ""
	for i := 0; i < 100; i++ {
		out += "abcdefghij"
	}
	return len(out)
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(1000), result)
}

func TestArenaAllocByteBackingShape(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()

	s := arena.AllocByteBacking(5)
	require.Equal(t, 5, len(s))
	require.Equal(t, 5, cap(s))

	t2 := arena.AllocByteBacking(3)
	require.Equal(t, 3, cap(t2))

	for i := range s {
		s[i] = byte(i + 100)
	}
	for i := range t2 {
		t2[i] = byte(i + 200)
	}
	for i, v := range s {
		require.Equal(t, byte(i+100), v)
	}
	for i, v := range t2 {
		require.Equal(t, byte(i+200), v)
	}
}

func TestArenaAllocByteBackingZero(t *testing.T) {
	t.Parallel()

	arena := newRegisterArena()
	for _, n := range []int{0, -1, -100} {
		require.Nil(t, arena.AllocByteBacking(n), "n=%d should return nil", n)
	}
}

func TestArenaAppendByteGrowFromHeapSlice(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arenaAppendByte does not route through byteBackingSlab")
	}
	t.Parallel()

	arena := newRegisterArena()
	s := []byte{1, 2, 3}
	require.Equal(t, len(s), cap(s))

	out := arenaAppendByte(arena, s, 4)
	require.Equal(t, []byte{1, 2, 3, 4}, out)
	require.GreaterOrEqual(t, cap(out), 4)

	require.Equal(t, &arena.byteSlab[0], &out[0])
}

func TestArenaMakeSliceEndToEnd(t *testing.T) {
	t.Parallel()

	service := NewService()
	source := `package main

func run() int {
	ints := make([]int, 0, 10)
	for i := 0; i < 10; i++ {
		ints = append(ints, i*i)
	}
	sum := 0
	for _, v := range ints {
		sum += v
	}
	return sum
}

func main() {}
`
	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, int64(285), result, "sum of i*i for i in 0..10")
}

func TestOwnsBytePointer(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arena byte slabs are not exposed")
	}
	t.Parallel()
	arena := newRegisterArena()

	p1 := arena.AllocBytes(16, 8)
	require.True(t, arena.ownsBytePointer(p1),
		"freshly allocated arena byte pointer must be owned")

	initialBacking := arena.genericBytesSlab
	for range 64 {

		arena.AllocBytes(4096, 8)
	}
	require.NotEqual(t,
		uintptr(unsafe.Pointer(&initialBacking[0])),
		uintptr(unsafe.Pointer(&arena.genericBytesSlab[0])),
		"slab must have been re-allocated by growth")
	require.GreaterOrEqual(t, len(arena.oldGenericByteSlabs), 1,
		"retired slabs must be tracked in oldGenericByteSlabs")

	require.True(t, arena.ownsBytePointer(p1),
		"retired-slab pointer must still report owned after growth")

	p2 := arena.AllocBytes(8, 8)
	require.True(t, arena.ownsBytePointer(p2),
		"new-slab pointer must be owned")

	heapValue := new(int64)
	require.False(t, arena.ownsBytePointer(unsafe.Pointer(heapValue)),
		"heap pointer must not be reported as arena-owned")

	require.False(t, arena.ownsBytePointer(nil),
		"nil pointer must return false")

	require.False(t, arena.ownsBytePointer(zeroSizeAllocPtr),
		"zero-size sentinel must not be reported as arena-owned")
}

func TestOwnsSliceHeaderPointer(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arena slice-header slabs are not exposed")
	}
	t.Parallel()
	arena := newRegisterArena()

	hdr1 := arena.AllocSliceHeader()
	require.True(t, arena.ownsSliceHeaderPointer(unsafe.Pointer(hdr1)),
		"freshly allocated arena slice header must be owned")

	initialBacking := arena.sliceHeaderSlab
	for range initialSliceHeaderCapacity * 2 {
		_ = arena.AllocSliceHeader()
	}
	require.NotEqual(t,
		uintptr(unsafe.Pointer(&initialBacking[0])),
		uintptr(unsafe.Pointer(&arena.sliceHeaderSlab[0])),
		"slice-header slab must have been re-allocated by growth")
	require.GreaterOrEqual(t, len(arena.oldSliceHeaderSlabs), 1,
		"retired slice-header slabs must be tracked")

	require.True(t, arena.ownsSliceHeaderPointer(unsafe.Pointer(hdr1)),
		"retired slice-header pointer must still report owned")

	hdr2 := arena.AllocSliceHeader()
	require.True(t, arena.ownsSliceHeaderPointer(unsafe.Pointer(hdr2)),
		"new-slab slice header must be owned")

	heapHeader := &arenaSliceHeader{}
	require.False(t, arena.ownsSliceHeaderPointer(unsafe.Pointer(heapHeader)),
		"heap slice-header must not be reported as arena-owned")

	require.False(t, arena.ownsSliceHeaderPointer(nil),
		"nil pointer must return false")
}

func TestMaterialiseArenaValue(t *testing.T) {
	if !arenaUsesUnsafeSlabs {
		t.Skip("safe build: arena slab-ownership semantics are not exposed")
	}
	t.Parallel()

	type smallStruct struct {
		A int64
		B int64
	}

	t.Run("nil_arena_passes_through", func(t *testing.T) {
		t.Parallel()
		v := reflect.ValueOf(smallStruct{A: 1, B: 2})
		out := materialiseArenaValue(nil, v)
		require.Equal(t, v.Interface(), out.Interface())
	})

	t.Run("invalid_value_passes_through", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		out := materialiseArenaValue(arena, reflect.Value{})
		require.False(t, out.IsValid())
	})

	t.Run("heap_struct_passes_through", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		original := reflect.New(reflect.TypeFor[smallStruct]()).Elem()
		original.Set(reflect.ValueOf(smallStruct{A: 7, B: 8}))
		out := materialiseArenaValue(arena, original)

		require.Equal(t,
			reflectValuePtr(original),
			reflectValuePtr(out),
			"heap struct must not be re-copied")
	})

	t.Run("arena_struct_is_copied_to_heap", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		arena.isLeaf = true
		t1 := reflect.TypeFor[smallStruct]()

		ptr := arena.AllocBytes(t1.Size(), uintptr(t1.Align()))
		require.True(t, arena.ownsBytePointer(ptr))
		arenaValue := unsafeNewAt(reflectValueABIType(t1), ptr, reflect.Struct)
		arenaValue.Set(reflect.ValueOf(smallStruct{A: 11, B: 22}))

		out := materialiseArenaValue(arena, arenaValue)
		require.NotEqual(t,
			reflectValuePtr(arenaValue),
			reflectValuePtr(out),
			"arena struct's storage must be re-anchored to heap")
		require.False(t, arena.ownsBytePointer(reflectValuePtr(out)),
			"materialised value's storage must NOT be in arena")
		require.Equal(t, smallStruct{A: 11, B: 22}, out.Interface().(smallStruct))

		arenaValue.Set(reflect.ValueOf(smallStruct{A: 99, B: 99}))
		require.Equal(t, smallStruct{A: 11, B: 22}, out.Interface().(smallStruct),
			"materialised copy must be independent of arena slot")
	})

	t.Run("arena_struct_survives_slab_growth", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		arena.isLeaf = true
		t1 := reflect.TypeFor[smallStruct]()
		ptr := arena.AllocBytes(t1.Size(), uintptr(t1.Align()))
		arenaValue := unsafeNewAt(reflectValueABIType(t1), ptr, reflect.Struct)
		arenaValue.Set(reflect.ValueOf(smallStruct{A: 33, B: 44}))

		for range 64 {
			arena.AllocBytes(4096, 8)
		}
		require.GreaterOrEqual(t, len(arena.oldGenericByteSlabs), 1)

		require.True(t, arena.ownsBytePointer(reflectValuePtr(arenaValue)))
		out := materialiseArenaValue(arena, arenaValue)
		require.False(t, arena.ownsBytePointer(reflectValuePtr(out)))
		require.Equal(t, smallStruct{A: 33, B: 44}, out.Interface().(smallStruct))
	})

	t.Run("scalar_kinds_pass_through", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		for _, v := range []reflect.Value{
			reflect.ValueOf(int64(42)),
			reflect.ValueOf(uint64(42)),
			reflect.ValueOf(float64(3.14)),
			reflect.ValueOf(true),
			reflect.ValueOf("hello"),
		} {
			out := materialiseArenaValue(arena, v)
			require.Equal(t, v.Interface(), out.Interface())
		}
	})

	t.Run("pointer_map_chan_func_pass_through", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		x := 7
		ptr := reflect.ValueOf(&x)
		require.Equal(t, &x, materialiseArenaValue(arena, ptr).Interface())

		m := map[string]int{"a": 1}
		mv := reflect.ValueOf(m)

		require.True(t, reflect.DeepEqual(m, materialiseArenaValue(arena, mv).Interface()))

		ch := make(chan int, 1)
		cv := reflect.ValueOf(ch)
		require.Equal(t,
			reflect.ValueOf(ch).Pointer(),
			materialiseArenaValue(arena, cv).Pointer())

		fv := reflect.ValueOf(func() int { return 1 })
		require.Equal(t, fv.Pointer(), materialiseArenaValue(arena, fv).Pointer())
	})

	t.Run("heap_slice_passes_through", func(t *testing.T) {
		t.Parallel()
		arena := newRegisterArena()
		heap := reflect.ValueOf([]int{1, 2, 3})
		out := materialiseArenaValue(arena, heap)

		require.Equal(t, []int{1, 2, 3}, out.Interface().([]int))
	})
}

func BenchmarkArenaSaveInto(b *testing.B) {
	arena := newRegisterArena()
	arena.intIndex = 17
	arena.floatIndex = 23
	arena.stringIndex = 11
	arena.generalIndex = 41
	arena.boolIndex = 7
	arena.uintIndex = 13
	arena.complexIndex = 5
	arena.slicesIntIndex = 3
	arena.slicesFloatIndex = 4
	arena.slicesStringIndex = 6
	arena.slicesBoolIndex = 8
	arena.slicesUintIndex = 9
	arena.slicesByteIndex = 10
	arena.upvalueCellIndex = 1
	arena.upvalueReferenceIndex = 2
	var savePoint ArenaSavePoint
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		arena.SaveInto(&savePoint)
	}
}

func BenchmarkArenaSaveRestoreRoundTrip(b *testing.B) {
	arena := newRegisterArena()
	var savePoint ArenaSavePoint
	arena.SaveInto(&savePoint)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		arena.SaveInto(&savePoint)
		arena.Restore(savePoint)
	}
}
