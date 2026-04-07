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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/email/email_dto"
)

func TestEmailError_String(t *testing.T) {
	e := &EmailError{
		Email:   email_dto.SendParams{To: []string{"alice@example.com"}},
		Attempt: 2,
		Error:   errors.New("connection refused"),
	}
	got := e.String()
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("expected recipient in output, got %q", got)
	}
	if !strings.Contains(got, "attempt 2") {
		t.Errorf("expected attempt number in output, got %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("expected error message in output, got %q", got)
	}
}

func TestNewMultiError_Empty(t *testing.T) {
	me := newMultiError(nil)
	if me != nil {
		t.Error("expected nil for empty slice")
	}
	me = newMultiError([]EmailError{})
	if me != nil {
		t.Error("expected nil for zero-length slice")
	}
}

func TestNewMultiError_NonEmpty(t *testing.T) {
	errs := []EmailError{
		{Error: errors.New("err1"), Attempt: 1},
	}
	me := newMultiError(errs)
	require.NotNil(t, me, "expected non-nil MultiError")
	if len(me.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(me.Errors))
	}
}

func TestMultiError_Error_Empty(t *testing.T) {
	me := &MultiError{}
	if got := me.Error(); got != "no errors" {
		t.Errorf("expected 'no errors', got %q", got)
	}
}

func TestMultiError_Error_Single(t *testing.T) {
	me := &MultiError{
		Errors: []EmailError{
			{
				Email:   email_dto.SendParams{To: []string{"bob@example.com"}},
				Attempt: 1,
				Error:   errors.New("timeout"),
			},
		},
	}
	got := me.Error()
	if !strings.Contains(got, "bob@example.com") {
		t.Errorf("expected recipient in error, got %q", got)
	}
	if strings.Contains(got, "multiple") {
		t.Errorf("single error should not say 'multiple', got %q", got)
	}
}

func TestMultiError_Error_Multiple(t *testing.T) {
	me := &MultiError{
		Errors: []EmailError{
			{Email: email_dto.SendParams{To: []string{"a@x.com"}}, Attempt: 1, Error: errors.New("err1")},
			{Email: email_dto.SendParams{To: []string{"b@x.com"}}, Attempt: 2, Error: errors.New("err2")},
		},
	}
	got := me.Error()
	if !strings.Contains(got, "multiple email send errors (2)") {
		t.Errorf("expected 'multiple email send errors (2)', got %q", got)
	}
	if !strings.Contains(got, "; ") {
		t.Errorf("expected semicolon separator, got %q", got)
	}
}

func TestMultiError_Add(t *testing.T) {
	me := &MultiError{}
	me.Add(&EmailError{Error: errors.New("e1"), Attempt: 1})
	me.Add(&EmailError{Error: errors.New("e2"), Attempt: 2})
	if len(me.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(me.Errors))
	}
}

func TestMultiError_HasErrors(t *testing.T) {
	me := &MultiError{}
	if me.HasErrors() {
		t.Error("expected HasErrors false for empty")
	}
	me.Add(&EmailError{Error: errors.New("e"), Attempt: 1})
	if !me.HasErrors() {
		t.Error("expected HasErrors true after add")
	}
}

func TestMultiError_Count(t *testing.T) {
	me := &MultiError{}
	if me.Count() != 0 {
		t.Errorf("expected 0, got %d", me.Count())
	}
	me.Add(&EmailError{Error: errors.New("e1"), Attempt: 1})
	me.Add(&EmailError{Error: errors.New("e2"), Attempt: 1})
	if me.Count() != 2 {
		t.Errorf("expected 2, got %d", me.Count())
	}
}

func TestMultiError_GetEmails(t *testing.T) {
	me := &MultiError{
		Errors: []EmailError{
			{Email: email_dto.SendParams{Subject: "s1"}, Error: errors.New("e"), Attempt: 1},
			{Email: email_dto.SendParams{Subject: "s2"}, Error: errors.New("e"), Attempt: 1},
		},
	}
	emails := me.GetEmails()
	if len(emails) != 2 {
		t.Fatalf("expected 2, got %d", len(emails))
	}
	if emails[0].Subject != "s1" || emails[1].Subject != "s2" {
		t.Error("emails not in expected order")
	}
}

func TestMultiError_GetReadyForRetry(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	me := &MultiError{
		Errors: []EmailError{
			{Error: errors.New("e1"), NextRetry: time.Time{}, Attempt: 1},
			{Error: errors.New("e2"), NextRetry: now.Add(-time.Minute), Attempt: 1},
			{Error: errors.New("e3"), NextRetry: now.Add(time.Minute), Attempt: 1},
			{Error: errors.New("e4"), NextRetry: now, Attempt: 1},
		},
	}
	ready := me.GetReadyForRetry(now)
	if len(ready) != 3 {
		t.Errorf("expected 3 ready, got %d", len(ready))
	}
}

func TestMultiError_Split(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	me := &MultiError{
		Errors: []EmailError{
			{Error: errors.New("e1"), NextRetry: time.Time{}, Attempt: 1},
			{Error: errors.New("e2"), NextRetry: now.Add(time.Hour), Attempt: 1},
			{Error: errors.New("e3"), NextRetry: now.Add(-time.Second), Attempt: 1},
		},
	}
	ready, waiting := me.Split(now)
	if len(ready) != 2 {
		t.Errorf("expected 2 ready, got %d", len(ready))
	}
	if len(waiting) != 1 {
		t.Errorf("expected 1 waiting, got %d", len(waiting))
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := defaultRetryConfig()
	if config.MaxRetries != defaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", config.MaxRetries, defaultMaxRetries)
	}
	if config.InitialDelay != defaultInitialDelay {
		t.Errorf("InitialDelay = %v, want %v", config.InitialDelay, defaultInitialDelay)
	}
	if config.MaxDelay != defaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", config.MaxDelay, defaultMaxDelay)
	}
	if config.BackoffFactor != defaultBackoffFactor {
		t.Errorf("BackoffFactor = %v, want %v", config.BackoffFactor, defaultBackoffFactor)
	}
	if !config.DeadLetterQueue {
		t.Error("expected DeadLetterQueue = true")
	}
}

func TestFlatJitter(t *testing.T) {
	for range 100 {
		d := flatJitter(5 * time.Second)
		if d < 0 || d >= 1000*time.Millisecond {
			t.Fatalf("jitter out of range [0, 1000ms): %v", d)
		}
	}
}
