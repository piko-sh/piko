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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/component/component_dto"
	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestWithCSRFTokenMaxAge(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCSRFTokenMaxAge(24 * time.Hour)
	opt(c)

	assert.Equal(t, 24*time.Hour, c.csrfTokenMaxAge)
}

func TestWithCSRFTokenMaxAge_ZeroValue(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCSRFTokenMaxAge(0)
	opt(c)

	assert.Equal(t, time.Duration(0), c.csrfTokenMaxAge)
}

func TestWithCSRFTokenMaxAge_Negative(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCSRFTokenMaxAge(-5 * time.Minute)
	opt(c)

	assert.Equal(t, -5*time.Minute, c.csrfTokenMaxAge)
}

func TestWithCSRFTokenMaxAge_SubSecond(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCSRFTokenMaxAge(500 * time.Millisecond)
	opt(c)

	assert.Equal(t, 500*time.Millisecond, c.csrfTokenMaxAge)
}

func TestWithCSRFTokenMaxAge_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithCSRFTokenMaxAge(48 * time.Hour))

	assert.Equal(t, 48*time.Hour, c.csrfTokenMaxAge)
}

func TestWithTrustedProxies(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithTrustedProxies("10.0.0.0/8", "192.168.0.0/16")
	opt(c)

	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, c.configServerOverrides.Security.RateLimit.TrustedProxies)
}

func TestWithTrustedProxies_SingleCIDR(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithTrustedProxies("172.16.0.0/12")
	opt(c)

	assert.Equal(t, []string{"172.16.0.0/12"}, c.configServerOverrides.Security.RateLimit.TrustedProxies)
}

func TestWithTrustedProxies_Empty(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithTrustedProxies()
	opt(c)

	assert.Nil(t, c.configServerOverrides.Security.RateLimit.TrustedProxies)
}

func TestWithTrustedProxies_OverwritesPrevious(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithTrustedProxies("10.0.0.0/8")(c)
	WithTrustedProxies("192.168.0.0/16")(c)

	assert.Equal(t, []string{"192.168.0.0/16"}, c.configServerOverrides.Security.RateLimit.TrustedProxies)
}

func TestWithTrustedProxies_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithTrustedProxies("10.0.0.0/8", "172.16.0.0/12"))

	assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12"}, c.configServerOverrides.Security.RateLimit.TrustedProxies)
}

func TestWithCloudflareEnabled_True(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCloudflareEnabled(true)
	opt(c)

	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.CloudflareEnabled, false))
}

func TestWithCloudflareEnabled_False(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithCloudflareEnabled(false)
	opt(c)

	assert.False(t, deref(c.configServerOverrides.Security.RateLimit.CloudflareEnabled, false))
}

func TestWithCloudflareEnabled_ToggleTrueThenFalse(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithCloudflareEnabled(true)(c)
	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.CloudflareEnabled, false))

	WithCloudflareEnabled(false)(c)
	assert.False(t, deref(c.configServerOverrides.Security.RateLimit.CloudflareEnabled, false))
}

func TestWithCloudflareEnabled_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithCloudflareEnabled(true))

	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.CloudflareEnabled, false))
}

func TestWithRateLimitEnabled_True(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithRateLimitEnabled(true)
	opt(c)

	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.Enabled, false))
}

func TestWithRateLimitEnabled_False(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithRateLimitEnabled(false)
	opt(c)

	assert.False(t, deref(c.configServerOverrides.Security.RateLimit.Enabled, false))
}

func TestWithRateLimitEnabled_ToggleFalseThenTrue(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithRateLimitEnabled(false)(c)
	assert.False(t, deref(c.configServerOverrides.Security.RateLimit.Enabled, false))

	WithRateLimitEnabled(true)(c)
	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.Enabled, false))
}

func TestWithRateLimitEnabled_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithRateLimitEnabled(true))

	assert.True(t, deref(c.configServerOverrides.Security.RateLimit.Enabled, false))
}

type stubResolver struct {
	prefix string
}

