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

package interp_domain

import (
	"context"
	"io"
	"slices"

	"piko.sh/piko/internal/safeerror"
)

// SessionDeclKind classifies a session-scope declaration so the session can perform
// name-based redeclaration checks and so consumers (REPL, notebook UI) can render the
// right glyph next to each entry.
type SessionDeclKind int

const (
	// SessionDeclVar identifies a session-scope `var` declaration.
	SessionDeclVar SessionDeclKind = iota + 1

	// SessionDeclFunc identifies a session-scope function declaration.
	SessionDeclFunc

	// SessionDeclType identifies a session-scope `type` declaration.
	SessionDeclType

	// SessionDeclConst identifies a session-scope `const` declaration.
	SessionDeclConst
)

// String returns the lowercase Go keyword for the declaration kind.
//
// Returns string which is the kind's keyword, or "?" when unknown.
func (k SessionDeclKind) String() string {
	switch k {
	case SessionDeclVar:
		return "var"
	case SessionDeclFunc:
		return "func"
	case SessionDeclType:
		return "type"
	case SessionDeclConst:
		return "const"
	default:
		return "?"
	}
}

// SessionDecl describes a single session-scope declaration. The pair (Name, Kind) is the
// redeclaration key; two decls with the same Name are considered to clash regardless of
// Kind.
type SessionDecl struct {
	// Name is the bare identifier declared.
	Name string

	// Kind is the declaration kind (var / func / type / const).
	Kind SessionDeclKind
}

// SessionState is a point-in-time snapshot of what the session currently knows. It is
// intended for human-facing tooling (REPL `:inspect`, notebook outline panel) and is safe
// to retain past the Inspect() call that produced it.
type SessionState struct {
	// Imports lists every import path currently in scope. Sorted.
	Imports []string

	// Declarations lists every declared name in declaration order.
	Declarations []SessionDecl

	// SubmitCount is the number of successful Submit calls.
	SubmitCount int
}

// SessionOption configures a Session at construction.
type SessionOption func(*sessionConfig)

// sessionConfig holds the unexported per-session configuration.
type sessionConfig struct{}

// Session accumulates state across Submit calls to provide REPL-style semantics on top of
// Service. State preserved across submissions: declared names (var/func/type/const),
// imports, package-level variable values, and compiled function bytecode.
//
// A Session is NOT safe for concurrent use; each Submit must complete before the next
// begins. Multiple sessions may share a Service, but they also share its underlying
// global value store, so distinct REPL workspaces should use distinct Services.
type Session struct {
	// service is the backing interpreter. Symbols, limits, features, and globals come from
	// here; the session owns nothing the service also owns.
	service *Service

	// rootFunction holds the persistent function table that grows as new session-scope
	// functions compile. Lazy-initialised on first Submit so a discarded session allocates
	// nothing.
	rootFunction *CompiledFunction

	// imports is the set of import paths currently in scope. Insertion order is preserved
	// separately via decls so Inspect output is deterministic.
	imports map[string]struct{}

	// declaredNames maps a declared identifier to its kind, used for O(1) redeclaration
	// detection before re-Check.
	declaredNames map[string]SessionDeclKind

	// funcTable maps function name to index in rootFunction.functions. Persistent across
	// Submits so the per-Submit compiler resolves references to previously-declared
	// functions via the standard compileIdent path.
	funcTable map[string]uint16

	// globalVariables maps package-level variable name to its index and register kind in the
	// backing globalStore. Persistent across Submits so identifier resolution sees
	// previously-declared vars.
	globalVariables map[string]globalVariableInfo

	// compiledDecls tracks declaration names whose bytecode has been emitted; the per-Submit
	// compile loop skips entries in this set rather than re-walking and clashing on indices.
	compiledDecls map[string]bool

	// executedInits tracks init function indices that have already run, gating Go's
	// once-per-program init() semantics for the session.
	executedInits map[uint16]bool

	// compiledInits tracks init function bodies that have already been registered and
	// executed in this session, keyed by a printer-rendered representation of the init decl.
	// Re-encountering the same init in a later Submit is a no-op (would otherwise
	// re-register and re-run the init each time, breaking init-runs-once semantics).
	compiledInits map[string]bool

	// importOrder remembers the order in which imports were first introduced, for
	// deterministic Inspect rendering.
	importOrder []string

	// decls is the ordered list of session-scope declarations as the user introduced them.
	// Each entry's source is appended verbatim (post-shortVar rewrite) to the synthesised
	// file on every Submit.
	decls []declRecord

	// submitCount counts successful Submit calls. Failed Submits do not increment.
	submitCount int
}

