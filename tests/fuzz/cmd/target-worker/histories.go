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

package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"

	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/search/search_adapters"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_dto"
	fb "piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"piko.sh/piko/internal/wal/wal_domain"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/wdk/cache"
	otter "piko.sh/piko/wdk/cache/cache_provider_otter"
	"piko.sh/safeconv"
)

const (
	// cacheCapacity bounds the cache's entry count.
	cacheCapacity = 24

	// cacheOperationCount is the number of cache history operations.
	cacheOperationCount = 7

	// cacheKeySpan is the step stride between drawn cache tags.
	cacheKeySpan = cacheOperationCount * 4

	// cacheStepRange is the number of distinct drawn cache steps.
	cacheStepRange = cacheKeySpan * 3

	// minExtraCacheSteps is the smallest number of drawn cache steps.
	minExtraCacheSteps = 3

	// extraCacheStepSpread is the number of distinct drawn cache step counts.
	extraCacheStepSpread = 10

	// cacheOperationSetAlternative sets a value through the second set step.
	cacheOperationSetAlternative = 1

	// cacheOperationInvalidate invalidates one key.
	cacheOperationInvalidate = 2

	// cacheOperationInvalidateAll invalidates every key.
	cacheOperationInvalidateAll = 3

	// cacheOperationInvalidateTag invalidates every key carrying the step's tag.
	cacheOperationInvalidateTag = 4

	// cacheOperationBulkSet sets two keys in one bulk update.
	cacheOperationBulkSet = 5

	// cacheOperationReopen closes and reopens the persistent cache.
	cacheOperationReopen = 6

	// searchOperationCount is the number of search history operations.
	searchOperationCount = 3

	// searchOperationDelete deletes the step's route.
	searchOperationDelete = 2

	// searchRouteCount is the number of distinct modelled search routes.
	searchRouteCount = 4

	// maxExtraSearchSteps bounds the search steps drawn beyond the first.
	maxExtraSearchSteps = 5

	// searchStepRange is the number of distinct drawn search steps.
	searchStepRange = 24

	// searchResultLimit bounds the routes returned for one term.
	searchResultLimit = 8
)

var (
	// errUnexpectedLoad reports a bulk read that needed a loader for a written key.
	errUnexpectedLoad = errors.New("previously written key unexpectedly needed a loader")

	// cacheKeys lists every key the cache history may write.
	cacheKeys = []string{"alpha", "beta", "gamma", "delta"}

	// cacheTags lists every tag the cache history may attach.
	cacheTags = []string{"red", "blue", "ochre"}
)

// stringCodec encodes cache keys and values as UTF-8 bytes.
type stringCodec struct{}

// cacheHistory holds a persistent cache under test and its independent model.
type cacheHistory struct {
	// provider is the open cache, or nil after a failed reopen.
	provider cache.Cache[string, string]

	// want maps keys to their modelled values.
	want map[string]string

	// tags maps keys to their modelled tags.
	tags map[string]string

	// options reopens the persistent cache.
	options cache.Options[string, string]
}

// searchProbe runs ranked searches against one loaded index.
type searchProbe struct {
	// query processes search terms against the index.
	query *search_domain.QueryProcessor

	// reader holds the loaded index.
	reader *search_adapters.FlatBufferIndexReader

	// scorer ranks matching documents.
	scorer *search_domain.BM25Scorer
}

// EncodeKey encodes a string cache key as bytes.
//
// Takes key (string) which is the cache key to encode.
//
// Returns []byte which holds the cache key as UTF-8 bytes.
// Returns error which is always nil.
func (stringCodec) EncodeKey(key string) ([]byte, error) { return []byte(key), nil }

// DecodeKey decodes a cache key as a string.
//
// Takes encoded ([]byte) which is the encoded cache key.
//
// Returns string which is the decoded cache key.
// Returns error which is always nil.
func (stringCodec) DecodeKey(encoded []byte) (string, error) { return string(encoded), nil }

