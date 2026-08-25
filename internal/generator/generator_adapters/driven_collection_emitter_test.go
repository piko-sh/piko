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

package generator_adapters

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/wdk/safedisk"
)

type mockCollectionEncoder struct {
	encodeErr    error
	encodeResult []byte
}

func (m *mockCollectionEncoder) EncodeCollection(_ []collection_dto.ContentItem) ([]byte, error) {
	return m.encodeResult, m.encodeErr
}

func (m *mockCollectionEncoder) DecodeCollectionItem(_ []byte, _ string) ([]byte, []byte, []byte, error) {
	return nil, nil, nil, nil
}

type collectionWriteRecord struct {
	path string
	data []byte
}

func newCollectionTrackingFSWriter(writeErr error, writeErrOnCall int) (*generator_domain.MockFSWriter, *[]collectionWriteRecord) {
	var writes []collectionWriteRecord
	callCount := 0
	return &generator_domain.MockFSWriter{
		WriteFileFunc: func(_ context.Context, filePath string, data []byte) error {
			callCount++
			if writeErrOnCall > 0 && callCount == writeErrOnCall {
				return writeErr
			}
			if writeErrOnCall == 0 && writeErr != nil {
				return writeErr
			}
			writes = append(writes, collectionWriteRecord{path: filePath, data: data})
			return nil
		},
	}, &writes
}

func TestNewDrivenCollectionEmitter(t *testing.T) {
	t.Parallel()

	encoder := &mockCollectionEncoder{}
	fsWriter := &generator_domain.MockFSWriter{}
	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()

	emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

	require.NotNil(t, emitter)
}

func TestDrivenCollectionEmitter_EmitCollection(t *testing.T) {
	t.Parallel()

	items := []collection_dto.ContentItem{
		{ID: "1", Slug: "hello-world"},
	}

	t.Run("success writes binary and Go wrapper", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("binary-data")}
		fsWriter, writes := newCollectionTrackingFSWriter(nil, 0)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		packagePath, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.NoError(t, err)
		assert.Equal(t, "mymod/dist/collections/docs", packagePath)
		require.Len(t, *writes, 2)

		assert.Equal(t, "dist/collections/docs/data.bin", (*writes)[0].path)
		assert.Equal(t, []byte("binary-data"), (*writes)[0].data)

		assert.Equal(t, "dist/collections/docs/generated.go", (*writes)[1].path)
		goCode := string((*writes)[1].data)
		assert.Contains(t, goCode, "package docs")
		assert.Contains(t, goCode, "//go:embed data.bin")
		assert.Contains(t, goCode, `RegisterStaticCollectionBlob(context.Background(), "docs"`)
	})

	t.Run("MkdirAll error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.MkdirAllErr = errors.New("cannot create directory")

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create collection directory")
	})

	t.Run("encode error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeErr: errors.New("encode failed")}
		fsWriter := &generator_domain.MockFSWriter{}
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode collection")
	})

	t.Run("binary write error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter, _ := newCollectionTrackingFSWriter(errors.New("write failed"), 1)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write binary data")
	})

	t.Run("Go wrapper write error", func(t *testing.T) {
		t.Parallel()

		encoder := &mockCollectionEncoder{encodeResult: []byte("data")}
		fsWriter, _ := newCollectionTrackingFSWriter(errors.New("write failed"), 2)
		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()

		emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

		_, err := emitter.EmitCollection(context.Background(), "docs", items, "dist")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write Go wrapper")
	})
}

func TestCollectionPackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		collection   string
		expected     string
		expectRename bool
	}{
		{name: "plain name is kept", collection: "docs", expected: "docs", expectRename: false},
		{name: "underscored name is kept", collection: "blog_posts", expected: "blog_posts", expectRename: false},
		{name: "mixed case name is kept", collection: "BlogPosts", expected: "BlogPosts", expectRename: false},
		{name: "trailing digits are kept", collection: "posts2024", expected: "posts2024", expectRename: false},
		{name: "hyphen is replaced", collection: "blog-posts", expected: "blog_posts_" + goastutil.ShortHash("blog-posts"), expectRename: true},
		{name: "space is replaced", collection: "My Collection", expected: "my_collection_" + goastutil.ShortHash("My Collection"), expectRename: true},
		{name: "dot is replaced", collection: "blog.posts", expected: "blog_posts_" + goastutil.ShortHash("blog.posts"), expectRename: true},
		{name: "leading digit is prefixed", collection: "2024posts", expected: "p2024posts_" + goastutil.ShortHash("2024posts"), expectRename: true},
		{name: "keyword is suffixed", collection: "range", expected: "range__" + goastutil.ShortHash("range"), expectRename: true},
		{name: "predeclared name is suffixed", collection: "string", expected: "string__" + goastutil.ShortHash("string"), expectRename: true},
		{name: "traversal loses its separators", collection: "../x", expected: "x_" + goastutil.ShortHash("../x"), expectRename: true},
		{name: "main is renamed, since a package called main cannot be imported", collection: "main", expected: "main_" + goastutil.ShortHash("main"), expectRename: true},
		{name: "empty name falls back", collection: "", expected: goastutil.DefaultGoPackageName + "_" + goastutil.ShortHash(""), expectRename: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			packageName, renamed := collectionPackageName(tt.collection)

			assert.Equal(t, tt.expected, packageName)
			assert.Equal(t, tt.expectRename, renamed)
			assert.True(t, token.IsIdentifier(packageName), "a package clause must be a legal identifier")
		})
	}
}

func TestCollectionPackageNameKeepsFoldedNamesApart(t *testing.T) {
	t.Parallel()

	hyphenated, _ := collectionPackageName("blog-posts")
	dotted, _ := collectionPackageName("blog.posts")

	assert.NotEqual(t, hyphenated, dotted,
		"two collections whose names sanitise alike must not share one generated package")
}

func TestDrivenCollectionEmitter_EmitCollectionSanitisesPackageName(t *testing.T) {
	t.Parallel()

	items := []collection_dto.ContentItem{{ID: "1", Slug: "hello-world"}}
	suffix := "_" + goastutil.ShortHash("blog-posts")

	encoder := &mockCollectionEncoder{encodeResult: []byte("binary-data")}
	fsWriter, writes := newCollectionTrackingFSWriter(nil, 0)
	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()

	emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

	packagePath, err := emitter.EmitCollection(context.Background(), "blog-posts", items, "dist")

	require.NoError(t, err)
	assert.Equal(t, "mymod/dist/collections/blog_posts"+suffix, packagePath)
	require.Len(t, *writes, 2)
	assert.Equal(t, "dist/collections/blog_posts"+suffix+"/data.bin", (*writes)[0].path)
	assert.Equal(t, "dist/collections/blog_posts"+suffix+"/generated.go", (*writes)[1].path)

	goCode := string((*writes)[1].data)
	assert.Contains(t, goCode, "package blog_posts"+suffix)
	assert.Contains(t, goCode, `RegisterStaticCollectionBlob(context.Background(), "blog-posts"`,
		"the raw name stays the registration key so templates keep resolving")

	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "generated.go", goCode, parser.AllErrors)
	require.NoError(t, parseErr, "the generated wrapper must be parseable Go")
}

func TestDrivenCollectionEmitter_EmitCollectionQuotesRegistrationKey(t *testing.T) {
	t.Parallel()

	items := []collection_dto.ContentItem{{ID: "1", Slug: "hello-world"}}

	encoder := &mockCollectionEncoder{encodeResult: []byte("binary-data")}
	fsWriter, writes := newCollectionTrackingFSWriter(nil, 0)
	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()

	emitter := NewDrivenCollectionEmitter(encoder, fsWriter, sandbox, "mymod")

	_, err := emitter.EmitCollection(context.Background(), `say "hi"`, items, "dist")

	require.NoError(t, err)
	require.Len(t, *writes, 2)

	goCode := string((*writes)[1].data)
	fset := token.NewFileSet()
	_, parseErr := parser.ParseFile(fset, "generated.go", goCode, parser.AllErrors)
	require.NoError(t, parseErr, "a quote in the collection name must not close the registration literal")
}
