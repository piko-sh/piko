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

package campaign

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
	"piko.sh/piko/tests/fuzz/internal/resource"
	"piko.sh/piko/tests/fuzz/internal/runner"
	"piko.sh/piko/tests/fuzz/internal/targets"
	"piko.sh/safeconv"
)

const (
	// mebibyte is the number of bytes in one mebibyte.
	mebibyte = 1 << 20

	// microsecondsPerSecond is the number of microseconds in one second.
	microsecondsPerSecond = 1e6

	// checkStatusPass marks a passed check or case.
	checkStatusPass = "pass"

	// modeInterp names the interpreted server mode.
	modeInterp = "interp"

	// modeCompiled names the compiled server mode.
	modeCompiled = "compiled"
)

// CheckMetric describes one selected mode or package check within a case.
type CheckMetric struct {
	// BrowserRoutes holds route navigation evidence.
	BrowserRoutes map[string]runner.BrowserRouteEvidence `json:",omitempty"`

	// GoHost holds measured Go-host operation evidence.
	GoHost map[string]runner.HostEvidence `json:",omitempty"`

	// PublicContract holds the public-consumer check receipt.
	PublicContract *publiccontracts.Receipt `json:",omitempty"`

	// BrowserEngine identifies the browser engine used for this check.
	BrowserEngine string `json:",omitempty"`

	// BrowserVersion identifies the browser version used for this check.
	BrowserVersion string `json:",omitempty"`

	// PrimaryBrowserVersion identifies the primary browser version.
	PrimaryBrowserVersion string `json:",omitempty"`

	// Status is pending, running, pass, failed or incomplete.
	Status string

	// Verification records the first independent browser verification failure.
	Verification string `json:",omitempty"`

	// Editions holds source-edition evidence.
	Editions []runner.EditionReceipt `json:",omitempty"`

	// GeneratedArtifactChecks lists verified generated artefacts.
	GeneratedArtifactChecks []string `json:",omitempty"`

	// Mutations holds checker-sensitivity results.
	Mutations []runner.MutationCheck `json:",omitempty"`

	// Memory holds sampled phase memory.
	Memory resource.PhaseMemory

	// Cost holds measured CPU and working-directory costs.
	Cost resource.PhaseCost

	// Admission records the resource-admission attempt.
	Admission AdmissionMetric

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// WorkflowObserved counts observed workflow steps.
	WorkflowObserved int `json:",omitempty"`

	// WorkflowVerified counts independently verified workflow steps.
	WorkflowVerified int `json:",omitempty"`

	// BrowserSelected reports whether this check selected a browser.
	BrowserSelected bool

	// BrowserObserved reports whether the worker supplied browser evidence.
	BrowserObserved bool

	// FrontendAccepted reports whether the frontend accepted the source.
	FrontendAccepted bool

	// FrontendRejected reports whether the frontend rejected the source.
	FrontendRejected bool

	// ProgramVerified reports whether the program contract passed.
	ProgramVerified bool

	// FirstInInvocation marks a check that includes initial worker setup.
	FirstInInvocation bool
}

// AdmissionMetric covers the time and estimated incremental memory required before this
// check could start.
type AdmissionMetric struct {
	// EstimatedBytes is the estimated incremental memory requirement.
	EstimatedBytes uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// Granted reports whether resource admission succeeded.
	Granted bool
}

// MemoryMetrics accumulates absolute cgroup readings and the largest rise above a check's
// starting memory.current.
type MemoryMetrics struct {
	// MeasuredChecks counts checks with at least one memory sample.
	MeasuredChecks uint64

	// Samples counts memory samples.
	Samples uint64

	// ReadErrors counts failed memory reads.
	ReadErrors uint64

	// MaxSampledBytes is the largest sampled memory reading in bytes.
	MaxSampledBytes uint64

	// MaxRiseFromStartBytes is the largest sampled rise above a check's starting memory, in
	// bytes.
	MaxRiseFromStartBytes uint64
}

// BrowserRouteCounts counts browser routes at each evidence stage.
type BrowserRouteCounts struct {
	// Eligible counts routes eligible for browser observation.
	Eligible uint64

	// Selected counts routes selected for browser observation.
	Selected uint64

	// Visited counts routes the browser visited.
	Visited uint64

	// Verified counts routes whose browser evidence was verified.
	Verified uint64
}

// MutationCounts counts a checker's controlled observations.
type MutationCounts struct {
	// Control marks an unmodified checker control.
	Control bool

	// Attempts counts controlled observations.
	Attempts uint64

	// ExpectedResults counts controlled observations with the expected checker result.
	ExpectedResults uint64
}

// SensitivityMetrics aggregates checker sensitivity and its resource cost.
type SensitivityMetrics struct {
	// Checks counts mutation attempts and expected results by checker.
	Checks map[string]MutationCounts

	// Cases counts cases with sensitivity checks.
	Cases uint64

	// MeasuredCPUChecks counts sensitivity checks with measured CPU use.
	MeasuredCPUChecks uint64

	// CPUUsec is the measured sensitivity CPU use in microseconds.
	CPUUsec uint64

	// MaxSampledMemoryBytes is the largest sampled sensitivity memory reading in bytes.
	MaxSampledMemoryBytes uint64
}

