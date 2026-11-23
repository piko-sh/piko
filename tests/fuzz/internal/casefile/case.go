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

// Package casefile stores self-contained Piko projects and their independent
// expectations.
package casefile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"piko.sh/json"
)

const (
	// Version identifies the saved-case schema version.
	Version = 1

	// Module is the fixed generated application module path.
	Module = "example.org/pikofuzzcase"

	// CollectionModule is the fixed external collection module path.
	CollectionModule = "example.org/pikofuzzcontent"

	// CollectionDir is the external collection's saved source directory.
	CollectionDir = "collection-source"

	// maxFile bounds one saved source file in bytes.
	maxFile = 1 << 20

	// maxTree bounds the total saved source bytes of one case.
	maxTree = 8 * maxFile

	// maxFiles bounds the number of saved source files in one case.
	maxFiles = 128

	// temporaryPrefix names the staging directories Write creates beside a case.
	temporaryPrefix = ".case-"

	// staleTemporaryAge is the age after which an abandoned staging directory is removed.
	staleTemporaryAge = time.Hour

	// directoryPermissions is the mode of every directory a case write creates.
	directoryPermissions = 0o700

	// filePermissions is the mode of every file a case write creates.
	filePermissions = 0o600

	// manifestName is the saved case manifest's file name.
	manifestName = "case.json"

	// sourceDirectory is the directory holding a case's saved sources.
	sourceDirectory = "src"
)

const (
	// SetInput selects the browser operation that sets the component input.
	SetInput ClientOp = "set-input"

	// SetLabel selects the browser operation that changes the reflected component label.
	SetLabel ClientOp = "set-label"

	// ReverseRows selects the browser operation that reverses the keyed rows.
	ReverseRows ClientOp = "reverse-rows"

	// RemoveRow selects the browser operation that removes a keyed row.
	RemoveRow ClientOp = "remove-row"

	// ModifiedClick selects the browser operation that dispatches a click with event
	// modifiers.
	ModifiedClick ClientOp = "modified-click"

	// ParentClick selects the browser operation that clicks the parent component.
	ParentClick ClientOp = "parent-click"

	// ToggleCheck selects the browser operation that toggles the checkbox.
	ToggleCheck ClientOp = "toggle-check"

	// SelectOption selects the browser operation that selects an option.
	SelectOption ClientOp = "select-option"

	// ResetControls selects the browser operation that resets native controls.
	ResetControls ClientOp = "reset-controls"

	// ChooseRadio selects the browser operation that selects the blue radio option.
	ChooseRadio ClientOp = "radio-blue"

	// SelectMany selects the browser operation that selects multiple options.
	SelectMany ClientOp = "select-many"

	// TypeNumber selects the browser operation that enters a numeric value.
	TypeNumber ClientOp = "type-number"

	// RefEdit selects the browser operation that edits through a component reference.
	RefEdit ClientOp = "ref-edit"

	// HideMemo selects the browser operation that hides memoised content.
	HideMemo ClientOp = "hide-memo"

	// ShowRefocus selects the browser operation that shows content and restores focus.
	ShowRefocus ClientOp = "show-refocus"

	// ChildPick selects the browser operation that selects a child value.
	ChildPick ClientOp = "child-pick"

	// RemountChild selects the browser operation that remounts the child component.
	RemountChild ClientOp = "remount-child"

	// SeekTimeline selects the browser operation that seeks the component timeline.
	SeekTimeline ClientOp = "seek-timeline"
)

var (
	// errInvalidPath reports an unsafe or non-canonical saved source path.
	errInvalidPath = errors.New("invalid source path")

	// errSizeLimit reports a saved file or source tree above its byte limit.
	errSizeLimit = errors.New("case size limit exceeded")

	// errFileCountLimit reports a source tree with too many files.
	errFileCountLimit = errors.New("case file count limit exceeded")

	// errDestinationExists reports a case destination that is already present.
	errDestinationExists = errors.New("case destination exists")

	// errInvalidManifest reports an unsupported or malformed case manifest.
	errInvalidManifest = errors.New("unsupported or invalid case manifest")
)

