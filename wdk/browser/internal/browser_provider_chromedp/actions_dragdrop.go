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
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp/scripts"
)

const (
	// dragStepCount is the number of intermediate steps used in drag operations.
	dragStepCount = 10

	// dragStepInterval is the delay between each mouse movement step during drag.
	dragStepInterval = 10 * time.Millisecond
)

// DragAndDrop drags an element from source to target selector.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetSelector (string) which identifies the drop target element.
//
// Returns error when the source or target element cannot be found, or when
// the drag operation fails.
func DragAndDrop(ctx *ActionContext, sourceSelector, targetSelector string) error {
	sourceX, sourceY, err := getElementCentre(ctx, sourceSelector)
	if err != nil {
		return fmt.Errorf("getting source element position: %w", err)
	}

	targetX, targetY, err := getElementCentre(ctx, targetSelector)
	if err != nil {
		return fmt.Errorf("getting target element position: %w", err)
	}

	return performDrag(ctx, sourceX, sourceY, targetX, targetY)
}

// DragTo drags an element to specific coordinates.
//
// Takes ctx (*ActionContext) which provides the browser action context.
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetX (float64) which specifies the target x coordinate.
// Takes targetY (float64) which specifies the target y coordinate.
//
// Returns error when the source element position cannot be determined or the
// drag operation fails.
func DragTo(ctx *ActionContext, sourceSelector string, targetX, targetY float64) error {
	sourceX, sourceY, err := getElementCentre(ctx, sourceSelector)
	if err != nil {
		return fmt.Errorf("getting source element position: %w", err)
	}

	return performDrag(ctx, sourceX, sourceY, targetX, targetY)
}

// DragByOffset drags an element by a relative offset.
//
// Takes ctx (*ActionContext) which provides the browser automation context.
// Takes selector (string) which identifies the element to drag.
// Takes offsetX (float64) which specifies the horizontal distance to drag.
// Takes offsetY (float64) which specifies the vertical distance to drag.
//
// Returns error when the element cannot be found or the drag fails.
func DragByOffset(ctx *ActionContext, selector string, offsetX, offsetY float64) error {
	x, y, err := getElementCentre(ctx, selector)
	if err != nil {
		return fmt.Errorf("getting element position: %w", err)
	}

	return performDrag(ctx, x, y, x+offsetX, y+offsetY)
}

// DragAndDropHTML5 performs an HTML5 drag and drop operation using
// dataTransfer. This is more reliable for HTML5 drag-and-drop implementations.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetSelector (string) which identifies the drop target element.
//
// Returns error when the drag and drop operation fails to execute.
func DragAndDropHTML5(ctx *ActionContext, sourceSelector, targetSelector string) error {
	js := scripts.MustExecute("drag_and_drop.js.tmpl", map[string]any{
		"SourceSelector": sourceSelector,
		"TargetSelector": targetSelector,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return fmt.Errorf("HTML5 drag and drop: %w", err)
	}
	return nil
}

// DragAndDropWithData performs an HTML5 drag and drop with custom data.
//
// Takes ctx (*ActionContext) which provides the browser context.
// Takes sourceSelector (string) which identifies the element to drag.
// Takes targetSelector (string) which identifies the drop target element.
// Takes data (map[string]string) which maps MIME types to their values.
//
// Returns error when the drag and drop operation fails.
func DragAndDropWithData(ctx *ActionContext, sourceSelector, targetSelector string, data map[string]string) error {
	var builder strings.Builder
	for mimeType, value := range data {
		_, _ = fmt.Fprintf(&builder, `dataTransfer.setData(%q, %q);`, mimeType, value)
	}
	dataSetup := builder.String()

	js := scripts.MustExecute("drag_and_drop_with_data.js.tmpl", map[string]any{
		"SourceSelector": sourceSelector,
		"TargetSelector": targetSelector,
		"DataSetup":      dataSetup,
	})

	var result bool
	err := chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return fmt.Errorf("drag and drop with data: %w", err)
	}
	return nil
}