func (r *stubResolver) GetPrefix() string                                       { return r.prefix }
func (r *stubResolver) Resolve(_ context.Context, value string) (string, error) { return value, nil }

func TestWithConfigResolvers_Single(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	r := &stubResolver{prefix: "env:"}
	opt := WithConfigResolvers(r)
	opt(c)

	require.Len(t, c.configResolvers, 1)
	assert.Equal(t, "env:", c.configResolvers[0].GetPrefix())
}

func TestWithConfigResolvers_Multiple(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	r1 := &stubResolver{prefix: "env:"}
	r2 := &stubResolver{prefix: "file:"}
	opt := WithConfigResolvers(r1, r2)
	opt(c)

	require.Len(t, c.configResolvers, 2)
	assert.Equal(t, "env:", c.configResolvers[0].GetPrefix())
	assert.Equal(t, "file:", c.configResolvers[1].GetPrefix())
}

func TestWithConfigResolvers_Appends(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	r1 := &stubResolver{prefix: "env:"}
	r2 := &stubResolver{prefix: "vault:"}

	WithConfigResolvers(r1)(c)
	WithConfigResolvers(r2)(c)

	require.Len(t, c.configResolvers, 2)
	assert.Equal(t, "env:", c.configResolvers[0].GetPrefix())
	assert.Equal(t, "vault:", c.configResolvers[1].GetPrefix())
}

func TestWithConfigResolvers_Empty(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithConfigResolvers()
	opt(c)

	assert.Empty(t, c.configResolvers)
}

func TestWithConfigResolvers_ViaNewContainer(t *testing.T) {
	t.Parallel()
	r := &stubResolver{prefix: "ssm:"}
	c := NewContainer(WithConfigResolvers(r))

	require.Len(t, c.configResolvers, 1)
	assert.Equal(t, "ssm:", c.configResolvers[0].GetPrefix())
}

func TestWithExperimentalPrerendering_True(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithExperimentalPrerendering(true)
	opt(c)

	assert.True(t, c.experimentalPrerendering)
}

func TestWithExperimentalPrerendering_False(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithExperimentalPrerendering(false)
	opt(c)

	assert.False(t, c.experimentalPrerendering)
}

func TestWithExperimentalPrerendering_ToggleTrueThenFalse(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithExperimentalPrerendering(true)(c)
	assert.True(t, c.experimentalPrerendering)

	WithExperimentalPrerendering(false)(c)
	assert.False(t, c.experimentalPrerendering)
}

func TestWithExperimentalPrerendering_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithExperimentalPrerendering(true))

	assert.True(t, c.experimentalPrerendering)
}

func TestWithExperimentalCommentStripping_True(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithExperimentalCommentStripping(true)
	opt(c)

	assert.True(t, c.experimentalCommentStripping)
}

func TestWithExperimentalCommentStripping_False(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithExperimentalCommentStripping(false)
	opt(c)

	assert.False(t, c.experimentalCommentStripping)
}

func TestWithExperimentalCommentStripping_ToggleTrueThenFalse(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithExperimentalCommentStripping(true)(c)
	assert.True(t, c.experimentalCommentStripping)

	WithExperimentalCommentStripping(false)(c)
	assert.False(t, c.experimentalCommentStripping)
}

func TestWithExperimentalCommentStripping_ViaNewContainer(t *testing.T) {
	t.Parallel()
	c := NewContainer(WithExperimentalCommentStripping(true))

	assert.True(t, c.experimentalCommentStripping)
}

func TestWithComponents_Single(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	comp := component_dto.ComponentDefinition{TagName: "my-button"}
	opt := WithComponents(comp)
	opt(c)

	require.Len(t, c.externalComponents, 1)
	assert.Equal(t, "my-button", c.externalComponents[0].TagName)
}

func TestWithComponents_Multiple(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	components := []component_dto.ComponentDefinition{
		{TagName: "my-button"},
		{TagName: "my-card"},
		{TagName: "my-modal"},
	}

	opt := WithComponents(components...)
	opt(c)

	require.Len(t, c.externalComponents, 3)
	assert.Equal(t, "my-button", c.externalComponents[0].TagName)
	assert.Equal(t, "my-card", c.externalComponents[1].TagName)
	assert.Equal(t, "my-modal", c.externalComponents[2].TagName)
}

