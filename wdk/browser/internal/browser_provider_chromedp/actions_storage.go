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
	"fmt"

	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// StorageType represents the type of web storage (local or session).
type StorageType string

const (
	// StorageTypeLocal is the storage type for browser local storage.
	StorageTypeLocal StorageType = "localStorage"

	// StorageTypeSession is the storage type for session storage.
	StorageTypeSession StorageType = "sessionStorage"
)

// GetLocalStorageItem retrieves a value from localStorage by key.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes key (string) which specifies the storage key to retrieve.
//
// Returns string which contains the stored value if found, or empty string.
// Returns bool which indicates whether the key was found in storage.
// Returns error when the storage operation fails.
func GetLocalStorageItem(ctx *ActionContext, key string) (string, bool, error) {
	return getStorageItem(ctx, StorageTypeLocal, key)
}

// SetLocalStorageItem sets a value in localStorage.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes key (string) which specifies the storage key.
// Takes value (string) which specifies the value to store.
//
// Returns error when the browser action fails.
func SetLocalStorageItem(ctx *ActionContext, key, value string) error {
	js := scripts.MustExecute("storage_set_item.js.tmpl", map[string]any{
		"StorageType": "localStorage",
		"Key":         key,
		"Value":       value,
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("setting localStorage item %q: %w", key, err)
	}
	return nil
}

// RemoveLocalStorageItem removes a value from localStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes key (string) which specifies the localStorage key to remove.
//
// Returns error when the item cannot be removed from localStorage.
func RemoveLocalStorageItem(ctx *ActionContext, key string) error {
	js := scripts.MustExecute("storage_remove_item.js.tmpl", map[string]any{
		"StorageType": "localStorage",
		"Key":         key,
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("removing localStorage item %q: %w", key, err)
	}
	return nil
}

// ClearLocalStorage clears all items from localStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for the
// operation.
//
// Returns error when the localStorage cannot be cleared.
func ClearLocalStorage(ctx *ActionContext) error {
	js := scripts.MustGet("local_storage_clear.js")
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("clearing localStorage: %w", err)
	}
	return nil
}

// GetAllLocalStorage returns all key-value pairs in localStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns map[string]string which contains all localStorage entries.
// Returns error when retrieving the localStorage items fails.
func GetAllLocalStorage(ctx *ActionContext) (map[string]string, error) {
	js := scripts.MustGet("get_all_local_storage.js")

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting all localStorage items: %w", err)
	}

	items := make(map[string]string, len(result))
	for k, v := range result {
		if str, ok := v.(string); ok {
			items[k] = str
		}
	}
	return items, nil
}

// GetLocalStorageLength returns the number of items in localStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns int which is the count of items stored in localStorage.
// Returns error when the JavaScript evaluation fails.
func GetLocalStorageLength(ctx *ActionContext) (int, error) {
	js := scripts.MustGet("local_storage_length.js")
	var length float64
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &length))
	if err != nil {
		return 0, fmt.Errorf("getting localStorage length: %w", err)
	}
	return int(length), nil
}

// GetSessionStorageItem retrieves a value from sessionStorage by key.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes key (string) which specifies the storage key to look up.
//
// Returns string which contains the stored value if found, or empty string.
// Returns bool which indicates whether the key was found in storage.
// Returns error when the storage operation fails.
func GetSessionStorageItem(ctx *ActionContext, key string) (string, bool, error) {
	return getStorageItem(ctx, StorageTypeSession, key)
}

// SetSessionStorageItem sets a value in sessionStorage.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes key (string) which specifies the storage key name.
// Takes value (string) which specifies the value to store.
//
// Returns error when the JavaScript execution fails.
func SetSessionStorageItem(ctx *ActionContext, key, value string) error {
	js := scripts.MustExecute("storage_set_item.js.tmpl", map[string]any{
		"StorageType": "sessionStorage",
		"Key":         key,
		"Value":       value,
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("setting sessionStorage item %q: %w", key, err)
	}
	return nil
}

// RemoveSessionStorageItem removes a value from sessionStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes key (string) which specifies the storage key to remove.
//
// Returns error when the storage item cannot be removed.
func RemoveSessionStorageItem(ctx *ActionContext, key string) error {
	js := scripts.MustExecute("storage_remove_item.js.tmpl", map[string]any{
		"StorageType": "sessionStorage",
		"Key":         key,
	})

	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("removing sessionStorage item %q: %w", key, err)
	}
	return nil
}

// ClearSessionStorage clears all items from sessionStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns error when the sessionStorage cannot be cleared.
func ClearSessionStorage(ctx *ActionContext) error {
	js := scripts.MustGet("session_storage_clear.js")
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, nil))
	if err != nil {
		return fmt.Errorf("clearing sessionStorage: %w", err)
	}
	return nil
}

// GetAllSessionStorage returns all key-value pairs in sessionStorage.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns map[string]string which contains all sessionStorage items as
// key-value pairs.
// Returns error when the sessionStorage cannot be retrieved.
func GetAllSessionStorage(ctx *ActionContext) (map[string]string, error) {
	js := scripts.MustGet("get_all_session_storage.js")

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting all sessionStorage items: %w", err)
	}

	items := make(map[string]string, len(result))
	for k, v := range result {
		if str, ok := v.(string); ok {
			items[k] = str
		}
	}
	return items, nil
}

// GetSessionStorageLength returns the number of items in sessionStorage.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
//
// Returns int which is the count of items stored in sessionStorage.
// Returns error when the sessionStorage length cannot be retrieved.
func GetSessionStorageLength(ctx *ActionContext) (int, error) {
	js := scripts.MustGet("session_storage_length.js")
	var length float64
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &length))
	if err != nil {
		return 0, fmt.Errorf("getting sessionStorage length: %w", err)
	}
	return int(length), nil
}

// getStorageItem retrieves a value from the specified storage by key.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes storageType (StorageType) which specifies local or session storage.
// Takes key (string) which identifies the item to retrieve.
//
// Returns string which contains the stored value if found.
// Returns bool which indicates whether the key exists in storage.
// Returns error when the storage operation fails.
func getStorageItem(ctx *ActionContext, storageType StorageType, key string) (string, bool, error) {
	js := scripts.MustExecute("get_storage_item.js.tmpl", map[string]any{
		"StorageType": string(storageType),
		"Key":         key,
	})

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", false, fmt.Errorf("getting %s item %q: %w", storageType, key, err)
	}

	exists, ok := result["exists"].(bool)
	if !ok || !exists {
		return "", false, nil
	}

	value, ok := result["value"].(string)
	if !ok {
		return "", false, nil
	}
	return value, true, nil
}