// ModeMetrics counts planned checks and their outcomes in one mode.
type ModeMetrics struct {
	// BrowserRoutes holds route navigation evidence.
	BrowserRoutes map[string]BrowserRouteCounts `json:",omitempty"`

	// PublicContracts aggregates public-consumer check receipts.
	PublicContracts map[string]PublicContractCounts `json:",omitempty"`

	// Matrices holds evidence for finite generation matrices.
	Matrices map[string]*MatrixEvidence `json:",omitempty"`

	// Editions holds source-edition evidence.
	Editions map[string]TransitionCounts `json:",omitempty"`

	// GeneratedArtifactChecks counts successful generated-artefact checks by artefact ID.
	GeneratedArtifactChecks map[string]uint64 `json:",omitempty"`

	// BrowserEngines counts checks by browser engine and version.
	BrowserEngines map[string]uint64 `json:",omitempty"`

	// GoHost holds measured Go-host operation evidence.
	GoHost map[string]runner.HostEvidence `json:",omitempty"`

	// Sensitivity holds checker-sensitivity totals.
	Sensitivity *SensitivityMetrics `json:",omitempty"`

	// Workflows holds evidence by workflow name.
	Workflows map[string]TransitionCounts `json:",omitempty"`

	// SubsequentMemory holds the memory of the remaining checks.
	SubsequentMemory MemoryMetrics

	// FirstInInvocationMemory holds the memory of checks that include worker setup.
	FirstInInvocationMemory MemoryMetrics

	// ProgramRequests counts planned program requests.
	ProgramRequests uint64

	// ObservedProgramOutputs counts program output rows with an observed receipt.
	ObservedProgramOutputs uint64

	// LargestEndWorkDirBytes is the largest final work-directory size in bytes.
	LargestEndWorkDirBytes uint64

	// ThrottledUsec is the measured CPU throttling in microseconds.
	ThrottledUsec uint64

	// CPUUsec is the measured CPU use in microseconds.
	CPUUsec uint64

	// MeasuredCPUChecks counts checks with measured CPU use.
	MeasuredCPUChecks uint64

	// LegacyVerifiedProgramOutputs preserves output totals from older report formats.
	LegacyVerifiedProgramOutputs uint64 `json:",omitempty"`

	// BrowserSelected counts checks that selected a browser.
	BrowserSelected uint64

	// BrowserObserved counts checks with browser evidence.
	BrowserObserved uint64

	// LegacyVerifiedProgramRequests preserves request totals from older report formats.
	LegacyVerifiedProgramRequests uint64 `json:",omitempty"`

	// ReachedProgramRequests counts program requests the worker reached.
	ReachedProgramRequests uint64

	// ObservedProgramRequests counts program requests with an observed receipt.
	ObservedProgramRequests uint64

	// VerifiedProgramRequests counts independently verified program requests.
	VerifiedProgramRequests uint64

	// ProgramOutputs counts planned program output rows.
	ProgramOutputs uint64

	// ReachedProgramOutputs counts program output rows the worker reached.
	ReachedProgramOutputs uint64

	// MeasuredWorkDirs counts checks with a measured work directory.
	MeasuredWorkDirs uint64

	// VerifiedProgramOutputs counts independently verified program output rows.
	VerifiedProgramOutputs uint64

	// AdmissionGranted counts granted resource admissions.
	AdmissionGranted uint64

	// AdmissionDenied counts denied resource admissions.
	AdmissionDenied uint64

	// LargestEstimateBytes is the largest admission memory estimate in bytes.
	LargestEstimateBytes uint64

	// AdmissionElapsed is the total resource-admission wait.
	AdmissionElapsed time.Duration

	// Selected counts selected checks.
	Selected uint64

	// Attempted counts started checks.
	Attempted uint64

	// Completed counts checks that ran to an outcome.
	Completed uint64

	// Failed counts failed checks.
	Failed uint64

	// Incomplete counts interrupted checks.
	Incomplete uint64

	// Skipped counts pending or skipped checks.
	Skipped uint64

	// Unknown counts checks with an unrecognised status.
	Unknown uint64

	// FrontendAccepted counts checks whose source the frontend accepted.
	FrontendAccepted uint64

	// FrontendRejected counts checks whose source the frontend rejected.
	FrontendRejected uint64

	// FrontendUnknown counts checks without a frontend outcome.
	FrontendUnknown uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration
}

// TransitionCounts records coverage of one named, modelled interaction in one host mode.
type TransitionCounts struct {
	// Planned counts planned interactions.
	Planned uint64

	// Executed counts executed interactions.
	Executed uint64

	// Reached counts interactions the worker reached.
	Reached uint64

	// Observed counts interactions with an observed receipt.
	Observed uint64

	// Verified counts independently verified interactions.
	Verified uint64

	// LegacyExecuted preserves execution totals from older reports.
	LegacyExecuted uint64 `json:",omitempty"`

	// LegacyVerified preserves verification totals from older reports.
	LegacyVerified uint64 `json:",omitempty"`
}

// MatrixEvidence records observations for distinct cells in a finite matrix.
type MatrixEvidence struct {
	// Cells holds evidence by distinct cell identity.
	Cells map[string]CellEvidence

	// Feasible is the finite matrix's number of possible cells.
	Feasible int
}

// TargetMetrics measures the distribution and cost of completed case attempts.
type TargetMetrics struct {
	// Cases counts case attempts.
	Cases uint64

	// Passed counts passed case attempts.
	Passed uint64

	// Failed counts failed case attempts.
	Failed uint64

	// Incomplete counts interrupted or unclassified case attempts.
	Incomplete uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration
}

// CampaignCPU counts measured CPU time across campaign invocations.
type CampaignCPU struct {
	// Usec is the measured CPU time in microseconds.
	Usec uint64

	// MeasuredInvocations counts invocations with CPU accounting.
	MeasuredInvocations uint64

	// UnmeasuredInvocations counts invocations without CPU accounting.
	UnmeasuredInvocations uint64
}

