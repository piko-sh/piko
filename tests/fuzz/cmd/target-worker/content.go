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
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"piko.sh/fuzzkit/tape"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/markdown/markdown_dto"
	svg "piko.sh/piko/internal/pdfwriter/pdfwriter_adapters/driven_svgwriter"
	"piko.sh/piko/internal/registry/registry_adapters"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/route_pattern"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

const (
	// maxAssetBytes bounds the asset bytes read back from the registry.
	maxAssetBytes = 1024

	// defaultLocale is the store's default locale.
	defaultLocale = "en"

	// frenchLocale is the base language of the regional locale.
	frenchLocale = "fr"

	// routeGroupRange is the number of distinct route group prefixes.
	routeGroupRange = 32

	// routeNumberRange is the number of distinct drawn route numbers.
	routeNumberRange = 99

	// staticRouteBase offsets static routes away from parameter values.
	staticRouteBase = 100

	// pluralCountRange is the number of distinct drawn plural counts.
	pluralCountRange = 5

	// fixedLocaleSteps is the number of fixed locale replacement steps.
	fixedLocaleSteps = 5

	// localeStepRange is the number of distinct drawn locale replacement steps.
	localeStepRange = 6

	// messageRange is the number of distinct drawn message suffixes.
	messageRange = 1000

	// sectionLevel is the heading level of a top-level section.
	sectionLevel = 2

	// subsectionLevel is the heading level of a subsection.
	subsectionLevel = 3

	// maxExtraSections bounds the sections drawn beyond the first.
	maxExtraSections = 4

	// maxSubsections bounds the subsections drawn per section.
	maxSubsections = 4

	// assetContentRange is the number of distinct drawn asset contents.
	assetContentRange = 10000

	// canvasRange is the number of distinct drawn starting coordinates.
	canvasRange = 128

	// maxExtraSegments bounds the path segments drawn beyond the first.
	maxExtraSegments = 8

	// relativeStepRange is the number of distinct drawn relative steps.
	relativeStepRange = 129

	// relativeStepOffset centres drawn relative steps on zero.
	relativeStepOffset = 64

	// translationRange is the number of distinct drawn translations.
	translationRange = 31

	// translationOffset centres drawn translations on zero.
	translationOffset = 15

	// maxExtraScale bounds the drawn scale factor beyond one.
	maxExtraScale = 5

	// rightAngle is a quarter turn in degrees.
	rightAngle = 90

	// quarterTurns is the number of distinct drawn rotations.
	quarterTurns = 4

	// transformTolerance bounds the accepted floating-point transform error.
	transformTolerance = 1e-7
)

var (
	// errReadLimitExceeded reports a body longer than the caller's read limit.
	errReadLimitExceeded = errors.New("read limit exceeded")
)

// checkRouting checks selected routes against independent path expectations.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkRouting(ctx context.Context, choices *tape.Tape, _ *casefile.Case) error {
	prefix := "/group" + strconv.FormatUint(choices.Uint(routeGroupRange), decimalBase)
	name := pick(choices, []string{"slug", "id", "item"})
	expression := pick(choices, []string{"[0-9]+", "[0-9]{1,3}"})
	pattern := prefix + "/{" + name + ":" + expression + "}"
	part := route_pattern.ParseTrailing(pattern)
	if !part.Found || !part.HasRegex || part.Name != name || part.Regex != expression || part.Prefix != prefix+"/" {
		return property("route-fields", "trailing route fields differ: %+v", part)
	}
	static := strconv.Itoa(drawInt(choices, routeNumberRange) + staticRouteBase)
	pagePaths := []string{pattern, prefix + "/" + static}
	router, err := mountManifestRoutes(pagePaths, name)
	if err != nil {
		return err
	}
	parameter := strconv.Itoa(drawInt(choices, routeNumberRange))
	for _, probe := range []struct {
		suffix string
		body   string
		code   int
	}{
		{suffix: static, body: "static", code: http.StatusOK},
		{suffix: parameter, body: parameter, code: http.StatusOK},
		{suffix: "x", body: "", code: http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, prefix+"/"+probe.suffix, nil)
		router.ServeHTTP(recorder, request)
		if recorder.Code != probe.code || (probe.code == http.StatusOK && recorder.Body.String() != probe.body) {
			return property("dispatch", "route %q got HTTP %d %q", probe.suffix, recorder.Code, recorder.Body.String())
		}
	}
	return nil
}

