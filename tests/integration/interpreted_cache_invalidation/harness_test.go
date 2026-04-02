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

package cache_invalidation_test

import (
	"regexp"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/caller"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/config/config_domain"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/render/render_domain"
	"piko.sh/piko/internal/shutdown"
	"piko.sh/piko/wdk/logger"
)

type testCase struct {
	Name string
	Path string
}

var csrfTokenRegex = regexp.MustCompile(`<meta name="csrf-(token|ephemeral)" content="[^"]*">`)

func normaliseHTML(html []byte) []byte {
	return csrfTokenRegex.ReplaceAll(html, []byte(`<meta name="csrf-$1" content="[NORMALIZED]">`))
}

func resetGlobalStateForTestIsolation() {
	shutdown.Reset()
	logger.ResetLogger()

	config_domain.ResetGlobalFlagCoordinator()
	config_domain.ResetGlobalResolverRegistry()
	config_domain.ResetDotEnvCache()
	config_domain.ResetSecretManager()

	ast_domain.ClearExpressionCache()
	ast_domain.ResetAllPools()

	compiler_domain.ClearIdentifierRegistry()
	compiler_domain.ClearBindingRegistry()
	compiler_domain.ClearLocRefRegistry()

	collection_domain.ResetStaticCollectionRegistry()
	collection_domain.ResetRuntimeProviderRegistry()
	collection_domain.ResetHybridRegistry()

	goastutil.ResetDynamicCaches()
	ast_domain.ClearSelectorCache()
	i18n_domain.ClearExpressionCache()
	generator_helpers.ClearModulePathCaches()
	render_domain.ClearHTMLLinksCache()
	caller.ResetFrameCache()
	logger_domain.ResetCallerCache()
}