// RunMetrics aggregates committed case attempts across invocations.
type RunMetrics struct {
	// Checks aggregates check outcomes and incomplete evidence by mode.
	Checks map[string]ModeMetrics

	// Targets holds per-target outcomes.
	Targets map[string]TargetMetrics

	// FeaturePairs counts observed source-feature pairs.
	FeaturePairs map[string]uint64

	// ConfirmedFingerprints counts cold-confirmed failure fingerprints.
	ConfirmedFingerprints map[string]uint64

	// InvocationCPU aggregates whole-invocation CPU accounting.
	InvocationCPU CampaignCPU

	// Cases counts recorded case attempts.
	Cases uint64

	// Elapsed is the measured elapsed time.
	Elapsed time.Duration

	// BrowserSelected counts browser selections.
	BrowserSelected uint64

	// BrowserObserved counts observed browser results.
	BrowserObserved uint64

	// CasesWithoutModeData counts legacy cases without per-mode evidence.
	CasesWithoutModeData uint64
}

// recordObservation records only interaction evidence supplied by the worker.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*runner.Observation) which holds the worker observation.
// Takes err (error) which is the operation failure to inspect.
//
// Returns error which preserves the worker error unless the independent observation check
// fails.
func (check *CheckMetric) recordObservation(c *casefile.Case, o *runner.Observation, err error) error {
	check.recordReceipt(c, o)
	for _, record := range []func(*CheckMetric, *casefile.Case, *runner.Observation) bool{
		(*CheckMetric).recordFormWorkflow,
		(*CheckMetric).recordLifecycle,
		(*CheckMetric).recordPartialComponents,
		(*CheckMetric).recordEvents,
		(*CheckMetric).recordCheckout,
		(*CheckMetric).recordDOM,
	} {
		if record(check, c, o) {
			return err
		}
	}
	if err == nil && check.BrowserSelected && !check.BrowserObserved {
		return errIncompleteBrowser
	}
	return err
}

// recordReceipt records the worker's edition, host, route, artefact and frontend
// evidence.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes o (*runner.Observation) which holds the worker observation.
func (check *CheckMetric) recordReceipt(c *casefile.Case, o *runner.Observation) {
	check.Editions = o.Editions
	if o.Client != nil {
		check.PrimaryBrowserVersion = o.Client.PrimaryBrowserVersion
	}
	check.GoHost = runner.GoHostEvidence(c, o)
	routes, routeErr := runner.BrowserRouteResults(c, o, check.BrowserSelected)
	check.BrowserRoutes = routes
	check.recordVerification(routeErr)
	if o.GeneratedArtifacts != nil {
		check.GeneratedArtifactChecks = o.GeneratedArtifacts.Checks
	}
	check.FrontendAccepted = o.ComponentCompiled || o.Status > 0
	check.BrowserObserved = len(o.Counts) == 2
}

// recordFormWorkflow records form-workflow evidence, or client step evidence without one.
//
// Takes c (*casefile.Case) which holds the expected form workflow or client steps.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether an incomplete form trace ends the recording.
func (check *CheckMetric) recordFormWorkflow(c *casefile.Case, o *runner.Observation) bool {
	workflow := c.Expected.FormWorkflow
	if workflow == nil {
		if expected := c.Expected.Client; expected != nil && check.BrowserObserved {
			check.BrowserObserved = clientStepsObserved(o.Client, expected)
		}
		return false
	}
	actual := o.FormWorkflow
	check.BrowserObserved = actual != nil && actual.DriverError == "" && len(actual.Steps) == len(workflow.Steps)
	if actual == nil {
		return false
	}
	check.WorkflowObserved = len(actual.Steps)
	check.WorkflowVerified = runner.VerifiedFormSteps(workflow, actual)
	return actual.DriverError != "" || len(actual.Steps) != len(workflow.Steps)
}

// recordLifecycle records navigation and cleanup evidence.
//
// Takes c (*casefile.Case) which holds the expected lifecycle journey.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether an incomplete trace ends the recording.
func (check *CheckMetric) recordLifecycle(c *casefile.Case, o *runner.Observation) bool {
	expected := c.Expected.Client
	if expected == nil || expected.Lifecycle == nil || !check.BrowserSelected {
		return false
	}
	check.BrowserObserved = check.BrowserObserved && o.Client != nil && o.Client.Lifecycle != nil
	if o.Client == nil || o.Client.Lifecycle == nil {
		return false
	}
	trace := o.Client.Lifecycle
	check.WorkflowObserved = len(trace.States)
	check.WorkflowVerified = runner.VerifiedLifecycleSteps(trace, expected.Lifecycle)
	if trace.DriverError != "" || trace.FailedStep != "" || len(trace.States) != len(expected.Lifecycle.States) {
		check.BrowserObserved = false
		return true
	}
	return false
}

// recordPartialComponents records partial-component journey evidence.
//
// Takes c (*casefile.Case) which holds the expected partial-component journey.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether an incomplete trace ends the recording.
func (check *CheckMetric) recordPartialComponents(c *casefile.Case, o *runner.Observation) bool {
	expected := c.Expected.Client
	if expected == nil || expected.PartialComponents == nil || !check.BrowserSelected {
		return false
	}
	actual := o.Client
	check.BrowserObserved = check.BrowserObserved && actual != nil && actual.PartialComponents != nil
	if actual == nil || actual.PartialComponents == nil {
		return false
	}
	trace := actual.PartialComponents
	check.WorkflowObserved = len(trace.States)
	check.WorkflowVerified = runner.VerifiedPartialComponentSteps(trace, expected.PartialComponents)
	if trace.DriverError != "" || len(trace.States) != len(expected.PartialComponents.States) {
		check.BrowserObserved = false
		return true
	}
	return false
}

