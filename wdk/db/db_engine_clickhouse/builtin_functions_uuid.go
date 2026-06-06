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

const (
	// uuidFixedStringBytes is the byte width of a UUID when modelled as FixedString. UUIDs
	// occupy sixteen bytes regardless of variant, so the numeric conversion helpers share
	// this width.
	uuidFixedStringBytes = 16
)

// registerUUIDSnowflakeFunctions covers the UUID byte-form conversions, the UUIDv7
// timestamp helpers and the Snowflake-ID time conversion helpers.
//
// ClickHouse exposes these alongside the generators so analytics queries can both mint
// and decode the surrogate keys.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register on.
func registerUUIDSnowflakeFunctions(b *FunctionCatalogueBuilder) {
	registerUUIDStringNumConversions(b)
	registerUUIDV7AndSnowflakeID(b)
	registerSnowflakeToDateTime(b)
	registerUUIDSafetyConversions(b)
}

// registerUUIDStringNumConversions covers UUIDStringToNum, UUIDNumToString and UUIDToNum
// across the optional variant argument.
//
// Each variant pair documents whether the source is the dashed-string form or the raw
// UUID type; both reach the same FixedString(16) representation.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register on.
func registerUUIDStringNumConversions(b *FunctionCatalogueBuilder) {
	fixed := fixedStringType(uuidFixedStringBytes)
	b.Register("UUIDStringToNum", fixed, b.textType)
	b.Register("UUIDStringToNum", fixed, b.textType, b.uint64Type)
	b.Register("UUIDNumToString", b.textType, fixed)
	b.Register("UUIDNumToString", b.textType, fixed, b.uint64Type)
	b.Register("UUIDToNum", fixed, b.uuidType)
	b.Register("UUIDToNum", fixed, b.uuidType, b.uint64Type)
}

// registerUUIDV7AndSnowflakeID covers UUIDv7ToDateTime with optional precision and the
// dateTime to Snowflake-ID conversion helpers with optional epoch.
//
// UUIDv7 carries a millisecond timestamp prefix that maps cleanly onto a DateTime64.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register on.
func registerUUIDV7AndSnowflakeID(b *FunctionCatalogueBuilder) {
	b.Register("UUIDv7ToDateTime", b.dateTime64Type, b.uuidType)
	b.Register("UUIDv7ToDateTime", b.dateTime64Type, b.uuidType, b.uint64Type)
	b.Register("dateTimeToSnowflakeID", b.uint64Type, b.dateTimeType)
	b.Register("dateTimeToSnowflakeID", b.uint64Type, b.dateTimeType, b.uint64Type)
	b.Register("dateTime64ToSnowflakeID", b.uint64Type, b.dateTime64Type)
	b.Register("dateTime64ToSnowflakeID", b.uint64Type, b.dateTime64Type, b.uint64Type)
	b.Register("dateTimeToUUIDv7", b.uuidType, b.dateTimeType)
}

// registerSnowflakeToDateTime covers the inverse direction: snowflakeToDateTime and
// snowflakeToDateTime64.
//
// These are kept distinct from snowflakeIDToDateTime, registered in
// registerSystemFunctions, because ClickHouse exposes both spellings.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register on.
func registerSnowflakeToDateTime(b *FunctionCatalogueBuilder) {
	b.Register("snowflakeToDateTime", b.dateTimeType, b.uint64Type)
	b.Register("snowflakeToDateTime64", b.dateTime64Type, b.uint64Type)
}

// registerUUIDSafetyConversions covers toUUIDOrNull and toUUIDOrDefault.
//
// The base toUUID is registered in registerBaseConversions; the safety variants land here
// so the catalogue covers every failure-mode spelling.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register on.
func registerUUIDSafetyConversions(b *FunctionCatalogueBuilder) {
	b.Register("toUUIDOrNull", b.uuidType, b.textType)
	b.Register("toUUIDOrDefault", b.uuidType, b.textType)
}
