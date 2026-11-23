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

package cache_provider_redis

import (
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/cache"
	"piko.sh/piko/wdk/cache/cache_encoder_json"
)

type searchDoc struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type ftHit struct {
	key    string
	fields []string
}

type ftFakeConfig struct {
	search       func(peer *server.Peer)
	jsonGet      func(peer *server.Peer, key string)
	createError  string
	dropError    string
	jsonSetError string
	indexExists  bool
}

type ftFake struct {
	db       *miniredis.Miniredis
	commands map[string][][]string
	config   ftFakeConfig
	mu       sync.Mutex
	created  bool
}

func (f *ftFake) record(name string, args []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commands[name] = append(f.commands[name], slices.Clone(args))
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

func registerFTFake(t *testing.T, db *miniredis.Miniredis, config ftFakeConfig) *ftFake {
	t.Helper()

	fake := &ftFake{db: db, commands: map[string][][]string{}, config: config, created: config.indexExists}
	srv := db.Server()
	require.NoError(t, srv.Register("FT.INFO", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		fake.mu.Lock()
		created := fake.created
		fake.mu.Unlock()
		if !created {
			peer.WriteError("Unknown index name")
			return
		}
		peer.WriteLen(0)
	}))
	require.NoError(t, srv.Register("FT.CREATE", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		if config.createError != "" {
			peer.WriteError(config.createError)
			return
		}
		fake.mu.Lock()
		fake.created = true
		fake.mu.Unlock()
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("FT.DROPINDEX", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		if config.dropError != "" {
			peer.WriteError(config.dropError)
			return
		}
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("FT.SEARCH", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		if config.search == nil {
			writeSearchReply(peer, 0)
			return
		}
		config.search(peer)
	}))
	require.NoError(t, srv.Register("JSON.SET", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		if config.jsonSetError != "" {
			peer.WriteError(config.jsonSetError)
			return
		}
		if err := db.Set(args[0], args[2]); err != nil {
			peer.WriteError(err.Error())
			return
		}
		peer.WriteOK()
	}))
	require.NoError(t, srv.Register("JSON.GET", func(peer *server.Peer, cmd string, args []string) {
		fake.record(cmd, args)
		if config.jsonGet != nil {
			config.jsonGet(peer, args[0])
			return
		}
		value, err := db.Get(args[0])
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
	return ftHit{key: key, fields: []string{redisJSONPath, document}}
}

func vectorHit(key, document, distance string) ftHit {
	return ftHit{key: key, fields: []string{redisJSONPath, document, "__vec_score", distance}}
}

func vectorSchema() *cache.SearchSchema {
	return cache.NewSearchSchema(
		cache.FieldSchema{Name: "name", Type: cache.FieldTypeText},
		cache.FieldSchema{Name: "vec", Type: cache.FieldTypeVector, Dimension: 2},
	)
}

func newSearchAdapter(t *testing.T, schema *cache.SearchSchema, config ftFakeConfig) (*RedisAdapter[string, searchDoc], *ftFake) {
	t.Helper()

	db := miniredis.RunT(t)
	fake := registerFTFake(t, db, config)
	encoder, ok := cache_encoder_json.New[searchDoc]().(cache.AnyEncoder)
	require.True(t, ok)
	provider, err := NewRedisProvider(Config{Address: db.Addr(), Registry: cache.NewEncodingRegistry(encoder), DefaultTTL: time.Hour})
	require.NoError(t, err)
	t.Cleanup(func() { _ = provider.Close() })

	created, err := createNamespaceGeneric(provider, "docs", cache.Options[string, searchDoc]{SearchSchema: schema})
	require.NoError(t, err)
	adapter, ok := created.(*RedisAdapter[string, searchDoc])
	require.True(t, ok)
	return adapter, fake
}

func argumentsAfter(t *testing.T, arguments []string, keyword string, count int) []string {
	t.Helper()

	start := slices.Index(arguments, keyword)
	require.NotEqual(t, -1, start, "%s missing from %v", keyword, arguments)
	require.LessOrEqual(t, start+1+count, len(arguments))
	return arguments[start+1 : start+1+count]
}

func minScore(score float32) *float32 {
	return new(score)
}

func TestRedisVectorSearchWindow(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		options   cache.SearchOptions
		wantKNN   string
		wantLimit []string
		wantPage  int
	}{
		{
			name:      "NoLimit returns every candidate",
			options:   cache.SearchOptions{TopK: 50, Limit: cache.NoLimit},
			wantKNN:   "(*)=>[KNN 50 @vec $vec]",
			wantLimit: []string{"0", "50"},
			wantPage:  50,
		},
		{
			name:      "a zero limit uses the default page size",
			options:   cache.SearchOptions{TopK: 50, Limit: 0},
			wantKNN:   "(*)=>[KNN 50 @vec $vec]",
			wantLimit: []string{"0", "10"},
			wantPage:  10,
		},
		{
			name:      "no TopK ranks enough candidates to fill the page after the offset",
			options:   cache.SearchOptions{Offset: 10, Limit: 5},
			wantKNN:   "(*)=>[KNN 15 @vec $vec]",
			wantLimit: []string{"10", "5"},
			wantPage:  5,
		},
		{
			name:      "a page larger than TopK is capped",
			options:   cache.SearchOptions{TopK: 3, Limit: 20},
			wantKNN:   "(*)=>[KNN 3 @vec $vec]",
			wantLimit: []string{"0", "3"},
			wantPage:  3,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{})
			options := testCase.options
			options.Vector = []float32{1, 0}

			result, err := adapter.Search(t.Context(), "", &options)
			require.NoError(t, err)

			arguments := fake.lastSearch(t)
			assert.Contains(t, arguments, testCase.wantKNN)
			assert.Equal(t, testCase.wantLimit, argumentsAfter(t, arguments, "LIMIT", 2))
			assert.Equal(t, []string{"__vec_score"}, argumentsAfter(t, arguments, "SORTBY", 1))
			assert.Equal(t, []string{"2"}, argumentsAfter(t, arguments, "DIALECT", 1))
			assert.Equal(t, testCase.wantPage, result.Limit)
			assert.Equal(t, 0, result.Candidates)
			assert.Empty(t, result.Items)
		})
	}
}

