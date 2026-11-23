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
	"cmp"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"reflect"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// spaceSeparator joins accessible-name fragments.
	spaceSeparator = " "

	// attributeID is the HTML id attribute name.
	attributeID = "id"
)

// subjectError is a failure that carries a stable failed-property identity.
type subjectError interface {
	error

	// FailureSubject identifies the failed property independently of generated values.
	//
	// Returns string which is the stable failed-property identity.
	FailureSubject() string
}

// responseMismatch identifies a response field and its owning request.
type responseMismatch struct {
	// subject identifies the failed response property.
	subject string

	// detail describes the mismatch.
	detail string

	// owner identifies the owning element within the response, if any.
	owner string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e *responseMismatch) Error() string { return e.detail }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e *responseMismatch) FailureSubject() string { return e.subject }

// actorClient creates or reuses an independent cookie jar for the saved actor.
//
// Takes base (*http.Client) which is the HTTP client used for the request.
// Takes actors (map[string]*http.Client) which maps actor names to independent cookie-jar
// clients.
// Takes origin (string) which is the application origin.
// Takes probe (casefile.Probe) which holds the saved request and its expected response.
//
// Returns *http.Client which is the selected actor's independently maintained cookie-jar
// client.
// Returns error which reports an invalid actor origin or cookie-jar setup failure, or nil
// on success.
func actorClient(base *http.Client, actors map[string]*http.Client, origin string, probe casefile.Probe) (*http.Client, error) {
	if probe.Actor == "" {
		if len(probe.Cookies) > 0 {
			return nil, errors.New("probe/actor: cookies need a named actor")
		}
		return base, nil
	}
	client, ok := actors[probe.Actor]
	if !ok || client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		clone := *base
		clone.Jar = jar
		client = &clone
		actors[probe.Actor] = client
	}
	jarURL, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}

	jarURL.Host = requestHost
	for name, value := range probe.Cookies {
		cookie := &http.Cookie{Name: name, Value: value, Path: "/"} //nolint:gosec // G124: a client-side jar cookie for the plain-HTTP loopback host; Secure would stop it being sent
		if err := cookie.Valid(); err != nil {
			return nil, fmt.Errorf("probe/cookie: %w", err)
		}
		client.Jar.SetCookies(jarURL, []*http.Cookie{cookie})
	}
	return client, nil
}

// elementName calculates an accessible name from supported public HTML fields.
//
// Takes node (*html.Node) which is the HTML or template node to inspect.
// Takes ids (map[string]*html.Node) which maps element IDs to HTML nodes.
// Takes labels (map[string][]string) which maps form-control IDs to explicit label text.
//
// Returns string which is the element's accessible name under the supported naming rules.
func elementName(node *html.Node, ids map[string]*html.Node, labels map[string][]string) string {
	if name := labelledByName(node, ids); name != "" {
		return name
	}
	if label := attr(node, "aria-label"); label != "" {
		return label
	}
	if node.Data == "input" || node.Data == "select" || node.Data == "textarea" {
		if text := labels[attr(node, attributeID)]; len(text) > 0 {
			return strings.Join(text, spaceSeparator)
		}
	}
	if node.Data == "img" {
		return attr(node, "alt")
	}
	if node.Data == "form" {
		return ""
	}
	return strings.Join(strings.Fields(contents(node)), spaceSeparator)
}

// labelledByName joins the text of the elements an aria-labelledby attribute names.
//
// Takes node (*html.Node) which is the element to name.
// Takes ids (map[string]*html.Node) which maps element IDs to HTML nodes.
//
// Returns string which is the referenced text, or empty when no reference resolves.
func labelledByName(node *html.Node, ids map[string]*html.Node) string {
	var parts []string
	for reference := range strings.FieldsSeq(attr(node, "aria-labelledby")) {
		if target, ok := ids[reference]; ok && target != nil {
			parts = append(parts, contents(target))
		}
	}
	return strings.Join(strings.Fields(strings.Join(parts, spaceSeparator)), spaceSeparator)
}

