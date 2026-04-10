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

package analytics_dto

import (
	"net/http"
	"sync"
	"time"

	"piko.sh/piko/wdk/maths"
)

// EventType classifies a backend analytics event.
type EventType int

const (
	// EventPageView is fired automatically for each page request.
	EventPageView EventType = iota

	// EventAction is fired when a server action executes.
	EventAction

	// EventCustom is a user-defined event fired manually from action handlers.
	EventCustom

	// eventTypeCount is the sentinel value for array dispatch sizing.
	eventTypeCount
)

// eventTypeNames maps each EventType to its string representation.
var eventTypeNames = [eventTypeCount]string{
	EventPageView: "pageview",
	EventAction:   "action",
	EventCustom:   "custom",
}

// String returns the human-readable name of the event type.
//
// Returns string which is the event type name.
func (t EventType) String() string {
	if int(t) < len(eventTypeNames) {
		return eventTypeNames[t]
	}
	return "unknown"
}

// Event carries the data for a single backend analytics event.
//
// Instances are pooled via [AcquireEvent] and [ReleaseEvent] to avoid
// allocation on the hot path. Collectors must not retain a pointer to
// the Event after [Collector.Collect] returns; they should copy any
// data they need.
type Event struct {
	// Timestamp is when the event occurred.
	Timestamp time.Time

	// Request is the raw HTTP request. Adapters may need the full
	// request for Accept-Language and Client Hints headers. This is a
	// reference only — do not read the body.
	Request *http.Request

	// Revenue holds optional monetary data for e-commerce analytics
	// events (e.g. purchases, refunds). Nil when the event does not
	// carry revenue information.
	Revenue *maths.Money

	// Properties holds arbitrary key-value metadata. Adapters that
	// support custom properties read from here.
	Properties map[string]string

	// MatchedPattern is the route pattern that matched (e.g. "/blog/{slug}").
	MatchedPattern string

	// Hostname is the request host (e.g. "example.com"). Required by
	// backends like Plausible that associate events with a site domain.
	Hostname string

	// URL is the full request URL including query parameters (e.g.
	// "/blog/my-post?utm_source=twitter"). Useful for UTM attribution
	// and campaign tracking.
	URL string

	// ClientIP is the real client IP as resolved by the RealIP middleware.
	ClientIP string

	// Path is the URL path of the request (e.g. "/blog/my-post").
	Path string

	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// UserAgent is the User-Agent header value.
	UserAgent string

	// Referrer is the Referer header value.
	Referrer string

	// Locale is the request locale (e.g. "en", "de").
	Locale string

	// UserID is the authenticated user's identifier, empty if anonymous.
	UserID string

	// ActionName is the name of the server action, empty for page views.
	ActionName string

	// EventName is an explicit name for custom analytics events (e.g.
	// "signup", "purchase"). Separate from ActionName which is specific
	// to Piko server actions. Used by adapters that track named goals
	// or conversions.
	EventName string

	// Duration is the time taken to handle the request.
	Duration time.Duration

	// Type classifies the event.
	Type EventType

	// StatusCode is the HTTP response status code.
	StatusCode int
}

// eventPool provides reusable Event instances.
var eventPool = sync.Pool{
	New: func() any { return &Event{} },
}

// AcquireEvent returns a zeroed Event from the pool.
//
// Returns *Event which is a reset instance ready for use.
func AcquireEvent() *Event {
	ev, ok := eventPool.Get().(*Event)
	if !ok {
		ev = &Event{}
	}
	*ev = Event{}
	return ev
}

// ReleaseEvent returns an Event to the pool. The caller must not use
// the Event after this call.
//
// Takes ev (*Event) which is the instance to return.
func ReleaseEvent(ev *Event) {
	if ev == nil {
		return
	}
	ev.Request = nil
	ev.Properties = nil
	ev.Revenue = nil
	eventPool.Put(ev)
}