// mountManifestRoutes builds a route manifest for the pages and mounts its routes.
//
// Takes pagePaths ([]string) which lists the parameterised route then the static route.
// Takes name (string) which is the route parameter echoed by the parameterised route.
//
// Returns chi.Router which serves the manifest's routes.
// Returns error which reports a failed manifest build or a manifest route mismatch.
func mountManifestRoutes(pagePaths []string, name string) (chi.Router, error) {
	base := filepath.Join(string(filepath.Separator), "fuzz-fixture")
	var paths generator_domain.GeneratorPathsConfig
	paths.PagesSourceDir = "pages"
	manifestBuilder := generator_domain.NewManifestBuilder(paths, defaultLocale, base)
	artefacts := make([]*generator_dto.GeneratedArtefact, 0, len(pagePaths))
	for index, route := range pagePaths {
		artefacts = append(artefacts, pageArtefact(filepath.Join(base, "pages", filepath.FromSlash(strings.TrimPrefix(route, "/")+".pk")), index))
	}
	manifest, err := manifestBuilder.Build(artefacts)
	if err != nil {
		return nil, fmt.Errorf("building route manifest: %w", err)
	}
	router := chi.NewRouter()
	for index, route := range pagePaths {
		key := "pages" + route + ".pk"
		page, ok := manifest.Pages[key]
		if !ok {
			return nil, property("manifest-route", "source %s has no manifest page", key)
		}
		actual := page.RoutePatterns[defaultLocale]
		if actual != route {
			return nil, property("manifest-route", "source %s mapped to route %q, want %q", key, actual, route)
		}
		if index == 0 {
			router.Get(actual, func(writer http.ResponseWriter, request *http.Request) {
				_, _ = io.WriteString(writer, chi.URLParam(request, name))
			})
		} else {
			router.Get(actual, func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(writer, "static")
			})
		}
	}
	return router, nil
}

// pageArtefact builds a generated public page artefact for a page source.
//
// Takes source (string) which is the page's absolute source path.
// Takes index (int) which distinguishes the page's hashed name.
//
// Returns *generator_dto.GeneratedArtefact which describes the generated page.
func pageArtefact(source string, index int) *generator_dto.GeneratedArtefact {
	var parsed annotator_dto.ParsedComponent
	parsed.SourcePath = source
	parsed.ComponentType = "page"
	parsed.Script = &annotator_dto.ParsedScript{}
	var component annotator_dto.VirtualComponent
	component.IsPage = true
	component.IsPublic = true
	component.HashedName = fmt.Sprintf("page%d", index)
	component.Source = &parsed
	var artefact generator_dto.GeneratedArtefact
	artefact.Component = &component
	artefact.Result = &annotator_dto.AnnotationResult{}
	return &artefact
}

// checkResolver checks source resolution through the production resolver.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkResolver(ctx context.Context, choices *tape.Tape, _ *casefile.Case) error {
	base := filepath.Join(string(filepath.Separator), "fixture")
	module := fmt.Sprintf("example.org/f%d", choices.Uint(byteRange))
	name := fmt.Sprintf("partials/layout%d.pk", choices.Uint(byteRange))
	resolver := resolver_adapters.NewInMemoryModuleResolver(module, base)
	actual, err := resolver.ResolvePKPath(ctx, module+"/"+name, module+"/pages/main.pk")
	if err != nil {
		return fmt.Errorf("resolving %s: %w", name, err)
	}
	want := filepath.Join(base, filepath.FromSlash(name))
	if actual != want {
		return property("local-module", "resolved %q, want %q", actual, want)
	}
	if _, err := resolver.ResolvePKPath(ctx, module+"/"+name+".css", module+"/pages/main.pk"); err == nil {
		return property("extension", "non-PK component path was accepted")
	}
	return nil
}

