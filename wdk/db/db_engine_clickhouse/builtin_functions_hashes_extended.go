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
	// md4DigestBytes is the byte length of an MD4 hash digest. The base SHA and MD5 lengths
	// live in builtin_functions.go; this value is kept local because only the extended hash
	// registrations consume it.
	md4DigestBytes = 16

	// sha512Slash256DigestBytes is the byte length of the SHA-512/256 truncated digest.
	sha512Slash256DigestBytes = 32

	// ripemd160DigestBytes is the byte length of a RIPEMD-160 digest.
	ripemd160DigestBytes = 20

	// keccak256DigestBytes is the byte length of a Keccak-256 digest.
	keccak256DigestBytes = 32

	// blake3DigestBytes is the byte length of the default BLAKE3 digest.
	blake3DigestBytes = 32

	// sipHash128DigestBytes is the byte length of a SipHash-128 digest.
	sipHash128DigestBytes = 16

	// murmurHash3X128DigestBytes is the byte length of the MurmurHash3 128-bit digest.
	murmurHash3X128DigestBytes = 16

	// xxh3X128DigestBytes is the byte length of the XXH3 128-bit digest.
	xxh3X128DigestBytes = 16
)

// registerExtendedHashFunctions covers the long tail of hash variants beyond the base
// SHA, MD5, cityHash, and xxHash families already registered by registerHashingFunctions.
//
// It delegates to topical helpers so each function stays within the linter budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedHashFunctions(b *FunctionCatalogueBuilder) {
	registerExtendedCryptographicDigests(b)
	registerExtendedFastIntegerHashes(b)
	registerExtendedSipHashFamily(b)
	registerConsistentAndDistributedHashes(b)
	registerNgramAndShingleHashes(b)
}

// registerExtendedCryptographicDigests covers MD4, SHA-512/256, RIPEMD-160, Keccak-256,
// and BLAKE3.
//
// Each returns a FixedString of the digest's native byte length so callers receive
// untruncated bytes even when the digest contains embedded NUL bytes.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedCryptographicDigests(b *FunctionCatalogueBuilder) {
	b.Register("MD4", fixedStringType(md4DigestBytes), b.textType)
	b.Register("SHA512_256", fixedStringType(sha512Slash256DigestBytes), b.textType)
	b.Register("RIPEMD160", fixedStringType(ripemd160DigestBytes), b.textType)
	b.Register("keccak256", fixedStringType(keccak256DigestBytes), b.textType)
	b.Register("BLAKE3", fixedStringType(blake3DigestBytes), b.textType)
}

// registerExtendedFastIntegerHashes covers the non-cryptographic integer-result hashes
// (xxh3, wyHash64, metroHash64, intHash32, murmurHash variants, farmFingerprint64,
// URLHash) used for partitioning, sharding, and dictionary lookups.
//
// intHash32 and murmurHash2_32 return UInt32 in ClickHouse; the wider 64-bit variants
// live alongside in registerHashingFunctions.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedFastIntegerHashes(b *FunctionCatalogueBuilder) {
	b.Register("xxh3", b.uint64Type, b.unknownType)
	b.Register("xxh3_128", fixedStringType(xxh3X128DigestBytes), b.unknownType)
	b.Register("wyHash64", b.uint64Type, b.unknownType)
	b.Register("metroHash64", b.uint64Type, b.unknownType)
	b.Register("intHash32", b.uint32Type, b.unknownType)
	b.Register("murmurHash2_32", b.uint32Type, b.unknownType)
	b.Register("murmurHash3_128", fixedStringType(murmurHash3X128DigestBytes), b.unknownType)
	b.Register("farmFingerprint64", b.uint64Type, b.unknownType)
	b.Register("URLHash", b.uint64Type, b.textType)
	b.Register("URLHash", b.uint64Type, b.textType, b.uint64Type)
}

// registerExtendedSipHashFamily covers the SipHash 128 variants and the keyed forms
// beyond the existing sipHash64 registration.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerExtendedSipHashFamily(b *FunctionCatalogueBuilder) {
	b.Register("sipHash64Keyed", b.uint64Type, b.unknownType, b.unknownType)
	b.Register("sipHash128", fixedStringType(sipHash128DigestBytes), b.unknownType)
	b.Register("sipHash128Keyed", fixedStringType(sipHash128DigestBytes), b.unknownType, b.unknownType)
	b.Register("sipHash128Reference", fixedStringType(sipHash128DigestBytes), b.unknownType)
	b.Register("sipHash128ReferenceKeyed", fixedStringType(sipHash128DigestBytes), b.unknownType, b.unknownType)
}

