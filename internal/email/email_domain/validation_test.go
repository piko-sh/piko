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

package email_domain

import (
	"reflect"
	"strings"
	"testing"

	"piko.sh/piko/internal/email/email_dto"
)

var defaultTestConfig = ServiceConfig{
	MaxTotalRecipients:  10,
	MaxPayloadSizeBytes: 1024,
}

func Test_validateSingle(t *testing.T) {
	testCases := []struct {
		params      *email_dto.SendParams
		name        string
		errContains string
		config      ServiceConfig
		wantErr     bool
	}{
		{
			name:    "1. Success - Minimal Valid Email (Plain Text)",
			params:  &email_dto.SendParams{To: []string{"test@example.com"}, BodyPlain: "Hello"},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name:    "2. Success - Minimal Valid Email (HTML)",
			params:  &email_dto.SendParams{To: []string{"test@example.com"}, BodyHTML: "<p>Hello</p>"},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name:    "3. Success - Email with All Recipient Types",
			params:  &email_dto.SendParams{To: []string{"to@example.com"}, Cc: []string{"cc@example.com"}, Bcc: []string{"bcc@example.com"}, BodyPlain: "Hello"},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name: "4. Success - Email with Attachments (Within Limits)",
			params: &email_dto.SendParams{
				To:        []string{"test@example.com"},
				BodyPlain: "Check attachment.",
				Attachments: []email_dto.Attachment{
					{Filename: "file.txt", Content: make([]byte, 512)},
				},
			},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name:    "5. Success - Exactly at Recipient Limit",
			params:  &email_dto.SendParams{To: make([]string, 10), BodyPlain: "Hello"},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name:    "6. Success - Exactly at Payload Limit",
			params:  &email_dto.SendParams{To: []string{"test@example.com"}, BodyPlain: string(make([]byte, 1024))},
			config:  defaultTestConfig,
			wantErr: false,
		},
		{
			name:        "7. Failure - Nil SendParams",
			params:      nil,
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "params cannot be nil",
		},
		{
			name:        "8. Failure - No Body Content",
			params:      &email_dto.SendParams{To: []string{"test@example.com"}},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "either BodyPlain or BodyHTML must be provided",
		},
		{
			name:        "9. Failure - No 'To' Recipients",
			params:      &email_dto.SendParams{BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "at least one recipient in the 'To' field is required",
		},
		{
			name:        "10. Failure - Exceed Recipient Limit by One",
			params:      &email_dto.SendParams{To: make([]string, 11), BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "exceeds the limit of 10",
		},
		{
			name:        "11. Failure - Exceed Recipient Limit Across All Fields",
			params:      &email_dto.SendParams{To: make([]string, 4), Cc: make([]string, 4), Bcc: make([]string, 3), BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "total number of recipients (11)",
		},
		{
			name:        "12. Failure - Exceed Payload Limit by One Byte",
			params:      &email_dto.SendParams{To: []string{"test@example.com"}, BodyPlain: string(make([]byte, 1025))},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "exceeds the limit",
		},
		{
			name: "13. Failure - Exceed Payload Limit with Attachment",
			params: &email_dto.SendParams{
				To:        []string{"test@example.com"},
				BodyPlain: string(make([]byte, 512)),
				Attachments: []email_dto.Attachment{
					{Filename: "file.txt", Content: make([]byte, 513)},
				},
			},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "total message size",
		},
		{
			name:        "14. Failure - Invalid Email Syntax in 'To' field",
			params:      &email_dto.SendParams{To: []string{"invalid-email"}, BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "invalid email address(es) found in 'To' field",
		},
		{
			name:        "15. Failure - Invalid Email Syntax in 'Cc' field",
			params:      &email_dto.SendParams{To: []string{"test@example.com"}, Cc: []string{"test@.com"}, BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "invalid email address(es) found in 'Cc' field",
		},
		{
			name:        "16. Failure - Invalid Email Syntax in 'Bcc' field",
			params:      &email_dto.SendParams{To: []string{"test@example.com"}, Bcc: []string{"@domain.com"}, BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "invalid email address(es) found in 'Bcc' field",
		},
		{
			name:        "17. Failure - One Invalid Email Among Many Valid",
			params:      &email_dto.SendParams{To: []string{"good@one.com", "bad", "good@two.com"}, BodyPlain: "Hello"},
			config:      defaultTestConfig,
			wantErr:     true,
			errContains: "'bad'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSingle(tc.params, tc.config)

			if (err != nil) != tc.wantErr {
				t.Errorf("validateSingle() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("validateSingle() error = %q, want error to contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func Test_validateAndSplitBulk(t *testing.T) {
	testCases := []struct {
		name             string
		inputEmails      []*email_dto.SendParams
		config           ServiceConfig
		wantValidCount   int
		wantInvalidCount int
	}{
		{
			name: "18. Bulk - All Valid Emails",
			inputEmails: []*email_dto.SendParams{
				{To: []string{"a@example.com"}, BodyPlain: "1"},
				{To: []string{"b@example.com"}, BodyPlain: "2"},
			},
			config:           defaultTestConfig,
			wantValidCount:   2,
			wantInvalidCount: 0,
		},
		{
			name: "19. Bulk - All Invalid Emails",
			inputEmails: []*email_dto.SendParams{
				{To: []string{"a@example.com"}},
				{BodyPlain: "2"},
			},
			config:           defaultTestConfig,
			wantValidCount:   0,
			wantInvalidCount: 2,
		},
		{
			name: "20. Bulk - Mixed Valid and Invalid Emails",
			inputEmails: []*email_dto.SendParams{
				{To: []string{"a@example.com"}, BodyPlain: "1"},
				{To: []string{"bad-email"}, BodyPlain: "2"},
				{To: []string{"c@example.com"}, BodyPlain: "3"},
			},
			config:           defaultTestConfig,
			wantValidCount:   2,
			wantInvalidCount: 1,
		},
		{
			name:             "21. Bulk - Empty Input Slice",
			inputEmails:      []*email_dto.SendParams{},
			config:           defaultTestConfig,
			wantValidCount:   0,
			wantInvalidCount: 0,
		},
		{
			name:             "22. Bulk - Nil Input Slice",
			inputEmails:      nil,
			config:           defaultTestConfig,
			wantValidCount:   0,
			wantInvalidCount: 0,
		},
		{
			name: "23. Bulk - Slice Containing a Nil Entry",
			inputEmails: []*email_dto.SendParams{
				{To: []string{"a@example.com"}, BodyPlain: "1"},
				nil,
				{To: []string{"c@example.com"}, BodyPlain: "3"},
			},
			config:           defaultTestConfig,
			wantValidCount:   2,
			wantInvalidCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, errs := validateAndSplitBulk(tc.inputEmails, tc.config)

			if len(valid) != tc.wantValidCount {
				t.Errorf("validateAndSplitBulk() returned %d valid emails, want %d", len(valid), tc.wantValidCount)
			}

			invalidCount := 0
			if errs != nil {
				invalidCount = len(errs.Errors)
			}
			if invalidCount != tc.wantInvalidCount {
				t.Errorf("validateAndSplitBulk() returned %d invalid emails, want %d", invalidCount, tc.wantInvalidCount)
			}
		})
	}
}

func Test_sanitiseRecipients(t *testing.T) {
	testCases := []struct {
		name        string
		inputParams *email_dto.SendParams
		expectedTo  []string
		expectedCc  []string
		expectedBcc []string
	}{
		{
			name:        "24. Sanitise - No Duplicates",
			inputParams: &email_dto.SendParams{To: []string{"a@a.com"}, Cc: []string{"b@b.com"}, Bcc: []string{"c@c.com"}},
			expectedTo:  []string{"a@a.com"},
			expectedCc:  []string{"b@b.com"},
			expectedBcc: []string{"c@c.com"},
		},
		{
			name:        "25. Sanitise - Duplicate in 'To' removed from 'Cc' and 'Bcc'",
			inputParams: &email_dto.SendParams{To: []string{"a@a.com"}, Cc: []string{"b@b.com", "a@a.com"}, Bcc: []string{"a@a.com", "c@c.com"}},
			expectedTo:  []string{"a@a.com"},
			expectedCc:  []string{"b@b.com"},
			expectedBcc: []string{"c@c.com"},
		},
		{
			name:        "26. Sanitise - Duplicate in 'Cc' removed from 'Bcc'",
			inputParams: &email_dto.SendParams{To: []string{"a@a.com"}, Cc: []string{"b@b.com"}, Bcc: []string{"b@b.com", "c@c.com"}},
			expectedTo:  []string{"a@a.com"},
			expectedCc:  []string{"b@b.com"},
			expectedBcc: []string{"c@c.com"},
		},
		{
			name:        "27. Sanitise - Duplicates within a single list are removed",
			inputParams: &email_dto.SendParams{To: []string{"a@a.com", "a@a.com"}},
			expectedTo:  []string{"a@a.com"},
			expectedCc:  nil,
			expectedBcc: nil,
		},
		{
			name: "28. Sanitise - Complex case with multiple duplicates",
			inputParams: &email_dto.SendParams{
				To:  []string{"a@a.com", "b@b.com"},
				Cc:  []string{"c@c.com", "a@a.com", "d@d.com"},
				Bcc: []string{"d@d.com", "a@a.com", "e@e.com", "b@b.com"},
			},
			expectedTo:  []string{"a@a.com", "b@b.com"},
			expectedCc:  []string{"c@c.com", "d@d.com"},
			expectedBcc: []string{"e@e.com"},
		},
		{
			name:        "29. Sanitise - Nil params is a no-op",
			inputParams: nil,
			expectedTo:  nil,
			expectedCc:  nil,
			expectedBcc: nil,
		},
		{
			name:        "30. Sanitise - Empty recipient lists",
			inputParams: &email_dto.SendParams{},
			expectedTo:  []string{},
			expectedCc:  []string{},
			expectedBcc: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sanitiseRecipients(tc.inputParams)

			if tc.inputParams == nil {
				return
			}

			if !reflect.DeepEqual(tc.inputParams.To, tc.expectedTo) {
				t.Errorf("sanitiseRecipients() To = %v, want %v", tc.inputParams.To, tc.expectedTo)
			}
			if !reflect.DeepEqual(tc.inputParams.Cc, tc.expectedCc) {
				t.Errorf("sanitiseRecipients() Cc = %v, want %v", tc.inputParams.Cc, tc.expectedCc)
			}
			if !reflect.DeepEqual(tc.inputParams.Bcc, tc.expectedBcc) {
				t.Errorf("sanitiseRecipients() Bcc = %v, want %v", tc.inputParams.Bcc, tc.expectedBcc)
			}
		})
	}
}
