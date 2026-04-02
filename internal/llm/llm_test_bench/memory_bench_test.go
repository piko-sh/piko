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

package llm_test_bench

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"piko.sh/piko/internal/llm/llm_adapters/memory_memory"
	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

func seedMemoryStore(b *testing.B, store *memory_memory.Store, convID string, msgCount, msgLen int) {
	b.Helper()
	ctx := context.Background()

	state := llm_dto.NewConversationState(convID)
	for i := range msgCount {
		role := llm_dto.RoleUser
		if i%2 == 1 {
			role = llm_dto.RoleAssistant
		}
		state.AddMessage(llm_dto.Message{
			Role:    role,
			Content: fmt.Sprintf("Message %d: %s", i, strings.Repeat("x", msgLen)),
		})
	}
	if err := store.Save(ctx, state); err != nil {
		b.Fatalf("seeding memory store: %v", err)
	}
}

func benchmarkMemoryLoadSave(b *testing.B, msgCount int) {
	b.Helper()

	store := memory_memory.New()
	convID := fmt.Sprintf("bench-conv-%d", msgCount)
	seedMemoryStore(b, store, convID, msgCount, 100)

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		state, err := store.Load(ctx, convID)
		if err != nil {
			b.Fatalf("load failed: %v", err)
		}
		state.AddMessage(llm_dto.Message{
			Role:    llm_dto.RoleUser,
			Content: "New message",
		})
		if err := store.Save(ctx, state); err != nil {
			b.Fatalf("save failed: %v", err)
		}
	}
}

func BenchmarkMemoryStore_LoadSave_10msgs(b *testing.B) {
	benchmarkMemoryLoadSave(b, 10)
}

func BenchmarkMemoryStore_LoadSave_100msgs(b *testing.B) {
	benchmarkMemoryLoadSave(b, 100)
}

func BenchmarkMemoryStore_LoadSave_1000msgs(b *testing.B) {
	benchmarkMemoryLoadSave(b, 1000)
}

func BenchmarkBufferMemory_AddMessage(b *testing.B) {
	store := memory_memory.New()
	const bufferSize = 20
	memory := llm_domain.NewBufferMemory(store, llm_domain.WithBufferSize(bufferSize))
	ctx := context.Background()

	for i := range bufferSize {
		role := llm_dto.RoleUser
		if i%2 == 1 {
			role = llm_dto.RoleAssistant
		}
		if err := memory.AddMessage(ctx, "bench-buffer", llm_dto.Message{
			Role:    role,
			Content: fmt.Sprintf("Pre-fill message %d", i),
		}); err != nil {
			b.Fatalf("pre-fill failed: %v", err)
		}
	}

	message := llm_dto.Message{
		Role:    llm_dto.RoleUser,
		Content: "Benchmark message at capacity",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := memory.AddMessage(ctx, "bench-buffer", message); err != nil {
			b.Fatalf("add message failed: %v", err)
		}
	}
}

func BenchmarkWindowMemory_AddMessage(b *testing.B) {
	store := memory_memory.New()
	const tokenLimit = 500
	memory := llm_domain.NewWindowMemory(store, llm_domain.WithTokenLimit(tokenLimit))
	ctx := context.Background()

	for i := range 17 {
		role := llm_dto.RoleUser
		if i%2 == 1 {
			role = llm_dto.RoleAssistant
		}
		if err := memory.AddMessage(ctx, "bench-window", llm_dto.Message{
			Role:    role,
			Content: strings.Repeat("x", 100),
		}); err != nil {
			b.Fatalf("pre-fill failed: %v", err)
		}
	}

	message := llm_dto.Message{
		Role:    llm_dto.RoleUser,
		Content: strings.Repeat("y", 100),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if err := memory.AddMessage(ctx, "bench-window", message); err != nil {
			b.Fatalf("add message failed: %v", err)
		}
	}
}

func BenchmarkMemoryStore_DeepCopy(b *testing.B) {
	store := memory_memory.New()
	convID := "bench-deepcopy"
	seedMemoryStore(b, store, convID, 100, 200)

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {

		_, err := store.Load(ctx, convID)
		if err != nil {
			b.Fatalf("load failed: %v", err)
		}
	}
}