// Probe is a follow-up request in the same application lifetime.
type Probe struct {
	// Cookies supplies cookie values for the actor's jar.
	Cookies map[string]string `json:",omitempty"`

	// ByMode holds explicit mode-specific response contracts.
	ByMode map[string]ModeResponse `json:",omitempty"`

	// Semantic holds the selected semantic response fields.
	Semantic *ResponseContract `json:",omitempty"`

	// Asset holds delivered resource bytes or their expected identity.
	Asset *ResponseAsset `json:",omitempty"`

	// ResponseHeaders holds the response header values.
	ResponseHeaders map[string][]string `json:",omitempty"`

	// Actor selects the request's session.
	Actor string `json:",omitempty"`

	// Stage identifies the request's interaction stage.
	Stage string `json:",omitempty"`

	// IfNoneMatch supplies the request's conditional entity tag.
	IfNoneMatch string `json:",omitempty"`

	// ProgramRoot is the expected program root tag.
	ProgramRoot string `json:",omitempty"`

	// Path is the request or source path.
	Path string

	// Location is the response's redirect location.
	Location string

	// Text is the expected response text.
	Text string

	// Output holds the ordered program leaves.
	Output []ProgramOutput `json:",omitempty"`

	// Status is the HTTP response status.
	Status int

	// BrowserProgram selects this program route for browser observation.
	BrowserProgram bool `json:",omitempty"`

	// Program reports whether program output is checked.
	Program bool `json:",omitempty"`
}

// ProgramOutput records one program leaf in document order.
type ProgramOutput struct {
	// ID identifies the leaf.
	ID string

	// Text is the leaf's rendered text.
	Text string

	// Value is the leaf's value attribute.
	Value string

	// ScopedKey holds the expected scope endpoints.
	ScopedKey string `json:",omitempty"`

	// FullKey holds the complete rendered scope path.
	FullKey string `json:",omitempty"`

	// Owner is the DOM ancestor ownership path.
	Owner string `json:",omitempty"`

	// Hidden reports whether the leaf is hidden.
	Hidden bool `json:",omitempty"`

	// HasValue distinguishes an absent value attribute from an empty one.
	HasValue bool
}

// ClientInstance describes observable shadow DOM and projected slot content.
type ClientInstance struct {
	// Advanced holds the extended component's public state.
	Advanced map[string]string `json:",omitempty"`

	// Controls holds native control and node-identity state.
	Controls *ClientControls `json:",omitempty"`

	// Count is the displayed counter value.
	Count string

	// Label is the displayed label.
	Label string

	// Slot holds the projected slot text.
	Slot string

	// Lifecycle records the component's lifecycle text.
	Lifecycle string

	// Rows holds the ordered row values.
	Rows []string

	// Visible reports whether the content is visible.
	Visible bool
}

// ClientControls describes rendered controls and node identities within one shadow root.
type ClientControls struct {
	// Form holds native and reflected form control state.
	Form *ClientForm `json:",omitempty"`

	// Input is the native input's value.
	Input string

	// HostLabel is the label reflected onto the host element.
	HostLabel string

	// Handled is the counter of events handled inside the component.
	Handled string

	// Bubbled is the counter of events that bubbled out of the component.
	Bubbled string

	// Keys holds row-node identity markers.
	Keys []int
}

// ClientForm compares native controls, rendered state and reflected host attributes.
type ClientForm struct {
	// CheckedState is the rendered checkbox state.
	CheckedState string

	// Choice is the native choice value.
	Choice string

	// ChoiceState is the rendered choice value.
	ChoiceState string

	// HostChoice is the choice value reflected onto the host element.
	HostChoice string

	// Checked is the native checkbox state.
	Checked bool

	// HostChecked is the checkbox state reflected onto the host element.
	HostChecked bool
}

