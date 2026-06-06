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

// registerExtendedEncodingFunctions covers character-from-codepoint helpers, bit position
// helpers, space-filling curve encoders, the Sqid and Bech32 string identifier encoders,
// and the IDNA and Punycode hostname encoding family.
//
// It delegates to topical helpers so each function stays within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedEncodingFunctions(b *FunctionCatalogueBuilder) {
	registerCharAndBitmaskFunctions(b)
	registerSpaceFillingCurveFunctions(b)
	registerStringIdentifierEncoders(b)
	registerExtendedBaseEncoders(b)
	registerIDNAAndPunycodeFunctions(b)
	registerTryDecryptFunction(b)
}

// registerCharAndBitmaskFunctions covers char (build string from byte codes),
// bitmaskToArray, bitmaskToList and the canonical bitPositionsToArray alias.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerCharAndBitmaskFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("char", b.textType, 1, b.uint64Type)
	b.Register("bitmaskToArray", arrayOf(b.uint64Type), b.int64Type)
	b.Register("bitmaskToList", b.textType, b.int64Type)
	b.Register("bitPositionsToArray", arrayOf(b.uint64Type), b.int64Type)
}

// registerSpaceFillingCurveFunctions covers Morton and Hilbert encoders and decoders.
//
// The encoder accepts an arbitrary number of co-ordinates; the decoder takes the
// dimensionality followed by the encoded value.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerSpaceFillingCurveFunctions(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("mortonEncode", b.uint64Type, 1, b.uint64Type)
	b.Register("mortonDecode", b.unknownType, b.uint64Type, b.uint64Type)
	b.RegisterVariadic("hilbertEncode", b.uint64Type, 1, b.uint64Type)
	b.Register("hilbertDecode", b.unknownType, b.uint64Type, b.uint64Type)
}

// registerStringIdentifierEncoders covers Sqid (variable list of integers to compact
// string) and Bech32 (human-readable prefix plus data part) identifier encoders.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerStringIdentifierEncoders(b *FunctionCatalogueBuilder) {
	b.RegisterVariadic("sqidEncode", b.textType, 1, b.uint64Type)
	b.Register("sqidDecode", arrayOf(b.uint64Type), b.textType)
	b.Register("bech32Encode", b.textType, b.textType, b.textType)
	b.Register("bech32Decode", b.unknownType, b.textType)
}

// registerExtendedBaseEncoders covers base32, base58 and base64URL encoders and decoders
// alongside their try variants which return NULL on decode failure.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedBaseEncoders(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"base32Encode", "base32Decode", "tryBase32Decode",
		"base58Encode", "base58Decode", "tryBase58Decode",
		"base64URLEncode", "base64URLDecode", "tryBase64URLDecode",
	} {
		b.Register(name, b.textType, b.textType)
	}
}

// registerIDNAAndPunycodeFunctions covers the IDNA and Punycode helpers used to convert
// between Unicode hostnames and their ASCII-safe wire-format encoding.
//
// The try variants return NULL on parse failure rather than raising an error.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerIDNAAndPunycodeFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"idnaDecode", "idnaEncode", "tryIdnaEncode", "punycodeDecode", "punycodeEncode", "tryPunycodeDecode"} {
		b.Register(name, b.textType, b.textType)
	}
}

// registerTryDecryptFunction covers tryDecrypt which returns the plaintext on success or
// NULL on failure.
//
// The arity range mirrors decrypt, with and without iv and aad.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerTryDecryptFunction(b *FunctionCatalogueBuilder) {
	b.Register("tryDecrypt", b.textType, b.textType, b.textType, b.textType)
	b.Register("tryDecrypt", b.textType, b.textType, b.textType, b.textType, b.textType)
	b.Register("tryDecrypt", b.textType, b.textType, b.textType, b.textType, b.textType, b.textType)
}
