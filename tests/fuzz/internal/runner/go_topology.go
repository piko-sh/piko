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
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// topologyRegistry registers the case-specific external Go topology package.
	topologyRegistry = `package main
import ("reflect";_ "example.org/pikofuzzcase/dist";ledger "example.org/fuzzledger/v2/service")
func init(){
 extraSymbols=map[string]map[string]reflect.Value{
  "example.org/fuzzledger/v2/service":{
    "Read":reflect.ValueOf(ledger.Read),
    "Previous":reflect.ValueOf(ledger.Previous),
    "NewCallback":reflect.ValueOf(ledger.NewCallback),
    "Summary":reflect.ValueOf(ledger.Summary),
    "Increment":reflect.ValueOf(ledger.Increment),
    "Current":reflect.ValueOf(ledger.Current),
    "TryZero":reflect.ValueOf(ledger.TryZero),
    "Entry":reflect.ValueOf((*ledger.Entry)(nil)),
    "Save":reflect.ValueOf(ledger.Save),
    "Recall":reflect.ValueOf(ledger.Recall),
  },
 }
}`

	// topologyInstanceCount is the number of fresh hosts a topology case observes.
	topologyInstanceCount = 2

	// topologyReadCount is the number of direct ledger reads in one host instance.
	topologyReadCount = 4

	// topologyPastCallbackOffset is the value the past-installed callback adds.
	topologyPastCallbackOffset = 10

	// topologyPresentCallbackOffset is the value the present-installed callback adds.
	topologyPresentCallbackOffset = 20

	// topologyHooksPath is the callback page route.
	topologyHooksPath = "/hooks"

	// topologyPhaseBefore names the reads before the ledger action.
	topologyPhaseBefore = "before"

	// topologyPhaseAfter names the reads after the ledger action.
	topologyPhaseAfter = "after"
)

// GoTopologyReceipt records the application workspace and host instances.
type GoTopologyReceipt struct {
	// Workspace is the selected module directory basename under the private workspace.
	Workspace string

	// Module is the selected module directory basename with workspace resolution disabled.
	Module string

	// Instances holds observations from fresh host instances.
	Instances []GoTopologyInstance
}

// GoTopologyInstance records reads and callbacks in one server instance.
type GoTopologyInstance struct {
	// Callbacks holds callback response observations.
	Callbacks []ProbeObservation

	// Reads holds direct response observations.
	Reads []ProbeObservation

	// ActionStatus is the ledger action's HTTP status.
	ActionStatus int

	// Revision is the ledger revision returned by the action.
	Revision int
}

// topologyAnswer is the decoded ledger action result.
type topologyAnswer struct {
	// Revision is the ledger revision after the adjustment.
	Revision int
}

// topologyCallbackKey is one expected callback visit and its recorded values.
type topologyCallbackKey struct {
	// path is the callback page route.
	path string

	// past is the value recorded by the past-installed callback, or -1.
	past int

	// present is the value recorded by the present-installed callback, or -1.
	present int
}

// topologyResolution resolves the private application's workspace and module identities.
//
// Takes app (string) which is the private generated application directory.
// Takes workspaceFile (string) which is the private Go workspace file.
// Takes env ([]string) which supplies the private process environment.
//
// Returns string which is the selected module directory basename under the private
// workspace.
// Returns string which is the selected module directory basename with workspace
// resolution disabled.
// Returns error which reports workspace resolution or invalid selected-module metadata,
// or nil on success.
func topologyResolution(ctx context.Context, app, workspaceFile string, env []string) (workspace, module string, err error) {
	for index, goWork := range []string{workspaceFile, "off"} {
		environment := append(slices.Clone(env), "GOWORK="+goWork)
		output, resolveErr := command(ctx, app, environment, "topology/module-resolution", "go", "list", "-m", "-f", "{{.Dir}}", casefile.TopologyModule)
		if resolveErr != nil {
			return "", "", resolveErr
		}
		name := filepath.Base(strings.TrimSpace(output))
		if index == 0 {
			workspace = name
		} else {
			module = name
		}
	}
	return workspace, module, nil
}

// observeTopology reads Go-host state before and after callbacks and actions.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes page (string) which holds the rendered page markup.
// Takes want (*casefile.GoTopology) which holds the independently expected result.
//
// Returns GoTopologyInstance which records callback, action and direct-read evidence from
// one host instance.
// Returns error which reports a host request or state-receipt decoding failure, or nil on
// success.
func observeTopology(ctx context.Context, client *http.Client, base, page string, want *casefile.GoTopology) (GoTopologyInstance, error) {
	var trace GoTopologyInstance
	for _, phase := range []string{topologyPhaseBefore, topologyPhaseAfter} {
		if phase == topologyPhaseAfter {
			if err := adjustTopologyLedger(ctx, client, base, page, want.Step, &trace); err != nil {
				return trace, err
			}
		}
		if err := readTopologyPhase(ctx, client, base, phase, want.GuestCallbacks, &trace); err != nil {
			return trace, err
		}
	}
	return trace, nil
}

