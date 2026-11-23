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

// Package runner stages and exercises a generated Piko application in one server mode.
package runner

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// componentHashLength is the hexadecimal hash length in a component module's name.
	componentHashLength = 16

	// bodyLimit bounds one HTTP response body in bytes.
	bodyLimit = 2 << 20

	// requestHost is the stable host used for generated request comparisons.
	requestHost = "fuzz.test"

	// probeLimit bounds the follow-up requests of one case.
	probeLimit = 16

	// integritySHA256 ranks the sha256 integrity algorithm.
	integritySHA256 = 1

	// integritySHA384 ranks the sha384 integrity algorithm.
	integritySHA384 = 2

	// integritySHA512 ranks the sha512 integrity algorithm.
	integritySHA512 = 3
)

// ProbeObservation captures one follow-up response before redirects are followed.
type ProbeObservation struct {
	// Semantic holds the selected semantic response fields.
	Semantic *casefile.ResponseContract `json:",omitempty"`

	// Validation holds typed validation evidence.
	Validation *AssetValidation `json:",omitempty"`

	// Asset holds delivered resource bytes or their expected identity.
	Asset *casefile.ResponseAsset `json:",omitempty"`

	// ResponseHeaders holds the response header values.
	ResponseHeaders map[string][]string `json:",omitempty"`

	// Path is the requested route.
	Path string

	// ProgramRoot is the observed program root tag.
	ProgramRoot string `json:",omitempty"`

	// Actor selects the request's session.
	Actor string `json:",omitempty"`

	// Location is the response Location header.
	Location string

	// Text is the response's visible text outside scripts and styles.
	Text string

	// RawHTML is the bounded response markup.
	RawHTML string

	// Output holds the ordered program leaves.
	Output []casefile.ProgramOutput `json:",omitempty"`

	// Status is the HTTP response status.
	Status int

	// RootHidden reports whether the program root is hidden.
	RootHidden bool `json:",omitempty"`

	// Program reports whether the program root and output were observed.
	Program bool `json:",omitempty"`
}

// Observation keeps raw evidence and the fields used by the independent and differential
// checks.
type Observation struct {
	// Semantic holds the selected semantic response fields.
	Semantic *casefile.ResponseContract `json:",omitempty"`

	// FormWorkflow holds the form workflow evidence.
	FormWorkflow *casefile.FormWorkflowObservation `json:",omitempty"`

	// Client holds browser interaction evidence.
	Client *casefile.ClientTransition

	// GeneratedArtifacts holds generated-artefact checks.
	GeneratedArtifacts *GeneratedArtifactReceipt `json:",omitempty"`

	// Topology holds Go-host topology evidence.
	Topology *GoTopologyReceipt `json:",omitempty"`

	// Deployment holds compiled artefact deployment evidence.
	Deployment *DeploymentReceipt `json:",omitempty"`

	// ComponentModules maps component names to delivered module identities.
	ComponentModules map[string]string `json:",omitempty"`

	// CatalogMismatch holds the first differing rebuild response.
	CatalogMismatch *ProbeObservation `json:",omitempty"`

	// Auth holds authentication interaction evidence.
	Auth *AuthReceipt `json:",omitempty"`

	// Action holds the action interaction evidence.
	Action *ActionObservation `json:",omitempty"`

	// FormShim holds form-client interaction evidence.
	FormShim *FormShimReceipt `json:",omitempty"`

	// WorkerLog retains the bounded worker output.
	WorkerLog string

	// ComponentAsset is the delivered component module path.
	ComponentAsset string

	// Mode identifies the server or check mode.
	Mode string `json:",omitempty"`

	// RawHTML retains the bounded response markup.
	RawHTML string

	// ComponentTag is the registered custom element name.
	ComponentTag string

	// ContentType is the primary response media type.
	ContentType string

	// Location is the primary response Location header.
	Location string

	// MetadataTitle is the document title element text.
	MetadataTitle string

	// Heading is the generated heading text.
	Heading string

	// ApplicationDOM is the normalised application markup.
	ApplicationDOM string

	// CatalogRebuild holds warm and clean rebuild comparisons.
	CatalogRebuild []CatalogRebuildStep `json:",omitempty"`

	// Rows holds the ordered row values.
	Rows []string

	// Scripts lists referenced scripts or their coverage measurements.
	Scripts []string

	// Counts holds the component counter snapshots.
	Counts []string

	// CatalogWatch holds observed source-watch events.
	CatalogWatch []string `json:",omitempty"`

	// Probes holds follow-up requests in execution order.
	Probes []ProbeObservation

	// Editions holds source-edition evidence.
	Editions []EditionReceipt `json:",omitempty"`

	// Status is the HTTP response status.
	Status int

	// Visible reports whether the conditional element rendered.
	Visible bool

	// Component reports whether the component element is present.
	Component bool

	// ComponentCompiled reports whether the component was compiled.
	ComponentCompiled bool

	// Frame reports whether the generated application uses a frame.
	Frame bool
}

// asset holds an asset reference and its integrity requirement.
type asset struct {
	// path is the referenced asset path.
	path string

	// integrity is the declared subresource integrity value.
	integrity string

	// script reports whether the asset is JavaScript.
	script bool
}

// verifiedAsset caches the integrity values and component identity already verified for
// one asset path.
type verifiedAsset struct {
	// integrity records the integrity values already verified.
	integrity map[string]bool

	// tag is the component whose module the asset delivers.
	tag string
}

// get reads a bounded same-origin response without following redirects.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes path (string) which is the same-origin HTTP request path.
//
// Returns int which is the HTTP response status.
// Returns http.Header which holds response headers.
// Returns []byte which holds the bounded response body.
// Returns error which reports request construction, HTTP transport or bounded-body
// reading failure, or nil on success.
func get(ctx context.Context, client *http.Client, base, path string) (int, http.Header, []byte, error) {
	return getConditional(ctx, client, base, path, "")
}

