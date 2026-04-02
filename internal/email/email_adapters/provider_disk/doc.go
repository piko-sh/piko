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

// Package provider_disk implements the email provider port by writing
// emails to disk as RFC 5322 compliant .eml files suitable for
// development, testing, and auditing.
//
// The output files are standard MIME messages that can be opened by
// any email client. Emails are written atomically using a temporary
// file with rename. Filenames follow the pattern
// {timestamp}_{sanitised_recipient}.eml.
//
// All methods are safe for concurrent use.
package provider_disk