// adjustTopologyLedger submits the ledger adjustment action and records its result.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes page (string) which holds the rendered page markup carrying the action tokens.
// Takes step (int) which is the ledger adjustment.
// Takes trace (*GoTopologyInstance) which receives the action status and revision.
//
// Returns error which reports a token, request or decoding failure.
func adjustTopologyLedger(ctx context.Context, client *http.Client, base, page string, step int, trace *GoTopologyInstance) error {
	token, ephemeral, err := pageActionTokens(page)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"delta": step, "_csrf_ephemeral_token": ephemeral})
	if err != nil {
		return fmt.Errorf("topology/action-encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/_piko/actions/ledger.Adjust", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	setActionHeaders(request, mediaJSON, mediaJSON, token)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	trace.ActionStatus = response.StatusCode
	var answer topologyAnswer
	if err := decodeAction(response, &answer); err != nil {
		return err
	}
	trace.Revision = answer.Revision
	return nil
}

// readTopologyPhase reads the ledger pages and, when selected, the callback pages of one
// phase.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes phase (string) which is topologyPhaseBefore or topologyPhaseAfter.
// Takes callbacks (bool) which selects the guest callback visits.
// Takes trace (*GoTopologyInstance) which receives the reads and callbacks.
//
// Returns error which reports a failed page read.
func readTopologyPhase(ctx context.Context, client *http.Client, base, phase string, callbacks bool, trace *GoTopologyInstance) error {
	for _, path := range []string{"/ledger", "/ledger-detail"} {
		read, err := readTopologyPage(ctx, client, base, path)
		if err != nil {
			return err
		}
		trace.Reads = append(trace.Reads, read)
	}
	if !callbacks {
		return nil
	}
	hooks := []string{topologyHooksPath, "/hooks?install=past"}
	if phase == topologyPhaseAfter {
		hooks = []string{"/hooks?install=present", topologyHooksPath}
	}
	for _, path := range hooks {
		callback, err := readTopologyPage(ctx, client, base, path)
		if err != nil {
			return err
		}
		trace.Callbacks = append(trace.Callbacks, callback)
	}
	return nil
}

// readTopologyPage reads one Go-host response and its semantic output.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes path (string) which is the Go-host page route to read.
//
// Returns ProbeObservation which records the Go-host page response and semantic output.
// Returns error which reports a host request or state-receipt decoding failure, or nil on
// success.
func readTopologyPage(ctx context.Context, client *http.Client, base, path string) (ProbeObservation, error) {
	var observation ProbeObservation
	status, _, body, err := get(ctx, client, base, path)
	if err != nil {
		return observation, err
	}
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return observation, err
	}
	observation.Path, observation.Status, observation.RawHTML = path, status, string(body)
	readProgram(document, &observation, false)
	return observation, nil
}

// checkTopology checks shared Go-host state against the saved model.
//
// Takes model (*casefile.GoTopology) which holds the independent expected interaction
// model.
// Takes got (*GoTopologyReceipt) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkTopology(model *casefile.GoTopology, got *GoTopologyReceipt) error {
	if got == nil {
		return responseMismatchf("topology/receipt", "topology/receipt: missing service and module observations")
	}
	if got.Module != "module" || got.Workspace != "workspace" {
		return responseMismatchf("topology/resolution", "topology/resolution: go list reported module=%q, workspace=%q", got.Module, got.Workspace)
	}
	if len(got.Instances) != topologyInstanceCount {
		return responseMismatchf("topology/instances", "topology/instances: got %d, want two fresh hosts", len(got.Instances))
	}
	for instanceIndex := range got.Instances {
		if err := checkTopologyInstance(model, fmt.Sprintf("instance-%d", instanceIndex+1), &got.Instances[instanceIndex]); err != nil {
			return err
		}
	}
	return nil
}

// checkTopologyInstance checks one host instance's action, callbacks and reads.
//
// Takes model (*casefile.GoTopology) which holds the expected interaction model.
// Takes instance (string) which names the instance in failure subjects.
// Takes run (*GoTopologyInstance) which holds the instance's observations.
//
// Returns error which reports the first mismatch, or nil when the instance matches.
func checkTopologyInstance(model *casefile.GoTopology, instance string, run *GoTopologyInstance) error {
	if run.ActionStatus != http.StatusOK || run.Revision != model.Initial+model.Step {
		return responseMismatchf(instance+"/action/revision", "topology/action: %s got status=%d revision=%d, want 200 and %d", instance, run.ActionStatus, run.Revision, model.Initial+model.Step)
	}
	if len(run.Reads) != topologyReadCount {
		return responseMismatchf(instance+"/reads", "topology/reads: %s got %d, want 4", instance, len(run.Reads))
	}
	keys := topologyCallbackKeys(model)
	if len(run.Callbacks) != len(keys) {
		return responseMismatchf(instance+"/callbacks", "topology/callbacks: got %d callback visits, want %d", len(run.Callbacks), len(keys))
	}
	var empty casefile.Case
	for callbackIndex, key := range keys {
		probe := topologyProbe(key.path, fmt.Sprintf("%s-callback-%d", instance, callbackIndex+1),
			programOutput("past", strconv.Itoa(key.past)), programOutput("present", strconv.Itoa(key.present)))
		if err := checkProbe(&empty, "", probe, &run.Callbacks[callbackIndex]); err != nil {
			return locateProbe(probe, err)
		}
	}
	for readIndex := range run.Reads {
		phase := topologyPhaseBefore
		value := model.Initial
		if readIndex >= 2 {
			phase = topologyPhaseAfter
			value += model.Step
		}
		path := []string{"/ledger", "/ledger-detail"}[readIndex%2]
		probe := topologyProbe(path, instance+"-"+phase,
			programOutput("page", strings.TrimPrefix(path, "/")),
			programOutput("value", strconv.Itoa(value)),
			programOutput("old", strconv.Itoa(model.Initial)),
			programOutput("new", strconv.Itoa(value)),
			programOutput("origin", "left/right/1/1/1/workspace/positive delta required/positive delta required"),
		)
		if err := checkProbe(&empty, "", probe, &run.Reads[readIndex]); err != nil {
			return locateProbe(probe, err)
		}
	}
	return nil
}