// getConditional reads a bounded response using the selected entity validator.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes path (string) which is the same-origin HTTP request path.
// Takes tag (string) which is the conditional entity validator.
//
// Returns int which is the HTTP response status.
// Returns http.Header which holds response headers.
// Returns []byte which holds the bounded response body.
// Returns error which reports request construction, HTTP transport or bounded-body
// reading failure, or nil on success.
func getConditional(ctx context.Context, client *http.Client, base, path, tag string) (int, http.Header, []byte, error) {
	parsed, err := url.ParseRequestURI(path)
	if err != nil || !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || parsed.IsAbs() {
		return 0, nil, nil, fmt.Errorf("expected local path, got %q", path)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return 0, nil, nil, err
	}
	if tag != "" {
		request.Header.Set("If-None-Match", tag)
	}
	request.Host = requestHost
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, nil, err
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, bodyLimit+1))
	if err == nil && len(data) > bodyLimit {
		err = fmt.Errorf("size: response to %s exceeds %d bytes", path, bodyLimit)
	}
	err = errors.Join(err, closeResponse(response))
	return response.StatusCode, response.Header, data, err
}

// attr reads a named HTML attribute.
//
// Takes node (*html.Node) which is the HTML node to inspect.
// Takes key (string) which names the HTML attribute.
//
// Returns string which is the selected attribute value, or an empty string when absent.
func attr(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

// contents collects descendant text in document order.
//
// Takes root (*html.Node) which is the HTML subtree to inspect.
//
// Returns string which is the concatenated descendant text.
func contents(root *html.Node) string {
	var text strings.Builder
	for node := range root.Descendants() {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
	}
	return text.String()
}

// findID finds the first HTML element with the selected ID.
//
// Takes document (*html.Node) which is the parsed HTML document.
// Takes id (string) which is the HTML element ID.
//
// Returns *html.Node which is the first element with the requested ID, or nil when
// absent.
func findID(document *html.Node, id string) *html.Node {
	for node := range document.Descendants() {
		if node.Type == html.ElementNode && attr(node, "id") == id {
			return node
		}
	}
	return nil
}

// canonicalDOM normalises application markup for mode comparison.
//
// Takes root (*html.Node) which is the HTML subtree to inspect.
//
// Returns string which is the normalised application markup.
// Returns error which reports HTML parsing or normalised-markup encoding failure, or nil
// on success.
func canonicalDOM(root *html.Node) (string, error) {
	for node := range root.Descendants() {
		slices.SortFunc(node.Attr, func(left, right html.Attribute) int {
			return cmp.Or(strings.Compare(left.Namespace, right.Namespace), strings.Compare(left.Key, right.Key))
		})
	}
	var out bytes.Buffer
	err := html.Render(&out, root)
	return out.String(), err
}

// extract extracts response fields and referenced assets from the application markup.
//
// Takes o (*Observation) which receives the application observation.
// Takes tag (string) which identifies the custom element.
//
// Returns []asset which lists asset values in their recorded order.
// Returns error which reports invalid application markup or an unsupported asset
// reference, or nil on success.
func extract(o *Observation, tag string) ([]asset, error) {
	document, err := html.Parse(strings.NewReader(o.RawHTML))
	if err != nil {
		return nil, err
	}
	o.Semantic = readResponseContract(document)
	assets := extractDocumentAssets(document, o)
	app := findID(document, "app")
	if app == nil {
		return nil, errors.New("document has no #app")
	}
	o.ApplicationDOM, err = canonicalDOM(app)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		extractBaselineApplication(app, o)
		return assets, nil
	}
	if err := extractCaseRoot(app, tag, o); err != nil {
		return nil, err
	}
	return assets, nil
}

// extractDocumentAssets records the document title and its integrity-bound scripts and
// links.
//
// Takes document (*html.Node) which is the parsed document.
// Takes o (*Observation) which receives the title and script paths.
//
// Returns []asset which lists the referenced assets in document order.
func extractDocumentAssets(document *html.Node, o *Observation) []asset {
	var assets []asset
	for node := range document.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		switch node.Data {
		case "title":
			o.MetadataTitle = contents(node)
		case "script":
			if source := attr(node, "src"); source != "" {
				assets = append(assets, asset{path: source, integrity: attr(node, "integrity"), script: true})
				o.Scripts = append(o.Scripts, source)
			}
		case "link":
			if sri := attr(node, "integrity"); sri != "" {
				assets = append(assets, asset{path: attr(node, "href"), integrity: sri, script: false})
			}
		}
	}
	return assets
}

// extractBaselineApplication records the heading and paragraphs of a baseline page.
//
// Takes app (*html.Node) which is the #app element.
// Takes o (*Observation) which receives the heading and rows.
func extractBaselineApplication(app *html.Node, o *Observation) {
	for node := range app.Descendants() {
		if node.Type != html.ElementNode {
			continue
		}
		switch node.Data {
		case "h1":
			o.Heading = contents(node)
		case "p":
			o.Rows = append(o.Rows, contents(node))
		}
	}
}

// extractCaseRoot records the generated case root's frame, heading, rows, visibility and
// component presence.
//
// Takes app (*html.Node) which is the #app element.
// Takes tag (string) which identifies the custom element.
// Takes o (*Observation) which receives the case fields.
//
// Returns error which reports a missing case root, title or list.
func extractCaseRoot(app *html.Node, tag string, o *Observation) error {
	root := findID(app, "case-root")
	if root == nil {
		return errors.New("application has no #case-root")
	}
	for parent := root.Parent; parent != nil; parent = parent.Parent {
		if parent.Data == "section" && attr(parent, "id") == "case-frame" {
			o.Frame = true
			break
		}
	}
	heading := findID(root, "case-title")
	if heading == nil {
		return errors.New("case has no #case-title")
	}
	o.Heading = contents(heading)
	list := findID(root, "case-list")
	if list == nil {
		return errors.New("case has no #case-list")
	}
	for node := range list.Descendants() {
		if node.Type == html.ElementNode && node.Data == "li" {
			o.Rows = append(o.Rows, contents(node))
		}
	}
	o.Visible = findID(root, "case-visible") != nil
	for node := range root.Descendants() {
		if node.Type == html.ElementNode && node.Data == tag {
			o.Component = true
		}
	}
	return nil
}

