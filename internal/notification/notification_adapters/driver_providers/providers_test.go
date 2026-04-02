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

package driver_providers

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/notification/notification_dto"
)

func testSendParams() *notification_dto.SendParams {
	return &notification_dto.SendParams{
		Content: notification_dto.NotificationContent{
			Title:   "Test Alert",
			Message: "Something went wrong in production.",
			Fields:  map[string]string{"region": "eu-west-1", "host": "web-01"},
		},
		Context: notification_dto.NotificationContext{
			Priority:    notification_dto.PriorityHigh,
			Source:      "monitoring",
			Environment: "production",
			Timestamp:   time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC),
		},
	}
}

func captureRequestHandler(t *testing.T, statusCode int, responseBody string) (http.HandlerFunc, *bytes.Buffer) {
	t.Helper()
	var captured bytes.Buffer
	handler := func(w http.ResponseWriter, r *http.Request) {
		captured.Reset()
		_, _ = io.Copy(&captured, r.Body)
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	}
	return handler, &captured
}

func TestSlackProvider_Send_Success(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "ok")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewSlackProvider(server.URL, server.Client())
	ctx := context.Background()
	err := provider.Send(ctx, testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if _, ok := payload["text"]; !ok {
		t.Error("expected 'text' field in Slack payload")
	}
	blocks, ok := payload["blocks"].([]any)
	if !ok {
		t.Fatal("expected 'blocks' array in Slack payload")
	}
	if len(blocks) == 0 {
		t.Error("expected at least one block")
	}

	firstBlock, ok := blocks[0].(map[string]any)
	require.True(t, ok, "expected blocks[0] to be map[string]any")
	if firstBlock["type"] != "header" {
		t.Errorf("expected first block type 'header', got %v", firstBlock["type"])
	}
}

