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

package scripts

import (
	"bytes"
	"embed"
	"fmt"
	"strconv"
	"sync"
	"text/template"
)

var (
	// scriptsFS holds the embedded JavaScript files and templates.
	//
	//go:embed *.js *.js.tmpl
	scriptsFS embed.FS

	templateCache = make(map[string]*template.Template)

	templateCacheMu sync.RWMutex

	// templateFuncs provides functions available in JS templates.
	templateFuncs = template.FuncMap{
		"jsStr": func(s string) string {
			return strconv.Quote(s)
		},
		"jsInt": func(i int64) string {
			return strconv.FormatInt(i, 10)
		},
		"jsFloat": func(f float64) string {
			return strconv.FormatFloat(f, 'f', -1, 64)
		},
		"jsBool": func(b bool) string {
			if b {
				return "true"
			}
			return "false"
		},
		"jsRaw": func(s string) string {
			return s
		},
	}
)

// Get returns the contents of a static JavaScript file.
//
// Takes name (string) which specifies the filename to read from the embedded
// filesystem.
//
// Returns string which contains the file contents.
//
// Panics if the file cannot be read from the embedded filesystem.
func Get(name string) string {
	data, err := scriptsFS.ReadFile(name)
	if err != nil {
		panic(fmt.Sprintf("scripts: failed to read %s: %v", name, err))
	}
	return string(data)
}

// MustGet returns the contents of a static JavaScript file by name.
// This is an alias for Get for semantic clarity.
//
// Takes name (string) which specifies the JavaScript file to retrieve.
//
// Returns string which contains the file contents.
func MustGet(name string) string {
	return Get(name)
}

// Execute runs a named JavaScript template with the given data and returns
// the result.
//
// Templates use Go's text/template syntax with the following functions:
//   - {{jsStr .Value}} - safely quotes a string for JS
//   - {{jsInt .Value}} - formats an integer
//   - {{jsFloat .Value}} - formats a float
//   - {{jsBool .Value}} - formats a boolean
//   - {{jsRaw .Value}} - passes through raw JS code
//
// Takes name (string) which identifies the template to execute.
// Takes data (any) which provides the data passed to the template.
//
// Returns string which contains the rendered template output.
// Returns error when the template is not found or execution fails.
func Execute(name string, data any) (string, error) {
	parsedTemplate, err := getTemplate(name)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := parsedTemplate.Execute(&buffer, data); err != nil {
		return "", fmt.Errorf("scripts: failed to execute %s: %w", name, err)
	}
	return buffer.String(), nil
}

// MustExecute executes a JavaScript template, panicking on error.
//
// Takes name (string) which identifies the template to execute.
// Takes data (any) which provides the data to pass to the template.
//
// Returns string which contains the executed template result.
//
// Panics if the template execution fails.
func MustExecute(name string, data any) string {
	result, err := Execute(name, data)
	if err != nil {
		panic(err)
	}
	return result
}

// getTemplate returns a cached or newly parsed template.
//
// Takes name (string) which specifies the template file to retrieve.
//
// Returns *template.Template which is the parsed template ready for use.
// Returns error when the file cannot be read or the template fails to parse.
//
// Safe for concurrent use. Uses a read-write mutex with double-checked
// locking to protect the template cache.
func getTemplate(name string) (*template.Template, error) {
	templateCacheMu.RLock()
	if cachedTemplate, ok := templateCache[name]; ok {
		templateCacheMu.RUnlock()
		return cachedTemplate, nil
	}
	templateCacheMu.RUnlock()

	templateCacheMu.Lock()
	defer templateCacheMu.Unlock()

	if cachedTemplate, ok := templateCache[name]; ok {
		return cachedTemplate, nil
	}

	data, err := scriptsFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("scripts: failed to read %s: %w", name, err)
	}

	parsedTemplate, err := template.New(name).Funcs(templateFuncs).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("scripts: failed to parse %s: %w", name, err)
	}

	templateCache[name] = parsedTemplate
	return parsedTemplate, nil
}