// checkTranslations checks locale replacement and language fallback.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkTranslations(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	store := i18n_domain.NewStore(defaultLocale)
	word := label(choices)
	if problems := store.AddTranslations(defaultLocale, map[string]string{"welcome": "Hello ${name}"}); len(problems) > 0 {
		return property("template", "translation parse: %v", problems)
	}
	if got, ok := store.ResolveMessage("welcome", frenchLocale, map[string]any{"name": word}, 0); !ok || got != "Hello "+word {
		return property("fallback", "locale fallback and substitution got %q (%v)", got, ok)
	}
	if _, ok := store.ResolveMessage("absent", defaultLocale, nil, 0); ok {
		return property("missing-key", "missing translation resolved")
	}
	count := drawInt(choices, pluralCountRange)
	pluralLocale := pick(choices, []string{defaultLocale, frenchLocale})
	want := "many"
	if count == 1 || (pluralLocale == frenchLocale && count == 0) {
		want = "one"
	}
	if got := i18n_domain.SelectPluralForm(count, pluralLocale, []string{"one", "many"}); got != want {
		return property("plural", "plural %s %d got %q, want %q", pluralLocale, count, got, want)
	}
	return checkLocaleSnapshots(store, choices)
}

// checkLocaleSnapshots checks lookups after each locale snapshot replacement.
//
// Each locale snapshot replaces its entries. The same key is resolved via regional,
// base-language and default lookups after each replacement.
//
// Takes store (*i18n_domain.Store) which is the production translation store.
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkLocaleSnapshots(store *i18n_domain.Store, choices *tape.Tape) error {
	locales := []string{defaultLocale, frenchLocale, "fr-CA"}
	if problems := store.AddTranslations("fr-CA", map[string]string{"registered": "yes"}); len(problems) > 0 {
		return property("locale-data", "regional registration: %v", problems)
	}
	steps := make([]int, 0, fixedLocaleSteps+2)
	for step := range fixedLocaleSteps {
		steps = append(steps, step)
	}
	steps = append(steps, drawInt(choices, localeStepRange), drawInt(choices, localeStepRange))
	model := map[string]string{}
	for _, step := range steps {
		name := locales[step%len(locales)]
		text := fmt.Sprintf("%s-%d", name, choices.Uint(messageRange))
		messages := map[string]string{"registered": "yes"}
		if step < len(locales) {
			messages["colour"] = text
			model[name] = text
		} else {
			delete(model, name)
		}
		if problems := store.AddTranslations(name, messages); len(problems) > 0 {
			return property("locale-data", "%v", problems)
		}
		for _, locale := range locales {
			expected := modelledMessage(model, locale)
			got, ok := store.ResolveMessage("colour", locale, nil, 0)
			if got != expected || ok != (expected != "") {
				return property("locale-chain", "locale %s got %q %t, want %q", locale, got, ok, expected)
			}
		}
	}
	return nil
}

// modelledMessage resolves a locale's message through the modelled fallback chain.
//
// Takes model (map[string]string) which maps locales to their modelled message.
// Takes locale (string) which is the requested locale.
//
// Returns string which is the first modelled message on the chain, or empty when none.
func modelledMessage(model map[string]string, locale string) string {
	chain := []string{locale}
	if locale == "fr-CA" {
		chain = append(chain, frenchLocale)
	}
	chain = append(chain, defaultLocale)
	for _, candidate := range chain {
		if modelled, ok := model[candidate]; ok {
			return modelled
		}
	}
	return ""
}

// checkCollections checks collection heading-section trees against the independent model.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkCollections(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	sections := 1 + drawInt(choices, maxExtraSections)
	headings := make([]markdown_dto.SectionData, 0, sections)
	want := make([]collection_dto.SectionNode, 0, sections)
	for section := range sections {
		parent := collection_dto.SectionNode{Level: sectionLevel, Slug: fmt.Sprintf("s%d", section), Title: label(choices), Children: nil}
		headings = append(headings, markdown_dto.SectionData{Level: sectionLevel, Slug: parent.Slug, Title: parent.Title})
		for subsection := range drawInt(choices, maxSubsections) {
			child := collection_dto.SectionNode{Level: subsectionLevel, Slug: fmt.Sprintf("s%d-%d", section, subsection), Title: label(choices), Children: nil}
			headings = append(headings, markdown_dto.SectionData{Level: subsectionLevel, Slug: child.Slug, Title: child.Title})
			parent.Children = append(parent.Children, child)
		}
		want = append(want, parent)
	}
	got := collection_domain.BuildSectionTree(headings)
	if !reflect.DeepEqual(got, want) {
		return property("section-tree", "heading hierarchy got %+v, want %+v", got, want)
	}
	return nil
}

