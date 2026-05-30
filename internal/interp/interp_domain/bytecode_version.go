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

package interp_domain

const (
	// BytecodeVersionMajor identifies the major bytecode format version.
	//
	// A change in major version indicates that the instruction encoding, opcode iota
	// assignments, or operand layout has changed in a way that is not backwards-compatible:
	// bytecode emitted at a different major version cannot be decoded by this interpreter
	// and is rejected on load.
	//
	// The serialised bytecode stream itself does not carry the version constant directly;
	// the interp_schema package embeds it in the FlatBuffer schema hash so cached bytecode
	// from incompatible versions auto-invalidates via fbs.ErrSchemaVersionMismatch on load.
	BytecodeVersionMajor uint16 = 6

	// BytecodeVersionMinor identifies an additive bytecode revision.
	//
	// Bumped when new opcodes or sub-opcodes are appended without disturbing existing iota
	// assignments. A minor-version difference is not a load-time error; the loader ignores
	// trailing opcodes it does not recognise.
	BytecodeVersionMinor uint16 = 2
)