func TestSlackProvider_Send_ServerError(t *testing.T) {
	handler, _ := captureRequestHandler(t, http.StatusInternalServerError, "error")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewSlackProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestSlackProvider_Send_RequestError(t *testing.T) {
	provider := NewSlackProvider("http://localhost:1/invalid", &http.Client{Timeout: 100 * time.Millisecond})
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestSlackProvider_PriorityEmoji(t *testing.T) {
	testCases := []struct {
		name     string
		emoji    string
		priority notification_dto.NotificationPriority
	}{
		{name: "critical", priority: notification_dto.PriorityCritical, emoji: ":rotating_light:"},
		{name: "high", priority: notification_dto.PriorityHigh, emoji: ":warning:"},
		{name: "normal", priority: notification_dto.PriorityNormal, emoji: ":information_source:"},
		{name: "low", priority: notification_dto.PriorityLow, emoji: ":bulb:"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, body := captureRequestHandler(t, http.StatusOK, "ok")
			server := httptest.NewServer(handler)
			defer server.Close()

			params := testSendParams()
			params.Context.Priority = tc.priority
			provider := NewSlackProvider(server.URL, server.Client())
			_ = provider.Send(context.Background(), params)

			if !strings.Contains(body.String(), tc.emoji) {
				t.Errorf("expected emoji %q in payload, got: %s", tc.emoji, body.String())
			}
		})
	}
}

func TestSlackProvider_SendBulk(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewSlackProvider(server.URL, server.Client())
	notifications := []*notification_dto.SendParams{testSendParams(), testSendParams()}
	err := provider.SendBulk(context.Background(), notifications)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestSlackProvider_SupportsBulkSending(t *testing.T) {
	provider := NewSlackProvider("http://example.com", nil)
	if provider.SupportsBulkSending() {
		t.Error("expected false")
	}
}

func TestSlackProvider_GetCapabilities(t *testing.T) {
	provider := NewSlackProvider("http://example.com", nil)
	caps := provider.GetCapabilities()
	if !caps.SupportsRichFormatting {
		t.Error("expected SupportsRichFormatting")
	}
	if !caps.SupportsImages {
		t.Error("expected SupportsImages")
	}
	if caps.MaxMessageLength != 40000 {
		t.Errorf("expected 40000, got %d", caps.MaxMessageLength)
	}
}

func TestSlackProvider_Close(t *testing.T) {
	provider := NewSlackProvider("http://example.com", nil)
	if err := provider.Close(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSlackProvider_ContextBlock(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "ok")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewSlackProvider(server.URL, server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	raw := body.String()
	if !strings.Contains(raw, "monitoring") {
		t.Error("expected source 'monitoring' in payload")
	}
	if !strings.Contains(raw, "production") {
		t.Error("expected environment 'production' in payload")
	}
}

func TestDiscordProvider_Send_Success(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewDiscordProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	embeds, ok := payload["embeds"].([]any)
	if !ok || len(embeds) == 0 {
		t.Fatal("expected 'embeds' array")
	}

	embed, ok := embeds[0].(map[string]any)
	require.True(t, ok, "expected embeds[0] to be map[string]any")
	if embed["title"] != "Test Alert" {
		t.Errorf("expected title 'Test Alert', got %v", embed["title"])
	}
}

func TestDiscordProvider_Send_ServerError(t *testing.T) {
	handler, _ := captureRequestHandler(t, http.StatusInternalServerError, "error")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewDiscordProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestDiscordProvider_PriorityColorMapping(t *testing.T) {
	testCases := []struct {
		name     string
		priority notification_dto.NotificationPriority
		color    float64
	}{
		{name: "critical", priority: notification_dto.PriorityCritical, color: 15158332},
		{name: "high", priority: notification_dto.PriorityHigh, color: 15844367},
		{name: "normal", priority: notification_dto.PriorityNormal, color: 3447003},
		{name: "low", priority: notification_dto.PriorityLow, color: 9807270},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, body := captureRequestHandler(t, http.StatusOK, "")
			server := httptest.NewServer(handler)
			defer server.Close()

			params := testSendParams()
			params.Context.Priority = tc.priority
			provider := NewDiscordProvider(server.URL, server.Client())
			_ = provider.Send(context.Background(), params)

			var payload map[string]any
			_ = json.Unmarshal(body.Bytes(), &payload)
			embeds, ok := payload["embeds"].([]any)
			require.True(t, ok, "expected payload[\"embeds\"] to be []any")
			embed, ok := embeds[0].(map[string]any)
			require.True(t, ok, "expected embeds[0] to be map[string]any")
			if embed["color"] != tc.color {
				t.Errorf("expected color %v, got %v", tc.color, embed["color"])
			}
		})
	}
}

func TestDiscordProvider_SendBulk(t *testing.T) {
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewDiscordProvider(server.URL, server.Client())
	err := provider.SendBulk(context.Background(), []*notification_dto.SendParams{testSendParams(), testSendParams()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2, got %d", callCount)
	}
}

func TestDiscordProvider_GetCapabilities(t *testing.T) {
	provider := NewDiscordProvider("http://example.com", nil)
	caps := provider.GetCapabilities()
	if !caps.SupportsRichFormatting {
		t.Error("expected SupportsRichFormatting")
	}
	if !caps.SupportsImages {
		t.Error("expected SupportsImages")
	}
	if caps.MaxMessageLength != 6000 {
		t.Errorf("expected 6000, got %d", caps.MaxMessageLength)
	}
	if caps.SupportsBulkSending {
		t.Error("expected no bulk support")
	}
}

func TestDiscordProvider_Close(t *testing.T) {
	provider := NewDiscordProvider("http://example.com", nil)
	if err := provider.Close(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTeamsProvider_Send_Success(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "1")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewTeamsProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if payload["@type"] != "MessageCard" {
		t.Errorf("expected @type 'MessageCard', got %v", payload["@type"])
	}
	if payload["@context"] != "http://schema.org/extensions" {
		t.Errorf("unexpected @context: %v", payload["@context"])
	}
}

func TestTeamsProvider_Send_ServerError(t *testing.T) {
	handler, _ := captureRequestHandler(t, http.StatusInternalServerError, "error")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewTeamsProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestTeamsProvider_Send_WrongResponseBody(t *testing.T) {

	handler, _ := captureRequestHandler(t, http.StatusOK, "0")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewTeamsProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error when response body is not '1'")
	}
}

func TestTeamsProvider_PriorityThemeColor(t *testing.T) {
	testCases := []struct {
		name     string
		color    string
		priority notification_dto.NotificationPriority
	}{
		{name: "critical", priority: notification_dto.PriorityCritical, color: "FF0000"},
		{name: "high", priority: notification_dto.PriorityHigh, color: "FFA500"},
		{name: "normal", priority: notification_dto.PriorityNormal, color: "0078D4"},
		{name: "low", priority: notification_dto.PriorityLow, color: "808080"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, body := captureRequestHandler(t, http.StatusOK, "1")
			server := httptest.NewServer(handler)
			defer server.Close()

			params := testSendParams()
			params.Context.Priority = tc.priority
			provider := NewTeamsProvider(server.URL, server.Client())
			_ = provider.Send(context.Background(), params)

			var payload map[string]any
			_ = json.Unmarshal(body.Bytes(), &payload)
			if payload["themeColor"] != tc.color {
				t.Errorf("expected themeColor %q, got %v", tc.color, payload["themeColor"])
			}
		})
	}
}

func TestTeamsProvider_Sections(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "1")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewTeamsProvider(server.URL, server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	var payload map[string]any
	_ = json.Unmarshal(body.Bytes(), &payload)
	sections, ok := payload["sections"].([]any)
	if !ok || len(sections) == 0 {
		t.Fatal("expected 'sections' array")
	}

	section, ok := sections[0].(map[string]any)
	require.True(t, ok, "expected sections[0] to be map[string]any")
	if section["activityTitle"] != "Test Alert" {
		t.Errorf("expected activityTitle 'Test Alert', got %v", section["activityTitle"])
	}
}

func TestTeamsProvider_GetCapabilities(t *testing.T) {
	provider := NewTeamsProvider("http://example.com", nil)
	caps := provider.GetCapabilities()
	if !caps.SupportsRichFormatting {
		t.Error("expected SupportsRichFormatting")
	}
	if caps.SupportsImages {
		t.Error("expected no image support for Teams")
	}
	if caps.MaxMessageLength != 28000 {
		t.Errorf("expected 28000, got %d", caps.MaxMessageLength)
	}
}

func TestGoogleChatProvider_Send_Success(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, `{}`)
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewGoogleChatProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	cardsV2, ok := payload["cardsV2"].([]any)
	if !ok || len(cardsV2) == 0 {
		t.Fatal("expected 'cardsV2' array")
	}

	card, ok := cardsV2[0].(map[string]any)
	require.True(t, ok, "expected cardsV2[0] to be map[string]any")
	if card["cardId"] != "notification-card" {
		t.Errorf("expected cardId 'notification-card', got %v", card["cardId"])
	}
}

func TestGoogleChatProvider_Send_ServerError(t *testing.T) {
	handler, _ := captureRequestHandler(t, http.StatusInternalServerError, "")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewGoogleChatProvider(server.URL, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGoogleChatProvider_HeaderSubtitle(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, `{}`)
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewGoogleChatProvider(server.URL, server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	raw := body.String()
	if !strings.Contains(raw, "Priority") {
		t.Error("expected 'Priority' in header subtitle")
	}
}

func TestGoogleChatProvider_GetCapabilities(t *testing.T) {
	provider := NewGoogleChatProvider("http://example.com", nil)
	caps := provider.GetCapabilities()
	if !caps.SupportsRichFormatting {
		t.Error("expected SupportsRichFormatting")
	}
	if !caps.SupportsImages {
		t.Error("expected SupportsImages")
	}
	if caps.MaxMessageLength != 4096 {
		t.Errorf("expected 4096, got %d", caps.MaxMessageLength)
	}
}

func TestPagerDutyProvider_FormatPayload(t *testing.T) {
	provider := &PagerDutyProvider{
		httpClient: http.DefaultClient,
		routingKey: "test-routing-key",
	}

	params := testSendParams()
	payload, err := provider.formatPagerDutyPayload(params)
	if err != nil {
		t.Fatalf("failed to format payload: %v", err)
	}

	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if event["routing_key"] != "test-routing-key" {
		t.Errorf("expected routing_key 'test-routing-key', got %v", event["routing_key"])
	}
	if event["event_action"] != "trigger" {
		t.Errorf("expected event_action 'trigger', got %v", event["event_action"])
	}

	pdPayload, ok := event["payload"].(map[string]any)
	require.True(t, ok, "expected event[\"payload\"] to be map[string]any")
	if pdPayload["summary"] != "Test Alert" {
		t.Errorf("expected summary 'Test Alert', got %v", pdPayload["summary"])
	}
}

func TestPagerDutyProvider_SeverityMapping(t *testing.T) {
	testCases := []struct {
		name     string
		severity string
		priority notification_dto.NotificationPriority
	}{
		{name: "critical", priority: notification_dto.PriorityCritical, severity: "critical"},
		{name: "high", priority: notification_dto.PriorityHigh, severity: "error"},
		{name: "normal", priority: notification_dto.PriorityNormal, severity: "warning"},
		{name: "low", priority: notification_dto.PriorityLow, severity: "info"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider := &PagerDutyProvider{routingKey: "key", httpClient: http.DefaultClient}
			params := testSendParams()
			params.Context.Priority = tc.priority
			payload, err := provider.formatPagerDutyPayload(params)
			if err != nil {
				t.Fatalf("format error: %v", err)
			}

			var event map[string]any
			_ = json.Unmarshal(payload, &event)
			pdPayload, ok := event["payload"].(map[string]any)
			require.True(t, ok, "expected event[\"payload\"] to be map[string]any")
			if pdPayload["severity"] != tc.severity {
				t.Errorf("expected severity %q, got %v", tc.severity, pdPayload["severity"])
			}
		})
	}
}

func TestPagerDutyProvider_DedupKey(t *testing.T) {
	provider := &PagerDutyProvider{routingKey: "key", httpClient: http.DefaultClient}
	payload, _ := provider.formatPagerDutyPayload(testSendParams())

	var event map[string]any
	_ = json.Unmarshal(payload, &event)

	dedupKey, ok := event["dedup_key"].(string)
	if !ok || dedupKey == "" {
		t.Error("expected non-empty dedup_key")
	}
}

func TestPagerDutyProvider_CustomDetails(t *testing.T) {
	provider := &PagerDutyProvider{routingKey: "key", httpClient: http.DefaultClient}
	payload, _ := provider.formatPagerDutyPayload(testSendParams())

	var event map[string]any
	_ = json.Unmarshal(payload, &event)
	pdPayload, ok := event["payload"].(map[string]any)
	require.True(t, ok, "expected event[\"payload\"] to be map[string]any")
	details, ok := pdPayload["custom_details"].(map[string]any)
	if !ok {
		t.Fatal("expected custom_details map")
	}

	if details["message"] != "Something went wrong in production." {
		t.Errorf("expected message in custom_details, got %v", details["message"])
	}
}

func TestPagerDutyProvider_GetCapabilities(t *testing.T) {
	provider := NewPagerDutyProvider("key", nil)
	caps := provider.GetCapabilities()
	if caps.SupportsRichFormatting {
		t.Error("expected no rich formatting support")
	}
	if caps.SupportsImages {
		t.Error("expected no image support")
	}
	if !caps.RequiresAuthentication {
		t.Error("expected RequiresAuthentication")
	}
	if caps.MaxMessageLength != 1024 {
		t.Errorf("expected 1024, got %d", caps.MaxMessageLength)
	}
}

func TestPagerDutyProvider_Send_NetworkError(t *testing.T) {

	provider := NewPagerDutyProvider("test-key", &http.Client{Timeout: 100 * time.Millisecond})
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error when PagerDuty is unreachable")
	}
}

func TestNtfyProvider_Send_Success(t *testing.T) {
	var capturedHeaders http.Header
	var capturedBody bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		capturedBody.Reset()
		_, _ = io.Copy(&capturedBody, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewNtfyProvider(server.URL, "test-topic", server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedHeaders.Get("Title") != "Test Alert" {
		t.Errorf("expected Title header 'Test Alert', got %q", capturedHeaders.Get("Title"))
	}
	if capturedHeaders.Get("Priority") == "" {
		t.Error("expected Priority header")
	}
	if capturedHeaders.Get("Tags") == "" {
		t.Error("expected Tags header")
	}

	bodyString := capturedBody.String()
	if !strings.Contains(bodyString, "Something went wrong") {
		t.Error("expected message in body")
	}
}

func TestNtfyProvider_Send_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := NewNtfyProvider(server.URL, "test-topic", server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestNtfyProvider_PriorityMapping(t *testing.T) {
	testCases := []struct {
		name     string
		ntfyPri  string
		priority notification_dto.NotificationPriority
	}{
		{name: "critical", priority: notification_dto.PriorityCritical, ntfyPri: "urgent"},
		{name: "high", priority: notification_dto.PriorityHigh, ntfyPri: "high"},
		{name: "normal", priority: notification_dto.PriorityNormal, ntfyPri: "default"},
		{name: "low", priority: notification_dto.PriorityLow, ntfyPri: "low"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedHeaders http.Header
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedHeaders = r.Header.Clone()
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			params := testSendParams()
			params.Context.Priority = tc.priority
			provider := NewNtfyProvider(server.URL, "test-topic", server.Client())
			_ = provider.Send(context.Background(), params)

			if capturedHeaders.Get("Priority") != tc.ntfyPri {
				t.Errorf("expected priority %q, got %q", tc.ntfyPri, capturedHeaders.Get("Priority"))
			}
		})
	}
}

func TestNtfyProvider_MessageFormat(t *testing.T) {
	var capturedBody bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody.Reset()
		_, _ = io.Copy(&capturedBody, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewNtfyProvider(server.URL, "topic", server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	body := capturedBody.String()

	if !strings.Contains(body, "region") {
		t.Error("expected field 'region' in body")
	}

	if !strings.Contains(body, "monitoring") {
		t.Error("expected source in body")
	}
}

func TestNtfyProvider_GetCapabilities(t *testing.T) {
	provider := NewNtfyProvider("http://example.com", "topic", nil)
	caps := provider.GetCapabilities()
	if caps.SupportsRichFormatting {
		t.Error("expected no rich formatting")
	}
	if caps.SupportsImages {
		t.Error("expected no image support")
	}
	if caps.MaxMessageLength != 4096 {
		t.Errorf("expected 4096, got %d", caps.MaxMessageLength)
	}
}

func TestNtfyProvider_TrailingSlashRemoved(t *testing.T) {
	var requestURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewNtfyProvider(server.URL+"/", "my-topic", server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	if strings.Contains(requestURL, "//") {
		t.Errorf("expected no double slashes, got path %q", requestURL)
	}
}

func TestWebhookProvider_Send_Success(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "ok")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewWebhookProvider(server.URL, nil, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if payload["title"] != "Test Alert" {
		t.Errorf("expected title 'Test Alert', got %v", payload["title"])
	}
	if payload["message"] != "Something went wrong in production." {
		t.Errorf("expected message, got %v", payload["message"])
	}
	if payload["source"] != "monitoring" {
		t.Errorf("expected source 'monitoring', got %v", payload["source"])
	}
	if payload["environment"] != "production" {
		t.Errorf("expected environment 'production', got %v", payload["environment"])
	}
}

func TestWebhookProvider_Send_ServerError(t *testing.T) {
	handler, _ := captureRequestHandler(t, http.StatusInternalServerError, "error")
	server := httptest.NewServer(handler)
	defer server.Close()

	provider := NewWebhookProvider(server.URL, nil, server.Client())
	err := provider.Send(context.Background(), testSendParams())
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestWebhookProvider_CustomHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	headers := map[string]string{
		"X-Custom-Token": "secret-123",
		"X-Source":       "piko",
	}
	provider := NewWebhookProvider(server.URL, headers, server.Client())
	_ = provider.Send(context.Background(), testSendParams())

	if capturedHeaders.Get("X-Custom-Token") != "secret-123" {
		t.Errorf("expected custom header, got %q", capturedHeaders.Get("X-Custom-Token"))
	}
	if capturedHeaders.Get("X-Source") != "piko" {
		t.Errorf("expected X-Source header, got %q", capturedHeaders.Get("X-Source"))
	}
}

func TestWebhookProvider_PayloadFields(t *testing.T) {
	handler, body := captureRequestHandler(t, http.StatusOK, "ok")
	server := httptest.NewServer(handler)
	defer server.Close()

	params := testSendParams()
	params.Content.ImageURL = "https://example.com/img.png"

	provider := NewWebhookProvider(server.URL, nil, server.Client())
	_ = provider.Send(context.Background(), params)

	var payload map[string]any
	_ = json.Unmarshal(body.Bytes(), &payload)

	fields, ok := payload["fields"].(map[string]any)
	if !ok {
		t.Fatal("expected 'fields' map")
	}
	if fields["region"] != "eu-west-1" {
		t.Errorf("expected region 'eu-west-1', got %v", fields["region"])
	}
	if payload["image_url"] != "https://example.com/img.png" {
		t.Errorf("expected image_url, got %v", payload["image_url"])
	}
}

func TestWebhookProvider_GetCapabilities(t *testing.T) {
	provider := NewWebhookProvider("http://example.com", nil, nil)
	caps := provider.GetCapabilities()
	if caps.SupportsRichFormatting {
		t.Error("expected no rich formatting")
	}
	if !caps.SupportsImages {
		t.Error("expected SupportsImages")
	}
	if caps.MaxMessageLength != 100000 {
		t.Errorf("expected 100000, got %d", caps.MaxMessageLength)
	}
}

func TestWebhookProvider_Close(t *testing.T) {
	provider := NewWebhookProvider("http://example.com", nil, nil)
	if err := provider.Close(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStdoutProvider_Send_Success(t *testing.T) {
	provider := NewStdoutProvider()
	err := provider.Send(context.Background(), testSendParams())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStdoutProvider_SendBulk(t *testing.T) {
	provider := NewStdoutProvider()
	notifications := []*notification_dto.SendParams{testSendParams(), testSendParams()}
	err := provider.SendBulk(context.Background(), notifications)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStdoutProvider_SupportsBulkSending(t *testing.T) {
	provider := NewStdoutProvider()
	if !provider.SupportsBulkSending() {
		t.Error("expected SupportsBulkSending to be true")
	}
}

func TestStdoutProvider_GetCapabilities(t *testing.T) {
	provider := NewStdoutProvider()
	caps := provider.GetCapabilities()
	if caps.SupportsRichFormatting {
		t.Error("expected no rich formatting")
	}
	if caps.SupportsImages {
		t.Error("expected no image support")
	}
	if !caps.SupportsBulkSending {
		t.Error("expected SupportsBulkSending")
	}
	if caps.MaxMessageLength != 0 {
		t.Errorf("expected 0 (unlimited), got %d", caps.MaxMessageLength)
	}
}

func TestStdoutProvider_Close(t *testing.T) {
	provider := NewStdoutProvider()
	if err := provider.Close(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewSlackProvider_NilClient(t *testing.T) {
	provider := NewSlackProvider("http://example.com", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewDiscordProvider_NilClient(t *testing.T) {
	provider := NewDiscordProvider("http://example.com", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewTeamsProvider_NilClient(t *testing.T) {
	provider := NewTeamsProvider("http://example.com", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewGoogleChatProvider_NilClient(t *testing.T) {
	provider := NewGoogleChatProvider("http://example.com", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewPagerDutyProvider_NilClient(t *testing.T) {
	provider := NewPagerDutyProvider("key", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewNtfyProvider_NilClient(t *testing.T) {
	provider := NewNtfyProvider("http://example.com", "topic", nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewWebhookProvider_NilClient(t *testing.T) {
	provider := NewWebhookProvider("http://example.com", nil, nil)
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}
