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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"piko.sh/fuzzkit/tape"
	core "piko.sh/piko/internal/binder"
	"piko.sh/piko/wdk/maths"
	"piko.sh/safeconv"
)

const (
	// stockCodeRange is the number of distinct stock codes.
	stockCodeRange = 65536

	// maxWholePrice bounds the drawn whole part of a price beyond one.
	maxWholePrice = 4096

	// priceFractionRange is the number of distinct two-digit price fractions.
	priceFractionRange = 100

	// invoiceCopyRange is the number of distinct invoice copy indices.
	invoiceCopyRange = 4

	// maxExtraKeyLength bounds the drawn label key length beyond two.
	maxExtraKeyLength = 8

	// subtreeDepthLimit is the smallest depth accepting the nested raw subtree.
	subtreeDepthLimit = 3

	// maxExtraFieldErrors bounds the drawn field-error limit beyond two.
	maxExtraFieldErrors = 5

	// sliceElementBudget is the shared slice element budget of one bind call.
	sliceElementBudget = 1000

	// minFirstSliceLength is the smallest drawn first-slice length.
	minFirstSliceLength = 450

	// firstSliceSpread is the number of distinct first-slice lengths.
	firstSliceSpread = 50

	// budgetNotes is the number of repeated notes bound beside the slices.
	budgetNotes = 400

	// firstSliceMarker is the value bound at the end of the first slice.
	firstSliceMarker = 17

	// secondSliceMarker is the value bound at the end of the second slice.
	secondSliceMarker = 23

	// stockCodeBits is the bit size of a stock code.
	stockCodeBits = 16
)

var (
	// errMissingStockPrefix reports a stock code without its sku- prefix.
	errMissingStockPrefix = errors.New("stock code needs sku prefix")
)

// stockCode tests binding through a nested named value.
type stockCode struct {
	// ID identifies the recorded item.
	ID uint16
}

// priceLine holds the invoice's numeric binding inputs.
type priceLine struct {
	// Total is the bound monetary total.
	Total maths.Money

	// Amount is the bound decimal amount.
	Amount maths.Decimal

	// Code holds the nested stock identifier.
	Code stockCode
}

// invoiceSchema tests nested pointers, slices and maps in one bind.
type invoiceSchema struct {
	// Line holds the primary nested invoice line.
	Line *priceLine

	// Labels holds the fixture's row labels.
	Labels map[string]string

	// Copies holds repeated invoice lines.
	Copies []*priceLine
}

// checkInvoice checks nested decimal and money binding.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkInvoice(ctx context.Context, choices *tape.Tape) error {
	astBinder := core.NewASTBinder()
	astBinder.RegisterConverter(reflect.TypeFor[stockCode](), func(text string) (reflect.Value, error) {
		digits, ok := strings.CutPrefix(text, "sku-")
		if !ok {
			return reflect.Value{}, errMissingStockPrefix
		}
		number, err := strconv.ParseUint(digits, decimalBase, stockCodeBits)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("parsing stock code %q: %w", text, err)
		}
		return reflect.ValueOf(stockCode{ID: safeconv.Uint64ToUint16(number)}), nil
	})
	id := safeconv.Uint64ToUint16(choices.Uint(stockCodeRange))
	whole := 1 + drawInt(choices, maxWholePrice)
	fraction := drawInt(choices, priceFractionRange)
	price := fmt.Sprintf("%d.%02d", whole, fraction)
	if choices.Bool(1, 2) {
		price = "-" + price
	}
	path := "Line"
	index := drawInt(choices, invoiceCopyRange)
	if choices.Bool(1, 2) {
		path = fmt.Sprintf("Copies[%d]", index)
	}
	values := map[string][]string{
		path + ".Code":   {fmt.Sprintf("sku-%d", id)},
		path + ".Amount": {price},
		path + ".Total":  {price},
	}
	var invoice invoiceSchema
	if err := astBinder.Bind(ctx, &invoice, values); err != nil {
		return property("invoice-accept", "%s: %w", path, err)
	}
	line, err := boundInvoiceLine(&invoice, path, index)
	if err != nil {
		return err
	}
	if err := checkInvoiceValue(line, id, price); err != nil {
		return err
	}
	var rejected invoiceSchema
	return fieldRefusal(astBinder.Bind(ctx, &rejected, map[string][]string{path + ".Code": {"invalid"}}), path+".Code", "invoice-converter")
}

