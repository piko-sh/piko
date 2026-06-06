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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestBuiltinFunctions_StatisticalAggregates(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"kolmogorovsmirnovtest",
		"mannwhitneyutest",
		"welchttest",
		"studentttest",
		"studentttestonesample",
		"meanztest",
		"proportionsztest",
		"cramersv",
		"cramersvbiascorrected",
		"theilsu",
		"contingency",
		"categoricalinformationvalue",
		"rankcorr",
		"analysisofvariance",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
			assert.True(testRunner, signatures[0].IsAggregate, "function %q should be tagged as an aggregate", key)
		})
	}
}

func TestBuiltinFunctions_ExtendedAggregates(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"argandmin", "argandmax",
		"corrmatrix", "corrstable",
		"covarpopmatrix", "covarpopstable",
		"covarsampmatrix", "covarsampstable",
		"varpopstable", "varsampstable",
		"stddevpopstable", "stddevsampstable",
		"stochasticlogisticregression",
		"distinctdynamictypes", "distinctjsonpaths", "distinctjsonpathsandtypes",
		"estimatecompressionratio", "flamegraph",
		"grouparrayarray", "grouparrayintersect",
		"maxintersections", "maxintersectionsposition",
		"sparkbar",
		"quantiledd", "quantileexactexclusive", "quantileexactinclusive",
		"quantileexacthigh", "quantileexactlow",
		"quantileexactweightedinterpolated", "quantileprometheushistogram",
		"quantilesexactexclusive", "quantilesexactinclusive",
		"quantilesgk", "quantilestimingweighted", "quantiletdigestweighted",
		"summapwithoverflow", "summapfiltered", "summapfilteredwithoverflow",
		"timeserieschangestogrid", "timeseriesdeltatogrid", "timeseriesderivtogrid",
		"timeseriesgrouparray", "timeseriesinstantdeltatogrid",
		"timeseriesinstantratetogrid", "timeserieslasttwosamples",
		"timeseriespredictlineartogrid", "timeseriesratetogrid",
		"timeseriesresampletogridwithstaleness", "timeseriesresetstogrid",
		"sequencematchevents", "sequencenextnode", "uniqupto",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
			assert.True(testRunner, signatures[0].IsAggregate, "function %q should be tagged as an aggregate", key)
		})
	}
}

func TestBuiltinFunctions_BitmapFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []struct {
		key       string
		aggregate bool
	}{
		{key: "bitmapbuild"},
		{key: "bitmaptoarray"},
		{key: "bitmapsubsetinrange"},
		{key: "bitmapsubsetlimit"},
		{key: "subbitmap"},
		{key: "bitmapcontains"},
		{key: "bitmaphasall"},
		{key: "bitmaphasany"},
		{key: "bitmapmin"},
		{key: "bitmapmax"},
		{key: "bitmaptransform"},
		{key: "bitmapand"},
		{key: "bitmapor"},
		{key: "bitmapxor"},
		{key: "bitmapandnot"},
		{key: "bitmapandcardinality"},
		{key: "bitmaporcardinality"},
		{key: "bitmapxorcardinality"},
		{key: "bitmapandnotcardinality"},
		{key: "bitmapcardinality"},
		{key: "groupbitmap", aggregate: true},
		{key: "groupbitmapand", aggregate: true},
		{key: "groupbitmapor", aggregate: true},
		{key: "groupbitmapxor", aggregate: true},
	}
	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[testCase.key]
			require.True(testRunner, found, "function %q is missing from the catalogue", testCase.key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", testCase.key)
			if testCase.aggregate {
				assert.True(testRunner, signatures[0].IsAggregate, "function %q should be tagged as an aggregate", testCase.key)
			}
		})
	}
}

