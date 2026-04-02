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

package markdown_domain

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_ast"
)

func TestNewMarkdownService(t *testing.T) {
	t.Run("CreatesServiceWithParser", func(t *testing.T) {
		parser := &MockMarkdownParser{}

		service := NewMarkdownService(parser, nil)

		assert.NotNil(t, service)
		assert.Implements(t, (*MarkdownService)(nil), service)
	})
}

func TestMarkdownService_Process(t *testing.T) {
	t.Run("ValidMarkdownWithMinimalFrontmatter", func(t *testing.T) {
		content := []byte("# Test")
		frontmatter := map[string]any{
			"title": "Test Post",
		}

		parser := &MockMarkdownParser{
			ParseFunc: func(ctx context.Context, c []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), frontmatter, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), content, "test.md")

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotNil(t, result.PageAST)
		assert.NotNil(t, result.Metadata)
		assert.Equal(t, "Test Post", result.Metadata.Title)
	})

	t.Run("ParserError", func(t *testing.T) {
		content := []byte("# Test")

		parser := &MockMarkdownParser{
			ParseFunc: func(ctx context.Context, c []byte) (*markdown_ast.Document, map[string]any, error) {
				return nil, nil, errors.New("parser error")
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), content, "test.md")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "markdown parsing failed")
		assert.Contains(t, err.Error(), "test.md")
	})

	t.Run("FrontmatterValidationError", func(t *testing.T) {
		content := []byte("# Test")
		frontmatter := map[string]any{
			"description": "Some description",
		}

		parser := &MockMarkdownParser{
			ParseFunc: func(ctx context.Context, c []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), frontmatter, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), content, "test.md")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "frontmatter processing failed")
		assert.Contains(t, err.Error(), "test.md")
	})

	t.Run("SetsReadingTimeField", func(t *testing.T) {
		content := []byte("Test content")
		frontmatter := map[string]any{
			"title": "Test Post",
		}

		parser := &MockMarkdownParser{
			ParseFunc: func(ctx context.Context, c []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), frontmatter, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), content, "test.md")

		require.NoError(t, err)
		assert.NotNil(t, result.Metadata)
		assert.GreaterOrEqual(t, result.Metadata.ReadingTime, 0)
	})

	t.Run("PreservesFrontmatterCustomFields", func(t *testing.T) {
		content := []byte("# Test")
		frontmatter := map[string]any{
			"title":        "Test Post",
			"custom_field": "custom_value",
			"author":       "John Doe",
		}

		parser := &MockMarkdownParser{
			ParseFunc: func(ctx context.Context, c []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), frontmatter, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), content, "test.md")

		require.NoError(t, err)
		assert.NotNil(t, result.Metadata.Frontmatter)
		assert.Contains(t, result.Metadata.Frontmatter, "custom_field")
		assert.Equal(t, "custom_value", result.Metadata.Frontmatter["custom_field"])
		assert.Contains(t, result.Metadata.Frontmatter, "author")
		assert.Equal(t, "John Doe", result.Metadata.Frontmatter["author"])
	})
}

func TestCalculateReadingTime(t *testing.T) {
	tests := []struct {
		name         string
		wordCount    int
		expectedTime int
	}{
		{
			name:         "ZeroWords",
			wordCount:    0,
			expectedTime: 0,
		},
		{
			name:         "SingleWord",
			wordCount:    1,
			expectedTime: 1,
		},
		{
			name:         "VeryShortContent",
			wordCount:    10,
			expectedTime: 1,
		},
		{
			name:         "ExactlyOneMinute",
			wordCount:    225,
			expectedTime: 1,
		},
		{
			name:         "SlightlyOverOneMinute",
			wordCount:    226,
			expectedTime: 2,
		},
		{
			name:         "TwoMinutes",
			wordCount:    450,
			expectedTime: 2,
		},
		{
			name:         "LongArticle",
			wordCount:    2250,
			expectedTime: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateReadingTime(tt.wordCount)
			assert.Equal(t, tt.expectedTime, result)
		})
	}
}

func TestProcess_PropagatesAllFrontmatterFields(t *testing.T) {
	t.Parallel()

	t.Run("DraftFieldPropagated", func(t *testing.T) {
		t.Parallel()

		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title": "Test",
					"draft": true,
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.True(t, result.Metadata.Draft)
	})

	t.Run("DraftFieldFalseByDefault", func(t *testing.T) {
		t.Parallel()

		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title": "Test",
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.False(t, result.Metadata.Draft)
	})

	t.Run("DescriptionPropagated", func(t *testing.T) {
		t.Parallel()

		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title":       "Test",
					"description": "A test description",
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.Equal(t, "A test description", result.Metadata.Description)
	})

	t.Run("TagsPropagated", func(t *testing.T) {
		t.Parallel()

		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title": "Test",
					"tags":  []any{"go", "testing"},
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.Equal(t, []string{"go", "testing"}, result.Metadata.Tags)
	})

	t.Run("PublishDatePropagated", func(t *testing.T) {
		t.Parallel()

		publishDate := time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC)
		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title": "Test",
					"date":  publishDate,
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.Equal(t, publishDate, result.Metadata.PublishDate)
	})

	t.Run("AllFieldsTogether", func(t *testing.T) {
		t.Parallel()

		publishDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title":       "Full Post",
					"description": "Complete description",
					"draft":       true,
					"date":        publishDate,
					"tags":        []any{"complete", "test"},
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.Equal(t, "Full Post", result.Metadata.Title)
		assert.Equal(t, "Complete description", result.Metadata.Description)
		assert.True(t, result.Metadata.Draft)
		assert.Equal(t, publishDate, result.Metadata.PublishDate)
		assert.Equal(t, []string{"complete", "test"}, result.Metadata.Tags)
	})

	t.Run("EmptyOptionalFields", func(t *testing.T) {
		t.Parallel()

		parser := &MockMarkdownParser{
			ParseFunc: func(_ context.Context, _ []byte) (*markdown_ast.Document, map[string]any, error) {
				return markdown_ast.NewDocument(), map[string]any{
					"title": "Minimal Post",
				}, nil
			},
		}

		service := NewMarkdownService(parser, nil)
		result, err := service.Process(context.Background(), []byte("# Test"), "test.md")

		require.NoError(t, err)
		assert.Equal(t, "", result.Metadata.Description)
		assert.Nil(t, result.Metadata.Tags)
		assert.True(t, result.Metadata.PublishDate.IsZero())
		assert.False(t, result.Metadata.Draft)
	})
}
