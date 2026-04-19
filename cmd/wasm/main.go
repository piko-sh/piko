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

//go:build js && wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"syscall/js"
	"time"

	"piko.sh/piko/cmd/wasm/internal/wasmrecover"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/render/render_adapters"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/wasm/wasm_adapters"
	"piko.sh/piko/internal/wasm/wasm_data"
	"piko.sh/piko/internal/wasm/wasm_domain"
	"piko.sh/piko/internal/wasm/wasm_dto"
	pikointerp "piko.sh/piko/wdk/interp/interp_provider_piko"
)

const (
	// defaultOperationTimeout is the default timeout for long-running operations.
	defaultOperationTimeout = 30 * time.Second

	// invalidRequestFmt is the format string used when a JS
	// request cannot be unmarshalled.
	invalidRequestFmt = "invalid request: %v"
)

// orchestrator holds the global WASM orchestrator that manages
// analysis, generation, and rendering.
var orchestrator *wasm_domain.Orchestrator

// main sets up the Piko WASM module and registers JavaScript bindings.
func main() {
	fmt.Println("[Piko WASM] Initialising...")

	console := wasm_adapters.NewJSConsole()
	stdlibLoader := wasm_adapters.NewStdlibLoader(
		wasm_adapters.WithLoadFunc(func() ([]byte, error) {
			return wasm_data.StdlibFBS, nil
		}),
		wasm_adapters.WithDecoder(func(_ []byte) (*inspector_dto.TypeData, error) {
			return wasm_data.GetStdlibTypeData()
		}),
	)

	generator := wasm_adapters.NewGeneratorAdapter(
		wasm_adapters.WithStdlibDataGetter(func() (*inspector_dto.TypeData, error) {
			return orchestrator.GetStdlibData()
		}),
		wasm_adapters.WithModuleName("playground"),
	)

	renderOrchestrator := render_domain.NewRenderOrchestrator(nil, nil, nil, nil)
	headlessRenderer := render_adapters.NewHeadlessRendererAdapter(renderOrchestrator)
	renderer := wasm_adapters.NewRenderAdapter(
		wasm_adapters.WithRendererStdlibDataGetter(func() (*inspector_dto.TypeData, error) {
			return orchestrator.GetStdlibData()
		}),
		wasm_adapters.WithRendererModuleName("playground"),
		wasm_adapters.WithHeadlessRenderer(headlessRenderer),
	)

	symbolProvider := pikointerp.NewWASMSymbolProvider()
	symbolLoader := pikointerp.NewWASMSymbolAdapter(symbolProvider)
	interpreterFactory := pikointerp.NewWASMInterpreterFactory()
	interpreter := wasm_adapters.NewInterpreterAdapter(
		wasm_adapters.WithSymbolLoader(symbolLoader),
		wasm_adapters.WithInterpreterFactory(interpreterFactory),
	)

	orchestrator = wasm_domain.NewOrchestrator(
		wasm_domain.WithStdlibLoader(stdlibLoader),
		wasm_domain.WithConsole(console),
		wasm_domain.WithGenerator(generator),
		wasm_domain.WithRenderer(renderer),
		wasm_domain.WithInterpreter(interpreter),
		wasm_domain.WithConfig(wasm_domain.Config{
			DefaultModuleName: "playground",
			MaxSourceSize:     2 * 1024 * 1024,
			EnableMetrics:     false,
		}),
	)

	registerJSFunctions()

	fmt.Println("[Piko WASM] Ready. Call piko.init() to initialise.")

	select {}
}

// registerJSFunctions sets up the piko global object and registers all
// JavaScript-callable functions for the WebAssembly module.
func registerJSFunctions() {
	piko := js.Global().Get("Object").New()
	js.Global().Set("piko", piko)

	piko.Set("init", js.FuncOf(jsInit))
	piko.Set("analyse", js.FuncOf(jsAnalyse))
	piko.Set("generate", js.FuncOf(jsGenerate))
	piko.Set("render", js.FuncOf(jsRender))
	piko.Set("dynamicRender", js.FuncOf(jsDynamicRender))
	piko.Set("getCompletions", js.FuncOf(jsGetCompletions))
	piko.Set("getHover", js.FuncOf(jsGetHover))
	piko.Set("validate", js.FuncOf(jsValidate))
	piko.Set("getRuntimeInfo", js.FuncOf(jsGetRuntimeInfo))

	piko.Set("parseTemplate", js.FuncOf(jsParseTemplate))
	piko.Set("renderPreview", js.FuncOf(jsRenderPreview))
}