// declRecord captures everything Session needs to reconstruct the combined source for
// re-Check and to render Inspect output.
type declRecord struct {
	// name is the bare identifier introduced by this declaration.
	name string

	// source is the verbatim source text the user submitted (after optional short-var
	// rewrite). Appended to the synthesised file on every Submit.
	source string

	// kind is the declaration's keyword class.
	kind SessionDeclKind
}

// NewSession returns a fresh session backed by this service.
//
// The session inherits the service's symbol registry, limits, and global store; each
// session owns its own declaration history. Multiple sessions on the same service share
// global value cells, so use distinct services when isolation is required.
//
// Takes opts (SessionOption variadic) configuring the session.
//
// Returns *Session ready to accept Submit calls.
func (s *Service) NewSession(opts ...SessionOption) *Session {
	config := &sessionConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return &Session{
		service:       s,
		imports:       make(map[string]struct{}),
		declaredNames: make(map[string]SessionDeclKind),
	}
}

// Service returns the underlying interpreter for callers that need to reach below the
// session abstraction (e.g. to register additional host symbols mid-session).
//
// Returns *Service which is the backing service.
func (sess *Session) Service() *Service {
	return sess.service
}

// SetStderr redirects print/println output for every subsequent Submit. REPL and notebook
// hosts use this to capture output for display in their own UI rather than letting it
// leak to the real process stderr (where it would be invisible inside a Bubble Tea
// alt-screen, for example).
//
// Takes writer (io.Writer) which receives output, or nil to reset to the default.
func (sess *Session) SetStderr(writer io.Writer) {
	sess.service.SetStderr(writer)
}

// Submit evaluates one user submission against the session, returning the result of the
// trailing expression (or nil when the submission ends in a statement). State accumulates
// only on success: a failed Submit leaves the session exactly as it was before the call.
//
// Takes ctx (context.Context) for cancellation and deadlines.
// Takes code (string) which is the Go source the user submitted.
//
// Returns any which is the value of the trailing expression, or nil.
// Returns error when preprocessing, classification, type-checking, compilation, or
// execution fails.
func (sess *Session) Submit(ctx context.Context, code string) (any, error) {
	result, err := sess.submit(ctx, code)
	if err != nil {
		return nil, safeerror.NewError(safeMessageSubmissionFailed, err)
	}
	return result, nil
}

// Reset clears all session-scope declarations and resets the backing service's global
// value store. The symbol registry (host packages) and the service itself are preserved.
func (sess *Session) Reset() {
	sess.imports = make(map[string]struct{})
	sess.importOrder = sess.importOrder[:0]
	sess.decls = sess.decls[:0]
	sess.declaredNames = make(map[string]SessionDeclKind)
	sess.rootFunction = nil
	sess.funcTable = nil
	sess.globalVariables = nil
	sess.compiledDecls = nil
	sess.executedInits = nil
	sess.compiledInits = nil
	sess.submitCount = 0
	sess.service.Reset()
}

// CompiledFunctions returns the session's accumulated function table.
//
// The table contains every CompiledFunction emitted across all Submits, in registration
// order. The first call after a fresh session (or after Reset) returns nil. The returned
// slice aliases internal state and must not be mutated; callers that need to retain it
// should copy first.
//
// Returns []*CompiledFunction in registration order.
func (sess *Session) CompiledFunctions() []*CompiledFunction {
	if sess.rootFunction == nil {
		return nil
	}
	return sess.rootFunction.functions
}

// Inspect returns a snapshot of what the session currently knows. The returned value owns
// its slices and is safe to keep past this call.
//
// Returns SessionState which is the snapshot.
func (sess *Session) Inspect() SessionState {
	imports := make([]string, len(sess.importOrder))
	copy(imports, sess.importOrder)
	slices.Sort(imports)

	declarations := make([]SessionDecl, len(sess.decls))
	for index, record := range sess.decls {
		declarations[index] = SessionDecl{Name: record.name, Kind: record.kind}
	}

	return SessionState{
		Imports:      imports,
		Declarations: declarations,
		SubmitCount:  sess.submitCount,
	}
}
