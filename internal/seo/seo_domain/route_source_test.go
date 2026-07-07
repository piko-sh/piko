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

package seo_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/seo/seo_dto"
)

func TestRouteContext_Expand_PrefixStrategyPrependsLocaleAndEscapesValue(t *testing.T) {
	rc := RouteContext{
		RoutePattern:  "/blog/{slug}",
		ParamName:     "slug",
		DefaultLocale: "en",
		Strategy:      i18n_domain.StrategyPrefix,
	}

	assert.Equal(t, "/fr/blog/hello%20world", rc.Expand("hello world", "fr"))
}

func TestRouteContext_Expand_PrefixExceptDefaultKeepsDefaultLocaleBare(t *testing.T) {
	rc := RouteContext{
		RoutePattern:  "/blog/{slug}",
		ParamName:     "slug",
		DefaultLocale: "en",
		Strategy:      i18n_domain.StrategyPrefixExceptDefault,
	}

	assert.Equal(t, "/blog/x", rc.Expand("x", "en"))
	assert.Equal(t, "/fr/blog/x", rc.Expand("x", "fr"))
}

func TestRouteContext_Expand_DisabledStrategyKeepsPatternBare(t *testing.T) {

	for _, strategy := range []string{i18n_domain.StrategyDisabled, i18n_domain.StrategyQueryOnly, ""} {
		rc := RouteContext{
			RoutePattern:  "/blog/{slug}",
			ParamName:     "slug",
			DefaultLocale: "en",
			Strategy:      strategy,
		}
		assert.Equal(t, "/blog/x", rc.Expand("x", "fr"), "strategy %q should keep the pattern bare", strategy)
	}
}

func TestRouteContext_Expand_SlashValueIsNotEncodedAcrossSegments(t *testing.T) {
	rc := RouteContext{
		RoutePattern:  "/blog/{slug}",
		ParamName:     "slug",
		DefaultLocale: "en",
		Strategy:      i18n_domain.StrategyDisabled,
	}

	assert.Equal(t, "/blog/a/b", rc.Expand("a/b", "en"))
}

func TestRouteContext_Expand_AccentedValueIsPercentEncoded(t *testing.T) {
	rc := RouteContext{
		RoutePattern:  "/blog/{slug}",
		ParamName:     "slug",
		DefaultLocale: "en",
		Strategy:      i18n_domain.StrategyDisabled,
	}

	assert.Equal(t, "/blog/caf%C3%A9", rc.Expand("café", "en"))
}

func TestRouteContext_Expand_EmptyParamNameYieldsEmptyString(t *testing.T) {
	rc := RouteContext{
		RoutePattern: "/blog/{slug}",
		ParamName:    "",
	}

	assert.Equal(t, "", rc.Expand("x", "en"))
}

func TestRouteContext_Expand_PatternWithoutLeadingSlashIsNormalised(t *testing.T) {
	rc := RouteContext{
		RoutePattern:  "blog/{slug}",
		ParamName:     "slug",
		DefaultLocale: "en",
		Strategy:      i18n_domain.StrategyDisabled,
	}

	assert.Equal(t, "/blog/x", rc.Expand("x", "en"))
}

func TestRouteSourceFunc_Name_ReturnsConfiguredSourceName(t *testing.T) {
	source := RouteSourceFunc{
		SourceName: "locations",
		Fn: func(_ context.Context, _ RouteContext) ([]RouteURL, error) {
			return nil, nil
		},
	}

	assert.Equal(t, "locations", source.Name())
}

func TestRouteSourceFunc_Enumerate_InvokesFnAndReturnsItsResult(t *testing.T) {
	want := []RouteURL{
		{ParamValue: "-jersey", SEO: seo_dto.SitemapURLInput{Priority: 0.7}},
	}

	var gotRC RouteContext
	source := RouteSourceFunc{
		SourceName: "locations",
		Fn: func(_ context.Context, rc RouteContext) ([]RouteURL, error) {
			gotRC = rc
			return want, nil
		},
	}

	rc := RouteContext{SourceName: "locations", RoutePattern: "/x/{slug}", ParamName: "slug"}
	got, err := source.Enumerate(context.Background(), rc)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, rc, gotRC, "the wrapped closure receives the passed RouteContext")
}

func TestRouteSourceFunc_Enumerate_PropagatesFnError(t *testing.T) {
	wantErr := errors.New("boom")
	source := RouteSourceFunc{
		SourceName: "locations",
		Fn: func(_ context.Context, _ RouteContext) ([]RouteURL, error) {
			return nil, wantErr
		},
	}

	got, err := source.Enumerate(context.Background(), RouteContext{})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, got)
}