// ClientOp names a closed set of DOM operations understood by the local browser worker.
type ClientOp string

// ClientAction selects an operation on one component instance.
type ClientAction struct {
	// Op selects the operation.
	Op ClientOp

	// Value holds the operation's value.
	Value string `json:",omitempty"`

	// Instance selects the component instance.
	Instance int
}

// ClientStep pairs a browser action with its resulting component states.
type ClientStep struct {
	// Action is the performed component operation and its input.
	Action ClientAction

	// After records the state after the interaction.
	After []ClientInstance

	// Prevented reports whether the action prevented its default behaviour.
	Prevented bool `json:",omitempty"`
}

// ClientTransition records public browser interactions and component states.
type ClientTransition struct {
	// FormShim holds form-client interaction evidence.
	FormShim *FormShimBrowser `json:",omitempty"`

	// PrimaryBrowserVersion identifies the primary browser version.
	PrimaryBrowserVersion string `json:",omitempty"`

	// Checkout holds checkout interaction evidence.
	Checkout *CheckoutTrace `json:",omitempty"`

	// DOM holds the DOM operation plan or trace.
	DOM *DOMTrace `json:",omitempty"`

	// PartialComponents holds partial-component interaction evidence.
	PartialComponents *PartialComponentsTrace `json:",omitempty"`

	// BrowserRoutes holds route navigation evidence.
	BrowserRoutes []BrowserRouteVisit `json:",omitempty"`

	// Events holds the component event journey trace.
	Events *EventsTrace `json:",omitempty"`

	// Lifecycle holds component lifecycle evidence.
	Lifecycle *LifecycleTrace `json:",omitempty"`

	// Steps holds the ordered interaction steps.
	Steps []ClientStep `json:",omitempty"`

	// Before holds the initial component snapshots.
	Before []ClientInstance

	// After holds the final component snapshots.
	After []ClientInstance
}

// ActionValue holds the action's label and quantity.
type ActionValue struct {
	// Label is the action label or payload.
	Label string

	// Quantity is the requested item quantity.
	Quantity int
}

// ActionEvent pairs an event name with its action value.
type ActionEvent struct {
	// Name is the server-sent event type.
	Name string

	// Value holds the event's action value.
	Value ActionValue
}

// ActionRefusal specifies the public error returned by an action.
type ActionRefusal struct {
	// Errors holds public field-validation errors.
	Errors map[string]string `json:",omitempty"`

	// Operation names the refused operation.
	Operation string

	// Code is the public error code.
	Code string

	// Message is the public error message.
	Message string

	// Status is the HTTP response status.
	Status int
}

// ActionContract specifies action results and optional transport checks.
type ActionContract struct {
	// Refusals holds expected action refusals.
	Refusals []ActionRefusal `json:",omitempty"`

	// Input holds the supplied interaction values.
	Input ActionValue

	// Result holds the action result.
	Result ActionValue

	// Events holds the emitted events.
	Events []ActionEvent

	// Batch enables batch-dispatch checks.
	Batch bool `json:",omitempty"`

	// Notifications enables notification delivery checks.
	Notifications bool `json:",omitempty"`

	// Cache enables action-cache checks.
	Cache bool `json:",omitempty"`

	// Upload enables upload checks.
	Upload bool `json:",omitempty"`

	// UploadRemoval enables uploaded-file removal checks.
	UploadRemoval bool `json:",omitempty"`

	// UploadPublic enables public uploaded-file checks.
	UploadPublic bool `json:",omitempty"`

	// Actors enables per-actor state checks.
	Actors bool `json:",omitempty"`

	// CancelStream enables stream-cancellation checks.
	CancelStream bool `json:",omitempty"`
}

