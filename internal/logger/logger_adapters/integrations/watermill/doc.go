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

// Package watermill bridges Piko's structured logging with
// Watermill's LoggerAdapter interface so that Watermill components
// (publishers, subscribers, routers) log through Piko's unified
// infrastructure. Watermill's Info and Debug levels map to Piko's
// Internal level; Trace maps to Piko's Trace level.
//
// # Usage
//
//	wmLogger := watermill.NewAdapter(log)
//	router, _ := message.NewRouter(message.RouterConfig{}, wmLogger)
//
// # Integration
//
// This adapter converts Watermill's LogFields to Piko's structured attributes,
// handling type conversion for common types (string, int, int64, bool, error).
// Use this when configuring any Watermill component that accepts a
// LoggerAdapter.
package watermill