func TestRedisVectorSearchCandidates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		hits           []ftHit
		minScore       *float32
		wantKeys       []string
		total          int
		offset         int
		wantCandidates int
	}{
		{
			name: "no hit below MinScore keeps the server total",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.1"),
				vectorHit("docs:b", `{"name":"b"}`, "0.2"),
				vectorHit("docs:c", `{"name":"c"}`, "0.3"),
			},
			minScore:       minScore(0.5),
			total:          7,
			wantKeys:       []string{"a", "b", "c"},
			wantCandidates: 7,
		},
		{
			name: "no MinScore keeps the server total",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.9"),
			},
			total:          4,
			wantKeys:       []string{"a"},
			wantCandidates: 4,
		},
		{
			name: "the first hit below MinScore ends the candidates",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.1"),
				vectorHit("docs:b", `{"name":"b"}`, "0.2"),
				vectorHit("docs:c", `{"name":"c"}`, "0.7"),
				vectorHit("docs:d", `{"name":"d"}`, "0.8"),
			},
			minScore:       minScore(0.5),
			total:          9,
			wantKeys:       []string{"a", "b"},
			wantCandidates: 2,
		},
		{
			name: "the cut counts the offset",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.1"),
				vectorHit("docs:b", `{"name":"b"}`, "0.9"),
			},
			minScore:       minScore(0.5),
			total:          40,
			offset:         20,
			wantKeys:       []string{"a"},
			wantCandidates: 21,
		},
		{
			name: "the cut never exceeds the server total",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.1"),
				vectorHit("docs:b", `{"name":"b"}`, "0.9"),
			},
			minScore:       minScore(0.5),
			total:          3,
			offset:         20,
			wantKeys:       []string{"a"},
			wantCandidates: 3,
		},
		{
			name: "undecodable hits are dropped but still counted",
			hits: []ftHit{
				vectorHit("docs:a", `{"name":"a"}`, "0.1"),
				vectorHit("docs:b", `not json`, "0.15"),
				vectorHit("other:c", `{"name":"c"}`, "0.2"),
				{key: "docs:d", fields: []string{"__vec_score", "0.2"}},
				{key: "docs:e", fields: []string{redisJSONPath}},
				vectorHit("docs:f", `{"name":"f"}`, "0.9"),
			},
			minScore:       minScore(0.5),
			total:          12,
			wantKeys:       []string{"a"},
			wantCandidates: 5,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
				search: func(peer *server.Peer) { writeSearchReply(peer, testCase.total, testCase.hits...) },
			})

			result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{
				Vector:   []float32{1, 0},
				TopK:     50,
				Offset:   testCase.offset,
				Limit:    cache.NoLimit,
				MinScore: testCase.minScore,
			})
			require.NoError(t, err)

			keys := make([]string, 0, len(result.Items))
			for _, item := range result.Items {
				keys = append(keys, item.Key)
			}
			assert.Equal(t, testCase.wantKeys, keys)
			assert.Equal(t, testCase.wantCandidates, result.Candidates)
			assert.Equal(t, int64(testCase.total), result.Total)
			assert.Equal(t, testCase.offset, result.Offset)
		})
	}
}

