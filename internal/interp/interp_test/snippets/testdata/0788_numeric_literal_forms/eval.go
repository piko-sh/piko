package main

import "fmt"

func run() string {
	million := 1_000_000
	binary := 0b1010_1010
	octalModern := 0o755
	octalLegacy := 0755
	hexUpper := 0xABCD_EF01
	hexFloat := 0x1p10
	hexFloatNeg := 0x1.8p4
	floatExp := 1.5e2
	floatUnderscore := 1_234.567_8

	return fmt.Sprintf(
		"million=%d;bin=%d;oct_o=%d;oct_legacy=%d;hex=%d;hexf=%g;hexfneg=%g;sci=%g;ufloat=%g",
		million, binary, octalModern, octalLegacy, hexUpper, hexFloat, hexFloatNeg, floatExp, floatUnderscore)
}
