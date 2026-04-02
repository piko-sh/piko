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

package caller_test

import (
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/caller"
)

func TestCaller_ReturnsNonZeroPC(t *testing.T) {
	pc := caller.Caller(0)
	assert.NotEqual(t, caller.PC(0), pc, "Caller should return non-zero PC")
}

func TestCaller_SkipFrames(t *testing.T) {
	pc0 := caller.Caller(0)
	pc1 := innerCallerFunc()
	assert.NotEqual(t, pc0, pc1, "Different call sites should have different PCs")
}

func innerCallerFunc() caller.PC {
	return caller.Caller(0)
}

func TestCaller_MatchesRuntimeCaller(t *testing.T) {

	var runtimePC uintptr
	var callerPC caller.PC

	func() {
		runtimePC, _, _, _ = runtime.Caller(0)
		callerPC = caller.Caller(0)
	}()

	runtimeFunc := runtime.FuncForPC(runtimePC)
	require.NotNil(t, runtimeFunc)

	name, _, _ := callerPC.NameFileLine()
	assert.Contains(t, name, "TestCaller_MatchesRuntimeCaller")
	assert.Contains(t, runtimeFunc.Name(), "TestCaller_MatchesRuntimeCaller")
}

func TestNameFileLine_ReturnsCorrectInfo(t *testing.T) {
	pc := caller.Caller(0)
	name, file, line := pc.NameFileLine()

	assert.Contains(t, name, "TestNameFileLine_ReturnsCorrectInfo",
		"Function name should contain test function name")
	assert.True(t, strings.HasSuffix(file, "caller_test.go"),
		"File should end with caller_test.go, got: %s", file)
	assert.Greater(t, line, 0, "Line number should be positive")
}

func TestNameFileLine_CropsFilePath(t *testing.T) {
	pc := caller.Caller(0)
	_, file, _ := pc.NameFileLine()

	assert.False(t, strings.HasPrefix(file, "/"),
		"File path should be cropped, not absolute: %s", file)

	assert.True(t, strings.HasSuffix(file, "caller_test.go"),
		"File path should end with caller_test.go: %s", file)
}

func TestNameFileLine_CachesResults(t *testing.T) {
	pc := caller.Caller(0)

	name1, file1, line1 := pc.NameFileLine()

	name2, file2, line2 := pc.NameFileLine()

	assert.Equal(t, name1, name2, "Cached name should match")
	assert.Equal(t, file1, file2, "Cached file should match")
	assert.Equal(t, line1, line2, "Cached line should match")
}

func TestNameFileLine_ZeroPCReturnsEmpty(t *testing.T) {
	var pc caller.PC
	name, file, line := pc.NameFileLine()

	assert.Empty(t, name, "Zero PC should return empty name")
	assert.Empty(t, file, "Zero PC should return empty file")
	assert.Equal(t, 0, line, "Zero PC should return zero line")
}

func TestCallers_CapturesMultipleFrames(t *testing.T) {
	pcs := caller.Callers(0, 10)
	assert.Greater(t, len(pcs), 1, "Should capture multiple frames")
}

func TestCallers_RespectsMaxFrames(t *testing.T) {
	pcs := caller.Callers(0, 3)
	assert.LessOrEqual(t, len(pcs), 3, "Should not exceed requested frame count")
}

func TestCallers_SkipsFrames(t *testing.T) {
	pcs0 := caller.Callers(0, 5)
	pcs1 := caller.Callers(1, 5)

	require.Greater(t, len(pcs0), 1)
	require.Greater(t, len(pcs1), 0)

	assert.NotEqual(t, pcs0[0], pcs1[0],
		"Skipping frames should change the first captured frame")
}

func TestCallersFill_ReusesSlice(t *testing.T) {
	buffer := make(caller.PCs, 10)
	result := caller.CallersFill(0, buffer)

	assert.Greater(t, len(result), 0, "Should capture frames")
	assert.LessOrEqual(t, len(result), 10, "Should not exceed buffer size")

	if len(result) > 0 {
		assert.Equal(t, &buffer[0], &result[0],
			"Result should share backing memory with input")
	}
}

