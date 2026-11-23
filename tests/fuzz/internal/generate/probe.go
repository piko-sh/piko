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
	"net/http"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// domClick names the click interaction.
	domClick = "click"

	// domFocus names the focus interaction and the focus read.
	domFocus = "focus"

	// domText names the text-content read.
	domText = "text"

	// domOpen names the open-state read.
	domOpen = "open"

	// domTrue is the read value of a true boolean property.
	domTrue = "true"
)

// newProbe builds a request probe that expects the given status.
//
// Takes path (string) which is the request path and query.
// Takes status (int) which is the expected HTTP status.
//
// Returns casefile.Probe which holds the request and its expected status.
func newProbe(path string, status int) casefile.Probe {
	var probe casefile.Probe
	probe.Path = path
	probe.Status = status
	return probe
}

// textProbe builds a successful request probe that expects visible text.
//
// Takes path (string) which is the request path and query.
// Takes text (string) which is the expected visible text.
//
// Returns casefile.Probe which holds the request and its expected text.
func textProbe(path, text string) casefile.Probe {
	probe := newProbe(path, http.StatusOK)
	probe.Text = text
	return probe
}

// programProbe builds a successful request probe that checks program outputs.
//
// Takes path (string) which is the request path and query.
// Takes outputs (...casefile.ProgramOutput) which lists the expected program outputs.
//
// Returns casefile.Probe which holds the request and its expected outputs.
func programProbe(path string, outputs ...casefile.ProgramOutput) casefile.Probe {
	probe := newProbe(path, http.StatusOK)
	probe.Program = true
	probe.Output = outputs
	return probe
}

// semanticProbe builds a successful request probe that checks a semantic contract.
//
// Takes path (string) which is the request path and query.
// Takes contract (*casefile.ResponseContract) which holds the expected landmarks.
//
// Returns casefile.Probe which holds the request and its expected contract.
func semanticProbe(path string, contract *casefile.ResponseContract) casefile.Probe {
	probe := newProbe(path, http.StatusOK)
	probe.Semantic = contract
	return probe
}

// programOutput builds an expected program output that carries only text.
//
// Takes id (string) which identifies the output element.
// Takes text (string) which is the expected rendered text.
//
// Returns casefile.ProgramOutput which holds the expected output.
func programOutput(id, text string) casefile.ProgramOutput {
	var output casefile.ProgramOutput
	output.ID = id
	output.Text = text
	return output
}

// valuedOutput builds an expected program output that also carries a value.
//
// Takes id (string) which identifies the output element.
// Takes text (string) which is the expected rendered text.
// Takes value (string) which is the expected bound value.
//
// Returns casefile.ProgramOutput which holds the expected output and value.
func valuedOutput(id, text, value string) casefile.ProgramOutput {
	output := programOutput(id, text)
	output.Value = value
	output.HasValue = true
	return output
}

// responseContract builds a semantic contract from head fields and landmarks.
//
// Takes head (map[string][]string) which holds the expected head fields, or nil.
// Takes elements (...casefile.PageElement) which lists the expected landmarks.
//
// Returns *casefile.ResponseContract which holds the expected contract.
func responseContract(head map[string][]string, elements ...casefile.PageElement) *casefile.ResponseContract {
	return &casefile.ResponseContract{AppText: nil, Head: head, Ordinary: nil, Elements: elements}
}

// clientAction builds one client interaction against a component instance.
//
// Takes op (casefile.ClientOp) which selects the interaction.
// Takes instance (int) which selects the component instance.
// Takes value (string) which is the interaction's input, or empty.
//
// Returns casefile.ClientAction which holds the interaction.
func clientAction(op casefile.ClientOp, instance int, value string) casefile.ClientAction {
	return casefile.ClientAction{Op: op, Instance: instance, Value: value}
}

// domStep builds one step of a DOM journey.
//
// Takes id (string) which identifies the step.
// Takes op (string) which names the interaction performed before reading fields.
// Takes host (string) which names the component host, or empty for the page.
// Takes target (string) which selects the interaction target, or empty.
// Takes value (string) which is the interaction's input, or empty.
// Takes fields (...casefile.DOMField) which lists the values read after the step.
//
// Returns casefile.DOMStep which holds the step and its expected reads.
func domStep(id, op, host, target, value string, fields ...casefile.DOMField) casefile.DOMStep {
	return casefile.DOMStep{ID: id, Op: op, Target: target, Host: host, Value: value, Fields: fields}
}

// domQuery builds a DOM read against the page rather than a component host.
//
// Takes selector (string) which selects the read element.
// Takes read (string) which names the property read.
//
// Returns casefile.DOMQuery which holds the page-level read.
func domQuery(selector, read string) casefile.DOMQuery {
	return casefile.DOMQuery{Host: "", Selector: selector, Read: read, Argument: ""}
}