func TestRedisVectorSearchScoresHits(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) {
			writeSearchReply(peer, 2,
				vectorHit("docs:a", `{"name":"a","price":2}`, "0.25"),
				ftHit{key: "docs:b", fields: []string{redisJSONPath, `{"name":"b"}`, "__vec_score", "not a number"}},
			)
		},
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}})
	require.NoError(t, err)

	require.Len(t, result.Items, 2)
	assert.Equal(t, searchDoc{Name: "a", Price: 2}, result.Items[0].Value)
	assert.InDelta(t, 0.75, result.Items[0].Score, 1e-9)
	assert.InDelta(t, 0.0, result.Items[1].Score, 1e-9)
}

func TestRedisVectorSearchEmptyReply(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) { peer.WriteLen(0) },
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}, TopK: 25, Offset: 4, Limit: cache.NoLimit})
	require.NoError(t, err)

	assert.NotNil(t, result.Items)
	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 0, result.Candidates)
	assert.Equal(t, 25, result.Limit)
	assert.Equal(t, 4, result.Offset)
}

func TestRedisVectorSearchErrors(t *testing.T) {
	t.Parallel()

	t.Run("no vector field", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, cache.NewSearchSchema(cache.TextField("name")), ftFakeConfig{})

		_, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}})
		require.ErrorContains(t, err, "no vector field")
	})

	t.Run("total is not an integer", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
			search: func(peer *server.Peer) {
				peer.WriteLen(1)
				peer.WriteBulk("seven")
			},
		})

		_, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}})
		require.ErrorContains(t, err, "total count not int64")
	})

	t.Run("the server rejects the query", func(t *testing.T) {
		t.Parallel()

		adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
			search: func(peer *server.Peer) { peer.WriteError("Syntax error") },
		})

		_, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Vector: []float32{1, 0}})
		require.ErrorContains(t, err, "FT.SEARCH vector query failed")
	})
}