// Expected is the observable contract of one generated request and client interaction.
type Expected struct {
	// Action holds the action contract.
	Action *ActionContract `json:",omitempty"`

	// FormShim holds the form-client contract.
	FormShim *FormShimContract `json:"formShim,omitempty"`

	// Diagnostic holds the paired-source diagnostic contract.
	Diagnostic *DiagnosticPair `json:",omitempty"`

	// Auth holds the authentication contract.
	Auth *AuthContract `json:"auth,omitempty"`

	// Topology holds the Go-host state contract.
	Topology *GoTopology `json:",omitempty"`

	// DOM holds the public DOM operation plan.
	DOM *DOMPlan `json:",omitempty"`

	// Semantic holds the semantic response contract.
	Semantic *ResponseContract `json:",omitempty"`

	// FormWorkflow holds the form command history.
	FormWorkflow *FormWorkflow `json:",omitempty"`

	// Client holds expected component transitions.
	Client *ClientTransition `json:",omitempty"`

	// Title is the page title.
	Title string

	// Tag is the HTML element name.
	Tag string

	// AdditionalComponents lists companion custom-element tag names.
	AdditionalComponents []string `json:",omitempty"`

	// Editions holds source edits and their expected responses.
	Editions []Edition `json:",omitempty"`

	// Rows holds the ordered row values.
	Rows []string

	// Count is the expected component counter.
	Count int

	// Status is the expected HTTP status.
	Status int

	// Frame reports whether the generated application uses a frame.
	Frame bool `json:",omitempty"`

	// Show selects conditional content visibility.
	Show bool
}

// Case stores the generator identity, input recording and resulting project.
type Case struct {
	// Files maps source paths to their contents.
	Files map[string]string `json:"-"`

	// Request is the initial request path.
	Request string

	// Size selects the structural generation bounds.
	Size string `json:",omitempty"`

	// Generator identifies the source generator version.
	Generator string

	// Profile identifies the generation profile.
	Profile string

	// ProfileSHA256 identifies the embedded generator sources.
	ProfileSHA256 string `json:",omitempty"`

	// Target selects the subsystem under test.
	Target string

	// Probes holds follow-up requests in execution order.
	Probes []Probe

	// Data holds the replayable choice recording.
	Data []byte

	// Features lists generated source features.
	Features []string

	// Paths lists the saved source paths.
	Paths []string

	// Expected holds the independently modelled result.
	Expected Expected

	// Index identifies the case's position in the stream.
	Index uint64

	// Seed initialises the deterministic choice stream.
	Seed uint64

	// Version identifies the saved-case schema.
	Version int

	// BrowserSampleVersion selects the saved browser-sampling policy.
	BrowserSampleVersion int `json:",omitempty"`
}

// caseIdentity selects the case fields that contribute to its content identity.
type caseIdentity struct {
	// Files maps source paths to their contents.
	Files map[string]string

	// Request is the initial request path.
	Request string

	// Probes holds follow-up requests in execution order.
	Probes []Probe

	// Data holds the replayable choice recording.
	Data []byte

	// Expected holds the independently modelled result.
	Expected Expected
}

// RequiresProjectInterpreterHost reports whether native registrations from this specific
// application must be linked into the interpreted server.
//
// Returns bool which reports whether native registrations must come from this
// application.
func (c *Case) RequiresProjectInterpreterHost() bool {
	expected := &c.Expected
	return expected.Auth != nil || expected.FormShim != nil || expected.Action != nil || expected.FormWorkflow != nil ||
		expected.Topology != nil || c.UsesCollections() || c.Target == "partial-components" || c.Target == "checkout-workflow"
}

// Hash identifies the saved case's metadata and source bytes.
//
// Hash cannot fail: it hashes the canonical encoding of strings, integers, booleans, byte
// slices, maps and slices only. Should encoding ever fail, it returns an "unhashable:"
// marker with a random suffix, so the marker matches no digest and no two markers compare
// equal.
//
// Returns string which is the saved case's content identity.
func (c *Case) Hash() string {
	digest, err := CanonicalSHA256(caseIdentity{Files: c.Files, Request: c.Request, Expected: c.Expected, Probes: c.Probes, Data: c.Data})
	if err != nil {
		return "unhashable: " + err.Error() + " " + rand.Text()
	}
	return digest
}

