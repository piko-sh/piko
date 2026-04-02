// Copyright 2026 PolitePixels Limited
// Benchmark to verify allocation behaviour for style string building

//go:build bench

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
)

func warmupStylePools() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func largeFunctionStyleConcat(colour string) *[]byte {

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

	return StylesFromStringBytes("--gradient-colour: var(" + colour + ")")
}

func largeFunctionStyleFixed(colour string) *[]byte {

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

	return BuildStyleStringBytes3("--gradient-colour: var(", colour, ")")
}

func largeFunctionStyleMultiProp(prop1, prop2 string) *[]byte {
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

	return StylesFromStringBytes("color: " + prop1 + "; background: " + prop2)
}

func largeFunctionStyleMultiPropFixed(prop1, prop2 string) *[]byte {
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

	return BuildStyleStringBytesV("color: ", prop1, "; background: ", prop2)
}

func BenchmarkLargeFunc_StyleConcat(b *testing.B) {
	warmupStylePools()
	colour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionStyleConcat(colour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_StyleFixed(b *testing.B) {
	warmupStylePools()
	colour := "blue"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionStyleFixed(colour)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_StyleMultiProp(b *testing.B) {
	warmupStylePools()
	prop1 := "red"
	prop2 := "white"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionStyleMultiProp(prop1, prop2)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_StyleMultiPropFixed(b *testing.B) {
	warmupStylePools()
	prop1 := "red"
	prop2 := "white"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionStyleMultiPropFixed(prop1, prop2)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}
