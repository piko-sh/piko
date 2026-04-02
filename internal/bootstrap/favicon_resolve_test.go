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
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/config"
)

func TestResolveFaviconSrcPaths_EmptySrc(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Href: "/favicon.ico", Type: "image/x-icon"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "/favicon.ico", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_BasicSrc(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "assets/favicon.png"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "/_piko/assets/assets/favicon.png", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_ModuleAlias(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "@/assets/favicon.ico"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "/_piko/assets/mymodule/assets/favicon.ico", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_AbsoluteURL(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "https://cdn.example.com/icon.png"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "https://cdn.example.com/icon.png", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_HttpURL(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "http://example.com/icon.png"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "http://example.com/icon.png", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_DataURI(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "data:image/png;base64,iVBOR"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "data:image/png;base64,iVBOR", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_AbsolutePath(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "/already/absolute.ico"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "/already/absolute.ico", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_SrcTakesPrecedenceOverHref(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Href: "/old-href.ico", Src: "@/assets/favicon.ico"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Equal(t, "/_piko/assets/mymodule/assets/favicon.ico", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_SrcClearedAfterResolution(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "@/assets/favicon.ico"},
		{Rel: "apple-touch-icon", Src: "https://example.com/apple.png"},
		{Rel: "icon", Src: "/absolute.ico"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mod")

	for i := range favicons {
		assert.Empty(t, favicons[i].Src, "Src should be cleared for favicon index %d", i)
	}
}

func TestResolveFaviconSrcPaths_PathCleaning(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "@/static/../icons/./fav.png"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymod")

	assert.Equal(t, "/_piko/assets/mymod/icons/fav.png", favicons[0].Href)
}

func TestResolveFaviconSrcPaths_EmptyModuleName(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "@/assets/favicon.ico"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "")

	assert.Equal(t, "/_piko/assets/@/assets/favicon.ico", favicons[0].Href)
	assert.Empty(t, favicons[0].Src)
}

func TestResolveFaviconSrcPaths_MultipleFavicons(t *testing.T) {
	t.Parallel()
	favicons := []config.FaviconDefinition{
		{Rel: "icon", Src: "@/assets/favicon.ico", Type: "image/x-icon"},
		{Rel: "apple-touch-icon", Src: "@/assets/apple-touch-icon.png", Sizes: "180x180"},
		{Rel: "icon", Href: "/external.svg"},
	}

	resolveFaviconSrcPaths(context.Background(), favicons, "mymod")

	assert.Equal(t, "/_piko/assets/mymod/assets/favicon.ico", favicons[0].Href)
	assert.Equal(t, "image/x-icon", favicons[0].Type)
	assert.Equal(t, "/_piko/assets/mymod/assets/apple-touch-icon.png", favicons[1].Href)
	assert.Equal(t, "180x180", favicons[1].Sizes)
	assert.Equal(t, "/external.svg", favicons[2].Href, "Href-only favicon should be untouched")
}

func TestResolveFaviconSrcPaths_EmptySlice(t *testing.T) {
	t.Parallel()
	var favicons []config.FaviconDefinition

	resolveFaviconSrcPaths(context.Background(), favicons, "mymodule")

	assert.Empty(t, favicons)
}

func TestWithWebsiteConfig_SetsOverride(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())

	websiteConfig := config.WebsiteConfig{
		Name:        "Test Site",
		Description: "A test site",
		Favicons: []config.FaviconDefinition{
			{Rel: "icon", Href: "/favicon.ico"},
		},
	}

	opt := WithWebsiteConfig(websiteConfig)
	opt(c)

	assert.NotNil(t, c.websiteConfigOverride)
	assert.Equal(t, "Test Site", c.websiteConfigOverride.Name)
	assert.Len(t, c.websiteConfigOverride.Favicons, 1)
}

func TestWithWebsiteConfig_DefaultIsNil(t *testing.T) {
	t.Parallel()
	c := NewContainer(config.NewConfigProvider())

	assert.Nil(t, c.websiteConfigOverride)
}
