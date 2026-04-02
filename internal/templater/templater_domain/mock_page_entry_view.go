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

package templater_domain

import (
	"net/http"
	"sync/atomic"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

// MockPageEntryView is a test double for PageEntryView that returns
// zero values from nil function fields and tracks call counts
// atomically.
type MockPageEntryView struct {
	// GetHasMiddlewareFunc is the function called by
	// GetHasMiddleware.
	GetHasMiddlewareFunc func() bool

	// GetMiddlewareFuncNameFunc is the function called
	// by GetMiddlewareFuncName.
	GetMiddlewareFuncNameFunc func() string

	// GetHasCachePolicyFunc is the function called by
	// GetHasCachePolicy.
	GetHasCachePolicyFunc func() bool

	// GetCachePolicyFunc is the function called by
	// GetCachePolicy.
	GetCachePolicyFunc func(r *templater_dto.RequestData) templater_dto.CachePolicy

	// GetCachePolicyFuncNameFunc is the function called
	// by GetCachePolicyFuncName.
	GetCachePolicyFuncNameFunc func() string

	// GetMiddlewaresFunc is the function called by
	// GetMiddlewares.
	GetMiddlewaresFunc func() []func(http.Handler) http.Handler

	// GetIsPageFunc is the function called by
	// GetIsPage.
	GetIsPageFunc func() bool

	// GetRoutePatternFunc is the function called by
	// GetRoutePattern.
	GetRoutePatternFunc func() string

	// GetRoutePatternsFunc is the function called by
	// GetRoutePatterns.
	GetRoutePatternsFunc func() map[string]string

	// GetI18nStrategyFunc is the function called by
	// GetI18nStrategy.
	GetI18nStrategyFunc func() string

	// GetOriginalPathFunc is the function called by
	// GetOriginalPath.
	GetOriginalPathFunc func() string

	// GetASTRootFunc is the function called by
	// GetASTRoot.
	GetASTRootFunc func(r *templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata)

	// GetASTRootWithPropsFunc is the function called by
	// GetASTRootWithProps.
	GetASTRootWithPropsFunc func(r *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata)

	// GetStylingFunc is the function called by
	// GetStyling.
	GetStylingFunc func() string

	// GetAssetRefsFunc is the function called by
	// GetAssetRefs.
	GetAssetRefsFunc func() []templater_dto.AssetRef

	// GetCustomTagsFunc is the function called by
	// GetCustomTags.
	GetCustomTagsFunc func() []string

	// GetSupportedLocalesFunc is the function called by
	// GetSupportedLocales.
	GetSupportedLocalesFunc func() []string

	// GetLocalStoreFunc is the function called by
	// GetLocalStore.
	GetLocalStoreFunc func() *i18n_domain.Store

	// GetJSScriptMetasFunc is the function called by
	// GetJSScriptMetas.
	GetJSScriptMetasFunc func() []templater_dto.JSScriptMeta

	// GetIsE2EOnlyFunc is the function called by
	// GetIsE2EOnly.
	GetIsE2EOnlyFunc func() bool

	// GetStaticMetadataFunc is the function called by
	// GetStaticMetadata.
	GetStaticMetadataFunc func() *templater_dto.InternalMetadata

	// GetHasAuthPolicyFunc is the function called by GetHasAuthPolicy.
	GetHasAuthPolicyFunc func() bool

	// GetAuthPolicyFunc is the function called by GetAuthPolicy.
	GetAuthPolicyFunc func(r *templater_dto.RequestData) daemon_dto.AuthPolicy

	// GetHasPreviewFunc is the function called by GetHasPreview.
	GetHasPreviewFunc func() bool

	// GetPreviewScenariosFunc is the function called by GetPreviewScenarios.
	GetPreviewScenariosFunc func() []templater_dto.PreviewScenario

	// GetHasMiddlewareCallCount tracks how many times
	// GetHasMiddleware was called.
	GetHasMiddlewareCallCount int64

	// GetMiddlewareFuncNameCallCount tracks how many
	// times GetMiddlewareFuncName was called.
	GetMiddlewareFuncNameCallCount int64

	// GetHasCachePolicyCallCount tracks how many times
	// GetHasCachePolicy was called.
	GetHasCachePolicyCallCount int64

	// GetCachePolicyCallCount tracks how many times
	// GetCachePolicy was called.
	GetCachePolicyCallCount int64

	// GetCachePolicyFuncNameCallCount tracks how many
	// times GetCachePolicyFuncName was called.
	GetCachePolicyFuncNameCallCount int64

	// GetMiddlewaresCallCount tracks how many times
	// GetMiddlewares was called.
	GetMiddlewaresCallCount int64

	// GetIsPageCallCount tracks how many times
	// GetIsPage was called.
	GetIsPageCallCount int64

	// GetRoutePatternCallCount tracks how many times
	// GetRoutePattern was called.
	GetRoutePatternCallCount int64

	// GetRoutePatternsCallCount tracks how many times
	// GetRoutePatterns was called.
	GetRoutePatternsCallCount int64

	// GetI18nStrategyCallCount tracks how many times
	// GetI18nStrategy was called.
	GetI18nStrategyCallCount int64

	// GetOriginalPathCallCount tracks how many times
	// GetOriginalPath was called.
	GetOriginalPathCallCount int64

	// GetASTRootCallCount tracks how many times
	// GetASTRoot was called.
	GetASTRootCallCount int64

	// GetASTRootWithPropsCallCount tracks how many
	// times GetASTRootWithProps was called.
	GetASTRootWithPropsCallCount int64

	// GetStylingCallCount tracks how many times
	// GetStyling was called.
	GetStylingCallCount int64

	// GetAssetRefsCallCount tracks how many times
	// GetAssetRefs was called.
	GetAssetRefsCallCount int64

	// GetCustomTagsCallCount tracks how many times
	// GetCustomTags was called.
	GetCustomTagsCallCount int64

	// GetSupportedLocalesCallCount tracks how many
	// times GetSupportedLocales was called.
	GetSupportedLocalesCallCount int64

	// GetLocalStoreCallCount tracks how many times
	// GetLocalStore was called.
	GetLocalStoreCallCount int64

	// GetJSScriptMetasCallCount tracks how many times
	// GetJSScriptMetas was called.
	GetJSScriptMetasCallCount int64

	// GetIsE2EOnlyCallCount tracks how many times
	// GetIsE2EOnly was called.
	GetIsE2EOnlyCallCount int64

	// GetStaticMetadataCallCount tracks how many times
	// GetStaticMetadata was called.
	GetStaticMetadataCallCount int64

	// GetHasAuthPolicyCallCount tracks how many times
	// GetHasAuthPolicy was called.
	GetHasAuthPolicyCallCount int64

	// GetAuthPolicyCallCount tracks how many times
	// GetAuthPolicy was called.
	GetAuthPolicyCallCount int64

	// GetHasPreviewCallCount tracks how many times
	// GetHasPreview was called.
	GetHasPreviewCallCount int64

	// GetPreviewScenariosCallCount tracks how many times
	// GetPreviewScenarios was called.
	GetPreviewScenariosCallCount int64
}

var _ PageEntryView = (*MockPageEntryView)(nil)

// GetHasMiddleware reports whether the handler has middleware attached.
//
// Returns bool, or false if GetHasMiddlewareFunc is nil.
func (m *MockPageEntryView) GetHasMiddleware() bool {
	atomic.AddInt64(&m.GetHasMiddlewareCallCount, 1)
	if m.GetHasMiddlewareFunc != nil {
		return m.GetHasMiddlewareFunc()
	}
	return false
}

// GetMiddlewareFuncName returns the name of the middleware function.
//
// Returns string, or "" if GetMiddlewareFuncNameFunc is nil.
func (m *MockPageEntryView) GetMiddlewareFuncName() string {
	atomic.AddInt64(&m.GetMiddlewareFuncNameCallCount, 1)
	if m.GetMiddlewareFuncNameFunc != nil {
		return m.GetMiddlewareFuncNameFunc()
	}
	return ""
}

// GetHasCachePolicy reports whether the directive has a cache policy set.
//
// Returns bool, or false if GetHasCachePolicyFunc is nil.
func (m *MockPageEntryView) GetHasCachePolicy() bool {
	atomic.AddInt64(&m.GetHasCachePolicyCallCount, 1)
	if m.GetHasCachePolicyFunc != nil {
		return m.GetHasCachePolicyFunc()
	}
	return false
}

// GetCachePolicy returns the cache policy for the given request.
//
// Takes r (*templater_dto.RequestData) which is the
// request data to evaluate the policy for.
//
// Returns CachePolicy, or zero value if GetCachePolicyFunc is nil.
func (m *MockPageEntryView) GetCachePolicy(r *templater_dto.RequestData) templater_dto.CachePolicy {
	atomic.AddInt64(&m.GetCachePolicyCallCount, 1)
	if m.GetCachePolicyFunc != nil {
		return m.GetCachePolicyFunc(r)
	}
	return templater_dto.CachePolicy{}
}

// GetCachePolicyFuncName returns the name of the cache policy function.
//
// Returns string, or "" if GetCachePolicyFuncNameFunc is nil.
func (m *MockPageEntryView) GetCachePolicyFuncName() string {
	atomic.AddInt64(&m.GetCachePolicyFuncNameCallCount, 1)
	if m.GetCachePolicyFuncNameFunc != nil {
		return m.GetCachePolicyFuncNameFunc()
	}
	return ""
}

// GetMiddlewares returns the middleware chain for the router.
//
// Returns []func(http.Handler) http.Handler, or nil if GetMiddlewaresFunc
// is nil.
func (m *MockPageEntryView) GetMiddlewares() []func(http.Handler) http.Handler {
	atomic.AddInt64(&m.GetMiddlewaresCallCount, 1)
	if m.GetMiddlewaresFunc != nil {
		return m.GetMiddlewaresFunc()
	}
	return nil
}

// GetIsPage reports whether the element represents a page.
//
// Returns bool, or false if GetIsPageFunc is nil.
func (m *MockPageEntryView) GetIsPage() bool {
	atomic.AddInt64(&m.GetIsPageCallCount, 1)
	if m.GetIsPageFunc != nil {
		return m.GetIsPageFunc()
	}
	return false
}

// GetRoutePattern returns the primary route pattern for this entry.
//
// Returns string, or "" if GetRoutePatternFunc is nil.
func (m *MockPageEntryView) GetRoutePattern() string {
	atomic.AddInt64(&m.GetRoutePatternCallCount, 1)
	if m.GetRoutePatternFunc != nil {
		return m.GetRoutePatternFunc()
	}
	return ""
}

// GetRoutePatterns returns the route patterns for the HTTP handler.
//
// Returns map[string]string, or nil if GetRoutePatternsFunc is nil.
func (m *MockPageEntryView) GetRoutePatterns() map[string]string {
	atomic.AddInt64(&m.GetRoutePatternsCallCount, 1)
	if m.GetRoutePatternsFunc != nil {
		return m.GetRoutePatternsFunc()
	}
	return nil
}

// GetI18nStrategy returns the internationalisation strategy identifier.
//
// Returns string, or "" if GetI18nStrategyFunc is nil.
func (m *MockPageEntryView) GetI18nStrategy() string {
	atomic.AddInt64(&m.GetI18nStrategyCallCount, 1)
	if m.GetI18nStrategyFunc != nil {
		return m.GetI18nStrategyFunc()
	}
	return ""
}

// GetOriginalPath returns the original file path before any changes.
//
// Returns string, or "" if GetOriginalPathFunc is nil.
func (m *MockPageEntryView) GetOriginalPath() string {
	atomic.AddInt64(&m.GetOriginalPathCallCount, 1)
	if m.GetOriginalPathFunc != nil {
		return m.GetOriginalPathFunc()
	}
	return ""
}

// GetASTRoot retrieves the root AST node for the given request data.
//
// Takes r (*templater_dto.RequestData) which provides
// the request context for AST generation.
//
// Returns (*TemplateAST, InternalMetadata), or
// (nil, InternalMetadata{}) if GetASTRootFunc is nil.
func (m *MockPageEntryView) GetASTRoot(r *templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
	atomic.AddInt64(&m.GetASTRootCallCount, 1)
	if m.GetASTRootFunc != nil {
		return m.GetASTRootFunc(r)
	}
	return nil, templater_dto.InternalMetadata{}
}

// GetASTRootWithProps returns the AST root with props for email rendering.
//
// Takes r (*templater_dto.RequestData) which provides
// the request context for AST generation.
// Takes props (any) which contains the properties to
// pass to the template.
//
// Returns (*TemplateAST, InternalMetadata), or (nil, InternalMetadata{}) if
// GetASTRootWithPropsFunc is nil.
func (m *MockPageEntryView) GetASTRootWithProps(r *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
	atomic.AddInt64(&m.GetASTRootWithPropsCallCount, 1)
	if m.GetASTRootWithPropsFunc != nil {
		return m.GetASTRootWithPropsFunc(r, props)
	}
	return nil, templater_dto.InternalMetadata{}
}

// GetStyling returns the styling settings as a string.
//
// Returns string, or "" if GetStylingFunc is nil.
func (m *MockPageEntryView) GetStyling() string {
	atomic.AddInt64(&m.GetStylingCallCount, 1)
	if m.GetStylingFunc != nil {
		return m.GetStylingFunc()
	}
	return ""
}

// GetAssetRefs returns the asset references associated with this entry.
//
// Returns []AssetRef, or nil if GetAssetRefsFunc is nil.
func (m *MockPageEntryView) GetAssetRefs() []templater_dto.AssetRef {
	atomic.AddInt64(&m.GetAssetRefsCallCount, 1)
	if m.GetAssetRefsFunc != nil {
		return m.GetAssetRefsFunc()
	}
	return nil
}

// GetCustomTags returns the list of custom tags for this entry.
//
// Returns []string, or nil if GetCustomTagsFunc is nil.
func (m *MockPageEntryView) GetCustomTags() []string {
	atomic.AddInt64(&m.GetCustomTagsCallCount, 1)
	if m.GetCustomTagsFunc != nil {
		return m.GetCustomTagsFunc()
	}
	return nil
}

// GetSupportedLocales returns the locale codes this entry supports.
//
// Returns []string, or nil if GetSupportedLocalesFunc is nil.
func (m *MockPageEntryView) GetSupportedLocales() []string {
	atomic.AddInt64(&m.GetSupportedLocalesCallCount, 1)
	if m.GetSupportedLocalesFunc != nil {
		return m.GetSupportedLocalesFunc()
	}
	return nil
}

// GetLocalStore returns the pre-built translation store for this page.
//
// Returns *Store, or nil if GetLocalStoreFunc is nil.
func (m *MockPageEntryView) GetLocalStore() *i18n_domain.Store {
	atomic.AddInt64(&m.GetLocalStoreCallCount, 1)
	if m.GetLocalStoreFunc != nil {
		return m.GetLocalStoreFunc()
	}
	return nil
}

// GetJSScriptMetas returns metadata for client-side JavaScript modules.
//
// Returns []JSScriptMeta, or nil if GetJSScriptMetasFunc is nil.
func (m *MockPageEntryView) GetJSScriptMetas() []templater_dto.JSScriptMeta {
	atomic.AddInt64(&m.GetJSScriptMetasCallCount, 1)
	if m.GetJSScriptMetasFunc != nil {
		return m.GetJSScriptMetasFunc()
	}
	return nil
}

// GetIsE2EOnly reports whether this entry is from the e2e/ directory.
//
// Returns bool, or false if GetIsE2EOnlyFunc is nil.
func (m *MockPageEntryView) GetIsE2EOnly() bool {
	atomic.AddInt64(&m.GetIsE2EOnlyCallCount, 1)
	if m.GetIsE2EOnlyFunc != nil {
		return m.GetIsE2EOnlyFunc()
	}
	return false
}

// GetStaticMetadata returns a pointer to pre-computed static metadata.
//
// Returns *InternalMetadata, or nil if GetStaticMetadataFunc is nil.
func (m *MockPageEntryView) GetStaticMetadata() *templater_dto.InternalMetadata {
	atomic.AddInt64(&m.GetStaticMetadataCallCount, 1)
	if m.GetStaticMetadataFunc != nil {
		return m.GetStaticMetadataFunc()
	}
	return nil
}

// GetHasAuthPolicy reports whether the page declares auth requirements.
//
// Returns bool, or false if GetHasAuthPolicyFunc is nil.
func (m *MockPageEntryView) GetHasAuthPolicy() bool {
	atomic.AddInt64(&m.GetHasAuthPolicyCallCount, 1)
	if m.GetHasAuthPolicyFunc != nil {
		return m.GetHasAuthPolicyFunc()
	}
	return false
}

// GetAuthPolicy returns the auth policy for the given request.
//
// Takes r (*templater_dto.RequestData) which provides the request context
// for policy evaluation.
//
// Returns daemon_dto.AuthPolicy, or zero value if GetAuthPolicyFunc is nil.
func (m *MockPageEntryView) GetAuthPolicy(r *templater_dto.RequestData) daemon_dto.AuthPolicy {
	atomic.AddInt64(&m.GetAuthPolicyCallCount, 1)
	if m.GetAuthPolicyFunc != nil {
		return m.GetAuthPolicyFunc(r)
	}
	return daemon_dto.AuthPolicy{}
}

// GetHasPreview reports whether this component defines a Preview function.
//
// Returns bool, or false if GetHasPreviewFunc is nil.
func (m *MockPageEntryView) GetHasPreview() bool {
	atomic.AddInt64(&m.GetHasPreviewCallCount, 1)
	if m.GetHasPreviewFunc != nil {
		return m.GetHasPreviewFunc()
	}
	return false
}

// GetPreviewScenarios returns preview scenarios for this component.
//
// Returns []templater_dto.PreviewScenario, or nil if
// GetPreviewScenariosFunc is nil.
func (m *MockPageEntryView) GetPreviewScenarios() []templater_dto.PreviewScenario {
	atomic.AddInt64(&m.GetPreviewScenariosCallCount, 1)
	if m.GetPreviewScenariosFunc != nil {
		return m.GetPreviewScenariosFunc()
	}
	return nil
}
