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

package wasm_adapters

import (
	"fmt"
	"syscall/js"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/wasm/wasm_domain"
)

var _ wasm_domain.JSInteropPort = (*jsInterop)(nil)
var _ wasm_domain.ConsolePort = (*jsConsole)(nil)

// jsInterop provides JavaScript interoperability via syscall/js.
type jsInterop struct {
	// global is the JavaScript global object used to evaluate expressions.
	global js.Value
}

// RegisterFunction registers a Go function to be callable from JavaScript.
// The function is exposed on the global 'piko' object.
//
// Takes name (string) which specifies the name to expose the function as in
// JavaScript.
// Takes handler (func(...)) which is the Go function to call when invoked from
// JavaScript.
func (j *jsInterop) RegisterFunction(name string, handler func(arguments []any) (any, error)) {
	piko := j.global.Get("piko")
	if piko.IsUndefined() {
		piko = js.Global().Get("Object").New()
		j.global.Set("piko", piko)
	}

	wrapper := js.FuncOf(func(this js.Value, jsArgs []js.Value) any {
		goArgs := make([]any, len(jsArgs))
		for i, arg := range jsArgs {
			goArgs[i] = jsValueToGo(arg)
		}

		result, err := handler(goArgs)
		if err != nil {
			return map[string]any{
				"error": err.Error(),
			}
		}

		return goToJSValue(result)
	})

	piko.Set(name, wrapper)
}

// Log writes a message to the JavaScript console.
//
// Takes level (string) which specifies the console method to use (debug, info,
// warn, error, or log for any other value).
// Takes message (string) which is the main log message.
// Takes arguments (...any) which provides optional values appended to the message.
func (j *jsInterop) Log(level string, message string, arguments ...any) {
	console := j.global.Get("console")
	if console.IsUndefined() {
		return
	}

	formattedMessage := message
	if len(arguments) > 0 {
		formattedMessage = fmt.Sprintf("%s %v", message, arguments)
	}

	switch level {
	case "debug":
		console.Call("debug", formattedMessage)
	case "info":
		console.Call("info", formattedMessage)
	case "warn":
		console.Call("warn", formattedMessage)
	case "error":
		console.Call("error", formattedMessage)
	default:
		console.Call("log", formattedMessage)
	}
}

// MarshalToJS converts a Go value to a JS-compatible representation.
//
// Takes v (any) which is the Go value to convert.
//
// Returns any which is the JavaScript object parsed from the JSON encoding.
// Returns error when the value cannot be marshalled to JSON.
func (j *jsInterop) MarshalToJS(v any) (any, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	jsonParse := j.global.Get("JSON").Get("parse")
	return jsonParse.Invoke(string(jsonBytes)), nil
}

// UnmarshalFromJS converts a JS value to a Go type.
//
// Takes jsValue (any) which is the JavaScript value to convert.
// Takes target (any) which is a pointer to the Go value to populate.
//
// Returns error when jsValue is not a js.Value or unmarshalling fails.
func (j *jsInterop) UnmarshalFromJS(jsValue any, target any) error {
	jsVal, ok := jsValue.(js.Value)
	if !ok {
		return fmt.Errorf("expected js.Value, got %T", jsValue)
	}

	jsonStringify := js.Global().Get("JSON").Get("stringify")
	jsonString := jsonStringify.Invoke(jsVal).String()

	return json.Unmarshal([]byte(jsonString), target)
}

// jsConsole implements ConsolePort using the JavaScript console.
type jsConsole struct {
	// console is the JavaScript console object used for logging.
	console js.Value
}

// Debug logs a debug message to the JS console.
//
// Takes message (string) which is the message format string.
// Takes arguments (...any) which are the values to format into the message.
func (c *jsConsole) Debug(message string, arguments ...any) {
	c.log("debug", message, arguments...)
}

// Info logs an info message to the JS console.
//
// Takes message (string) which is the message format string.
// Takes arguments (...any) which are the values to interpolate into the message.
func (c *jsConsole) Info(message string, arguments ...any) {
	c.log("info", message, arguments...)
}

// Warn logs a warning message to the JS console.
//
// Takes message (string) which specifies the message format string.
// Takes arguments (...any) which provides values to interpolate into the message.
func (c *jsConsole) Warn(message string, arguments ...any) {
	c.log("warn", message, arguments...)
}

// Error logs an error message to the JS console.
//
// Takes message (string) which is the format string for the error message.
// Takes arguments (...any) which are the values to substitute into the format.
func (c *jsConsole) Error(message string, arguments ...any) {
	c.log("error", message, arguments...)
}

// log writes a formatted message to the JavaScript console at the given level.
//
// Takes level (string) which specifies the console method to call.
// Takes message (string) which is the message to log.
// Takes arguments (...any) which are optional values appended to the message.
func (c *jsConsole) log(level, message string, arguments ...any) {
	if c.console.IsUndefined() {
		return
	}

	formattedMessage := message
	if len(arguments) > 0 {
		formattedMessage = fmt.Sprintf("%s %v", message, arguments)
	}

	c.console.Call(level, formattedMessage)
}

// NewJSConsole creates a new JavaScript console adapter.
//
// Returns wasm_domain.ConsolePort which wraps the browser console for logging.
func NewJSConsole() wasm_domain.ConsolePort {
	return &jsConsole{
		console: js.Global().Get("console"),
	}
}

// newJSInterop creates a new JavaScript interop adapter.
//
// Returns *jsInterop which provides functions for working with JavaScript.
func newJSInterop() *jsInterop {
	return &jsInterop{
		global: js.Global(),
	}
}

// jsValueToGo converts a JavaScript value to its Go equivalent.
//
// Takes v (js.Value) which is the JavaScript value to convert.
//
// Returns any which is the matching Go type: nil for undefined or null, bool
// for booleans, float64 for numbers, string for strings, []any for arrays, or
// map[string]any for objects.
func jsValueToGo(v js.Value) any {
	switch v.Type() {
	case js.TypeUndefined, js.TypeNull:
		return nil
	case js.TypeBoolean:
		return v.Bool()
	case js.TypeNumber:
		return v.Float()
	case js.TypeString:
		return v.String()
	case js.TypeObject:
		if js.Global().Get("Array").Call("isArray", v).Bool() {
			length := v.Length()
			arr := make([]any, length)
			for i := range length {
				arr[i] = jsValueToGo(v.Index(i))
			}
			return arr
		}
		return jsObjectToMap(v)
	default:
		return nil
	}
}

// jsObjectToMap converts a JavaScript object to a Go map.
//
// Takes v (js.Value) which is the JavaScript object to convert.
//
// Returns map[string]any which contains the converted key-value pairs, or nil
// if the conversion fails.
func jsObjectToMap(v js.Value) map[string]any {
	jsonStringify := js.Global().Get("JSON").Get("stringify")
	jsonString := jsonStringify.Invoke(v).String()

	var result map[string]any
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil
	}
	return result
}

// goToJSValue converts a Go value to a JavaScript value.
//
// Takes v (any) which is the Go value to convert.
//
// Returns any which is the JavaScript form of the value.
func goToJSValue(v any) any {
	if v == nil {
		return js.Null()
	}

	switch value := v.(type) {
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, string:
		return value
	case []byte:
		arr := js.Global().Get("Uint8Array").New(len(value))
		js.CopyBytesToJS(arr, value)
		return arr
	default:
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		jsonParse := js.Global().Get("JSON").Get("parse")
		return jsonParse.Invoke(string(jsonBytes))
	}
}