// jsInit sets up the WASM runtime.
//
// Returns any which is a Promise that resolves when setup is complete.
func jsInit(_ js.Value, _ []js.Value) any {
	return newPromise(func() (any, error) {
		ctx := context.Background()
		if err := orchestrator.Initialise(ctx); err != nil {
			return nil, err
		}
		return map[string]any{"success": true}, nil
	})
}

// jsAnalyse checks Go source code and returns the results as a Promise.
//
// Expects a request object with the shape:
// { sources: { "path": "content" }, moduleName?: string }
//
// Takes arguments ([]js.Value) which contains the request object as the
// first element.
//
// Returns any which is a Promise that resolves to an
// AnalyseResponse or rejects with an error message.
func jsAnalyse(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.AnalyseRequest
	if errMessage := parseRequestSafely("piko.jsAnalyse", arguments, &request, "analyse requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromiseWithTimeout(defaultOperationTimeout, func(ctx context.Context) (any, error) {
		response, err := orchestrator.Analyse(ctx, &request)
		if err != nil {
			return nil, err
		}
		return response, nil
	})
}

// jsGenerate generates Go code from PK template sources.
//
// Expects a request object with the shape:
// { sources: { "path": "content" }, moduleName: string, baseDir?: string }
//
// Takes arguments ([]js.Value) which contains the request object as the
// first element.
//
// Returns any which is a Promise that resolves to a
// GenerateFromSourcesResponse or rejects with an error message.
func jsGenerate(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.GenerateFromSourcesRequest
	if errMessage := parseRequestSafely("piko.jsGenerate", arguments, &request, "generate requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromiseWithTimeout(defaultOperationTimeout, func(ctx context.Context) (any, error) {
		response, err := orchestrator.Generate(ctx, &request)
		if err != nil {
			return nil, err
		}
		return response, nil
	})
}

// jsRender renders PK templates to HTML.
//
// Expects a request object with the shape:
// { sources: { "path": "content" }, moduleName?: string,
// entryPoint?: string }
//
// Takes arguments ([]js.Value) which contains the request object as the
// first element.
//
// Returns any which is a Promise that resolves to a
// RenderFromSourcesResponse or rejects with an error message.
func jsRender(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.RenderFromSourcesRequest
	if errMessage := parseRequestSafely("piko.jsRender", arguments, &request, "render requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromiseWithTimeout(defaultOperationTimeout, func(ctx context.Context) (any, error) {
		response, err := orchestrator.Render(ctx, &request)
		if err != nil {
			return nil, err
		}
		return response, nil
	})
}

// jsDynamicRender performs full dynamic rendering: generates Go code,
// interprets it, and renders the resulting AST to HTML.
//
// Expects a request object with the shape:
// { sources: { "path": "content" }, moduleName: string,
// requestURL?: string, props?: object }
//
// Takes arguments ([]js.Value) which contains the request object as the
// first element.
//
// Returns any which is a Promise that resolves to a
// DynamicRenderResponse or rejects with an error message.
func jsDynamicRender(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.DynamicRenderRequest
	if errMessage := parseRequestSafely("piko.jsDynamicRender", arguments, &request, "dynamicRender requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromiseWithTimeout(defaultOperationTimeout, func(ctx context.Context) (any, error) {
		response, err := orchestrator.DynamicRender(ctx, &request)
		if err != nil {
			return nil, err
		}
		return response, nil
	})
}