// checkAssets checks asset aliases and content-addressed storage updates.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkAssets(ctx context.Context, choices *tape.Tape, _ *casefile.Case) error {
	file := fmt.Sprintf("icons/item%d.%s", choices.Uint(byteRange), pick(choices, []string{"svg", "css", "js"}))
	module := "example.org/demo"
	prefix := "/_piko/assets"
	want := prefix + "/" + module + "/" + file
	for _, ref := range []string{"@/" + file, module + "/styles/../" + file, want} {
		if got := assetpath.Transform(ref, module, prefix); got != want {
			return property("rewrite", "asset %q mapped to %q, want %q", ref, got, want)
		}
	}
	got := string(assetpath.AppendTransformed(nil, module+"/"+file, prefix))
	if got != want {
		return property("render-reference", "rendered reference %q, want %q", got, want)
	}

	service := registry_domain.NewRegistryService(registry_adapters.NewMockMetadataStore(), map[string]registry_domain.BlobStore{"memory": registry_adapters.NewMockBlobStore()}, nil, nil)
	key := strings.TrimPrefix(got, prefix+"/")
	twin := module + "/duplicate/" + file
	data := fmt.Sprintf("/* content %d */", choices.Uint(assetContentRange))
	write := func(artefactKey, content string) error {
		if _, err := service.UpsertArtefact(ctx, artefactKey, artefactKey, strings.NewReader(content), "memory", nil); err != nil {
			return fmt.Errorf("upserting asset %s: %w", artefactKey, err)
		}
		return nil
	}
	if err := write(key, data); err != nil {
		return err
	}
	if err := write(twin, data); err != nil {
		return err
	}
	mediaTypes := map[string]string{".js": "application/javascript", ".css": "text/css; charset=utf-8", ".svg": "image/svg+xml"}
	mediaType, ok := mediaTypes[filepath.Ext(file)]
	if !ok {
		return fmt.Errorf("no expected media type for %s", file)
	}
	first, err := assetContent(ctx, service, key, data, mediaType)
	if err != nil {
		return err
	}
	second, err := assetContent(ctx, service, twin, data, mediaType)
	if err != nil {
		return err
	}
	if first != second {
		return property("deduplication", "identical sources received different blob keys")
	}
	if err := write(key, data+" changed"); err != nil {
		return err
	}
	if _, err := assetContent(ctx, service, key, data+" changed", mediaType); err != nil {
		return err
	}
	if err := service.DeleteArtefact(ctx, key); err != nil {
		return fmt.Errorf("deleting asset %s: %w", key, err)
	}
	_, err = assetContent(ctx, service, twin, data, mediaType)
	return err
}

// assetContent reads an asset and checks its bytes and media type.
//
// Takes service (registry_domain.RegistryService) which is the production service under
// test.
// Takes key (string) which identifies the asset in the registry.
// Takes expected (string) which holds the independently expected result.
// Takes mediaType (string) which is the expected resource media type.
//
// Returns string which is the content-addressed asset identity.
// Returns error which reports an asset lookup, byte-content or media-type mismatch, or
// nil on success.
func assetContent(ctx context.Context, service registry_domain.RegistryService, key, expected, mediaType string) (string, error) {
	metadata, err := service.GetArtefact(ctx, key)
	if err != nil {
		return "", fmt.Errorf("reading asset metadata %s: %w", key, err)
	}
	for index := range metadata.ActualVariants {
		variant := &metadata.ActualVariants[index]
		if variant.VariantID != "source" {
			continue
		}
		if variant.MimeType != mediaType {
			return "", property("mime", "%s has MIME %q, want %q", key, variant.MimeType, mediaType)
		}
		if err := checkVariantContent(ctx, service, key, variant, expected); err != nil {
			return "", err
		}
		return variant.StorageKey, nil
	}
	return "", property("source-variant", "asset %s has no source variant", key)
}

