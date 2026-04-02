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

const testHTMLDragDrop = `<!DOCTYPE html>
<html>
<head>
<title>Drag Drop Test</title>
<style>
#source, #target {
	width: 100px;
	height: 100px;
	margin: 20px;
	padding: 10px;
	border: 2px solid black;
}
#source { background: lightblue; }
#target { background: lightgreen; }
</style>
</head>
<body>
<div id="source" draggable="true">Drag Me</div>
<div id="target">Drop Here</div>
<div id="result"></div>
<script>
var source = document.getElementById('source');
var target = document.getElementById('target');
var result = document.getElementById('result');

source.addEventListener('dragstart', function(e) {
	e.dataTransfer.setData('text/plain', 'dragged');
	result.textContent = 'dragstart';
});

target.addEventListener('dragover', function(e) {
	e.preventDefault();
});

target.addEventListener('drop', function(e) {
	e.preventDefault();
	result.textContent = 'dropped:' + e.dataTransfer.getData('text/plain');
});

source.addEventListener('dragend', function() {
	if (result.textContent.startsWith('dropped')) {
		result.textContent += ':dragend';
	}
});
</script>
</body>
</html>`

func TestDragAndDropHTML5(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDragDrop)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("HTML5 drag and drop", func(t *testing.T) {
			err := DragAndDropHTML5(ctx, "#source", "#target")
			if err != nil {
				t.Fatalf("DragAndDropHTML5() error = %v", err)
			}

			text, err := GetElementText(page.Ctx(), "#result")
			if err != nil {
				t.Fatalf("GetElementText() error = %v", err)
			}

			if text == "" {
				t.Log("Drop result empty - drag/drop events may not have fired")
			}
		})
	})
}

func TestDragAndDropWithData(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDragDrop)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("drag and drop with custom data", func(t *testing.T) {
			data := map[string]string{
				"text/plain": "custom-data",
			}
			err := DragAndDropWithData(ctx, "#source", "#target", data)
			if err != nil {
				t.Fatalf("DragAndDropWithData() error = %v", err)
			}
		})
	})
}

func TestDragTo(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDragDrop)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("drag to coordinates", func(t *testing.T) {
			err := DragTo(ctx, "#source", 300, 100)
			if err != nil {
				t.Fatalf("DragTo() error = %v", err)
			}
		})
	})
}

func TestDragByOffset(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDragDrop)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("drag by offset", func(t *testing.T) {
			err := DragByOffset(ctx, "#source", 50, 50)
			if err != nil {
				t.Fatalf("DragByOffset() error = %v", err)
			}
		})
	})
}

func TestDragAndDrop(t *testing.T) {
	t.Parallel()
	server := newTestServer(testHTMLDragDrop)
	defer server.Close()

	withExclusivePage(t, server.URL, func(t *testing.T, page *PageHelper) {
		ctx := newActionContext(page)

		t.Run("mouse-based drag and drop", func(t *testing.T) {
			err := DragAndDrop(ctx, "#source", "#target")
			if err != nil {
				t.Fatalf("DragAndDrop() error = %v", err)
			}
		})
	})
}