// validIntegrity checks the delivered bytes against their integrity declaration.
//
// Uses the strongest supported algorithm in the declaration, ordered sha512, sha384 and
// sha256. The bytes match when any digest for that algorithm matches. A declaration
// without a supported algorithm fails.
//
// Takes sri (string) which is the declared subresource integrity value.
// Takes body ([]byte) which holds the delivered asset bytes.
//
// Returns bool which reports whether the delivered bytes match the declared digest.
func validIntegrity(sri string, body []byte) bool {
	strongest := 0
	var digests []string
	for token := range strings.FieldsSeq(sri) {
		algorithm, digest, _ := strings.Cut(token, "-")
		digest, _, _ = strings.Cut(digest, "?")
		strength := integrityStrength(algorithm)
		switch {
		case strength == 0 || strength < strongest:
			continue
		case strength > strongest:
			strongest, digests = strength, digests[:0]
		}
		digests = append(digests, digest)
	}
	if strongest == 0 {
		return false
	}
	var actual []byte
	switch strongest {
	case integritySHA256:
		sum := sha256.Sum256(body)
		actual = sum[:]
	case integritySHA384:
		sum := sha512.Sum384(body)
		actual = sum[:]
	default:
		sum := sha512.Sum512(body)
		actual = sum[:]
	}
	return slices.Contains(digests, base64.StdEncoding.EncodeToString(actual))
}

// integrityStrength ranks a subresource integrity hash algorithm.
//
// Takes algorithm (string) which is the declared algorithm prefix.
//
// Returns int which is integritySHA256, integritySHA384 or integritySHA512, or 0 when the
// algorithm is not supported.
func integrityStrength(algorithm string) int {
	switch algorithm {
	case "sha256":
		return integritySHA256
	case "sha384":
		return integritySHA384
	case "sha512":
		return integritySHA512
	}
	return 0
}

// observe reads application responses and independently verifies referenced assets.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
//
// Returns Observation which holds the application evidence collected before completion or
// failure.
// Returns error which reports an HTTP response, linked-asset or receipt-extraction
// failure.
func observe(ctx context.Context, client *http.Client, base string, c *casefile.Case) (Observation, error) {
	var o Observation
	status, header, body, err := get(ctx, client, base, c.Request)
	if err != nil {
		return o, err
	}
	o.Status = status
	o.ContentType = mediaType(header)
	o.Location = header.Get("Location")
	o.RawHTML = string(body)
	if status != c.Expected.Status {
		return o, fmt.Errorf("status: got %d, want %d", status, c.Expected.Status)
	}
	assets, err := extract(&o, c.Expected.Tag)
	if err != nil {
		return o, err
	}
	if err := verifyAssets(ctx, client, base, c, assets, &o); err != nil {
		return o, err
	}
	if len(c.Probes) > probeLimit {
		return o, fmt.Errorf("probe limit: more than %d follow-up requests", probeLimit)
	}
	actors := map[string]*http.Client{}
	for index := range c.Probes {
		got, err := observeProbe(ctx, client, actors, base, c.Probes[index])
		if err != nil {
			return o, err
		}
		o.Probes = append(o.Probes, got)
	}
	err = observeFeatures(ctx, client, base, c, &o)
	return o, err
}

// observeFeatures runs the auth, form-shim, action and topology observations the case
// selects.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes c (*casefile.Case) which selects the features.
// Takes o (*Observation) which receives the feature evidence.
//
// Returns error which reports the first failed feature observation, or nil on success.
func observeFeatures(ctx context.Context, client *http.Client, base string, c *casefile.Case, o *Observation) error {
	var err error
	if c.Expected.Auth != nil {
		if o.Auth, err = observeAuth(ctx, client, base, c.Expected.Auth); err != nil {
			return err
		}
	}
	if c.Expected.FormShim != nil {
		if o.FormShim, err = observeFormShim(ctx, client, base, o.RawHTML, c.Expected.FormShim); err != nil {
			return err
		}
	}
	if c.Expected.Action != nil {
		if o.Action, err = observeAction(ctx, client, base, o.RawHTML, c.Expected.Action); err != nil {
			return err
		}
	}
	if c.Expected.Topology != nil {
		instance, err := observeTopology(ctx, client, base, o.RawHTML, c.Expected.Topology)
		o.Topology = &GoTopologyReceipt{Workspace: "", Module: "", Instances: []GoTopologyInstance{instance}}
		if err != nil {
			return fmt.Errorf("topology/service: %w", err)
		}
	}
	return nil
}

// verifyAssets fetches each referenced asset once per integrity value and records the
// delivered component modules.
//
// References are same-origin and each read has a byte cap. Only integrity strings and
// component identity are cached, so large asset bodies do not accumulate.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes assets ([]asset) which lists the referenced assets in document order.
// Takes o (*Observation) which receives the component module evidence.
//
// Returns error which reports a failed, empty or integrity-mismatched asset response, or
// nil on success.
func verifyAssets(ctx context.Context, client *http.Client, base string, c *casefile.Case, assets []asset, o *Observation) error {
	tags := append([]string{c.Expected.Tag}, c.Expected.AdditionalComponents...)
	seen := map[string]verifiedAsset{}
	for _, reference := range assets {
		verified, ok := seen[reference.path]
		if !ok || (reference.integrity != "" && !verified.integrity[reference.integrity]) {
			var err error
			verified, err = fetchVerifiedAsset(ctx, client, base, reference, tags, verified)
			if err != nil {
				return err
			}
			seen[reference.path] = verified
		}
		if !reference.script || verified.tag == "" {
			continue
		}
		if o.ComponentModules == nil {
			o.ComponentModules = map[string]string{}
		}
		o.ComponentModules[verified.tag] = reference.path
		if verified.tag == c.Expected.Tag {
			o.ComponentTag, o.ComponentAsset = verified.tag, reference.path
		}
	}
	return nil
}

