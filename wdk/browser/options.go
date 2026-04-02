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

package browser

import (
	"flag"
	"time"

	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultBuildTimeout is the time limit for building the test binary.
	defaultBuildTimeout = 5 * time.Minute

	// defaultWaitTimeout is the default timeout used by wait methods.
	defaultWaitTimeout = 5 * time.Second

	// defaultSpecTimeout is the default time limit for running a spec.
	defaultSpecTimeout = 5 * time.Minute
)

var (
	flagHeaded = flag.Bool("headed", false, "Run browser in headed mode (show browser window)")

	flagInteractive = flag.Bool("interactive", false, "Enable interactive step-through mode with TUI")

	flagInteractiveSimple = flag.Bool("interactive-simple", false, "Use basic ANSI mode instead of TUI")

	flagUpdateGolden = flag.Bool("update-goldens", false, "Update golden files instead of comparing")
)

// harnessOptions holds the settings for a Harness.
type harnessOptions struct {
	// env holds extra environment variables for the server process.
	env map[string]string

	// sandboxFactory creates sandboxes for filesystem access. When nil,
	// safedisk.NewNoOpSandbox is used as a fallback.
	sandboxFactory safedisk.Factory

	// projectDir is the path to the project folder used for building and testing.
	projectDir string

	// outputDir is the folder for saving screenshots, PDFs, and other test
	// output. Defaults to the current working folder.
	outputDir string

	// serverCommand specifies a custom command to start the server; overrides
	// the default.
	serverCommand []string

	// serverArgs holds extra command-line arguments to pass to the server.
	serverArgs []string

	// buildTimeout is the maximum time allowed to build the test binary.
	buildTimeout time.Duration

	// port specifies the TCP port number; 0 uses a random available port.
	port int

	// headless indicates whether the browser runs without a visible window.
	headless bool

	// interactive enables interactive mode for debugging tests.
	interactive bool

	// interactiveTUI enables full TUI mode rather than simple ANSI output.
	interactiveTUI bool

	// skipBuild skips building the binary when set to true.
	skipBuild bool
}

// HarnessOption configures a Harness for end-to-end browser testing.
type HarnessOption func(*harnessOptions)

// pageOptions holds settings for a Page.
type pageOptions struct {
	// Future options can be added here
}

// PageOption configures pagination behaviour for a Page.
type PageOption func(*pageOptions)

// waitConfig holds settings for wait methods.
type waitConfig struct {
	// timeout is the maximum time to wait; 0 uses the default.
	timeout time.Duration
}

// WaitOption configures the behaviour of wait methods.
type WaitOption func(*waitConfig)

// specOptions holds the settings for RunSpec.
type specOptions struct {
	// updateGoldens indicates whether to update golden test files rather than
	// compare against them.
	updateGoldens bool

	// timeout is the maximum duration allowed for spec execution; 0 means no limit.
	timeout time.Duration
}

// SpecOption configures how a test spec runs.
type SpecOption func(*specOptions)

// WithProjectDir sets the project directory to test. Defaults to ".".
//
// Takes directory (string) which specifies the path to the project directory.
//
// Returns HarnessOption which configures the harness with the given directory.
func WithProjectDir(directory string) HarnessOption {
	return func(o *harnessOptions) {
		o.projectDir = directory
	}
}

// WithOutputDir sets the directory for saving test artefacts such as
// screenshots and PDFs.
//
// Paths passed to Save* methods are relative to this directory. Defaults to
// the current working directory.
//
// Takes directory (string) which specifies the output directory path.
//
// Returns HarnessOption which configures the output directory.
func WithOutputDir(directory string) HarnessOption {
	return func(o *harnessOptions) {
		o.outputDir = directory
	}
}

// WithPort sets the server port.
// A value of 0 means an available port is chosen automatically (default).
//
// Takes port (int) which specifies the port number to use.
//
// Returns HarnessOption which configures the harness with the given port.
func WithPort(port int) HarnessOption {
	return func(o *harnessOptions) {
		o.port = port
	}
}

// WithHeadless sets whether to run the browser in headless mode.
// Defaults to true unless -headed or -interactive flags are set.
//
// Takes headless (bool) which specifies whether to run without a visible
// browser window.
//
// Returns HarnessOption which configures the headless mode setting.
func WithHeadless(headless bool) HarnessOption {
	return func(o *harnessOptions) {
		o.headless = headless
	}
}

// WithInteractive enables step-through mode with a terminal interface.
// Defaults to false unless the -interactive flag is set.
//
// Takes interactive (bool) which enables or disables interactive mode.
//
// Returns HarnessOption which sets up the harness for interactive use.
func WithInteractive(interactive bool) HarnessOption {
	return func(o *harnessOptions) {
		o.interactive = interactive
		if interactive {
			o.headless = false
			o.interactiveTUI = true
		}
	}
}

