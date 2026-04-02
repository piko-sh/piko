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

// Package memory implements [llm_domain.BudgetStorePort] using
// in-memory storage. It tracks LLM cost and usage data per scope in
// a mutex-guarded map, automatically resetting hourly and daily
// spending counters when their time windows elapse.
//
// # Usage
//
//	store := memory.New()
//	manager := llm_domain.NewBudgetManager(store, calculator)
//
// # Thread safety
//
// All exported methods on Store are safe for concurrent use. Write
// operations acquire an exclusive lock; read operations use a shared
// read lock.
package memory
