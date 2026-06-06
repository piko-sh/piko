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

package querier_domain

import (
	"fmt"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// asyncExecBodyKey is the key the ClickHouse parser writes into the RawQueryAnalysis
	// EngineSpecific map for ALTER TABLE UPDATE / DELETE statements. The presence of the key
	// signals an asynchronous mutation body the engine has captured opaquely; the diagnostic
	// pass uses it to detect when exec should be promoted to asyncexec.
	asyncExecBodyKey = "ASYNC_BODY"
)

// asyncExecPass validates pairings of the QueryCommandAsyncExec command with the engine's
// capability flag and recommends asyncexec over exec when the underlying statement is an
// asynchronous mutation.
//
// The pass runs after command output validation so the recommendation never fires for
// queries that already declare a row-producing command.
//
// Two diagnostic paths exist. Q041 is a hard error when the query declares asyncexec but
// the engine reports SupportsAsyncMutations() == false; this prevents a silent downgrade
// to synchronous Exec on engines that lack the distinction between mutation acceptance
// and completion. Q040 is a hint when the query declares exec on an engine that supports
// async mutations and the raw analysis carries the ASYNC_BODY marker; the user is nudged
// to declare asyncexec so the generated method reflects the fire-and-forget semantics.
type asyncExecPass struct{}

// Analyse inspects the query command against the engine's asynchronous-mutation
// capability and the raw analysis's engine-specific markers.
//
// Takes context (*diagnosticContext) which holds the query, raw analysis, and the engine
// handle used to inspect capability flags. nil Engine is tolerated so the test harness
// can exercise the pass in isolation; in that case both diagnostic paths skip.
//
// Returns []querier_dto.SourceError which holds at most one diagnostic. The hard-error
// path (Q041) takes precedence over the info-level recommendation.
func (*asyncExecPass) Analyse(context *diagnosticContext) []querier_dto.SourceError {
	if context.Engine == nil {
		return nil
	}

	engineSupports := context.Engine.SupportsAsyncMutations()

	if context.Query.Command == querier_dto.QueryCommandAsyncExec && !engineSupports {
		return []querier_dto.SourceError{
			{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q declares asyncexec but engine %q does not surface asynchronous mutation semantics: %s",
					context.Query.Name, context.Engine.Dialect(), ErrAsyncExecNotSupported.Error(),
				),
				Severity: querier_dto.SeverityError,
				Code:     querier_dto.CodeAsyncExecNotSupported,
			},
		}
	}

	if context.Query.Command == querier_dto.QueryCommandExec && engineSupports && hasAsyncBodyMarker(context.RawAnalysis) {
		return []querier_dto.SourceError{
			{
				Filename: context.Filename,
				Line:     context.Query.Line,
				Column:   1,
				Message: fmt.Sprintf(
					"query %q uses 'exec' with an asynchronous mutation; consider 'asyncexec' to surface the fire-and-forget semantics",
					context.Query.Name,
				),
				Severity: querier_dto.SeverityHint,
				Code:     querier_dto.CodeAsyncExecRecommended,
			},
		}
	}

	return nil
}

// hasAsyncBodyMarker reports whether the raw analysis carries the ASYNC_BODY
// engine-specific marker. The ClickHouse parser writes this key for ALTER TABLE UPDATE
// and DELETE bodies; other engines leave the map untouched so the check defaults to
// false.
//
// Takes analysis (*querier_dto.RawQueryAnalysis) which may be nil or may carry a nil
// EngineSpecific map; both cases return false.
//
// Returns bool which is true when the marker is present.
func hasAsyncBodyMarker(analysis *querier_dto.RawQueryAnalysis) bool {
	if analysis == nil || analysis.EngineSpecific == nil {
		return false
	}
	_, present := analysis.EngineSpecific[asyncExecBodyKey]
	return present
}