// registerConsistentAndDistributedHashes covers the consistent-hash and
// ecosystem-specific hashes used for downstream system compatibility.
//
// The ecosystem hashes target Hive, Iceberg, Java, Kafka, and GCC. The Hive integer
// returns are unsigned in ClickHouse but signed in the downstream protocols; the
// catalogue records the canonical engine-native return type for each entry. The
// consistent-hash `buckets` argument is signed (Int32) in ClickHouse because the
// underlying API accepts negative sentinels for default-bucket signalling.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerConsistentAndDistributedHashes(b *FunctionCatalogueBuilder) {
	b.Register("hiveHash", b.uint64Type, b.textType)
	b.Register("icebergHash", b.int32Type, b.textType)
	b.Register("javaHash", b.int32Type, b.textType)
	b.Register("javaHashUTF16LE", b.int32Type, b.textType)
	b.Register("jumpConsistentHash", b.int32Type, b.uint64Type, b.int32Type)
	b.Register("kostikConsistentHash", b.int32Type, b.uint64Type, b.int32Type)
	b.Register("kafkaMurmurHash", b.int32Type, b.textType)
	b.Register("gccMurmurHash", b.int64Type, b.textType)
}

// registerNgramAndShingleHashes covers the family of n-gram and word shingle min and sim
// hashes used for fuzzy similarity.
//
// Each spelling (the case and UTF-8 variants) is registered separately because ClickHouse
// exposes them as distinct functions.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue builder to register into.
func registerNgramAndShingleHashes(b *FunctionCatalogueBuilder) {
	tupleType := ngramMinHashTupleType(b)
	for _, name := range []string{"ngramMinHash", "ngramMinHashCaseInsensitive", "ngramMinHashUTF8", "ngramMinHashCaseInsensitiveUTF8"} {
		b.Register(name, tupleType, b.textType, b.uint64Type, b.uint64Type)
	}
	for _, name := range []string{"ngramSimHash", "ngramSimHashCaseInsensitive", "ngramSimHashUTF8", "ngramSimHashCaseInsensitiveUTF8"} {
		b.Register(name, b.uint64Type, b.textType, b.uint64Type)
	}
	for _, name := range []string{
		"wordShingleMinHash", "wordShingleMinHashCaseInsensitive",
		"wordShingleMinHashUTF8", "wordShingleMinHashCaseInsensitiveUTF8",
		"wordShingleMinHashArg", "wordShingleMinHashArgCaseInsensitive",
		"wordShingleMinHashArgUTF8", "wordShingleMinHashArgCaseInsensitiveUTF8",
	} {
		b.Register(name, tupleType, b.textType, b.uint64Type, b.uint64Type)
	}
	for _, name := range []string{
		"wordShingleSimHash", "wordShingleSimHashCaseInsensitive",
		"wordShingleSimHashUTF8", "wordShingleSimHashCaseInsensitiveUTF8",
	} {
		b.Register(name, b.uint64Type, b.textType, b.uint64Type)
	}
}

// ngramMinHashTupleType constructs the Tuple(UInt64, UInt64) shape returned by the
// ngram-MinHash and word-shingle MinHash family.
//
// The tuple's first slot holds the minimum-hash bucket and the second holds the
// maximum-hash bucket; downstream consumers use both ends when computing similarity
// scores.
//
// Takes b (*FunctionCatalogueBuilder) which supplies the uint64 element type.
//
// Returns querier_dto.SQLType which is the Tuple(UInt64, UInt64) struct type.
func ngramMinHashTupleType(b *FunctionCatalogueBuilder) querier_dto.SQLType {
	return querier_dto.SQLType{
		Category:   querier_dto.TypeCategoryStruct,
		EngineName: "Tuple",
		StructFields: []querier_dto.StructField{
			{Name: "min", SQLType: b.uint64Type},
			{Name: "max", SQLType: b.uint64Type},
		},
	}
}