// recordEvents records component event journey evidence.
//
// Takes c (*casefile.Case) which holds the expected event journey.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether an incomplete trace ends the recording.
func (check *CheckMetric) recordEvents(c *casefile.Case, o *runner.Observation) bool {
	expected := c.Expected.Client
	if expected == nil || expected.Events == nil || !check.BrowserSelected {
		return false
	}
	check.BrowserObserved = check.BrowserObserved && o.Client != nil && o.Client.Events != nil
	if o.Client == nil || o.Client.Events == nil {
		return false
	}
	events := o.Client.Events
	check.WorkflowObserved = len(events.States)
	check.WorkflowVerified = runner.VerifiedEventSteps(events, expected.Events)
	check.BrowserObserved = check.BrowserObserved && check.WorkflowObserved == len(expected.Events.States)
	if events.DriverError != "" || len(events.States) != len(expected.Events.States) {
		check.BrowserObserved = false
		return true
	}
	return false
}

// recordCheckout records checkout journey evidence.
//
// Takes c (*casefile.Case) which holds the expected checkout journey.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether an incomplete trace ends the recording.
func (check *CheckMetric) recordCheckout(c *casefile.Case, o *runner.Observation) bool {
	expected := c.Expected.Client
	if expected == nil || expected.Checkout == nil || !check.BrowserSelected {
		return false
	}
	var actual *casefile.CheckoutTrace
	if o.Client != nil {
		actual = o.Client.Checkout
	}
	check.BrowserObserved = check.BrowserObserved && actual != nil
	if actual == nil {
		return false
	}
	check.WorkflowObserved = len(actual.States)
	check.WorkflowVerified = runner.VerifiedCheckoutSteps(actual, expected.Checkout)
	check.BrowserObserved = check.BrowserObserved && actual.DriverError == "" && len(actual.States) == len(expected.Checkout.States)
	return !check.BrowserObserved
}

// recordDOM records DOM plan evidence and its independent verification.
//
// Takes c (*casefile.Case) which holds the expected DOM plan.
// Takes o (*runner.Observation) which holds the worker observation.
//
// Returns bool which reports whether a driver failure ends the recording.
func (check *CheckMetric) recordDOM(c *casefile.Case, o *runner.Observation) bool {
	if c.Expected.DOM == nil || !check.BrowserSelected {
		return false
	}
	if o.Client == nil || o.Client.DOM == nil {
		check.BrowserObserved = false
		return false
	}
	trace := o.Client.DOM
	check.BrowserEngine, check.BrowserVersion = trace.Engine, trace.Version
	check.WorkflowObserved = len(trace.States)
	count, domErr := runner.VerifyDOM(c.Expected.DOM, trace)
	check.WorkflowVerified = count
	check.recordVerification(domErr)
	check.BrowserObserved = check.BrowserObserved && trace.DriverError == "" && len(trace.States) == len(c.Expected.DOM.Steps)
	return trace.DriverError != ""
}

// recordVerification keeps the first independent browser verification failure.
//
// Independent expectation checks report the failure. This keeps the evidence on the check
// that produced it.
//
// Takes err (error) which is the verification failure, or nil.
func (check *CheckMetric) recordVerification(err error) {
	if err != nil && check.Verification == "" {
		check.Verification = err.Error()
	}
}

// observe accumulates phase samples and memory high-water marks.
//
// Takes p (resource.PhaseMemory) which holds the phase memory readings.
func (m *MemoryMetrics) observe(p resource.PhaseMemory) {
	m.ReadErrors += p.ReadErrors
	if p.Samples == 0 {
		return
	}
	m.MeasuredChecks++
	m.Samples += p.Samples
	m.MaxSampledBytes = max(m.MaxSampledBytes, p.SampledPeakBytes)
	if p.StartKnown && p.SampledPeakBytes >= p.StartBytes {
		m.MaxRiseFromStartBytes = max(m.MaxRiseFromStartBytes, p.SampledPeakBytes-p.StartBytes)
	}
}

// observeWorkflows accumulates workflow proof stages from a worker receipt.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
// Takes steps ([]runner.Proof) which holds ordered proof or interaction observations.
func (s *ModeMetrics) observeWorkflows(c *casefile.Case, check *CheckMetric, steps []runner.Proof) {
	for _, sequence := range workflowSequences(c) {
		for index, name := range sequence {
			measured := check.BrowserSelected && index < len(steps)
			stage := runner.Unseen
			if measured {
				stage = steps[index]
			}
			s.recordWorkflow(name, stage, measured)
		}
	}
}

// recordWorkflow counts one planned workflow step and its proof stage.
//
// Takes name (string) which identifies the workflow step.
// Takes stage (runner.Proof) which is the step's proof stage.
// Takes measured (bool) which reports whether the browser measured the step.
func (s *ModeMetrics) recordWorkflow(name string, stage runner.Proof, measured bool) {
	if s.Workflows == nil {
		s.Workflows = map[string]TransitionCounts{}
	}
	transition := s.Workflows[name]
	transition.Planned++
	if measured {
		if stage >= runner.Reached {
			transition.Reached++
		}
		if stage >= runner.Observed {
			transition.Observed++
			transition.Executed++
		}
		if stage == runner.Verified {
			transition.Verified++
		}
	}
	s.Workflows[name] = transition
}