// boundInvoiceLine selects the invoice line a bind populated.
//
// Takes invoice (*invoiceSchema) which is the bound invoice.
// Takes path (string) which is the bound line's field path.
// Takes index (int) which is the bound copy index when path names a copy.
//
// Returns *priceLine which is the populated line.
// Returns error which reports an unexpected pointer, collection or populated element.
func boundInvoiceLine(invoice *invoiceSchema, path string, index int) (*priceLine, error) {
	if path == "Line" {
		return invoice.Line, nil
	}
	if invoice.Line != nil || len(invoice.Copies) != index+1 {
		return nil, property("invoice-pointer", "unexpected pointer or collection")
	}
	for _, prior := range invoice.Copies[:index] {
		if prior != nil {
			return nil, property("invoice-sparse", "unassigned element populated")
		}
	}
	return invoice.Copies[index], nil
}

// checkInvoiceValue compares a bound invoice line with its independently drawn values.
//
// Takes line (*priceLine) which is the bound invoice line.
// Takes id (uint16) which is the expected stock identifier.
// Takes price (string) which is the expected decimal spelling.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkInvoiceValue(line *priceLine, id uint16, price string) error {
	if line == nil || line.Code.ID != id {
		return property("invoice-code", "expected sku-%d", id)
	}

	canonical := strings.TrimSuffix(strings.TrimRight(price, "0"), ".")
	decimal, err := line.Amount.String()
	if err != nil || decimal != canonical {
		return property("invoice-decimal", "got %q %v, want %q", decimal, err, price)
	}
	amount, err := line.Total.Amount()
	if err != nil {
		return property("invoice-money", "invalid money: %v", err)
	}
	money, err := amount.String()
	code, codeErr := line.Total.CurrencyCode()
	if err != nil || codeErr != nil || code != "GBP" || money != canonical {
		return property("invoice-money", "got %q %q want GBP %q", code, money, price)
	}
	return nil
}

// fieldRefusal checks the public identity of a field-binding failure.
//
// Takes err (error) which is the operation failure to inspect.
// Takes path (string) which is the expected field-error path.
// Takes id (string) which identifies the semantic property in a failure message.
//
// Returns error which reports an absent or incorrectly identified field refusal, or nil
// on success.
func fieldRefusal(err error, path, id string) error {
	fields, ok := errors.AsType[core.MultiError](err)
	if !ok || fields[path] == nil || len(fields) != 1 {
		return property(id, "expected one field refusal for %q, got %v", path, err)
	}
	return nil
}

// checkBinderLimits checks nested-container depth and binding budgets.
//
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkBinderLimits(ctx context.Context, choices *tape.Tape) error {
	astBinder := core.NewASTBinder()
	if err := checkPathLengthLimit(ctx, astBinder, choices); err != nil {
		return err
	}
	if err := checkSubtreeDepthLimit(ctx, astBinder); err != nil {
		return err
	}
	if err := checkFieldErrorLimit(ctx, astBinder, choices); err != nil {
		return err
	}
	return checkSliceBudget(ctx, astBinder, choices)
}

// checkPathLengthLimit checks the field-path length limit around its boundary.
//
// Takes astBinder (*core.ASTBinder) which is the production binder under test.
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkPathLengthLimit(ctx context.Context, astBinder *core.ASTBinder, choices *tape.Tape) error {
	key := strings.Repeat("a", 2+drawInt(choices, maxExtraKeyLength))
	path := fmt.Sprintf("Labels[%q]", key)
	input := map[string][]string{path: {"ink"}}
	for _, delta := range []int{-1, 0, 1} {
		var destination invoiceSchema
		err := astBinder.Bind(ctx, &destination, input, core.WithMaxPathLength(len(path)+delta))
		if delta < 0 {
			if failure := fieldRefusal(err, path, "schema-path-length"); failure != nil {
				return failure
			}
		} else if err != nil || destination.Labels[key] != "ink" {
			return property("schema-path-length", "limit %d, got %v (%v)", len(path)+delta, destination.Labels, err)
		}
	}
	return nil
}

