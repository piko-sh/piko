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

package bootstrap

import (
	"context"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
)

// resolveFaviconSources resolves any Src fields on favicon definitions in the
// website config using the resolver for @/ alias expansion, doing nothing if
// no favicons have Src set.
//
// Takes container (*Container) which provides access to the website
// configuration and the resolver needed for alias expansion.
func resolveFaviconSources(container *Container) {
	favicons := container.websiteConfig.Favicons
	if len(favicons) == 0 {
		return
	}

	hasSrc := false
	for i := range favicons {
		if favicons[i].Src != "" {
			hasSrc = true
			break
		}
	}
	if !hasSrc {
		return
	}

	_, l := logger_domain.From(container.GetAppContext(), log)

	resolver, err := container.GetResolver()
	if err != nil {
		l.Warn("Could not initialise resolver for favicon Src resolution; Src fields will not be resolved.",
			logger_domain.Error(err))
		return
	}

	moduleName := resolver.GetModuleName()
	resolveFaviconSrcPaths(container.GetAppContext(), favicons, moduleName)
}

// resolveFaviconSrcPaths resolves Src fields on favicon definitions into Href
// values using the asset pipeline path format, including @/ module alias
// expansion, and clears each Src field after resolution to prevent
// double-resolution.
//
// Takes ctx (context.Context) which carries the application context for logging.
// Takes favicons ([]config.FaviconDefinition) which contains the favicon
// definitions to process (modified in place via the slice).
// Takes moduleName (string) which is the Go module name for @/ alias
// resolution.
func resolveFaviconSrcPaths(ctx context.Context, favicons []config.FaviconDefinition, moduleName string) {
	_, l := logger_domain.From(ctx, log)

	for i := range favicons {
		fav := &favicons[i]
		if fav.Src == "" {
			continue
		}

		if strings.HasPrefix(fav.Src, assetpath.ModuleAliasPrefix) && moduleName == "" {
			l.Warn("Favicon Src uses @/ alias but module name is empty; path will not resolve correctly.",
				logger_domain.String("src", fav.Src))
		}

		fav.Href = assetpath.Transform(fav.Src, moduleName, assetpath.DefaultServePath)
		fav.Src = ""
	}
}