// readResponseContract extracts public head fields and semantic landmarks.
//
// Takes document (*html.Node) which is the parsed HTML document.
//
// Returns *casefile.ResponseContract which holds selected head fields and semantic
// landmarks from the document.
func readResponseContract(document *html.Node) *casefile.ResponseContract {
	contract := &casefile.ResponseContract{Head: map[string][]string{}, AppText: nil, Ordinary: nil, Elements: nil}
	ids, labels := map[string]*html.Node{}, map[string][]string{}
	var head *html.Node
	for node := range document.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if id := attr(node, attributeID); id != "" && ids[id] == nil {
			ids[id] = node
		}
		switch node.Data {
		case "label":
			if key := attr(node, "for"); key != "" {
				labels[key] = append(labels[key], strings.Join(strings.Fields(contents(node)), spaceSeparator))
			}
		case "html":
			contract.Head["lang"] = append(contract.Head["lang"], attr(node, "lang"))
		case "head":
			head = node
		}
	}
	if head != nil {
		readHeadFields(head, contract)
	}
	scope := findID(document, "app")
	if scope == nil {
		scope = document
	}
	readPageElements(scope, ids, labels, contract)
	return contract
}

// readHeadFields records the selected head fields.
//
// Takes head (*html.Node) which is the document head.
// Takes contract (*casefile.ResponseContract) which receives the head fields.
func readHeadFields(head *html.Node, contract *casefile.ResponseContract) {
	for node := range head.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		if key, value := headField(node); key != "" {
			contract.Head[key] = append(contract.Head[key], value)
		}
	}
}

// headField names one head element's recorded field.
//
// Takes node (*html.Node) which is the head element.
//
// Returns key (string) which names the field, or empty when the element is not recorded.
// Returns value (string) which is the field value.
func headField(node *html.Node) (key, value string) {
	switch node.Data {
	case "title":
		return "title", contents(node)
	case "link":
		if attr(node, "rel") == "canonical" {
			return "canonical", attr(node, "href")
		}
		if attr(node, "rel") == "alternate" && attr(node, "hreflang") != "" {
			return "alternate/" + attr(node, "hreflang"), attr(node, "href")
		}
	case "meta":
		name := attr(node, "name")
		property := attr(node, "property")
		if name == "description" || name == "robots" || strings.HasPrefix(name, "twitter:") {
			return name, attr(node, "content")
		}
		if strings.HasPrefix(property, "og:") {
			return property, attr(node, "content")
		}
	}
	return "", ""
}

// readPageElements records keyed landmarks and ordinary interactive elements.
//
// Takes scope (*html.Node) which is the subtree to read.
// Takes ids (map[string]*html.Node) which maps element IDs to HTML nodes.
// Takes labels (map[string][]string) which maps form-control IDs to explicit label text.
// Takes contract (*casefile.ResponseContract) which receives the elements.
func readPageElements(scope *html.Node, ids map[string]*html.Node, labels map[string][]string, contract *casefile.ResponseContract) {
	for node := range scope.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		key := attr(node, "data-fuzz-element")
		if key == "" && !ordinaryElement(node.Data) {
			continue
		}
		element := casefile.PageElement{Key: key, Tag: node.Data, Name: elementName(node, ids, labels), Attributes: map[string]string{}}
		for _, attribute := range node.Attr {
			if recordedAttribute(attribute.Key) {
				element.Attributes[attribute.Key] = attribute.Val
			}
		}
		if key != "" {
			contract.Elements = append(contract.Elements, element)
		} else {
			contract.Ordinary = append(contract.Ordinary, element)
		}
	}
}

// ordinaryElement reports whether an unkeyed element is recorded as an ordinary landmark.
//
// Takes tag (string) which is the element name.
//
// Returns bool which reports whether the element is interactive or an image.
func ordinaryElement(tag string) bool {
	switch tag {
	case "a", "form", "input", "select", "textarea", "button", "img":
		return true
	}
	return false
}

// recordedAttribute reports whether a landmark attribute is part of the contract.
//
// Takes name (string) which is the attribute name.
//
// Returns bool which reports whether the attribute is recorded.
func recordedAttribute(name string) bool {
	switch name {
	case attributeID, "href", "action", "method", "type", "name", "src", "alt", "for", "aria-label", "aria-labelledby", "value", "disabled", "data-sku", "data-theme":
		return true
	}
	return false
}

// responseMismatchf creates a response failure with a stable subject.
//
// Takes subject (string) which identifies the checked property.
// Takes format (string) which is the failure-message format.
// Takes values (...any) which supplies the formatted failure values.
//
// Returns error which is the non-nil response mismatch for the stable subject.
func responseMismatchf(subject, format string, values ...any) error {
	return &responseMismatch{subject: subject, detail: fmt.Sprintf(format, values...), owner: ""}
}