// checkSubtreeDepthLimit checks the raw-subtree depth limit around its boundary.
//
// RawMessage subtrees consume one depth unit per nested container plus the enclosing
// field.
//
// Takes astBinder (*core.ASTBinder) which is the production binder under test.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSubtreeDepthLimit(ctx context.Context, astBinder *core.ASTBinder) error {
	nested := map[string]any{"a": map[string]any{"b": "v"}}
	for _, limit := range []int{subtreeDepthLimit - 1, subtreeDepthLimit, subtreeDepthLimit + 1} {
		var destination struct {
			Raw json.RawMessage
		}
		err := astBinder.BindMap(ctx, &destination, map[string]any{"Raw": nested}, core.WithMaxPathDepth(limit))
		if limit < subtreeDepthLimit {
			if failure := fieldRefusal(err, "Raw", "schema-subtree-depth"); failure != nil {
				return failure
			}
		} else if err != nil || string(destination.Raw) != `{"a":{"b":"v"}}` {
			return property("schema-subtree-depth", "depth %d got %s (%v)", limit, destination.Raw, err)
		}
	}
	return nil
}

// checkFieldErrorLimit checks field-error truncation and ordering around its limit.
//
// Takes astBinder (*core.ASTBinder) which is the production binder under test.
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkFieldErrorLimit(ctx context.Context, astBinder *core.ASTBinder, choices *tape.Tape) error {
	errorLimit := 2 + drawInt(choices, maxExtraFieldErrors)
	astBinder.SetMaxFieldErrors(int64(errorLimit))
	for _, fieldCount := range []int{errorLimit - 1, errorLimit, errorLimit + 1} {
		data := make(map[string][]string, fieldCount)
		schema := make([]reflect.StructField, 0, fieldCount)
		for index := range fieldCount {
			name := fmt.Sprintf("A%02d", index)
			schema = append(schema, reflect.StructField{Name: name, Type: reflect.TypeFor[uint8]()})
			data[name] = []string{"v"}
		}
		destination := reflect.New(reflect.StructOf(schema)).Interface()
		err := astBinder.Bind(ctx, destination, data)
		fields, ok := errors.AsType[core.MultiError](err)
		if !ok || len(fields) != fieldCount || (fields[core.FieldErrorsTruncatedKey] != nil) != (fieldCount > errorLimit) {
			return property("schema-error-count", "%d at limit %d: %v", fieldCount, errorLimit, err)
		}
		for index := range min(errorLimit, fieldCount) {
			if fields[fmt.Sprintf("A%02d", index)] == nil {
				return property("schema-error-order", "missing sorted field %d", index)
			}
		}
	}
	return nil
}

// checkSliceBudget checks that a bind-call budget is shared across slices.
//
// Takes astBinder (*core.ASTBinder) which is the production binder under test.
// Takes choices (*tape.Tape) which supplies bounded deterministic choices.
//
// Returns error which reports a model mismatch or failed production operation, or nil
// when the check passes.
func checkSliceBudget(ctx context.Context, astBinder *core.ASTBinder, choices *tape.Tape) error {
	type shipment struct {
		First []int

		Second []int

		Notes []string
	}
	first := minFirstSliceLength + drawInt(choices, firstSliceSpread)
	for _, tc := range []struct{ total, notes int }{
		{total: sliceElementBudget - 1, notes: 0},
		{total: sliceElementBudget, notes: 0},
		{total: sliceElementBudget + 1, notes: 0},
		{total: sliceElementBudget + 1, notes: budgetNotes},
	} {
		second := tc.total - first
		input := map[string][]string{
			fmt.Sprintf("First[%d]", first-1):   {strconv.Itoa(firstSliceMarker)},
			fmt.Sprintf("Second[%d]", second-1): {strconv.Itoa(secondSliceMarker)},
		}
		if tc.notes > 0 {
			input["Notes"] = strings.Fields(strings.Repeat("label ", tc.notes))
		}
		var destination shipment
		err := astBinder.Bind(ctx, &destination, input, core.WithMaxSliceSize(sliceElementBudget))
		if tc.total > sliceElementBudget && tc.notes == 0 {
			if !errors.Is(err, core.ErrSliceElementBudgetExhausted) {
				return property("schema-slice-budget", "expected shared slice limit, got %v", err)
			}
			continue
		}
		if err != nil || len(destination.First) != first || len(destination.Second) != second ||
			destination.First[first-1] != firstSliceMarker || destination.Second[second-1] != secondSliceMarker ||
			len(destination.Notes) != tc.notes {
			return property("schema-slice-budget", "%d elements and %d notes: got lengths %d, %d, %d (%v)",
				tc.total, tc.notes, len(destination.First), len(destination.Second), len(destination.Notes), err)
		}
	}
	return nil
}
