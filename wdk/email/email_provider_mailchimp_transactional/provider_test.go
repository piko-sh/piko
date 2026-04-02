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

package email_provider_mailchimp_transactional

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"piko.sh/piko/internal/email/email_domain"
	"piko.sh/piko/internal/email/email_dto"
)

func newTestProvider(t *testing.T, serverURL string) *MailchimpTransactionalProvider {
	t.Helper()

	provider, err := NewMailchimpTransactionalProvider(context.Background(), MailchimpTransactionalProviderArgs{
		APIKey:    "test-api-key",
		FromEmail: "sender@example.com",
	})
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	p, ok := provider.(*MailchimpTransactionalProvider)
	if !ok {
		t.Fatal("expected *MailchimpTransactionalProvider")
	}
	p.baseURL = serverURL

	return p
}

func newTestServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	}))
}

func TestNewMailchimpTransactionalProvider_EmptyAPIKey(t *testing.T) {
	_, err := NewMailchimpTransactionalProvider(context.Background(), MailchimpTransactionalProviderArgs{
		APIKey:    "",
		FromEmail: "sender@example.com",
	})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if err.Error() != "mailchimp transactional API key must not be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewMailchimpTransactionalProvider_EmptyFromEmail(t *testing.T) {
	_, err := NewMailchimpTransactionalProvider(context.Background(), MailchimpTransactionalProviderArgs{
		APIKey:    "test-key",
		FromEmail: "",
	})
	if err == nil {
		t.Fatal("expected error for empty from email")
	}
	if err.Error() != "mailchimp transactional from email must not be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewMailchimpTransactionalProvider_Success(t *testing.T) {
	provider, err := NewMailchimpTransactionalProvider(context.Background(), MailchimpTransactionalProviderArgs{
		APIKey:    "test-key",
		FromEmail: "sender@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestValidateSendParams_NoRecipients(t *testing.T) {
	err := validateSendParams(&email_dto.SendParams{
		BodyHTML: "<p>Hello</p>",
	})
	if !errors.Is(err, email_domain.ErrRecipientRequired) {
		t.Fatalf("expected ErrRecipientRequired, got: %v", err)
	}
}

func TestValidateSendParams_NoBody(t *testing.T) {
	err := validateSendParams(&email_dto.SendParams{
		To: []string{"to@example.com"},
	})
	if !errors.Is(err, email_domain.ErrBodyRequired) {
		t.Fatalf("expected ErrBodyRequired, got: %v", err)
	}
}

func TestValidateSendParams_HTMLOnly(t *testing.T) {
	err := validateSendParams(&email_dto.SendParams{
		To:       []string{"to@example.com"},
		BodyHTML: "<p>Hello</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSendParams_PlainOnly(t *testing.T) {
	err := validateSendParams(&email_dto.SendParams{
		To:        []string{"to@example.com"},
		BodyPlain: "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildRecipients_AllTypes(t *testing.T) {
	params := &email_dto.SendParams{
		To:  []string{"to1@example.com", "to2@example.com"},
		Cc:  []string{"cc@example.com"},
		Bcc: []string{"bcc@example.com"},
	}

	recipients := buildRecipients(params)

	if len(recipients) != 4 {
		t.Fatalf("expected 4 recipients, got %d", len(recipients))
	}

	if recipients[0].Email != "to1@example.com" || recipients[0].Type != "to" {
		t.Errorf("unexpected recipient[0]: %+v", recipients[0])
	}
	if recipients[1].Email != "to2@example.com" || recipients[1].Type != "to" {
		t.Errorf("unexpected recipient[1]: %+v", recipients[1])
	}

	if recipients[2].Email != "cc@example.com" || recipients[2].Type != "cc" {
		t.Errorf("unexpected recipient[2]: %+v", recipients[2])
	}

	if recipients[3].Email != "bcc@example.com" || recipients[3].Type != "bcc" {
		t.Errorf("unexpected recipient[3]: %+v", recipients[3])
	}
}

func TestBuildRecipients_ToOnly(t *testing.T) {
	params := &email_dto.SendParams{
		To: []string{"to@example.com"},
	}

	recipients := buildRecipients(params)

	if len(recipients) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(recipients))
	}
	if recipients[0].Email != "to@example.com" || recipients[0].Type != "to" {
		t.Errorf("unexpected recipient: %+v", recipients[0])
	}
}

func TestBuildAttachmentsAndImages_Empty(t *testing.T) {
	attachments, images := buildAttachmentsAndImages(nil)
	if attachments != nil {
		t.Errorf("expected nil attachments, got %v", attachments)
	}
	if images != nil {
		t.Errorf("expected nil images, got %v", images)
	}
}

func TestBuildAttachmentsAndImages_RegularAttachment(t *testing.T) {
	input := []email_dto.Attachment{
		{
			Filename: "report.pdf",
			MIMEType: "application/pdf",
			Content:  []byte("pdf-content"),
		},
	}

	attachments, images := buildAttachmentsAndImages(input)

	if len(attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(attachments))
	}
	if len(images) != 0 {
		t.Fatalf("expected 0 images, got %d", len(images))
	}

	att := attachments[0]
	if att.Name != "report.pdf" {
		t.Errorf("expected name 'report.pdf', got '%s'", att.Name)
	}
	if att.Type != "application/pdf" {
		t.Errorf("expected type 'application/pdf', got '%s'", att.Type)
	}

	expectedContent := base64.StdEncoding.EncodeToString([]byte("pdf-content"))
	if att.Content != expectedContent {
		t.Errorf("expected base64 content '%s', got '%s'", expectedContent, att.Content)
	}
}

func TestBuildAttachmentsAndImages_InlineImage(t *testing.T) {
	input := []email_dto.Attachment{
		{
			Filename:  "logo.png",
			MIMEType:  "image/png",
			ContentID: "logo-cid",
			Content:   []byte("png-content"),
		},
	}

	attachments, images := buildAttachmentsAndImages(input)

	if len(attachments) != 0 {
		t.Fatalf("expected 0 attachments, got %d", len(attachments))
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	img := images[0]
	if img.Name != "logo-cid" {
		t.Errorf("expected name 'logo-cid', got '%s'", img.Name)
	}
	if img.Type != "image/png" {
		t.Errorf("expected type 'image/png', got '%s'", img.Type)
	}
}

func TestBuildAttachmentsAndImages_Mixed(t *testing.T) {
	input := []email_dto.Attachment{
		{
			Filename: "doc.pdf",
			MIMEType: "application/pdf",
			Content:  []byte("pdf"),
		},
		{
			Filename:  "banner.jpg",
			MIMEType:  "image/jpeg",
			ContentID: "banner",
			Content:   []byte("jpg"),
		},
		{
			Filename: "data.csv",
			MIMEType: "text/csv",
			Content:  []byte("csv"),
		},
	}

	attachments, images := buildAttachmentsAndImages(input)

	if len(attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(attachments))
	}
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}
}

func TestApplyProviderOptions_Nil(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	applyProviderOptions(message, request, nil)

	if message.Tags != nil {
		t.Error("expected nil tags")
	}
}

func TestApplyProviderOptions_Tags(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"tags": []string{"tag1", "tag2"},
	}

	applyProviderOptions(message, request, options)

	if len(message.Tags) != 2 || message.Tags[0] != "tag1" || message.Tags[1] != "tag2" {
		t.Errorf("unexpected tags: %v", message.Tags)
	}
}

func TestApplyProviderOptions_TrackOpens(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"track_opens": true,
	}

	applyProviderOptions(message, request, options)

	if message.TrackOpens == nil || !*message.TrackOpens {
		t.Error("expected track_opens to be true")
	}
}

