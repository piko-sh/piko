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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestTopologyContractRejectsSharedWrongOriginsAndStaleCallbacks(t *testing.T) {
	model := casefile.GoTopology{Initial: 3, Step: 2, GuestCallbacks: true}
	trace := func() *GoTopologyReceipt {
		receipt := &GoTopologyReceipt{Module: "module", Workspace: "workspace"}
		for range 2 {
			host := GoTopologyInstance{Revision: 5, ActionStatus: http.StatusOK}
			for index, path := range []string{"/ledger", "/ledger-detail", "/ledger", "/ledger-detail"} {
				value := "3"
				if index > 1 {
					value = "5"
				}
				host.Reads = append(host.Reads, ProbeObservation{Path: path, Status: http.StatusOK, Program: true, Output: []casefile.ProgramOutput{
					{ID: "page", Text: strings.TrimPrefix(path, "/")},
					{ID: "value", Text: value},
					{ID: "old", Text: "3"},
					{ID: "new", Text: value},
					{ID: "origin", Text: "left/right/1/1/1/workspace/positive delta required/positive delta required"},
				}})
			}
			paths := []string{"/hooks", "/hooks?install=past", "/hooks?install=present", "/hooks"}
			for index, pair := range [][]string{{"-1", "-1"}, {"13", "-1"}, {"13", "25"}, {"13", "25"}} {
				host.Callbacks = append(host.Callbacks, ProbeObservation{Path: paths[index], Status: http.StatusOK, Program: true, Output: []casefile.ProgramOutput{{ID: "past", Text: pair[0]}, {ID: "present", Text: pair[1]}}})
			}
			receipt.Instances = append(receipt.Instances, host)
		}
		return receipt
	}
	good := trace()
	require.NoError(t, checkTopology(&model, good))
	for _, change := range []func(*GoTopologyReceipt){
		func(receipt *GoTopologyReceipt) { receipt.Workspace = "module" },
		func(receipt *GoTopologyReceipt) { receipt.Instances[1].Callbacks[3].Output[0].Text = "25" },
		func(receipt *GoTopologyReceipt) { receipt.Instances[0].Reads[2].Output[2].Text = "5" },
		func(receipt *GoTopologyReceipt) { receipt.Instances[1].Reads[0].Output[1].Text = "5" },
		func(receipt *GoTopologyReceipt) {
			receipt.Instances[0].Reads[3].Output[4].Text = "left/right/2/1/1/workspace"
		},
	} {
		wrong := trace()
		change(wrong)
		assert.Error(t, compareTopology(good, wrong), "one-sided difference escaped")
		assert.Error(t, checkTopology(&model, wrong), "matching wrong results escaped expectation")
	}
}

func newFakeLedgerServer(t *testing.T, model *casefile.GoTopology) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(fakeLedgerHandler(model, fakeActionPage))
	t.Cleanup(server.Close)
	return server
}

func fakeLedgerHandler(model *casefile.GoTopology, page string) *http.ServeMux {
	var mu sync.Mutex
	value, past, present := model.Initial, -1, -1
	program := func(writer http.ResponseWriter, outputs ...string) {
		var body strings.Builder
		body.WriteString(`<html><body><div id="fuzz-program">`)
		for index := 0; index < len(outputs); index += 2 {
			fmt.Fprintf(&body, `<output data-fuzz-output="%s">%s</output>`, outputs[index], outputs[index+1])
		}
		body.WriteString(`</div></body></html>`)
		_, _ = io.WriteString(writer, body.String())
	}
	ledger := func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		current := strconv.Itoa(value)
		mu.Unlock()
		program(writer, "page", strings.TrimPrefix(request.URL.Path, "/"), "value", current, "old", strconv.Itoa(model.Initial), "new", current,
			"origin", "left/right/1/1/1/workspace/positive delta required/positive delta required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, page)
	})
	mux.HandleFunc("GET /ledger", ledger)
	mux.HandleFunc("GET /ledger-detail", ledger)
	mux.HandleFunc("GET /hooks", func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		switch request.URL.Query().Get("install") {
		case "past":
			past = value + topologyPastCallbackOffset
		case "present":
			present = value + topologyPresentCallbackOffset
		}
		pastText, presentText := strconv.Itoa(past), strconv.Itoa(present)
		mu.Unlock()
		program(writer, "past", pastText, "present", presentText)
	})
	mux.HandleFunc("POST /_piko/actions/ledger.Adjust", func(writer http.ResponseWriter, request *http.Request) {
		var input struct{ Delta int }
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.Delta <= 0 {
			http.Error(writer, "positive delta required", http.StatusBadRequest)
			return
		}
		mu.Lock()
		value += input.Delta
		answer := topologyAnswer{Revision: value}
		mu.Unlock()
		writeFakeJSON(writer, http.StatusOK, answer)
	})
	return mux
}

func TestObserveFeaturesRecordsEachTopologyHost(t *testing.T) {
	model := &casefile.GoTopology{Initial: 3, Step: 2, GuestCallbacks: true}
	testCase := &casefile.Case{Expected: casefile.Expected{Topology: model}}
	receipt := &GoTopologyReceipt{Module: "module", Workspace: "workspace"}
	for range topologyInstanceCount {
		server := newFakeLedgerServer(t, model)
		o := Observation{RawHTML: fakeActionPage}
		require.NoError(t, observeFeatures(t.Context(), server.Client(), server.URL, testCase, &o))
		require.NotNil(t, o.Topology)
		require.Len(t, o.Topology.Instances, 1)
		receipt.Instances = append(receipt.Instances, o.Topology.Instances[0])
	}

	require.NoError(t, checkTopology(model, receipt))
	assert.NoError(t, compareTopology(receipt, receipt))
}

func TestObserveTopologyReportsARefusedAdjustment(t *testing.T) {
	model := &casefile.GoTopology{Initial: 3, Step: 0}
	server := newFakeLedgerServer(t, model)
	testCase := &casefile.Case{Expected: casefile.Expected{Topology: model}}
	o := Observation{RawHTML: fakeActionPage}

	err := observeFeatures(t.Context(), server.Client(), server.URL, testCase, &o)

	require.ErrorContains(t, err, "topology/service")
	require.NotNil(t, o.Topology)
	instance := o.Topology.Instances[0]
	assert.Equal(t, http.StatusBadRequest, instance.ActionStatus)
	assert.Len(t, instance.Reads, 2)
	assert.Empty(t, instance.Callbacks)
	_, err = observeTopology(t.Context(), server.Client(), server.URL, "<html></html>", &casefile.GoTopology{Step: 1})
	assert.ErrorContains(t, err, "action/csrf")
}
