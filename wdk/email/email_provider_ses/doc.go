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

// Package email_provider_ses implements the email provider port
// using Amazon SES.
//
// The adapter sends emails through the AWS Simple Email Service
// API, supporting both simple and raw (MIME) message formats with
// built-in rate limiting to respect SES quotas. AWS credentials
// follow the standard SDK default chain unless overridden in
// configuration.
//
// # Thread safety
//
// All methods are safe for concurrent use.
package email_provider_ses