// observe accumulates one mode check's outcome and measured cost.
//
// Takes name (string) which identifies the recorded check or server mode.
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
func (s *ModeMetrics) observe(name string, check *CheckMetric) {
	if check.PublicContract != nil {
		observePublicContract(s, check)
	}
	s.observeEvidence(check)
	if len(check.Mutations) > 0 {
		s.observeSensitivity(check)
	}
	s.observeCost(check)
	s.observeStatus(name, check)
}

// observeEvidence accumulates generated-artefact, browser-engine and Go-host evidence.
//
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
func (s *ModeMetrics) observeEvidence(check *CheckMetric) {
	if len(check.GeneratedArtifactChecks) > 0 {
		if s.GeneratedArtifactChecks == nil {
			s.GeneratedArtifactChecks = map[string]uint64{}
		}
		for _, field := range check.GeneratedArtifactChecks {
			s.GeneratedArtifactChecks[field]++
		}
	}
	if check.BrowserEngine != "" {
		s.countBrowserEngine(strings.TrimSpace(check.BrowserEngine + " " + check.BrowserVersion))
	}
	if check.PrimaryBrowserVersion != "" && check.BrowserEngine != "chromium" {
		s.countBrowserEngine("chromium " + check.PrimaryBrowserVersion)
	}
	for property, evidence := range check.GoHost {
		if s.GoHost == nil {
			s.GoHost = map[string]runner.HostEvidence{}
		}
		count := s.GoHost[property]
		count.Planned += evidence.Planned
		count.Executed += evidence.Executed
		count.Verified += evidence.Verified
		s.GoHost[property] = count
	}
}

// countBrowserEngine counts one check against a browser engine and version.
//
// Takes engine (string) which names the browser engine and version.
func (s *ModeMetrics) countBrowserEngine(engine string) {
	if s.BrowserEngines == nil {
		s.BrowserEngines = map[string]uint64{}
	}
	s.BrowserEngines[engine]++
}

// observeSensitivity accumulates checker-sensitivity results and their cost.
//
// Takes check (*CheckMetric) which holds one mode check's mutation results.
func (s *ModeMetrics) observeSensitivity(check *CheckMetric) {
	if s.Sensitivity == nil {
		s.Sensitivity = &SensitivityMetrics{
			Cases:                 0,
			MeasuredCPUChecks:     0,
			CPUUsec:               0,
			MaxSampledMemoryBytes: 0,
			Checks:                map[string]MutationCounts{},
		}
	}
	sensitivity := s.Sensitivity
	sensitivity.Cases++
	sensitivity.MaxSampledMemoryBytes = max(sensitivity.MaxSampledMemoryBytes, check.Memory.SampledPeakBytes)
	if check.Cost.CPUKnown {
		sensitivity.MeasuredCPUChecks++
		sensitivity.CPUUsec += check.Cost.CPUUsec
	}
	for index := range check.Mutations {
		mutation := &check.Mutations[index]
		counts := sensitivity.Checks[mutation.Name]
		counts.Control = mutation.Control
		counts.Attempts++
		if mutation.Passed {
			counts.ExpectedResults++
		}
		sensitivity.Checks[mutation.Name] = counts
	}
}

// observeCost accumulates measured CPU, work-directory, route, memory and admission cost.
//
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
func (s *ModeMetrics) observeCost(check *CheckMetric) {
	if check.Cost.CPUKnown {
		s.MeasuredCPUChecks++
		s.CPUUsec += check.Cost.CPUUsec
		s.ThrottledUsec += check.Cost.ThrottledUsec
	}
	if check.Cost.WorkDirKnown {
		s.MeasuredWorkDirs++
		s.LargestEndWorkDirBytes = max(s.LargestEndWorkDirBytes, check.Cost.EndWorkDirBytes)
	}
	for path, route := range check.BrowserRoutes {
		s.observeBrowserRoute(path, route)
	}
	if check.FirstInInvocation {
		s.FirstInInvocationMemory.observe(check.Memory)
	} else {
		s.SubsequentMemory.observe(check.Memory)
	}
	if admission := check.Admission; admission.EstimatedBytes != 0 {
		s.LargestEstimateBytes = max(s.LargestEstimateBytes, admission.EstimatedBytes)
		s.AdmissionElapsed += admission.Elapsed
		if admission.Granted {
			s.AdmissionGranted++
		} else {
			s.AdmissionDenied++
		}
	}
}

// observeBrowserRoute counts one browser route's selection and proof stages.
//
// Takes path (string) which is the route path.
// Takes route (runner.BrowserRouteEvidence) which holds the route's browser evidence.
func (s *ModeMetrics) observeBrowserRoute(path string, route runner.BrowserRouteEvidence) {
	if s.BrowserRoutes == nil {
		s.BrowserRoutes = map[string]BrowserRouteCounts{}
	}
	key := path
	if route.Role == "primary" || route.Role == "workflow" {
		key, _, _ = strings.Cut(path, "?")
	}
	counts := s.BrowserRoutes[key]
	counts.Eligible++
	if route.Selected {
		counts.Selected++
	}
	if route.Visited {
		counts.Visited++
	}
	if route.Verified {
		counts.Verified++
	}
	s.BrowserRoutes[key] = counts
}

