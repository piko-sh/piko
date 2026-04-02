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

//go:build js && wasm

// WASM-specific symbol registration for piko.sh/piko.
//
// The generated gen_piko.sh_piko.go imports the root piko.sh/piko facade which
// transitively pulls in the entire server (bootstrap, daemon, email, LLM,
// monitoring, orchestrator, WAL, cache, etc.) - adding ~157 unnecessary
// packages and ~40MB to the WASM binary.
//
// This file registers the same symbols by importing directly from internal
// packages that are already in the WASM dependency graph. Go type aliases make
// this transparent: reflect.ValueOf((*templater_dto.RequestData)(nil))
// registered under "piko.sh/piko"/"RequestData" is identical to
// reflect.ValueOf((*piko.RequestData)(nil)) per the Go spec.
//
// Symbols from packages NOT in the WASM dep graph (daemon_domain, bootstrap,
// daemon_adapters, daemon_frontend, safeerror, pikotest) are either omitted or
// redefined locally when the underlying type is trivial (e.g. string).

package driven_piko_symbols

import (
	"log/slog"
	"reflect"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/runtime"
)

// httpMethod mirrors daemon_domain.HTTPMethod to avoid importing daemon_domain
// which pulls in 609 transitive dependencies. Must be a named type (not alias)
// so that reflect.TypeOf gives a distinct type - an alias to string would
// poison the type synthesiser's seen map, causing all string parameters to
// resolve as piko.HTTPMethod.
type httpMethod string

const (
	// methodGet represents the HTTP GET method.
	methodGet httpMethod = "GET"

	// methodHead represents the HTTP HEAD method.
	methodHead httpMethod = "HEAD"

	// methodPost represents the HTTP POST method.
	methodPost httpMethod = "POST"

	// methodPut represents the HTTP PUT method.
	methodPut httpMethod = "PUT"

	// methodDelete represents the HTTP DELETE method.
	methodDelete httpMethod = "DELETE"

	// methodOptions represents the HTTP OPTIONS method.
	methodOptions httpMethod = "OPTIONS"

	// methodPatch represents the HTTP PATCH method.
	methodPatch httpMethod = "PATCH"
)

// transport mirrors daemon_domain.Transport. Named type for the same reason
// as httpMethod above.
type transport string

const (
	// transportHTTP represents the HTTP transport protocol.
	transportHTTP transport = "http"

	// transportSSE represents the Server-Sent Events transport protocol.
	transportSSE transport = "sse"
)

// version mirrors piko.Version. In WASM context the exact value is
// informational only; user code rarely depends on it.
var version = "0.1.0-alpha"

