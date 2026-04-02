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

package browser_provider_chromedp

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
)

func TestNewDialogHandler(t *testing.T) {
	dh := NewDialogHandler()

	if dh == nil {
		t.Fatal("NewDialogHandler() returned nil")
	}
	if dh.enabled {
		t.Error("new handler should not be enabled")
	}
	if dh.autoAccept {
		t.Error("new handler should not auto-accept")
	}
	if dh.lastDialog != nil {
		t.Error("new handler should have nil lastDialog")
	}
	if dh.closed {
		t.Error("new handler should not be closed")
	}
}

func TestDialogHandler_RecordDialog(t *testing.T) {
	dh := NewDialogHandler()

	event := &page.EventJavascriptDialogOpening{
		Type:    page.DialogTypeAlert,
		Message: "test alert",
		URL:     "https://example.com",
	}

	info := dh.recordDialog(event)

	if info.Type != DialogTypeAlert {
		t.Errorf("Type = %q, expected %q", info.Type, DialogTypeAlert)
	}
	if info.Message != "test alert" {
		t.Errorf("Message = %q, expected %q", info.Message, "test alert")
	}
	if info.URL != "https://example.com" {
		t.Errorf("URL = %q, expected %q", info.URL, "https://example.com")
	}

	last := dh.GetLastDialog()
	if last == nil {
		t.Fatal("GetLastDialog() returned nil after recordDialog")
	}
	if last.Message != "test alert" {
		t.Errorf("lastDialog.Message = %q, expected %q", last.Message, "test alert")
	}
}

func TestDialogHandler_RecordDialog_Types(t *testing.T) {
	testCases := []struct {
		name     string
		cdpType  page.DialogType
		expected DialogType
	}{
		{name: "alert", cdpType: page.DialogTypeAlert, expected: DialogTypeAlert},
		{name: "confirm", cdpType: page.DialogTypeConfirm, expected: DialogTypeConfirm},
		{name: "prompt", cdpType: page.DialogTypePrompt, expected: DialogTypePrompt},
		{name: "beforeunload", cdpType: page.DialogTypeBeforeunload, expected: DialogTypeBeforeunload},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dh := NewDialogHandler()
			event := &page.EventJavascriptDialogOpening{
				Type:    tc.cdpType,
				Message: "test",
			}
			info := dh.recordDialog(event)
			if info.Type != tc.expected {
				t.Errorf("Type = %q, expected %q", info.Type, tc.expected)
			}
		})
	}
}

func TestDialogHandler_RecordDialog_ChannelDelivery(t *testing.T) {
	dh := NewDialogHandler()

	event := &page.EventJavascriptDialogOpening{
		Type:    page.DialogTypeAlert,
		Message: "channel test",
	}

	dh.recordDialog(event)

	select {
	case info := <-dh.dialogChan:
		if info.Message != "channel test" {
			t.Errorf("channel message = %q, expected %q", info.Message, "channel test")
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for dialog on channel")
	}
}

func TestDialogHandler_RecordDialog_ClosedChannelNoPanic(t *testing.T) {
	dh := NewDialogHandler()
	dh.mu.Lock()
	dh.closed = true
	dh.mu.Unlock()

	event := &page.EventJavascriptDialogOpening{
		Type:    page.DialogTypeAlert,
		Message: "should not send",
	}

	info := dh.recordDialog(event)
	if info.Message != "should not send" {
		t.Errorf("Message = %q, expected %q", info.Message, "should not send")
	}

	select {
	case <-dh.dialogChan:
		t.Error("should not have received on channel when closed")
	default:

	}
}

func TestDialogHandler_SetPromptText(t *testing.T) {
	dh := NewDialogHandler()

	dh.SetPromptText("hello")

	dh.mu.RLock()
	text := dh.autoText
	dh.mu.RUnlock()

	if text != "hello" {
		t.Errorf("autoText = %q, expected %q", text, "hello")
	}
}

func TestDialogHandler_GetLastDialog_NilWhenNoDialogs(t *testing.T) {
	dh := NewDialogHandler()
	if dh.GetLastDialog() != nil {
		t.Error("GetLastDialog() should be nil for new handler")
	}
}

func TestDialogHandler_ClearLastDialog(t *testing.T) {
	dh := NewDialogHandler()

	event := &page.EventJavascriptDialogOpening{
		Type:    page.DialogTypeAlert,
		Message: "to be cleared",
	}
	dh.recordDialog(event)

	if dh.GetLastDialog() == nil {
		t.Fatal("expected non-nil lastDialog after recordDialog")
	}

	dh.ClearLastDialog()

	if dh.GetLastDialog() != nil {
		t.Error("GetLastDialog() should be nil after ClearLastDialog")
	}
}

func TestDialogHandler_WaitForDialog_Timeout(t *testing.T) {
	dh := NewDialogHandler()

	_, err := dh.WaitForDialog(10 * time.Millisecond)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestDialogHandler_WaitForDialog_Receives(t *testing.T) {
	dh := NewDialogHandler()

	go func() {
		time.Sleep(5 * time.Millisecond)
		event := &page.EventJavascriptDialogOpening{
			Type:    page.DialogTypeConfirm,
			Message: "async dialog",
		}
		dh.recordDialog(event)
	}()

	info, err := dh.WaitForDialog(time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Message != "async dialog" {
		t.Errorf("Message = %q, expected %q", info.Message, "async dialog")
	}
}

func TestDialogHandler_IsStopped(t *testing.T) {
	dh := NewDialogHandler()

	if !dh.isStopped() {
		t.Error("new disabled handler should report as stopped")
	}

	dh.mu.Lock()
	dh.enabled = true
	dh.mu.Unlock()

	if dh.isStopped() {
		t.Error("enabled handler should not report as stopped")
	}

	close(dh.stopChan)

	if !dh.isStopped() {
		t.Error("handler with closed stopChan should report as stopped")
	}
}

func TestDialogTypeConstants(t *testing.T) {
	if DialogTypeAlert != "alert" {
		t.Errorf("DialogTypeAlert = %q, expected %q", DialogTypeAlert, "alert")
	}
	if DialogTypeConfirm != "confirm" {
		t.Errorf("DialogTypeConfirm = %q, expected %q", DialogTypeConfirm, "confirm")
	}
	if DialogTypePrompt != "prompt" {
		t.Errorf("DialogTypePrompt = %q, expected %q", DialogTypePrompt, "prompt")
	}
	if DialogTypeBeforeunload != "beforeunload" {
		t.Errorf("DialogTypeBeforeunload = %q, expected %q", DialogTypeBeforeunload, "beforeunload")
	}
}
