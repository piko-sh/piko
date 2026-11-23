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
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/generate"
	"piko.sh/piko/tests/fuzz/internal/runner"
)

// CellEvidence counts a finite cell's generation and observed proof stages.
type CellEvidence struct {
	// Generated counts cases that generated the cell.
	Generated uint64

	// Selected counts cases that selected the cell for checking.
	Selected uint64

	// Reached counts cases whose worker reached the cell.
	Reached uint64

	// Observed counts cases with an observed receipt for the cell.
	Observed uint64

	// Verified counts cases that independently verified the cell.
	Verified uint64

	// LegacyVerified preserves verification totals from older reports.
	LegacyVerified uint64 `json:",omitempty"`
}

// matrixProofs holds the independent proof stages used to credit finite matrix cells.
type matrixProofs struct {
	// c holds the saved case.
	c *casefile.Case

	// check holds the mode check's measured outcome.
	check *CheckMetric

	// o holds the worker observation.
	o *runner.Observation

	// serverCells maps server-rendered matrix cells to their program proof stages.
	serverCells map[string]map[string]runner.Proof

	// programs holds independent program proof stages aligned with the saved probes.
	programs []runner.ProgramProof

	// workflows holds independent workflow proof stages.
	workflows []runner.Proof
}

// UnmarshalJSON decodes a receipt while retaining legacy evidence separately.
//
// Takes data ([]byte) which holds the encoded receipt JSON.
//
// Returns error which reports invalid receipt JSON or unsupported legacy field values, or
// nil on success.
func (e *CellEvidence) UnmarshalJSON(data []byte) error {
	type alias CellEvidence
	var record struct {
		Reached *uint64
		alias
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return err
	}
	*e = CellEvidence(record.alias)
	if record.Reached == nil {
		e.LegacyVerified += e.Verified
		e.Verified = 0
	} else {
		e.Reached = *record.Reached
	}
	return nil
}

// observeCaseObligations accumulates proof for the saved case's selected obligations.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes mode (string) which selects the server mode.
// Takes check (*CheckMetric) which holds one mode check's measured outcome.
// Takes o (*runner.Observation) which holds the worker observation.
func (s *ModeMetrics) observeCaseObligations(c *casefile.Case, mode string, check *CheckMetric, o *runner.Observation) {
	if o.Mode != "" && o.Mode != mode {
		o = &runner.Observation{}
	}
	programs := runner.ProgramProofs(c, o)
	var workflows []runner.Proof
	if check.BrowserSelected {
		workflows = runner.WorkflowProofs(c, o)
	}
	s.observeMatrices(newMatrixProofs(c, check, o, programs, workflows))
	if mode == modeInterp || mode == modeCompiled {
		s.observePrograms(c, programs)
		s.observeWorkflows(c, check, workflows)
	}
}

// observePrograms accumulates request and output proof stages.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes audits ([]runner.ProgramProof) which holds program proofs aligned by index with
// the saved probes.
func (s *ModeMetrics) observePrograms(c *casefile.Case, audits []runner.ProgramProof) {
	for index := range c.Probes {
		probe := &c.Probes[index]
		if !probe.Program {
			continue
		}
		s.ProgramRequests++
		s.ProgramOutputs += uint64(len(probe.Output))
		if index >= len(audits) {
			continue
		}
		audit := audits[index]
		s.ReachedProgramRequests += countStage(audit.Request >= runner.Reached)
		s.ObservedProgramRequests += countStage(audit.Request >= runner.Observed)
		s.VerifiedProgramRequests += countStage(audit.Request == runner.Verified)
		for _, stage := range audit.Rows {
			s.ReachedProgramOutputs += countStage(stage >= runner.Reached)
			s.ObservedProgramOutputs += countStage(stage >= runner.Observed)
			s.VerifiedProgramOutputs += countStage(stage == runner.Verified)
		}
	}
}

// observeMatrices accumulates finite-cell proof from independently measured operations.
//
// Takes proofs (*matrixProofs) which holds the case, its check and the independent proof
// stages.
func (s *ModeMetrics) observeMatrices(proofs *matrixProofs) {
	c := proofs.c
	for _, matrix := range generate.Matrices(c) {
		if s.Matrices == nil {
			s.Matrices = map[string]*MatrixEvidence{}
		}
		key := c.Target + "/" + matrix.Name
		evidence := s.Matrices[key]
		if evidence == nil {
			evidence = &MatrixEvidence{Feasible: matrix.Feasible, Cells: map[string]CellEvidence{}}
			s.Matrices[key] = evidence
		}
		selected := matrix.Environment != "browser" || proofs.check.BrowserSelected
		for _, id := range matrix.Cells {
			cell := evidence.Cells[id]
			cell.Generated++
			if selected {
				stage := proofs.stage(&matrix, id)
				cell.Selected++
				cell.Reached += countStage(stage >= runner.Reached)
				cell.Observed += countStage(stage >= runner.Observed)
				cell.Verified += countStage(stage == runner.Verified)
			}
			evidence.Cells[id] = cell
		}
	}
}

// UnmarshalJSON decodes a receipt while retaining legacy evidence separately.
//
// Takes data ([]byte) which holds the encoded receipt JSON.
//
// Returns error which reports invalid receipt JSON or unsupported legacy field values, or
// nil on success.
func (s *ModeMetrics) UnmarshalJSON(data []byte) error {
	type alias ModeMetrics
	var record struct {
		ReachedProgramRequests *uint64
		alias
	}
	if err := json.Unmarshal(data, &record); err != nil {
		return err
	}
	*s = ModeMetrics(record.alias)
	if record.ReachedProgramRequests != nil {
		s.ReachedProgramRequests = *record.ReachedProgramRequests
		return nil
	}
	s.LegacyVerifiedProgramRequests += s.VerifiedProgramRequests
	s.LegacyVerifiedProgramOutputs += s.VerifiedProgramOutputs
	s.VerifiedProgramRequests, s.VerifiedProgramOutputs = 0, 0

	for name, transition := range s.Workflows {
		transition.LegacyExecuted += transition.Executed
		transition.LegacyVerified += transition.Verified
		transition.Executed, transition.Verified = 0, 0
		s.Workflows[name] = transition
	}
	return nil
}

