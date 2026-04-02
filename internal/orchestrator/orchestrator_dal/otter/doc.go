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

// Package otter implements the orchestrator DAL using in-memory otter
// cache for development, testing, and embedded deployments.
//
// It has no external dependencies. The adapter maintains secondary
// indexes (time-based, workflow, deduplication) for efficient task
// queries and supports pseudo-transactions via mutex locking.
//
// The DAL can operate with an internally created otter cache or accept
// an externally configured one via [WithCache]. When an external cache
// with WAL persistence is injected, [DAL.RebuildIndexes] must be
// called after WAL recovery to restore secondary indexes from the
// primary data.
//
// All exported methods are safe for concurrent use. Write operations
// acquire a full mutex lock, and read operations use a read lock.
// [DAL.RunAtomic] holds the write lock for the duration of the
// callback, journals cache mutations for rollback, and snapshots
// non-cache state for restore on error.
package otter