// fetchVerifiedAsset reads one asset, checks its integrity and identifies the component
// module it delivers.
//
// Takes client (*http.Client) which is the HTTP client used for the request.
// Takes base (string) which is the application base URL.
// Takes reference (asset) which is the asset reference to read.
// Takes tags ([]string) which lists the case's component tags.
// Takes verified (verifiedAsset) which holds what is already known about the asset.
//
// Returns verifiedAsset which records the verified integrity value and component.
// Returns error which reports a failed, empty or integrity-mismatched response.
func fetchVerifiedAsset(ctx context.Context, client *http.Client, base string, reference asset, tags []string, verified verifiedAsset) (verifiedAsset, error) {
	code, headers, data, err := get(ctx, client, base, reference.path)
	if err != nil {
		return verified, err
	}
	if code != http.StatusOK || len(data) == 0 {
		return verified, fmt.Errorf("asset: empty or failed response for %s (HTTP %d)", reference.path, code)
	}
	if reference.integrity != "" && !validIntegrity(reference.integrity, data) {
		return verified, fmt.Errorf("integrity: digest mismatch for %s", reference.path)
	}
	if verified.integrity == nil {
		verified.integrity = map[string]bool{}
	}
	verified.integrity[reference.integrity] = true
	if !strings.Contains(mediaType(headers), "javascript") {
		return verified, nil
	}
	for _, tag := range tags {
		if componentVariant(reference.path, tag) && bytes.Contains(data, []byte(tag)) {
			verified.tag = tag
			break
		}
	}
	return verified, nil
}

// observeProbe performs one follow-up request as its selected actor.
//
// Takes client (*http.Client) which is the primary HTTP client.
// Takes actors (map[string]*http.Client) which caches per-actor sessions.
// Takes base (string) which is the application base URL.
// Takes probe (casefile.Probe) which holds the saved request and its expected response.
//
// Returns ProbeObservation which holds the observed follow-up response.
// Returns error which reports session, request, revalidation or HTML parsing failure, or
// nil on success.
func observeProbe(ctx context.Context, client *http.Client, actors map[string]*http.Client, base string, probe casefile.Probe) (ProbeObservation, error) {
	var got ProbeObservation
	actor, err := actorClient(client, actors, base, probe)
	if err != nil {
		return got, err
	}
	code, header, data, err := getConditional(ctx, actor, base, probe.Path, probe.IfNoneMatch)
	if err != nil {
		return got, fmt.Errorf("probe/request: %w", err)
	}
	got.Actor, got.Path, got.Status, got.Location = probe.Actor, probe.Path, code, header.Get("Location")
	if probe.Asset != nil {
		err = observeAssetProbe(ctx, actor, base, &probe, header, data, &got)
		return got, err
	}
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		var empty ProbeObservation
		return empty, fmt.Errorf("probe/html: %w", err)
	}
	got.Text, got.RawHTML = documentText(document), string(data)
	if len(probe.ResponseHeaders) > 0 {
		got.ResponseHeaders = map[string][]string{}
		for key := range probe.ResponseHeaders {
			got.ResponseHeaders[key] = slices.Clone(header.Values(key))
		}
	}
	if code >= http.StatusOK && code < http.StatusMultipleChoices || probe.Semantic != nil || len(probe.ByMode) > 0 {
		got.Semantic = readResponseContract(document)
		if probe.Semantic != nil && probe.Semantic.AppText != nil {
			got.Semantic.AppText = visibleAppText(document)
		}
	}
	if probe.Program {
		readProgram(document, &got, probe.ProgramRoot != "")
	}
	return got, nil
}

// observeAssetProbe records a delivered asset's identity and, when requested, its
// conditional revalidation.
//
// Takes actor (*http.Client) which is the probe's session client.
// Takes base (string) which is the application base URL.
// Takes probe (*casefile.Probe) which holds the saved asset request.
// Takes header (http.Header) which holds the response headers.
// Takes data ([]byte) which holds the delivered asset bytes.
// Takes got (*ProbeObservation) which receives the asset evidence.
//
// Returns error which reports a failed revalidation request, or nil on success.
func observeAssetProbe(ctx context.Context, actor *http.Client, base string, probe *casefile.Probe, header http.Header, data []byte, got *ProbeObservation) error {
	got.Asset = &casefile.ResponseAsset{MediaType: mediaType(header), SHA256: sha256Hex(data), Revalidate: false}
	if !probe.Asset.Revalidate {
		return nil
	}
	got.Asset.Revalidate = true
	var err error
	got.Validation, err = observeValidation(ctx, actor, base, probe.Path, header.Get("ETag"))
	return err
}

// documentText concatenates a document's text outside scripts and styles.
//
// Takes document (*html.Node) which is the parsed document.
//
// Returns string which is the concatenated text.
func documentText(document *html.Node) string {
	var plain strings.Builder
	for node := range document.Descendants() {
		if node.Type == html.TextNode && node.Parent != nil && node.Parent.Data != "script" && node.Parent.Data != "style" {
			plain.WriteString(node.Data)
		}
	}
	return plain.String()
}

// mediaType reads the media type of a response's Content-Type header.
//
// Takes header (http.Header) which holds the response headers.
//
// Returns string which is the parsed media type, or the raw header value when it does not
// parse, so a malformed type still fails comparison visibly.
func mediaType(header http.Header) string {
	raw := header.Get(headerContentType)
	media, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return raw
	}
	return media
}

// Check compares one execution with the generator's known result.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func Check(c *casefile.Case, o *Observation) error {
	failures := independentFailures(c, o)
	if len(failures) > 0 {
		return failures[0]
	}
	return nil
}

// IndependentFailures collects discrepancies in independently observed responses and
// subsystems.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists error values in their recorded order.
func IndependentFailures(c *casefile.Case, o *Observation) []error {
	return independentFailures(c, o)
}