func TestBuiltinFunctions_DistanceAndNormFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"lpdistance", "lpnorm",
		"l2squareddistance", "l2squarednorm",
		"l1normalize", "l2normalize", "linfnormalize", "lpnormalize",
		"l2distancetransposed", "cosinedistancetransposed",
		"tuplehammingdistance", "bytehammingdistance",
		"arrayhammingdistance", "bithammingdistance",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_NLPFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"detectcharset", "detectlanguage", "detectlanguagemixed",
		"detectlanguageunknown", "stem", "lemmatize", "synonyms",
		"detecttonality",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_GeoS2Functions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"geotos2", "s2togeo", "s2getneighbors", "s2cellsintersect",
		"s2capcontains", "s2capunion",
		"s2rectadd", "s2rectcontains", "s2rectunion", "s2rectintersection",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_GeoPolygonFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"readwktpoint", "readwktlinestring", "readwktpolygon",
		"readwktmultipolygon", "readwktring",
		"readwkbpoint", "readwkblinestring", "readwkbpolygon",
		"readwkbmultipolygon", "readwkbmultilinestring",
		"wkt",
		"polygonsintersectioncartesian", "polygonsintersectionspherical",
		"polygonsunioncartesian", "polygonsunionspherical",
		"polygonssymdifferencecartesian", "polygonssymdifferencespherical",
		"polygonsintersectcartesian", "polygonsintersectspherical",
		"polygonswithincartesian", "polygonswithinspherical",
		"polygonsequalscartesian", "polygonconvexhullcartesian",
		"polygonareacartesian", "polygonareaspherical",
		"polygonperimetercartesian", "polygonperimeterspherical",
		"polygonsdistancecartesian", "polygonsdistancespherical",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_H3ExtendedFunctions(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"h3edgeangle", "h3edgelengthm", "h3edgelengthkm",
		"h3exactedgelengthm", "h3exactedgelengthkm", "h3exactedgelengthrads",
		"h3hexaream2", "h3hexareakm2", "h3cellaream2", "h3cellarearads2",
		"h3numhexagons", "h3getbasecell", "h3getfaces",
		"h3tochildren", "h3tocenterchild", "h3tostring", "stringtoh3",
		"h3getres0indexes", "h3getpentagonindexes", "h3togeoboundary",
		"h3isresclassiii", "h3ispentagon", "h3indexesareneighbors",
		"h3line", "h3distance", "h3hexring",
		"h3pointdistm", "h3pointdistkm", "h3pointdistrads",
		"h3polygontocells", "h3polygontocellswithcontainment",
		"h3getunidirectionaledge", "h3unidirectionaledgeisvalid",
		"h3getoriginindexfromunidirectionaledge",
		"h3getdestinationindexfromunidirectionaledge",
		"h3getindexesfromunidirectionaledge",
		"h3getunidirectionaledgesfromhexagon",
		"h3getunidirectionaledgeboundary",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_ExtendedMathRoundingBit(t *testing.T) {
	t.Parallel()

	engine := NewClickHouseEngine()
	functions := engine.BuiltinFunctions()
	require.NotNil(t, functions)

	cases := []string{
		"sinh", "cosh", "tanh", "asinh", "acosh", "atanh",
		"erf", "erfc", "lgamma", "tgamma",
		"exp2", "exp10", "intexp2", "intexp10", "log1p",
		"hypot", "degrees", "radians",
		"isprime", "isprobableprime",
		"ceiling", "trunc",
		"roundage", "roundduration", "rounddown", "roundtoexp2",
		"bitslice", "bitpositionstoarray",
	}
	for index := range cases {
		key := cases[index]
		t.Run(key, func(testRunner *testing.T) {
			testRunner.Parallel()
			signatures, found := functions.Functions[key]
			require.True(testRunner, found, "function %q is missing from the catalogue", key)
			require.NotEmpty(testRunner, signatures, "function %q has no signatures", key)
		})
	}
}

func TestBuiltinFunctions_StatisticalCombinatorSuffix(t *testing.T) {
	t.Parallel()

	assert.True(t, isAggregateName("welchTTest"), "base welchTTest should be tagged as aggregate")
	assert.True(t, isAggregateName("welchTTestIf"), "welchTTestIf should resolve after combinator strip")
	assert.True(t, isAggregateName("kolmogorovSmirnovTestIf"), "kolmogorovSmirnovTestIf should resolve after combinator strip")
	assert.True(t, isAggregateName("mannWhitneyUTestArray"), "Array combinator should not block resolution")
	assert.True(t, isAggregateName("quantileDD"), "quantileDD should be tagged as aggregate")
	assert.True(t, isAggregateName("quantileDDIf"), "quantileDDIf should resolve after combinator strip")
}

func TestBuiltinFunctions_AggregateCombinatorReturnTypes(t *testing.T) {
	t.Parallel()

	resolver := NewClickHouseFunctionResolver()

	sumArgs := []querier_dto.SQLType{
		{Category: querier_dto.TypeCategoryFloat, EngineName: "Float64"},
	}

	cases := []struct {
		name string

		args []querier_dto.SQLType

		assertion func(testRunner *testing.T, returnType querier_dto.SQLType)
	}{
		{
			name: "sumIf",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "If preserves base return type")
				assert.False(testRunner, returnType.Nullable, "If does not introduce nullability")
			},
		},
		{
			name: "sumOrNull",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "OrNull preserves engine name")
				assert.True(testRunner, returnType.Nullable, "OrNull wraps the result in Nullable")
			},
		},
		{
			name: "sumOrDefault",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "OrDefault preserves base type")
				assert.False(testRunner, returnType.Nullable, "OrDefault produces a non-null fallback")
			},
		},
		{
			name: "sumArray",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "Array combinator preserves base engine name")
			},
		},
		{
			name: "sumState",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "AggregateFunction", returnType.EngineName, "State wraps the result in AggregateFunction")
			},
		},
		{
			name: "sumMerge",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "Merge reads the state back to the base type")
			},
		},
		{
			name: "sumMergeState",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "AggregateFunction", returnType.EngineName, "MergeState yields the same shape as State")
			},
		},
		{
			name: "sumSimpleState",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "SimpleAggregateFunction", returnType.EngineName, "SimpleState produces a SimpleAggregateFunction wrapper")
			},
		},
		{
			name: "sumResample",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Array", returnType.EngineName, "Resample wraps the result in Array")
				require.NotNil(testRunner, returnType.ElementType)
				assert.Equal(testRunner, "Float64", returnType.ElementType.EngineName, "Resample element type matches base")
			},
		},
		{
			name: "sumDistinct",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "Distinct only affects the input pipeline, not the return")
			},
		},
		{
			name: "sumForEach",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Array", returnType.EngineName, "ForEach wraps the result in Array")
				require.NotNil(testRunner, returnType.ElementType)
				assert.Equal(testRunner, "Float64", returnType.ElementType.EngineName, "ForEach element type matches base")
			},
		},
		{
			name: "sumMap",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Map", returnType.EngineName, "Map wraps the result in Map(String, T)")
				require.NotNil(testRunner, returnType.KeyType)
				require.NotNil(testRunner, returnType.ElementType)
				assert.Equal(testRunner, "String", returnType.KeyType.EngineName, "Map key falls back to String")
				assert.Equal(testRunner, "Float64", returnType.ElementType.EngineName, "Map value matches base")
			},
		},
		{
			name: "sumArgMin",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "ArgMin preserves the base type")
			},
		},
		{
			name: "sumArgMax",
			args: sumArgs,
			assertion: func(testRunner *testing.T, returnType querier_dto.SQLType) {
				assert.Equal(testRunner, "Float64", returnType.EngineName, "ArgMax preserves the base type")
			},
		},
	}
	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(testRunner *testing.T) {
			testRunner.Parallel()
			resolution, err := resolver.ResolveFunctionCall(nil, testCase.name, "", testCase.args)
			require.NoError(testRunner, err, "resolver should not return an error for %q", testCase.name)
			require.NotNil(testRunner, resolution, "resolver should produce a resolution for %q", testCase.name)
			testCase.assertion(testRunner, resolution.ReturnType)
		})
	}
}

