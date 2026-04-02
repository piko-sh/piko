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

// Package crypto_provider_aws_kms provides an AWS Key Management Service
// (KMS) encryption provider.
//
// This provider delegates all cryptographic operations to AWS KMS.
// Master keys never leave AWS's Hardware Security Modules (HSMs).
// It supports both direct encryption and envelope encryption
// for efficient bulk operations, as well as streaming
// encryption for large files with constant memory usage.
//
// Authentication uses the AWS SDK's default credential chain.
// The KMS principal needs the kms:Encrypt, kms:Decrypt,
// kms:GenerateDataKey, and kms:DescribeKey permissions.
//
// # Thread safety
//
// All methods on [Provider] are safe for concurrent use. The
// underlying AWS KMS client manages connection pooling internally.
package crypto_provider_aws_kms