// independentFailures collects discrepancies in independently observed responses and
// subsystems.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists error values in their recorded order.
func independentFailures(c *casefile.Case, o *Observation) []error {
	failures := subsystemFailures(c, o)
	if o.Client != nil {
		if err := checkDOMEngine(c, o.Client.DOM); err != nil {
			failures = append(failures, err)
		}
		_, fields := domFailures(c.Expected.DOM, o.Client.DOM)
		failures = append(failures, fields...)
		_, routes := browserRouteFailures(c, o, true)
		failures = append(failures, routes...)
	}
	if c.Target == "content-editions" {
		if err := checkEditions(o.Editions); err != nil {
			failures = append(failures, err)
		}
	}
	if c.Target == "catalog-rebuild" {
		if err := checkCatalogueReceipt(o.CatalogRebuild); err != nil {
			failures = append(failures, err)
		}
	}
	return append(failures, applicationFailures(c, o)...)
}

// subsystemFailures checks the auth, form-shim, topology and deployment evidence the case
// selects.
//
// Takes c (*casefile.Case) which selects the subsystems.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists the subsystem failures in check order.
func subsystemFailures(c *casefile.Case, o *Observation) []error {
	var failures []error
	add := func(err error) {
		if err != nil {
			failures = append(failures, err)
		}
	}
	if c.Expected.Auth != nil {
		add(checkAuth(c.Expected.Auth, o.Auth))
	}
	if c.Expected.FormShim != nil {
		add(checkFormShim(c.Expected.FormShim, o.FormShim))
		add(checkBrowserIntake(c.Expected.FormShim, o.FormShim, o.Client))
	}
	if c.Expected.Topology != nil {
		add(checkTopology(c.Expected.Topology, o.Topology))
	}
	if c.Target == "embedded-deploy" {
		add(checkDeployment(o.Deployment))
	}
	return failures
}

// checkApplication compares the primary response and independent probes with the model.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkApplication(c *casefile.Case, o *Observation) error {
	failures := applicationFailures(c, o)
	if len(failures) > 0 {
		return failures[0]
	}
	return nil
}

// applicationFailures retains independent primary-response and subsystem failures.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists error values in their recorded order.
func applicationFailures(c *casefile.Case, o *Observation) []error {
	failures := primaryFailures(c, o)
	add := func(err error) {
		if err != nil {
			failures = append(failures, err)
		}
	}
	expected := &c.Expected
	if expected.Client != nil && o.Client != nil {
		add(compareClient(o.Client, expected.Client))
	}
	if len(o.Probes) != len(c.Probes) {
		add(fmt.Errorf("probe/count: got %d, want %d", len(o.Probes), len(c.Probes)))
	}
	for index := range min(len(c.Probes), len(o.Probes)) {
		if err := checkProbe(c, o.Mode, c.Probes[index], &o.Probes[index]); err != nil {
			add(locateProbe(c.Probes[index], err))
		}
	}
	if o.FormWorkflow != nil {
		add(verifyFormWorkflow(c.Expected.FormWorkflow, o.FormWorkflow))
	}
	action, err := publicActionResult(o.Action, o.Mode)
	if err != nil {
		add(err)
	} else {
		add(checkAction(expected.Action, action))
	}
	return failures
}

// primaryFailures compares the primary response's fields and components with the model.
//
// Takes c (*casefile.Case) which holds the saved request and expected results.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists the primary-response failures in check order.
func primaryFailures(c *casefile.Case, o *Observation) []error {
	var failures []error
	expected := &c.Expected
	if o.Status != expected.Status {
		failures = append(failures, fmt.Errorf("status: got %d, want %d", o.Status, expected.Status))
	}
	if o.ContentType != "text/html" {
		failures = append(failures, fmt.Errorf("content type: got %q", o.ContentType))
	}
	wantTitle := expected.Title
	if expected.Tag == "" {
		wantTitle = "Piko Page"
	}
	if o.MetadataTitle != wantTitle {
		failures = append(failures, fmt.Errorf("metadata title: got %q, want %q", o.MetadataTitle, wantTitle))
	}
	if o.Heading != expected.Title {
		failures = append(failures, fmt.Errorf("heading: got %q, want %q", o.Heading, expected.Title))
	}
	if !slices.Equal(o.Rows, expected.Rows) {
		failures = append(failures, fmt.Errorf("rows: got %q, want %q", o.Rows, expected.Rows))
	}
	if o.Visible != expected.Show {
		failures = append(failures, fmt.Errorf("conditional: got %t, want %t", o.Visible, expected.Show))
	}
	if o.Frame != expected.Frame {
		failures = append(failures, fmt.Errorf("partial/slot: frame around generated content = %t, want %t", o.Frame, expected.Frame))
	}
	failures = append(failures, componentFailures(expected, o)...)
	if o.Counts != nil && expected.Client == nil && !slices.Equal(o.Counts, []string{strconv.Itoa(expected.Count), strconv.Itoa(expected.Count + 1)}) {
		failures = append(failures, fmt.Errorf("browser/count: got %q, want [%d %d]", o.Counts, expected.Count, expected.Count+1))
	}
	if err := checkResponseContract(expected.Semantic, o.Semantic); err != nil {
		var primary casefile.Probe
		primary.Path = c.Request
		failures = append(failures, locateProbe(primary, fmt.Errorf("primary/%w", err)))
	}
	for _, tag := range expected.AdditionalComponents {
		if o.ComponentModules[tag] == "" {
			failures = append(failures, responseMismatchf("component/"+tag+"/asset", "asset: component %q has no delivered module", tag))
		}
	}
	return failures
}

// componentFailures checks that the case's component was compiled, rendered and
// delivered.
//
// Takes expected (*casefile.Expected) which holds the expected component tag.
// Takes o (*Observation) which holds the application observation.
//
// Returns []error which lists the component failures in check order.
func componentFailures(expected *casefile.Expected, o *Observation) []error {
	if expected.Tag == "" {
		return nil
	}
	var failures []error
	if !o.ComponentCompiled {
		failures = append(failures, fmt.Errorf("pkc-compile: component %q was never compiled", expected.Tag))
	}
	if !o.Component {
		failures = append(failures, fmt.Errorf("tag: component %q missing", expected.Tag))
	}
	if o.ComponentAsset == "" {
		failures = append(failures, responseMismatchf("component/"+expected.Tag+"/asset", "asset: component %q has no delivered module", expected.Tag))
	}
	return failures
}

