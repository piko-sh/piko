// Copyright 2026 PolitePixels Limited
// Benchmark to verify allocation behaviour for closure vs pre-evaluated conditionals

//go:build bench

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func warmupClosurePools() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func largeFunctionWithClosure(isActive bool) *[]byte {

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

	return MergeClassesBytes("option", func() string {
		if isActive {
			return "active"
		}
		return ""
	}())
}

func largeFunctionPreEval(isActive bool) *[]byte {

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

	var conditionalClass string
	if isActive {
		conditionalClass = "active"
	}
	return MergeClassesBytes("option", conditionalClass)
}

func largeFunctionMultipleTernaries(a, b, c bool) *[]byte {
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

	return MergeClassesBytes(
		"base",
		func() string {
			if a {
				return "a-class"
			}
			return ""
		}(),
		func() string {
			if b {
				return "b-class"
			}
			return ""
		}(),
		func() string {
			if c {
				return "c-class"
			}
			return ""
		}(),
	)
}

func largeFunctionMultipleTernariesPreEval(a, b, c bool) *[]byte {
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

	var classA, classB, classC string
	if a {
		classA = "a-class"
	}
	if b {
		classB = "b-class"
	}
	if c {
		classC = "c-class"
	}
	return MergeClassesBytes("base", classA, classB, classC)
}

func BenchmarkLargeFunc_Closure(b *testing.B) {
	warmupClosurePools()
	isActive := true
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionWithClosure(isActive)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_PreEval(b *testing.B) {
	warmupClosurePools()
	isActive := true
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionPreEval(isActive)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_MultipleClosure(b *testing.B) {
	warmupClosurePools()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionMultipleTernaries(true, false, true)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_MultiplePreEval(b *testing.B) {
	warmupClosurePools()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionMultipleTernariesPreEval(true, false, true)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}