func TestBuiltin_Now64ZeroArgOverload(t *testing.T) {
	t.Parallel()

	functions := NewClickHouseEngine().BuiltinFunctions()
	signatures, found := functions.Functions["now64"]
	require.True(t, found)
	argCounts := make(map[int]bool)
	for _, signature := range signatures {
		argCounts[len(signature.Arguments)] = true
	}
	assert.True(t, argCounts[0], "now64() zero-argument overload must be registered")
	assert.True(t, argCounts[1], "now64(precision) overload must remain registered")
}

func TestBuiltin_ExtractURLParameterArity(t *testing.T) {
	t.Parallel()

	functions := NewClickHouseEngine().BuiltinFunctions()
	signatures, found := functions.Functions["extracturlparameter"]
	require.True(t, found)
	require.NotEmpty(t, signatures)
	assert.Len(t, signatures[0].Arguments, 2, "extractURLParameter(URL, name) takes two arguments")
}

func TestBuiltin_PredicateReturnTypes(t *testing.T) {
	t.Parallel()

	functions := NewClickHouseEngine().BuiltinFunctions()
	for _, name := range []string{"bitmapcontains", "s2cellsintersect", "polygonswithincartesian"} {
		signatures, found := functions.Functions[name]
		require.True(t, found, "%q missing", name)
		require.NotEmpty(t, signatures)
		assert.Equal(t, querier_dto.TypeCategoryBoolean, signatures[0].ReturnType.Category, "%q should return Bool", name)
	}
}
