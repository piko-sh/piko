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

// Package daemon_frontend manages embedded frontend assets and module
// registration for the Piko development daemon. It handles serving
// and content-negotiation for compressed static assets (JavaScript,
// CSS), supports both built-in framework modules and custom
// user-provided modules, and generates the HTML needed to load them
// in the browser.
//
// # Usage
//
// Initialise the asset store at startup, then retrieve assets by
// path:
//
//	err := daemon_frontend.InitAssetStore(ctx)
//	if err != nil {
//	    return err
//	}
//
//	// Register a custom module
//	daemon_frontend.RegisterCustomModule(ctx, "mymod", js, etag)
//
//	// Get best compressed variant based on Accept-Encoding
//	path := daemon_frontend.DetermineBestAssetPath(
//	    ctx, basePath, acceptEncoding,
//	)
//	asset, found := daemon_frontend.GetAsset(ctx, path)
//
// # Thread safety
//
// The asset store is initialised once at startup and is read-only
// thereafter. All read operations are safe for concurrent use.
package daemon_frontend
