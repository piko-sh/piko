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
	"slices"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

// programMismatch identifies a program output mismatch.
type programMismatch struct {
	// subject identifies the failed output property.
	subject string

	// detail describes the mismatch.
	detail string
}

// Error formats the recorded failure.
//
// Returns string which is the formatted failure message.
func (e programMismatch) Error() string { return "probe/program: " + e.subject + " " + e.detail }

// FailureSubject identifies the failed property independently of generated values.
//
// Returns string which is the stable failed-property identity.
func (e programMismatch) FailureSubject() string { return e.subject }

// readProgram extracts program leaves and their structural ownership.
//
// Takes document (*html.Node) which is the parsed HTML document.
// Takes got (*ProbeObservation) which receives observed program roots, ownership and leaf
// values.
// Takes structural (bool) which enables scope and ownership checks.
func readProgram(document *html.Node, got *ProbeObservation, structural bool) {
	root := findID(document, "fuzz-program")
	if root == nil {
		return
	}
	got.Program = true
	if structural {
		got.ProgramRoot = root.Data
		got.RootHidden = htmlHidden(root)
	}
	for node := range root.Descendants() {
		if node.Type != html.ElementNode || node.Data != "output" {
			continue
		}
		got.Output = append(got.Output, readProgramOutput(root, node, structural))
	}
}

// readProgramOutput reads one program output leaf.
//
// Takes root (*html.Node) which is the program root.
// Takes node (*html.Node) which is the output element.
// Takes structural (bool) which enables ownership, key and visibility fields.
//
// Returns casefile.ProgramOutput which is the observed leaf.
func readProgramOutput(root, node *html.Node, structural bool) casefile.ProgramOutput {
	var item casefile.ProgramOutput
	item.ID, item.Text = attr(node, "data-fuzz-output"), contents(node)
	if structural {
		item.Owner = programOwner(root, node)
		item.FullKey = attr(node, "p-key")
		item.Hidden = htmlHidden(node)
	}
	if attr(node, "data-fuzz-check-key") == "true" {
		key := attr(node, "p-key")
		parts := strings.FieldsFunc(key, func(character rune) bool { return character == '.' || character == ':' })
		if len(parts) > 1 {
			item.ScopedKey = parts[0] + "/" + parts[len(parts)-1]
		}
	}
	for _, attribute := range node.Attr {
		if attribute.Key == "data-value" {
			item.HasValue = true
			item.Value = attribute.Val
			break
		}
	}
	return item
}

// programOwner names an output's ancestry from the program root down.
//
// Takes root (*html.Node) which is the program root.
// Takes node (*html.Node) which is the output element.
//
// Returns string which joins each ancestor's tag and id with "/".
func programOwner(root, node *html.Node) string {
	var owners []string
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		name := parent.Data
		if id := attr(parent, attributeID); id != "" {
			name += "#" + id
		}
		owners = append(owners, name)
		if parent == root {
			break
		}
	}
	slices.Reverse(owners)
	return strings.Join(owners, "/")
}

// htmlHidden reports whether authored HTML ancestry hides an element.
//
// Takes node (*html.Node) which is the HTML or template node to inspect.
//
// Returns bool which reports whether authored ancestry hides the element.
func htmlHidden(node *html.Node) bool {
	for ; node != nil; node = node.Parent {
		if slices.ContainsFunc(node.Attr, attributeHides) {
			return true
		}
	}
	return false
}

// attributeHides reports whether one authored attribute hides its element.
//
// Takes attribute (html.Attribute) which is the attribute to inspect.
//
// Returns bool which reports a hidden, aria-hidden or hiding style attribute.
func attributeHides(attribute html.Attribute) bool {
	switch attribute.Key {
	case "hidden":
		return true
	case "aria-hidden":
		return strings.EqualFold(strings.TrimSpace(attribute.Val), "true")
	case "style":
		return styleHides(attribute.Val)
	}
	return false
}