// observeStatus counts one check's execution stage, outcome and frontend result.
//
// Takes name (string) which identifies the recorded check or server mode.
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
func (s *ModeMetrics) observeStatus(name string, check *CheckMetric) {
	s.Selected++
	if check.BrowserSelected {
		s.BrowserSelected++
	}
	if check.BrowserObserved {
		s.BrowserObserved++
	}
	s.Elapsed += check.Elapsed
	switch check.Status {
	case checkStatusPass:
		s.Attempted++
		s.Completed++
	case StatusFailed:
		s.Attempted++
		s.Completed++
		s.Failed++
	case "running", "incomplete":
		s.Attempted++
		s.Incomplete++
	case "pending", "skipped":
		s.Skipped++
	default:
		s.Unknown++
	}
	if name == modeCompiled || name == modeInterp {
		switch {
		case check.FrontendAccepted:
			s.FrontendAccepted++
		case check.FrontendRejected:
			s.FrontendRejected++
		default:
			s.FrontendUnknown++
		}
	}
}

// observeEditions counts the planned and verified source editions of one check.
//
// Takes c (*casefile.Case) which holds the planned editions.
// Takes check (*CheckMetric) which holds the check's edition receipts.
func (s *ModeMetrics) observeEditions(c *casefile.Case, check *CheckMetric) {
	for _, stage := range c.Expected.Editions {
		if s.Editions == nil {
			s.Editions = map[string]TransitionCounts{}
		}
		row := s.Editions[stage.ID]
		row.Planned++
		for _, actual := range check.Editions {
			if actual.ID != stage.ID {
				continue
			}
			if actual.Reached {
				row.Executed++
			}
			if actual.Verified {
				row.Verified++
			}
			break
		}
		s.Editions[stage.ID] = row
	}
}

// Observe commits one case's measured outcomes to the campaign totals.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes r (*Result) which holds the case execution result.
// Takes confirmed (bool) which reports whether a fresh worker confirmed the failure.
func (m *RunMetrics) Observe(c *casefile.Case, r *Result, confirmed bool) {
	if m.Targets == nil {
		m.Targets = map[string]TargetMetrics{}
	}
	if m.Checks == nil {
		m.Checks = map[string]ModeMetrics{}
	}
	if m.FeaturePairs == nil {
		m.FeaturePairs = map[string]uint64{}
	}
	if m.ConfirmedFingerprints == nil {
		m.ConfirmedFingerprints = map[string]uint64{}
	}
	m.Cases++
	m.Elapsed += r.Elapsed
	m.observeTarget(c.Target, r)
	if confirmed && confirmable(*r) {
		m.ConfirmedFingerprints[r.Fingerprint()]++
	}
	if len(r.Checks) == 0 {
		m.CasesWithoutModeData++
	}
	for name := range r.Checks {
		m.observeCheck(c, r, name)
	}

	if !targets.IsApplication(c.Target) && !slices.Contains([]string{"ast", "formatter", "syntax"}, c.Target) {
		return
	}
	features := slices.Clone(c.Features)
	slices.Sort(features)
	features = slices.Compact(features)
	for index, feature := range features {
		for _, other := range features[index+1:] {
			m.FeaturePairs[feature+" + "+other]++
		}
	}
}

// Summary formats outcome counts with their check and browser denominators.
//
// Returns string which is the formatted check and browser outcome summary.
func (m *RunMetrics) Summary() string {
	var out strings.Builder
	fmt.Fprintf(&out, "%d cases, %d confirmed fingerprints", m.Cases, len(m.ConfirmedFingerprints))
	if m.Elapsed > 0 {
		fmt.Fprintf(&out, ", %.1f cases/hour of case execution", float64(m.Cases)/m.Elapsed.Hours())
	}
	if m.InvocationCPU.MeasuredInvocations > 0 {
		fmt.Fprintf(&out, ", %s whole-campaign cgroup CPU before the current checkpoint", microseconds(m.InvocationCPU.Usec))
	}
	for _, name := range slices.Sorted(maps.Keys(m.Checks)) {
		s := m.Checks[name]
		summariseMode(&out, name, &s)
	}
	fmt.Fprintf(&out, "\nbrowser observations: %d/%d selected", m.BrowserObserved, m.BrowserSelected)
	if m.CasesWithoutModeData > 0 {
		fmt.Fprintf(&out, "\ncases without mode records: %d", m.CasesWithoutModeData)
	}
	return out.String()
}

// observeTarget counts one case attempt against its target.
//
// Takes name (string) which is the case's target.
// Takes r (*Result) which holds the case execution result.
func (m *RunMetrics) observeTarget(name string, r *Result) {
	target := m.Targets[name]
	target.Cases++
	target.Elapsed += r.Elapsed
	switch r.Kind {
	case checkStatusPass:
		target.Passed++
	case "worker", "target", "expectation", "disagreement", "timeout":
		target.Failed++
	default:
		target.Incomplete++
	}
	m.Targets[name] = target
}

// observeCheck commits one mode check's outcome, obligations and editions.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes r (*Result) which holds the case execution result.
// Takes name (string) which identifies the check mode.
func (m *RunMetrics) observeCheck(c *casefile.Case, r *Result, name string) {
	check := r.Checks[name]
	s := m.Checks[name]
	s.observe(name, &check)
	observation := r.Observations[name]
	s.observeCaseObligations(c, name, &check, &observation)
	if name == modeInterp || name == modeCompiled {
		s.observeEditions(c, &check)
	}
	if check.BrowserSelected {
		m.BrowserSelected++
	}
	if check.BrowserObserved {
		m.BrowserObserved++
	}
	m.Checks[name] = s
}

