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

import (
	"time"
)

// tickMessage is sent at regular intervals to trigger a data refresh.
type tickMessage struct {
	// time is when this tick event occurred.
	time time.Time
}

// dataRefreshedMessage is sent when provider data has been refreshed.
type dataRefreshedMessage struct {
	// providerName identifies which data provider completed the refresh.
	providerName string
}

// errorMessage holds an error to be sent through the message system.
type errorMessage struct {
	// err holds the error that occurred.
	err error
}

// providerStatusMessage is sent when a provider's status changes.
type providerStatusMessage struct {
	// err holds any error from the provider, or nil on success.
	err error

	// name identifies the provider whose status changed.
	name string

	// status is the current operational state of the provider.
	status ProviderStatus
}

// quitMessage signals that the user has asked to quit.
type quitMessage struct{}

// focusPanelMessage is sent to change which panel has focus.
type focusPanelMessage struct {
	// panelID is the identifier of the panel to focus.
	panelID string
}

// nextPanelMessage signals that focus should move to the next panel.
type nextPanelMessage struct{}

// previousPanelMessage signals a request to move focus to the previous panel.
type previousPanelMessage struct{}

// toggleHelpMessage is a message sent to show or hide the help panel.
type toggleHelpMessage struct{}

// forceRefreshMessage signals that data should be refreshed at once.
type forceRefreshMessage struct{}