// EncodeValue encodes a string cache value as bytes.
//
// Takes value (string) which is the cache value to encode.
//
// Returns []byte which holds the cache value as UTF-8 bytes.
// Returns error which is always nil.
func (stringCodec) EncodeValue(value string) ([]byte, error) { return []byte(value), nil }

// DecodeValue decodes a cache value as a string.
//
// Takes encoded ([]byte) which is the encoded cache value.
//
// Returns string which is the decoded cache value.
// Returns error which is always nil.
func (stringCodec) DecodeValue(encoded []byte) (string, error) { return string(encoded), nil }

// apply performs one cache history step on the cache and the model.
//
// Takes stepIndex (int) which identifies the current history step.
// Takes step (int) which encodes the operation, key and tag.
// Takes value (string) which is the value written by set steps.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the step succeeds.
func (h *cacheHistory) apply(ctx context.Context, stepIndex, step int, value string) error {
	key := cacheKeys[step/cacheOperationCount%len(cacheKeys)]
	tag := cacheTags[step/cacheKeySpan%len(cacheTags)]
	var err error
	switch step % cacheOperationCount {
	case 0, cacheOperationSetAlternative:
		err = h.provider.Set(ctx, key, value, tag)
		h.want[key] = value
		h.tags[key] = tag
	case cacheOperationInvalidate:
		err = h.provider.Invalidate(ctx, key)
		delete(h.want, key)
		delete(h.tags, key)
	case cacheOperationInvalidateAll:
		err = h.provider.InvalidateAll(ctx)
		clear(h.want)
		clear(h.tags)
	case cacheOperationInvalidateTag:
		err = h.invalidateTag(ctx, stepIndex, tag)
	case cacheOperationBulkSet:
		updates := map[string]string{key: value, cacheKeys[(step/cacheOperationCount+1)%len(cacheKeys)]: value + "+"}
		err = h.provider.BulkSet(ctx, updates)

		maps.Copy(h.want, updates)
	case cacheOperationReopen:
		err = h.reopen(ctx)
	}
	if err != nil {
		return fmt.Errorf("cache step %d: %w", stepIndex, err)
	}
	return nil
}

// invalidateTag invalidates a tag and checks the reported removal count.
//
// Takes stepIndex (int) which identifies the current history step.
// Takes tag (string) which is the invalidated tag.
//
// Returns error which reports the production failure or a removal-count mismatch.
func (h *cacheHistory) invalidateTag(ctx context.Context, stepIndex int, tag string) error {
	count, err := h.provider.InvalidateByTags(ctx, tag)
	removed := 0
	for taggedKey, keyTag := range h.tags {
		if keyTag == tag {
			delete(h.want, taggedKey)
			delete(h.tags, taggedKey)
			removed++
		}
	}
	if err == nil && count != removed {
		return property("tag-invalidation", "step %d tag %s removed %d, want %d", stepIndex, tag, count, removed)
	}
	return err
}

// reopen closes the persistent cache and opens it again from disk.
//
// Returns error which reports a failed close or reopen.
func (h *cacheHistory) reopen(ctx context.Context) error {
	if err := h.provider.Close(ctx); err != nil {
		return err
	}
	h.provider = nil
	provider, err := otter.OtterProviderFactory(h.options)
	h.provider = provider
	return err
}

// shutdown closes the cache when it is open.
//
// Returns error which reports a failed close, or nil when closed or already closed.
func (h *cacheHistory) shutdown(ctx context.Context) error {
	if h.provider == nil {
		return nil
	}
	return h.provider.Close(ctx)
}

