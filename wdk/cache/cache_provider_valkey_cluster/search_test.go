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

package cache_provider_valkey_cluster

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

type searchDoc struct {
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}

type ftHit struct {
	key    string
	fields []string
}

type ftFakeConfig struct {
	search       func(peer *server.Peer)
	createError  string
	dropError    string
	jsonSetError string
	jsonGetReply string
	indexExists  bool
}

type ftFake struct {
	db       *miniredis.Miniredis
	commands map[string][][]string
	config   ftFakeConfig
	mu       sync.Mutex
	created  bool
}

func (f *ftFake) record(name string, arguments []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands[name] = append(f.commands[name], slices.Clone(arguments))
}

func (f *ftFake) calls(name string) [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.commands[name])
}

func (f *ftFake) lastSearch(t *testing.T) []string {
	t.Helper()

	searches := f.calls("FT.SEARCH")
	require.NotEmpty(t, searches, "no FT.SEARCH was sent")
	return searches[len(searches)-1]
}

func registerFTFake(t *testing.T, db *miniredis.Miniredis, indexName string, config ftFakeConfig) *ftFake {
	t.Helper()

	fake := &ftFake{db: db, commands: map[string][][]string{}, config: config, created: config.indexExists}
	srv := db.Server()
	require.NoError(t, srv.Register("FT._LIST", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		fake.mu.Lock()
		created := fake.created
		fake.mu.Unlock()
		if !created {
			peer.WriteStrings([]string{"other"})
			return
		}
		peer.WriteStrings([]string{"other", indexName})
	}))
	require.NoError(t, srv.Register("FT.CREATE", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		if config.createError != "" {
			peer.WriteError(config.createError)
			return
		}
		fake.mu.Lock()
		fake.created = true
		fake.mu.Unlock()
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("FT.DROPINDEX", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		if config.dropError != "" {
			peer.WriteError(config.dropError)
			return
		}
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("FT.SEARCH", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		if config.search == nil {
			writeSearchReply(peer, 0)
			return
		}
		config.search(peer)
	}))
	require.NoError(t, srv.Register("JSON.SET", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		if config.jsonSetError != "" {
			peer.WriteError(config.jsonSetError)
			return
		}
		if err := db.Set(arguments[0], arguments[2]); err != nil {
			peer.WriteError(err.Error())
			return
		}
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("JSON.GET", func(peer *server.Peer, cmd string, arguments []string) {
		fake.record(cmd, arguments)
		if config.jsonGetReply != "" {
			peer.WriteBulk(config.jsonGetReply)
			return
		}
		value, err := db.Get(arguments[0])
		if err != nil {
			peer.WriteNull()
			return
		}
		peer.WriteBulk("[" + value + "]")
	}))
	return fake
}

func writeSearchReply(peer *server.Peer, total int, hits ...ftHit) {
	peer.WriteLen(1 + 2*len(hits))
	peer.WriteInt(total)
	for _, hit := range hits {
		peer.WriteBulk(hit.key)
		peer.WriteStrings(hit.fields)
	}
}

func documentHit(key, document string) ftHit {
	return ftHit{key: key, fields: []string{valkeyJSONPath, document}}
}

func searchSchema() *cache.SearchSchema {
	return cache.NewSearchSchema(
		cache.TextField("name"),
		cache.FieldSchema{Name: "category", Type: cache.FieldTypeTag, Sortable: true},
		cache.SortableNumericField("price"),
		cache.GeoField("location"),
		cache.VectorField("embedding", 2),
	)
}

func newSearchAdapter(t *testing.T, schema *cache.SearchSchema, config ftFakeConfig) (*ValkeyClusterAdapter[string, searchDoc], *ftFake) {
	t.Helper()

	db := miniredis.RunT(t)
	encoder, ok := cache_encoder_json.New[searchDoc]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewValkeyClusterProvider(Config{
		InitAddress: []string{db.Addr()},
		Registry:    cache.NewEncodingRegistry(encoder),
		DefaultTTL:  time.Hour,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, provider.Close()) })

	created, err := createNamespaceGeneric(provider, "docs", cache.Options[string, searchDoc]{SearchSchema: schema})
	require.NoError(t, err)
	adapter, ok := created.(*ValkeyClusterAdapter[string, searchDoc])
	require.True(t, ok)
	return adapter, registerFTFake(t, db, adapter.indexName, config)
}

