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

package email_provider_mailgun_test

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/email/email_provider_mailgun"
)

func TestMailgunProviderFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"MailgunProviderArgs": email_provider_mailgun.MailgunProviderArgs{},

		"NewMailgunProvider": email_provider_mailgun.NewMailgunProvider,
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