func TestCallers_AllFramesResolvable(t *testing.T) {
	pcs := caller.Callers(0, 10)
	require.Greater(t, len(pcs), 0)

	for i, pc := range pcs {
		name, file, line := pc.NameFileLine()
		assert.NotEmpty(t, name, "Frame %d should have a function name", i)
		assert.NotEmpty(t, file, "Frame %d should have a file path", i)
		assert.Greater(t, line, 0, "Frame %d should have a positive line number", i)
	}
}

func TestFormattedFrame_ReturnsCorrectFormat(t *testing.T) {
	pc := caller.Caller(0)
	frame := pc.FormattedFrame()

	assert.True(t, strings.HasPrefix(frame, "\t"),
		"Frame should start with tab, got: %q", frame)

	assert.Contains(t, frame, ":",
		"Frame should contain colon separator")

	_, file, line := pc.NameFileLine()
	expectedSuffix := file + ":"
	assert.Contains(t, frame, expectedSuffix,
		"Frame should contain file:, got: %q", frame)

	assert.Greater(t, line, 0, "Line number should be positive")
}

func TestFormattedFrame_ZeroPCReturnsEmpty(t *testing.T) {
	var pc caller.PC
	frame := pc.FormattedFrame()
	assert.Empty(t, frame, "Zero PC should return empty frame")
}

func TestFormattedFrame_CachesResults(t *testing.T) {
	pc := caller.Caller(0)

	frame1 := pc.FormattedFrame()

	frame2 := pc.FormattedFrame()

	assert.Equal(t, frame1, frame2, "Cached frame should match")
}

func TestFormattedFrame_ConsistentWithNameFileLine(t *testing.T) {
	pc := caller.Caller(0)
	_, file, line := pc.NameFileLine()
	frame := pc.FormattedFrame()

	expected := "\t" + file + ":" + strconv.Itoa(line)

	assert.Equal(t, expected, frame,
		"FormattedFrame should be consistent with NameFileLine")
}

func TestFormattedFrame_AsFirstCall(t *testing.T) {

	pc := caller.Caller(0)
	frame := pc.FormattedFrame()

	assert.True(t, strings.HasPrefix(frame, "\t"),
		"Frame should start with tab")
	assert.Contains(t, frame, "caller_test.go:",
		"Frame should contain filename and colon")

	_, file, line := pc.NameFileLine()
	expected := "\t" + file + ":" + strconv.Itoa(line)
	assert.Equal(t, expected, frame,
		"FormattedFrame should match NameFileLine data")
}

func TestCallers_ZeroMaxFrames(t *testing.T) {
	pcs := caller.Callers(0, 0)
	assert.Empty(t, pcs, "Zero max frames should return empty slice")
}

func TestCallersFill_EmptySlice(t *testing.T) {
	buffer := make(caller.PCs, 0)
	result := caller.CallersFill(0, buffer)
	assert.Empty(t, result, "Empty buffer should return empty result")
}

func TestCallersFill_NilSlice(t *testing.T) {
	var buffer caller.PCs
	result := caller.CallersFill(0, buffer)
	assert.Empty(t, result, "Nil buffer should return empty result")
}

func TestCropFilename_NoSlashInFuncName(t *testing.T) {

	file := "/home/user/project/main.go"
	functionName := "main.Run"

	result := caller.CropFilename(file, functionName)

	assert.Equal(t, "main.go", result,
		"Should return base filename when functionName has no slash")
}

func TestCropFilename_NoDotAfterSlash(t *testing.T) {

	file := "/home/user/project/src/main.go"
	functionName := "github.com/user/project/src"

	result := caller.CropFilename(file, functionName)

	assert.Equal(t, "main.go", result,
		"Should return base filename when no dot after last slash")
}

func TestCropFilename_NormalCase(t *testing.T) {
	file := "/home/user/go/src/piko.sh/piko/internal/caller/caller.go"
	functionName := "piko.sh/piko/internal/caller.Caller"

	result := caller.CropFilename(file, functionName)

	assert.True(t, strings.HasSuffix(result, "caller.go"),
		"Result should end with caller.go, got: %s", result)
	assert.False(t, strings.HasPrefix(result, "/"),
		"Result should not be absolute path, got: %s", result)
}