// checkVariantContent reads a variant's bytes and compares them with the expected text.
//
// Takes service (registry_domain.RegistryService) which is the production service under
// test.
// Takes key (string) which identifies the asset in the registry.
// Takes variant (*registry_dto.Variant) which is the variant to read.
// Takes expected (string) which holds the independently expected content.
//
// Returns error which reports a read failure or content mismatch, or nil on success.
func checkVariantContent(ctx context.Context, service registry_domain.RegistryService, key string, variant *registry_dto.Variant, expected string) error {
	reader, err := service.GetVariantData(ctx, variant)
	if err != nil {
		return fmt.Errorf("opening asset %s: %w", key, err)
	}
	content, err := readLimited(reader, maxAssetBytes)
	closeErr := reader.Close()
	if errors.Is(err, errReadLimitExceeded) {
		return property("contents", "asset %s: %w", key, err)
	}
	if err != nil {
		return fmt.Errorf("reading asset %s: %w", key, err)
	}
	if closeErr != nil {
		return fmt.Errorf("closing asset %s: %w", key, closeErr)
	}
	if string(content) != expected {
		return property("contents", "asset %s has %q, want %q", key, content, expected)
	}
	return nil
}

// checkDocuments checks composed document transformations.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkDocuments(_ context.Context, choices *tape.Tape, _ *casefile.Case) error {
	x, y := drawInt(choices, canvasRange), drawInt(choices, canvasRange)
	var path strings.Builder
	fmt.Fprintf(&path, "M%d %d", x, y)
	segments := 1 + drawInt(choices, maxExtraSegments)
	want := make([]svg.PathCommand, 0, segments+1)
	want = append(want, svg.PathCommand{Type: 'M', Args: []float64{float64(x), float64(y)}})
	for range segments {
		dx := drawInt(choices, relativeStepRange) - relativeStepOffset
		dy := drawInt(choices, relativeStepRange) - relativeStepOffset
		fmt.Fprintf(&path, " l%d %d", dx, dy)
		x += dx
		y += dy
		want = append(want, svg.PathCommand{Type: 'L', Args: []float64{float64(x), float64(y)}})
	}
	got, err := svg.ParsePathData(path.String())
	if err != nil {
		return fmt.Errorf("parsing SVG path %s: %w", path.String(), err)
	}
	if !reflect.DeepEqual(got, want) {
		return property("relative-geometry", "document SVG path %s produced %+v, want %+v", path.String(), got, want)
	}

	tx := drawInt(choices, translationRange) - translationOffset
	ty := drawInt(choices, translationRange) - translationOffset
	sx := 1 + drawInt(choices, maxExtraScale)
	sy := 1 + drawInt(choices, maxExtraScale)
	angle := rightAngle * drawInt(choices, quarterTurns)
	order := choices.Uint(2)
	parts := []string{fmt.Sprintf("translate(%d,%d)", tx, ty), fmt.Sprintf("rotate(%d)", angle), fmt.Sprintf("scale(%d,%d)", sx, sy)}
	if order == 1 {
		parts[0], parts[2] = parts[2], parts[0]
	}
	transform := svg.ParseTransform(strings.Join(parts, " "))
	for _, point := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {x, y}} {
		a, b := point[0], point[1]
		if order == 0 {
			a, b = a*sx, b*sy
		} else {
			a, b = a+tx, b+ty
		}
		for range angle / rightAngle {
			a, b = -b, a
		}
		if order == 0 {
			a, b = a+tx, b+ty
		} else {
			a, b = a*sx, b*sy
		}
		gotX := transform.A*float64(point[0]) + transform.C*float64(point[1]) + transform.E
		gotY := transform.B*float64(point[0]) + transform.D*float64(point[1]) + transform.F
		if math.Abs(gotX-float64(a)) > transformTolerance || math.Abs(gotY-float64(b)) > transformTolerance {
			return property("svg-transform", "%v at %v got (%f,%f), want (%d,%d)", parts, point, gotX, gotY, a, b)
		}
	}
	return nil
}

// readLimited reads a body while detecting input beyond the limit.
//
// Takes reader (io.Reader) which supplies the body.
// Takes limit (int64) which is the largest accepted body length in bytes.
//
// Returns []byte which holds the complete body when it fits.
// Returns error which wraps errReadLimitExceeded when the body exceeds limit, or reports
// a read failure.
func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w: body exceeds %d bytes", errReadLimitExceeded, limit)
	}
	return data, nil
}