// routes searches one term and lists the matching routes in rank order.
//
// Takes word (string) which is the search term.
// Takes limit (int) which bounds the returned routes.
// Takes offset (int) which skips leading ranked routes.
//
// Returns []string which lists the matching routes in rank order.
// Returns error which reports a failed search or metadata read.
func (p *searchProbe) routes(ctx context.Context, word string, limit, offset int) ([]string, error) {
	var config search_dto.SearchConfig
	config.Limit = limit
	config.Offset = offset
	matches, err := p.query.Search(ctx, word, p.reader, p.scorer, config)
	if err != nil {
		return nil, fmt.Errorf("searching %q: %w", word, err)
	}
	got := make([]string, 0, len(matches))
	for _, match := range matches {
		metadata, err := p.reader.GetDocMetadata(match.DocumentID)
		if err != nil {
			return nil, fmt.Errorf("reading document %d metadata: %w", match.DocumentID, err)
		}
		got = append(got, metadata.Route)
	}
	return got, nil
}

// compareHistoryRows compares the current storage rows with the independent model.
//
// Takes domain (string) which identifies the checked subsystem.
// Takes step (int) which identifies the current history step.
// Takes want (map[string]string) which holds the independently expected result.
// Takes got (map[string]string) which holds the observed result.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareHistoryRows(domain string, step int, want, got map[string]string) error {
	if !maps.Equal(want, got) {
		return property(domain+"-rows", "step %d got %v, want %v", step, got, want)
	}
	return nil
}

// checkCache checks cache updates, tags and deletions against a model.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCache(ctx context.Context, choices *tape.Tape, _ *casefile.Case) (retErr error) {
	directory, err := os.MkdirTemp("", "piko-cache-history-")
	if err != nil {
		return infrastructure("creating cache directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(directory); removeErr != nil && retErr == nil {
			retErr = infrastructure("removing cache directory: %w", removeErr)
		}
	}()
	history := &cacheHistory{
		provider: nil,
		want:     map[string]string{},
		tags:     map[string]string{},
		options:  cacheOptions(directory, choices.Uint(2) == 1),
	}
	history.provider, err = otter.OtterProviderFactory(history.options)
	if err != nil {
		return fmt.Errorf("opening cache: %w", err)
	}
	defer func() {
		if closeErr := history.shutdown(ctx); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("closing cache: %w", closeErr)
		}
	}()
	for stepIndex, step := range drawCacheSteps(choices) {
		value := fmt.Sprintf("%d:%s", stepIndex, label(choices))
		if err := history.apply(ctx, stepIndex, step, value); err != nil {
			return err
		}
		if err := compareCacheStep(ctx, history.provider, cacheKeys, history.want, stepIndex); err != nil {
			return err
		}
	}
	return nil
}

// cacheOptions configures a persistent cache in a directory.
//
// Takes directory (string) which holds the cache's write-ahead log.
// Takes compressed (bool) which enables log compression.
//
// Returns cache.Options[string, string] which configures the persistent cache.
func cacheOptions(directory string, compressed bool) cache.Options[string, string] {
	persistConfig := otter.DefaultPersistConfig(directory)
	persistConfig.SyncMode = wal_domain.SyncModeEveryWrite
	persistConfig.EnableCompression = compressed
	var options cache.Options[string, string]
	options.MaximumEntries = cacheCapacity
	options.ProviderSpecific = otter.PersistenceConfig[string, string]{
		Enabled:           true,
		ReadOnly:          false,
		QuarantineCorrupt: false,
		WALConfig:         persistConfig,
		KeyCodec:          stringCodec{},
		ValueCodec:        stringCodec{},
	}
	return options
}

// drawCacheSteps lists the fixed opening cache steps followed by drawn steps.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns []int which lists the encoded cache steps.
func drawCacheSteps(choices *tape.Tape) []int {
	extra := minExtraCacheSteps + drawInt(choices, extraCacheStepSpread)
	opening := []int{
		cacheStep(0, 0),
		cacheStep(0, 1),
		cacheStep(0, 2),
		cacheStep(cacheOperationReopen, 0),
		cacheStep(cacheOperationSetAlternative, 1),
		cacheStep(cacheOperationInvalidateTag, 0),
		cacheStep(cacheOperationSetAlternative, 0),
		cacheStep(cacheOperationBulkSet, 0),
		cacheStep(cacheOperationReopen, 0),
	}
	steps := make([]int, 0, len(opening)+extra)
	steps = append(steps, opening...)
	for range extra {
		steps = append(steps, drawInt(choices, cacheStepRange))
	}
	return steps
}