// checkProbe compares one follow-up response with its mode-specific contract.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes mode (string) which selects the server mode.
// Takes probe (casefile.Probe) which holds the saved request and its expected response.
// Takes got (*ProbeObservation) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkProbe(c *casefile.Case, mode string, probe casefile.Probe, got *ProbeObservation) error {
	if legacyNotFound(c, &probe) && mode != "" {
		if err := checkLegacyNotFound(&probe, got, mode); err != nil {
			return err
		}
	}
	if len(probe.ByMode) > 0 {
		override, ok := probe.ByMode[mode]
		if !ok {
			return fmt.Errorf("probe/mode: %s has no %s contract", probe.Path, mode)
		}
		probe.Status, probe.Semantic = override.Status, &override.Semantic
	}
	if err := checkProbeTransport(&probe, got); err != nil {
		return err
	}
	if err := checkResponseContract(probe.Semantic, got.Semantic); err != nil {
		return fmt.Errorf("probe/%w on %s (%s)", err, probe.Path, probe.Actor)
	}
	switch {
	case got.Path != probe.Path:
		return fmt.Errorf("probe/path: got %q, want %q", got.Path, probe.Path)
	case got.Status != probe.Status:
		return fmt.Errorf("probe/status: %s got %d, want %d", probe.Path, got.Status, probe.Status)
	case got.Location != probe.Location:
		return fmt.Errorf("probe/location: %s got %q, want %q", probe.Path, got.Location, probe.Location)
	case !strings.Contains(got.Text, probe.Text):
		return fmt.Errorf("probe/text: %s missing %q", probe.Path, probe.Text)
	}
	if probe.Program {
		return checkProgram(probe, *got)
	}
	return nil
}

// checkProbeTransport compares a probe's revalidation, asset identity, headers and actor.
//
// Takes probe (*casefile.Probe) which holds the mode-resolved probe contract.
// Takes got (*ProbeObservation) which holds the observed result.
//
// Returns error which reports the first transport mismatch, or nil when they agree.
func checkProbeTransport(probe *casefile.Probe, got *ProbeObservation) error {
	if probe.Asset != nil && probe.Asset.Revalidate {
		if err := checkValidation(probe.Path, probe.Asset, got.Validation); err != nil {
			return err
		}
	}
	if probe.Asset != nil && (got.Asset == nil || *probe.Asset != *got.Asset) {
		return fmt.Errorf("probe/asset: %s got %+v want %+v", probe.Path, got.Asset, probe.Asset)
	}
	for _, key := range slices.Sorted(maps.Keys(probe.ResponseHeaders)) {
		if !slices.Equal(got.ResponseHeaders[key], probe.ResponseHeaders[key]) {
			return responseMismatchf("header/"+key, "probe/header: %s %s got %q, want %q", probe.Path, key, got.ResponseHeaders[key], probe.ResponseHeaders[key])
		}
	}
	if got.Actor != probe.Actor {
		return fmt.Errorf("probe/actor: %s got %q want %q", probe.Path, got.Actor, probe.Actor)
	}
	return nil
}

// componentVariant reports whether the asset is the selected component's module variant.
//
// Takes path (string) which is the referenced JavaScript asset path.
// Takes tag (string) which identifies the custom element.
//
// Returns bool which reports whether the path names the selected component variant.
func componentVariant(path, tag string) bool {
	const prefix = "/_piko/assets/generated/components/"
	if tag == "" || !strings.HasPrefix(path, prefix+tag+"_") {
		return false
	}
	remaining := strings.TrimPrefix(path, prefix+tag+"_")
	hash, ok := strings.CutSuffix(remaining, ".js")
	hash = strings.TrimSuffix(hash, ".min")
	if !ok || len(hash) != componentHashLength {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}

// logicalScripts normalises script references for server-mode comparison.
//
// Takes o (*Observation) which holds the application observation.
//
// Returns []string which lists script references with generated component assets replaced
// by their component identities.
func logicalScripts(o *Observation) []string {
	result := slices.Clone(o.Scripts)
	for index, source := range result {
		if source == o.ComponentAsset && componentVariant(source, o.ComponentTag) {
			result[index] = "component:" + o.ComponentTag
			continue
		}
		for tag, path := range o.ComponentModules {
			if source == path && componentVariant(source, tag) {
				result[index] = "component:" + tag
				break
			}
		}
	}
	return result
}

// compareObservations checks application markup, headers, metadata, script references and
// client results across two modes.
//
// Takes a (*Observation) which holds the first mode observation.
// Takes b (*Observation) which holds the second mode observation.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareObservations(a, b *Observation) error {
	if err := compareSubsystems(a, b); err != nil {
		return err
	}
	if err := compareResponseFields(a, b); err != nil {
		return err
	}
	if err := compareResponseContracts(a.Semantic, b.Semantic); err != nil {
		return fmt.Errorf("primary/%w", err)
	}
	if err := compareClient(a.Client, b.Client); err != nil {
		return err
	}
	if err := compareFormWorkflows(a.FormWorkflow, b.FormWorkflow); err != nil {
		return err
	}
	if err := compareProbes(a.Probes, b.Probes); err != nil {
		return err
	}
	left, err := publicActionResult(a.Action, a.Mode)
	if err != nil {
		return err
	}
	right, err := publicActionResult(b.Action, b.Mode)
	if err != nil {
		return err
	}
	return compareActions(left, right)
}