func TestApplyProviderOptions_TrackClicks(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"track_clicks": false,
	}

	applyProviderOptions(message, request, options)

	if message.TrackClicks == nil || *message.TrackClicks {
		t.Error("expected track_clicks to be false")
	}
}

func TestApplyProviderOptions_Metadata(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"metadata": map[string]string{"order_id": "123"},
	}

	applyProviderOptions(message, request, options)

	if message.Metadata["order_id"] != "123" {
		t.Errorf("unexpected metadata: %v", message.Metadata)
	}
}

func TestApplyProviderOptions_Important(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"important": true,
	}

	applyProviderOptions(message, request, options)

	if message.Important == nil || !*message.Important {
		t.Error("expected important to be true")
	}
}

func TestApplyProviderOptions_AutoText(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"auto_text": true,
	}

	applyProviderOptions(message, request, options)

	if message.AutoText == nil || !*message.AutoText {
		t.Error("expected auto_text to be true")
	}
}

func TestApplyProviderOptions_InlineCSS(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"inline_css": true,
	}

	applyProviderOptions(message, request, options)

	if message.InlineCSS == nil || !*message.InlineCSS {
		t.Error("expected inline_css to be true")
	}
}

func TestApplyProviderOptions_Headers(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"headers": map[string]string{"X-Custom": "value"},
	}

	applyProviderOptions(message, request, options)

	if message.Headers["X-Custom"] != "value" {
		t.Errorf("unexpected headers: %v", message.Headers)
	}
}

