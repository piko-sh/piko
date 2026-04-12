package main

import (
	"fmt"
	"math"
)

func run() string {
	result := ""

	nan := math.NaN()
	result += fmt.Sprintf("nan==nan:%v;", nan == nan)
	result += fmt.Sprintf("nan<1:%v;", nan < 1.0)
	result += fmt.Sprintf("nan>1:%v;", nan > 1.0)
	result += fmt.Sprintf("nan<=1:%v;", nan <= 1.0)
	result += fmt.Sprintf("nan!=nan:%v;", nan != nan)

	zero := 0.0
	one := 1.0
	negOne := -1.0

	posInf := one / zero
	negInf := negOne / zero
	result += fmt.Sprintf("posInf==posInf:%v;", posInf == posInf)
	result += fmt.Sprintf("negInf<posInf:%v;", negInf < posInf)
	result += fmt.Sprintf("posInf+1==posInf:%v;", posInf+1 == posInf)
	result += fmt.Sprintf("isInf+:%v;", math.IsInf(posInf, 1))
	result += fmt.Sprintf("isInf-:%v;", math.IsInf(negInf, -1))

	zeroDivZero := zero / zero
	result += fmt.Sprintf("0/0_isNaN:%v;", math.IsNaN(zeroDivZero))

	huge := 1e20
	smallInt := int32(huge)
	result += fmt.Sprintf("1e20_as_int32:%d;", smallInt)

	negHuge := -1e20
	negSmallInt := int32(negHuge)
	result += fmt.Sprintf("neg1e20_as_int32:%d;", negSmallInt)

	posZero := 0.0
	negZero := -0.0
	result += fmt.Sprintf("0==-0:%v;", posZero == negZero)
	result += fmt.Sprintf("signBit_posZero:%v;", math.Signbit(posZero))
	result += fmt.Sprintf("signBit_negZero:%v", math.Signbit(negZero))

	return result
}