// Write publishes a case only after all saved source files are durable.
//
// Takes dir (string) which is the new case destination directory.
// Takes c (*Case) which holds the saved sources, choices and expected results; it is not
// modified.
//
// Returns error which reports a byte limit or file-publication failure, wrapped with the
// destination, or nil on success.
func Write(dir string, c *Case) error {
	owned := *c
	if err := write(dir, &owned); err != nil {
		return fmt.Errorf("writing case %s: %w", dir, err)
	}
	return nil
}

// Read loads saved source bytes without invoking the generator.
//
// Takes dir (string) which is the saved case directory.
//
// Returns Case which holds the saved sources, choices and expected results.
// Returns error which reports saved-file access, bounds or manifest validation failure,
// wrapped with the case directory, or nil on success.
func Read(dir string) (Case, error) {
	c, err := readCase(dir)
	if err != nil {
		return c, fmt.Errorf("reading case %s: %w", dir, err)
	}
	return c, nil
}

// CanonicalSHA256 hashes a value's JSON encoding with object keys sorted at every level.
//
// The value is encoded, decoded into generic maps and slices, and encoded again, so
// struct field order does not change the digest. Numbers keep their exact text. The
// standard library encoder keeps the digest independent of the JSON provider.
//
// Takes value (any) which is the value to identify.
//
// Returns string which is the hexadecimal SHA-256 of the canonical encoding.
// Returns error which reports a value that cannot be encoded, or nil on success.
func CanonicalSHA256(value any) (string, error) {
	data, err := stdjson.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encoding digest input: %w", err)
	}
	decoder := stdjson.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var tree any
	if err := decoder.Decode(&tree); err != nil {
		return "", fmt.Errorf("decoding digest input: %w", err)
	}
	canonical, err := stdjson.Marshal(tree)
	if err != nil {
		return "", fmt.Errorf("encoding canonical digest input: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// validatePath rejects unsafe saved-source paths.
//
// Takes name (string) which is the saved source path to validate.
//
// Returns error which reports an unsafe or non-canonical relative source path, or nil on
// success.
func validatePath(name string) error {
	if !fs.ValidPath(name) || !filepath.IsLocal(name) || strings.ContainsAny(name, "\\:") || name == "." || path.Clean(name) != name {
		return fmt.Errorf("%w %q", errInvalidPath, name)
	}
	return nil
}

// write stages, syncs and publishes a case directory.
//
// Takes dir (string) which is the new case destination directory.
// Takes c (*Case) which holds the saved sources, choices and expected results; its Paths
// are replaced with the sorted source paths.
//
// Returns error which reports a byte limit or file-publication failure, or nil on
// success.
func write(dir string, c *Case) error {
	data, err := encodeManifest(c)
	if err != nil {
		return err
	}
	parent := filepath.Dir(dir)
	if err := claimDestination(parent, dir); err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.Remove(dir)
		}
	}()

	staging, err := os.MkdirTemp(parent, temporaryPrefix)
	if err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := stageCase(staging, c, data); err != nil {
		return err
	}
	if err := syscall.Rename(staging, dir); err != nil {
		return fmt.Errorf("publishing case: %w", err)
	}
	published = true
	return syncDirectory(parent)
}

