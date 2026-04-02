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
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/daemon/daemon_frontend"
)

var pikoAssetStoreOnce sync.Once

func initPikoAssetStore() error {
	var initErr error
	pikoAssetStoreOnce.Do(func() {
		initErr = daemon_frontend.InitAssetStore(context.Background())
	})
	return initErr
}

func newPikoTestServer(html string) (*httptest.Server, error) {
	if err := initPikoAssetStore(); err != nil {
		return nil, err
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.HasPrefix(r.URL.Path, "/_piko/dist/") {
			servePikoAsset(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	})), nil
}

func servePikoAsset(w http.ResponseWriter, r *http.Request) {

	fileParam := strings.TrimPrefix(r.URL.Path, "/_piko/dist/")
	basePath := path.Join("built", fileParam)

	finalPath := daemon_frontend.DetermineBestAssetPath(r.Context(), basePath, r.Header.Get("Accept-Encoding"))

	asset, found := daemon_frontend.GetAsset(r.Context(), finalPath)
	if !found {
		asset, found = daemon_frontend.GetAsset(r.Context(), basePath)
		if !found {
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("ETag", asset.ETag)
	w.Header().Set("Cache-Control", "public, no-cache")

	if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, asset.ETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", asset.MimeType)
	if asset.Encoding != "" {
		w.Header().Set("Content-Encoding", asset.Encoding)
	}
	w.Header().Add("Vary", "Accept-Encoding")

	http.ServeContent(w, r, path.Base(basePath), time.Time{}, bytes.NewReader(asset.Content))
}

const (
	testHTMLPikoBusEvent = `<!DOCTYPE html>
<html>
<head>
<title>Piko Bus Event Test</title>
<script type="module">
import { piko } from '/_piko/dist/ppframework.core.es.js';
piko.bus.on('test-event', (detail) => {
    document.getElementById('result').textContent = JSON.stringify(detail);
});
window.pikoBus = piko.bus;
</script>
</head>
<body>
<div id="result"></div>
</body>
</html>`
	testHTMLPikoPartial = `<!DOCTYPE html>
<html>
<head>
<title>Piko Partial Test</title>
<script type="module">
import { piko } from '/_piko/dist/ppframework.core.es.js';
window.pikoPartial = piko.partial;
window.pikoBus = piko.bus;
piko.bus.on('partial:reloaded', (detail) => {
    document.getElementById('reload-log').textContent = 'reloaded:' + detail.name;
});
</script>
</head>
<body>
<div pk-partial="test-partial">
    <span id="partial-content">Initial Content</span>
</div>
<div id="reload-log"></div>
</body>
</html>`
	testHTMLPikoComplete = `<!DOCTYPE html>
<html>
<head>
<title>Piko Complete Test</title>
<script type="module">
import { piko } from '/_piko/dist/ppframework.core.es.js';
window.pikoBus = piko.bus;
window.pikoPartial = piko.partial;
window.pikoOnCleanup = piko.onCleanup;
window.eventLog = [];
piko.bus.on('*', (detail, eventName) => {
    window.eventLog.push({ event: eventName, detail: detail });
    document.getElementById('event-log').textContent = JSON.stringify(window.eventLog);
});
piko.bus.on('custom-event', (detail) => {
    document.getElementById('custom-result').textContent = JSON.stringify(detail);
});
</script>
</head>
<body>
<div partial="main-partial" data-partial-name="main-partial">
    <span id="partial-content">Main Content</span>
</div>
<div id="custom-result"></div>
<div id="event-log">[]</div>
<button id="emit-btn" onclick="window.pikoBus.emit('button-clicked', {source: 'button'})">Emit Event</button>
</body>
</html>`
)