// compareSubsystems compares the auth, form-shim, topology and browser receipts across
// two modes.
//
// Takes a (*Observation) which holds the first mode observation.
// Takes b (*Observation) which holds the second mode observation.
//
// Returns error which reports the first differing subsystem, or nil when they agree.
func compareSubsystems(a, b *Observation) error {
	if err := compareAuth(a.Auth, b.Auth); err != nil {
		return err
	}
	if err := compareFormShim(a.FormShim, b.FormShim); err != nil {
		return err
	}
	if err := compareTopology(a.Topology, b.Topology); err != nil {
		return err
	}
	if a.Client == nil || b.Client == nil {
		return nil
	}
	if !reflect.DeepEqual(a.Client.FormShim, b.Client.FormShim) {
		return responseMismatchf("form-shim/browser/differential", "host browser action receipts differ")
	}
	if !reflect.DeepEqual(a.Client.DOM, b.Client.DOM) {
		return errors.New("browser/dom: observed browser states differ")
	}
	return nil
}

// compareResponseFields compares the primary response's scalar fields across two modes.
//
// Takes a (*Observation) which holds the first mode observation.
// Takes b (*Observation) which holds the second mode observation.
//
// Returns error which reports the first differing field, or nil when they agree.
func compareResponseFields(a, b *Observation) error {
	switch {
	case a.Status != b.Status:
		return fmt.Errorf("status: %d != %d", a.Status, b.Status)
	case a.ContentType != b.ContentType:
		return fmt.Errorf("content type: %q != %q", a.ContentType, b.ContentType)
	case a.Location != b.Location:
		return fmt.Errorf("redirect: %q != %q", a.Location, b.Location)
	case a.MetadataTitle != b.MetadataTitle:
		return fmt.Errorf("metadata title: %q != %q", a.MetadataTitle, b.MetadataTitle)
	case a.ApplicationDOM != b.ApplicationDOM:
		return fmt.Errorf("dom: application differs at byte %d", firstDifference(a.ApplicationDOM, b.ApplicationDOM))
	case !slices.Equal(logicalScripts(a), logicalScripts(b)):
		return fmt.Errorf("scripts: %q != %q", a.Scripts, b.Scripts)
	case !slices.Equal(a.Counts, b.Counts):
		return fmt.Errorf("browser/count: %q != %q", a.Counts, b.Counts)
	}
	return nil
}

// compareProbes compares follow-up responses position by position across two modes.
//
// Takes left ([]ProbeObservation) which holds the first mode's responses.
// Takes right ([]ProbeObservation) which holds the second mode's responses.
//
// Returns error which reports a count, contract or response difference, or nil.
func compareProbes(left, right []ProbeObservation) error {
	if len(left) != len(right) {
		return fmt.Errorf("probe/count: %d != %d", len(left), len(right))
	}
	for index := range left {
		first, second := &left[index], &right[index]
		var owner casefile.Probe
		owner.Path, owner.Actor = first.Path, first.Actor
		if err := compareResponseContracts(first.Semantic, second.Semantic); err != nil {
			return locateProbe(owner, fmt.Errorf("probe/%w on %s", err, first.Path))
		}
		if !probesEqual(first, second) {
			return locateProbe(owner, fmt.Errorf("probe/parity: response differs for request %d (%s)", index, first.Path))
		}
	}
	return nil
}

// firstDifference locates the first differing byte in two strings.
//
// Takes a (string) which holds the first value to compare.
// Takes b (string) which holds the second value to compare.
//
// Returns int which is the first differing byte position, or the shared prefix length.
func firstDifference(a, b string) int {
	for index := range min(len(a), len(b)) {
		if a[index] != b[index] {
			return index
		}
	}
	return min(len(a), len(b))
}

// probesEqual reports whether two modes' follow-up responses agree on every public field.
//
// Takes left (*ProbeObservation) which holds the first mode's response.
// Takes right (*ProbeObservation) which holds the second mode's response.
//
// Returns bool which reports whether the responses agree.
func probesEqual(left, right *ProbeObservation) bool {
	if (left.Asset == nil) != (right.Asset == nil) || left.Asset != nil && *left.Asset != *right.Asset {
		return false
	}
	return reflect.DeepEqual(left.ResponseHeaders, right.ResponseHeaders) &&
		left.Actor == right.Actor && left.Path == right.Path && left.Location == right.Location &&
		left.Status == right.Status && left.Text == right.Text && left.Program == right.Program &&
		left.ProgramRoot == right.ProgramRoot && left.RootHidden == right.RootHidden &&
		slices.Equal(left.Output, right.Output)
}

// compareClient compares browser interactions with their saved expectations.
//
// Takes actual (*casefile.ClientTransition) which holds the observed result.
// Takes expected (*casefile.ClientTransition) which holds the independently expected
// result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareClient(actual, expected *casefile.ClientTransition) error {
	if actual == nil && expected == nil {
		return nil
	}
	if actual == nil || expected == nil {
		return errors.New("browser/states: missing transition")
	}
	if _, err := checkCheckout(actual.Checkout, expected.Checkout); err != nil {
		return err
	}
	if _, err := checkPartialComponents(actual.PartialComponents, expected.PartialComponents); err != nil {
		return err
	}
	if err := checkEvents(actual.Events, expected.Events); err != nil {
		return err
	}
	if err := checkLifecycle(actual.Lifecycle, expected.Lifecycle); err != nil {
		return err
	}
	if err := compareInstances("initial", actual.Before, expected.Before); err != nil {
		return err
	}
	if err := compareInstances("after-click", actual.After, expected.After); err != nil {
		return err
	}
	return compareClientSteps(actual.Steps, expected.Steps)
}

// compareClientSteps compares each replayed browser action and its resulting state.
//
// Takes actual ([]casefile.ClientStep) which holds the observed steps.
// Takes expected ([]casefile.ClientStep) which holds the expected steps.
//
// Returns error which reports the first step mismatch, or nil when they agree.
func compareClientSteps(actual, expected []casefile.ClientStep) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("browser/steps: got %d, want %d", len(actual), len(expected))
	}
	for index := range expected {
		want, got := &expected[index], &actual[index]
		if got.Action != want.Action {
			return fmt.Errorf("browser/recipe: action %d got %+v, want %+v", index, got.Action, want.Action)
		}
		if got.Prevented != want.Prevented {
			return fmt.Errorf("browser/prevent: action %d got %t want %t", index, got.Prevented, want.Prevented)
		}
		if err := compareInstances(fmt.Sprintf("action %d (%s)", index, want.Action.Op), got.After, want.After); err != nil {
			return err
		}
	}
	return nil
}

