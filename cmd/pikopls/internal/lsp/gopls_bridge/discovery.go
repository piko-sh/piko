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

package gopls_bridge

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// goplsBinaryName is the executable name of the Go language server.
	goplsBinaryName = "gopls"

	// goBinaryName is the executable name of the Go toolchain driver.
	goBinaryName = "go"

	// goEnvGOBIN and goEnvGOPATH are the only `go env` keys discovery consults. Restricting
	// to this fixed set keeps every subprocess argument a literal.
	goEnvGOBIN = "GOBIN"

	// goEnvGOPATH names the `go env` GOPATH key whose bin subdirectory holds
	// editor-installed tools.
	goEnvGOPATH = "GOPATH"

	// goEnvTimeout bounds how long discovery waits on a `go env` query so a wedged Go
	// toolchain cannot stall server start-up.
	goEnvTimeout = 5 * time.Second

	// executablePermBits masks the POSIX owner/group/other execute permission bits.
	executablePermBits = 0o111
)

var (
	// errGoEnvTimeout is the cause attached to the bounded `go env` context so a wedged Go
	// toolchain is distinguishable from a deliberate cancellation when the cause is
	// inspected.
	errGoEnvTimeout = errors.New("go env query timed out")
)

// DiscoverGoplsPath resolves the gopls executable. An explicit path (from --gopls-path or
// PIKO_GOPLS_PATH) is honoured first; otherwise the binary is looked up on PATH and then
// in the conventional Go install locations that an editor-launched process may not have
// on PATH (GOBIN, GOPATH/bin, ~/go/bin, ~/.local/bin).
//
// Takes explicit (string) which is the operator-supplied gopls path or name, or empty to
// use the default binary name.
//
// Returns the resolved absolute path and whether a usable gopls was found.
func DiscoverGoplsPath(explicit string) (string, bool) {
	name := strings.TrimSpace(explicit)
	if name == "" {
		name = goplsBinaryName
	}

	if strings.ContainsRune(name, filepath.Separator) || strings.ContainsRune(name, '/') {
		return executableAt(name)
	}

	if resolved, lookErr := exec.LookPath(name); lookErr == nil {
		return resolved, true
	}

	for _, directory := range goplsSearchDirectories() {
		if resolved, found := executableAt(filepath.Join(directory, name)); found {
			return resolved, true
		}
	}

	return "", false
}

// executableAt reports whether candidate names an existing, executable regular file,
// returning its absolute path.
//
// Takes candidate (string) which is the file system path to probe.
//
// Returns the absolute path and whether the candidate is a usable executable.
func executableAt(candidate string) (string, bool) {
	info, statErr := os.Stat(candidate)
	if statErr != nil || info.IsDir() {
		return "", false
	}

	if runtime.GOOS != "windows" && info.Mode().Perm()&executablePermBits == 0 {
		return "", false
	}
	absolute, absErr := filepath.Abs(candidate)
	if absErr != nil {
		return candidate, true
	}
	return absolute, true
}

// goSubprocessEnv returns the go/gopls subprocess environment with GOTOOLCHAIN=auto.
//
// The setting matches how piko's own package loading runs (the inspector's builder_loader
// also forces GOTOOLCHAIN=auto). A project may legitimately require a newer Go toolchain
// than the one installed, declared by the "go 1.X.Y" line in its go.mod or go.work; auto
// lets go resolve and run that toolchain so gopls can load the same packages piko's core
// already loads. Pinning "local" instead breaks every such project (gopls' packages.Load
// fails with "go.work requires go >= 1.X.Y ... GOTOOLCHAIN=local" and the bridge goes
// dark) while adding no protection the core does not already accept: the core runs go
// list with GOTOOLCHAIN=auto on the same untrusted workspace, and auto only ever fetches
// official, checksum-verified Go releases (GOSUMDB), the same trust model as `go build`.
//
// Returns the environment slice for the subprocess with GOTOOLCHAIN=auto appended.
func goSubprocessEnv() []string {
	return append(os.Environ(), "GOTOOLCHAIN=auto")
}

// goplsSearchDirectories returns the fallback directories to probe for gopls, in priority
// order, when it is not on PATH.
//
// Returns the candidate directories in priority order.
func goplsSearchDirectories() []string {
	var directories []string

	if goBin := goEnv(goEnvGOBIN); goBin != "" {
		directories = append(directories, goBin)
	}
	if goPath := goEnv(goEnvGOPATH); goPath != "" {
		for _, entry := range filepath.SplitList(goPath) {
			if entry != "" {
				directories = append(directories, filepath.Join(entry, "bin"))
			}
		}
	}
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		directories = append(directories, filepath.Join(home, "go", "bin"), filepath.Join(home, ".local", "bin"))
	}

	return directories
}

// goEnv reads a single value from `go env`.
//
// An empty string is returned when the Go toolchain is unavailable or the key is
// unsupported. The LookPath probe avoids spawning a process when no Go toolchain is
// installed. Only the fixed set of keys discovery needs is accepted, so every subprocess
// argument is a literal constant rather than caller-influenced input.
//
// Takes key (string) which is the `go env` variable name to read.
//
// Returns the trimmed value, or empty when the toolchain or key is unavailable.
func goEnv(key string) string {
	if _, lookErr := exec.LookPath(goBinaryName); lookErr != nil {
		return ""
	}
	ctx, cancel := context.WithTimeoutCause(context.Background(), goEnvTimeout, errGoEnvTimeout)
	defer cancel()

	var command *exec.Cmd
	switch key {
	case goEnvGOBIN:
		command = exec.CommandContext(ctx, goBinaryName, "env", goEnvGOBIN)
	case goEnvGOPATH:
		command = exec.CommandContext(ctx, goBinaryName, "env", goEnvGOPATH)
	default:
		return ""
	}
	command.Env = goSubprocessEnv()

	output, runErr := command.Output()
	if runErr != nil {
		_, l := logger_domain.From(ctx, log)
		cause := runErr
		if ctxErr := context.Cause(ctx); ctxErr != nil {
			cause = ctxErr
		}
		l.Trace("Discovering gopls: 'go env' query failed; omitting this entry from the search path",
			logger_domain.String("key", key),
			logger_domain.Error(cause))
		return ""
	}
	return strings.TrimSpace(string(output))
}
