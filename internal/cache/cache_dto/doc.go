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

// Package cache_dto defines data transfer objects for the cache hexagon.
//
// It covers configuration options, cache entry representations, search
// schemas, statistics tracking, and value transformation configuration.
//
// # Loaders
//
// The package provides both single-key and bulk loading abstractions:
//
//	loader := cache_dto.LoaderFunc[string, User](
//	    func(ctx context.Context, id string) (User, error) {
//	        return db.GetUser(ctx, id)
//	    },
//	)
//
//	bulkLoader := cache_dto.BulkLoaderFunc[string, User](
//	    func(ctx context.Context, ids []string) (map[string]User, error) {
//	        return db.GetUsers(ctx, ids)
//	    },
//	)
//
// # Search
//
// Define searchable fields and query with filters:
//
//	schema := cache_dto.NewSearchSchema(
//	    cache_dto.TextField("name"),
//	    cache_dto.TagField("category"),
//	    cache_dto.SortableNumericField("price"),
//	)
package cache_dto
