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
	"strings"
	"testing"

	"piko.sh/piko/internal/email/email_dto"
)

func TestBuildMIMEMessage_PlainTextOnly(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"user@example.com"},
		Subject:   "plain test",
		BodyPlain: "Hello, world!",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "text/plain") {
		t.Error("expected text/plain content type")
	}
	if !strings.Contains(message, "Hello, world!") {
		t.Error("expected plain text body in output")
	}
}

func TestBuildMIMEMessage_HTMLOnly(t *testing.T) {
	params := &email_dto.SendParams{
		To:       []string{"user@example.com"},
		Subject:  "html test",
		BodyHTML: "<p>Hello</p>",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "text/html") {
		t.Error("expected text/html content type")
	}
}

func TestBuildMIMEMessage_Multipart(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"user@example.com"},
		Subject:   "multipart test",
		BodyPlain: "plain version",
		BodyHTML:  "<p>html version</p>",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "multipart/alternative") {
		t.Error("expected multipart/alternative for both HTML and plain text")
	}
}

func TestBuildMIMEMessage_WithFrom(t *testing.T) {
	params := &email_dto.SendParams{
		From:      new("custom@example.com"),
		To:        []string{"user@example.com"},
		Subject:   "from test",
		BodyPlain: "body",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "custom@example.com") {
		t.Error("expected custom From address in headers")
	}
}

func TestBuildMIMEMessage_DefaultFrom(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"user@example.com"},
		Subject:   "default from test",
		BodyPlain: "body",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, defaultFromAddress) {
		t.Errorf("expected default From address %q in headers", defaultFromAddress)
	}
}

func TestBuildMIMEMessage_WithCcBcc(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"to@example.com"},
		Cc:        []string{"cc@example.com"},
		Bcc:       []string{"bcc@example.com"},
		Subject:   "cc/bcc test",
		BodyPlain: "body",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "cc@example.com") {
		t.Error("expected Cc address in headers")
	}

}

func TestBuildMIMEMessage_WithAttachment(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"user@example.com"},
		Subject:   "attachment test",
		BodyPlain: "see attached",
		Attachments: []email_dto.Attachment{
			{Filename: "report.pdf", MIMEType: "application/pdf", Content: []byte("PDF content")},
		},
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "report.pdf") {
		t.Error("expected attachment filename in MIME message")
	}
}

func TestBuildMIMEMessage_WithInlineImage(t *testing.T) {
	params := &email_dto.SendParams{
		To:       []string{"user@example.com"},
		Subject:  "inline image test",
		BodyHTML: `<img src="cid:logo">`,
		Attachments: []email_dto.Attachment{
			{Filename: "logo.png", MIMEType: "image/png", ContentID: "logo", Content: []byte("PNG data")},
		},
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "logo") {
		t.Error("expected inline image reference in MIME message")
	}
}

func TestBuildMIMEMessage_InvalidAddress(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"not-an-email"},
		Subject:   "invalid test",
		BodyPlain: "body",
	}
	_, err := BuildMIMEMessage(params)
	if err == nil {
		t.Error("expected error for invalid email address")
	}
}

func TestBuildMIMEMessage_Subject(t *testing.T) {
	params := &email_dto.SendParams{
		To:        []string{"user@example.com"},
		Subject:   "Important Subject",
		BodyPlain: "body",
	}
	data, err := BuildMIMEMessage(params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	message := string(data)
	if !strings.Contains(message, "Important Subject") {
		t.Error("expected Subject in message headers")
	}
}
