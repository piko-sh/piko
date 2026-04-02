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

package pdfwriter_dto

// TransformerType categorises PDF transformers into groups that determine
// their role in the transformation pipeline.
type TransformerType string

const (
	// TransformerContent applies content-level mutations such as redaction,
	// flattening, and watermarking.
	TransformerContent TransformerType = "content"

	// TransformerCompliance applies compliance conversions such as PDF/A
	// and PDF/UA.
	TransformerCompliance TransformerType = "compliance"

	// TransformerDelivery applies delivery optimisations such as
	// linearisation and object stream compression.
	TransformerDelivery TransformerType = "delivery"

	// TransformerSecurity applies security operations such as encryption
	// and digital signatures.
	TransformerSecurity TransformerType = "security"

	// TransformerCompression applies byte-level compression to the final
	// PDF output. This should run after all other transformers since the
	// output is no longer a valid PDF.
	TransformerCompression TransformerType = "compression"
)
