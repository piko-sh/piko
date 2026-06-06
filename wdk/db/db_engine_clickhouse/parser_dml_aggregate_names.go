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

package db_engine_clickhouse

import (
	"strings"
)

var (
	// aggregateFunctionNames is the lowercased catalogue of every ClickHouse aggregate
	// function recognised by the expression analyser's hasAggregate flag.
	//
	// The flag drives downstream HAVING validation and the projection-level
	// aggregate-vs-window disambiguation so the set must stay in lockstep with the function
	// catalogue registered by the builtin_functions_*.go files.
	//
	// The membership test is a single map lookup; adding a new aggregate is one line.
	// Combinator suffixes (If, OrNull, Array, State, Merge, MergeState, Distinct, ForEach,
	// Resample, Map, SimpleState) are stripped by splitCombinatorSuffixes before the lookup
	// so countIf, sumOrNull and quantileExactIfState all resolve.
	aggregateFunctionNames = map[string]struct{}{
		// Core scalar aggregates.
		"count":                       {},
		"sum":                         {},
		"avg":                         {},
		"min":                         {},
		"max":                         {},
		"any":                         {},
		"anylast":                     {},
		"anyheavy":                    {},
		"sumkahan":                    {},
		"sumwithoverflow":             {},
		"deltasum":                    {},
		"deltasumtimestamp":           {},
		"intervallengthsum":           {},
		"exponentialmovingaverage":    {},
		"boundingratio":               {},
		"largesttrianglethreebuckets": {},
		"categoricalinformationvalue": {},

		// Cardinality and distinct-count aggregates.
		"uniq":                      {},
		"uniqexact":                 {},
		"uniqhll12":                 {},
		"uniqcombined":              {},
		"uniqcombined64":            {},
		"uniqtheta":                 {},
		"distinctdynamictypes":      {},
		"distinctjsonpaths":         {},
		"distinctjsonpathsandtypes": {},

		// Group / collection aggregates.
		"grouparray":          {},
		"groupuniqarray":      {},
		"grouparraymovingsum": {},
		"grouparraymovingavg": {},
		"grouparrayarray":     {},
		"grouparrayintersect": {},
		"groupbitor":          {},
		"groupbitand":         {},
		"groupbitxor":         {},
		"groupbitmap":         {},
		"groupbitmapand":      {},
		"groupbitmapor":       {},
		"groupbitmapxor":      {},

		// argMin / argMax family.
		"argmin":    {},
		"argmax":    {},
		"argandmin": {},
		"argandmax": {},

		// Quantile aggregates and their distribution-specific variants.
		"quantile":                          {},
		"quantileexact":                     {},
		"quantiletdigest":                   {},
		"median":                            {},
		"quantiles":                         {},
		"quantilebfloat16":                  {},
		"quantilegk":                        {},
		"quantileinterpolatedweighted":      {},
		"quantileexactexclusive":            {},
		"quantileexactinclusive":            {},
		"quantileexacthigh":                 {},
		"quantileexactlow":                  {},
		"quantileexactweightedinterpolated": {},
		"quantileprometheushistogram":       {},
		"quantiletdigestweighted":           {},
		"quantiletiming":                    {},
		"quantiletimingweighted":            {},
		"quantilesexactexclusive":           {},
		"quantilesexactinclusive":           {},
		"quantilesgk":                       {},
		"quantilestimingweighted":           {},
		"quantiledeterministic":             {},
		"quantiledd":                        {},

		// Variance / standard deviation aggregates.
		"varpop":           {},
		"varsamp":          {},
		"varpopstable":     {},
		"varsampstable":    {},
		"stddevpop":        {},
		"stddevsamp":       {},
		"stddevpopstable":  {},
		"stddevsampstable": {},

		// Covariance and correlation aggregates.
		"covarpop":        {},
		"covarsamp":       {},
		"covarpopstable":  {},
		"covarsampstable": {},
		"covarpopmatrix":  {},
		"covarsampmatrix": {},
		"corr":            {},
		"corrstable":      {},
		"corrmatrix":      {},

		// Higher moment aggregates.
		"kurtpop":  {},
		"kurtsamp": {},
		"skewpop":  {},
		"skewsamp": {},
		"entropy":  {},

		// Top-k and ranking aggregates.
		"topk":         {},
		"topkweighted": {},
		"rankcorr":     {},

		// Regression aggregates.
		"simplelinearregression":       {},
		"stochasticlinearregression":   {},
		"stochasticlogisticregression": {},

		// Sequence / pattern aggregates.
		"sequencematch":       {},
		"sequencecount":       {},
		"sequencematchevents": {},
		"sequencenextnode":    {},
		"windowfunnel":        {},
		"retention":           {},
		"uniqupto":            {},

		// Map aggregates.
		"summap":                     {},
		"minmap":                     {},
		"maxmap":                     {},
		"summapwithoverflow":         {},
		"summapfiltered":             {},
		"summapfilteredwithoverflow": {},

		// Statistical test aggregates.
		"mannwhitneyutest":      {},
		"studentttest":          {},
		"studentttestonesample": {},
		"welchttest":            {},
		"kolmogorovsmirnovtest": {},
		"meanztest":             {},
		"proportionsztest":      {},
		"analysisofvariance":    {},
		"cramersv":              {},
		"cramersvbiascorrected": {},
		"theilsu":               {},
		"contingency":           {},

		// Intersection and series helpers.
		"maxintersections":         {},
		"maxintersectionsposition": {},
		"sparkbar":                 {},
		"estimatecompressionratio": {},
		"flamegraph":               {},

		// Time-series grid aggregates.
		"timeserieschangestogrid":               {},
		"timeseriesdeltatogrid":                 {},
		"timeseriesderivtogrid":                 {},
		"timeseriesgrouparray":                  {},
		"timeseriesinstantdeltatogrid":          {},
		"timeseriesinstantratetogrid":           {},
		"timeserieslasttwosamples":              {},
		"timeseriespredictlineartogrid":         {},
		"timeseriesratetogrid":                  {},
		"timeseriesresampletogridwithstaleness": {},
		"timeseriesresetstogrid":                {},

		// Window-shaped aggregates that double as first / last selectors.
		"first_value": {},
		"last_value":  {},
	}
)

// isAggregateName reports whether the supplied function name is one of the ClickHouse
// aggregate functions.
//
// Combinator suffixes are stripped before lookup so countIf, sumOrNull and
// quantileExactIf all resolve correctly. The lookup is a single map fetch.
//
// Takes name (string) which is the candidate function name to test.
//
// Returns bool which is true when the name resolves to a known aggregate.
func isAggregateName(name string) bool {
	base, _ := splitCombinatorSuffixes(name)
	_, found := aggregateFunctionNames[strings.ToLower(base)]
	return found
}
