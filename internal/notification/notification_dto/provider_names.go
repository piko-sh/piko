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

package notification_dto

const (
	// NotificationNameDefault is the key for the default notification provider.
	NotificationNameDefault = "default"

	// NotificationNameSlack identifies the Slack notification provider.
	NotificationNameSlack = "slack"

	// NotificationNameDiscord identifies the Discord notification provider.
	NotificationNameDiscord = "discord"

	// NotificationNamePagerDuty identifies the PagerDuty notification provider.
	NotificationNamePagerDuty = "pagerduty"

	// NotificationNameTeams identifies the Microsoft Teams notification provider.
	NotificationNameTeams = "teams"

	// NotificationNameGoogleChat identifies the Google Chat notification provider.
	NotificationNameGoogleChat = "googlechat"

	// NotificationNameNtfy identifies the Ntfy notification provider.
	NotificationNameNtfy = "ntfy"

	// NotificationNameWebhook identifies the generic webhook notification provider.
	NotificationNameWebhook = "webhook"

	// NotificationNameStdout identifies the stdout notification provider,
	// used for development and testing.
	NotificationNameStdout = "stdout"
)
