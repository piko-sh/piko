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
	"slices"
	"strings"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// formShimBaseScore is the lowest generated captcha score percentage.
	formShimBaseScore = 70

	// formShimScoreRange bounds the drawn captcha score above the base score.
	formShimScoreRange = 21

	// formShimBaseQuota is the lowest generated action quota.
	formShimBaseQuota = 5

	// formShimQuota names the submission refused by the action quota.
	formShimQuota = "quota"

	// formShimMissing names the submission sent without a captcha ticket.
	formShimMissing = "missing"
)

var (
	// formShimActions is the formatted Go source staged in the generated project.
	formShimActions = fixturego.Fixed("actions/intake/forms.go")
)

// addFormShimContracts adds source and expected observations for form-client
// interactions.
//
// Takes c (*casefile.Case) which receives generated sources and expected results.
// Takes p (Project) which holds the drawn application model.
//
// Returns error which reports a stale provider fixture edit, or nil on success.
func addFormShimContracts(c *casefile.Case, p Project) error {
	edition := fmt.Sprintf("%d-%d", p.Count, c.Index)
	score := formShimBaseScore + int(uint(p.Count)%formShimScoreRange)
	action := "intake-" + edition
	quota := formShimBaseQuota + int(uint(p.Count)%2)
	model := &casefile.FormShimContract{BrowserSubmission: true, Page: "/intake", Steps: nil}
	kinds := []string{"accepted", "declined", formShimMissing, "unavailable", "recovered", formShimQuota}
	if p.Count%2 != 0 {
		kinds = []string{"accepted", "declined", "unavailable", formShimMissing, "recovered", "accepted-again", formShimQuota}
	}
	var state casefile.FormShimState
	var page strings.Builder
	page.WriteString(`<template><main id="intake"><h1>Local submissions</h1>`)
	for position, kind := range kinds {
		label := fmt.Sprintf("message-%s-%d", edition, position+1)
		step := formShimStep(kind, label, fmt.Sprintf("fixture-%s-%d", edition, position+1))
		if kind != formShimQuota && kind != formShimMissing {
			state.Checks++
			state.Events = append(state.Events, "verify:"+step.Ticket+":"+action+":loopback")
		}
		if step.Status == http.StatusOK {
			state.Writes++
			step.Write = &casefile.FormShimWrite{Label: label, Method: http.MethodPost, Path: "/_piko/actions/intake.Submit", Revision: state.Writes, ScorePercent: score}
			state.Events = append(state.Events, "write:"+label)
		}
		step.After = state
		step.After.Events = slices.Clone(state.Events)
		model.Steps = append(model.Steps, step)
		fmt.Fprintf(&page, `<form data-case="%s" method="post" action="/_piko/actions/intake.Submit"><input name="label" value="%s">`, kind, label)
		if step.Ticket != "" {
			fmt.Fprintf(&page, `<input type="hidden" name="_captcha_token" value="%s">`, step.Ticket)
		}
		page.WriteString(`<button type="submit">Send</button></form>`)
	}
	page.WriteString("</main></template>")
	c.Files["pages/intake.pk"] = page.String()
	provider, err := formShimProvider(quota, score, action)
	if err != nil {
		return err
	}
	c.Files["internal/formfixture/provider.go"] = provider
	c.Files["actions/intake/forms.go"] = formShimActions
	c.Expected.FormShim = model
	c.Features = append(c.Features, "csrf-form", "captcha-provider", "captcha-score-metadata", "action-quota", "independent-write-history")
	return nil
}

// formShimStep models one submission's expected response before its write.
//
// Takes kind (string) which names the submission and selects its outcome.
// Takes label (string) which is the submitted label.
// Takes ticket (string) which is the captcha ticket, dropped for the missing kind.
//
// Returns casefile.FormShimStep which holds the submission and its expected status.
func formShimStep(kind, label, ticket string) casefile.FormShimStep {
	step := casefile.FormShimStep{
		Name:   kind,
		Label:  label,
		Ticket: ticket,
		Status: http.StatusOK,
		Code:   "",
		Write:  nil,
		After:  casefile.FormShimState{},
	}
	switch kind {
	case formShimQuota:
		step.Status = http.StatusTooManyRequests
	case formShimMissing:
		step.Ticket = ""
		step.Status = http.StatusForbidden
		step.Code = "CAPTCHA_FAILED"
	case "declined", "unavailable":
		step.Status = http.StatusForbidden
		step.Code = "CAPTCHA_FAILED"
	}
	return step
}
