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

package crypto_dto

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProviderType_IsValid(t *testing.T) {
	valid := []ProviderType{
		ProviderTypeLocalAESGCM, ProviderTypeAWSKMS, ProviderTypeGCPKMS,
		ProviderTypeAzureKeyVault, ProviderTypeHashiCorpVault,
	}
	for _, p := range valid {
		if !p.IsValid() {
			t.Errorf("IsValid() = false for %q", p)
		}
	}
	if ProviderType("unknown").IsValid() {
		t.Error("IsValid() should be false for unknown provider")
	}
}

func TestKeyInfo_IsUsable(t *testing.T) {
	tests := []struct {
		status KeyStatus
		want   bool
	}{
		{status: KeyStatusActive, want: true},
		{status: KeyStatusDeprecated, want: true},
		{status: KeyStatusDisabled, want: false},
		{status: KeyStatusDestroyed, want: false},
	}
	for _, tt := range tests {
		k := &KeyInfo{Status: tt.status}
		if got := k.IsUsable(); got != tt.want {
			t.Errorf("IsUsable(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestKeyInfo_CanDecrypt(t *testing.T) {
	tests := []struct {
		status KeyStatus
		want   bool
	}{
		{status: KeyStatusActive, want: true},
		{status: KeyStatusDeprecated, want: true},
		{status: KeyStatusDisabled, want: false},
		{status: KeyStatusDestroyed, want: false},
	}
	for _, tt := range tests {
		k := &KeyInfo{Status: tt.status}
		if got := k.CanDecrypt(); got != tt.want {
			t.Errorf("CanDecrypt(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestEncryptionError_Error(t *testing.T) {
	err := NewEncryptionError("Encrypt", "aws_kms", "key-123", errors.New("timeout"))
	message := err.Error()
	if !strings.Contains(message, "key-123") {
		t.Errorf("error should contain key ID: %s", message)
	}
	if !strings.Contains(message, "aws_kms") {
		t.Errorf("error should contain provider: %s", message)
	}

	err2 := NewEncryptionError("Decrypt", "local", "", errors.New("failed"))
	msg2 := err2.Error()
	if strings.Contains(msg2, "key") {
		t.Errorf("error without key ID should not mention key: %s", msg2)
	}
}

func TestEncryptionError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	err := NewEncryptionError("Encrypt", "local", "", inner)
	if !errors.Is(err, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestDefaultKeyRotationPolicy(t *testing.T) {
	p := DefaultKeyRotationPolicy()
	if p.Enabled {
		t.Error("default policy should have Enabled=false")
	}
	if p.AutoRotate {
		t.Error("default policy should have AutoRotate=false")
	}
	if p.RotationInterval != 90*24*time.Hour {
		t.Errorf("RotationInterval = %v, want 90 days", p.RotationInterval)
	}
	if p.RetainOldKeys != 5 {
		t.Errorf("RetainOldKeys = %d, want 5", p.RetainOldKeys)
	}
	if p.GracePeriod != 7*24*time.Hour {
		t.Errorf("GracePeriod = %v, want 7 days", p.GracePeriod)
	}
}

func TestSecureBytes_ReadAfterClose(t *testing.T) {
	secureBytes, err := NewSecureBytesFromSlice([]byte("secret"), WithID("test"))
	if err != nil {
		t.Fatalf("NewSecureBytesFromSlice: %v", err)
	}

	if secureBytes.ID() != "test" {
		t.Errorf("ID() = %q, want %q", secureBytes.ID(), "test")
	}
	if secureBytes.Len() != 6 {
		t.Errorf("Len() = %d, want 6", secureBytes.Len())
	}

	buffer := make([]byte, 6)
	n, _ := secureBytes.Read(buffer)
	if n != 6 || string(buffer) != "secret" {
		t.Errorf("Read() = %d, %q; want 6, %q", n, buffer, "secret")
	}

	if err := secureBytes.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !secureBytes.IsClosed() {
		t.Error("IsClosed() should be true after Close")
	}

	_, err = secureBytes.Read(buffer)
	if err == nil {
		t.Error("Read after Close should fail")
	}

	if err := secureBytes.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestSecureBytes_WithAccess(t *testing.T) {
	secureBytes, err := NewSecureBytesFromSlice([]byte("hello"))
	if err != nil {
		t.Fatalf("NewSecureBytesFromSlice: %v", err)
	}
	defer func() { _ = secureBytes.Close() }()

	var got string
	if err := secureBytes.WithAccess(func(data []byte) error {
		got = string(data)
		return nil
	}); err != nil {
		t.Fatalf("WithAccess: %v", err)
	}
	if got != "hello" {
		t.Errorf("WithAccess data = %q, want %q", got, "hello")
	}

	_ = secureBytes.Close()
	err = secureBytes.WithAccess(func([]byte) error { return nil })
	if err == nil {
		t.Error("WithAccess after Close should fail")
	}
}

func TestSecureBytes_Clone(t *testing.T) {
	secureBytes, err := NewSecureBytesFromSlice([]byte("data"), WithID("orig"))
	if err != nil {
		t.Fatalf("NewSecureBytesFromSlice: %v", err)
	}
	defer func() { _ = secureBytes.Close() }()

	clone, err := secureBytes.Clone()
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer func() { _ = clone.Close() }()

	if clone.ID() != "orig-clone" {
		t.Errorf("Clone ID = %q, want %q", clone.ID(), "orig-clone")
	}
	if clone.Len() != secureBytes.Len() {
		t.Errorf("Clone Len = %d, want %d", clone.Len(), secureBytes.Len())
	}

	_ = secureBytes.Close()
	_, err = secureBytes.Clone()
	if err == nil {
		t.Error("Clone after Close should fail")
	}
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()
	if config.ActiveKeyID != "default" {
		t.Errorf("ActiveKeyID = %q, want %q", config.ActiveKeyID, "default")
	}
	if config.ProviderType != ProviderTypeLocalAESGCM {
		t.Errorf("ProviderType = %q, want %q", config.ProviderType, ProviderTypeLocalAESGCM)
	}
	if config.EnableAutoReEncrypt {
		t.Error("EnableAutoReEncrypt should be false")
	}
	if !config.EnableEnvelopeEncryption {
		t.Error("EnableEnvelopeEncryption should be true")
	}
	if config.DirectModeMaxConcurrency != 50 {
		t.Errorf("DirectModeMaxConcurrency = %d, want 50", config.DirectModeMaxConcurrency)
	}
	if config.DataKeyCacheTTL != 5*time.Minute {
		t.Errorf("DataKeyCacheTTL = %v, want 5m", config.DataKeyCacheTTL)
	}
	if config.DataKeyCacheMaxSize != 100 {
		t.Errorf("DataKeyCacheMaxSize = %d, want 100", config.DataKeyCacheMaxSize)
	}
}

func TestKeyInfo_CanEncrypt(t *testing.T) {
	tests := []struct {
		status KeyStatus
		want   bool
	}{
		{status: KeyStatusActive, want: true},
		{status: KeyStatusDeprecated, want: false},
		{status: KeyStatusDisabled, want: false},
		{status: KeyStatusDestroyed, want: false},
	}
	for _, tt := range tests {
		k := &KeyInfo{Status: tt.status}
		if got := k.CanEncrypt(); got != tt.want {
			t.Errorf("CanEncrypt(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestProviderType_String(t *testing.T) {
	if s := ProviderTypeLocalAESGCM.String(); s != "local_aes_gcm" {
		t.Errorf("String() = %q, want %q", s, "local_aes_gcm")
	}
}

func TestKeyStatus_String(t *testing.T) {
	if s := KeyStatusActive.String(); s != "active" {
		t.Errorf("String() = %q, want %q", s, "active")
	}
}

func TestZeroMemory(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	zeroMemory(data)
	for i, b := range data {
		if b != 0 {
			t.Errorf("data[%d] = %d, want 0", i, b)
		}
	}
}
