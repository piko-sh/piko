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

//go:build bench

package ast_test_bench

import "testing"

type TokenType uint8

const (
	TokenIdent TokenType = iota
	TokenNumber
	TokenString
	TokenPlus
	TokenMinus
	TokenMul
	TokenDiv
	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket
	TokenComma
	TokenDot
	TokenColon
	TokenSemicolon
	TokenEqual
	TokenNotEqual
	TokenLess
	tokenCount
)

var (
	sink          int
	dispatchArray = [tokenCount]func() int{
		TokenIdent:     handleIdent,
		TokenNumber:    handleNumber,
		TokenString:    handleString,
		TokenPlus:      handlePlus,
		TokenMinus:     handleMinus,
		TokenMul:       handleMul,
		TokenDiv:       handleDiv,
		TokenLParen:    handleLParen,
		TokenRParen:    handleRParen,
		TokenLBrace:    handleLBrace,
		TokenRBrace:    handleRBrace,
		TokenLBracket:  handleLBracket,
		TokenRBracket:  handleRBracket,
		TokenComma:     handleComma,
		TokenDot:       handleDot,
		TokenColon:     handleColon,
		TokenSemicolon: handleSemicolon,
		TokenEqual:     handleEqual,
		TokenNotEqual:  handleNotEqual,
		TokenLess:      handleLess,
	}
	dispatchMap = map[TokenType]func() int{
		TokenIdent:     handleIdent,
		TokenNumber:    handleNumber,
		TokenString:    handleString,
		TokenPlus:      handlePlus,
		TokenMinus:     handleMinus,
		TokenMul:       handleMul,
		TokenDiv:       handleDiv,
		TokenLParen:    handleLParen,
		TokenRParen:    handleRParen,
		TokenLBrace:    handleLBrace,
		TokenRBrace:    handleRBrace,
		TokenLBracket:  handleLBracket,
		TokenRBracket:  handleRBracket,
		TokenComma:     handleComma,
		TokenDot:       handleDot,
		TokenColon:     handleColon,
		TokenSemicolon: handleSemicolon,
		TokenEqual:     handleEqual,
		TokenNotEqual:  handleNotEqual,
		TokenLess:      handleLess,
	}
	interfaceArray = [tokenCount]Handler{
		TokenIdent:     identHandler{},
		TokenNumber:    numberHandler{},
		TokenString:    stringHandler{},
		TokenPlus:      plusHandler{},
		TokenMinus:     minusHandler{},
		TokenMul:       mulHandler{},
		TokenDiv:       divHandler{},
		TokenLParen:    lparenHandler{},
		TokenRParen:    rparenHandler{},
		TokenLBrace:    lbraceHandler{},
		TokenRBrace:    rbraceHandler{},
		TokenLBracket:  lbracketHandler{},
		TokenRBracket:  rbracketHandler{},
		TokenComma:     commaHandler{},
		TokenDot:       dotHandler{},
		TokenColon:     colonHandler{},
		TokenSemicolon: semicolonHandler{},
		TokenEqual:     equalHandler{},
		TokenNotEqual:  notequalHandler{},
		TokenLess:      lessHandler{},
	}
	testTokens = []TokenType{
		TokenIdent, TokenPlus, TokenNumber, TokenLParen, TokenString,
		TokenRParen, TokenMinus, TokenLBrace, TokenMul, TokenRBrace,
		TokenDiv, TokenComma, TokenDot, TokenColon, TokenEqual,
		TokenLess, TokenSemicolon, TokenLBracket, TokenRBracket, TokenNotEqual,
	}
)

func handleIdent() int     { return 1 }
func handleNumber() int    { return 2 }
func handleString() int    { return 3 }
func handlePlus() int      { return 4 }
func handleMinus() int     { return 5 }
func handleMul() int       { return 6 }
func handleDiv() int       { return 7 }
func handleLParen() int    { return 8 }
func handleRParen() int    { return 9 }
func handleLBrace() int    { return 10 }
func handleRBrace() int    { return 11 }
func handleLBracket() int  { return 12 }
func handleRBracket() int  { return 13 }
func handleComma() int     { return 14 }
func handleDot() int       { return 15 }
func handleColon() int     { return 16 }
func handleSemicolon() int { return 17 }
func handleEqual() int     { return 18 }
func handleNotEqual() int  { return 19 }
func handleLess() int      { return 20 }

