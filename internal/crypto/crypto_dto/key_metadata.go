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

import "time"

// KeyStatus represents the current state of an encryption key in its lifecycle.
type KeyStatus string

const (
	// defaultRotationIntervalDays is the number of days between key rotations.
	defaultRotationIntervalDays = 90

	// hoursPerDay is the number of hours in a day.
	hoursPerDay = 24

	// defaultRetainOldKeys is the number of old keys to keep after rotation.
	defaultRetainOldKeys = 5

	// defaultGracePeriodDays is the default grace period in days before marking
	// old keys as disabled.
	defaultGracePeriodDays = 7

	// KeyStatusActive indicates the key is in use for new encryptions.
	KeyStatusActive KeyStatus = "active"

	// KeyStatusDeprecated marks a key as no longer used for new encryptions but
	// still able to decrypt existing data.
	KeyStatusDeprecated KeyStatus = "deprecated"

	// KeyStatusDisabled indicates the key cannot be used for any operations.
	KeyStatusDisabled KeyStatus = "disabled"

	// KeyStatusDestroyed indicates the key has been permanently deleted and cannot
	// be recovered.
	KeyStatusDestroyed KeyStatus = "destroyed"
)

// String returns the string form of the key status.
//
// Returns string which is the underlying string value of the status.
func (k KeyStatus) String() string {
	return string(k)
}

// KeyInfo holds metadata about an encryption key.
type KeyInfo struct {
	// CreatedAt is when the key was created.
	CreatedAt time.Time

	// RotatedAt is when the key was last rotated; nil if never rotated.
	RotatedAt *time.Time

	// ExpirationDate is when the key will expire; nil means it does not expire.
	ExpirationDate *time.Time

	// DeletionDate is when the key is scheduled for deletion; nil if not
	// scheduled.
	DeletionDate *time.Time

	// Metadata contains extra key details specific to the provider.
	Metadata map[string]string

	// KeyID is the unique identifier for this key.
	KeyID string

	// Provider identifies which encryption provider manages this key.
	Provider ProviderType

	// Algorithm specifies the encryption method (e.g., "AES-256-GCM", "RSA-4096").
	Algorithm string

	// Status is the current lifecycle state of the key.
	Status KeyStatus

	// Description provides a short, readable explanation of what the key is used
	// for.
	Description string

	// Origin indicates where the key was created (e.g. "AWS_KMS", "LOCAL",
	// "HSM").
	Origin string
}

// IsUsable reports whether the key can be used for encryption or decryption.
//
// Returns bool which is true when the key status is active or deprecated.
func (k *KeyInfo) IsUsable() bool {
	return k.Status == KeyStatusActive || k.Status == KeyStatusDeprecated
}

// CanEncrypt reports whether the key can be used for new encryptions.
//
// Returns bool which is true when the key status is active.
func (k *KeyInfo) CanEncrypt() bool {
	return k.Status == KeyStatusActive
}

// CanDecrypt reports whether the key can be used for decryption.
//
// Returns bool which is true when the key status is active or deprecated.
func (k *KeyInfo) CanDecrypt() bool {
	return k.Status == KeyStatusActive || k.Status == KeyStatusDeprecated
}

// KeyRotationPolicy defines when and how cryptographic keys should be rotated.
type KeyRotationPolicy struct {
	// RotationInterval specifies how often keys are rotated (e.g. 90 days).
	RotationInterval time.Duration

	// RetainOldKeys specifies how many deprecated keys to keep. Set to 0 to keep
	// all old keys indefinitely.
	RetainOldKeys int

	// GracePeriod specifies how long to wait before marking old keys as disabled,
	// giving old data time to be re-encrypted without disruption.
	GracePeriod time.Duration

	// Enabled indicates whether key rotation is active.
	Enabled bool

	// AutoRotate indicates whether rotation happens automatically. If false,
	// rotation must be triggered manually.
	AutoRotate bool
}

// DefaultKeyRotationPolicy returns a rotation policy with sensible defaults.
//
// Returns *KeyRotationPolicy which has rotation disabled, no auto-rotation,
// and standard retention and grace period settings.
func DefaultKeyRotationPolicy() *KeyRotationPolicy {
	return &KeyRotationPolicy{
		Enabled:          false,
		RotationInterval: defaultRotationIntervalDays * hoursPerDay * time.Hour,
		AutoRotate:       false,
		RetainOldKeys:    defaultRetainOldKeys,
		GracePeriod:      defaultGracePeriodDays * hoursPerDay * time.Hour,
	}
}
