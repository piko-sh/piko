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

package cache_test

import (
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/apitest"
	"piko.sh/piko/wdk/cache"
)

func TestCacheFacadeAPI(t *testing.T) {

	surface := apitest.Surface{

		"Cache": (*cache.Cache[string, string])(nil),

		"Service": (*cache.Service)(nil),

		"Provider": (*cache.Provider)(nil),

		"ProviderPort": (*cache.ProviderPort[string, string])(nil),

		"EncoderPort": (*cache.EncoderPort[string])(nil),

		"AnyEncoder": (*cache.AnyEncoder)(nil),

		"TransformerPort": (*cache.TransformerPort)(nil),

		"Builder": (*cache.Builder[string, string])(nil),

		"Options": cache.Options[string, string]{},

		"Entry": cache.Entry[string, string]{},

		"DeletionEvent": cache.DeletionEvent[string, string]{},

		"Loader": (*cache.Loader[string, string])(nil),

		"LoaderFunc": cache.LoaderFunc[string, string](nil),

		"BulkLoader": (*cache.BulkLoader[string, string])(nil),

		"BulkLoaderFunc": cache.BulkLoaderFunc[string, string](nil),

		"LoadResult": cache.LoadResult[string]{},

		"ExpiryCalculator": (*cache.ExpiryCalculator[string, string])(nil),

		"RefreshCalculator": (*cache.RefreshCalculator[string, string])(nil),

		"Clock": (*cache.Clock)(nil),

		"Logger": (*cache.Logger)(nil),

		"TransformConfig": cache.TransformConfig{},

		"Stats": cache.Stats{},

		"StatsRecorder": (*cache.StatsRecorder)(nil),

		"DeletionCause":     cache.DeletionCause(0),
		"CauseInvalidation": cache.CauseInvalidation,
		"CauseReplacement":  cache.CauseReplacement,
		"CauseOverflow":     cache.CauseOverflow,
		"CauseExpiration":   cache.CauseExpiration,

		"ComputeAction":       cache.ComputeAction(0),
		"ComputeActionSet":    cache.ComputeActionSet,
		"ComputeActionDelete": cache.ComputeActionDelete,
		"ComputeActionNoop":   cache.ComputeActionNoop,

		"TransformerType":        cache.TransformerType(""),
		"TransformerCompression": cache.TransformerCompression,
		"TransformerEncryption":  cache.TransformerEncryption,
		"TransformerCustom":      cache.TransformerCustom,

		"ErrNotFound": cache.ErrNotFound,

		"NewService": cache.NewService,

		"NewCache": cache.NewCache[string, string],

		"NewCacheBuilder": cache.NewCacheBuilder[string, string],

		"GetDefaultService": cache.GetDefaultService,

		"NewCacheFromDefault": cache.NewCacheFromDefault[string, string],

		"NewCacheBuilderFromDefault": cache.NewCacheBuilderFromDefault[string, string],
	}

	apitest.Check(t, surface, filepath.Join("facade_test.golden.yaml"))
}
