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

package htmllexer

// TokenType identifies the kind of token produced by the lexer. Values are
// sequential starting from zero so that array dispatch tables can be indexed
// directly by the token type.
type TokenType uint8

const (
	// ErrorToken signals end-of-input or a lexer error. Check Err() to
	// distinguish io.EOF from a parse failure.
	ErrorToken TokenType = iota

	// TextToken carries text content between tags.
	TextToken

	// StartTagToken marks an opening tag such as <div.
	//
	// The tag name is available via Text(). Attributes follow as subsequent
	// tokens.
	StartTagToken

	// EndTagToken marks a closing tag such as </div>. The tag name
	// (without the </ prefix and > suffix) is available via Text().
	EndTagToken

	// CommentToken carries an HTML comment. Text() returns the inner
	// content without the <!-- and --> delimiters.
	CommentToken

	// SVGToken carries an entire <svg>...</svg> block as a single token.
	// Text() returns the complete block including the opening and closing
	// tags.
	SVGToken

	// MathToken carries an entire <math>...</math> block as a single
	// token. Text() returns the complete block including the opening and
	// closing tags.
	MathToken

	// AttributeToken carries an attribute within a start tag. Text()
	// returns the attribute key; AttrVal() returns the value bytes
	// (possibly including surrounding quotes), or nil for boolean
	// attributes.
	AttributeToken

	// StartTagCloseToken marks the > that closes a start tag.
	StartTagCloseToken

	// StartTagVoidToken marks a self-closing /> that closes a start tag.
	StartTagVoidToken

	// tokenCount is an unexported sentinel used to size array dispatch
	// tables indexed by TokenType.
	tokenCount
)

// tokenTypeNames maps each TokenType to its display name. Used by String()
// via direct array indexing.
var tokenTypeNames = [tokenCount]string{
	ErrorToken:         "Error",
	TextToken:          "Text",
	StartTagToken:      "StartTag",
	EndTagToken:        "EndTag",
	CommentToken:       "Comment",
	SVGToken:           "SVG",
	MathToken:          "Math",
	AttributeToken:     "Attribute",
	StartTagCloseToken: "StartTagClose",
	StartTagVoidToken:  "StartTagVoid",
}

// String returns a human-readable name for the token type.
//
// Returns string which holds the display name for this token type.
func (t TokenType) String() string {
	if int(t) < len(tokenTypeNames) {
		return tokenTypeNames[t]
	}

	return "Unknown"
}