// encodeManifest validates a case's source tree and encodes its manifest.
//
// Takes c (*Case) which holds the saved sources; its Paths are replaced with the sorted
// source paths.
//
// Returns []byte which holds the indented manifest without a trailing newline.
// Returns error which reports an unsafe path or a byte or file-count limit, or nil on
// success.
func encodeManifest(c *Case) ([]byte, error) {
	if len(c.Files) > maxFiles {
		return nil, fmt.Errorf("%w: %d source files", errFileCountLimit, len(c.Files))
	}
	c.Paths = slices.Sorted(maps.Keys(c.Files))
	total := 0
	for _, name := range c.Paths {
		if err := validatePath(name); err != nil {
			return nil, err
		}
		size := len(c.Files[name])
		total += size
		if size > maxFile || total > maxTree {
			return nil, fmt.Errorf("%w: source tree at %s", errSizeLimit, name)
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding manifest: %w", err)
	}
	if len(data)+1 > maxFile {
		return nil, fmt.Errorf("%w: manifest", errSizeLimit)
	}
	return data, nil
}

// claimDestination creates the parent directory and claims the empty case destination.
//
// Takes parent (string) which is the directory holding cases and staging directories.
// Takes dir (string) which is the new case destination directory.
//
// Returns error which reports an existing destination or a directory failure, or nil on
// success.
func claimDestination(parent, dir string) error {
	if err := os.MkdirAll(parent, directoryPermissions); err != nil {
		return fmt.Errorf("creating parent: %w", err)
	}
	if err := removeStaleTemporaries(parent, time.Now()); err != nil {
		return err
	}
	if err := os.Mkdir(dir, directoryPermissions); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return errDestinationExists
		}
		return fmt.Errorf("claiming destination: %w", err)
	}
	return nil
}

// stageCase writes and syncs every saved file inside the staging directory.
//
// Takes staging (string) which is the staging directory confining every write.
// Takes c (*Case) which holds the saved sources and their sorted paths.
// Takes manifest ([]byte) which holds the encoded manifest.
//
// Returns error which reports a write or sync failure, or nil on success.
func stageCase(staging string, c *Case, manifest []byte) (err error) {
	root, err := os.OpenRoot(staging)
	if err != nil {
		return fmt.Errorf("opening staging directory: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing staging directory: %w", closeErr)
		}
	}()
	for _, name := range c.Paths {
		if err := writeDurable(root, path.Join(sourceDirectory, name), []byte(c.Files[name])); err != nil {
			return err
		}
	}
	if err := writeDurable(root, manifestName, append(manifest, '\n')); err != nil {
		return err
	}
	return syncTree(root)
}

// writeDurable writes a file, creating its parent directories, and syncs its contents.
//
// Takes root (*os.Root) which confines the write to the staging directory.
// Takes name (string) which is the slash-separated destination path inside root.
// Takes data ([]byte) which holds the file contents.
//
// Returns error which reports a create, write, sync or close failure, or nil on success.
func writeDurable(root *os.Root, name string, data []byte) error {
	if err := root.MkdirAll(path.Dir(name), directoryPermissions); err != nil {
		return fmt.Errorf("creating directory for %s: %w", name, err)
	}
	file, err := root.OpenFile(filepath.FromSlash(name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
	if err != nil {
		return fmt.Errorf("creating %s: %w", name, err)
	}
	_, writeErr := file.Write(data)
	var syncErr error
	if writeErr == nil {
		syncErr = file.Sync()
	}
	if err := errors.Join(writeErr, syncErr, file.Close()); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}

// syncTree syncs every directory below and including the root.
//
// Takes root (*os.Root) which is the staged case directory.
//
// Returns error which reports a walk or directory sync failure, or nil on success.
func syncTree(root *os.Root) error {
	return fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		return syncRootDirectory(root, name)
	})
}

// syncDirectory makes a directory's entries durable.
//
// Takes dir (string) which is the directory to sync.
//
// Returns error which reports an open, sync or close failure, or nil on success.
func syncDirectory(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return fmt.Errorf("opening directory %s: %w", dir, err)
	}
	return errors.Join(syncRootDirectory(root, "."), root.Close())
}

