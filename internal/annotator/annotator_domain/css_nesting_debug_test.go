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

// Test to verify the exact CSS from nd-estates-website
package annotator_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/esbuild/compat"
	"piko.sh/piko/internal/esbuild/config"
)

func TestCSSProcessor_NDEstatesFormElements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	inputCSS := `.form-section {
    display: flex;
    flex-direction: column;
    gap: 25px;
    &:not(:last-of-type) {
        margin-bottom: 50px;
    }
    .form-heading {
        font-weight: 500;
        font-size: 1.25rem;
        color: var(--g-colour-grey-2);
        display: flex;
        align-items: center;
        gap: 8px;
        text-wrap: nowrap;
        &::after {
            content: '';
            flex: 1;
            height: 1px;
            background-color: var(--g-colour-grey-5);
        }
    }
}

@media screen and (max-width: 768px) {
    .form-section {
        gap: 15px;
        &:not(:last-of-type) {
            margin-bottom: 30px;
        }
        .form-heading {
            font-size: 1.125rem;
        }
    }
}`

	output, diagnostics, err := processor.Process(ctx, inputCSS, "test.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0}, fsReader)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(output)
	t.Logf("Output: %s", normalised)

	assert.NotContains(t, normalised, "&:not(:last-of-type)", "Should not contain unexpanded & syntax")
	assert.NotContains(t, normalised, "&::after", "Should not contain unexpanded & syntax")

	assert.Contains(t, normalised, ".form-section:not(:last-of-type)", "Should contain expanded selector")
	assert.Contains(t, normalised, ".form-heading:after", "Should contain expanded pseudo-element (esbuild normalises :: to :)")
}
