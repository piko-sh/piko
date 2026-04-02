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
	"net/http"
	"net/http/httptest"
)

func newTestServer(html string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
}

func newTestServerWithRoutes(routes map[string]string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if html, ok := routes[r.URL.Path]; ok {
			_, _ = w.Write([]byte(html))
			return
		}
		http.NotFound(w, r)
	}))
}

const (
	testHTMLEmpty = `<!DOCTYPE html>
<html><head><title>Test</title></head><body></body></html>`
	testHTMLButton = `<!DOCTYPE html>
<html>
<head><title>Button Test</title></head>
<body>
<button id="btn">Click Me</button>
<div id="result"></div>
<script>
document.getElementById('btn').addEventListener('click', function() {
    document.getElementById('result').textContent = 'clicked';
});
</script>
</body>
</html>`
	testHTMLDoubleClickButton = `<!DOCTYPE html>
<html>
<head><title>Double Click Test</title></head>
<body>
<button id="btn">Double Click Me</button>
<div id="result"></div>
<script>
document.getElementById('btn').addEventListener('dblclick', function() {
    document.getElementById('result').textContent = 'double-clicked';
});
</script>
</body>
</html>`
	testHTMLRightClickButton = `<!DOCTYPE html>
<html>
<head><title>Right Click Test</title></head>
<body>
<button id="btn">Right Click Me</button>
<div id="result"></div>
<script>
document.getElementById('btn').addEventListener('contextmenu', function(e) {
    e.preventDefault();
    document.getElementById('result').textContent = 'right-clicked';
});
</script>
</body>
</html>`
	testHTMLInput = `<!DOCTYPE html>
<html>
<head><title>Input Test</title></head>
<body>
<input type="text" id="input" />
<div id="mirror"></div>
<script>
document.getElementById('input').addEventListener('input', function(e) {
    document.getElementById('mirror').textContent = e.target.value;
});
</script>
</body>
</html>`
	testHTMLKeyboard = `<!DOCTYPE html>
<html>
<head><title>Keyboard Test</title></head>
<body>
<input type="text" id="input" />
<div id="keylog"></div>
<script>
document.getElementById('input').addEventListener('keydown', function(e) {
    var key = e.key;
    if (e.ctrlKey) key = 'Ctrl+' + key;
    if (e.shiftKey) key = 'Shift+' + key;
    if (e.altKey) key = 'Alt+' + key;
    document.getElementById('keylog').textContent = key;
});
</script>
</body>
</html>`
	testHTMLCheckbox = `<!DOCTYPE html>
<html>
<head><title>Checkbox Test</title></head>
<body>
<input type="checkbox" id="checkbox1" />
<label for="checkbox1">Checkbox 1</label>
<input type="checkbox" id="checkbox2" checked />
<label for="checkbox2">Checkbox 2</label>
<div id="status"></div>
<script>
function updateStatus() {
    var c1 = document.getElementById('checkbox1').checked;
    var c2 = document.getElementById('checkbox2').checked;
    document.getElementById('status').textContent = 'c1:' + c1 + ',c2:' + c2;
}
document.getElementById('checkbox1').addEventListener('change', updateStatus);
document.getElementById('checkbox2').addEventListener('change', updateStatus);
updateStatus();
</script>
</body>
</html>`
	testHTMLShadowDOM = `<!DOCTYPE html>
<html>
<head><title>Shadow DOM Test</title></head>
<body>
<div id="host"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = '<span id="inner">Shadow Content</span><button id="shadow-btn">Shadow Button</button>';
shadow.getElementById('shadow-btn').addEventListener('click', function() {
    shadow.getElementById('inner').textContent = 'Shadow Clicked';
});
</script>
</body>
</html>`
	testHTMLVisibility = `<!DOCTYPE html>
<html>
<head>
<title>Visibility Test</title>
<style>
#hidden-display { display: none; }
#hidden-visibility { visibility: hidden; }
#hidden-opacity { opacity: 0; }
#visible { display: block; }
</style>
</head>
<body>
<div id="hidden-display">Hidden by display</div>
<div id="hidden-visibility">Hidden by visibility</div>
<div id="hidden-opacity">Hidden by opacity</div>
<div id="visible">Visible element</div>
</body>
</html>`
	testHTMLAttributes = `<!DOCTYPE html>
<html>
<head><title>Attributes Test</title></head>
<body>
<div id="target"
     data-custom="custom-value"
     class="class1 class2"
     title="Element Title">
    Content Text
</div>
<input id="disabled-input" type="text" disabled value="disabled" />
<input id="enabled-input" type="text" value="enabled" />
<a id="link" href="https://example.com">Link</a>
</body>
</html>`
	testHTMLConsole = `<!DOCTYPE html>
<html>
<head><title>Console Test</title></head>
<body>
<button id="log-btn" onclick="console.log('log message')">Log</button>
<button id="warn-btn" onclick="console.warn('warn message')">Warn</button>
<button id="error-btn" onclick="console.error('error message')">Error</button>
<button id="info-btn" onclick="console.info('info message')">Info</button>
</body>
</html>`
	testHTMLContentEditable = `<!DOCTYPE html>
<html>
<head><title>ContentEditable Test</title></head>
<body>
<div id="editor" contenteditable="true">Hello World</div>
<div id="selection-info"></div>
<script>
document.getElementById('editor').addEventListener('selectionchange', function() {
    var sel = window.getSelection();
    if (sel.rangeCount > 0) {
        document.getElementById('selection-info').textContent =
            'offset:' + sel.anchorOffset + '-' + sel.focusOffset;
    }
});
</script>
</body>
</html>`
	testHTMLMultipleElements = `<!DOCTYPE html>
<html>
<head><title>Multiple Elements Test</title></head>
<body>
<ul id="list">
    <li class="item">Item 1</li>
    <li class="item">Item 2</li>
    <li class="item">Item 3</li>
    <li class="item special">Item 4</li>
    <li class="item special">Item 5</li>
</ul>
<div class="box">Box 1</div>
<div class="box">Box 2</div>
</body>
</html>`
	testHTMLHover = `<!DOCTYPE html>
<html>
<head><title>Hover Test</title></head>
<body>
<div id="hover-target">Hover over me</div>
<div id="hover-result"></div>
<script>
document.getElementById('hover-target').addEventListener('mouseenter', function() {
    document.getElementById('hover-result').textContent = 'hovered';
});
document.getElementById('hover-target').addEventListener('mouseleave', function() {
    document.getElementById('hover-result').textContent = 'left';
});
</script>
</body>
</html>`
	testHTMLFocus = `<!DOCTYPE html>
<html>
<head><title>Focus Test</title></head>
<body>
<input type="text" id="input1" />
<input type="text" id="input2" />
<div id="focus-log"></div>
<script>
document.getElementById('input1').addEventListener('focus', function() {
    document.getElementById('focus-log').textContent = 'input1-focused';
});
document.getElementById('input1').addEventListener('blur', function() {
    document.getElementById('focus-log').textContent = 'input1-blurred';
});
document.getElementById('input2').addEventListener('focus', function() {
    document.getElementById('focus-log').textContent = 'input2-focused';
});
</script>
</body>
</html>`
	testHTMLScroll = `<!DOCTYPE html>
<html>
<head>
<title>Scroll Test</title>
<style>
.spacer { height: 2000px; }
#target { background: yellow; padding: 20px; }
</style>
</head>
<body>
<div class="spacer"></div>
<div id="target">Target Element</div>
<div class="spacer"></div>
</body>
</html>`
	testHTMLFileInput = `<!DOCTYPE html>
<html>
<head><title>File Input Test</title></head>
<body>
<input type="file" id="file-input" />
<div id="file-name"></div>
<script>
document.getElementById('file-input').addEventListener('change', function(e) {
    if (e.target.files.length > 0) {
        document.getElementById('file-name').textContent = e.target.files[0].name;
    }
});
</script>
</body>
</html>`
	testHTMLCustomEvent = `<!DOCTYPE html>
<html>
<head><title>Custom Event Test</title></head>
<body>
<div id="target">Event Target</div>
<div id="event-data"></div>
<script>
document.getElementById('target').addEventListener('custom-event', function(e) {
    document.getElementById('event-data').textContent = JSON.stringify(e.detail);
});
</script>
</body>
</html>`
	testHTMLShadowDOMComprehensive = `<!DOCTYPE html>
<html>
<head><title>Shadow DOM Comprehensive Test</title></head>
<body>
<div id="host"></div>
<div id="result"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = ` + "`" + `
    <style>
        .log { margin: 5px 0; }
    </style>
    <span id="inner">Shadow Content</span>
    <button id="shadow-btn">Shadow Button</button>
    <button id="shadow-dblclick-btn">Double Click</button>
    <button id="shadow-rightclick-btn">Right Click</button>
    <input type="text" id="shadow-input" value="" placeholder="Shadow Input" />
    <input type="checkbox" id="shadow-checkbox" />
    <input type="checkbox" id="shadow-checkbox-checked" checked />
    <div id="shadow-hover-target">Hover Target</div>
    <input type="text" id="shadow-focus-input" placeholder="Focus me" />
    <form id="shadow-form">
        <input type="text" name="name" value="test" />
        <button type="submit" id="shadow-submit">Submit</button>
    </form>
    <div id="shadow-log"></div>
` + "`" + `;
const log = (message) => {
    shadow.getElementById('shadow-log').textContent = message;
    document.getElementById('result').textContent = message;
};
shadow.getElementById('shadow-btn').addEventListener('click', () => log('clicked'));
shadow.getElementById('shadow-dblclick-btn').addEventListener('dblclick', () => log('double-clicked'));
shadow.getElementById('shadow-rightclick-btn').addEventListener('contextmenu', (e) => {
    e.preventDefault();
    log('right-clicked');
});
shadow.getElementById('shadow-input').addEventListener('input', (e) => log('input:' + e.target.value));
shadow.getElementById('shadow-checkbox').addEventListener('change', (e) => log('checkbox:' + e.target.checked));
shadow.getElementById('shadow-checkbox-checked').addEventListener('change', (e) => log('checkbox-checked:' + e.target.checked));
shadow.getElementById('shadow-hover-target').addEventListener('mouseenter', () => log('hovered'));
shadow.getElementById('shadow-hover-target').addEventListener('mouseover', () => log('hovered'));
shadow.getElementById('shadow-focus-input').addEventListener('focus', () => log('focused'));
shadow.getElementById('shadow-focus-input').addEventListener('blur', () => log('blurred'));
shadow.getElementById('shadow-form').addEventListener('submit', (e) => {
    e.preventDefault();
    log('submitted');
});
</script>
</body>
</html>`
	testHTMLShadowDOMContenteditable = `<!DOCTYPE html>
<html>
<head><title>Shadow DOM Contenteditable Test</title></head>
<body>
<div id="host"></div>
<div id="result"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = ` + "`" + `
    <style>
        .editor {
            border: 1px solid #ccc;
            padding: 10px;
            min-height: 50px;
            min-width: 200px;
        }
    </style>
    <div id="editor" class="editor" contenteditable="true">Hello World</div>
` + "`" + `;
const editor = shadow.getElementById('editor');
// Track cursor position after click
editor.addEventListener('click', () => {
    const sel = shadow.getSelection ? shadow.getSelection() : window.getSelection();
    if (sel && sel.rangeCount > 0) {
        const range = sel.getRangeAt(0);
        document.getElementById('result').textContent = 'cursor:' + range.startOffset;
    }
});
// Track input for typing tests
editor.addEventListener('input', () => {
    document.getElementById('result').textContent = 'content:' + editor.textContent;
});
</script>
</body>
</html>`
	testHTMLInlineElements = `<!DOCTYPE html>
<html>
<head><title>Inline Elements Test</title></head>
<body>
<div id="editor" contenteditable="true">Text with <strong id="bold">bold</strong> and <em id="italic">italic</em> text</div>
<div id="result"></div>
<script>
const editor = document.getElementById('editor');
// Report cursor position on selection change
document.addEventListener('selectionchange', () => {
    const sel = window.getSelection();
    if (sel && sel.rangeCount > 0 && editor.contains(sel.anchorNode)) {
        const range = sel.getRangeAt(0);
        // Find which inline element contains the cursor
        let container = range.startContainer;
        while (container && container !== editor) {
            if (container.id) {
                document.getElementById('result').textContent = 'inside:' + container.id + ':' + range.startOffset;
                return;
            }
            container = container.parentElement;
        }
        document.getElementById('result').textContent = 'plain:' + range.startOffset;
    }
});
</script>
</body>
</html>`
	testHTMLShadowDOMInlineElements = `<!DOCTYPE html>
<html>
<head><title>Shadow DOM Inline Elements Test</title></head>
<body>
<div id="host"></div>
<div id="result"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = ` + "`" + `
    <style>
        .editor {
            border: 1px solid #ccc;
            padding: 10px;
        }
    </style>
    <div id="editor" class="editor" contenteditable="true">Text with <strong id="bold">bold</strong> and <em id="italic">italic</em> text</div>
` + "`" + `;
const editor = shadow.getElementById('editor');
// Report cursor position on selection change (works for shadow DOM)
const updateResult = () => {
    const sel = shadow.getSelection ? shadow.getSelection() : window.getSelection();
    if (sel && sel.rangeCount > 0) {
        const range = sel.getRangeAt(0);
        // Check if inside our editor
        let node = range.startContainer;
        while (node && node !== shadow) {
            if (node === editor) break;
            node = node.parentNode;
        }
        if (node !== editor && node !== shadow) return;
        // Find which inline element contains the cursor
        let container = range.startContainer;
        while (container && container !== editor) {
            if (container.id) {
                document.getElementById('result').textContent = 'inside:' + container.id + ':' + range.startOffset;
                return;
            }
            container = container.parentElement;
        }
        document.getElementById('result').textContent = 'plain:' + range.startOffset;
    }
};
// Shadow DOM selectionchange is not well supported, so also listen to click/keyup
editor.addEventListener('click', () => setTimeout(updateResult, 10));
editor.addEventListener('keyup', updateResult);
document.addEventListener('selectionchange', updateResult);
</script>
</body>
</html>`
	testHTMLShadowDOMFileInput = `<!DOCTYPE html>
<html>
<head><title>Shadow DOM File Input Test</title></head>
<body>
<div id="host"></div>
<div id="result"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = ` + "`" + `
    <input type="file" id="file-input" accept="*/*" />
    <div id="file-info">No files</div>
` + "`" + `;
const fileInput = shadow.getElementById('file-input');
const fileInfo = shadow.getElementById('file-info');
fileInput.addEventListener('change', (e) => {
    const files = e.target.files;
    if (files.length === 0) {
        fileInfo.textContent = 'No files';
        document.getElementById('result').textContent = 'files:0';
    } else {
        const names = Array.from(files).map(f => f.name).join(', ');
        fileInfo.textContent = names;
        document.getElementById('result').textContent = 'files:' + files.length + ':' + names;
    }
});
// Also listen to input event for coverage
fileInput.addEventListener('input', (e) => {
    const files = e.target.files;
    if (files && files.length > 0) {
        const names = Array.from(files).map(f => f.name).join(', ');
        document.getElementById('result').textContent = 'files:' + files.length + ':' + names;
    }
});
</script>
</body>
</html>`
	testHTMLForm = `<!DOCTYPE html>
<html>
<head><title>Form Test</title></head>
<body>
<form id="test-form">
	<input type="text" name="username" value="alice">
	<input type="email" name="email" value="alice@example.com">
	<input type="hidden" name="token" value="abc123">
</form>
<form id="empty-form"></form>
</body>
</html>`
	testHTMLEventListener = `<!DOCTYPE html>
<html>
<head><title>Event Listener Test</title></head>
<body>
<div id="target">Target</div>
<button id="fire" onclick="document.getElementById('target').dispatchEvent(new CustomEvent('greeting', {detail: {message: 'hello', count: 42}}))">Fire</button>
</body>
</html>`
	testHTMLShadowEventListener = `<!DOCTYPE html>
<html>
<head><title>Shadow Event Listener Test</title></head>
<body>
<div id="host"></div>
<script>
const host = document.getElementById('host');
const shadow = host.attachShadow({mode: 'open'});
shadow.innerHTML = '<button id="inner-btn">Click</button>';
shadow.getElementById('inner-btn').addEventListener('click', function() {
	host.dispatchEvent(new CustomEvent('shadow-click', {detail: {from: 'shadow'}}));
});
</script>
</body>
</html>`
	testHTMLSerialisableShadow = `<!DOCTYPE html>
<html>
<head><title>Serialisable Shadow Root Test</title></head>
<body>
<div id="container">
<div id="shadow-host"></div>
</div>
<script>
const host = document.getElementById('shadow-host');
const shadow = host.attachShadow({mode: 'open', serializable: true});
shadow.innerHTML = '<style>:host { display: block; }</style><span id="shadow-inner">Shadow Content</span>';
</script>
</body>
</html>`
	testHTMLStaleDOMReplacement = `<!DOCTYPE html>
<html>
<head><title>Stale DOM Replacement Test</title></head>
<body>
<div id="container">
    <button id="btn">Click Me</button>
    <div id="result"></div>
    <input type="text" id="input" value="original" />
</div>
<script>
document.getElementById('btn').addEventListener('click', function() {
    document.getElementById('result').textContent = 'clicked';
});
</script>
</body>
</html>`
)
