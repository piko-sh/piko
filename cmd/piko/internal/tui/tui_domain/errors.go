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

package tui_domain

import "errors"

var (
	// ErrProviderNotReady indicates the provider is not yet connected or initialised.
	ErrProviderNotReady = errors.New("provider not ready")

	// ErrProviderClosed is returned when an operation is attempted on a closed
	// provider.
	ErrProviderClosed = errors.New("provider closed")

	// ErrResourceNotFound is returned when the requested resource does not exist.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrTraceNotFound indicates the requested trace does not exist.
	ErrTraceNotFound = errors.New("trace not found")

	// ErrMetricNotFound is returned when the requested metric does not exist.
	ErrMetricNotFound = errors.New("metric not found")

	// ErrInvalidResourceKind is returned when a resource kind is not recognised.
	ErrInvalidResourceKind = errors.New("invalid resource kind")

	// ErrConnectionFailed is returned when a connection to a data source fails.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrRefreshFailed is returned when a data refresh operation fails.
	ErrRefreshFailed = errors.New("refresh failed")

	// ErrPanelNotFound is returned when the requested panel does not exist.
	ErrPanelNotFound = errors.New("panel not found")

	// ErrNoProviders is returned when no providers have been set up.
	ErrNoProviders = errors.New("no providers configured")
)
