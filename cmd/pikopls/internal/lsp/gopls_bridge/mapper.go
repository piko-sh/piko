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

package gopls_bridge

import (
	protocol "github.com/politepixels/golang-language-server"
	"piko.sh/piko/wdk/safeconv"
)

// Mapper translates positions and ranges between a .pk file and the virtual Go document
// generated from its <script type="application/x-go"> block.
type Mapper struct {
	// realURI is the .pk document URI that the editor sees.
	realURI protocol.DocumentURI

	// virtualURI is the synthetic Go document URI presented to gopls.
	virtualURI protocol.DocumentURI

	// contentLine is the 1-based .pk line where the Go block content starts.
	contentLine int

	// contentColumn is the 1-based UTF-16 column on contentLine where the Go content begins.
	contentColumn int
}

// NewMapper builds a mapper for one Go block.
//
// Takes realURI (protocol.DocumentURI) which is the .pk document URI the editor sees.
// Takes virtualURI (protocol.DocumentURI) which is the synthetic Go document URI for
// gopls.
// Takes contentLine (int) which is the 1-based .pk line where the Go block content
// starts.
// Takes contentColumn (int) which is the 1-based UTF-16 column on that line where the Go
// content begins (see firstContentLineUTF16Column at the call site).
//
// Returns *Mapper which translates positions and ranges between the two documents.
func NewMapper(realURI, virtualURI protocol.DocumentURI, contentLine, contentColumn int) *Mapper {
	return &Mapper{
		realURI:       realURI,
		virtualURI:    virtualURI,
		contentLine:   contentLine,
		contentColumn: contentColumn,
	}
}

// RealURI returns the .pk document URI.
//
// Returns protocol.DocumentURI which is the .pk document URI.
func (m *Mapper) RealURI() protocol.DocumentURI {
	return m.realURI
}

// VirtualURI returns the synthetic Go document URI.
//
// Returns protocol.DocumentURI which is the synthetic Go document URI.
func (m *Mapper) VirtualURI() protocol.DocumentURI {
	return m.virtualURI
}

// ToVirtual maps a .pk position into the virtual Go document.
//
// Takes position (protocol.Position) which is the .pk position to translate.
//
// Returns protocol.Position which is the equivalent position in the virtual Go document.
func (m *Mapper) ToVirtual(position protocol.Position) protocol.Position {
	lineOffset := m.contentLine - 1
	virtualLine := int(position.Line) - lineOffset
	virtualCharacter := int(position.Character)
	if virtualLine == 0 {
		virtualCharacter -= m.contentColumn - 1
	}
	return protocol.Position{
		Line:      safeconv.IntToUint32(virtualLine),
		Character: safeconv.IntToUint32(virtualCharacter),
	}
}

// ToReal maps a virtual Go document position back into the .pk file.
//
// Takes position (protocol.Position) which is the virtual Go document position to
// translate.
//
// Returns protocol.Position which is the equivalent position in the .pk file.
func (m *Mapper) ToReal(position protocol.Position) protocol.Position {
	lineOffset := m.contentLine - 1
	realLine := int(position.Line) + lineOffset
	realCharacter := int(position.Character)
	if position.Line == 0 {
		realCharacter += m.contentColumn - 1
	}
	return protocol.Position{
		Line:      safeconv.IntToUint32(realLine),
		Character: safeconv.IntToUint32(realCharacter),
	}
}

// RangeToReal maps a virtual Go document range back into the .pk file.
//
// Takes rng (protocol.Range) which is the virtual Go document range to translate.
//
// Returns protocol.Range which is the equivalent range in the .pk file.
func (m *Mapper) RangeToReal(rng protocol.Range) protocol.Range {
	return protocol.Range{
		Start: m.ToReal(rng.Start),
		End:   m.ToReal(rng.End),
	}
}

// RangeToVirtual maps a .pk range into the virtual Go document.
//
// Takes rng (protocol.Range) which is the .pk range to translate.
//
// Returns protocol.Range which is the equivalent range in the virtual Go document.
func (m *Mapper) RangeToVirtual(rng protocol.Range) protocol.Range {
	return protocol.Range{
		Start: m.ToVirtual(rng.Start),
		End:   m.ToVirtual(rng.End),
	}
}