func TestWithComponents_Appends(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithComponents(component_dto.ComponentDefinition{TagName: "comp-a"})(c)
	WithComponents(component_dto.ComponentDefinition{TagName: "comp-b"})(c)

	require.Len(t, c.externalComponents, 2)
	assert.Equal(t, "comp-a", c.externalComponents[0].TagName)
	assert.Equal(t, "comp-b", c.externalComponents[1].TagName)
}

func TestWithComponents_Empty(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithComponents()
	opt(c)

	assert.Empty(t, c.externalComponents)
}

func TestWithComponents_PreservesAllFields(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	comp := component_dto.ComponentDefinition{
		TagName:    "uikit-card",
		SourcePath: "components/card.pkc",
		ModulePath: "github.com/someone/uikit",
		IsExternal: true,
	}

	WithComponents(comp)(c)

	require.Len(t, c.externalComponents, 1)
	assert.Equal(t, "uikit-card", c.externalComponents[0].TagName)
	assert.Equal(t, "components/card.pkc", c.externalComponents[0].SourcePath)
	assert.Equal(t, "github.com/someone/uikit", c.externalComponents[0].ModulePath)
	assert.True(t, c.externalComponents[0].IsExternal)
}

func TestWithComponents_ViaNewContainer(t *testing.T) {
	t.Parallel()
	comp := component_dto.ComponentDefinition{TagName: "my-widget"}
	c := NewContainer(WithComponents(comp))

	require.Len(t, c.externalComponents, 1)
	assert.Equal(t, "my-widget", c.externalComponents[0].TagName)
}

func TestWithSandboxFactory(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.Nil(t, c.sandboxFactory)

	factory := func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error) {
		return nil, nil
	}

	opt := WithSandboxFactory(factory)
	opt(c)

	assert.NotNil(t, c.sandboxFactory)
}

func TestWithSandboxFactory_OverwritesPrevious(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	factory1 := func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error) {
		return nil, nil
	}
	factory2 := func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error) {
		return nil, nil
	}

	WithSandboxFactory(factory1)(c)
	assert.NotNil(t, c.sandboxFactory)

	WithSandboxFactory(factory2)(c)
	assert.NotNil(t, c.sandboxFactory)
}

func TestWithSandboxFactory_NilFactory(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	opt := WithSandboxFactory(nil)
	opt(c)

	assert.Nil(t, c.sandboxFactory)
}

func TestWithSandboxFactory_ViaNewContainer(t *testing.T) {
	t.Parallel()
	factory := func(name, baseDir string, mode safedisk.Mode) (safedisk.Sandbox, error) {
		return nil, nil
	}
	c := NewContainer(WithSandboxFactory(factory))

	assert.NotNil(t, c.sandboxFactory)
}

func TestWithMonitoringAddress(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	opt := WithMonitoringAddress(":9999")
	opt(&monitoringConfig)

	assert.Equal(t, ":9999", monitoringConfig.Address)
}

func TestWithMonitoringAddress_EmptyString(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{Address: ":9091"}

	opt := WithMonitoringAddress("")
	opt(&monitoringConfig)

	assert.Equal(t, "", monitoringConfig.Address)
}

func TestWithMonitoringAddress_FullAddress(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	opt := WithMonitoringAddress("127.0.0.1:8080")
	opt(&monitoringConfig)

	assert.Equal(t, "127.0.0.1:8080", monitoringConfig.Address)
}

func TestWithMonitoringAddress_OverwritesPrevious(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	WithMonitoringAddress(":1111")(&monitoringConfig)
	WithMonitoringAddress(":2222")(&monitoringConfig)

	assert.Equal(t, ":2222", monitoringConfig.Address)
}

func TestWithMonitoringBindAddress(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	opt := WithMonitoringBindAddress("0.0.0.0")
	opt(&monitoringConfig)

	assert.Equal(t, "0.0.0.0", monitoringConfig.BindAddress)
}