// stage selects the proof stage of one matrix cell.
//
// Takes matrix (*generate.Matrix) which is the finite matrix holding the cell.
// Takes id (string) which identifies the cell.
//
// Returns runner.Proof which is the cell's independent proof stage.
func (proofs *matrixProofs) stage(matrix *generate.Matrix, id string) runner.Proof {
	switch {
	case matrix.Environment == "package" && proofs.check.Status == checkStatusPass:
		return runner.Verified
	case matrix.Name == generate.ArithmeticMatrix || matrix.Name == generate.TypedMatrix:
		return proofs.serverCells[matrix.Name][id]
	case matrix.Name == generate.AuthPageMatrix:
		return runner.AuthPageProof(proofs.c, proofs.o, id)
	case matrix.Name == generate.AuthClaimMatrix:
		return min(runner.AuthPageProof(proofs.c, proofs.o, "reader-desk"), runner.AuthPageProof(proofs.c, proofs.o, "editor-desk"))
	case matrix.Name == generate.AuthActionMatrix:
		return runner.AuthActionProof(proofs.c, proofs.o, id)
	case matrix.Name == generate.FormShimMatrix:
		return runner.FormShimProof(proofs.c, proofs.o, id)
	case proofs.c.Target == "graph-rebuild":
		return graphProof(proofs.c, proofs.programs)
	case proofs.c.Target == "markup" || proofs.c.Target == "firefox-markup":
		return runner.DOMFamilyProof(proofs.c, proofs.o, id)
	case matrix.Environment == "browser":
		return runner.BrowserHistoryProof(proofs.c, proofs.o, proofs.workflows)
	}
	return runner.Unseen
}

// graphProof finds the weakest proof among the partial graph's program requests.
//
// Takes c (*casefile.Case) which holds the saved sources, choices and expected results.
// Takes programs ([]runner.ProgramProof) which holds independent program proof stages.
//
// Returns runner.Proof which is the graph request proof stage, or Unseen when none was
// reached.
func graphProof(c *casefile.Case, programs []runner.ProgramProof) runner.Proof {
	paths := map[string]bool{
		"/independent":              true,
		"/partial-graph?run=first":  true,
		"/partial-graph?run=second": true,
		"/twin?run=first":           true,
		"/twin?run=second":          true,
	}
	stage := runner.Verified
	reached := false
	for index := range c.Probes {
		probe := &c.Probes[index]
		if !paths[probe.Path] || !probe.Program || index >= len(programs) {
			continue
		}
		delete(paths, probe.Path)
		request := programs[index].Request
		stage = min(stage, request)
		reached = reached || request >= runner.Reached
	}
	if !reached {
		return runner.Unseen
	}
	if len(paths) != 0 || stage < runner.Observed {
		return runner.Reached
	}
	return stage
}

// newMatrixProofs gathers the independent proof stages that credit matrix cells.
//
// Takes c (*casefile.Case) which holds the saved case.
// Takes check (*CheckMetric) which holds the mode check's measured outcome.
// Takes o (*runner.Observation) which holds the worker observation.
// Takes programs ([]runner.ProgramProof) which holds program proofs aligned with the
// probes.
// Takes workflows ([]runner.Proof) which holds independent workflow proof stages.
//
// Returns *matrixProofs which holds the proof stages and the server-rendered cells.
func newMatrixProofs(
	c *casefile.Case, check *CheckMetric, o *runner.Observation, programs []runner.ProgramProof, workflows []runner.Proof,
) *matrixProofs {
	return &matrixProofs{
		c:           c,
		check:       check,
		o:           o,
		serverCells: serverMatrixCells(c, programs),
		programs:    programs,
		workflows:   workflows,
	}
}

// serverMatrixCells maps the server-rendered matrix cells of a case to their proof
// stages.
//
// Only the generated and bounded cells from Matrices are retained.
//
// Takes c (*casefile.Case) which holds the saved probes.
// Takes programs ([]runner.ProgramProof) which holds program proofs aligned with the
// probes.
//
// Returns map[string]map[string]runner.Proof which maps matrix names and cell identities
// to proof stages.
func serverMatrixCells(c *casefile.Case, programs []runner.ProgramProof) map[string]map[string]runner.Proof {
	cells := map[string]map[string]runner.Proof{}
	for probeIndex := range c.Probes {
		if probeIndex >= len(programs) {
			break
		}
		probe := &c.Probes[probeIndex]
		for rowIndex, stage := range programs[probeIndex].Rows {
			if rowIndex >= len(probe.Output) {
				break
			}
			name, id := generate.ProgramMatrixCell(*probe, probe.Output[rowIndex])
			if name == "" {
				continue
			}
			if cells[name] == nil {
				cells[name] = map[string]runner.Proof{}
			}
			cells[name][id] = stage
		}
	}
	return cells
}

// countStage converts a proof-stage test into a counter increment.
//
// Takes reached (bool) which reports whether the stage was reached.
//
// Returns uint64 which is one when reached and zero otherwise.
func countStage(reached bool) uint64 {
	if reached {
		return 1
	}
	return 0
}
