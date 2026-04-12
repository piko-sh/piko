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

// Package analytics_collector_stdout provides an analytics collector
// that prints events to the structured logger.
//
// This collector is intended for development and debugging. Events
// are logged at INFO level with all fields visible, making it easy
// to verify that analytics events fire correctly without requiring
// an external service.
//
// # Usage
//
//	server := piko.New(
//	    piko.WithBackendAnalytics(
//	        analytics_collector_stdout.NewCollector(),
//	    ),
//	)
package analytics_collector_stdout
