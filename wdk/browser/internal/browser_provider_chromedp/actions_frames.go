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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

// FrameInfo holds details about an iframe element in the browser.
type FrameInfo struct {
	// ID is the unique identifier for this frame.
	ID string

	// Name is the identifier for this frame.
	Name string

	// URL is the source location or path associated with this frame.
	URL string

	// ParentID is the identifier of the parent frame; empty if this is a root frame.
	ParentID string
}

// GetFrames returns information about all frames in the page.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns []FrameInfo which contains details about each frame found.
// Returns error when the frame information cannot be retrieved.
func GetFrames(ctx *ActionContext) ([]FrameInfo, error) {
	var frames []FrameInfo

	err := chromedp.Run(ctx.Ctx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		frameTree, err := target.GetTargets().Do(ctx2)
		if err != nil {
			return err
		}

		for _, t := range frameTree {
			if t.Type == "iframe" || t.Type == "page" {
				frames = append(frames, FrameInfo{
					ID:       string(t.TargetID),
					Name:     "",
					URL:      t.URL,
					ParentID: "",
				})
			}
		}
		return nil
	}))

	if err != nil {
		return nil, fmt.Errorf("getting frames: %w", err)
	}

	return frames, nil
}

// GetFrameBySelector returns information about a frame identified by selector.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes selector (string) which specifies the CSS selector for the iframe.
//
// Returns *FrameInfo which contains the frame's ID, name, URL, and parent ID.
// Returns error when the frame info cannot be retrieved or the iframe is not
// found.
func GetFrameBySelector(ctx *ActionContext, selector string) (*FrameInfo, error) {
	js := scripts.MustExecute("iframe_get_info.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result map[string]any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("getting frame info: %w", err)
	}

	if result == nil {
		return nil, fmt.Errorf("iframe not found: %s", selector)
	}

	return &FrameInfo{
		ID:       getString(result, "id"),
		Name:     getString(result, "name"),
		URL:      getString(result, "src"),
		ParentID: "",
	}, nil
}

// EvalInFrame evaluates JavaScript within a specific iframe.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes frameSelector (string) which identifies the target iframe element.
// Takes script (string) which contains the JavaScript code to evaluate.
//
// Returns any which is the result of the JavaScript evaluation.
// Returns error when the script cannot be evaluated in the frame.
func EvalInFrame(ctx *ActionContext, frameSelector, script string) (any, error) {
	js := scripts.MustExecute("iframe_eval.js.tmpl", map[string]any{
		"FrameSelector": frameSelector,
		"Script":        script,
	})

	var result any
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return nil, fmt.Errorf("evaluating in frame: %w", err)
	}

	return result, nil
}

// ClickInFrame clicks an element within an iframe.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes frameSelector (string) which identifies the iframe element.
// Takes elementSelector (string) which identifies the element to click within
// the iframe.
//
// Returns error when the click action fails to execute.
func ClickInFrame(ctx *ActionContext, frameSelector, elementSelector string) error {
	js := scripts.MustExecute("iframe_click.js.tmpl", map[string]any{
		"FrameSelector":   frameSelector,
		"ElementSelector": elementSelector,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return fmt.Errorf("clicking in frame: %w", err)
	}

	return nil
}

// GetTextInFrame gets the text content of an element within an iframe.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes frameSelector (string) which identifies the target iframe.
// Takes elementSelector (string) which identifies the element within the frame.
//
// Returns string which is the text content of the matched element.
// Returns error when the script execution fails.
func GetTextInFrame(ctx *ActionContext, frameSelector, elementSelector string) (string, error) {
	js := scripts.MustExecute("iframe_get_text.js.tmpl", map[string]any{
		"FrameSelector":   frameSelector,
		"ElementSelector": elementSelector,
	})

	var result string
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", fmt.Errorf("getting text in frame: %w", err)
	}

	return result, nil
}

