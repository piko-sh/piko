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

package db_engine_clickhouse

import (
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// ipv6FixedStringBytes is the byte width of an IPv6 address. The reinterpretation and
	// conversion helpers use this width when modelling IPv6 values as FixedString to
	// preserve the precise byte layout for downstream emitters.
	ipv6FixedStringBytes = 16
)

// registerIPValidatorFunctions covers the extended IPv4 and IPv6 validation,
// CIDR-to-range conversion, classful display, validator, and safety conversion helpers
// that surface in ClickHouse for network analytics.
//
// The set complements registerURLAndIPFunctions which holds the core IPv4ToString and
// IPv6ToString family.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPValidatorFunctions(b *FunctionCatalogueBuilder) {
	registerIPRangeAndDisplayHelpers(b)
	registerIPStringSafetyConversions(b)
	registerIPv6Helpers(b)
	registerIPAddressInRangeAndValidators(b)
	registerIPSafetyToHelpers(b)
}

// registerIPRangeAndDisplayHelpers covers IPv4ToIPv6, the CIDR range expansion helpers,
// and IPv4NumToStringClassC.
//
// The CIDR helpers return a tuple of (low, high) addresses; the catalogue encodes the
// tuple shape with StructFields so emitters can decode the matched component types.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPRangeAndDisplayHelpers(b *FunctionCatalogueBuilder) {
	ipv4 := ipv4Type()
	ipv6 := ipv6Type()
	b.Register("IPv4ToIPv6", fixedStringType(ipv6FixedStringBytes), ipv4)
	ipv4Range := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "low", SQLType: ipv4},
			{Name: "high", SQLType: ipv4},
		},
	}
	ipv6Range := querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "low", SQLType: ipv6},
			{Name: "high", SQLType: ipv6},
		},
	}
	b.Register("IPv4CIDRToRange", ipv4Range, ipv4, b.uint64Type)
	b.Register("IPv6CIDRToRange", ipv6Range, ipv6, b.uint64Type)
	b.Register("IPv4NumToStringClassC", b.textType, b.uint64Type)
}

// registerIPStringSafetyConversions covers the OrDefault and OrNull safety variants of
// IPv4StringToNum and IPv6StringToNum.
//
// The base forms live in registerURLAndIPFunctions; the safety variants land here so the
// catalogue covers every failure-mode spelling.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPStringSafetyConversions(b *FunctionCatalogueBuilder) {
	for _, suffix := range defaultNullSafetyConversionSuffixes {
		b.Register("IPv4StringToNum"+suffix, b.uint32Type, b.textType)
		b.Register("IPv6StringToNum"+suffix, b.textType, b.textType)
	}
}

// registerIPv6Helpers covers IPv6NumToString and the cutIPv6 helper that drops trailing
// octets for anonymisation use cases.
//
// cutIPv6 takes the address plus the number of IPv6 and IPv4 octets to drop.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPv6Helpers(b *FunctionCatalogueBuilder) {
	ipv6 := ipv6Type()
	b.Register("IPv6NumToString", b.textType, ipv6)
	b.Register("cutIPv6", b.textType, ipv6, b.uint64Type, b.uint64Type)
}

// registerIPAddressInRangeAndValidators covers isIPAddressInRange (CIDR membership test)
// and the isIPv4String and isIPv6String parser validators.
//
// Each returns a Bool because the predicates carry plain true or false semantics;
// downstream emitters promote the value to the engine-native UInt8 width when
// serialising.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPAddressInRangeAndValidators(b *FunctionCatalogueBuilder) {
	b.Register("isIPAddressInRange", b.boolType, b.textType, b.textType)
	b.Register("isIPv4String", b.boolType, b.textType)
	b.Register("isIPv6String", b.boolType, b.textType)
}

// registerIPSafetyToHelpers covers the toIPv4 and toIPv6 safety conversion variants.
//
// Each returns the canonical IPv4 or IPv6 network type with the failure-mode suffix
// encoded in the function name.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerIPSafetyToHelpers(b *FunctionCatalogueBuilder) {
	ipv4 := ipv4Type()
	ipv6 := ipv6Type()
	for _, suffix := range safetyConversionSuffixes {
		b.Register("toIPv4"+suffix, ipv4, b.textType)
		b.Register("toIPv6"+suffix, ipv6, b.textType)
	}
}

// ipv4Type constructs the ClickHouse IPv4 network type.
//
// Lifted to a package-level helper so every IP family registration site shares a single
// source of truth.
//
// Returns querier_dto.SQLType which is the ClickHouse IPv4 network type.
func ipv4Type() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryNetwork, EngineName: "IPv4"}
}

// ipv6Type constructs the ClickHouse IPv6 network type.
//
// Lifted to a package-level helper so every IP family registration site shares a single
// source of truth.
//
// Returns querier_dto.SQLType which is the ClickHouse IPv6 network type.
func ipv6Type() querier_dto.SQLType {
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryNetwork, EngineName: "IPv6"}
}
