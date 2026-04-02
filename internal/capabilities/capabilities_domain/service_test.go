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

package capabilities_domain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

func echoCapability(_ context.Context, input io.Reader, _ CapabilityParams) (io.Reader, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func TestNewCapabilityService(t *testing.T) {
	service := NewCapabilityService(4)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := NewCapabilityService(4)
		err := service.Register("echo", echoCapability)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate returns error", func(t *testing.T) {
		service := NewCapabilityService(4)

		if err := service.Register("echo", echoCapability); err != nil {
			t.Fatalf("first register failed: %v", err)
		}

		err := service.Register("echo", echoCapability)
		if err == nil {
			t.Fatal("expected error for duplicate registration, got nil")
		}
		if !errors.Is(err, errCapabilityExists) {
			t.Errorf("expected errCapabilityExists, got: %v", err)
		}
	})

	t.Run("different names succeed", func(t *testing.T) {
		service := NewCapabilityService(4)

		if err := service.Register("a", echoCapability); err != nil {
			t.Fatalf("register a: %v", err)
		}
		if err := service.Register("b", echoCapability); err != nil {
			t.Fatalf("register b: %v", err)
		}
	})
}

func TestExecute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		service := NewCapabilityService(4)
		if err := service.Register("echo", echoCapability); err != nil {
			t.Fatalf("register failed: %v", err)
		}

		input := strings.NewReader("hello")
		output, err := service.Execute(context.Background(), "echo", input, nil)
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		data, err := io.ReadAll(output)
		if err != nil {
			t.Fatalf("read output failed: %v", err)
		}
		if string(data) != "hello" {
			t.Errorf("expected %q, got %q", "hello", string(data))
		}
	})

	t.Run("not found", func(t *testing.T) {
		service := NewCapabilityService(4)

		_, err := service.Execute(context.Background(), "missing", nil, nil)
		if err == nil {
			t.Fatal("expected error for missing capability, got nil")
		}
		if !errors.Is(err, errCapabilityNotFound) {
			t.Errorf("expected errCapabilityNotFound, got: %v", err)
		}
	})

	t.Run("propagates capability error", func(t *testing.T) {
		service := NewCapabilityService(4)

		capErr := errors.New("processing failed")
		failing := func(context.Context, io.Reader, CapabilityParams) (io.Reader, error) {
			return nil, capErr
		}

		if err := service.Register("fail", failing); err != nil {
			t.Fatalf("register failed: %v", err)
		}

		_, err := service.Execute(context.Background(), "fail", nil, nil)
		if !errors.Is(err, capErr) {
			t.Errorf("expected capability error, got: %v", err)
		}
	})

	t.Run("params passed through", func(t *testing.T) {
		service := NewCapabilityService(4)

		var gotParams CapabilityParams
		capture := func(_ context.Context, input io.Reader, params CapabilityParams) (io.Reader, error) {
			gotParams = params
			return input, nil
		}

		if err := service.Register("capture", capture); err != nil {
			t.Fatalf("register failed: %v", err)
		}

		params := CapabilityParams{"quality": "80"}
		_, err := service.Execute(context.Background(), "capture", nil, params)
		if err != nil {
			t.Fatalf("execute failed: %v", err)
		}

		if gotParams["quality"] != "80" {
			t.Errorf("expected quality=80, got %q", gotParams["quality"])
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	service := NewCapabilityService(4)

	if err := service.Register("echo", echoCapability); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	var wg sync.WaitGroup
	const goroutines = 20

	for range goroutines {
		wg.Go(func() {
			input := strings.NewReader("concurrent")
			output, err := service.Execute(context.Background(), "echo", input, nil)
			if err != nil {
				t.Errorf("concurrent execute failed: %v", err)
				return
			}
			data, err := io.ReadAll(output)
			if err != nil {
				t.Errorf("concurrent read failed: %v", err)
				return
			}
			if string(data) != "concurrent" {
				t.Errorf("expected %q, got %q", "concurrent", string(data))
			}
		})
	}

	wg.Wait()
}
