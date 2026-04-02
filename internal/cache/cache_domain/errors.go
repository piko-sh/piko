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

package cache_domain

import "errors"

var (
	// ErrProviderNotFound is returned when a requested cache provider has not been
	// registered.
	ErrProviderNotFound = errors.New("cache provider not found")

	// ErrSearchNotSupported is returned when a provider does not support search.
	// Providers that do not support search should wrap this error with extra
	// context, such as suggesting an alternative provider (e.g., RediSearch
	// instead of Redis).
	ErrSearchNotSupported = errors.New("search operations not supported by this provider")

	// errInvalidConfiguration is returned when the options provided to NewCache
	// are invalid.
	errInvalidConfiguration = errors.New("invalid cache configuration")

	// ErrTransactionFinalised is returned when an operation is attempted on a
	// transaction that has already been committed or rolled back.
	ErrTransactionFinalised = errors.New("transaction already finalised")

	// ErrNestedTransactionUnsupported is returned when RunAtomic is called
	// from within an existing transaction.
	ErrNestedTransactionUnsupported = errors.New("nested transactions are not supported")

	// ErrInvalidateByTagsUnsupported is returned when InvalidateByTags is
	// called within a transaction, because the cache interface does not
	// expose tag-to-key resolution for rollback journalling.
	ErrInvalidateByTagsUnsupported = errors.New("InvalidateByTags is not supported within a transaction")

	// ErrInvalidateAllUnsupported is returned when InvalidateAll is called
	// within a transaction, because bulk invalidation cannot be efficiently
	// journalled at the key level.
	ErrInvalidateAllUnsupported = errors.New("InvalidateAll is not supported within a transaction")

	errTransformerNil = errors.New("transformer cannot be nil")

	errTransformerNameEmpty = errors.New("transformer name cannot be empty")

	errEncoderNil = errors.New("encoder cannot be nil")

	errEncoderNoType = errors.New("encoder must handle a concrete type")
)
