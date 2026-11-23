//go:build integration

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

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestCheckoutWaitsForBothPageModules(t *testing.T) {
	requireChromium(t)
	clearRecipeVariables(t)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, err := fmt.Fprint(writer, `<form id="checkout-form"><input id="customer"><input id="quantity">
<button type="button" id="next">Review</button></form>
<template id="review-template"><section id="checkout-review" data-status="ready"
 data-attempts="0" data-total="0" data-committed="0" data-unit="1">
 <span id="review-customer"></span><span id="review-quantity"></span><span id="receipt"></span>
 <button id="restore">Restore</button><button id="place">Place</button><span id="error"></span>
</section></template>
<script>
const el = id => document.getElementById(id);
const ready = new Set();
let afterNavigate;
window.piko = {
  context: {get: () => ({hasFunction(name) {
    if (ready.has(name)) return true;
    requestAnimationFrame(() => ready.add(name));
    return false;
  }})},
  nav: {guard(hook) { afterNavigate = hook.afterNavigate; return () => {}; }}
};
el('next').click = () => {
  if (!ready.has('next')) throw Error('draft module unavailable');
  const customer = el('customer').value, quantity = el('quantity').value;
  history.pushState(null, '', '/checkout-review');
  el('checkout-form').replaceWith(el('review-template').content.cloneNode(true));
  el('restore').click = () => {
    if (!ready.has('restore')) throw Error('review module unavailable');
    el('review-customer').textContent = customer;
    el('review-quantity').textContent = quantity;
  };
  el('place').click = () => { throw Error('reached order submission'); };
  afterNavigate(location.href);
};
</script>`)
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx, session := openTestBrowser(t, 20*time.Second)
	input := casefile.CheckoutInput{Customer: "Alice", Quantity: 3}
	got := driveCheckout(ctx, session, server.URL, input)
	require.Len(t, got.States, 2, "%+v", got)
	assert.Equal(t, "draft", got.States[0].Step)
	assert.Equal(t, "review", got.States[1].Step)
	assert.Equal(t, input.Customer, got.States[1].Customer)
	assert.Equal(t, input.Quantity, got.States[1].Quantity)
	assert.Contains(t, got.DriverError, "reached order submission")
}