func argumentsAfter(t *testing.T, arguments []string, keyword string, count int) []string {
	t.Helper()

	start := slices.Index(arguments, keyword)
	require.NotEqual(t, -1, start, "%s missing from %v", keyword, arguments)
	require.LessOrEqual(t, start+1+count, len(arguments))
	return arguments[start+1 : start+1+count]
}

func TestEscapeTagValue_EscapesAllSpecialCharacters(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   any
		want string
	}{
		{"plain alphanumerics untouched", "hello123", "hello123"},
		{"comma escaped", "a,b", "a\\,b"},
		{"colons escaped", "ns:value", "ns\\:value"},
		{"dot escaped", "a.b", "a\\.b"},
		{"space escaped", "hello world", "hello\\ world"},
		{"hyphen escaped", "x-y", "x\\-y"},
		{"plus escaped", "1+2", "1\\+2"},
		{"complex value escapes everything", "(foo!)*", "\\(foo\\!\\)\\*"},
		{"integer formatted", 42, "42"},
		{"float formatted", 3.14, "3\\.14"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, escapeTagValue(tc.in))
		})
	}
}

func TestIsUnknownIndexError_DetectsUnknownPhrase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		err  error
		want bool
	}{
		{errors.New("Unknown index name"), true},
		{errors.New("ERR Unknown command"), true},
		{errors.New("connection refused"), false},
		{errors.New(""), false},
	}

	for _, tc := range cases {
		require.Equalf(t, tc.want, isUnknownIndexError(tc.err), "error: %q", tc.err.Error())
	}
}

func TestExtractJSONFromDocData_ReturnsEmptyForNilInput(t *testing.T) {
	t.Parallel()

	require.Empty(t, extractJSONFromDocData(nil))
}

func TestValkeyClusterSearchFilters(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{
		search: func(peer *server.Peer) {
			writeSearchReply(peer, 5,
				documentHit("docs:a", `{"name":"a","price":1.5}`),
				documentHit("docs:b", `{"name":"b"}`),
			)
		},
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{
		Filters:   []cache.Filter{cache.Gt("price", 1)},
		Offset:    2,
		Limit:     4,
		SortBy:    "price",
		SortOrder: cache.SortDesc,
	})
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, []string{adapter.indexName, "@price:[(1 +inf]"}, arguments[:2])
	assert.Equal(t, []string{"2", "4"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.Equal(t, []string{"price", "DESC"}, argumentsAfter(t, arguments, "SORTBY", 2))
	assert.Equal(t, []string{"1", valkeyJSONPath}, argumentsAfter(t, arguments, "RETURN", 2))
	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, 2, result.Offset)
	assert.Equal(t, 4, result.Limit)
	assert.Equal(t, []cache.SearchHit[string, searchDoc]{
		{Key: "a", Value: searchDoc{Name: "a", Price: 1.5}},
		{Key: "b", Value: searchDoc{Name: "b"}},
	}, result.Items)
	assert.True(t, adapter.SupportsSearch())
	assert.Same(t, adapter.schema, adapter.GetSchema())
}

func TestValkeyClusterCreateIndexKeepsOnlyTagAndNumericFields(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{})

	_, err := adapter.Search(t.Context(), "", nil)
	require.NoError(t, err)

	creates := fake.calls("FT.CREATE")
	require.Len(t, creates, 1)
	assert.Equal(t, []string{
		adapter.indexName, "ON", "JSON", "PREFIX", "1", "docs:", "SCHEMA",
		"$.category", "AS", "category", "TAG", "SORTABLE",
		"$.price", "AS", "price", "NUMERIC", "SORTABLE",
	}, creates[0])

	_, err = adapter.Search(t.Context(), "", nil)
	require.NoError(t, err)
	assert.Len(t, fake.calls("FT._LIST"), 1)
}