// styleHides reports whether an inline style hides its element.
//
// Takes style (string) which is the inline style declaration list.
//
// Returns bool which reports display:none or visibility:hidden or collapse.
func styleHides(style string) bool {
	for declaration := range strings.SplitSeq(style, ";") {
		name, value, _ := strings.Cut(declaration, ":")
		value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "!important"))
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "display":
			if value == "none" {
				return true
			}
		case "visibility":
			if value == "hidden" || value == "collapse" {
				return true
			}
		}
	}
	return false
}

// checkProgram compares program leaves with the independently evaluated model.
//
// Takes want (casefile.Probe) which holds the independently expected result.
// Takes got (ProbeObservation) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkProgram(want casefile.Probe, got ProbeObservation) error {
	if !got.Program {
		return programMismatch{subject: probeIdentity(want) + "/root", detail: "missing #fuzz-program"}
	}
	if want.ProgramRoot != "" && (want.ProgramRoot != got.ProgramRoot || got.RootHidden) {
		return programMismatch{subject: probeIdentity(want) + "/root", detail: fmt.Sprintf("got <%s> hidden=%t, want visible <%s>", got.ProgramRoot, got.RootHidden, want.ProgramRoot)}
	}
	if len(want.Output) != len(got.Output) {
		return programMismatch{subject: probeIdentity(want) + "/count", detail: fmt.Sprintf("got %d outputs, want %d", len(got.Output), len(want.Output))}
	}
	for index, expected := range want.Output {
		if err := checkProgramOutput(want, index, expected, got.Output[index]); err != nil {
			return err
		}
	}
	return nil
}

// checkProgramOutput checks one program leaf's identity, value and ownership.
//
// Takes want (casefile.Probe) which holds the independently expected result.
// Takes index (int) which is the zero-based item index.
// Takes expected (casefile.ProgramOutput) which holds the independently expected result.
// Takes actual (casefile.ProgramOutput) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkProgramOutput(want casefile.Probe, index int, expected, actual casefile.ProgramOutput) error {
	subject := probeIdentity(want) + "/" + expected.ID
	if expected.FullKey != "" {
		subject += "@" + expected.FullKey
	}
	switch {
	case actual.ID != expected.ID:
		return programMismatch{subject: subject + "/order", detail: fmt.Sprintf("at position %d got %q", index, actual.ID)}
	case expected.Owner != "" && actual.Owner != expected.Owner:
		return programMismatch{subject: subject + "/owner", detail: fmt.Sprintf("got %q, want %q", actual.Owner, expected.Owner)}
	case expected.Owner != "" && actual.Hidden != expected.Hidden:
		return programMismatch{subject: subject + "/visibility", detail: fmt.Sprintf("got hidden=%t, want hidden=%t", actual.Hidden, expected.Hidden)}
	case expected.FullKey != "" && actual.FullKey != expected.FullKey:
		return programMismatch{subject: subject + "/key", detail: fmt.Sprintf("got %q, want %q", actual.FullKey, expected.FullKey)}
	case expected.ScopedKey != "" && actual.ScopedKey != expected.ScopedKey:
		return programMismatch{subject: subject + "/key", detail: fmt.Sprintf("got %q, want %q", actual.ScopedKey, expected.ScopedKey)}
	case actual.HasValue != expected.HasValue:
		return programMismatch{subject: subject + "/attribute", detail: fmt.Sprintf("at position %d has data-value=%t, want %t", index, actual.HasValue, expected.HasValue)}
	case actual.Value != expected.Value:
		return programMismatch{subject: subject + "/value", detail: fmt.Sprintf("at position %d got %q, want %q", index, actual.Value, expected.Value)}
	case actual.Text != expected.Text:
		return programMismatch{subject: subject + "/text", detail: fmt.Sprintf("at position %d got %q, want %q", index, actual.Text, expected.Text)}
	}
	return nil
}
