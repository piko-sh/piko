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
	"os"
	"reflect"
)

// CapabilityHook is consulted before every gated operation that crosses from interpreted
// code into native side effects. A hook returning a non-nil error aborts the operation
// with that error surfaced as the call's failure value; interpreted code can recover or
// propagate as with any other Go error.
//
// The modulePath argument identifies which loaded module made the call. For calls
// originating from the main policy/script body (not a loaded module), modulePath is the
// empty string. Host implementations use modulePath to scope decisions per-module, e.g.
// allowing "github.com/foo/bar" to do network I/O while denying it for everything else.
//
// A nil CapabilityHook is treated as "no gating" by every consumer here, so hosts that do
// not opt in run without gating. Production hosts that load untrusted code MUST install a
// hook; the default permissive behaviour exists only to keep the opt-out path working.
//
// Hook methods may be called concurrently from goroutines spawned by interpreted code;
// implementations must be safe for concurrent use.
type CapabilityHook interface {
	// CheckFunctionCall is consulted before any native function dispatch.
	//
	// Covers the slow path through reflect and the fast path through specialised
	// trampolines. fnPath is the dotted package.symbol identifier such as "net/http.Get" or
	// "(*bytes.Buffer).Write". args are the reflected argument values about to be passed;
	// implementations may inspect them but must not mutate them.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes fnPath (string) which is the dotted package.symbol identifier of the target.
	// Takes args ([]reflect.Value) which are the arguments about to be passed.
	//
	// Returns error when the call must be aborted.
	CheckFunctionCall(ctx context.Context, modulePath, fnPath string, args []reflect.Value) error

	// CheckFileOpen is consulted before os.Open, os.OpenFile, os.ReadFile, and other
	// open-style operations.
	//
	// flag and mode mirror the os.OpenFile parameters; for read-only opens, flag is
	// os.O_RDONLY and mode may be zero.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes path (string) which is the file path the caller wants to open.
	// Takes flag (int) which is the os.OpenFile flag set.
	// Takes mode (os.FileMode) which is the permission bits used when the file is created.
	//
	// Returns error when the open must be denied.
	CheckFileOpen(ctx context.Context, modulePath, path string, flag int, mode os.FileMode) error

	// CheckFileWrite is consulted before write-side file operations that do not naturally
	// flow through CheckFileOpen, such as os.WriteFile, os.Mkdir, os.Remove, os.Rename.
	//
	// Hosts that want a unified policy may inspect modulePath and path identically across
	// both methods.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes path (string) which is the file path the caller wants to write.
	//
	// Returns error when the write must be denied.
	CheckFileWrite(ctx context.Context, modulePath, path string) error

	// CheckExec is consulted before os/exec process creation such as exec.Command or
	// exec.LookPath through Cmd.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes name (string) which is the command being invoked, resolved or not, as supplied
	// by the caller.
	// Takes argv ([]string) which is the full argument list, including the command name in
	// argv[0] when present.
	//
	// Returns error when exec must be denied.
	CheckExec(ctx context.Context, modulePath, name string, argv []string) error

	// CheckNetDial is consulted before outbound network connections such as net.Dial,
	// net.DialTimeout, or http.Client.Do.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes network (string) which is the Go network argument, such as "tcp", "udp", or
	// "unix".
	// Takes address (string) which is the destination as supplied by the caller.
	//
	// Returns error when the dial must be denied.
	CheckNetDial(ctx context.Context, modulePath, network, address string) error

	// CheckNetListen is consulted before inbound network listeners such as net.Listen or
	// http.Server.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes network (string) which is the net.Listen network argument.
	// Takes address (string) which is the net.Listen address argument.
	//
	// Returns error when the listen must be denied.
	CheckNetListen(ctx context.Context, modulePath, network, address string) error

	// CheckGetenv is consulted before os.Getenv, os.LookupEnv, and os.Environ entries.
	//
	// For Environ, the hook is consulted per variable; entries the hook denies are omitted
	// from the returned slice.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes name (string) which is the environment variable being read.
	//
	// Returns error when the read must be denied.
	CheckGetenv(ctx context.Context, modulePath, name string) error

	// CheckSetenv is consulted before os.Setenv, os.Unsetenv, and os.Clearenv.
	//
	// For Clearenv the hook is consulted with name == "" and the implementation may either
	// deny outright or audit.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes name (string) which is the environment variable being changed, empty for
	// Clearenv.
	// Takes value (string) which is the new value, empty when the variable is being unset.
	//
	// Returns error when the mutation must be denied.
	CheckSetenv(ctx context.Context, modulePath, name, value string) error

	// CheckSubprocess is consulted before any non-exec spawn path.
	//
	// Covers paths outside the os/exec.Command flow (e.g. syscall.ForkExec, runtime hooks).
	// For os/exec.Command-based flows, CheckExec is used instead.
	//
	// Takes modulePath (string) which names the loaded module making the call, empty for the
	// main policy or script body.
	// Takes name (string) which is the program being spawned.
	// Takes argv ([]string) which is the full argument list for the new process.
	//
	// Returns error when the spawn must be denied.
	CheckSubprocess(ctx context.Context, modulePath, name string, argv []string) error
}

// permissiveCapabilityHook is the default hook installed when no host hook is configured.
//
// Allows every operation, keeping the opt-out path ungated. Tests for the hook plumbing
// rely on this default being a no-op.
type permissiveCapabilityHook struct{}

// CheckFunctionCall always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckFunctionCall(_ context.Context, _, _ string, _ []reflect.Value) error {
	return nil
}

// CheckFileOpen always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckFileOpen(_ context.Context, _, _ string, _ int, _ os.FileMode) error {
	return nil
}

// CheckFileWrite always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckFileWrite(_ context.Context, _, _ string) error {
	return nil
}

// CheckExec always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckExec(_ context.Context, _, _ string, _ []string) error {
	return nil
}

// CheckNetDial always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckNetDial(_ context.Context, _, _, _ string) error {
	return nil
}

// CheckNetListen always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckNetListen(_ context.Context, _, _, _ string) error {
	return nil
}

// CheckGetenv always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckGetenv(_ context.Context, _, _ string) error {
	return nil
}

// CheckSetenv always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckSetenv(_ context.Context, _, _, _ string) error {
	return nil
}

// CheckSubprocess always returns nil.
//
// Returns error which is always nil.
func (permissiveCapabilityHook) CheckSubprocess(_ context.Context, _, _ string, _ []string) error {
	return nil
}

var (
	// permissiveHook is the singleton used in place of a nil hook so that call sites do not
	// need to nil-check on every dispatch.
	permissiveHook CapabilityHook = permissiveCapabilityHook{}
)

// effectiveCapabilityHook returns hook or the permissive default.
//
// Call-site code in dispatch hot paths should hold the hook in a local variable for the
// duration of a call to avoid re-checking nil; this helper exists for setup paths where
// readability dominates.
//
// Takes hook (CapabilityHook) which may be nil.
//
// Returns CapabilityHook which is hook when non-nil, otherwise the permissive default.
func effectiveCapabilityHook(hook CapabilityHook) CapabilityHook {
	if hook == nil {
		return permissiveHook
	}
	return hook
}