// probeIdentity identifies a request by its route, actor and stage.
//
// Takes probe (casefile.Probe) which holds the saved request and its expected response.
//
// Returns string which is the stable route, actor and stage identity.
func probeIdentity(probe casefile.Probe) string {
	identity := probe.Path
	if probe.Actor != "" {
		identity += "/actor=" + probe.Actor
	}
	if probe.Stage != "" {
		identity += "/stage=" + probe.Stage
	}
	return identity
}

// locateProbe attaches the owning request's stable identity to a failure.
//
// Takes probe (casefile.Probe) which holds the saved request and its expected response.
// Takes err (error) which is the operation failure to inspect.
//
// Returns error which is the supplied failure annotated with its owning request.
func locateProbe(probe casefile.Probe, err error) error {
	if _, ok := errors.AsType[programMismatch](err); ok {
		return err
	}
	field, _, _ := strings.Cut(err.Error(), ":")
	if located, ok := errors.AsType[subjectError](err); ok {
		field = located.FailureSubject()
	}
	if mismatch, ok := errors.AsType[*responseMismatch](err); ok && mismatch.owner != "" {
		field += "/owner=" + url.PathEscape(mismatch.owner)
	}
	return &responseMismatch{subject: probeIdentity(probe) + "/" + field, detail: err.Error(), owner: ""}
}

// checkResponseContract compares semantic response fields with the saved contract.
//
// Takes want (*casefile.ResponseContract) which holds the independently expected result.
// Takes got (*casefile.ResponseContract) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkResponseContract(want, got *casefile.ResponseContract) error {
	if want == nil {
		return nil
	}
	if got == nil {
		return responseMismatchf("semantic/missing", "semantic/missing: no response observation")
	}
	if want.AppText != nil && (got.AppText == nil || *got.AppText != *want.AppText) {
		if got.AppText == nil {
			return responseMismatchf("semantic/app-text", "semantic/app-text: missing #app text, want %q", *want.AppText)
		}
		return responseMismatchf("semantic/app-text", "semantic/app-text: got %q, want %q", *got.AppText, *want.AppText)
	}
	for _, key := range slices.Sorted(maps.Keys(want.Head)) {
		if !slices.Equal(want.Head[key], got.Head[key]) {
			return responseMismatchf("head/"+key, "head/%s: got %q want %q", key, got.Head[key], want.Head[key])
		}
	}
	if want.Ordinary != nil && !slices.EqualFunc(want.Ordinary, got.Ordinary, equalPageElement) {
		return responseMismatchf("semantic/ordinary", "semantic/ordinary: got %+v, want %+v", got.Ordinary, want.Ordinary)
	}
	if len(want.Elements) != len(got.Elements) {
		return responseMismatchf("semantic/count", "semantic/count: got %d want %d", len(got.Elements), len(want.Elements))
	}
	return checkPageElements(want.Elements, got.Elements)
}

// checkPageElements compares keyed landmarks position by position.
//
// Takes want ([]casefile.PageElement) which holds the expected landmarks.
// Takes got ([]casefile.PageElement) which holds the observed landmarks, of equal length.
//
// Returns error which reports the first landmark mismatch with its owner, or nil.
func checkPageElements(want, got []casefile.PageElement) error {
	occurrences := map[string]int{}
	for index := range want {
		expected := &want[index]
		occurrences[expected.Key]++
		owner := elementOwner(expected, occurrences[expected.Key])
		mismatch := func(field, format string, values ...any) error {
			return &responseMismatch{subject: "semantic/" + expected.Key + "/" + field, detail: fmt.Sprintf(format, values...), owner: owner}
		}
		observed := &got[index]
		if observed.Key != expected.Key || observed.Tag != expected.Tag {
			return mismatch("element", "semantic/element: position %d got %s/%s want %s/%s", index, observed.Key, observed.Tag, expected.Key, expected.Tag)
		}
		if observed.Name != expected.Name {
			return mismatch("name", "semantic/name: %s got %q want %q", expected.Key, observed.Name, expected.Name)
		}
		for _, name := range slices.Sorted(maps.Keys(expected.Attributes)) {
			value, ok := observed.Attributes[name]
			if !ok || value != expected.Attributes[name] {
				return mismatch(name, "semantic/%s: %s got %q (present %t) want %q", name, expected.Key, value, ok, expected.Attributes[name])
			}
		}
	}
	return nil
}