func TestRedisVectorSearchUsesTheNamedFieldAndQuery(t *testing.T) {
	t.Parallel()

	schema := cache.NewSearchSchema(
		cache.FieldSchema{Name: "name", Type: cache.FieldTypeText},
		cache.FieldSchema{Name: "category", Type: cache.FieldTypeTag},
		cache.FieldSchema{Name: "vec", Type: cache.FieldTypeVector, Dimension: 2},
		cache.FieldSchema{Name: "other", Type: cache.FieldTypeVector, Dimension: 2},
	)
	adapter, fake := newSearchAdapter(t, schema, ftFakeConfig{})

	_, err := adapter.Search(t.Context(), "shoes", &cache.SearchOptions{
		Vector:      []float32{1, 0},
		VectorField: "other",
		Filters:     []cache.Filter{cache.Eq("category", "red")},
		Limit:       3,
	})
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, "(shoes @category:{red})=>[KNN 3 @other $vec]", arguments[1])
	assert.Equal(t, []string{"__other_score"}, argumentsAfter(t, arguments, "SORTBY", 1))
	assert.Equal(t, []string{"2", redisJSONPath, "__other_score"}, argumentsAfter(t, arguments, "RETURN", 3))
	assert.Equal(t, string(vectorToBlob([]float32{1, 0})), argumentsAfter(t, arguments, "vec", 1)[0])
}

func TestRedisQueryWithVector(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		options        cache.QueryOptions
		wantKNN        string
		wantLimit      []string
		wantCandidates int
		wantPage       int
	}{
		{
			name:           "NoLimit returns every candidate",
			options:        cache.QueryOptions{TopK: 50, Limit: cache.NoLimit, MinScore: minScore(0.5)},
			wantKNN:        "(*)=>[KNN 50 @vec $vec]",
			wantLimit:      []string{"0", "50"},
			wantCandidates: 1,
			wantPage:       50,
		},
		{
			name:           "a zero limit uses the default page size",
			options:        cache.QueryOptions{Limit: 0},
			wantKNN:        "(*)=>[KNN 10 @vec $vec]",
			wantLimit:      []string{"0", "10"},
			wantCandidates: 30,
			wantPage:       10,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
				search: func(peer *server.Peer) {
					writeSearchReply(peer, 30,
						vectorHit("docs:a", `{"name":"a"}`, "0.1"),
						vectorHit("docs:b", `{"name":"b"}`, "0.8"),
					)
				},
			})
			options := testCase.options
			options.Vector = []float32{0, 1}

			result, err := adapter.Query(t.Context(), &options)
			require.NoError(t, err)

			arguments := fake.lastSearch(t)
			assert.Contains(t, arguments, testCase.wantKNN)
			assert.Equal(t, testCase.wantLimit, argumentsAfter(t, arguments, "LIMIT", 2))
			assert.Equal(t, testCase.wantCandidates, result.Candidates)
			assert.Equal(t, testCase.wantPage, result.Limit)
		})
	}
}

func TestRedisSearchText(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) {
			writeSearchReply(peer, 5,
				documentHit("docs:a", `{"name":"a","price":1.5}`),
				documentHit("docs:b", `{"name":"b"}`),
			)
		},
	})

	result, err := adapter.Search(t.Context(), "hello", &cache.SearchOptions{
		Filters:   []cache.Filter{cache.Gt("price", 1)},
		Offset:    2,
		Limit:     4,
		SortBy:    "price",
		SortOrder: cache.SortDesc,
	})
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, []string{adapter.indexName, "hello @price:[(1 +inf]"}, arguments[:2])
	assert.Equal(t, []string{"2", "4"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.Equal(t, []string{"price", "DESC"}, argumentsAfter(t, arguments, "SORTBY", 2))
	assert.Equal(t, []string{"1", redisJSONPath}, argumentsAfter(t, arguments, "RETURN", 2))
	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, 2, result.Offset)
	assert.Equal(t, 4, result.Limit)
	assert.Equal(t, []cache.SearchHit[string, searchDoc]{
		{Key: "a", Value: searchDoc{Name: "a", Price: 1.5}},
		{Key: "b", Value: searchDoc{Name: "b"}},
	}, result.Items)
}

