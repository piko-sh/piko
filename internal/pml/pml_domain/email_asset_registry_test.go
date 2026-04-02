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

package pml_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailAssetRegistry(t *testing.T) {
	registry := NewEmailAssetRegistry()

	require.NotNil(t, registry)
	assert.NotNil(t, registry.Requests)
	assert.Empty(t, registry.Requests)
}

func TestEmailAssetRegistry_RegisterAsset(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid := registry.RegisterAsset("/path/to/image.png", "default", 200, "1x")

	assert.NotEmpty(t, cid)
	assert.Len(t, registry.Requests, 1)

	request := registry.Requests[0]
	assert.Equal(t, "/path/to/image.png", request.SourcePath)
	assert.Equal(t, "default", request.Profile)
	assert.Equal(t, 200, request.Width)
	assert.Equal(t, "1x", request.Density)
	assert.Equal(t, cid, request.CID)
}

func TestEmailAssetRegistry_RegisterAsset_Deduplication(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.RegisterAsset("/path/to/image.png", "default", 200, "1x")
	cid2 := registry.RegisterAsset("/path/to/image.png", "default", 200, "1x")

	assert.Equal(t, cid1, cid2)
	assert.Len(t, registry.Requests, 1)
}

func TestEmailAssetRegistry_RegisterAsset_DifferentAssets(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.RegisterAsset("/path/to/image1.png", "default", 200, "1x")
	cid2 := registry.RegisterAsset("/path/to/image2.png", "default", 200, "1x")

	assert.NotEqual(t, cid1, cid2)
	assert.Len(t, registry.Requests, 2)
}

func TestEmailAssetRegistry_RegisterAsset_DifferentProfiles(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.RegisterAsset("/path/to/image.png", "profile1", 200, "1x")
	cid2 := registry.RegisterAsset("/path/to/image.png", "profile2", 200, "1x")

	assert.NotEqual(t, cid1, cid2)
	assert.Len(t, registry.Requests, 2)
}

func TestEmailAssetRegistry_RegisterAsset_DifferentWidths(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.RegisterAsset("/path/to/image.png", "default", 200, "1x")
	cid2 := registry.RegisterAsset("/path/to/image.png", "default", 400, "1x")

	assert.NotEqual(t, cid1, cid2)
	assert.Len(t, registry.Requests, 2)
}

func TestEmailAssetRegistry_RegisterAsset_DifferentDensities(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.RegisterAsset("/path/to/image.png", "default", 200, "1x")
	cid2 := registry.RegisterAsset("/path/to/image.png", "default", 200, "2x")

	assert.NotEqual(t, cid1, cid2)
	assert.Len(t, registry.Requests, 2)
}

func TestEmailAssetRegistry_GenerateCID_Stable(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid1 := registry.generateCID("/path/to/image.png", "default", 200, "1x")
	cid2 := registry.generateCID("/path/to/image.png", "default", 200, "1x")

	assert.Equal(t, cid1, cid2)
}

func TestEmailAssetRegistry_GenerateCID_Format(t *testing.T) {
	registry := NewEmailAssetRegistry()

	cid := registry.generateCID("/path/to/image.png", "default", 200, "1x")

	assert.Contains(t, cid, "asset_")
	assert.Len(t, cid, 22)
}

func TestEmailAssetRegistry_MultipleAssets(t *testing.T) {
	registry := NewEmailAssetRegistry()

	registry.RegisterAsset("/images/logo.png", "default", 100, "1x")
	registry.RegisterAsset("/images/banner.jpg", "default", 600, "1x")
	registry.RegisterAsset("/images/icon.svg", "default", 50, "1x")
	registry.RegisterAsset("/images/logo.png", "default", 200, "2x")

	assert.Len(t, registry.Requests, 4)

	cids := make(map[string]bool)
	for _, request := range registry.Requests {
		assert.NotEmpty(t, request.CID)
		assert.NotEmpty(t, request.SourcePath)
		assert.False(t, cids[request.CID], "Duplicate CID found: %s", request.CID)
		cids[request.CID] = true
	}
}