func TestValkeyClusterSearchReusesAnExistingIndex(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{indexExists: true})

	result, err := adapter.Search(t.Context(), "", nil)
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, "*", arguments[1])
	assert.Equal(t, []string{"0", "10"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.NotContains(t, arguments, "SORTBY")
	assert.Empty(t, result.Items)
	assert.Empty(t, fake.calls("FT.CREATE"))

	_, err = adapter.Search(t.Context(), "", &cache.SearchOptions{SortBy: "category"})
	require.NoError(t, err)
	assert.Equal(t, []string{"category", "ASC"}, argumentsAfter(t, fake.lastSearch(t), "SORTBY", 2))
}

func TestValkeyClusterSearchFilterClauses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		want   string
		filter cache.Filter
	}{
		{name: "equal escapes tag punctuation", filter: cache.Eq("category", "a b.c"), want: `@category:{a\ b\.c}`},
		{name: "not equal", filter: cache.Ne("category", "red"), want: `-@category:{red}`},
		{name: "greater or equal", filter: cache.Ge("price", 2), want: `@price:[2 +inf]`},
		{name: "less than", filter: cache.Lt("price", 3), want: `@price:[-inf (3]`},
		{name: "less or equal", filter: cache.Le("price", 4), want: `@price:[-inf 4]`},
		{name: "between", filter: cache.Between("price", 1, 9), want: `@price:[1 9]`},
		{name: "in", filter: cache.In("category", "red", "blue"), want: `@category:{red|blue}`},
		{name: "prefix", filter: cache.Prefix("category", "ab"), want: `@category:ab*`},
		{name: "in without values is dropped", filter: cache.Filter{Field: "category", Operation: cache.FilterOpIn}, want: "*"},
		{name: "between with one bound is dropped", filter: cache.Filter{Field: "price", Operation: cache.FilterOpBetween, Values: []any{1}}, want: "*"},
	}

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{})
	for _, testCase := range testCases {
		_, err := adapter.Query(t.Context(), &cache.QueryOptions{Filters: []cache.Filter{testCase.filter}})
		require.NoError(t, err, testCase.name)
		assert.Equal(t, testCase.want, fake.lastSearch(t)[1], testCase.name)
	}
}

func TestValkeyClusterQuery(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{
		search: func(peer *server.Peer) { writeSearchReply(peer, 1, documentHit("docs:a", `{"name":"a"}`)) },
	})

	result, err := adapter.Query(t.Context(), &cache.QueryOptions{
		Filters:   []cache.Filter{cache.Eq("category", "red")},
		Offset:    3,
		SortBy:    "category",
		SortOrder: cache.SortAsc,
	})
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, []string{"3", "10"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 3, result.Offset)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "a", result.Items[0].Key)

	result, err = adapter.Query(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultSearchResultLimit, result.Limit)
}

func TestValkeyClusterSearchDropsMalformedHits(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{
		search: func(peer *server.Peer) {
			peer.WriteLen(14)
			peer.WriteInt(6)
			peer.WriteLen(0)
			peer.WriteStrings([]string{valkeyJSONPath, `{}`})
			peer.WriteBulk("other:a")
			peer.WriteStrings([]string{valkeyJSONPath, `{}`})
			peer.WriteBulk("docs:b")
			peer.WriteBulk("not an array")
			peer.WriteBulk("docs:c")
			peer.WriteStrings([]string{"name", "c", "price", "2"})
			peer.WriteBulk("docs:d")
			peer.WriteStrings([]string{valkeyJSONPath, `{"name":`})
			peer.WriteBulk("docs:e")
			peer.WriteStrings([]string{valkeyJSONPath, `{"name":"e"}`})
			peer.WriteBulk("docs:f")
		},
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{})
	require.NoError(t, err)

	require.Len(t, result.Items, 1)
	assert.Equal(t, "e", result.Items[0].Key)
}

func TestValkeyClusterSearchReplyErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		reply   func(peer *server.Peer)
		name    string
		wantErr string
	}{
		{name: "the server rejects the query", reply: func(peer *server.Peer) { peer.WriteError("Syntax error") }, wantErr: "FT.SEARCH failed"},
		{name: "total is not an integer", reply: func(peer *server.Peer) {
			peer.WriteLen(1)
			peer.WriteBulk("one")
		}, wantErr: "total count not int64"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{search: testCase.reply})

			_, err := adapter.Search(t.Context(), "", nil)
			require.ErrorContains(t, err, testCase.wantErr)

			_, err = adapter.Query(t.Context(), nil)
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestValkeyClusterSearchEmptyReply(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{
		search: func(peer *server.Peer) { peer.WriteLen(0) },
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Offset: 1, Limit: 2})
	require.NoError(t, err)

	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 1, result.Offset)
	assert.Equal(t, 2, result.Limit)
}

