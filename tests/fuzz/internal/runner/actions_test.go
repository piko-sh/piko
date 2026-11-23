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

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
)

const (
	fakeActionToken    = "action-token"
	fakeEphemeralToken = "ephemeral-token"
	fakeActionPage     = `<html><head><meta name="csrf-token" content="action-token"><meta name="csrf-ephemeral" content="ephemeral-token"></head><body></body></html>`
)

type fakeActionApp struct {
	subscribers map[int]chan noticePayload
	cache       map[string]CacheRead
	rendezvous  chan struct{}
	contract    *casefile.ActionContract
	work        string
	history     []noticePayload
	mu          sync.Mutex
	cancelled   int
	executions  int64
	latest      int
	nextID      int
	waiting     bool
}

type fakeActionInput struct {
	Label     string
	Operation string
	Quantity  int
}

func TestActionModelRejectsSharedWrongResults(t *testing.T) {
	contract := generate.New("http-actions", 7, 1).Expected.Action
	observation := func() *ActionObservation {
		refusals := slices.Clone(contract.Refusals)
		upload := expectedUpload(contract.Input, true)
		upload.Removed, upload.DiskAbsent, upload.DeletedPublicStatus = "payload.bin", true, http.StatusNotFound
		for index := range refusals {
			refusals[index].Errors = maps.Clone(refusals[index].Errors)
		}
		return &ActionObservation{
			Batches:       []actionBatch{expectedBatch(contract, false), expectedBatch(contract, true)},
			Notifications: expectedNotices(contract.Input),
			JSON:          ActionResponse{Status: http.StatusOK, ContentType: "application/json"},
			SSE:           ActionResponse{Status: http.StatusOK, ContentType: "text/event-stream"},
			Result:        contract.Result,
			Events:        slices.Clone(contract.Events),
			Refusals:      refusals,
			Cancel: &StreamCancellation{
				First:       casefile.ActionEvent{Name: "waiting", Value: contract.Input},
				Before:      casefile.ActionValue{Label: contract.Input.Label},
				After:       casefile.ActionValue{Label: contract.Input.Label, Quantity: 1},
				NextRequest: contract.Result,
			},
			Cache:  expectedCacheReads(contract.Input.Label),
			Upload: &upload,
			Actors: expectedActors(contract.Input),
		}
	}
	require.NoError(t, checkAction(contract, observation()))
	t.Run("saved pre-cache and pre-upload contract", func(t *testing.T) {
		legacy := *contract
		legacy.Notifications = false
		legacy.Batch = false
		legacy.Cache, legacy.Upload, legacy.UploadPublic, legacy.UploadRemoval, legacy.Actors = false, false, false, false, false
		got := observation()
		got.Cache, got.Upload, got.Actors, got.Notifications = nil, nil, nil, nil
		got.Batches = nil
		require.NoError(t, checkAction(&legacy, got))
	})
	for name, mutate := range map[string]func(*ActionObservation){
		"wrong validation field": func(o *ActionObservation) {
			delete(o.Refusals[0].Errors, "Quantity")
			o.Refusals[0].Errors["Label"] = "must be positive"
		},
		"lost denial":       func(o *ActionObservation) { o.Refusals = o.Refusals[:1] },
		"status":            func(o *ActionObservation) { o.Refusals[1].Status = http.StatusOK },
		"code":              func(o *ActionObservation) { o.Refusals[2].Code = "UNAUTHORISED" },
		"batch success":     func(o *ActionObservation) { o.Batches[0].Success = true },
		"batch early exit":  func(o *ActionObservation) { o.Batches[0].Results = o.Batches[0].Results[:2] },
		"wrong batch field": func(o *ActionObservation) { delete(o.Batches[1].Results[1].Errors, "Quantity") },
		"parallel ordering": func(o *ActionObservation) {
			results := o.Batches[1].Results
			results[0], results[2] = results[2], results[0]
		},
		"early close":           func(o *ActionObservation) { o.Cancel.First.Name = "complete" },
		"missing disconnect":    func(o *ActionObservation) { o.Cancel.After.Quantity = 0 },
		"double disconnect":     func(o *ActionObservation) { o.Cancel.After.Quantity = 2 },
		"stale receiver":        func(o *ActionObservation) { o.Cancel.NextRequest.Label = "stale" },
		"phantom cache hit":     func(o *ActionObservation) { o.Cache[1].Execution++ },
		"mixed argument":        func(o *ActionObservation) { o.Cache[2] = o.Cache[1] },
		"lost vary header":      func(o *ActionObservation) { o.Cache[4] = o.Cache[3] },
		"expired ordinary":      func(o *ActionObservation) { o.Cache[5].Outcome = "MISS" },
		"stale after expiry":    func(o *ActionObservation) { o.Cache[7] = o.Cache[6] },
		"missing read":          func(o *ActionObservation) { o.Cache = o.Cache[:8] },
		"upload binding":        func(o *ActionObservation) { o.Upload.Name = "" },
		"upload media":          func(o *ActionObservation) { o.Upload.Media = "text/plain" },
		"upload truncation":     func(o *ActionObservation) { o.Upload.Size-- },
		"upload readback":       func(o *ActionObservation) { o.Upload.Digest = "" },
		"upload disk":           func(o *ActionObservation) { o.Upload.DiskDigest = "" },
		"storage byte range":    func(o *ActionObservation) { o.Upload.RangeDigest = o.Upload.Digest },
		"public download":       func(o *ActionObservation) { o.Upload.PublicDigest = "" },
		"public status":         func(o *ActionObservation) { o.Upload.PublicStatus = http.StatusNotFound },
		"public MIME":           func(o *ActionObservation) { o.Upload.PublicMedia = "text/plain" },
		"delete result":         func(o *ActionObservation) { o.Upload.Removed = "other.bin" },
		"deleted public lookup": func(o *ActionObservation) { o.Upload.DeletedPublicStatus = http.StatusOK },
		"disk after delete":     func(o *ActionObservation) { o.Upload.DiskAbsent = false },
		"public length":         func(o *ActionObservation) { o.Upload.PublicLength-- },
		"missing subscriber":    func(o *ActionObservation) { o.Notifications.Statuses[0].Active-- },
		"wrong recipient":       func(o *ActionObservation) { o.Notifications.First[1].Text = "other" },
		"duplicate replay":      func(o *ActionObservation) { o.Notifications.Reconnect[0] = o.Notifications.Second[1] },
		"out of order": func(o *ActionObservation) {
			o.Notifications.First[2], o.Notifications.First[3] = o.Notifications.First[3], o.Notifications.First[2]
		},
		"listener leak":  func(o *ActionObservation) { o.Notifications.Statuses[4].Active = 1 },
		"actor header":   func(o *ActionObservation) { o.Actors[0].After = o.Actors[1].After },
		"actor body":     func(o *ActionObservation) { o.Actors[1].Label = o.Actors[0].Label },
		"actor cookie":   func(o *ActionObservation) { o.Actors[1].Cookie = o.Actors[0].Cookie },
		"actor receiver": func(o *ActionObservation) { o.Actors[1].Receiver = o.Actors[0].Receiver },
	} {
		t.Run(name, func(t *testing.T) {
			o := observation()
			mutate(o)
			require.NoError(t, compareActions(o, o))
			assert.Error(t, checkAction(contract, o), "wrong action result passed")
		})
	}
}

