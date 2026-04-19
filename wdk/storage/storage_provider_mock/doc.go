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

// Package storage_provider_mock provides an in-memory mock implementation
// of the storage provider interface for testing and development.
//
// Allows you to exercise storage operations without requiring any external
// services, filesystem access, or cloud credentials. All data is held in memory
// and discarded when the provider is garbage-collected.
//
// # Usage
//
// Create a mock provider and register it with a storage service:
//
//	mock := storage_provider_mock.NewMockProvider()
//	service := storage.NewService("mock")
//	service.RegisterProvider(ctx, "mock", mock)
//
// The returned provider implements [storage.ProviderPort] and can be used
// anywhere a real storage provider is expected. It is designed for unit tests
// and local development.
package storage_provider_mock