// elementOwner identifies the landmark a failure belongs to.
//
// Takes element (*casefile.PageElement) which is the expected landmark.
// Takes occurrence (int) which is the landmark key's one-based occurrence.
//
// Returns string which is the landmark's SKU or id, a positional name for a repeated key,
// or empty.
func elementOwner(element *casefile.PageElement, occurrence int) string {
	owner := cmp.Or(element.Attributes["data-sku"], element.Attributes[attributeID])
	if owner == "" && occurrence > 1 {
		owner = fmt.Sprintf("%s[%d]", element.Key, occurrence)
	}
	return owner
}

// equalPageElement compares all fields of two semantic landmarks.
//
// Takes a (casefile.PageElement) which holds the first value to compare.
// Takes b (casefile.PageElement) which holds the second value to compare.
//
// Returns bool which reports whether all semantic landmark fields agree.
func equalPageElement(a, b casefile.PageElement) bool {
	return a.Key == b.Key && a.Tag == b.Tag && a.Name == b.Name && maps.Equal(a.Attributes, b.Attributes)
}

// commonLandmarks pairs observed landmarks shared by both mode contracts.
//
// Takes left ([]casefile.PageElement) which holds the first mode's semantic landmarks.
// Takes right ([]casefile.PageElement) which holds the second mode's semantic landmarks.
// Takes observedLeft ([]casefile.PageElement) which holds the first mode's observed
// landmarks.
// Takes observedRight ([]casefile.PageElement) which holds the second mode's observed
// landmarks.
//
// Returns []casefile.PageElement which holds the first mode's shared observed landmarks.
// Returns []casefile.PageElement which holds the second mode's shared observed landmarks.
func commonLandmarks(left, right, observedLeft, observedRight []casefile.PageElement) (sharedLeft, sharedRight []casefile.PageElement) {
	positions := map[string][]int{}
	for index, element := range right {
		positions[element.Key] = append(positions[element.Key], index)
	}
	for leftIndex, element := range left {
		matches := positions[element.Key]
		if len(matches) == 0 {
			continue
		}
		rightIndex := matches[0]
		positions[element.Key] = matches[1:]
		leftElement, rightElement := observedLeft[leftIndex], observedRight[rightIndex]
		if element.Name != right[rightIndex].Name {
			leftElement.Name, rightElement.Name = "", ""
		}
		if element.Tag != right[rightIndex].Tag {
			leftElement.Tag, rightElement.Tag = "", ""
		}
		leftElement.Attributes, rightElement.Attributes = maps.Clone(leftElement.Attributes), maps.Clone(rightElement.Attributes)
		for name, value := range element.Attributes {
			if other, ok := right[rightIndex].Attributes[name]; ok && value != other {
				delete(leftElement.Attributes, name)
				delete(rightElement.Attributes, name)
			}
		}
		sharedLeft, sharedRight = append(sharedLeft, leftElement), append(sharedRight, rightElement)
	}
	return sharedLeft, sharedRight
}

// compareResponseContracts compares selected semantic fields across server modes.
//
// Takes a (*casefile.ResponseContract) which holds the first value to compare.
// Takes b (*casefile.ResponseContract) which holds the second value to compare.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareResponseContracts(a, b *casefile.ResponseContract) error {
	if a == nil && b == nil {
		return nil
	}
	if a == nil || b == nil {
		return errors.New("semantic/missing: observation in one mode")
	}

	if err := checkResponseContract(a, b); err != nil {
		return err
	}
	return checkResponseContract(b, a)
}

// CompareCase preserves strict parity for common requests.
//
// A route declaring intentional mode differences passes only after both exact contracts
// pass.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes a (*Observation) which supplies the first observation without mutation.
// Takes b (*Observation) which supplies the second observation without mutation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func CompareCase(c *casefile.Case, a, b *Observation) error {
	first, second := *a, *b
	return compareCase(c, &first, &second)
}