// workflowSequences lists the named steps of every modelled workflow in a case.
//
// Takes c (*casefile.Case) which holds the expected workflows.
//
// Returns [][]string which holds each workflow's step names in execution order.
func workflowSequences(c *casefile.Case) [][]string {
	var sequences [][]string
	if plan := c.Expected.DOM; plan != nil {
		sequences = append(sequences, stepNames(plan.Steps, c.Target+"/browser/", func(step *casefile.DOMStep) string { return step.ID }))
	}
	if workflow := c.Expected.FormWorkflow; workflow != nil {
		sequences = append(sequences, stepNames(workflow.Steps, "form/browser/", func(step *casefile.FormCommand) string { return step.Op }))
	}
	if client := c.Expected.Client; client != nil {
		sequences = append(sequences, clientSequences(client)...)
	}
	return sequences
}

// clientSequences lists the named steps of every modelled browser-client journey.
//
// Takes client (*casefile.ClientTransition) which holds the expected client journeys.
//
// Returns [][]string which holds each journey's step names in execution order.
func clientSequences(client *casefile.ClientTransition) [][]string {
	var sequences [][]string
	if client.Checkout != nil {
		sequences = append(sequences, stepNames(client.Checkout.States, "checkout/browser/", func(state *casefile.CheckoutState) string { return state.Step }))
	}
	if client.PartialComponents != nil {
		sequences = append(sequences, stepNames(client.PartialComponents.States, "partial-components/browser/",
			func(state *casefile.PartialComponentState) string { return state.Step }))
	}
	if client.Events != nil {
		sequences = append(sequences, stepNames(client.Events.States, "events/browser/", func(state *casefile.EventState) string { return state.Step }))
	}
	if client.Lifecycle != nil {
		sequences = append(sequences, stepNames(client.Lifecycle.States, "navigation/browser/", func(state *casefile.LifecycleState) string { return state.Step }))
	}
	return sequences
}

// stepNames prefixes each item's step name.
//
// Takes items ([]T) which holds the ordered workflow items.
// Takes prefix (string) which qualifies each step name.
// Takes name (func(*T) string) which extracts an item's step name.
//
// Returns []string which holds the prefixed step names in item order.
func stepNames[T any](items []T, prefix string, name func(*T) string) []string {
	names := make([]string, len(items))
	for index := range items {
		names[index] = prefix + name(&items[index])
	}
	return names
}

// summariseMode writes one mode's outcome, cost and proof summary.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes name (string) which identifies the check mode.
// Takes s (*ModeMetrics) which holds the mode's accumulated metrics.
func summariseMode(out *strings.Builder, name string, s *ModeMetrics) {
	fmt.Fprintf(out, "\n%s: %d/%d checks completed, %d failed, %d incomplete, %d skipped, %d unknown", name, s.Completed, s.Selected, s.Failed, s.Incomplete, s.Skipped, s.Unknown)
	for _, property := range slices.Sorted(maps.Keys(s.GoHost)) {
		evidence := s.GoHost[property]
		fmt.Fprintf(out, "\n  %s: %d planned, %d observed, %d verified", property, evidence.Planned, evidence.Executed, evidence.Verified)
	}
	if len(s.BrowserRoutes) > 0 {
		summariseRoutes(out, s.BrowserRoutes)
	}
	if s.MeasuredCPUChecks > 0 {
		fmt.Fprintf(out, ", %s cgroup CPU over %d checks", microseconds(s.CPUUsec), s.MeasuredCPUChecks)
	}
	if sensitivity := s.Sensitivity; sensitivity != nil {
		summariseSensitivity(out, sensitivity)
	}
	if s.MeasuredWorkDirs > 0 {
		fmt.Fprintf(out, ", largest end-of-check work directory %d bytes", s.LargestEndWorkDirBytes)
	}
	if s.LargestEstimateBytes != 0 {
		fmt.Fprintf(out, ", %d/%d admissions granted (%s waiting)", s.AdmissionGranted, s.AdmissionGranted+s.AdmissionDenied, s.AdmissionElapsed.Round(time.Millisecond))
	}
	if len(s.Workflows) > 0 {
		summariseWorkflows(out, s.Workflows)
	}
	for _, matrixName := range slices.Sorted(maps.Keys(s.Matrices)) {
		summariseMatrix(out, matrixName, s.Matrices[matrixName])
	}
	if s.ProgramRequests > 0 {
		summariseProgram(out, s)
	}
	if s.FirstInInvocationMemory.MeasuredChecks+s.SubsequentMemory.MeasuredChecks > 0 {
		fmt.Fprintf(out, "\n  sampled cgroup memory.current peak: first check per invocation %s; subsequent %s",
			memoryPeak(&s.FirstInInvocationMemory), memoryPeak(&s.SubsequentMemory))
	}
}

// summariseRoutes writes the total browser-route counts of one mode.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes routes (map[string]BrowserRouteCounts) which holds the counts by route.
func summariseRoutes(out *strings.Builder, routes map[string]BrowserRouteCounts) {
	var total BrowserRouteCounts
	for _, route := range routes {
		total.Eligible += route.Eligible
		total.Selected += route.Selected
		total.Visited += route.Visited
		total.Verified += route.Verified
	}
	fmt.Fprintf(out, ", browser routes %d eligible/%d selected/%d visited/%d verified", total.Eligible, total.Selected, total.Visited, total.Verified)
}

