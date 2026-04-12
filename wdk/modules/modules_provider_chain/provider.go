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

package modules_provider_chain

import (
	"context"
	"errors"

	"piko.sh/piko/wdk/modules"
)

// Provider composes a sequence of modules.ModuleProviders.
//
// Each Resolve call walks the chain in construction order, returning the first
// non-not-found result. Hosts compose providers so that fast / offline / trusted sources
// are tried first; slower or internet-bound sources later.
//
// Safe for concurrent use as long as every constituent provider is safe for concurrent
// use.
type Provider struct {
	// chain holds the ordered provider links queried by Resolve.
	chain []modules.ModuleProvider
}

// New constructs a Provider that tries each supplied provider in order. nil entries are
// skipped (a convenience for hosts that conditionally include layers).
//
// Takes providers (modules.ModuleProvider variadic) which are the chain links in priority
// order.
//
// Returns a *Provider implementing modules.ModuleProvider.
func New(providers ...modules.ModuleProvider) *Provider {
	chain := make([]modules.ModuleProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		chain = append(chain, provider)
	}
	return &Provider{chain: chain}
}

// Resolve walks the chain and returns the first resolved bundle.
//
// modules.ErrModuleNotFound from a link causes the walk to continue; any other error
// short-circuits and is returned as-is. When every link returns ErrModuleNotFound,
// Resolve returns the same sentinel.
//
// Takes ctx (context.Context) which is forwarded to each link.
// Takes ref (modules.ModuleRef) which identifies the module to resolve.
//
// Returns *modules.ModuleBundle which is the first resolved bundle.
// Returns error when ctx is cancelled, every link reports ErrModuleNotFound, or any link
// returns a non-not-found error.
func (p *Provider) Resolve(ctx context.Context, ref modules.ModuleRef) (*modules.ModuleBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(p.chain) == 0 {
		return nil, modules.ErrModuleNotFound
	}
	for _, provider := range p.chain {
		bundle, err := provider.Resolve(ctx, ref)
		if err == nil {
			return bundle, nil
		}
		if errors.Is(err, modules.ErrModuleNotFound) {
			continue
		}
		return nil, err
	}
	return nil, modules.ErrModuleNotFound
}

// Len reports the number of non-nil links in the chain.
//
// Returns the link count.
func (p *Provider) Len() int {
	return len(p.chain)
}
