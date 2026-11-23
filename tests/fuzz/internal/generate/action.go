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

package generate

import (
	"fmt"
	"net/http"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// actionRoundChoices bounds the number of streamed progress rounds.
	actionRoundChoices = 3

	// collectActionPath is the generated path of the collect action source.
	collectActionPath = "actions/fuzz/collect.go"

	// collectActionTemplate formats the collect action source from its quantity offset and
	// streamed round count.
	collectActionTemplate = `package fuzz

import (
	"errors"
	"sync/atomic"
	"time"
	"piko.sh/piko"
)

type Input struct { Label string; Quantity int; Operation string }
var cancelled atomic.Int32
type Result struct { Label string; Quantity int }
type CollectAction struct { piko.ActionMetadata }

func (a *CollectAction) Call(input Input) (Result, error) {
	switch input.Operation {
	case "validate":
		if input.Quantity <= 0 { return Result{},piko.NewValidationError(map[string]string{"Quantity":"must be positive"}) }
	case "private":
		if a.Auth()==nil || !a.Auth().IsAuthenticated() {return Result{},piko.Unauthorised("sign in")}
	case "restricted":
		return Result{},piko.Forbidden("permission required")
	case "status":
		return Result{Label:input.Label,Quantity:int(cancelled.Load())},nil
	}
	return Result{Label: input.Label, Quantity: input.Quantity + %d}, nil
}

func (a *CollectAction) StreamProgress(stream *piko.SSEStream) error {
	input, ok := piko.ActionInput[Input](&a.ActionMetadata)
	if !ok { return errors.New("stream input was not bound") }
	if input.Operation=="watch" {
		if err:=stream.Send("waiting",Result{Label:input.Label,Quantity:input.Quantity});err!=nil{return err}
		select {case <-stream.Done(): cancelled.Add(1); return nil; case <-time.After(4*time.Second):return errors.New("client did not disconnect")}
	}
	for i := 0; i < %d; i++ {
		if err := stream.Send("progress", Result{Label:input.Label, Quantity:input.Quantity+i}); err != nil { return err }
	}
	out, err := a.Call(input)
	if err != nil { return err }
	return stream.SendComplete(out)
}
`
)

var (
	// actionCacheSource is the formatted Go source staged in the generated project.
	actionCacheSource = fixturego.Fixed("actions/fuzz/snapshot.go")

	// actionUploadSource is the formatted Go source staged in the generated project.
	actionUploadSource = fixturego.Fixed("actions/fuzz/store.go")

	// actionRemoveUploadSource is the formatted Go source staged in the generated project.
	actionRemoveUploadSource = fixturego.Fixed("actions/fuzz/remove_upload.go")

	// actionSendNoticeSource is the formatted Go source staged in the generated project.
	actionSendNoticeSource = fixturego.Fixed("actions/fuzz/send_notice.go")

	// actionSubscribeSource is the formatted Go source staged in the generated project.
	actionSubscribeSource = fixturego.Fixed("actions/fuzz/subscribe.go")

	// actionActorsSource is the formatted Go source staged in the generated project.
	actionActorsSource = fixturego.Fixed("actions/fuzz/rendezvous.go")
)

// addHTTPAction adds source and expected observations for HTTP actions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
func addHTTPAction(c *casefile.Case, p Project) {
	input := casefile.ActionValue{Label: p.Title, Quantity: p.Count + 1}
	offset := 1 + len(p.Rows)
	rounds := 1 + p.Followup%actionRoundChoices
	result := casefile.ActionValue{Label: input.Label, Quantity: input.Quantity + offset}
	contract := &casefile.ActionContract{
		Batch:         true,
		Notifications: false,
		Cache:         false,
		Upload:        false,
		UploadRemoval: false,
		UploadPublic:  false,
		Actors:        false,
		Refusals:      keptRefusals(p.skipRefusals),
		CancelStream:  !p.omitCancellation,
		Input:         input,
		Result:        result,
		Events:        nil,
	}
	for round := range rounds {
		contract.Events = append(contract.Events, casefile.ActionEvent{Name: "progress", Value: casefile.ActionValue{Label: input.Label, Quantity: input.Quantity + round}})
	}
	contract.Events = append(contract.Events, casefile.ActionEvent{Name: "complete", Value: result})
	c.Expected.Action = contract
	c.Features = append(c.Features, "serial-action-batch", "parallel-action-batch")
	addActionExtras(c, p, contract)
	c.Features = append(c.Features, "typed-action-http", "typed-action-sse")
	if len(contract.Refusals) > 0 {
		c.Features = append(c.Features, "action-refusals")
	}
	if contract.CancelStream {
		c.Features = append(c.Features, "stream-cancellation")
	}
	c.Files[collectActionPath] = fmt.Sprintf(collectActionTemplate, offset, rounds)
}

// addActionExtras adds the optional cache, upload and actor action sources.
//
// Takes c (*casefile.Case) which receives generated sources and features.
// Takes p (Project) which holds the drawn application model.
// Takes contract (*casefile.ActionContract) which receives the enabled behaviours.
func addActionExtras(c *casefile.Case, p Project, contract *casefile.ActionContract) {
	if !p.omitCache {
		contract.Cache = true
		c.Files["actions/fuzz/snapshot.go"] = actionCacheSource
		c.Features = append(c.Features, "http-action-cache")
	}
	if !p.omitUpload {
		contract.Upload = true
		contract.UploadPublic = true
		contract.UploadRemoval = true
		c.Files["actions/fuzz/remove_upload.go"] = actionRemoveUploadSource
		c.Files["actions/fuzz/store.go"] = actionUploadSource
		c.Features = append(c.Features, "multipart-disk-readback", "public-upload-download", "storage-byte-range", "upload-delete-visibility")
	}
	if !p.omitActors {
		contract.Actors = true
		contract.Notifications = true
		c.Files["actions/fuzz/send_notice.go"] = actionSendNoticeSource
		c.Files["actions/fuzz/subscribe.go"] = actionSubscribeSource
		c.Files["actions/fuzz/rendezvous.go"] = actionActorsSource
		c.Features = append(c.Features, "overlapping-action-requests", "sse-two-subscribers", "sse-reconnect-replay")
	}
}

// keptRefusals lists the action refusals that the skip mask leaves in place.
//
// Takes skip (uint8) which holds one bit per refusal to omit.
//
// Returns []casefile.ActionRefusal which holds the retained refusals, or nil.
func keptRefusals(skip uint8) []casefile.ActionRefusal {
	refusals := []casefile.ActionRefusal{
		{Operation: "validate", Code: "VALIDATION_FAILED", Message: "validation failed", Status: http.StatusUnprocessableEntity, Errors: map[string]string{"Quantity": "must be positive"}},
		{Operation: "private", Code: "UNAUTHORISED", Message: "sign in", Status: http.StatusUnauthorized, Errors: nil},
		{Operation: "restricted", Code: "FORBIDDEN", Message: "permission required", Status: http.StatusForbidden, Errors: nil},
	}
	var kept []casefile.ActionRefusal
	for position, refusal := range refusals {
		if skip&(1<<position) == 0 {
			kept = append(kept, refusal)
		}
	}
	return kept
}
