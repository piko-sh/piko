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

//go:build bench

package generator_helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func BenchmarkClassesFromString_Simple(b *testing.B) {
	input := "btn btn-primary"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = ClassesFromString(input)
	}
}

func BenchmarkClassesFromString_WithDuplicates(b *testing.B) {
	input := "btn btn-primary btn btn-secondary btn-primary"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = ClassesFromString(input)
	}
}

func BenchmarkClassesFromString_Long(b *testing.B) {

	input := "flex items-center justify-between p-4 bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 text-gray-800 font-medium"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = ClassesFromString(input)
	}
}

func BenchmarkClassesFromString_Empty(b *testing.B) {
	input := ""
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = ClassesFromString(input)
	}
}

func BenchmarkClassesFromString_Parallel(b *testing.B) {
	input := "btn btn-primary active"
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = ClassesFromString(input)
		}
	})
}

func BenchmarkBuildClassString_Multiple(b *testing.B) {
	inputs := []string{"btn", "btn-primary", "active"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = buildClassString(inputs...)
	}
}

func TestClassesFromStringBytes_Simple(t *testing.T) {
	t.Parallel()

	bufferPointer := ClassesFromStringBytes("btn btn-primary")
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary", string(*bufferPointer))
}

func TestClassesFromStringBytes_WithDuplicates(t *testing.T) {
	t.Parallel()

	bufferPointer := ClassesFromStringBytes("btn btn-primary btn btn-secondary btn-primary")
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary btn-secondary", string(*bufferPointer))
}

func TestClassesFromStringBytes_Empty(t *testing.T) {
	t.Parallel()

	bufferPointer := ClassesFromStringBytes("")
	assert.Nil(t, bufferPointer)
}

func TestClassesFromSliceBytes_Simple(t *testing.T) {
	t.Parallel()

	bufferPointer := ClassesFromSliceBytes([]string{"btn", "btn-primary"})
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary", string(*bufferPointer))
}

func TestClassesFromSliceBytes_Empty(t *testing.T) {
	t.Parallel()

	bufferPointer := ClassesFromSliceBytes([]string{})
	assert.Nil(t, bufferPointer)
}

func TestMergeClassesBytes_Simple(t *testing.T) {
	t.Parallel()

	bufferPointer := MergeClassesBytes("btn", "btn-primary")
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary", string(*bufferPointer))
}

func TestMergeClassesBytes_Empty(t *testing.T) {
	t.Parallel()

	bufferPointer := MergeClassesBytes()
	assert.Nil(t, bufferPointer)
}

func BenchmarkClassesFromStringBytes_Simple(b *testing.B) {
	input := "btn btn-primary"
	warmupBytePool()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := ClassesFromStringBytes(input)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkClassesFromStringBytes_Long(b *testing.B) {
	input := "flex items-center justify-between p-4 bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 text-gray-800 font-medium"
	warmupBytePool()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := ClassesFromStringBytes(input)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkClassesFromStringBytes_Parallel(b *testing.B) {
	input := "btn btn-primary active"
	warmupBytePool()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bufferPointer := ClassesFromStringBytes(input)
			if bufferPointer != nil {
				ast_domain.PutByteBuf(bufferPointer)
			}
		}
	})
}

func warmupBytePool() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func TestBuildClassBytesV_Simple(t *testing.T) {
	t.Parallel()

	bufferPointer := BuildClassBytesV("btn", " ", "btn-primary")
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary", string(*bufferPointer))
}

func TestBuildClassBytesV_WithDuplicates(t *testing.T) {
	t.Parallel()

	bufferPointer := BuildClassBytesV("btn", " ", "btn-primary", " ", "btn")
	require.NotNil(t, bufferPointer)
	defer ast_domain.PutByteBuf(bufferPointer)

	assert.Equal(t, "btn btn-primary", string(*bufferPointer))
}

func TestBuildClassBytesV_Empty(t *testing.T) {
	t.Parallel()

	bufferPointer := BuildClassBytesV()
	assert.Nil(t, bufferPointer)
}

func BenchmarkConcatenation_ClassesFromStringBytes(b *testing.B) {
	warmupBytePool()
	propsSize := "large"
	propsColour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {

		bufferPointer := ClassesFromStringBytes("badge " + propsSize + " " + propsColour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkVariadic_BuildClassBytesV(b *testing.B) {
	warmupBytePool()
	propsSize := "large"
	propsColour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {

		bufferPointer := BuildClassBytesV("badge ", propsSize, " ", propsColour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkConcatenation_LongString(b *testing.B) {
	warmupBytePool()
	theme := "dark"
	status := "active"
	size := "lg"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := ClassesFromStringBytes("card " + theme + " status--" + status + " size-" + size)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkVariadic_LongString(b *testing.B) {
	warmupBytePool()
	theme := "dark"
	status := "active"
	size := "lg"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := BuildClassBytesV("card ", theme, " status--", status, " size-", size)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}
