// Copyright 2026 PolitePixels Limited
// Benchmark to verify allocation behaviour for ActionArgument slice vs fixed-arity functions

//go:build bench

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func warmupActionPools() {
	buffers := make([]*[]byte, 100)
	for i := range buffers {
		buffers[i] = ast_domain.GetByteBuf()
	}
	for _, buffer := range buffers {
		ast_domain.PutByteBuf(buffer)
	}
}

func largeFunctionSliceArgs(functionName string, argValue any) *[]byte {

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

	return EncodeActionPayloadBytes(templater_dto.ActionPayload{
		Function: functionName,
		Args: []templater_dto.ActionArgument{
			{Type: "e"},
			{Type: "s", Value: argValue},
		},
	})
}

func largeFunctionFixedArgs2(functionName string, argValue any) *[]byte {

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

	return EncodeActionPayloadBytes2(functionName,
		templater_dto.ActionArgument{Type: "e"},
		templater_dto.ActionArgument{Type: "s", Value: argValue},
	)
}

func largeFunctionSliceArgs0(functionName string) *[]byte {
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

	return EncodeActionPayloadBytes(templater_dto.ActionPayload{
		Function: functionName,
		Args:     []templater_dto.ActionArgument{},
	})
}

func largeFunctionFixedArgs0(functionName string) *[]byte {
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

	return EncodeActionPayloadBytes0(functionName)
}

func BenchmarkLargeFunc_SliceArgs(b *testing.B) {
	warmupActionPools()
	argValue := "test"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionSliceArgs("handleClick", argValue)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_FixedArgs(b *testing.B) {
	warmupActionPools()
	argValue := "test"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionFixedArgs2("handleClick", argValue)
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_SliceArgs0(b *testing.B) {
	warmupActionPools()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionSliceArgs0("handleClick")
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}

func BenchmarkLargeFunc_FixedArgs0(b *testing.B) {
	warmupActionPools()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bufferPointer := largeFunctionFixedArgs0("handleClick")
		if bufferPointer != nil {
			ast_domain.PutByteBuf(bufferPointer)
		}
	}
}
