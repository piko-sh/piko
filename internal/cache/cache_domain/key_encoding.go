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

package cache_domain

import (
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// EncodeKey converts a key of type K to a string suitable for use as a cache
// key in any provider.
//
// It handles three strategies in order of priority:
//  1. String keys: fast path, returned directly (then namespace-prefixed).
//  2. No KeyRegistry: uses fmt.Sprintf (backward compatible for primitives).
//  3. KeyRegistry available: uses registry with base64 encoding for struct
//     keys.
//
// All keys are prefixed with the namespace if configured.
//
// Takes key (K) which is the cache key to encode.
// Takes namespace (string) which is the namespace prefix to prepend.
// Takes keyRegistry (*EncodingRegistry) which provides type-specific encoders,
// or nil for primitive keys.
//
// Returns string which is the encoded key, with namespace prefix if set.
// Returns error when no encoder is registered for the key type or when
// marshalling fails.
func EncodeKey[K comparable](key K, namespace string, keyRegistry *EncodingRegistry) (string, error) {
	var keyString string

	if strKey, ok := any(key).(string); ok {
		keyString = strKey
	} else if keyRegistry == nil {
		keyString = fmt.Sprintf("%v", key)
	} else {
		encoder, err := keyRegistry.Get(key)
		if err != nil {
			return "", fmt.Errorf("no encoder registered for key type %T: %w", key, err)
		}

		keyBytes, err := encoder.MarshalAny(key)
		if err != nil {
			return "", fmt.Errorf("failed to marshal key of type %T: %w", key, err)
		}

		keyString = base64.RawURLEncoding.EncodeToString(keyBytes)
	}

	if namespace != "" {
		return namespace + keyString, nil
	}

	return keyString, nil
}

// DecodeKey converts a cache key string back to a key of type K.
// It reverses the encoding strategy used in EncodeKey: strips namespace prefix
// if present, uses direct conversion for string keys, fmt.Sscan for primitives
// without a registry, or base64 decode with registry unmarshal for structs.
//
// Takes keyString (string) which is the cache key to decode.
// Takes namespace (string) which is the namespace prefix to strip.
// Takes keyRegistry (*EncodingRegistry) which provides type-specific decoders,
// or nil for primitive keys.
//
// Returns K which is the decoded key value.
// Returns error when the namespace prefix is missing, decoding fails, or no
// encoder is registered for the key type.
func DecodeKey[K comparable](keyString string, namespace string, keyRegistry *EncodingRegistry) (K, error) {
	var key K

	if namespace != "" {
		if !strings.HasPrefix(keyString, namespace) {
			return key, fmt.Errorf("key %q missing expected namespace prefix %q", keyString, namespace)
		}
		keyString = strings.TrimPrefix(keyString, namespace)
	}

	if _, ok := any(key).(string); ok {
		result, ok := any(keyString).(K)
		if !ok {
			return key, fmt.Errorf("unexpected type assertion failure for string key %q", keyString)
		}
		return result, nil
	}

	if keyRegistry == nil {
		if _, err := fmt.Sscan(keyString, &key); err != nil {
			return key, fmt.Errorf("failed to scan key %q into type %T: %w", keyString, key, err)
		}
		return key, nil
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(keyString)
	if err != nil {
		return key, fmt.Errorf("failed to base64-decode key %q: %w", keyString, err)
	}

	keyType := reflect.TypeOf(key)
	encoder, err := keyRegistry.GetByType(keyType)
	if err != nil {
		return key, fmt.Errorf("no encoder registered for key type %T: %w", key, err)
	}

	unmarshalled, err := encoder.UnmarshalAny(keyBytes)
	if err != nil {
		return key, fmt.Errorf("failed to unmarshal key bytes into type %T: %w", key, err)
	}

	result, ok := unmarshalled.(K)
	if !ok {
		return key, fmt.Errorf("type assertion failed: expected %T, got %T", key, unmarshalled)
	}

	return result, nil
}

// EncodeValue encodes a value of type V to bytes using the registry.
//
// Takes value (V) which is the value to encode.
// Takes registry (*EncodingRegistry) which provides type-specific encoders.
//
// Returns []byte which contains the encoded value.
// Returns error when no encoder is found for the value type or encoding fails.
func EncodeValue[V any](value V, registry *EncodingRegistry) ([]byte, error) {
	encoder, err := registry.Get(value)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoder for value: %w", err)
	}
	return encoder.MarshalAny(value)
}

// DecodeValue decodes bytes into a value of type V using the registry.
//
// Takes valBytes ([]byte) which contains the encoded data to decode.
// Takes registry (*EncodingRegistry) which provides type-specific decoders.
//
// Returns V which is the decoded value.
// Returns error when the encoder cannot be found, unmarshalling fails, or type
// assertion fails.
func DecodeValue[V any](valBytes []byte, registry *EncodingRegistry) (V, error) {
	var v V
	encoder, err := registry.GetByType(reflect.TypeOf(v))
	if err != nil {
		return v, fmt.Errorf("failed to get encoder: %w", err)
	}

	unmarshalled, err := encoder.UnmarshalAny(valBytes)
	if err != nil {
		return v, fmt.Errorf("failed to unmarshal: %w", err)
	}

	result, ok := unmarshalled.(V)
	if !ok {
		return v, errors.New("type assertion failed")
	}
	return result, nil
}
