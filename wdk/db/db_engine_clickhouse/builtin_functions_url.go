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

package db_engine_clickhouse

import (
	"strings"
)

var (
	// significantSubdomainVariantSuffixes lists the optional suffix modifiers ClickHouse
	// exposes on cutToFirstSignificantSubdomain and firstSignificantSubdomain (Custom /
	// CustomRFC / CustomWithWWW / CustomWithWWWRFC / RFC / WithWWW / WithWWWRFC). The
	// catalogue iterates over this list to register every variant without spelling each
	// entry individually.
	significantSubdomainVariantSuffixes = []string{
		"",
		"Custom",
		"CustomRFC",
		"CustomWithWWW",
		"CustomWithWWWRFC",
		"RFC",
		"WithWWW",
		"WithWWWRFC",
	}

	// firstSignificantSubdomainVariantSuffixes lists the optional suffix modifiers
	// ClickHouse exposes on firstSignificantSubdomain. The set is smaller than
	// cutToFirstSignificantSubdomain because the bare firstSignificantSubdomain family does
	// not include the WithWWW derivatives.
	firstSignificantSubdomainVariantSuffixes = []string{
		"",
		"Custom",
		"CustomRFC",
		"RFC",
	}
)

// registerExtendedURLFunctions covers the long tail of URL parsing helpers (cut*,
// decode*, encode*, extractURL*, firstSignificant*, netloc, port, queryStringAndFragment,
// URLHierarchy and the RFC domain helpers).
//
// Splitting by topical group keeps each helper function within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered URL functions.
func registerExtendedURLFunctions(b *FunctionCatalogueBuilder) {
	registerCutAndDecodeURLHelpers(b)
	registerCutToFirstSignificantSubdomainVariants(b)
	registerURLParameterAndWWWHelpers(b)
	registerEncodeURLHelpers(b)
	registerExtractURLParameters(b)
	registerFirstSignificantSubdomainVariants(b)
	registerNetlocAndPortHelpers(b)
	registerURLHierarchyAndDomainRFCHelpers(b)
}

// registerCutAndDecodeURLHelpers covers cutQueryString, cutFragment,
// cutQueryStringAndFragment and the decode helpers.
//
// The decode helpers transform percent-encoded sequences back into their original
// characters; the form variant additionally treats '+' as a space.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerCutAndDecodeURLHelpers(b *FunctionCatalogueBuilder) {
	b.Register("cutQueryString", b.textType, b.textType)
	b.Register("cutFragment", b.textType, b.textType)
	b.Register("cutQueryStringAndFragment", b.textType, b.textType)
	b.Register("decodeURLComponent", b.textType, b.textType)
	b.Register("decodeURLFormComponent", b.textType, b.textType)
}

// registerCutToFirstSignificantSubdomainVariants registers every suffix variant of
// cutToFirstSignificantSubdomain.
//
// The Custom variants accept an additional public-suffix list dictionary name, which the
// catalogue records as a Dynamic placeholder so callers can omit it for the bare form.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered variant functions.
func registerCutToFirstSignificantSubdomainVariants(b *FunctionCatalogueBuilder) {
	for _, suffix := range significantSubdomainVariantSuffixes {
		b.Register("cutToFirstSignificantSubdomain"+suffix, b.textType, b.textType)

		if strings.Contains(suffix, "Custom") {
			b.Register("cutToFirstSignificantSubdomain"+suffix, b.textType, b.textType, b.textType)
		}
	}
}

// registerURLParameterAndWWWHelpers covers cutURLParameter and cutWWW.
//
// cutURLParameter removes a named query parameter from a URL; cutWWW strips the leading
// "www." prefix from a hostname.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerURLParameterAndWWWHelpers(b *FunctionCatalogueBuilder) {
	b.Register("cutURLParameter", b.textType, b.textType, b.textType)
	b.Register("cutWWW", b.textType, b.textType)
}

// registerEncodeURLHelpers covers encodeURLComponent and encodeURLFormComponent.
//
// These percent-encode characters that are reserved in URLs; the form variant adds
// encoding of spaces as '+' for application/x-www-form-urlencoded payloads.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerEncodeURLHelpers(b *FunctionCatalogueBuilder) {
	b.Register("encodeURLComponent", b.textType, b.textType)
	b.Register("encodeURLFormComponent", b.textType, b.textType)
}

// registerExtractURLParameters covers extractURLParameters and extractURLParameterNames.
//
// extractURLParameters returns the full key=value strings as an Array;
// extractURLParameterNames returns just the names.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerExtractURLParameters(b *FunctionCatalogueBuilder) {
	b.Register("extractURLParameters", arrayOf(b.textType), b.textType)
	b.Register("extractURLParameterNames", arrayOf(b.textType), b.textType)
}

// registerFirstSignificantSubdomainVariants registers every suffix variant of
// firstSignificantSubdomain.
//
// The Custom forms accept the optional public-suffix list dictionary name as a second
// argument.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered variant functions.
func registerFirstSignificantSubdomainVariants(b *FunctionCatalogueBuilder) {
	for _, suffix := range firstSignificantSubdomainVariantSuffixes {
		b.Register("firstSignificantSubdomain"+suffix, b.textType, b.textType)
	}
}

// registerNetlocAndPortHelpers covers netloc, port (with its RFC variant) and
// queryStringAndFragment.
//
// netloc returns the user:password@host:port section of a URL; queryStringAndFragment
// returns the part after the path including both query and fragment.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerNetlocAndPortHelpers(b *FunctionCatalogueBuilder) {
	b.Register("netloc", b.textType, b.textType)

	b.Register("port", b.uint64Type, b.textType)
	b.Register("port", b.uint64Type, b.textType, b.uint64Type)
	b.Register("portRFC", b.uint64Type, b.textType)
	b.Register("portRFC", b.uint64Type, b.textType, b.uint64Type)
	b.Register("queryStringAndFragment", b.textType, b.textType)
}

// registerURLHierarchyAndDomainRFCHelpers covers URLHierarchy / URLPathHierarchy (which
// return the cumulative path prefixes as an Array) and the RFC variants of domain,
// domainWithoutWWW and topLevelDomain.
//
// Takes b (*FunctionCatalogueBuilder) which receives the registered helper functions.
func registerURLHierarchyAndDomainRFCHelpers(b *FunctionCatalogueBuilder) {
	b.Register("URLHierarchy", arrayOf(b.textType), b.textType)
	b.Register("URLPathHierarchy", arrayOf(b.textType), b.textType)
	b.Register("domainRFC", b.textType, b.textType)
	b.Register("domainWithoutWWWRFC", b.textType, b.textType)
	b.Register("topLevelDomainRFC", b.textType, b.textType)
}