func init() {
	Symbols["piko.sh/piko"] = map[string]reflect.Value{

		"RequestData": reflect.ValueOf((*templater_dto.RequestData)(nil)),
		"Metadata":    reflect.ValueOf((*templater_dto.Metadata)(nil)),
		"OGTag":       reflect.ValueOf((*templater_dto.OGTag)(nil)),
		"MetaTag":     reflect.ValueOf((*templater_dto.MetaTag)(nil)),
		"NoProps":     reflect.ValueOf((*templater_dto.NoProps)(nil)),
		"NoResponse":  reflect.ValueOf((*templater_dto.NoResponse)(nil)),
		"CachePolicy": reflect.ValueOf((*templater_dto.CachePolicy)(nil)),

		"CookieOption":    reflect.ValueOf((*daemon_dto.CookieOption)(nil)),
		"ActionMetadata":  reflect.ValueOf((*daemon_dto.ActionMetadata)(nil)),
		"RequestMetadata": reflect.ValueOf((*daemon_dto.RequestMetadata)(nil)),
		"ResponseWriter":  reflect.ValueOf((*daemon_dto.ResponseWriter)(nil)),
		"Session":         reflect.ValueOf((*daemon_dto.Session)(nil)),
		"HelperCall":      reflect.ValueOf((*daemon_dto.HelperCall)(nil)),
		"FileUpload":      reflect.ValueOf((*daemon_dto.FileUpload)(nil)),
		"RawBody":         reflect.ValueOf((*daemon_dto.RawBody)(nil)),

		"ActionError":       reflect.ValueOf((*daemon_dto.ActionError)(nil)),
		"ValidationError":   reflect.ValueOf((*daemon_dto.ValidationError)(nil)),
		"NotFoundError":     reflect.ValueOf((*daemon_dto.NotFoundError)(nil)),
		"ConflictError":     reflect.ValueOf((*daemon_dto.ConflictError)(nil)),
		"ForbiddenError":    reflect.ValueOf((*daemon_dto.ForbiddenError)(nil)),
		"UnauthorisedError": reflect.ValueOf((*daemon_dto.UnauthorisedError)(nil)),
		"BadRequestError":   reflect.ValueOf((*daemon_dto.BadRequestError)(nil)),
		"GenericPageError":  reflect.ValueOf((*daemon_dto.GenericPageError)(nil)),
		"TeapotError":       reflect.ValueOf((*daemon_dto.TeapotError)(nil)),
		"ErrorPageContext":  reflect.ValueOf((*daemon_dto.ErrorPageContext)(nil)),

		"BadRequest":       reflect.ValueOf(daemon_dto.BadRequest),
		"NotFound":         reflect.ValueOf(daemon_dto.NotFound),
		"NotFoundResource": reflect.ValueOf(daemon_dto.NotFoundResource),
		"Forbidden":        reflect.ValueOf(daemon_dto.Forbidden),
		"Unauthorised":     reflect.ValueOf(daemon_dto.Unauthorised),
		"Conflict":         reflect.ValueOf(daemon_dto.Conflict),
		"ConflictWithCode": reflect.ValueOf(daemon_dto.ConflictWithCode),
		"Teapot":           reflect.ValueOf(daemon_dto.Teapot),
		"PageError":        reflect.ValueOf(daemon_dto.PageError),

		"Cookie":                reflect.ValueOf(daemon_dto.Cookie),
		"SessionCookie":         reflect.ValueOf(daemon_dto.SessionCookie),
		"SessionCookieInsecure": reflect.ValueOf(daemon_dto.SessionCookieInsecure),
		"ClearCookie":           reflect.ValueOf(daemon_dto.ClearCookie),
		"ClearCookieInsecure":   reflect.ValueOf(daemon_dto.ClearCookieInsecure),
		"WithPath":              reflect.ValueOf(daemon_dto.WithPath),
		"WithDomain":            reflect.ValueOf(daemon_dto.WithDomain),
		"WithInsecure":          reflect.ValueOf(daemon_dto.WithInsecure),
		"WithJavaScriptAccess":  reflect.ValueOf(daemon_dto.WithJavaScriptAccess),
		"WithSameSiteStrict":    reflect.ValueOf(daemon_dto.WithSameSiteStrict),
		"WithSameSiteNone":      reflect.ValueOf(daemon_dto.WithSameSiteNone),

		"NewFileUpload":      reflect.ValueOf(daemon_dto.NewFileUpload),
		"NewRawBody":         reflect.ValueOf(daemon_dto.NewRawBody),
		"NewValidationError": reflect.ValueOf(daemon_dto.NewValidationError),
		"ValidationField":    reflect.ValueOf(daemon_dto.ValidationField),

		"GetErrorContext": reflect.ValueOf(daemon_dto.GetErrorPageContext),

		"HTTPMethod":    reflect.ValueOf((*httpMethod)(nil)),
		"MethodGet":     reflect.ValueOf(methodGet),
		"MethodHead":    reflect.ValueOf(methodHead),
		"MethodPost":    reflect.ValueOf(methodPost),
		"MethodPut":     reflect.ValueOf(methodPut),
		"MethodDelete":  reflect.ValueOf(methodDelete),
		"MethodOptions": reflect.ValueOf(methodOptions),
		"MethodPatch":   reflect.ValueOf(methodPatch),

		"Transport":     reflect.ValueOf((*transport)(nil)),
		"TransportHTTP": reflect.ValueOf(transportHTTP),
		"TransportSSE":  reflect.ValueOf(transportSSE),

		"Filter":            reflect.ValueOf((*runtime.Filter)(nil)),
		"FilterGroup":       reflect.ValueOf((*runtime.FilterGroup)(nil)),
		"FilterOperator":    reflect.ValueOf((*runtime.FilterOperator)(nil)),
		"SortOption":        reflect.ValueOf((*runtime.SortOption)(nil)),
		"SortOrder":         reflect.ValueOf((*runtime.SortOrder)(nil)),
		"PaginationOptions": reflect.ValueOf((*runtime.PaginationOptions)(nil)),
		"Section":           reflect.ValueOf((*runtime.Section)(nil)),
		"SectionNode":       reflect.ValueOf((*runtime.SectionNode)(nil)),
		"SectionTreeOption": reflect.ValueOf((*runtime.SectionTreeOption)(nil)),
		"NavigationGroups":  reflect.ValueOf((*runtime.NavigationGroups)(nil)),
		"NavigationTree":    reflect.ValueOf((*runtime.NavigationTree)(nil)),
		"NavigationNode":    reflect.ValueOf((*runtime.NavigationNode)(nil)),
		"NavigationConfig":  reflect.ValueOf((*runtime.NavigationConfig)(nil)),
		"SearchField":       reflect.ValueOf((*runtime.SearchField)(nil)),
		"SearchOption":      reflect.ValueOf((*runtime.SearchOption)(nil)),

		"FilterOpEquals":       reflect.ValueOf(runtime.FilterOpEquals),
		"FilterOpNotEquals":    reflect.ValueOf(runtime.FilterOpNotEquals),
		"FilterOpGreaterThan":  reflect.ValueOf(runtime.FilterOpGreaterThan),
		"FilterOpGreaterEqual": reflect.ValueOf(runtime.FilterOpGreaterEqual),
		"FilterOpLessThan":     reflect.ValueOf(runtime.FilterOpLessThan),
		"FilterOpLessEqual":    reflect.ValueOf(runtime.FilterOpLessEqual),
		"FilterOpContains":     reflect.ValueOf(runtime.FilterOpContains),
		"FilterOpStartsWith":   reflect.ValueOf(runtime.FilterOpStartsWith),
		"FilterOpEndsWith":     reflect.ValueOf(runtime.FilterOpEndsWith),
		"FilterOpIn":           reflect.ValueOf(runtime.FilterOpIn),
		"FilterOpNotIn":        reflect.ValueOf(runtime.FilterOpNotIn),
		"FilterOpExists":       reflect.ValueOf(runtime.FilterOpExists),
		"FilterOpFuzzyMatch":   reflect.ValueOf(runtime.FilterOpFuzzyMatch),
		"SortAsc":              reflect.ValueOf(runtime.SortAsc),
		"SortDesc":             reflect.ValueOf(runtime.SortDesc),

		"NewFilter":                   reflect.ValueOf(runtime.NewFilter),
		"And":                         reflect.ValueOf(runtime.And),
		"Or":                          reflect.ValueOf(runtime.Or),
		"NewSortOption":               reflect.ValueOf(runtime.NewSortOption),
		"NewPaginationOptions":        reflect.ValueOf(runtime.NewPaginationOptions),
		"GetAllCollectionItems":       reflect.ValueOf(runtime.GetAllCollectionItems),
		"GetSections":                 reflect.ValueOf(runtime.GetSections),
		"GetSectionsTree":             reflect.ValueOf(runtime.GetSectionsTree),
		"BuildNavigationFromMetadata": reflect.ValueOf(runtime.BuildNavigationFromMetadata),
		"DefaultNavigationConfig":     reflect.ValueOf(runtime.DefaultNavigationConfig),

		"WithSearchFields":   reflect.ValueOf(runtime.WithSearchFields),
		"WithFuzzyThreshold": reflect.ValueOf(runtime.WithFuzzyThreshold),
		"WithSearchLimit":    reflect.ValueOf(runtime.WithSearchLimit),
		"WithSearchOffset":   reflect.ValueOf(runtime.WithSearchOffset),
		"WithMinScore":       reflect.ValueOf(runtime.WithMinScore),
		"WithCaseSensitive":  reflect.ValueOf(runtime.WithCaseSensitive),
		"WithSearchMode":     reflect.ValueOf(runtime.WithSearchMode),
		"WithMinLevel":       reflect.ValueOf(runtime.WithMinLevel),
		"WithMaxLevel":       reflect.ValueOf(runtime.WithMaxLevel),

		"CSPBuilder":    reflect.ValueOf((*security_domain.CSPBuilder)(nil)),
		"NewCSPBuilder": reflect.ValueOf(security_domain.NewCSPBuilder),

		"CSPSelf":            reflect.ValueOf(security_domain.Self),
		"CSPNone":            reflect.ValueOf(security_domain.None),
		"CSPUnsafeInline":    reflect.ValueOf(security_domain.UnsafeInline),
		"CSPUnsafeEval":      reflect.ValueOf(security_domain.UnsafeEval),
		"CSPUnsafeHashes":    reflect.ValueOf(security_domain.UnsafeHashes),
		"CSPStrictDynamic":   reflect.ValueOf(security_domain.StrictDynamic),
		"CSPReportSample":    reflect.ValueOf(security_domain.ReportSample),
		"CSPWasmUnsafeEval":  reflect.ValueOf(security_domain.WasmUnsafeEval),
		"CSPData":            reflect.ValueOf(security_domain.Data),
		"CSPBlob":            reflect.ValueOf(security_domain.Blob),
		"CSPHTTPS":           reflect.ValueOf(security_domain.HTTPS),
		"CSPHTTP":            reflect.ValueOf(security_domain.HTTP),
		"CSPRequestToken":    reflect.ValueOf(security_domain.RequestTokenPlaceholder),
		"CSPHost":            reflect.ValueOf(security_domain.Host),
		"CSPScheme":          reflect.ValueOf(security_domain.Scheme),
		"CSPSHA256":          reflect.ValueOf(security_domain.SHA256),
		"CSPSHA384":          reflect.ValueOf(security_domain.SHA384),
		"CSPSHA512":          reflect.ValueOf(security_domain.SHA512),
		"CSPStaticToken":     reflect.ValueOf(security_domain.RequestToken),
		"CSPPolicyName":      reflect.ValueOf(security_domain.PolicyName),
		"CSPScript":          reflect.ValueOf(security_domain.Script),
		"CSPAllowDuplicates": reflect.ValueOf(security_domain.AllowDuplicates),
		"CSPWildcard":        reflect.ValueOf(security_domain.Wildcard),

		"CSPSandboxAllowDownloads":                      reflect.ValueOf(security_domain.SandboxAllowDownloads),
		"CSPSandboxAllowForms":                          reflect.ValueOf(security_domain.SandboxAllowForms),
		"CSPSandboxAllowModals":                         reflect.ValueOf(security_domain.SandboxAllowModals),
		"CSPSandboxAllowOrientationLock":                reflect.ValueOf(security_domain.SandboxAllowOrientationLock),
		"CSPSandboxAllowPointerLock":                    reflect.ValueOf(security_domain.SandboxAllowPointerLock),
		"CSPSandboxAllowPopups":                         reflect.ValueOf(security_domain.SandboxAllowPopups),
		"CSPSandboxAllowPopupsToEscapeSandbox":          reflect.ValueOf(security_domain.SandboxAllowPopupsToEscapeSandbox),
		"CSPSandboxAllowPresentation":                   reflect.ValueOf(security_domain.SandboxAllowPresentation),
		"CSPSandboxAllowSameOrigin":                     reflect.ValueOf(security_domain.SandboxAllowSameOrigin),
		"CSPSandboxAllowScripts":                        reflect.ValueOf(security_domain.SandboxAllowScripts),
		"CSPSandboxAllowStorageAccessByUserActivation":  reflect.ValueOf(security_domain.SandboxAllowStorageAccessByUserActivation),
		"CSPSandboxAllowTopNavigation":                  reflect.ValueOf(security_domain.SandboxAllowTopNavigation),
		"CSPSandboxAllowTopNavigationByUserActivation":  reflect.ValueOf(security_domain.SandboxAllowTopNavigationByUserActivation),
		"CSPSandboxAllowTopNavigationToCustomProtocols": reflect.ValueOf(security_domain.SandboxAllowTopNavigationToCustomProtocols),

		"Translation":        reflect.ValueOf((*i18n_domain.Translation)(nil)),
		"GenerateLocaleHead": reflect.ValueOf(runtime.GenerateLocaleHead),

		"LevelDebug": reflect.ValueOf(slog.LevelDebug),
		"LevelInfo":  reflect.ValueOf(slog.LevelInfo),
		"LevelWarn":  reflect.ValueOf(slog.LevelWarn),
		"LevelError": reflect.ValueOf(slog.LevelError),

		"Version": reflect.ValueOf(&version).Elem(),
	}
}
