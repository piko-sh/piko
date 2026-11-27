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

package casefile

const (
	// ReuseFirstListing names the listing the element-reuse journey starts on.
	ReuseFirstListing = "orders"

	// ReuseSecondListing names the listing the element-reuse journey navigates to.
	ReuseSecondListing = "invoices"

	// reusePathPrefix prefixes a listing's name to form its route.
	reusePathPrefix = "/reuse-"
)

// switchListing models SPA navigation to a listing followed by a change of its filter.
// The previous listing's pager disconnects, the shown listing's pager connects, and only
// the shown listing's configuration handles the change.
//
// Takes listing (string) which names the listing navigated to.
func (model *lifecycleModel) switchListing(listing string) {
	model.state.Path = ReuseListingPath(listing)
	model.state.Cleaned++
	model.state.Connected++
	model.filterListing(listing)
}

// filterListing models one change event on the shown listing's filter input.
//
// Takes listing (string) which names the listing whose configuration must handle it.
func (model *lifecycleModel) filterListing(listing string) {
	model.state.Deliveries++
	if model.state.Applied != "" {
		model.state.Applied += ","
	}
	model.state.Applied += listing
}

// expectedElementReuse models the element-reuse journey between two listing pages under
// one layout. Each listing renders a filter input and a pager component at the same
// position, so SPA navigation can morph one listing's elements into the other's.
//
// Whatever the morph reuses, leaving a listing must release its pager's binding, and the
// next listing's pager must bind to that listing's filter with that listing's
// configuration. One filter change is handled exactly once, by the listing shown.
//
// Takes input (LifecycleInput) which is recorded with the trace.
//
// Returns *LifecycleTrace which holds the expected states.
func expectedElementReuse(input LifecycleInput) *LifecycleTrace {
	model := newLifecycleModel(input, ReuseListingPath(ReuseFirstListing))
	model.state.Connected = 1
	model.record("reuse-mount")
	model.filterListing(ReuseFirstListing)
	model.record("reuse-filter-" + ReuseFirstListing)
	model.switchListing(ReuseSecondListing)
	model.record("reuse-link-" + ReuseSecondListing)
	model.switchListing(ReuseFirstListing)
	model.record("reuse-link-" + ReuseFirstListing)
	model.switchListing(ReuseSecondListing)
	model.record("reuse-history-back")
	return model.trace
}

// ReuseListingPath returns the route of an element-reuse listing.
//
// Takes listing (string) which names the listing.
//
// Returns string which is the listing's route.
func ReuseListingPath(listing string) string {
	return reusePathPrefix + listing
}