func TestActionDevelopmentDetailRequiresExactMessageAndFrame(t *testing.T) {
	const prefix = "provider daemon_adapters.actionSingle: "
	actual := &ActionObservation{Refusals: []casefile.ActionRefusal{{Message: prefix + "sign in"}}}
	clean, err := publicActionResult(actual, "interp")
	require.NoError(t, err)
	assert.Equal(t, "sign in", clean.Refusals[0].Message)
	assert.Equal(t, prefix+"sign in", actual.Refusals[0].Message)
	native, err := publicActionResult(actual, "compiled")
	require.NoError(t, err)
	assert.Equal(t, actual.Refusals[0].Message, native.Refusals[0].Message, "production lost unexpected detail")
	for _, bad := range []string{"sign in", "provider other: sign in"} {
		actual.Refusals[0].Message = bad
		_, err := publicActionResult(actual, "interp")
		assert.Error(t, err, "accepted an invalid dev response %q", bad)
	}
}

func TestAwaitCancelledStateReturnsOnceTheStateChanges(t *testing.T) {
	reads := 0
	query := func(_ string, value *casefile.ActionValue) error {
		reads++
		value.Quantity = min(reads, 3)
		return nil
	}
	o := &StreamCancellation{Before: casefile.ActionValue{Quantity: 1}}

	require.NoError(t, awaitCancelledState(t.Context(), query, o))

	assert.Equal(t, 2, o.After.Quantity)
}

func TestAwaitCancelledStateReportsParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	stopped := errors.New("campaign stopped")
	cancel(stopped)
	query := func(string, *casefile.ActionValue) error { return nil }

	err := awaitCancelledState(ctx, query, &StreamCancellation{})

	assert.ErrorIs(t, err, stopped)
}

func TestAwaitCancelledStateReportsQueryFailures(t *testing.T) {
	failed := errors.New("state read failed")
	query := func(string, *casefile.ActionValue) error { return failed }

	assert.ErrorIs(t, awaitCancelledState(t.Context(), query, &StreamCancellation{}), failed)
}

func newFakeActionServer(t *testing.T, contract *casefile.ActionContract) (*httptest.Server, *fakeActionApp) {
	t.Helper()
	app := &fakeActionApp{
		subscribers: map[int]chan noticePayload{},
		cache:       map[string]CacheRead{},
		rendezvous:  make(chan struct{}),
		contract:    contract,
		work:        t.TempDir(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /main", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set(headerContentType, "text/html")
		_, _ = io.WriteString(writer, fakeActionPage)
	})
	mux.HandleFunc("POST /_piko/actions/fuzz.Collect", app.collect)
	mux.HandleFunc("POST /_piko/actions/_batch", app.batch)
	mux.HandleFunc("POST /_piko/actions/fuzz.Snapshot", app.snapshot)
	mux.HandleFunc("POST /_piko/actions/fuzz.Store", app.store)
	mux.HandleFunc("POST /_piko/actions/fuzz.RemoveUpload", app.removeUpload)
	mux.HandleFunc("GET /_piko/storage/public/fuzz-upload/uploads/payload.bin", app.download)
	mux.HandleFunc("POST /_piko/actions/fuzz.Rendezvous", app.meet)
	mux.HandleFunc("POST /_piko/actions/fuzz.Subscribe", app.subscribe)
	mux.HandleFunc("POST /_piko/actions/fuzz.SendNotice", app.sendNotice)
	guarded := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.Header.Get("X-CSRF-Action-Token") != fakeActionToken {
			http.Error(writer, "csrf", http.StatusForbidden)
			return
		}
		mux.ServeHTTP(writer, request)
	})
	server := httptest.NewServer(guarded)
	t.Cleanup(server.Close)
	return server, app
}

