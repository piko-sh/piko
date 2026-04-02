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
)

const testHTMLStorage = `<!DOCTYPE html>
<html>
<head><title>Storage Test</title></head>
<body>
<div id="content">Storage Test Page</div>
</body>
</html>`

func TestLocalStorage(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLStorage)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("set and get item", func(t *testing.T) {
			err := SetLocalStorageItem(ctx, "testKey", "testValue")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}

			value, exists, err := GetLocalStorageItem(ctx, "testKey")
			if err != nil {
				t.Fatalf("GetLocalStorageItem() error = %v", err)
			}
			if !exists {
				t.Fatal("GetLocalStorageItem() returned exists=false, want true")
			}
			if value != "testValue" {
				t.Errorf("GetLocalStorageItem() = %q, want %q", value, "testValue")
			}
		})

		t.Run("get non-existent item returns exists=false", func(t *testing.T) {
			value, exists, err := GetLocalStorageItem(ctx, "nonExistent")
			if err != nil {
				t.Fatalf("GetLocalStorageItem() error = %v", err)
			}
			if exists {
				t.Errorf("GetLocalStorageItem() returned exists=true, want false")
			}
			if value != "" {
				t.Errorf("GetLocalStorageItem() = %q, want empty string", value)
			}
		})

		t.Run("remove item", func(t *testing.T) {
			if err := SetLocalStorageItem(ctx, "toRemove", "value"); err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}

			if err := RemoveLocalStorageItem(ctx, "toRemove"); err != nil {
				t.Fatalf("RemoveLocalStorageItem() error = %v", err)
			}

			_, exists, err := GetLocalStorageItem(ctx, "toRemove")
			if err != nil {
				t.Fatalf("GetLocalStorageItem() error = %v", err)
			}
			if exists {
				t.Errorf("item still exists after removal")
			}
		})

		t.Run("clear all items", func(t *testing.T) {
			err := SetLocalStorageItem(ctx, "key1", "value1")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}
			err = SetLocalStorageItem(ctx, "key2", "value2")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}

			err = ClearLocalStorage(ctx)
			if err != nil {
				t.Fatalf("ClearLocalStorage() error = %v", err)
			}

			length, err := GetLocalStorageLength(ctx)
			if err != nil {
				t.Fatalf("GetLocalStorageLength() error = %v", err)
			}
			if length != 0 {
				t.Errorf("GetLocalStorageLength() = %d, want 0", length)
			}
		})

		t.Run("get all items", func(t *testing.T) {
			err := ClearLocalStorage(ctx)
			if err != nil {
				t.Fatalf("ClearLocalStorage() error = %v", err)
			}

			err = SetLocalStorageItem(ctx, "a", "1")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}
			err = SetLocalStorageItem(ctx, "b", "2")
			if err != nil {
				t.Fatalf("SetLocalStorageItem() error = %v", err)
			}

			items, err := GetAllLocalStorage(ctx)
			if err != nil {
				t.Fatalf("GetAllLocalStorage() error = %v", err)
			}

			if len(items) != 2 {
				t.Errorf("GetAllLocalStorage() returned %d items, want 2", len(items))
			}
			if items["a"] != "1" || items["b"] != "2" {
				t.Errorf("GetAllLocalStorage() = %v, want {a:1, b:2}", items)
			}
		})
	})
}

func TestSessionStorage(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLStorage)
	defer server.Close()

	withTestPage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("set and get item", func(t *testing.T) {
			if err := SetSessionStorageItem(ctx, "sessionKey", "sessionValue"); err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}

			value, exists, err := GetSessionStorageItem(ctx, "sessionKey")
			if err != nil {
				t.Fatalf("GetSessionStorageItem() error = %v", err)
			}
			if !exists {
				t.Fatal("GetSessionStorageItem() returned exists=false, want true")
			}
			if value != "sessionValue" {
				t.Errorf("GetSessionStorageItem() = %q, want %q", value, "sessionValue")
			}
		})

		t.Run("clear session storage", func(t *testing.T) {
			err := SetSessionStorageItem(ctx, "temp", "temp")
			if err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}

			err = ClearSessionStorage(ctx)
			if err != nil {
				t.Fatalf("ClearSessionStorage() error = %v", err)
			}

			length, err := GetSessionStorageLength(ctx)
			if err != nil {
				t.Fatalf("GetSessionStorageLength() error = %v", err)
			}
			if length != 0 {
				t.Errorf("GetSessionStorageLength() = %d, want 0", length)
			}
		})

		t.Run("remove session storage item", func(t *testing.T) {
			err := SetSessionStorageItem(ctx, "toRemove", "value")
			if err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}

			err = RemoveSessionStorageItem(ctx, "toRemove")
			if err != nil {
				t.Fatalf("RemoveSessionStorageItem() error = %v", err)
			}

			_, exists, err := GetSessionStorageItem(ctx, "toRemove")
			if err != nil {
				t.Fatalf("GetSessionStorageItem() error = %v", err)
			}
			if exists {
				t.Error("item still exists after removal")
			}
		})

		t.Run("get all session storage items", func(t *testing.T) {
			err := ClearSessionStorage(ctx)
			if err != nil {
				t.Fatalf("ClearSessionStorage() error = %v", err)
			}

			err = SetSessionStorageItem(ctx, "x", "10")
			if err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}
			err = SetSessionStorageItem(ctx, "y", "20")
			if err != nil {
				t.Fatalf("SetSessionStorageItem() error = %v", err)
			}

			items, err := GetAllSessionStorage(ctx)
			if err != nil {
				t.Fatalf("GetAllSessionStorage() error = %v", err)
			}

			if len(items) != 2 {
				t.Errorf("GetAllSessionStorage() returned %d items, want 2", len(items))
			}
			if items["x"] != "10" || items["y"] != "20" {
				t.Errorf("GetAllSessionStorage() = %v, want {x:10, y:20}", items)
			}
		})
	})
}