// WithSimpleInteractive enables basic ANSI interactive mode instead of the
// bubbletea TUI. Use it for terminals that do not support the TUI.
//
// Returns HarnessOption which sets the harness to use simple interactive mode.
func WithSimpleInteractive() HarnessOption {
	return func(o *harnessOptions) {
		o.interactive = true
		o.interactiveTUI = false
		o.headless = false
	}
}

// WithBuildTimeout sets the timeout for building the project. The default
// value is 5 minutes.
//
// Takes d (time.Duration) which specifies the build timeout.
//
// Returns HarnessOption which configures the build timeout.
func WithBuildTimeout(d time.Duration) HarnessOption {
	return func(o *harnessOptions) {
		o.buildTimeout = d
	}
}

// WithSkipBuild skips the build step if the project is already built,
// speeding up repeated test runs.
//
// Takes skip (bool) which controls whether to skip the build step.
//
// Returns HarnessOption which sets up the harness to skip building.
func WithSkipBuild(skip bool) HarnessOption {
	return func(o *harnessOptions) {
		o.skipBuild = skip
	}
}

// WithEnv sets an environment variable for the server process.
//
// Use this to set test-specific settings like database URLs.
//
// Takes key (string) which specifies the environment variable name.
// Takes value (string) which specifies the environment variable value.
//
// Returns HarnessOption which sets the environment variable on the harness.
func WithEnv(key, value string) HarnessOption {
	return func(o *harnessOptions) {
		o.env[key] = value
	}
}

// WithServerCommand sets a custom command to start the server.
// This replaces the default "go run ./cmd/server" command.
//
// Takes arguments (...string) which specifies the command and its arguments.
//
// Returns HarnessOption which configures the harness to use the custom command.
//
// Example:
// browser.NewHarness(
//
//	browser.WithServerCommand("go", "run", "./cmd/testserver"),
//
// )
func WithServerCommand(arguments ...string) HarnessOption {
	return func(o *harnessOptions) {
		o.serverCommand = arguments
	}
}

// WithServerArgs adds extra arguments to the server command. These are added
// after the default or custom server command.
//
// Example:
// browser.NewHarness(
//
//	browser.WithServerArgs("--config", "test.yaml"),
//
// )
//
// Takes arguments (...string) which are the arguments to add.
//
// Returns HarnessOption which configures the harness with the arguments.
func WithServerArgs(arguments ...string) HarnessOption {
	return func(o *harnessOptions) {
		o.serverArgs = append(o.serverArgs, arguments...)
	}
}

// WithSandboxFactory sets a factory for creating sandboxes in the harness.
// When set, the factory is used instead of safedisk.NewNoOpSandbox for
// creating source and output sandboxes.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns HarnessOption which configures the harness with the given factory.
func WithSandboxFactory(factory safedisk.Factory) HarnessOption {
	return func(o *harnessOptions) {
		o.sandboxFactory = factory
	}
}

// WithTimeout sets the timeout for wait methods. Defaults to 5 seconds.
//
// Takes d (time.Duration) which specifies the maximum time to wait.
//
// Returns WaitOption which configures the timeout on a waitConfig.
func WithTimeout(d time.Duration) WaitOption {
	return func(c *waitConfig) {
		c.timeout = d
	}
}

// WithUpdateGoldens sets whether to update golden files.
// Defaults to the value of the -update-goldens flag.
//
// Takes update (bool) which controls whether golden files are updated.
//
// Returns SpecOption which configures the golden file update behaviour.
func WithUpdateGoldens(update bool) SpecOption {
	return func(o *specOptions) {
		o.updateGoldens = update
	}
}

// WithSpecTimeout sets the timeout for spec execution. The default is
// 5 minutes.
//
// Takes d (time.Duration) which specifies the timeout duration.
//
// Returns SpecOption which configures the spec timeout.
func WithSpecTimeout(d time.Duration) SpecOption {
	return func(o *specOptions) {
		o.timeout = d
	}
}

// defaultHarnessOptions returns the default options for the test harness.
//
// Returns harnessOptions which contains sensible defaults for test harness
// configuration.
func defaultHarnessOptions() harnessOptions {
	return harnessOptions{
		env:            make(map[string]string),
		projectDir:     ".",
		outputDir:      ".",
		serverCommand:  nil,
		serverArgs:     nil,
		buildTimeout:   defaultBuildTimeout,
		port:           0,
		headless:       !*flagHeaded && !*flagInteractive,
		interactive:    *flagInteractive,
		interactiveTUI: *flagInteractive && !*flagInteractiveSimple,
		skipBuild:      false,
	}
}

// defaultWaitConfig returns the default wait settings.
//
// Returns waitConfig which contains a five-second timeout.
func defaultWaitConfig() waitConfig {
	return waitConfig{
		timeout: defaultWaitTimeout,
	}
}

// defaultSpecOptions returns the default settings for running tests.
//
// Returns specOptions which holds the default test settings.
func defaultSpecOptions() specOptions {
	return specOptions{
		updateGoldens: *flagUpdateGolden,
		timeout:       defaultSpecTimeout,
	}
}