// jsGetCompletions returns code completions for a given source position.
//
// Takes arguments ([]js.Value) which expects a request object with
// source, line, column, and an optional moduleName.
//
// Returns any which is a Promise that resolves to a
// CompletionResponse.
func jsGetCompletions(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.CompletionRequest
	if errMessage := parseRequestSafely("piko.jsGetCompletions", arguments, &request, "getCompletions requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromise(func() (any, error) {
		ctx := context.Background()
		response, err := orchestrator.GetCompletions(ctx, &request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

// jsGetHover returns hover information for a position in Go source code.
// Expects a request object with source, line, and column fields.
//
// Takes arguments ([]js.Value) which contains the request object with
// source, line, and column fields.
//
// Returns any which is a Promise that resolves to a HoverResponse.
func jsGetHover(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.HoverRequest
	if errMessage := parseRequestSafely("piko.jsGetHover", arguments, &request, "getHover requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromise(func() (any, error) {
		ctx := context.Background()
		response, err := orchestrator.GetHover(ctx, &request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

// jsValidate checks Go source code and returns the results as a promise.
// Expects a request object with source (string) and optional filePath (string).
//
// Takes arguments ([]js.Value) which contains the request object with
// source code.
//
// Returns any which is a JavaScript Promise that resolves to a
// ValidateResponse.
func jsValidate(_ js.Value, arguments []js.Value) any {
	var request wasm_dto.ValidateRequest
	if errMessage := parseRequestSafely("piko.jsValidate", arguments, &request, "validate requires a request object"); errMessage != "" {
		return rejectedPromise(errMessage)
	}

	return newPromise(func() (any, error) {
		ctx := context.Background()
		response, err := orchestrator.Validate(ctx, &request)
		if err != nil {
			return nil, err
		}

		return response, nil
	})
}

// jsGetRuntimeInfo returns runtime information about the orchestrator.
//
// Returns any which is the runtime information as a JavaScript-compatible
// object, or an error result if the operation fails.
func jsGetRuntimeInfo(_ js.Value, _ []js.Value) any {
	var result any
	if errResult := runSyncSafely("piko.jsGetRuntimeInfo", func() {
		ctx := context.Background()
		info, err := orchestrator.GetRuntimeInfo(ctx)
		if err != nil {
			result = errorResult(err.Error())
			return
		}
		result = marshalToJS(info)
	}); errResult != nil {
		return *errResult
	}
	return result
}

// jsParseTemplate parses a PK template.
//
// Not yet working: returns a rejected Promise to indicate that the feature is
// still being built. The TS typings expose parseTemplate as Promise-returning,
// so this must be a thenable so JS callers using await observe a structured
// rejection rather than a TypeError on a plain object.
//
// Returns any which is a rejected JavaScript Promise carrying a not-implemented
// error message.
func jsParseTemplate(_ js.Value, _ []js.Value) any {
	return rejectedPromise("parseTemplate not yet implemented")
}

// jsRenderPreview renders a template preview.
//
// Not yet working: returns a rejected Promise so JS callers awaiting
// renderPreview observe a structured rejection rather than a TypeError.
//
// Returns any which is a rejected JavaScript Promise carrying a not-implemented
// error message.
func jsRenderPreview(_ js.Value, _ []js.Value) any {
	return rejectedPromise("renderPreview not yet implemented")
}

// newPromise creates a JavaScript Promise that runs the given function.
// Includes panic recovery and proper cleanup of js.Func to prevent memory leaks.
//
// Takes operation (func() (any, error)) which provides the operation to run.
//
// Returns js.Value which is the JavaScript Promise object.
//
// Spawns a goroutine to run the function. The goroutine finishes when operation
// returns, resolving or rejecting the Promise based on the result.
func newPromise(operation func() (any, error)) js.Value {
	var handler js.Func

	handler = js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		resolve := arguments[0]
		reject := arguments[1]

		go func() {
			defer handler.Release()

			defer func() {
				if r := recover(); r != nil {
					reject.Invoke(fmt.Sprintf("panic in WASM handler: %v", r))
				}
			}()

			result, err := operation()
			if err != nil {
				reject.Invoke(err.Error())
				return
			}
			resolve.Invoke(marshalToJS(result))
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// newPromiseWithTimeout creates a JavaScript Promise with a time limit for
// long-running tasks. If the task takes longer than the time limit, the
// Promise is rejected with a timeout error.
//
// Takes timeout (time.Duration) which sets the maximum time before the task
// is cancelled.
// Takes operation (func(ctx context.Context) (any, error)) which runs the async
// task using the given context.
//
// Returns js.Value which is a JavaScript Promise that resolves with the
// result or rejects with an error message.
//
// Spawns a goroutine to run the function and handle timeout cancellation.
func newPromiseWithTimeout(timeout time.Duration, operation func(ctx context.Context) (any, error)) js.Value {
	var handler js.Func

	handler = js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		resolve := arguments[0]
		reject := arguments[1]

		go func() {
			defer handler.Release()

			defer func() {
				if r := recover(); r != nil {
					reject.Invoke(fmt.Sprintf("panic in WASM handler: %v", r))
				}
			}()

			ctx, cancel := context.WithTimeoutCause(context.Background(), timeout, fmt.Errorf("WASM execution exceeded %s timeout", timeout))
			defer cancel()

			result, err := operation(ctx)
			if err != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					reject.Invoke(fmt.Sprintf("operation timed out after %v", timeout))
				} else {
					reject.Invoke(err.Error())
				}
				return
			}
			resolve.Invoke(marshalToJS(result))
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// unmarshalJSValue converts a JavaScript value to a Go struct using JSON.
//
// Takes v (js.Value) which is the JavaScript value to convert.
// Takes target (any) which is a pointer to the Go struct to fill.
//
// Returns error when JSON conversion or parsing fails.
func unmarshalJSValue(v js.Value, target any) error {
	jsonString := js.Global().Get("JSON").Call("stringify", v).String()
	return json.Unmarshal([]byte(jsonString), target)
}

// marshalToJS converts a Go value to a JavaScript value using JSON.
//
// Takes v (any) which is the Go value to convert.
//
// Returns js.Value which is the JavaScript value, or an error object if
// conversion fails.
func marshalToJS(v any) js.Value {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return js.ValueOf(map[string]any{"error": err.Error()})
	}
	return js.Global().Get("JSON").Call("parse", string(jsonBytes))
}

// errorResult creates an error response object.
//
// Takes message (string) which specifies the error message to include.
//
// Returns js.Value which contains a map with success set to false and the
// error message.
func errorResult(message string) js.Value {
	return js.ValueOf(map[string]any{
		"success": false,
		"error":   message,
	})
}

// rejectedPromise constructs a JS Promise that immediately rejects with
// message. Used by synchronous handler prologues so callers awaiting
// piko.<method>(...) always observe a thenable, even when argument
// validation fails before any async work runs.
//
// Without this helper a synchronous prologue could return a plain object,
// which JS clients using await would surface as "object is not awaitable"
// or ".then is not a function" instead of a structured rejection.
//
// Takes message (string) which is the rejection reason propagated to the JS
// caller.
//
// Returns js.Value which is a JavaScript Promise rejected with an
// errorResult-shaped value.
func rejectedPromise(message string) js.Value {
	handler := js.FuncOf(func(_ js.Value, arguments []js.Value) any {
		reject := arguments[1]
		reject.Invoke(errorResult(message))
		return nil
	})
	defer handler.Release()

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// parseRequestSafely unmarshals the first JS argument into target with
// panic recovery.
//
// Untrusted JS values can trigger panics in syscall/js when JSON.stringify
// rejects values like BigInts or circular references; without recovery
// those panics tear down the WASM instance instead of returning a
// structured error to the caller.
//
// Takes component (string) which identifies the JS handler for diagnostic
// logging when a panic is recovered.
// Takes arguments ([]js.Value) which is the slice of JS values supplied to the
// handler.
// Takes target (any) which is a pointer to the destination struct for
// unmarshalling.
// Takes missingArgMessage (string) which is the error message returned when no
// argument was provided.
//
// Returns string which is the error message describing why the prologue could
// not proceed (missing argument, unmarshal failure, or recovered panic); the
// caller must wrap it in a rejected Promise via rejectedPromise so JS callers
// awaiting the handler observe a thenable. Returns "" when target was
// populated successfully and the caller may continue.
func parseRequestSafely(component string, arguments []js.Value, target any, missingArgMessage string) string {
	var errMessage string
	if panicMessage, panicked := wasmrecover.Sync(component, func() {
		if len(arguments) < 1 {
			errMessage = missingArgMessage
			return
		}
		if err := unmarshalJSValue(arguments[0], target); err != nil {
			errMessage = fmt.Sprintf(invalidRequestFmt, err)
			return
		}
	}); panicked {
		return panicMessage
	}
	return errMessage
}

// runSyncSafely runs operation under a deferred recover that turns any panic
// into an errorResult addressed to JS land. Used by the synchronous prologue
// of every JS handler so a malformed JS argument cannot tear down the WASM
// instance.
//
// Takes component (string) which identifies the JS handler for diagnostic
// logging.
// Takes operation (func()) which is the synchronous body to run; it should
// store its result in an enclosing variable.
//
// Returns *js.Value which is a non-nil pointer to a recovered errorResult
// when operation panicked, or nil on a clean run.
func runSyncSafely(component string, operation func()) *js.Value {
	if errMessage, panicked := wasmrecover.Sync(component, operation); panicked {
		r := errorResult(errMessage)
		return &r
	}
	return nil
}