func TestCropFilename_NoMatchInFilePath(t *testing.T) {

	file := "/completely/different/path/file.go"
	functionName := "github.com/some/other/package.Function"

	result := caller.CropFilename(file, functionName)

	assert.Equal(t, "file.go", result,
		"Should return base filename when no match found")
}

func TestCropFilename_PartialMatch(t *testing.T) {

	file := "/home/user/go/src/internal/caller/caller.go"
	functionName := "piko.sh/piko/internal/caller.Caller"

	result := caller.CropFilename(file, functionName)

	assert.Contains(t, result, "caller.go",
		"Result should contain caller.go")
}

func TestConcurrentCacheAccess(t *testing.T) {
	const goroutines = 100
	const iterations = 100

	start := make(chan struct{})
	done := make(chan struct{}, goroutines)

	for range goroutines {
		go func() {
			<-start
			for range iterations {
				pc := caller.Caller(0)
				name, file, line := pc.NameFileLine()
				frame := pc.FormattedFrame()

				assert.NotEmpty(t, name)
				assert.NotEmpty(t, file)
				assert.Greater(t, line, 0)
				assert.NotEmpty(t, frame)
			}
			done <- struct{}{}
		}()
	}

	close(start)

	for range goroutines {
		<-done
	}
}

func TestConcurrentDifferentPCs(t *testing.T) {
	const goroutines = 50

	start := make(chan struct{})
	results := make(chan string, goroutines)

	helpers := []func() caller.PC{
		func() caller.PC { return caller.Caller(0) },
		func() caller.PC { return caller.Caller(0) },
		func() caller.PC { return caller.Caller(0) },
		func() caller.PC { return caller.Caller(0) },
		func() caller.PC { return caller.Caller(0) },
	}

	for i := range goroutines {
		go func(index int) {
			<-start
			pc := helpers[index%len(helpers)]()
			frame := pc.FormattedFrame()
			results <- frame
		}(i)
	}

	close(start)

	for range goroutines {
		frame := <-results
		assert.NotEmpty(t, frame)
		assert.True(t, strings.HasPrefix(frame, "\t"))
	}
}

func BenchmarkCaller(b *testing.B) {
	b.ReportAllocs()
	var pc caller.PC
	for b.Loop() {
		pc = caller.Caller(0)
	}
	_ = pc
}

var sinkPCs caller.PCs

func BenchmarkCallers(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		sinkPCs = caller.Callers(0, 10)
	}
}

func BenchmarkCallersFill(b *testing.B) {
	b.ReportAllocs()
	buffer := make(caller.PCs, 10)
	for b.Loop() {
		_ = caller.CallersFill(0, buffer)
	}
}

func BenchmarkNameFileLine_Cached(b *testing.B) {
	b.ReportAllocs()
	pc := caller.Caller(0)

	pc.NameFileLine()

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = pc.NameFileLine()
	}
}

func BenchmarkNameFileLine_Uncached(b *testing.B) {
	b.ReportAllocs()

	pcs := make([]caller.PC, 1000)
	for i := range pcs {
		pcs[i] = caller.Caller(i % 5)
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		_, _, _ = pcs[i%len(pcs)].NameFileLine()
		i++
	}
}

func BenchmarkRuntimeCaller(b *testing.B) {
	b.ReportAllocs()
	var pc uintptr
	for b.Loop() {
		pc, _, _, _ = runtime.Caller(0)
	}
	_ = pc
}

func BenchmarkRuntimeCallers(b *testing.B) {
	b.ReportAllocs()
	pcs := make([]uintptr, 10)
	for b.Loop() {
		runtime.Callers(0, pcs)
	}
}

var sinkFrame string

func BenchmarkFormattedFrame_Cached(b *testing.B) {
	b.ReportAllocs()
	pc := caller.Caller(0)

	pc.FormattedFrame()

	b.ResetTimer()
	for b.Loop() {
		sinkFrame = pc.FormattedFrame()
	}
}

func BenchmarkFormattedFrame_Uncached(b *testing.B) {
	b.ReportAllocs()

	pcs := make([]caller.PC, 1000)
	for i := range pcs {
		pcs[i] = caller.Caller(i % 5)
	}

	b.ResetTimer()
	i := 0
	for b.Loop() {
		sinkFrame = pcs[i%len(pcs)].FormattedFrame()
		i++
	}
}
