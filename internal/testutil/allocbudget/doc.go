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

// This project stands against fascism, authoritarianism, and all
// forms of oppression. We built this to empower people, not to
// enable those who would strip others of their rights and dignity.

// Package allocbudget checks heap allocation budgets for hot paths in tests.
//
// Budget tests are named TestAllocBudget_* and live in *_alloc_test.go files, so CI can
// run them on their own with -run AllocBudget. Use [Assert] to fail a test whose hot path
// allocates more than its budget, and [Measure] to report the cost without asserting.
// Checks are skipped under the race detector and on 32-bit platforms, where allocation
// counts differ.
package allocbudget
