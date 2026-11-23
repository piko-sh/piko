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

package browser

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"piko.sh/piko/wdk/goroutine"
)

// RequestID correlates request, cookie and response notifications in a browser tab.
type RequestID string

// NetworkEventKind distinguishes the notifications delivered by ObserveNetwork.
type NetworkEventKind uint8

const (
	// RequestStarted reports a request the page is about to send.
	RequestStarted NetworkEventKind = iota + 1

	// RequestCookies reports the cookies the browser considered for a request.
	RequestCookies

	// ResponseReceived reports a response's URL and status.
	ResponseReceived

	// JavaScriptException reports an uncaught JavaScript exception.
	JavaScriptException
)

var (
	// errMissingNetworkObserver is returned when ObserveNetwork is given no callback.
	errMissingNetworkObserver = errors.New("browser: missing network observer")
)

// RequestCookie identifies a cookie the browser considered for a request.
type RequestCookie struct {
	// Name is the cookie name.
	Name string

	// Blocked is true when the browser recorded a reason to withhold the cookie.
	Blocked bool
}

// NetworkEvent describes a request, response or uncaught JavaScript exception.
//
// RequestCookies can arrive before or after RequestStarted for the same ID.
type NetworkEvent struct {
	// ID correlates the notifications of one request.
	ID RequestID

	// Method is the request method.
	Method string

	// URL is the request or response URL.
	URL string

	// ContentType is the request's Content-Type header.
	ContentType string

	// Initiator is the browser's request initiator type, such as script or parser.
	Initiator string

	// Exception describes an uncaught JavaScript exception.
	Exception string

	// Cookies lists the cookies considered for the request.
	Cookies []RequestCookie

	// Status is the response status.
	Status int

	// Kind identifies the notification.
	Kind NetworkEventKind
}

// ObserveNetwork subscribes to network activity and uncaught exceptions on the tab.
//
// The subscription ends on cancellation, tab closure or a recovered callback panic. The
// browser keeps network and runtime domains enabled for navigation and waits. Issue
// browser commands outside the callback because it runs on the browser event loop.
//
// Takes observe (func(NetworkEvent)) which receives notifications in arrival order.
//
// Returns func() which unsubscribes and can be called more than once.
// Returns error when observe is nil or the domains cannot be enabled.
func (s *Session) ObserveNetwork(ctx context.Context, observe func(NetworkEvent)) (func(), error) {
	if observe == nil {
		return nil, errMissingNetworkObserver
	}
	listenCtx, release := s.operation(ctx)
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(release) }
	chromedp.ListenTarget(listenCtx, func(raw any) {
		event, ok := networkObservation(raw)
		if !ok || listenCtx.Err() != nil {
			return
		}
		err := goroutine.SafeCall(listenCtx, "browser.ObserveNetwork", func() error {
			observe(event)
			return nil
		})
		if err != nil {
			stop()
		}
	})
	if err := chromedp.Run(listenCtx, network.Enable(), runtime.Enable()); err != nil {
		stop()
		return nil, fmt.Errorf("enabling network observation: %w", err)
	}
	return stop, nil
}

// RequestPostData returns the post data retained by the browser for the observed request.
// Browsers can omit uploaded file contents and reject reads after evicting a request.
//
// Takes id (RequestID) which identifies the observed request.
//
// Returns []byte which is the retained post data.
// Returns error when the browser no longer holds the request or the read fails.
func (s *Session) RequestPostData(ctx context.Context, id RequestID) ([]byte, error) {
	data, err := s.readRetained(ctx, func(runCtx context.Context) ([]byte, error) {
		return network.GetRequestPostData(network.RequestID(id)).Do(runCtx)
	})
	if err != nil {
		return nil, fmt.Errorf("reading post data of request %s: %w", id, err)
	}
	return data, nil
}

// ResponseBody returns the response bytes retained for the observed request. Read after
// completion, while the browser still holds the response.
//
// Takes id (RequestID) which identifies the observed request.
//
// Returns []byte which is the retained response body.
// Returns error when the browser no longer holds the response or the read fails.
func (s *Session) ResponseBody(ctx context.Context, id RequestID) ([]byte, error) {
	data, err := s.readRetained(ctx, func(runCtx context.Context) ([]byte, error) {
		return network.GetResponseBody(network.RequestID(id)).Do(runCtx)
	})
	if err != nil {
		return nil, fmt.Errorf("reading response body of request %s: %w", id, err)
	}
	return data, nil
}

// readRetained runs one read of data the browser retained for a request.
//
// Takes read (func(context.Context) ([]byte, error)) which issues the protocol read.
//
// Returns []byte which is the data read.
// Returns error when the read fails.
func (s *Session) readRetained(ctx context.Context, read func(context.Context) ([]byte, error)) ([]byte, error) {
	operation, release := s.operation(ctx)
	defer release()
	var data []byte
	err := chromedp.Run(operation, chromedp.ActionFunc(func(runCtx context.Context) error {
		var err error
		data, err = read(runCtx)
		return err
	}))
	return data, err
}

// networkObservation converts a browser notification into a NetworkEvent.
//
// Takes raw (any) which is the browser notification.
//
// Returns NetworkEvent which is the converted notification.
// Returns bool which is false when the notification is not observed.
func networkObservation(raw any) (NetworkEvent, bool) {
	switch event := raw.(type) {
	case *network.EventRequestWillBeSent:
		return requestStarted(event)
	case *network.EventRequestWillBeSentExtraInfo:
		return requestCookies(event), true
	case *network.EventResponseReceived:
		if event.Response == nil {
			return NetworkEvent{}, false
		}
		var response NetworkEvent
		response.Kind = ResponseReceived
		response.ID = RequestID(event.RequestID)
		response.URL = event.Response.URL
		response.Status = int(event.Response.Status)
		return response, true
	case *runtime.EventExceptionThrown:
		if event.ExceptionDetails == nil {
			return NetworkEvent{}, false
		}
		var exception NetworkEvent
		exception.Kind = JavaScriptException
		exception.Exception = event.ExceptionDetails.Error()
		return exception, true
	default:
		return NetworkEvent{}, false
	}
}

// requestStarted converts a request notification.
//
// Takes event (*network.EventRequestWillBeSent) which is the notification.
//
// Returns NetworkEvent which describes the request.
// Returns bool which is false when the notification has no request.
func requestStarted(event *network.EventRequestWillBeSent) (NetworkEvent, bool) {
	if event.Request == nil {
		return NetworkEvent{}, false
	}
	var started NetworkEvent
	started.Kind = RequestStarted
	started.ID = RequestID(event.RequestID)
	started.Method = event.Request.Method
	started.URL = event.Request.URL
	for name, value := range event.Request.Headers {
		if strings.EqualFold(name, "Content-Type") {
			started.ContentType = fmt.Sprint(value)
			break
		}
	}
	if event.Initiator != nil {
		started.Initiator = string(event.Initiator.Type)
	}
	return started, true
}

// requestCookies converts a cookie notification.
//
// Takes event (*network.EventRequestWillBeSentExtraInfo) which is the notification.
//
// Returns NetworkEvent which lists the cookies considered for the request.
func requestCookies(event *network.EventRequestWillBeSentExtraInfo) NetworkEvent {
	var cookies NetworkEvent
	cookies.Kind = RequestCookies
	cookies.ID = RequestID(event.RequestID)
	for _, associated := range event.AssociatedCookies {
		if associated != nil && associated.Cookie != nil {
			cookies.Cookies = append(cookies.Cookies, RequestCookie{Name: associated.Cookie.Name, Blocked: len(associated.BlockedReasons) != 0})
		}
	}
	return cookies
}