// interpolateDragPoint computes the intermediate (x, y) position for a given
// step in a linear drag operation.
//
// Takes startX (float64) which specifies the x-coordinate of the drag origin.
// Takes startY (float64) which specifies the y-coordinate of the drag origin.
// Takes endX (float64) which specifies the x-coordinate of the drag destination.
// Takes endY (float64) which specifies the y-coordinate of the drag destination.
// Takes step (int) which is the current step number (1-based).
// Takes totalSteps (int) which is the total number of steps.
//
// Returns x (float64) which is the interpolated x-coordinate.
// Returns y (float64) which is the interpolated y-coordinate.
func interpolateDragPoint(startX, startY, endX, endY float64, step, totalSteps int) (x, y float64) {
	progress := float64(step) / float64(totalSteps)
	x = startX + (endX-startX)*progress
	y = startY + (endY-startY)*progress
	return x, y
}

// performDrag performs the actual drag operation using CDP input events.
//
// Takes ctx (*ActionContext) which provides the browser context for execution.
// Takes fromX (float64) which specifies the starting X coordinate.
// Takes fromY (float64) which specifies the starting Y coordinate.
// Takes toX (float64) which specifies the ending X coordinate.
// Takes toY (float64) which specifies the ending Y coordinate.
//
// Returns error when any mouse event fails to dispatch.
func performDrag(ctx *ActionContext, fromX, fromY, toX, toY float64) error {
	timedCtx, cancel := context.WithTimeoutCause(ctx.Ctx, DragActionTimeout, fmt.Errorf("drag-and-drop exceeded %s timeout", DragActionTimeout))
	defer cancel()

	err := chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return input.DispatchMouseEvent(input.MouseMoved, fromX, fromY).Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("moving to start position: %w", err)
	}

	err = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return input.DispatchMouseEvent(input.MousePressed, fromX, fromY).
			WithButton(input.Left).
			WithClickCount(1).
			Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("mouse down: %w", err)
	}

	for i := 1; i <= dragStepCount; i++ {
		x, y := interpolateDragPoint(fromX, fromY, toX, toY, i, dragStepCount)

		err = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
			return input.DispatchMouseEvent(input.MouseMoved, x, y).
				WithButton(input.Left).
				Do(ctx2)
		}))
		if err != nil {
			return fmt.Errorf("moving mouse step %d: %w", i, err)
		}
		time.Sleep(dragStepInterval)
	}

	err = chromedp.Run(timedCtx, chromedp.ActionFunc(func(ctx2 context.Context) error {
		return input.DispatchMouseEvent(input.MouseReleased, toX, toY).
			WithButton(input.Left).
			WithClickCount(1).
			Do(ctx2)
	}))
	if err != nil {
		return fmt.Errorf("mouse up: %w", err)
	}

	return nil
}

// getElementCentre returns the centre coordinates of an element.
//
// Takes ctx (*ActionContext) which provides the browser context for
// executing JavaScript.
// Takes selector (string) which identifies the element by CSS selector.
//
// Returns centreX (float64) which is the horizontal centre of the element.
// Returns centreY (float64) which is the vertical centre of the element.
// Returns err (error) when the element cannot be found or the coordinates
// are invalid.
func getElementCentre(ctx *ActionContext, selector string) (centreX float64, centreY float64, err error) {
	js := scripts.MustExecute("get_element_centre.js.tmpl", map[string]any{
		"Selector": selector,
	})

	var result map[string]any
	err = chromedp.Run(ctx.Ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return 0, 0, fmt.Errorf("getting element centre: %w", err)
	}

	if result == nil {
		return 0, 0, fmt.Errorf("element not found: %s", selector)
	}

	var ok bool
	centreX, ok = result["x"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("invalid x coordinate for element: %s", selector)
	}
	centreY, ok = result["y"].(float64)
	if !ok {
		return 0, 0, fmt.Errorf("invalid y coordinate for element: %s", selector)
	}
	return centreX, centreY, nil
}