// compareCase checks both observations and compares them with mode-specific differences
// neutralised.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes a (*Observation) which holds the first observation. Its probes are replaced by a
// neutralised copy.
// Takes b (*Observation) which holds the second observation. Its probes are replaced by a
// neutralised copy.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareCase(c *casefile.Case, a, b *Observation) error {
	for _, observation := range []*Observation{a, b} {
		if err := Check(c, observation); err != nil {
			return err
		}
	}
	if !hasModeSpecificProbe(c) {
		return compareObservations(a, b)
	}
	if a.Mode == b.Mode || !knownMode(a.Mode) || !knownMode(b.Mode) {
		return errors.New("probe/mode: need distinct interpreted and compiled observations")
	}
	a.Probes = slices.Clone(a.Probes)
	b.Probes = slices.Clone(b.Probes)
	for index := range c.Probes {
		probe := &c.Probes[index]
		if legacyNotFound(c, probe) {
			a.Probes[index].Text, b.Probes[index].Text = "", ""
			continue
		}
		if len(probe.ByMode) == 0 || reflect.DeepEqual(probe.ByMode[a.Mode], probe.ByMode[b.Mode]) {
			continue
		}
		neutraliseModeProbe(probe.ByMode[a.Mode], probe.ByMode[b.Mode], &a.Probes[index], &b.Probes[index])
	}
	return compareObservations(a, b)
}

// hasModeSpecificProbe reports whether any probe declares an intentional mode difference.
//
// Takes c (*casefile.Case) which holds the saved probes.
//
// Returns bool which reports a probe with per-mode contracts or a legacy not-found page.
func hasModeSpecificProbe(c *casefile.Case) bool {
	for index := range c.Probes {
		if len(c.Probes[index].ByMode) > 0 || legacyNotFound(c, &c.Probes[index]) {
			return true
		}
	}
	return false
}

// knownMode reports whether a mode names one of the two server modes.
//
// Takes mode (string) which is the observation's server mode.
//
// Returns bool which reports whether the mode is interpreted or compiled.
func knownMode(mode string) bool {
	return mode == modeInterpreted || mode == modeCompiled
}

// neutraliseModeProbe clears the probe fields whose contracts intentionally differ
// between modes, keeping unrelated head fields and identical landmarks comparable.
//
// Each mode's expected text and landmarks were already checked exactly.
//
// Takes left (casefile.ModeResponse) which holds the first mode's contract.
// Takes right (casefile.ModeResponse) which holds the second mode's contract.
// Takes first (*ProbeObservation) which holds the first mode's observation.
// Takes second (*ProbeObservation) which holds the second mode's observation.
func neutraliseModeProbe(left, right casefile.ModeResponse, first, second *ProbeObservation) {
	if left.Status != right.Status {
		first.Status, second.Status = 0, 0
	}
	first.Text, second.Text = "", ""
	leftSemantic, rightSemantic := *first.Semantic, *second.Semantic
	leftSemantic.Head, rightSemantic.Head = maps.Clone(leftSemantic.Head), maps.Clone(rightSemantic.Head)
	for key, leftValues := range left.Semantic.Head {
		if rightValues, ok := right.Semantic.Head[key]; ok && !slices.Equal(leftValues, rightValues) {
			delete(leftSemantic.Head, key)
			delete(rightSemantic.Head, key)
		}
	}
	if !reflect.DeepEqual(left.Semantic.Elements, right.Semantic.Elements) {
		leftSemantic.Elements, rightSemantic.Elements = commonLandmarks(left.Semantic.Elements, right.Semantic.Elements, leftSemantic.Elements, rightSemantic.Elements)
	}

	if left.Semantic.Ordinary != nil && right.Semantic.Ordinary != nil && !slices.EqualFunc(left.Semantic.Ordinary, right.Semantic.Ordinary, equalPageElement) {
		leftSemantic.Ordinary, rightSemantic.Ordinary = nil, nil
	}
	first.Semantic, second.Semantic = &leftSemantic, &rightSemantic
}

// visibleAppText extracts visible application text when an application root exists.
//
// Takes document (*html.Node) which is the parsed HTML document.
//
// Returns *string which holds visible application text, or is nil without an application
// root.
func visibleAppText(document *html.Node) *string {
	app := findID(document, "app")
	if app == nil {
		return nil
	}
	var content strings.Builder
	for node := range app.Descendants() {
		if node.Type != html.TextNode || node.Parent == nil || node.Parent.Data == "script" || node.Parent.Data == "style" || htmlHidden(node.Parent) {
			continue
		}
		content.WriteString(node.Data)
	}
	return new(strings.Join(strings.Fields(content.String()), " "))
}