func writeFakeJSON(writer http.ResponseWriter, status int, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set(headerContentType, mediaJSON)
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func writeFakeEvent(writer http.ResponseWriter, id, name string, value any) {
	payload, _ := json.Marshal(value)
	if id != "" {
		_, _ = fmt.Fprintf(writer, "id: %s\n", id)
	}
	_, _ = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", name, payload)
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (a *fakeActionApp) offset() int {
	return a.contract.Result.Quantity - a.contract.Input.Quantity
}

func (a *fakeActionApp) call(input fakeActionInput) (casefile.ActionValue, *casefile.ActionRefusal) {
	switch input.Operation {
	case "validate":
		if input.Quantity <= 0 {
			return casefile.ActionValue{}, &casefile.ActionRefusal{Code: "VALIDATION_FAILED", Message: "validation failed", Status: http.StatusUnprocessableEntity, Errors: map[string]string{"Quantity": "must be positive"}}
		}
	case "private":
		return casefile.ActionValue{}, &casefile.ActionRefusal{Code: "UNAUTHORISED", Message: "sign in", Status: http.StatusUnauthorized}
	case "restricted":
		return casefile.ActionValue{}, &casefile.ActionRefusal{Code: "FORBIDDEN", Message: "permission required", Status: http.StatusForbidden}
	case "status":
		a.mu.Lock()
		defer a.mu.Unlock()
		return casefile.ActionValue{Label: input.Label, Quantity: a.cancelled}, nil
	}
	return casefile.ActionValue{Label: input.Label, Quantity: input.Quantity + a.offset()}, nil
}

func (a *fakeActionApp) collect(writer http.ResponseWriter, request *http.Request) {
	var input fakeActionInput
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if request.Header.Get("Accept") != mediaEventStream {
		value, refusal := a.call(input)
		if refusal != nil {
			writeFakeJSON(writer, refusal.Status, refusal)
			return
		}
		writeFakeJSON(writer, http.StatusOK, value)
		return
	}
	writer.Header().Set(headerContentType, mediaEventStream)
	if input.Operation == "watch" {
		writeFakeEvent(writer, "", "waiting", casefile.ActionValue{Label: input.Label, Quantity: input.Quantity})
		<-request.Context().Done()
		a.mu.Lock()
		a.cancelled++
		a.mu.Unlock()
		return
	}
	for round := range len(a.contract.Events) - 1 {
		writeFakeEvent(writer, "", "progress", casefile.ActionValue{Label: input.Label, Quantity: input.Quantity + round})
	}
	value, _ := a.call(input)
	writeFakeEvent(writer, "", "complete", value)
}

func (a *fakeActionApp) batch(writer http.ResponseWriter, request *http.Request) {
	var envelope struct {
		Actions []struct {
			Name string
			Args fakeActionInput
		}
	}
	if err := json.NewDecoder(request.Body).Decode(&envelope); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	success := true
	results := make([]batchResult, 0, len(envelope.Actions))
	for _, item := range envelope.Actions {
		value, refusal := a.call(item.Args)
		if refusal != nil {
			success = false
			results = append(results, batchResult{Name: item.Name, Status: refusal.Status, Code: refusal.Code, Errors: refusal.Errors})
			continue
		}
		results = append(results, batchResult{Name: item.Name, Status: http.StatusOK, Data: &value})
	}
	writeFakeJSON(writer, http.StatusOK, batchEnvelope{Success: &success, Results: results})
}

func (a *fakeActionApp) snapshot(writer http.ResponseWriter, request *http.Request) {
	var input struct{ Key string }
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	language := request.Header.Get("Accept-Language")
	entry := input.Key + "|" + language
	a.mu.Lock()
	read, hit := a.cache[entry]
	if !hit {
		a.executions++
		read = CacheRead{Key: input.Key, Language: language, Execution: a.executions}
		if language != cacheLanguageGerman {
			a.cache[entry] = read
		}
	}
	a.mu.Unlock()
	writer.Header().Set("X-Action-Cache", map[bool]string{false: cacheOutcomeMiss, true: cacheOutcomeHit}[hit])
	writeFakeJSON(writer, http.StatusOK, read)
}

func (a *fakeActionApp) storedPath() string {
	return filepath.Join(a.work, storedUploadName)
}

func (a *fakeActionApp) store(writer http.ResponseWriter, request *http.Request) {
	file, header, err := request.FormFile("file")
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, uploadLimit+1))
	if err == nil {
		err = os.MkdirAll(filepath.Dir(a.storedPath()), 0o700)
	}
	if err == nil {
		err = os.WriteFile(a.storedPath(), data, 0o600)
	}
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writeFakeJSON(writer, http.StatusOK, UploadRead{
		Name:        header.Filename,
		Media:       header.Header.Get(headerContentType),
		Size:        int64(len(data)),
		Digest:      sha256Hex(data),
		RangeDigest: sha256Hex(data[2 : len(data)-1]),
	})
}