func TestApplyProviderOptions_Async(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"async": true,
	}

	applyProviderOptions(message, request, options)

	if !request.Async {
		t.Error("expected async to be true")
	}
}

func TestApplyProviderOptions_UnknownKeyIgnored(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"unknown_option": "value",
	}

	applyProviderOptions(message, request, options)

}

func TestApplyProviderOptions_WrongTypeIgnored(t *testing.T) {
	message := &mandrillMessage{}
	request := &mandrillSendRequest{}
	options := map[string]any{
		"tags":        "not-a-slice",
		"track_opens": "not-a-bool",
		"metadata":    "not-a-map",
		"async":       "not-a-bool",
	}

	applyProviderOptions(message, request, options)

	if message.Tags != nil {
		t.Error("expected nil tags for wrong type")
	}
	if message.TrackOpens != nil {
		t.Error("expected nil track_opens for wrong type")
	}
	if message.Metadata != nil {
		t.Error("expected nil metadata for wrong type")
	}
	if request.Async {
		t.Error("expected false async for wrong type")
	}
}

func TestSend_Success(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `[{"email":"to@example.com","status":"sent","_id":"msg-123"}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test Subject",
		BodyHTML: "<p>Hello</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_SuccessQueued(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `[{"email":"to@example.com","status":"queued","_id":"msg-456"}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test Subject",
		BodyHTML: "<p>Hello</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_APIError(t *testing.T) {
	server := newTestServer(t, http.StatusInternalServerError,
		`{"status":"error","code":-1,"name":"Invalid_Key","message":"Invalid API key"}`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if !contains(err.Error(), "Invalid_Key") {
		t.Errorf("expected error to contain 'Invalid_Key', got: %v", err)
	}
}

func TestSend_RecipientRejected(t *testing.T) {
	server := newTestServer(t, http.StatusOK,
		`[{"email":"bad@example.com","status":"rejected","_id":""}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"bad@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for rejected recipient")
	}
	if !contains(err.Error(), "rejected") {
		t.Errorf("expected error to contain 'rejected', got: %v", err)
	}
}

func TestSend_RecipientInvalid(t *testing.T) {
	server := newTestServer(t, http.StatusOK,
		`[{"email":"invalid@","status":"invalid","_id":""}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"invalid@"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for invalid recipient")
	}
	if !contains(err.Error(), "invalid") {
		t.Errorf("expected error to contain 'invalid', got: %v", err)
	}
}

func TestSend_MixedStatuses(t *testing.T) {
	server := newTestServer(t, http.StatusOK,
		`[{"email":"good@example.com","status":"sent","_id":"1"},{"email":"bad@example.com","status":"rejected","_id":""}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"good@example.com", "bad@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for mixed statuses with rejection")
	}
	if !contains(err.Error(), "bad@example.com") {
		t.Errorf("expected error to mention failed recipient, got: %v", err)
	}
}

func TestSend_ContextCancelled(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `[]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := provider.Send(ctx, &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSend_RequestBodyStructure(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}

		if err := json.Unmarshal(body, &receivedBody); err != nil {
			t.Errorf("failed to parse request body: %v", err)
			return
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"msg-1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		From:      new("custom@example.com"),
		To:        []string{"to@example.com"},
		Cc:        []string{"cc@example.com"},
		Bcc:       []string{"bcc@example.com"},
		Subject:   "Test Subject",
		BodyHTML:  "<p>Hello</p>",
		BodyPlain: "Hello",
		Attachments: []email_dto.Attachment{
			{
				Filename: "test.txt",
				MIMEType: "text/plain",
				Content:  []byte("file content"),
			},
		},
		ProviderOptions: map[string]any{
			"tags": []string{"test-tag"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody.Key != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got '%s'", receivedBody.Key)
	}

	if receivedBody.Message.FromEmail != "custom@example.com" {
		t.Errorf("expected from 'custom@example.com', got '%s'", receivedBody.Message.FromEmail)
	}

	if receivedBody.Message.Subject != "Test Subject" {
		t.Errorf("expected subject 'Test Subject', got '%s'", receivedBody.Message.Subject)
	}

	if receivedBody.Message.HTML != "<p>Hello</p>" {
		t.Errorf("unexpected HTML body: %s", receivedBody.Message.HTML)
	}
	if receivedBody.Message.Text != "Hello" {
		t.Errorf("unexpected text body: %s", receivedBody.Message.Text)
	}

	if len(receivedBody.Message.To) != 3 {
		t.Fatalf("expected 3 recipients, got %d", len(receivedBody.Message.To))
	}
	if receivedBody.Message.To[0].Type != "to" {
		t.Errorf("expected first recipient type 'to', got '%s'", receivedBody.Message.To[0].Type)
	}
	if receivedBody.Message.To[1].Type != "cc" {
		t.Errorf("expected second recipient type 'cc', got '%s'", receivedBody.Message.To[1].Type)
	}
	if receivedBody.Message.To[2].Type != "bcc" {
		t.Errorf("expected third recipient type 'bcc', got '%s'", receivedBody.Message.To[2].Type)
	}

	if len(receivedBody.Message.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(receivedBody.Message.Attachments))
	}
	expectedB64 := base64.StdEncoding.EncodeToString([]byte("file content"))
	if receivedBody.Message.Attachments[0].Content != expectedB64 {
		t.Errorf("expected base64 content '%s', got '%s'", expectedB64, receivedBody.Message.Attachments[0].Content)
	}

	if len(receivedBody.Message.Tags) != 1 || receivedBody.Message.Tags[0] != "test-tag" {
		t.Errorf("unexpected tags: %v", receivedBody.Message.Tags)
	}
}

func TestSend_DefaultFromEmail(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody.Message.FromEmail != "sender@example.com" {
		t.Errorf("expected default from email 'sender@example.com', got '%s'",
			receivedBody.Message.FromEmail)
	}
}

func TestSend_InlineImagesSeparated(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
		Attachments: []email_dto.Attachment{
			{
				Filename: "doc.pdf",
				MIMEType: "application/pdf",
				Content:  []byte("pdf"),
			},
			{
				Filename:  "logo.png",
				MIMEType:  "image/png",
				ContentID: "logo",
				Content:   []byte("png"),
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedBody.Message.Attachments) != 1 {
		t.Errorf("expected 1 regular attachment, got %d", len(receivedBody.Message.Attachments))
	}
	if len(receivedBody.Message.Images) != 1 {
		t.Errorf("expected 1 inline image, got %d", len(receivedBody.Message.Images))
	}
	if receivedBody.Message.Images[0].Name != "logo" {
		t.Errorf("expected image name 'logo', got '%s'", receivedBody.Message.Images[0].Name)
	}
}

func TestSendBulk_Empty(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	err := provider.SendBulk(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = provider.SendBulk(context.Background(), []*email_dto.SendParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendBulk_CollectsErrors(t *testing.T) {
	server := newTestServer(t, http.StatusInternalServerError,
		`{"status":"error","code":-1,"name":"ServerError","message":"Internal error"}`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	emails := []*email_dto.SendParams{
		{To: []string{"a@example.com"}, Subject: "A", BodyHTML: "<p>A</p>"},
		{To: []string{"b@example.com"}, Subject: "B", BodyHTML: "<p>B</p>"},
	}

	err := provider.SendBulk(context.Background(), emails)
	if err == nil {
		t.Fatal("expected error for bulk send failures")
	}

	var multiErr *email_domain.MultiError
	if !errors.As(err, &multiErr) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if multiErr.Count() != 2 {
		t.Errorf("expected 2 errors, got %d", multiErr.Count())
	}
}

func TestSupportsBulkSending(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	if provider.SupportsBulkSending() {
		t.Error("expected SupportsBulkSending to return false")
	}
}

func TestClose(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	err := provider.Close(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseErrorResponse_ValidJSON(t *testing.T) {
	body := []byte(`{"status":"error","code":-1,"name":"Invalid_Key","message":"Invalid API key"}`)

	err := parseErrorResponse(body, 500)

	if !contains(err.Error(), "Invalid_Key") {
		t.Errorf("expected error to contain 'Invalid_Key', got: %v", err)
	}
	if !contains(err.Error(), "Invalid API key") {
		t.Errorf("expected error to contain 'Invalid API key', got: %v", err)
	}
}

func TestParseErrorResponse_InvalidJSON(t *testing.T) {
	body := []byte(`not valid json`)

	err := parseErrorResponse(body, 503)

	if !contains(err.Error(), "503") {
		t.Errorf("expected error to contain status code, got: %v", err)
	}
	if !contains(err.Error(), "not valid json") {
		t.Errorf("expected error to contain raw body, got: %v", err)
	}
}

func TestCheckRecipientStatuses_AllSent(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "a@example.com", Status: "sent", ID: "1"},
		{Email: "b@example.com", Status: "sent", ID: "2"},
	}

	err := checkRecipientStatuses(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRecipientStatuses_AllQueued(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "a@example.com", Status: "queued", ID: "1"},
	}

	err := checkRecipientStatuses(results)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRecipientStatuses_Rejected(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "bad@example.com", Status: "rejected"},
	}

	err := checkRecipientStatuses(results)
	if err == nil {
		t.Fatal("expected error for rejected recipient")
	}
	if !contains(err.Error(), "bad@example.com") {
		t.Errorf("expected error to mention recipient, got: %v", err)
	}
}

func TestCheckRecipientStatuses_Invalid(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "invalid@", Status: "invalid"},
	}

	err := checkRecipientStatuses(results)
	if err == nil {
		t.Fatal("expected error for invalid recipient")
	}
}

func TestSend_UsesPostMethod(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST method, got '%s'", receivedMethod)
	}
}

func TestSend_UsesCorrectEndpointPath(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPath != "/messages/send" {
		t.Errorf("expected path '/messages/send', got '%s'", receivedPath)
	}
}

func TestSend_SetsUserAgentHeader(t *testing.T) {
	var receivedUserAgent string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedUserAgent != "piko-email-provider/1.0" {
		t.Errorf("expected User-Agent 'piko-email-provider/1.0', got '%s'", receivedUserAgent)
	}
}

func TestSend_APIKeyNotInHeaders(t *testing.T) {
	var authHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if authHeader != "" {
		t.Errorf("expected no Authorization header, got '%s'", authHeader)
	}
}

func TestSend_PlainTextOnly(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:        []string{"to@example.com"},
		Subject:   "Plain email",
		BodyPlain: "Just plain text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody.Message.Text != "Just plain text" {
		t.Errorf("expected plain text body, got '%s'", receivedBody.Message.Text)
	}
	if receivedBody.Message.HTML != "" {
		t.Errorf("expected empty HTML body, got '%s'", receivedBody.Message.HTML)
	}
}

func TestSend_BothHTMLAndPlainText(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:        []string{"to@example.com"},
		Subject:   "Both bodies",
		BodyHTML:  "<p>HTML version</p>",
		BodyPlain: "Plain version",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody.Message.HTML != "<p>HTML version</p>" {
		t.Errorf("unexpected HTML body: '%s'", receivedBody.Message.HTML)
	}
	if receivedBody.Message.Text != "Plain version" {
		t.Errorf("unexpected text body: '%s'", receivedBody.Message.Text)
	}
}

func TestSend_AsyncOptionSentInRequest(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"queued","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Async test",
		BodyHTML: "<p>Async</p>",
		ProviderOptions: map[string]any{
			"async": true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !receivedBody.Async {
		t.Error("expected async=true in request body")
	}
}

func TestSend_MultipleProviderOptionsSentInRequest(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Multi-option test",
		BodyHTML: "<p>Test</p>",
		ProviderOptions: map[string]any{
			"tags":         []string{"newsletter", "weekly"},
			"track_opens":  true,
			"track_clicks": true,
			"metadata":     map[string]string{"campaign": "spring-2026"},
			"important":    true,
			"inline_css":   true,
			"headers":      map[string]string{"X-Campaign-ID": "spring-2026"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedBody.Message.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(receivedBody.Message.Tags))
	}
	if receivedBody.Message.TrackOpens == nil || !*receivedBody.Message.TrackOpens {
		t.Error("expected track_opens=true")
	}
	if receivedBody.Message.TrackClicks == nil || !*receivedBody.Message.TrackClicks {
		t.Error("expected track_clicks=true")
	}
	if receivedBody.Message.Metadata["campaign"] != "spring-2026" {
		t.Errorf("unexpected metadata: %v", receivedBody.Message.Metadata)
	}
	if receivedBody.Message.Important == nil || !*receivedBody.Message.Important {
		t.Error("expected important=true")
	}
	if receivedBody.Message.InlineCSS == nil || !*receivedBody.Message.InlineCSS {
		t.Error("expected inline_css=true")
	}
	if receivedBody.Message.Headers["X-Campaign-ID"] != "spring-2026" {
		t.Errorf("unexpected headers: %v", receivedBody.Message.Headers)
	}
}

func TestSend_NoProviderOptionsOmitsOptionalFields(t *testing.T) {
	var rawBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &rawBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Minimal",
		BodyHTML: "<p>Minimal</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msg, ok := rawBody["message"].(map[string]any)
	if !ok {
		t.Fatal("expected message object in request body")
	}

	for _, field := range []string{"tags", "track_opens", "track_clicks", "metadata", "important", "auto_text", "inline_css", "headers", "images"} {
		if _, exists := msg[field]; exists {
			t.Errorf("expected field '%s' to be omitted from request, but it was present", field)
		}
	}

	if _, exists := rawBody["async"]; exists {
		t.Error("expected 'async' to be omitted from request")
	}
}

func TestSend_HTTP400BadRequest(t *testing.T) {
	server := newTestServer(t, http.StatusBadRequest,
		`{"status":"error","code":-2,"name":"ValidationError","message":"Validation failed"}`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !contains(err.Error(), "ValidationError") {
		t.Errorf("expected error to contain 'ValidationError', got: %v", err)
	}
}

func TestSend_HTTP401Unauthorised(t *testing.T) {
	server := newTestServer(t, http.StatusUnauthorized,
		`{"status":"error","code":-1,"name":"Invalid_Key","message":"Invalid API key"}`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !contains(err.Error(), "Invalid_Key") {
		t.Errorf("expected error to contain 'Invalid_Key', got: %v", err)
	}
}

func TestSend_HTTP429RateLimited(t *testing.T) {
	server := newTestServer(t, http.StatusTooManyRequests,
		`{"status":"error","code":-3,"name":"Too_Many_Requests","message":"Rate limit exceeded"}`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !contains(err.Error(), "Too_Many_Requests") {
		t.Errorf("expected error to contain 'Too_Many_Requests', got: %v", err)
	}
}

func TestSend_MalformedResponseJSON(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `not valid json`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for malformed response")
	}
	if !contains(err.Error(), "failed to parse response") {
		t.Errorf("expected parse error, got: %v", err)
	}
}

func TestSend_EmptyResponseArray(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `[]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCheckRecipientStatuses_MultipleRejected(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "good@example.com", Status: "sent", ID: "1"},
		{Email: "bad1@example.com", Status: "rejected"},
		{Email: "bad2@example.com", Status: "invalid"},
	}

	err := checkRecipientStatuses(results)
	if err == nil {
		t.Fatal("expected error for multiple failed recipients")
	}

	errMsg := err.Error()
	if !contains(errMsg, "bad1@example.com") {
		t.Errorf("expected error to mention bad1, got: %s", errMsg)
	}
	if !contains(errMsg, "bad2@example.com") {
		t.Errorf("expected error to mention bad2, got: %s", errMsg)
	}
	if !contains(errMsg, "rejected") {
		t.Errorf("expected error to mention rejected status, got: %s", errMsg)
	}
	if !contains(errMsg, "invalid") {
		t.Errorf("expected error to mention invalid status, got: %s", errMsg)
	}
}

func TestCheckRecipientStatuses_MixedSentAndQueued(t *testing.T) {
	results := []mandrillSendResponse{
		{Email: "a@example.com", Status: "sent", ID: "1"},
		{Email: "b@example.com", Status: "queued", ID: "2"},
		{Email: "c@example.com", Status: "sent", ID: "3"},
	}

	err := checkRecipientStatuses(results)
	if err != nil {
		t.Fatalf("unexpected error for mixed sent/queued: %v", err)
	}
}

func TestSendBulk_PartialSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if callCount == 2 {

			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status":"error","code":-1,"name":"ServerError","message":"Temporary failure"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"ok@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	emails := []*email_dto.SendParams{
		{To: []string{"ok1@example.com"}, Subject: "A", BodyHTML: "<p>A</p>"},
		{To: []string{"fail@example.com"}, Subject: "B", BodyHTML: "<p>B</p>"},
		{To: []string{"ok2@example.com"}, Subject: "C", BodyHTML: "<p>C</p>"},
	}

	err := provider.SendBulk(context.Background(), emails)
	if err == nil {
		t.Fatal("expected error for partial failure")
	}

	var multiErr *email_domain.MultiError
	if !errors.As(err, &multiErr) {
		t.Fatalf("expected MultiError, got %T", err)
	}
	if multiErr.Count() != 1 {
		t.Errorf("expected 1 error (second email), got %d", multiErr.Count())
	}

	if callCount != 3 {
		t.Errorf("expected 3 API calls (all emails attempted), got %d", callCount)
	}
}

func TestSendBulk_AllSucceed(t *testing.T) {
	server := newTestServer(t, http.StatusOK, `[{"email":"ok@example.com","status":"sent","_id":"1"}]`)
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	emails := []*email_dto.SendParams{
		{To: []string{"a@example.com"}, Subject: "A", BodyHTML: "<p>A</p>"},
		{To: []string{"b@example.com"}, Subject: "B", BodyHTML: "<p>B</p>"},
	}

	err := provider.SendBulk(context.Background(), emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendBulk_EachEmailSentSeparately(t *testing.T) {
	var receivedRequests []mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req mandrillSendRequest
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		receivedRequests = append(receivedRequests, req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"ok@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	emails := []*email_dto.SendParams{
		{To: []string{"a@example.com"}, Subject: "Email A", BodyHTML: "<p>A</p>"},
		{To: []string{"b@example.com"}, Subject: "Email B", BodyPlain: "B"},
	}

	err := provider.SendBulk(context.Background(), emails)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedRequests) != 2 {
		t.Fatalf("expected 2 separate API calls, got %d", len(receivedRequests))
	}

	if receivedRequests[0].Message.Subject != "Email A" {
		t.Errorf("expected first request subject 'Email A', got '%s'",
			receivedRequests[0].Message.Subject)
	}
	if receivedRequests[1].Message.Subject != "Email B" {
		t.Errorf("expected second request subject 'Email B', got '%s'",
			receivedRequests[1].Message.Subject)
	}
}

func TestBuildMandrillRequest_MinimalParams(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	params := &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Hello",
		BodyHTML: "<p>World</p>",
	}

	request := provider.buildMandrillRequest(params)

	if request.Key != "test-api-key" {
		t.Errorf("expected API key 'test-api-key', got '%s'", request.Key)
	}
	if request.Message.FromEmail != "sender@example.com" {
		t.Errorf("expected default from email, got '%s'", request.Message.FromEmail)
	}
	if request.Message.Subject != "Hello" {
		t.Errorf("expected subject 'Hello', got '%s'", request.Message.Subject)
	}
	if request.Message.HTML != "<p>World</p>" {
		t.Errorf("unexpected HTML: '%s'", request.Message.HTML)
	}
	if len(request.Message.To) != 1 {
		t.Fatalf("expected 1 recipient, got %d", len(request.Message.To))
	}
	if request.Message.To[0].Email != "to@example.com" {
		t.Errorf("expected recipient email, got '%s'", request.Message.To[0].Email)
	}
	if request.Async {
		t.Error("expected async=false by default")
	}
	if request.Message.Attachments != nil {
		t.Error("expected nil attachments for no attachments")
	}
	if request.Message.Images != nil {
		t.Error("expected nil images for no attachments")
	}
}

func TestBuildMandrillRequest_CustomFrom(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	params := &email_dto.SendParams{
		From:     new("override@example.com"),
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	}

	request := provider.buildMandrillRequest(params)

	if request.Message.FromEmail != "override@example.com" {
		t.Errorf("expected overridden from email, got '%s'", request.Message.FromEmail)
	}
}

func TestBuildMandrillRequest_WithAttachmentsAndImages(t *testing.T) {
	provider := newTestProvider(t, "http://unused")

	params := &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
		Attachments: []email_dto.Attachment{
			{Filename: "file.txt", MIMEType: "text/plain", Content: []byte("hello")},
			{Filename: "logo.png", MIMEType: "image/png", ContentID: "logo", Content: []byte("png")},
		},
	}

	request := provider.buildMandrillRequest(params)

	if len(request.Message.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(request.Message.Attachments))
	}
	if len(request.Message.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(request.Message.Images))
	}
}

func TestSend_ServerUnreachable(t *testing.T) {
	provider := newTestProvider(t, "http://127.0.0.1:1")

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if !contains(err.Error(), "failed to send") {
		t.Errorf("expected connection error, got: %v", err)
	}
}

func TestSend_ValidationFailsBeforeHTTPCall(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
	})
	if !errors.Is(err, email_domain.ErrRecipientRequired) {
		t.Fatalf("expected ErrRecipientRequired, got: %v", err)
	}

	err = provider.Send(context.Background(), &email_dto.SendParams{
		To:      []string{"to@example.com"},
		Subject: "Test",
	})
	if !errors.Is(err, email_domain.ErrBodyRequired) {
		t.Fatalf("expected ErrBodyRequired, got: %v", err)
	}

	if callCount != 0 {
		t.Errorf("expected 0 HTTP calls for validation failures, got %d", callCount)
	}
}

func TestSend_MultipleAttachmentsAllBase64Encoded(t *testing.T) {
	var receivedBody mandrillSendRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"email":"to@example.com","status":"sent","_id":"1"}]`))
	}))
	defer server.Close()

	provider := newTestProvider(t, server.URL)

	err := provider.Send(context.Background(), &email_dto.SendParams{
		To:       []string{"to@example.com"},
		Subject:  "Test",
		BodyHTML: "<p>Test</p>",
		Attachments: []email_dto.Attachment{
			{Filename: "a.txt", MIMEType: "text/plain", Content: []byte("content-a")},
			{Filename: "b.pdf", MIMEType: "application/pdf", Content: []byte("content-b")},
			{Filename: "c.zip", MIMEType: "application/zip", Content: []byte("content-c")},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedBody.Message.Attachments) != 3 {
		t.Fatalf("expected 3 attachments, got %d", len(receivedBody.Message.Attachments))
	}

	for i, att := range receivedBody.Message.Attachments {
		decoded, decErr := base64.StdEncoding.DecodeString(att.Content)
		if decErr != nil {
			t.Errorf("attachment[%d]: failed to decode base64: %v", i, decErr)
			continue
		}
		if len(decoded) == 0 {
			t.Errorf("attachment[%d]: decoded content is empty", i)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