func dispatchSwitch(t TokenType) int {
	switch t {
	case TokenIdent:
		return handleIdent()
	case TokenNumber:
		return handleNumber()
	case TokenString:
		return handleString()
	case TokenPlus:
		return handlePlus()
	case TokenMinus:
		return handleMinus()
	case TokenMul:
		return handleMul()
	case TokenDiv:
		return handleDiv()
	case TokenLParen:
		return handleLParen()
	case TokenRParen:
		return handleRParen()
	case TokenLBrace:
		return handleLBrace()
	case TokenRBrace:
		return handleRBrace()
	case TokenLBracket:
		return handleLBracket()
	case TokenRBracket:
		return handleRBracket()
	case TokenComma:
		return handleComma()
	case TokenDot:
		return handleDot()
	case TokenColon:
		return handleColon()
	case TokenSemicolon:
		return handleSemicolon()
	case TokenEqual:
		return handleEqual()
	case TokenNotEqual:
		return handleNotEqual()
	case TokenLess:
		return handleLess()
	default:
		return 0
	}
}

func dispatchArrayLookup(t TokenType) int {
	return dispatchArray[t]()
}

func dispatchMapLookup(t TokenType) int {
	if dispatchFunction, ok := dispatchMap[t]; ok {
		return dispatchFunction()
	}
	return 0
}

type Handler interface {
	Handle() int
}

type identHandler struct{}
type numberHandler struct{}
type stringHandler struct{}
type plusHandler struct{}
type minusHandler struct{}
type mulHandler struct{}
type divHandler struct{}
type lparenHandler struct{}
type rparenHandler struct{}
type lbraceHandler struct{}
type rbraceHandler struct{}
type lbracketHandler struct{}
type rbracketHandler struct{}
type commaHandler struct{}
type dotHandler struct{}
type colonHandler struct{}
type semicolonHandler struct{}
type equalHandler struct{}
type notequalHandler struct{}
type lessHandler struct{}

func (identHandler) Handle() int     { return 1 }
func (numberHandler) Handle() int    { return 2 }
func (stringHandler) Handle() int    { return 3 }
func (plusHandler) Handle() int      { return 4 }
func (minusHandler) Handle() int     { return 5 }
func (mulHandler) Handle() int       { return 6 }
func (divHandler) Handle() int       { return 7 }
func (lparenHandler) Handle() int    { return 8 }
func (rparenHandler) Handle() int    { return 9 }
func (lbraceHandler) Handle() int    { return 10 }
func (rbraceHandler) Handle() int    { return 11 }
func (lbracketHandler) Handle() int  { return 12 }
func (rbracketHandler) Handle() int  { return 13 }
func (commaHandler) Handle() int     { return 14 }
func (dotHandler) Handle() int       { return 15 }
func (colonHandler) Handle() int     { return 16 }
func (semicolonHandler) Handle() int { return 17 }
func (equalHandler) Handle() int     { return 18 }
func (notequalHandler) Handle() int  { return 19 }
func (lessHandler) Handle() int      { return 20 }

func dispatchInterface(t TokenType) int {
	return interfaceArray[t].Handle()
}

func BenchmarkDispatch_Switch(b *testing.B) {
	var r int
	for b.Loop() {
		for _, t := range testTokens {
			r = dispatchSwitch(t)
		}
	}
	sink = r
}

func BenchmarkDispatch_Array(b *testing.B) {
	var r int
	for b.Loop() {
		for _, t := range testTokens {
			r = dispatchArrayLookup(t)
		}
	}
	sink = r
}

func BenchmarkDispatch_Map(b *testing.B) {
	var r int
	for b.Loop() {
		for _, t := range testTokens {
			r = dispatchMapLookup(t)
		}
	}
	sink = r
}

func BenchmarkDispatch_Interface(b *testing.B) {
	var r int
	for b.Loop() {
		for _, t := range testTokens {
			r = dispatchInterface(t)
		}
	}
	sink = r
}