func (a *fakeActionApp) download(writer http.ResponseWriter, request *http.Request) {
	data, err := os.ReadFile(a.storedPath())
	if err != nil {
		http.NotFound(writer, request)
		return
	}
	writer.Header().Set(headerContentType, "application/octet-stream")
	writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = writer.Write(data)
}

func (a *fakeActionApp) removeUpload(writer http.ResponseWriter, request *http.Request) {
	var input struct{ Key string }
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err := os.Remove(a.storedPath()); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writeFakeJSON(writer, http.StatusOK, uploadRemoval{Removed: input.Key})
}

func (a *fakeActionApp) meet(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Label  string
		Number int64
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	before := request.Header.Get("X-Fuzz-Actor")
	a.mu.Lock()
	if a.waiting {
		close(a.rendezvous)
	}
	a.waiting = true
	a.mu.Unlock()
	select {
	case <-a.rendezvous:
	case <-request.Context().Done():
		return
	}
	cookie, err := request.Cookie("case-actor")
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	writeFakeJSON(writer, http.StatusOK, ActorRead{Label: input.Label, Before: before, After: request.Header.Get("X-Fuzz-Actor"), Cookie: cookie.Value, Number: input.Number, Receiver: input.Number})
}

func (a *fakeActionApp) subscribe(writer http.ResponseWriter, request *http.Request) {
	last, _ := strconv.Atoi(request.Header.Get("Last-Event-ID"))
	events := make(chan noticePayload, noticeFinalSequence)
	a.mu.Lock()
	identity := a.nextID
	a.nextID++
	a.subscribers[identity] = events
	var replay []noticePayload
	for _, notice := range a.history {
		if last > 0 && notice.ID > last {
			replay = append(replay, notice)
		}
	}
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.subscribers, identity)
		a.mu.Unlock()
	}()
	writer.Header().Set(headerContentType, mediaEventStream)
	for _, notice := range replay {
		writeFakeEvent(writer, strconv.Itoa(notice.ID), "notice", notice)
	}
	_, _ = io.WriteString(writer, "event: ready\ndata: true\n\n")
	writer.(http.Flusher).Flush()
	for {
		select {
		case notice := <-events:
			writeFakeEvent(writer, strconv.Itoa(notice.ID), "notice", notice)
		case <-request.Context().Done():
			return
		}
	}
}

func (a *fakeActionApp) sendNotice(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Text    string
		Inspect bool
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	a.mu.Lock()
	if !input.Inspect {
		a.latest++
		notice := noticePayload{ID: a.latest, Text: input.Text}
		a.history = append(a.history, notice)
		for _, events := range a.subscribers {
			events <- notice
		}
	}
	status := NoticeStatus{Active: len(a.subscribers), Latest: a.latest}
	a.mu.Unlock()
	writeFakeJSON(writer, http.StatusOK, status)
}

func fullActionContract() *casefile.ActionContract {
	contract := generate.New("http-actions", 7, 1).Expected.Action
	contract.Cache, contract.Upload, contract.UploadPublic, contract.UploadRemoval = true, true, true, true
	contract.Actors, contract.Notifications, contract.Batch, contract.CancelStream = true, true, true, true
	return contract
}