// cacheStep encodes a cache operation on a key with the first tag.
//
// Takes operation (int) which selects the cache operation.
// Takes key (int) which indexes the cache key.
//
// Returns int which is the encoded cache step.
func cacheStep(operation, key int) int {
	return key*cacheOperationCount + operation
}

// compareCacheStep compares single and bulk cache reads with the independent model.
//
// Takes provider (cache.Cache[string, string]) which is the cache under test.
// Takes keys ([]string) which lists every key the history may write.
// Takes want (map[string]string) which holds the independently expected rows.
// Takes step (int) which identifies the current history step.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func compareCacheStep(ctx context.Context, provider cache.Cache[string, string], keys []string, want map[string]string, step int) error {
	got := map[string]string{}
	for _, key := range keys {
		value, ok, err := provider.GetIfPresent(ctx, key)
		if err != nil {
			return fmt.Errorf("reading cache key %s: %w", key, err)
		}
		if ok {
			got[key] = value
		}
	}
	if err := compareHistoryRows("cache", step, want, got); err != nil {
		return err
	}
	bulk, err := provider.BulkGet(ctx, slices.Sorted(maps.Keys(want)), cache.BulkLoaderFunc[string, string](func(context.Context, []string) (map[string]string, error) {
		return nil, errUnexpectedLoad
	}))
	if err != nil {
		return fmt.Errorf("bulk-reading cache: %w", err)
	}
	return compareHistoryRows("bulk", step, want, bulk)
}

// checkSearch checks rebuilt search rankings against independent document counts.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSearch(ctx context.Context, choices *tape.Tape, _ *casefile.Case) error {
	words := []string{"bamboo", "cobalt", "hazelnut", "indigo", "quartz", "saffron"}
	documents := map[string]string{"/entry-0": pick(choices, words), "/entry-1": pick(choices, words)}
	extra := 1 + drawInt(choices, maxExtraSearchSteps)
	steps := make([]int, 0, extra+searchOperationCount)
	steps = append(steps, searchStep(0, 0), searchStep(1, 2), searchStep(0, 1))
	for range extra {
		steps = append(steps, drawInt(choices, searchStepRange))
	}
	for stepIndex, step := range steps {
		route := fmt.Sprintf("/entry-%d", step/searchOperationCount%searchRouteCount)
		if step%searchOperationCount == searchOperationDelete {
			delete(documents, route)
		} else {
			documents[route] = pick(choices, words)
		}
		if err := checkSearchStep(ctx, documents, words, stepIndex); err != nil {
			return err
		}
	}
	return nil
}

// searchStep encodes a search operation on a route.
//
// Takes operation (int) which selects the search operation.
// Takes route (int) which indexes the route.
//
// Returns int which is the encoded search step.
func searchStep(operation, route int) int {
	return route*searchOperationCount + operation
}

// checkSearchStep rebuilds the search index for one history step and checks its rankings.
//
// Takes documents (map[string]string) which maps routes to modelled document text.
// Takes words ([]string) which lists every modelled search term.
// Takes step (int) which identifies the current history step.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSearchStep(ctx context.Context, documents map[string]string, words []string, step int) error {
	items := make([]collection_dto.ContentItem, 0, len(documents)+2)
	for _, route := range slices.Sorted(maps.Keys(documents)) {
		items = append(items, contentItem(route, documents[route]))
	}

	ranked := []string{"/en/catalogue", "/fr/catalogue"}
	if step%2 == 1 {
		ranked[0], ranked[1] = ranked[1], ranked[0]
	}
	for rank, body := range []string{"zephyr zephyr zephyr zephyr", "zephyr spruce spruce spruce"} {
		items = append(items, contentItem(ranked[rank], body))
	}
	probe, err := loadSearchProbe(ctx, items, step)
	if err != nil {
		return err
	}
	if err := checkRankPages(ctx, probe, ranked, step); err != nil {
		return err
	}
	for _, word := range words {
		got, err := probe.routes(ctx, word, searchResultLimit, 0)
		if err != nil {
			return err
		}
		if err := matchSearchRoutes(documents, word, step, got); err != nil {
			return err
		}
	}
	return nil
}