// compareInstances compares component state without stopping at unrelated fields.
//
// Takes stage (string) which identifies the interaction stage.
// Takes observed ([]casefile.ClientInstance) which holds the observed component
// instances.
// Takes expected ([]casefile.ClientInstance) which holds the independently expected
// result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareInstances(stage string, observed, expected []casefile.ClientInstance) error {
	if len(observed) != len(expected) {
		return fmt.Errorf("browser/instances: %s got %d, want %d", stage, len(observed), len(expected))
	}
	for index := range observed {
		if err := compareInstance(stage, index, &observed[index], &expected[index]); err != nil {
			return err
		}
	}
	return nil
}

// compareInstance compares one component instance's saved evidence and state.
//
// Saved recipes specify the evidence that existed when they were created.
//
// Takes stage (string) which identifies the interaction stage.
// Takes index (int) which is the instance's position.
// Takes got (*casefile.ClientInstance) which holds the observed instance.
// Takes want (*casefile.ClientInstance) which holds the expected instance.
//
// Returns error which reports the first mismatched field, or nil when they agree.
func compareInstance(stage string, index int, got, want *casefile.ClientInstance) error {
	for _, field := range slices.Sorted(maps.Keys(want.Advanced)) {
		value, ok := got.Advanced[field]
		if !ok || value != want.Advanced[field] {
			return fmt.Errorf("browser/%s: %s instance %d got %q want %q", field, stage, index, value, want.Advanced[field])
		}
	}
	if (got.Controls == nil) != (want.Controls == nil) {
		return fmt.Errorf("browser/controls: %s instance %d missing extended controls", stage, index)
	}
	if got.Controls != nil {
		if err := compareControls(stage, index, got.Controls, want.Controls); err != nil {
			return err
		}
	}
	switch {
	case got.Count != want.Count:
		return fmt.Errorf("browser/count: %s instance %d got %q, want %q", stage, index, got.Count, want.Count)
	case got.Label != want.Label:
		return fmt.Errorf("browser/attribute: %s instance %d got %q, want %q", stage, index, got.Label, want.Label)
	case !slices.Equal(got.Rows, want.Rows):
		return fmt.Errorf("browser/rows: %s instance %d got %q, want %q", stage, index, got.Rows, want.Rows)
	case got.Visible != want.Visible:
		return fmt.Errorf("browser/condition: %s instance %d got %t, want %t", stage, index, got.Visible, want.Visible)
	case got.Slot != want.Slot:
		return fmt.Errorf("browser/slot: %s instance %d got %q, want %q", stage, index, got.Slot, want.Slot)
	case got.Lifecycle != want.Lifecycle:
		return fmt.Errorf("browser/lifecycle: %s instance %d got %q, want %q", stage, index, got.Lifecycle, want.Lifecycle)
	}
	return nil
}

// compareControls compares one instance's extended control observations.
//
// Takes stage (string) which identifies the interaction stage.
// Takes index (int) which is the instance's position.
// Takes got (*casefile.ClientControls) which holds the observed controls.
// Takes want (*casefile.ClientControls) which holds the expected controls.
//
// Returns error which reports the first mismatched control, or nil when they agree.
func compareControls(stage string, index int, got, want *casefile.ClientControls) error {
	if (got.Form == nil) != (want.Form == nil) {
		return fmt.Errorf("browser/form: %s instance %d missing form observations", stage, index)
	}
	if got.Form != nil {
		if err := compareFormControls(stage, index, got.Form, want.Form); err != nil {
			return err
		}
	}
	switch {
	case got.Input != want.Input:
		return fmt.Errorf("browser/input: %s instance %d got %q, want %q", stage, index, got.Input, want.Input)
	case got.HostLabel != want.HostLabel:
		return fmt.Errorf("browser/reflection: %s instance %d got %q, want %q", stage, index, got.HostLabel, want.HostLabel)
	case !slices.Equal(got.Keys, want.Keys):
		return fmt.Errorf("browser/key-identity: %s instance %d got %v, want %v", stage, index, got.Keys, want.Keys)
	case got.Handled != want.Handled || got.Bubbled != want.Bubbled:
		return fmt.Errorf("browser/propagation: %s instance %d got counts %s/%s, want %s/%s", stage, index, got.Handled, got.Bubbled, want.Handled, want.Bubbled)
	}
	return nil
}

// compareFormControls compares one instance's checkbox and select observations.
//
// Takes stage (string) which identifies the interaction stage.
// Takes index (int) which is the instance's position.
// Takes got (*casefile.ClientForm) which holds the observed form controls.
// Takes want (*casefile.ClientForm) which holds the expected form controls.
//
// Returns error which reports a checkbox or select mismatch, or nil when they agree.
func compareFormControls(stage string, index int, got, want *casefile.ClientForm) error {
	if got.Checked != want.Checked || got.HostChecked != want.HostChecked || got.CheckedState != want.CheckedState {
		return fmt.Errorf("browser/checkbox: %s instance %d native/host/state got %t/%t/%q want %t/%t/%q",
			stage, index, got.Checked, got.HostChecked, got.CheckedState, want.Checked, want.HostChecked, want.CheckedState)
	}
	if got.Choice != want.Choice || got.ChoiceState != want.ChoiceState || got.HostChoice != want.HostChoice {
		return fmt.Errorf("browser/select: %s instance %d native/host/state got %q/%q/%q want %q/%q/%q",
			stage, index, got.Choice, got.HostChoice, got.ChoiceState, want.Choice, want.HostChoice, want.ChoiceState)
	}
	return nil
}
