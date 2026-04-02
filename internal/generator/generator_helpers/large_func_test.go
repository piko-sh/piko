// Copyright 2026 PolitePixels Limited
// Test to verify allocation behaviour in large functions

//go:build bench

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func largeFunctionWithVariadic(size, colour string) *[]byte {

	var v1, v2, v3, v4, v5, v6, v7, v8, v9, v10 int
	var v11, v12, v13, v14, v15, v16, v17, v18, v19, v20 int
	var v21, v22, v23, v24, v25, v26, v27, v28, v29, v30 int
	var v31, v32, v33, v34, v35, v36, v37, v38, v39, v40 int
	var v41, v42, v43, v44, v45, v46, v47, v48, v49, v50 int

	_ = v1 + v2 + v3 + v4 + v5 + v6 + v7 + v8 + v9 + v10
	_ = v11 + v12 + v13 + v14 + v15 + v16 + v17 + v18 + v19 + v20
	_ = v21 + v22 + v23 + v24 + v25 + v26 + v27 + v28 + v29 + v30
	_ = v31 + v32 + v33 + v34 + v35 + v36 + v37 + v38 + v39 + v40
	_ = v41 + v42 + v43 + v44 + v45 + v46 + v47 + v48 + v49 + v50

	return BuildClassBytesV("badge ", size, " ", colour)
}

func largeFunctionWithFixed(size, colour string) *[]byte {

	var v1, v2, v3, v4, v5, v6, v7, v8, v9, v10 int
	var v11, v12, v13, v14, v15, v16, v17, v18, v19, v20 int
	var v21, v22, v23, v24, v25, v26, v27, v28, v29, v30 int
	var v31, v32, v33, v34, v35, v36, v37, v38, v39, v40 int
	var v41, v42, v43, v44, v45, v46, v47, v48, v49, v50 int

	_ = v1 + v2 + v3 + v4 + v5 + v6 + v7 + v8 + v9 + v10
	_ = v11 + v12 + v13 + v14 + v15 + v16 + v17 + v18 + v19 + v20
	_ = v21 + v22 + v23 + v24 + v25 + v26 + v27 + v28 + v29 + v30
	_ = v31 + v32 + v33 + v34 + v35 + v36 + v37 + v38 + v39 + v40
	_ = v41 + v42 + v43 + v44 + v45 + v46 + v47 + v48 + v49 + v50

	return BuildClassBytes4("badge ", size, " ", colour)
}

func BenchmarkLargeFunc_Variadic(b *testing.B) {
	warmupBytePool()
	size := "large"
	colour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionWithVariadic(size, colour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_Fixed(b *testing.B) {
	warmupBytePool()
	size := "large"
	colour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionWithFixed(size, colour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}