// contentItem builds a titled collection item for indexing.
//
// Takes route (string) which is the item's URL.
// Takes body (string) which is the item's plain content.
//
// Returns collection_dto.ContentItem which is the indexable item.
func contentItem(route, body string) collection_dto.ContentItem {
	var item collection_dto.ContentItem
	item.URL = route
	item.PlainContent = body
	item.Metadata = map[string]any{"Title": "Inventory"}
	return item
}

// loadSearchProbe builds and loads a search index over the items.
//
// Takes items ([]collection_dto.ContentItem) which lists the indexed items.
// Takes step (int) which identifies the current history step.
//
// Returns *searchProbe which searches the loaded index.
// Returns error which reports a failed build, a failed load or a wrong corpus size.
func loadSearchProbe(ctx context.Context, items []collection_dto.ContentItem, step int) (*searchProbe, error) {
	buildConfig := search_domain.DefaultIndexBuildConfig()
	buildConfig.AnalysisMode = fb.SearchModeFast
	data, err := search_domain.NewIndexBuilder().BuildIndex(ctx, "inventory", items, fb.SearchModeFast, buildConfig)
	if err != nil {
		return nil, fmt.Errorf("building search index at step %d: %w", step, err)
	}
	reader := search_adapters.NewFlatBufferIndexReader()
	if err = reader.LoadIndex(data); err != nil {
		return nil, fmt.Errorf("loading search index at step %d: %w", step, err)
	}
	if total := reader.GetCorpusStats().TotalDocuments; total != safeconv.IntToUint32(len(items)) {
		return nil, property("search-count", "step %d corpus size %d, want %d", step, total, len(items))
	}
	return &searchProbe{
		query:  search_domain.NewQueryProcessorForIndex(reader),
		reader: reader,
		scorer: search_domain.NewBM25Scorer(search_domain.BM25DefaultK1, search_domain.BM25DefaultB),
	}, nil
}

// checkRankPages checks limit and offset paging over the ranked routes.
//
// Takes probe (*searchProbe) which searches the loaded index.
// Takes ranked ([]string) which lists the expected ranked routes.
// Takes step (int) which identifies the current history step.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkRankPages(ctx context.Context, probe *searchProbe, ranked []string, step int) error {
	for _, page := range []struct {
		want          []string
		limit, offset int
	}{
		{want: ranked, limit: 2, offset: 0},
		{want: ranked[:1], limit: 1, offset: 0},
		{want: ranked[1:], limit: 1, offset: 1},
		{want: nil, limit: 1, offset: 2},
	} {
		got, err := probe.routes(ctx, "zephyr", page.limit, page.offset)
		if err != nil {
			return err
		}
		if !slices.Equal(got, page.want) {
			return property("search-rank-page", "step %d limit=%d offset=%d got %v, want %v", step, page.limit, page.offset, got, page.want)
		}
	}
	return nil
}

// matchSearchRoutes checks ranked routes for the modelled search term.
//
// Takes documents (map[string]string) which maps routes to independently modelled
// document text.
// Takes word (string) which is the search term.
// Takes step (int) which identifies the current history step.
// Takes got ([]string) which holds the observed result.
//
// Returns error which reports a route-order mismatch with the independent ranking model,
// or nil on success.
func matchSearchRoutes(documents map[string]string, word string, step int, got []string) error {
	want := []string{}
	for route, text := range documents {
		if text == word {
			want = append(want, route)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(want, got) {
		return property("search-routes", "step %d word %s: got %v, want %v", step, word, got, want)
	}
	return nil
}