// syncRootDirectory makes the entries of a directory inside a root durable.
//
// Takes root (*os.Root) which confines the directory.
// Takes name (string) which is the slash-separated directory path inside root.
//
// Returns error which reports an open, sync or close failure, or nil on success.
func syncRootDirectory(root *os.Root, name string) error {
	handle, err := root.Open(filepath.FromSlash(name))
	if err != nil {
		return fmt.Errorf("opening directory %s: %w", name, err)
	}
	if err := errors.Join(handle.Sync(), handle.Close()); err != nil {
		return fmt.Errorf("syncing directory %s: %w", name, err)
	}
	return nil
}

// removeStaleTemporaries removes staging directories abandoned by interrupted writes.
//
// Directories younger than staleTemporaryAge may belong to a concurrent writer and are
// kept.
//
// Takes parent (string) which is the directory holding cases and staging directories.
// Takes now (time.Time) which is the current time.
//
// Returns error which reports an unreadable parent or a failed removal, or nil on
// success.
func removeStaleTemporaries(parent string, now time.Time) error {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return fmt.Errorf("listing %s: %w", parent, err)
	}
	var removeErrs []error
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), temporaryPrefix) {
			continue
		}
		info, err := entry.Info()
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			removeErrs = append(removeErrs, fmt.Errorf("inspecting %s: %w", entry.Name(), err))
			continue
		}
		if now.Sub(info.ModTime()) < staleTemporaryAge {
			continue
		}
		if err := os.RemoveAll(filepath.Join(parent, entry.Name())); err != nil {
			removeErrs = append(removeErrs, fmt.Errorf("removing stale %s: %w", entry.Name(), err))
		}
	}
	return errors.Join(removeErrs...)
}

// read reads a saved source through its confined root.
//
// Takes root (*os.Root) which confines reads to the saved source directory.
// Takes name (string) which is the source path relative to the confined root.
//
// Returns []byte which holds the saved source contents.
// Returns error which reports saved-file access or bounds failure, or nil on success.
func read(root *os.Root, name string) (data []byte, err error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", name, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", name, closeErr)
		}
	}()
	data, err = io.ReadAll(io.LimitReader(file, maxFile+1))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	if len(data) > maxFile {
		return nil, fmt.Errorf("%w: %s exceeds %d bytes", errSizeLimit, name, maxFile)
	}
	return data, nil
}

// readCase loads and validates a saved case directory.
//
// Takes dir (string) which is the saved case directory.
//
// Returns Case which holds the saved sources, choices and expected results.
// Returns error which reports saved-file access, bounds or manifest validation failure,
// or nil on success.
func readCase(dir string) (c Case, err error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return c, fmt.Errorf("opening: %w", err)
	}
	defer func() {
		if closeErr := root.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing: %w", closeErr)
		}
	}()
	data, err := read(root, manifestName)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("decoding manifest: %w", err)
	}
	if c.Version != Version || c.Target == "" {
		return c, errInvalidManifest
	}
	if len(c.Paths) > maxFiles {
		return c, fmt.Errorf("%w: %d source files", errFileCountLimit, len(c.Paths))
	}
	if err := readSources(root, &c); err != nil {
		return c, err
	}
	return c, nil
}

// readSources loads every saved source listed by a case manifest.
//
// Takes root (*os.Root) which confines reads to the saved case directory.
// Takes c (*Case) which holds the manifest's paths and receives the source contents.
//
// Returns error which reports an invalid or duplicate path, a read failure or a size
// limit, or nil on success.
func readSources(root *os.Root, c *Case) error {
	c.Files = make(map[string]string, len(c.Paths))
	total := 0
	for _, name := range c.Paths {
		if err := validatePath(name); err != nil {
			return err
		}
		if _, ok := c.Files[name]; ok {
			return fmt.Errorf("%w: duplicate path %s", errInvalidManifest, name)
		}
		source, err := read(root, filepath.Join(sourceDirectory, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		total += len(source)
		if total > maxTree {
			return fmt.Errorf("%w: source tree at %s", errSizeLimit, name)
		}
		c.Files[name] = string(source)
	}
	return nil
}
