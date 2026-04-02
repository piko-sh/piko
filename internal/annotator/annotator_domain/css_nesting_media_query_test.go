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

func TestCSSProcessor_NestingInsideMediaQuery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	testCases := []struct {
		name             string
		inputCSS         string
		shouldContain    []string
		shouldNotContain []string
	}{
		{
			name: "ampersand with pseudo-class inside media query should be expanded",
			inputCSS: `
				.form-section {
					gap: 25px;
					&:not(:last-of-type) {
						margin-bottom: 50px;
					}
				}
				@media screen and (max-width: 768px) {
					.form-section {
						gap: 15px;
						&:not(:last-of-type) {
							margin-bottom: 30px;
						}
					}
				}
			`,
			shouldContain: []string{
				".form-section",
				"@media",

				".form-section:not(:last-of-type)",
			},
			shouldNotContain: []string{

				"&:not(:last-of-type)",
			},
		},
		{
			name: "nested class selector inside media query should be expanded",
			inputCSS: `
				.form-section {
					.form-heading {
						font-size: 1.25rem;
					}
				}
				@media screen and (max-width: 768px) {
					.form-section {
						.form-heading {
							font-size: 1.125rem;
						}
					}
				}
			`,
			shouldContain: []string{
				".form-section",
				".form-heading",
				"@media",
			},
			shouldNotContain: []string{},
		},
		{
			name: "multiple levels of nesting in media query",
			inputCSS: `
				.parent {
					color: red;
					.child {
						color: blue;
						&:hover {
							color: green;
						}
					}
				}
				@media (max-width: 768px) {
					.parent {
						.child {
							&:hover {
								color: purple;
							}
						}
					}
				}
			`,
			shouldContain: []string{
				".parent",
				".child",
				"@media",
			},
			shouldNotContain: []string{
				"&:hover",
			},
		},
		{
			name: "ampersand selector in both top-level and media query",
			inputCSS: `
				.button {
					&:hover {
						background: blue;
					}
				}
				@media (max-width: 768px) {
					.button {
						&:active {
							background: red;
						}
					}
				}
			`,
			shouldContain: []string{
				".button",
				"@media",
				".button:hover",
				".button:active",
			},
			shouldNotContain: []string{
				"&:hover",
				"&:active",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			output, diagnostics, err := processor.Process(ctx, tc.inputCSS, "test.css", ast_domain.Location{Line: 1, Column: 1, Offset: 0}, fsReader)
			require.NoError(t, err)
			assert.Empty(t, diagnostics, "Should not have any diagnostics")

			normalised := normaliseCSS(output)
			t.Logf("Normalised output: %s", normalised)

			for _, should := range tc.shouldContain {
				assert.Contains(t, normalised, normaliseCSS(should), "Output should contain: %s", should)
			}

			for _, shouldNot := range tc.shouldNotContain {
				assert.NotContains(t, normalised, normaliseCSS(shouldNot), "Output should NOT contain: %s", shouldNot)
			}
		})
	}
}

func TestCSSProcessor_NestingInsideMediaQuery_WithScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	fsReader := newTestFSReader()
	resolver := newTestResolver()
	scopeID := "p-test123"
	processor := NewCSSProcessor(config.LoaderLocalCSS, &config.Options{
		MinifyWhitespace:       true,
		MinifySyntax:           true,
		UnsupportedCSSFeatures: compat.Nesting,
	}, resolver)

	template := simpleParse(t, `<div class="form-section"><div class="form-heading"></div></div>`)
	css := `
		.form-section {
			gap: 25px;
			&:not(:last-of-type) {
				margin-bottom: 50px;
			}
			.form-heading {
				font-size: 1.25rem;
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
		}
	`

	diagnostics, err := processor.ProcessAndScope(ctx, &processAndScopeParams{
		template:      template,
		cssBlock:      &css,
		scopeID:       scopeID,
		sourcePath:    "test.css",
		startLocation: ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		fsReader:      fsReader,
	})
	require.NoError(t, err)
	assert.Empty(t, diagnostics)

	normalised := normaliseCSS(css)
	t.Logf("Scoped and normalised output: %s", normalised)

	assert.Contains(t, normalised, ".form-section", "Should contain .form-section")
	assert.Contains(t, normalised, ".form-heading", "Should contain .form-heading")
	assert.Contains(t, normalised, "@media", "Should contain @media")

	assert.NotContains(t, normalised, "&:not(:last-of-type)", "Should not contain unexpanded & syntax")
	assert.NotContains(t, normalised, "&:", "Should not contain any unexpanded & pseudo-selectors")
}