func TestRedisSearchDefaults(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{})

	result, err := adapter.Search(t.Context(), "", nil)
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, "*", arguments[1])
	assert.Equal(t, []string{"0", "10"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.NotContains(t, arguments, "SORTBY")
	assert.Equal(t, DefaultSearchResultLimit, result.Limit)

	_, err = adapter.Search(t.Context(), "", &cache.SearchOptions{SortBy: "name"})
	require.NoError(t, err)
	assert.Equal(t, []string{"name", "ASC"}, argumentsAfter(t, fake.lastSearch(t), "SORTBY", 2))
}

func TestRedisSearchDropsMalformedHits(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) {
			peer.WriteLen(14)
			peer.WriteInt(6)
			peer.WriteInt(1)
			peer.WriteStrings([]string{redisJSONPath, `{}`})
			peer.WriteBulk("other:a")
			peer.WriteStrings([]string{redisJSONPath, `{}`})
			peer.WriteBulk("docs:b")
			peer.WriteBulk("not an array")
			peer.WriteBulk("docs:c")
			peer.WriteStrings([]string{"name", "c", "price", "2"})
			peer.WriteBulk("docs:d")
			peer.WriteStrings([]string{redisJSONPath, `{"name":`})
			peer.WriteBulk("docs:e")
			peer.WriteStrings([]string{redisJSONPath, `{"name":"e"}`})
			peer.WriteBulk("docs:f")
		},
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{})
	require.NoError(t, err)

	require.Len(t, result.Items, 1)
	assert.Equal(t, "e", result.Items[0].Key)
}

func TestRedisSearchReplyErrors(t *testing.T) {
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

			adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{search: testCase.reply})

			_, err := adapter.Search(t.Context(), "", nil)
			require.ErrorContains(t, err, testCase.wantErr)

			_, err = adapter.Query(t.Context(), nil)
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestRedisSearchEmptyReply(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) { peer.WriteLen(0) },
	})

	result, err := adapter.Search(t.Context(), "", &cache.SearchOptions{Offset: 1, Limit: 2})
	require.NoError(t, err)

	assert.Empty(t, result.Items)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 1, result.Offset)
	assert.Equal(t, 2, result.Limit)
}

func TestRedisQueryWithoutVector(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
		search: func(peer *server.Peer) { writeSearchReply(peer, 1, documentHit("docs:a", `{"name":"a"}`)) },
	})

	result, err := adapter.Query(t.Context(), &cache.QueryOptions{
		Filters:   []cache.Filter{cache.Eq("name", "a b")},
		Offset:    3,
		SortBy:    "name",
		SortOrder: cache.SortAsc,
	})
	require.NoError(t, err)

	arguments := fake.lastSearch(t)
	assert.Equal(t, `@name:{a\ b}`, arguments[1])
	assert.Equal(t, []string{"3", "10"}, argumentsAfter(t, arguments, "LIMIT", 2))
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 3, result.Offset)
	require.Len(t, result.Items, 1)

	_, err = adapter.Query(t.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, "*", fake.lastSearch(t)[1])
}

func TestRedisBuildFilterClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		want   string
		filter cache.Filter
	}{
		{name: "equal", filter: cache.Filter{Field: "tag", Operation: cache.FilterOpEq, Value: "a.b"}, want: `@tag:{a\.b}`},
		{name: "not equal", filter: cache.Filter{Field: "tag", Operation: cache.FilterOpNe, Value: "x"}, want: `-@tag:{x}`},
		{name: "greater than", filter: cache.Filter{Field: "n", Operation: cache.FilterOpGt, Value: 1}, want: `@n:[(1 +inf]`},
		{name: "at least", filter: cache.Filter{Field: "n", Operation: cache.FilterOpGe, Value: 1}, want: `@n:[1 +inf]`},
		{name: "less than", filter: cache.Filter{Field: "n", Operation: cache.FilterOpLt, Value: 2}, want: `@n:[-inf (2]`},
		{name: "at most", filter: cache.Filter{Field: "n", Operation: cache.FilterOpLe, Value: 2}, want: `@n:[-inf 2]`},
		{name: "between", filter: cache.Filter{Field: "n", Operation: cache.FilterOpBetween, Values: []any{1, 5}}, want: `@n:[1 5]`},
		{name: "between with one bound", filter: cache.Filter{Field: "n", Operation: cache.FilterOpBetween, Values: []any{1}}, want: ""},
		{name: "in", filter: cache.Filter{Field: "tag", Operation: cache.FilterOpIn, Values: []any{"a", "b-c"}}, want: `@tag:{a|b\-c}`},
		{name: "in with no values", filter: cache.Filter{Field: "tag", Operation: cache.FilterOpIn}, want: ""},
		{name: "prefix", filter: cache.Filter{Field: "tag", Operation: cache.FilterOpPrefix, Value: "ab"}, want: `@tag:ab*`},
		{name: "unknown operation", filter: cache.Filter{Field: "tag", Operation: cache.FilterOp(99)}, want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := &RedisAdapter[string, string]{}

			assert.Equal(t, testCase.want, adapter.buildFilterClause(testCase.filter))
		})
	}
}

func TestRedisBuildSearchQuerySkipsEmptyClauses(t *testing.T) {
	t.Parallel()

	adapter := &RedisAdapter[string, string]{}

	assert.Equal(t, "*", adapter.buildSearchQuery("", []cache.Filter{{Field: "n", Operation: cache.FilterOpIn}}))
	assert.Equal(t, "q @n:[1 +inf]", adapter.buildSearchQuery("q", []cache.Filter{
		{Field: "n", Operation: cache.FilterOpIn},
		{Field: "n", Operation: cache.FilterOpGe, Value: 1},
	}))
}

func TestEscapeTagValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `plain`, escapeTagValue("plain"))
	assert.Equal(t, `42`, escapeTagValue(42))
	assert.Equal(t,
		`\,\.\<\>\{\}\[\]\"\'\:\;\!\@\#\$\%\^\&\*\(\)\-\+\=\~\ `,
		escapeTagValue(`,.<>{}[]"':;!@#$%^&*()-+=~ `),
	)
}

func TestVectorDistanceMetric(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "L2", vectorDistanceMetric("euclidean"))
	assert.Equal(t, "IP", vectorDistanceMetric("dot_product"))
	assert.Equal(t, "COSINE", vectorDistanceMetric("cosine"))
	assert.Equal(t, "COSINE", vectorDistanceMetric(""))
}

func TestIsUnknownIndexError(t *testing.T) {
	t.Parallel()

	assert.True(t, isUnknownIndexError(errors.New("Unknown index name")))
	assert.False(t, isUnknownIndexError(errors.New("connection refused")))
}

func TestExtractJSONFromDocData(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `{}`, extractJSONFromDocData([]any{"other", "x", redisJSONPath, `{}`}))
	assert.Empty(t, extractJSONFromDocData([]any{int64(1), "x"}))
	assert.Empty(t, extractJSONFromDocData([]any{redisJSONPath, int64(1)}))
	assert.Empty(t, extractJSONFromDocData([]any{redisJSONPath}))
}

func TestExtractVectorScore(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.75, extractVectorScore([]any{int64(1), "x", "s", "0.25"}, "s"), 1e-9)
	assert.InDelta(t, 0.0, extractVectorScore([]any{"s"}, "s"), 1e-9)
	assert.InDelta(t, 0.0, extractVectorScore([]any{"s", int64(1)}, "s"), 1e-9)
	assert.InDelta(t, 0.0, extractVectorScore([]any{"s", "nan-ish"}, "s"), 1e-9)
}