// FillInFrame fills an input element within an iframe.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes frameSelector (string) which identifies the iframe element.
// Takes elementSelector (string) which identifies the input element within the
// iframe.
// Takes value (string) which is the text to enter into the input element.
//
// Returns error when the JavaScript execution fails or the element cannot be
// found.
func FillInFrame(ctx *ActionContext, frameSelector, elementSelector, value string) error {
	js := scripts.MustExecute("iframe_fill.js.tmpl", map[string]any{
		"FrameSelector":   frameSelector,
		"ElementSelector": elementSelector,
		"Value":           value,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return fmt.Errorf("filling in frame: %w", err)
	}

	return nil
}

// WaitForElementInFrame waits for an element to appear within an iframe.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes frameSelector (string) which identifies the iframe to search within.
// Takes elementSelector (string) which identifies the element to wait for.
//
// Returns error when the element cannot be found or the evaluation fails.
func WaitForElementInFrame(ctx *ActionContext, frameSelector, elementSelector string) error {
	js := scripts.MustExecute("iframe_wait_for_element_poll.js.tmpl", map[string]any{
		"FrameSelector":   frameSelector,
		"ElementSelector": elementSelector,
	})

	const (
		pollInterval = 100 * time.Millisecond
		timeout      = 5 * time.Second
	)

	deadline := time.Now().Add(timeout)
	for {
		var found bool
		err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &found))
		if err != nil {
			return fmt.Errorf("waiting for element in frame: %w", err)
		}
		if found {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for element %q in frame %q", elementSelector, frameSelector)
		}
		time.Sleep(pollInterval)
	}
}

// GetFrameDocument returns the HTML content of an iframe's document.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes frameSelector (string) which specifies the CSS selector for the iframe.
//
// Returns string which contains the full HTML content of the iframe document.
// Returns error when the frame cannot be found or the script fails to execute.
func GetFrameDocument(ctx *ActionContext, frameSelector string) (string, error) {
	js := scripts.MustExecute("iframe_get_html.js.tmpl", map[string]any{
		"Selector": frameSelector,
	})

	var result string
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", fmt.Errorf("getting frame document: %w", err)
	}

	return result, nil
}

// IsFrameLoaded checks if an iframe has finished loading.
//
// Takes ctx (*ActionContext) which provides the browser execution context.
// Takes frameSelector (string) which specifies the CSS selector for the iframe.
//
// Returns bool which indicates whether the frame has finished loading.
// Returns error when the frame check fails.
func IsFrameLoaded(ctx *ActionContext, frameSelector string) (bool, error) {
	js := scripts.MustExecute("iframe_is_loaded.js.tmpl", map[string]any{
		"Selector": frameSelector,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return false, fmt.Errorf("checking frame loaded: %w", err)
	}

	return result, nil
}

// ExecuteInFrameContext executes actions within a specific frame context.
// This uses CDP to switch the execution context to the frame.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes frameSelector (string) which identifies the frame element to target.
// Takes callback (func(...)) which is the function to execute within the frame.
//
// Returns error when the frame cannot be found or execution fails.
func ExecuteInFrameContext(ctx *ActionContext, frameSelector string, callback func(frameCtx context.Context) error) error {
	var frameContextID cruntime.ExecutionContextID

	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DefaultActionTimeout, fmt.Errorf("frame ExecuteInFrameContext exceeded %s timeout", DefaultActionTimeout))
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		var frameNodes []*cdp.Node
		if err := chromedp.Nodes(frameSelector, &frameNodes, chromedp.ByQuery).Do(ctx2); err != nil {
			return fmt.Errorf("finding frame: %w", err)
		}
		if len(frameNodes) == 0 {
			return fmt.Errorf("frame not found: %s", frameSelector)
		}

		frameNode := frameNodes[0]
		if frameNode.ContentDocument == nil {
			return errors.New("frame has no content document")
		}

		js := `(() => {
			return { contextId: true };
		})()`

		result, _, err := cruntime.Evaluate(js).
			WithContextID(frameContextID).
			Do(ctx2)
		if err != nil {
			return fmt.Errorf("getting frame context: %w", err)
		}

		_ = result

		return nil
	}))

	if err != nil {
		return err
	}

	return callback(ctx.Ctx)
}

// CountFrames returns the number of iframes in the page.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
//
// Returns int which is the count of iframe elements found.
// Returns error when the JavaScript evaluation fails.
func CountFrames(ctx *ActionContext) (int, error) {
	js := `document.querySelectorAll('iframe').length`

	var count float64
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &count))
	if err != nil {
		return 0, fmt.Errorf("counting frames: %w", err)
	}

	return int(count), nil
}