func TestObserveActionRecordsAConformingApplication(t *testing.T) {
	contract := fullActionContract()
	server, app := newFakeActionServer(t, contract)

	got, err := observeAction(t.Context(), server.Client(), server.URL, fakeActionPage, contract)
	require.NoError(t, err)
	require.NotNil(t, got.Upload)
	require.NoError(t, observeUploadRemoval(t.Context(), server.Client(), server.URL, fakeActionPage, app.work, got.Upload))

	require.NoError(t, checkAction(contract, got))
	assert.NoError(t, compareActions(got, got))
}

func TestCompareActionsNamesTheDifferingEvidence(t *testing.T) {
	contract := fullActionContract()
	server, _ := newFakeActionServer(t, contract)
	reference, err := observeAction(t.Context(), server.Client(), server.URL, fakeActionPage, contract)
	require.NoError(t, err)
	encoded, err := json.Marshal(reference)
	require.NoError(t, err)

	assert.NoError(t, compareActions(nil, nil))
	assert.ErrorContains(t, compareActions(reference, nil), "missing runtime observation")
	for want, mutate := range map[string]func(*ActionObservation){
		"notification traces differ":      func(o *ActionObservation) { o.Notifications.First = nil },
		"ordered batch results differ":    func(o *ActionObservation) { o.Batches = o.Batches[:1] },
		"cache execution":                 func(o *ActionObservation) { o.Cache[0].Execution++ },
		"upload metadata":                 func(o *ActionObservation) { o.Upload.Size++ },
		"overlapping request data":        func(o *ActionObservation) { o.Actors[0].Cookie = "other" },
		"refusal or cancellation differs": func(o *ActionObservation) { o.Cancel.After.Quantity++ },
		"typed results or stream differ":  func(o *ActionObservation) { o.Events = o.Events[1:] },
	} {
		t.Run(want, func(t *testing.T) {
			var changed ActionObservation
			require.NoError(t, json.Unmarshal(encoded, &changed))
			mutate(&changed)
			assert.ErrorContains(t, compareActions(reference, &changed), want)
		})
	}
}

func TestObserveActionTransportRejectsMistypedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Accept") == mediaEventStream {
			writer.Header().Set(headerContentType, mediaJSON)
			_, _ = io.WriteString(writer, "{}")
			return
		}
		writer.Header().Set(headerContentType, mediaJSON)
		_, _ = writer.Write(make([]byte, actionResponseLimit+1))
	}))
	t.Cleanup(server.Close)
	session := &actionSession{client: server.Client(), base: server.URL, token: fakeActionToken, ephemeral: fakeEphemeralToken, label: "x"}

	var destination ActionResponse
	err := observeActionTransport(t.Context(), session, mediaEventStream, 1, &destination)
	assert.ErrorContains(t, err, "action/response: requested text/event-stream, got HTTP 200 application/json")
	err = observeActionTransport(t.Context(), session, mediaJSON, 1, &destination)
	assert.ErrorContains(t, err, "action/size")
	_, err = readFirstStreamFrame(t.Context(), session.post, casefile.ActionValue{Label: "x"})
	assert.ErrorContains(t, err, "action/watch-status")
}

func TestActionEventsRejectsBrokenFraming(t *testing.T) {
	events, err := actionEvents("event: progress\ndata: {\"Label\":\"a\",\"Quantity\":1}\n\n\n")
	require.NoError(t, err)
	assert.Equal(t, []casefile.ActionEvent{{Name: "progress", Value: casefile.ActionValue{Label: "a", Quantity: 1}}}, events)

	_, err = actionEvents("event: progress\ndata: {}")
	assert.ErrorContains(t, err, "truncated event")
	_, err = actionEvents("data: {}\n\n")
	assert.ErrorContains(t, err, "incomplete event")
	_, err = actionEvents("event: progress\ndata: {\n\n")
	assert.ErrorContains(t, err, "action/sse-data")
}