func TestWithMonitoringBindAddress_Localhost(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	opt := WithMonitoringBindAddress("127.0.0.1")
	opt(&monitoringConfig)

	assert.Equal(t, "127.0.0.1", monitoringConfig.BindAddress)
}

func TestWithMonitoringBindAddress_EmptyString(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{BindAddress: "127.0.0.1"}

	opt := WithMonitoringBindAddress("")
	opt(&monitoringConfig)

	assert.Equal(t, "", monitoringConfig.BindAddress)
}

func TestWithMonitoringBindAddress_OverwritesPrevious(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	WithMonitoringBindAddress("127.0.0.1")(&monitoringConfig)
	WithMonitoringBindAddress("0.0.0.0")(&monitoringConfig)

	assert.Equal(t, "0.0.0.0", monitoringConfig.BindAddress)
}

func TestWithMonitoringAutoNextPort(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	WithMonitoringAutoNextPort(true)(&monitoringConfig)
	assert.True(t, monitoringConfig.AutoNextPort)
}

func TestWithMonitoringAutoNextPort_False(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	WithMonitoringAutoNextPort(true)(&monitoringConfig)
	WithMonitoringAutoNextPort(false)(&monitoringConfig)
	assert.False(t, monitoringConfig.AutoNextPort)
}

func TestMonitoringOptions_Combined(t *testing.T) {
	t.Parallel()
	monitoringConfig := monitoring_domain.ServiceConfig{}

	WithMonitoringAddress(":7777")(&monitoringConfig)
	WithMonitoringBindAddress("10.0.0.1")(&monitoringConfig)

	assert.Equal(t, ":7777", monitoringConfig.Address)
	assert.Equal(t, "10.0.0.1", monitoringConfig.BindAddress)
}

func TestRateLimitOptions_Combined(t *testing.T) {
	t.Parallel()
	c := NewContainer(
		WithRateLimitEnabled(true),
		WithCloudflareEnabled(true),
		WithTrustedProxies("10.0.0.0/8", "172.16.0.0/12"),
	)

	rl := c.configServerOverrides.Security.RateLimit
	assert.True(t, deref(rl.Enabled, false))
	assert.True(t, deref(rl.CloudflareEnabled, false))
	assert.Equal(t, []string{"10.0.0.0/8", "172.16.0.0/12"}, rl.TrustedProxies)
}

func TestExperimentalOptions_Combined(t *testing.T) {
	t.Parallel()
	c := NewContainer(
		WithExperimentalPrerendering(true),
		WithExperimentalCommentStripping(true),
	)

	assert.True(t, c.experimentalPrerendering)
	assert.True(t, c.experimentalCommentStripping)
}

func TestWithComponents_PreservesMetadataFields(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	comp := component_dto.ComponentDefinition{
		TagName:    "piko-counter",
		SourcePath: "piko/piko-counter.pkc",
		ModulePath: "piko.sh/piko/components",
		IsExternal: true,
	}

	WithComponents(comp)(c)

	require.Len(t, c.externalComponents, 1)
	stored := c.externalComponents[0]
	assert.Equal(t, "piko-counter", stored.TagName)
	assert.Equal(t, "piko/piko-counter.pkc", stored.SourcePath)
	assert.Equal(t, "piko.sh/piko/components", stored.ModulePath)
	assert.True(t, stored.IsExternal)
}

func TestHasExternalModuleComponents_WithModulePath(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithComponents(component_dto.ComponentDefinition{
		TagName:    "piko-card",
		ModulePath: "piko.sh/piko/components",
	})(c)

	assert.True(t, c.hasExternalModuleComponents())
}

func TestHasExternalModuleComponents_WithoutModulePath(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	WithComponents(component_dto.ComponentDefinition{
		TagName: "my-button",
	})(c)

	assert.False(t, c.hasExternalModuleComponents(), "definition without ModulePath should not count as external module component")
}

func TestHasExternalModuleComponents_Empty(t *testing.T) {
	t.Parallel()
	c := NewContainer()

	assert.False(t, c.hasExternalModuleComponents())
}
