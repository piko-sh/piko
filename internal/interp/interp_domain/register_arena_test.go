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
	"sync"
	"testing"

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
	b.emit(opReturn, 1, 0, 0)

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

	a.Reset()

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
