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

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"piko.sh/json"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/textlimit"
)

const (
	// formStoreTimeout bounds one direct SQLite read.
	formStoreTimeout = 3 * time.Second

	// freshFormTimeout bounds one fresh server-rendered page read.
	freshFormTimeout = 4 * time.Second

	// freshFormLimit is the largest accepted server-rendered page in bytes.
	freshFormLimit = 1 << 20

	// formRowLimit is the largest number of workflow rows.
	formRowLimit = 16

	// formStoreStderrBytes bounds the SQLite reader's stderr quoted in an error.
	formStoreStderrBytes = 512

	// formQuery reads bounded workflow rows through read-only SQLite.
	formQuery = `import json,pathlib,sqlite3,sys
db=sqlite3.connect(pathlib.Path(sys.argv[1]).resolve().as_uri()+"?mode=ro",uri=True,timeout=1)
try:
    rows=[]
    for i,label,done in db.execute("SELECT id,substr(label,1,257),done FROM tasks ORDER BY id LIMIT 17"):
        if done not in (0,1) or len(rows)>=16 or len(label)>256: raise ValueError("invalid task row")
        rows.append(dict(ID=i,Label=label,Done=bool(done)))
    print(json.dumps(rows))
finally: db.close()`
)

var (
	// errMissingWorkflowDatabase reports an absent per-mode database path.
	errMissingWorkflowDatabase = errors.New("missing per-mode database path")

	// errFormStoreTimeout is the cause recorded when a direct SQLite read runs out of time.
	errFormStoreTimeout = fmt.Errorf("direct SQLite read timed out: %w", context.DeadlineExceeded)

	// errFreshFormTooLarge reports a server-rendered page above the read limit.
	errFreshFormTooLarge = errors.New("fresh page exceeds 1 MiB")

	// errInvalidFormRows reports malformed or incomplete rendered rows.
	errInvalidFormRows = errors.New("invalid SSR workflow rows")
)

// readFormStore reads workflow rows through an independent read-only SQLite connection.
//
// Returns []casefile.FormRow which lists form row values in their recorded order.
// Returns error which reports a SQLite read, process or row-decoding failure, or nil on
// success.
func readFormStore(ctx context.Context) ([]casefile.FormRow, error) {
	path := os.Getenv(envWorkflowDatabase)
	if path == "" {
		return nil, errMissingWorkflowDatabase
	}
	short, cancel := context.WithTimeoutCause(ctx, formStoreTimeout, errFormStoreTimeout)
	defer cancel()
	output, err := exec.CommandContext(short, "python3", "-I", "-c", formQuery, path).Output() //nolint:gosec // fixed python3 argv; harness-owned path argument
	if err != nil {
		if cause := context.Cause(short); cause != nil {
			err = fmt.Errorf("%w: %w", cause, err)
		}
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return nil, fmt.Errorf("reading workflow store: %w: %s", err, textlimit.Truncate(string(exitErr.Stderr), formStoreStderrBytes))
		}
		return nil, fmt.Errorf("reading workflow store: %w", err)
	}
	var rows []casefile.FormRow
	if err = json.Unmarshal(output, &rows); err != nil {
		return nil, fmt.Errorf("decoding workflow store rows: %w", err)
	}
	return rows, nil
}

// readFreshForm reads server-rendered rows with a fresh HTTP transport.
//
// Takes address (string) which is the server-rendered workflow list URL.
//
// Returns []casefile.FormRow which lists form row values in their recorded order.
// Returns error which reports an HTTP read, response-size or rendered-row failure, or nil
// on success.
func readFreshForm(ctx context.Context, address string) (rows []casefile.FormRow, err error) {
	transport := &http.Transport{}
	defer transport.CloseIdleConnections()
	client := http.Client{Transport: transport, Timeout: freshFormTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Cache-Control", "no-cache")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, drainAndClose(response.Body))
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fresh page returned HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, freshFormLimit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > freshFormLimit {
		return nil, errFreshFormTooLarge
	}
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return formRows(document)
}

// drainAndClose discards a bounded remainder of a response body and closes it so the
// connection can be reused.
//
// Takes body (io.ReadCloser) which is the response body.
//
// Returns error which reports a drain or close failure, or nil on success.
func drainAndClose(body io.ReadCloser) error {
	_, drainErr := io.Copy(io.Discard, io.LimitReader(body, freshFormLimit))
	return errors.Join(drainErr, body.Close())
}

// nodeAttribute finds a named HTML attribute.
//
// Takes node (*html.Node) which is the HTML node to inspect.
// Takes key (string) which names the HTML attribute.
//
// Returns string which is the attribute value.
// Returns bool which reports whether the attribute exists.
func nodeAttribute(node *html.Node, key string) (string, bool) {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val, true
		}
	}
	return "", false
}

// formRows extracts workflow rows from a server-rendered document.
//
// Takes document (*html.Node) which is the independently fetched list document.
//
// Returns []casefile.FormRow which lists form row values in their recorded order.
// Returns error which reports malformed or incomplete rendered row values, or nil on
// success.
func formRows(document *html.Node) ([]casefile.FormRow, error) {
	var rows []casefile.FormRow
	found := false
	for node := range document.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if id, _ := nodeAttribute(node, "id"); id == "workflow-form" {
			found = true
		}
		id, ok := nodeAttribute(node, "data-fuzz-task")
		if !ok {
			continue
		}
		row, err := formRow(node, id)
		if err != nil {
			return nil, err
		}
		if len(rows) >= formRowLimit {
			return nil, fmt.Errorf("%w: more than %d rows", errInvalidFormRows, formRowLimit)
		}
		rows = append(rows, row)
	}
	if !found {
		return nil, fmt.Errorf("%w: fresh SSR page has no workflow form", errInvalidFormRows)
	}
	return rows, nil
}

// formRow reads one rendered workflow row.
//
// Takes node (*html.Node) which is the row element.
// Takes id (string) which is the row's task identity attribute.
//
// Returns casefile.FormRow which is the row's values.
// Returns error which reports a malformed identity, completion state or label, or nil on
// success.
func formRow(node *html.Node, id string) (casefile.FormRow, error) {
	var row casefile.FormRow
	rowID, err := strconv.Atoi(id)
	if err != nil || rowID <= 0 {
		return row, fmt.Errorf("%w: task identity %q", errInvalidFormRows, id)
	}
	done, ok := nodeAttribute(node, "data-done")
	if !ok || (done != "true" && done != "false") {
		return row, fmt.Errorf("%w: task %d completion state", errInvalidFormRows, rowID)
	}
	label := taskLabel(node)
	if label == nil {
		return row, fmt.Errorf("%w: task %d has no label", errInvalidFormRows, rowID)
	}
	row.ID = rowID
	row.Label = nodeText(label)
	row.Done = done == "true"
	return row, nil
}

// taskLabel finds the label element of a rendered workflow row.
//
// Takes node (*html.Node) which is the row element.
//
// Returns *html.Node which is the first labelled descendant, or nil when there is none.
func taskLabel(node *html.Node) *html.Node {
	for descendant := range node.Descendants() {
		if _, ok := nodeAttribute(descendant, "data-task-label"); ok {
			return descendant
		}
	}
	return nil
}

// nodeText joins the text of every text node below a node.
//
// Takes node (*html.Node) which is the element to read.
//
// Returns string which is the concatenated text in document order.
func nodeText(node *html.Node) string {
	var text strings.Builder
	for descendant := range node.Descendants() {
		if descendant.Type == html.TextNode {
			text.WriteString(descendant.Data)
		}
	}
	return text.String()
}