func TestValkeyClusterSearchIndexFailures(t *testing.T) {
	t.Parallel()

	t.Run("the server refuses the index", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{createError: "index refused"})

		_, err := adapter.Search(t.Context(), "", nil)
		require.ErrorContains(t, err, "index refused")

		_, err = adapter.Query(t.Context(), nil)
		require.ErrorContains(t, err, "index refused")
	})

	t.Run("no field can be indexed", func(t *testing.T) {
		t.Parallel()

		adapter, fake := newSearchAdapter(t, cache.NewSearchSchema(cache.TextField("name")), ftFakeConfig{})

		_, err := adapter.Search(t.Context(), "", nil)
		require.ErrorContains(t, err, "no supported fields")
		assert.Empty(t, fake.calls("FT.CREATE"))
	})
}

func TestValkeyClusterSearchRejectsFullTextQueries(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{})

	_, err := adapter.Search(t.Context(), "hello", nil)

	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	assert.Empty(t, fake.calls("FT.SEARCH"))
}

func TestValkeyClusterSearchWithoutSchema(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, nil, ftFakeConfig{})

	_, err := adapter.Search(t.Context(), "", nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)

	_, err = adapter.Query(t.Context(), nil)
	require.ErrorIs(t, err, cache.ErrSearchNotSupported)
	assert.False(t, adapter.SupportsSearch())
	assert.Nil(t, adapter.GetSchema())
	require.ErrorIs(t, adapter.ensureIndexExists(t.Context()), cache.ErrSearchNotSupported)
}

func TestValkeyClusterJSONStorageRoundTrip(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{})
	ctx := t.Context()

	require.NoError(t, adapter.Set(ctx, "a", searchDoc{Name: "a", Price: 2}, "featured"))
	require.NoError(t, adapter.SetWithTTL(ctx, "b", searchDoc{Name: "b"}, time.Minute))
	require.NoError(t, adapter.SetWithTTL(ctx, "c", searchDoc{Name: "c"}, cache.NoExpiry, "featured"))

	value, found, err := adapter.GetIfPresent(ctx, "a")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, searchDoc{Name: "a", Price: 2}, value)

	_, found, err = adapter.GetIfPresent(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, found)

	assert.Equal(t, time.Hour, fake.db.TTL("docs:a"))
	assert.Equal(t, time.Minute, fake.db.TTL("docs:b"))
	assert.Zero(t, fake.db.TTL("docs:c"))
	require.Len(t, fake.calls("JSON.SET"), 3)
	assert.Equal(t, []string{"docs:a", valkeyJSONPath, `{"name":"a","category":"","price":2}`}, fake.calls("JSON.SET")[0])

	count, err := adapter.InvalidateByTags(ctx, "featured")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	assert.False(t, fake.db.Exists("docs:a"))

	require.NoError(t, adapter.InvalidateAll(ctx))
	assert.Len(t, fake.calls("FT.DROPINDEX"), 1)
	assert.False(t, adapter.indexCreated)
}

func TestValkeyClusterJSONStorageFailures(t *testing.T) {
	t.Parallel()

	t.Run("an unreadable JSON.GET reply is a miss", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{jsonGetReply: "not json"})

		_, found, err := adapter.GetIfPresent(t.Context(), "a")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("an empty JSON.GET array is a miss", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, searchSchema(), ftFakeConfig{jsonGetReply: "[]"})

		_, found, err := adapter.GetIfPresent(t.Context(), "a")
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("a failed index write is tolerated", func(t *testing.T) {
		t.Parallel()

		adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{jsonSetError: "read only"})

		require.NoError(t, adapter.Set(t.Context(), "a", searchDoc{Name: "a"}))

		assert.Len(t, fake.calls("JSON.SET"), 1)
		assert.False(t, fake.db.Exists("docs:a"))
	})

	t.Run("a failed index creation skips the write", func(t *testing.T) {
		t.Parallel()

		adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{createError: "index refused"})

		require.NoError(t, adapter.SetWithTTL(t.Context(), "a", searchDoc{Name: "a"}, time.Minute))

		assert.Empty(t, fake.calls("JSON.SET"))
	})

	t.Run("a failed drop is tolerated", func(t *testing.T) {
		t.Parallel()

		adapter, fake := newSearchAdapter(t, searchSchema(), ftFakeConfig{indexExists: true, dropError: "busy"})
		require.NoError(t, adapter.ensureIndexExists(t.Context()))

		adapter.dropIndex(t.Context())

		assert.Len(t, fake.calls("FT.DROPINDEX"), 1)
		assert.False(t, adapter.indexCreated)
	})
}