// summariseSensitivity writes the mutation kill and clean-control counts of one mode.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes sensitivity (*SensitivityMetrics) which holds the checker-sensitivity totals.
func summariseSensitivity(out *strings.Builder, sensitivity *SensitivityMetrics) {
	var mutated, killed, controls, valid uint64
	for _, counts := range sensitivity.Checks {
		if counts.Control {
			controls += counts.Attempts
			valid += counts.ExpectedResults
		} else {
			mutated += counts.Attempts
			killed += counts.ExpectedResults
		}
	}
	fmt.Fprintf(out, ", killed %d/%d observation mutations; %d/%d clean controls passed", killed, mutated, valid, controls)
	if sensitivity.CPUUsec > 0 && sensitivity.MeasuredCPUChecks == sensitivity.Cases {
		fmt.Fprintf(out, ", %.1f killed/check CPU-second", float64(killed)*microsecondsPerSecond/float64(sensitivity.CPUUsec))
	}
}

// summariseWorkflows writes the total workflow-transition counts of one mode.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes workflows (map[string]TransitionCounts) which holds the counts by step.
func summariseWorkflows(out *strings.Builder, workflows map[string]TransitionCounts) {
	var total TransitionCounts
	for _, transition := range workflows {
		total.Planned += transition.Planned
		total.Executed += transition.Executed
		total.Reached += transition.Reached
		total.Observed += transition.Observed
		total.LegacyVerified += transition.LegacyVerified
		total.Verified += transition.Verified
	}
	fmt.Fprintf(out, "\n  workflow transitions: %d planned, %d executed, %d independently verified", total.Planned, total.Executed, total.Verified)
	fmt.Fprintf(out, ", %d reached and %d observed", total.Reached, total.Observed)
	if total.LegacyVerified != 0 {
		fmt.Fprintf(out, ", %d historical prefix verifications", total.LegacyVerified)
	}
}

// summariseMatrix writes the distinct-cell proof counts of one finite matrix.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes name (string) which identifies the matrix.
// Takes matrix (*MatrixEvidence) which holds the matrix's cell evidence.
func summariseMatrix(out *strings.Builder, name string, matrix *MatrixEvidence) {
	selected, reached, observed, verified, legacy := 0, 0, 0, 0, 0
	for _, count := range matrix.Cells {
		if count.Selected > 0 {
			selected++
		}
		if count.Reached > 0 {
			reached++
		}
		if count.Observed > 0 {
			observed++
		}
		if count.LegacyVerified > 0 {
			legacy++
		}
		if count.Verified > 0 {
			verified++
		}
	}
	fmt.Fprintf(out, "\n  %s: %d generated, %d selected, %d reached, %d observed, %d/%d distinct cells verified, %d outstanding",
		name, len(matrix.Cells), selected, reached, observed, verified, matrix.Feasible, matrix.Feasible-verified)
	if legacy > 0 {
		fmt.Fprintf(out, ", %d cells with historical whole-mode credit", legacy)
	}
}

// summariseProgram writes the program-model proof counts of one mode.
//
// Takes out (*strings.Builder) which receives the summary text.
// Takes s (*ModeMetrics) which holds the mode's program counts.
func summariseProgram(out *strings.Builder, s *ModeMetrics) {
	fmt.Fprintf(out, "\n  program model: %d/%d requests and %d/%d output rows verified", s.VerifiedProgramRequests, s.ProgramRequests, s.VerifiedProgramOutputs, s.ProgramOutputs)
	fmt.Fprintf(out, ", %d requests reached/%d observed, %d rows reached/%d observed", s.ReachedProgramRequests, s.ObservedProgramRequests, s.ReachedProgramOutputs, s.ObservedProgramOutputs)
	if s.LegacyVerifiedProgramRequests > 0 || s.LegacyVerifiedProgramOutputs > 0 {
		fmt.Fprintf(out, ", historical whole-mode credit %d requests and %d rows", s.LegacyVerifiedProgramRequests, s.LegacyVerifiedProgramOutputs)
	}
}

// memoryPeak formats a memory high-water mark with its sample counts.
//
// Takes memory (*MemoryMetrics) which holds the accumulated memory readings.
//
// Returns string which is the formatted peak, or "unmeasured" without samples.
func memoryPeak(memory *MemoryMetrics) string {
	if memory.MeasuredChecks == 0 {
		return "unmeasured"
	}
	return fmt.Sprintf("%.1f MiB (%d checks, %d samples)", float64(memory.MaxSampledBytes)/mebibyte, memory.MeasuredChecks, memory.Samples)
}

// microseconds formats a microsecond count as a millisecond-rounded duration.
//
// Takes usec (uint64) which is the count in microseconds, saturated at the largest
// duration.
//
// Returns time.Duration which is the rounded duration.
func microseconds(usec uint64) time.Duration {
	return (time.Duration(safeconv.Uint64ToInt64(usec)) * time.Microsecond).Round(time.Millisecond)
}

// clientStepsObserved reports whether a client trace holds every expected action step.
//
// Takes actual (*casefile.ClientTransition) which holds the observed client trace, or
// nil.
// Takes expected (*casefile.ClientTransition) which holds the expected client steps.
//
// Returns bool which reports whether the trace is complete and matches each action.
func clientStepsObserved(actual *casefile.ClientTransition, expected *casefile.ClientTransition) bool {
	if actual == nil || len(actual.Before) == 0 || len(actual.After) == 0 || len(actual.Steps) != len(expected.Steps) {
		return false
	}
	for index := range actual.Steps {
		step := &actual.Steps[index]
		if step.Action != expected.Steps[index].Action || step.After == nil {
			return false
		}
	}
	return true
}