func TestRedisResolveVectorField(t *testing.T) {
	t.Parallel()

	assert.Empty(t, (&RedisAdapter[string, string]{}).resolveVectorField())
	assert.Empty(t, (&RedisAdapter[string, string]{schema: cache.NewSearchSchema(cache.TextField("name"))}).resolveVectorField())
	assert.Equal(t, "vec", (&RedisAdapter[string, string]{schema: vectorSchema()}).resolveVectorField())
}

func TestRedisEnsureIndexExistsCreatesTheIndex(t *testing.T) {
	t.Parallel()

	schema := cache.NewSearchSchema(
		cache.FieldSchema{Name: "title", Type: cache.FieldTypeText, Weight: 2, Sortable: true},
		cache.FieldSchema{Name: "body", Type: cache.FieldTypeText, Weight: 1},
		cache.FieldSchema{Name: "tag", Type: cache.FieldTypeTag},
		cache.FieldSchema{Name: "price", Type: cache.FieldTypeNumeric, Sortable: true},
		cache.FieldSchema{Name: "where", Type: cache.FieldTypeGeo},
		cache.FieldSchema{Name: "vec", Type: cache.FieldTypeVector, Dimension: 3, DistanceMetric: "euclidean"},
	)
	adapter, fake := newSearchAdapter(t, schema, ftFakeConfig{})

	require.NoError(t, adapter.ensureIndexExists(t.Context()))
	require.NoError(t, adapter.ensureIndexExists(t.Context()))

	assert.Len(t, fake.calls("FT.INFO"), 1)
	creates := fake.calls("FT.CREATE")
	require.Len(t, creates, 1)
	assert.Equal(t, []string{
		adapter.indexName, "ON", "JSON", "PREFIX", "1", "docs:", "SCHEMA",
		"$.title", "AS", "title", "TEXT", "WEIGHT", "2", "SORTABLE",
		"$.body", "AS", "body", "TEXT",
		"$.tag", "AS", "tag", "TAG",
		"$.price", "AS", "price", "NUMERIC", "SORTABLE",
		"$.where", "AS", "where", "GEO",
		"$.vec", "AS", "vec", "VECTOR", "HNSW", "6", "TYPE", "FLOAT32", "DIM", "3", "DISTANCE_METRIC", "L2",
	}, creates[0])
}

func TestRedisEnsureIndexExistsReusesAnExistingIndex(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{indexExists: true})

	require.NoError(t, adapter.ensureIndexExists(t.Context()))

	assert.Empty(t, fake.calls("FT.CREATE"))
	assert.True(t, adapter.indexCreated)
}

func TestRedisEnsureIndexExistsReportsCreateFailure(t *testing.T) {
	t.Parallel()

	adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{createError: "Bad schema"})

	err := adapter.ensureIndexExists(t.Context())
	require.ErrorContains(t, err, "failed to create RediSearch index")

	_, err = adapter.Search(t.Context(), "", nil)
	require.ErrorContains(t, err, "Bad schema")
	_, err = adapter.Query(t.Context(), nil)
	require.ErrorContains(t, err, "Bad schema")
}

func TestRedisEnsureIndexExistsWithoutSchema(t *testing.T) {
	t.Parallel()

	adapter := &RedisAdapter[string, string]{}

	require.ErrorIs(t, adapter.ensureIndexExists(t.Context()), cache.ErrSearchNotSupported)
	adapter.indexDocument(t.Context(), "k", "v")
	adapter.dropIndex(t.Context())
}

func TestRedisDropIndex(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		dropError string
	}{
		{name: "dropped", dropError: ""},
		{name: "an unknown index is ignored", dropError: "Unknown index name"},
		{name: "another failure is logged", dropError: "ERR index busy"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{dropError: testCase.dropError})
			adapter.dropIndex(t.Context())
			assert.Empty(t, fake.calls("FT.DROPINDEX"), "an index never created is not dropped")

			require.NoError(t, adapter.ensureIndexExists(t.Context()))
			require.NoError(t, adapter.InvalidateAll(t.Context()))

			assert.Equal(t, [][]string{{adapter.indexName}}, fake.calls("FT.DROPINDEX"))
			assert.False(t, adapter.indexCreated)
		})
	}
}

func TestRedisJSONStorageRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{})

	require.NoError(t, adapter.Set(ctx, "a", searchDoc{Name: "a", Price: 3}, "tag-a"))
	require.NoError(t, adapter.SetWithTTL(ctx, "b", searchDoc{Name: "b"}, time.Minute, "tag-b"))
	require.NoError(t, adapter.BulkSet(ctx, map[string]searchDoc{"c": {Name: "c"}}, "tag-c"))

	assert.Len(t, fake.calls("JSON.SET"), 3)
	assert.Equal(t, time.Hour, fake.db.TTL("docs:a"))
	assert.Equal(t, time.Minute, fake.db.TTL("docs:b"))
	assert.Equal(t, time.Hour, fake.db.TTL("docs:c"))

	value, found, err := adapter.GetIfPresent(ctx, "a")
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, searchDoc{Name: "a", Price: 3}, value)

	_, found, err = adapter.GetIfPresent(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisJSONSetFailureIsNotReturned(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{jsonSetError: "OOM"})

	require.NoError(t, adapter.Set(t.Context(), "a", searchDoc{Name: "a"}))

	assert.Len(t, fake.calls("JSON.SET"), 1)
	assert.False(t, fake.db.Exists("docs:a"))
}

func TestRedisIndexDocumentSkipsWriteWhenTheIndexCannotBeCreated(t *testing.T) {
	t.Parallel()

	adapter, fake := newSearchAdapter(t, vectorSchema(), ftFakeConfig{createError: "Bad schema"})

	require.NoError(t, adapter.Set(t.Context(), "a", searchDoc{Name: "a"}))

	assert.Empty(t, fake.calls("JSON.SET"))
}

func TestRedisGetJSONValueReplies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		reply     func(peer *server.Peer)
		name      string
		wantErr   string
		wantFound bool
	}{
		{name: "an empty array is not found", reply: func(peer *server.Peer) { peer.WriteBulk("[]") }},
		{name: "a non-string reply", reply: func(peer *server.Peer) { peer.WriteInt(1) }, wantErr: "unexpected result type"},
		{name: "invalid JSON", reply: func(peer *server.Peer) { peer.WriteBulk("{") }, wantErr: "failed to unmarshal"},
		{name: "a server error", reply: func(peer *server.Peer) { peer.WriteError("WRONGTYPE") }, wantErr: "JSON.GET failed"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter, _ := newSearchAdapter(t, vectorSchema(), ftFakeConfig{
				jsonGet: func(peer *server.Peer, _ string) { testCase.reply(peer) },
			})

			_, found, err := adapter.GetIfPresent(t.Context(), "a")
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.wantFound, found)
		})
	}
}

func TestRedisSetJSONValueReportsMarshalFailure(t *testing.T) {
	t.Parallel()

	db := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: db.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	adapter := &RedisAdapter[string, chan int]{client: client}

	require.ErrorContains(t, adapter.setJSONValue(t.Context(), "k", make(chan int)), "failed to marshal")
}

func TestSearchPageSize(t *testing.T) {
	testCases := []struct {
		name    string
		options *cache.SearchOptions
		want    int
	}{
		{name: "no options", options: nil, want: DefaultSearchResultLimit},
		{name: "zero limit", options: &cache.SearchOptions{Limit: 0}, want: DefaultSearchResultLimit},
		{name: "explicit limit", options: &cache.SearchOptions{Limit: 25}, want: 25},
		{name: "every match", options: &cache.SearchOptions{Limit: cache.NoLimit}, want: MaxSearchResultLimit},
		{name: "above the server maximum", options: &cache.SearchOptions{Limit: MaxSearchResultLimit + 1}, want: MaxSearchResultLimit},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, searchPageSize(testCase.options))
		})
	}
}