// topologyCallbackKeys lists the expected callback visits and their recorded values.
//
// Takes model (*casefile.GoTopology) which holds the expected interaction model.
//
// Returns []topologyCallbackKey which lists the visits, or nil without guest callbacks.
func topologyCallbackKeys(model *casefile.GoTopology) []topologyCallbackKey {
	if !model.GuestCallbacks {
		return nil
	}
	past := model.Initial + topologyPastCallbackOffset
	present := model.Initial + model.Step + topologyPresentCallbackOffset
	return []topologyCallbackKey{
		{path: topologyHooksPath, past: -1, present: -1},
		{path: "/hooks?install=past", past: past, present: -1},
		{path: "/hooks?install=present", past: past, present: present},
		{path: topologyHooksPath, past: past, present: present},
	}
}

// topologyProbe builds the program probe a topology page must satisfy.
//
// Takes path (string) which is the page route.
// Takes stageName (string) which identifies the visit.
// Takes outputs (...casefile.ProgramOutput) which lists the expected program outputs.
//
// Returns casefile.Probe which is the expected probe.
func topologyProbe(path, stageName string, outputs ...casefile.ProgramOutput) casefile.Probe {
	var probe casefile.Probe
	probe.Path, probe.Status, probe.Stage, probe.Program, probe.Output = path, http.StatusOK, stageName, true, outputs
	return probe
}

// programOutput builds an expected program output carrying only an identifier and text.
//
// Takes id (string) which identifies the output.
// Takes text (string) which is the expected text.
//
// Returns casefile.ProgramOutput which is the expected output.
func programOutput(id, text string) casefile.ProgramOutput {
	return casefile.ProgramOutput{ID: id, Text: text, Value: "", ScopedKey: "", FullKey: "", Owner: "", Hidden: false, HasValue: false}
}

// compareTopology compares Go-host state across server modes.
//
// Takes a (*GoTopologyReceipt) which holds the first value to compare.
// Takes b (*GoTopologyReceipt) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareTopology(a, b *GoTopologyReceipt) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil || a.Module != b.Module || a.Workspace != b.Workspace || len(a.Instances) != len(b.Instances) {
		return errors.New("topology/parity: module or instance counts differ")
	}
	for instanceIndex := range a.Instances {
		if err := compareTopologyInstance(instanceIndex+1, &a.Instances[instanceIndex], &b.Instances[instanceIndex]); err != nil {
			return err
		}
	}
	return nil
}

// compareTopologyInstance compares one host instance's callbacks, action and reads across
// server modes.
//
// Takes number (int) which is the instance's one-based position.
// Takes left (*GoTopologyInstance) which holds the first mode's instance.
// Takes right (*GoTopologyInstance) which holds the second mode's instance.
//
// Returns error which reports the first difference, or nil when the instances agree.
func compareTopologyInstance(number int, left, right *GoTopologyInstance) error {
	if len(left.Callbacks) != len(right.Callbacks) {
		return fmt.Errorf("topology/parity: instance %d callback counts differ", number)
	}
	for callbackIndex := range left.Callbacks {
		leftCallback, rightCallback := &left.Callbacks[callbackIndex], &right.Callbacks[callbackIndex]
		if leftCallback.Path != rightCallback.Path || leftCallback.Status != rightCallback.Status || !slices.Equal(leftCallback.Output, rightCallback.Output) {
			return fmt.Errorf("topology/parity: instance %d callback %d differs", number, callbackIndex+1)
		}
	}
	if left.ActionStatus != right.ActionStatus || left.Revision != right.Revision || len(left.Reads) != len(right.Reads) {
		return fmt.Errorf("topology/parity: instance %d status, revision or request count differs", number)
	}
	for readIndex := range left.Reads {
		leftRead, rightRead := &left.Reads[readIndex], &right.Reads[readIndex]
		if leftRead.Path != rightRead.Path || leftRead.Status != rightRead.Status || leftRead.Program != rightRead.Program || !reflect.DeepEqual(leftRead.Output, rightRead.Output) {
			return fmt.Errorf("topology/parity: instance %d read %d differs", number, readIndex+1)
		}
	}
	return nil
}
