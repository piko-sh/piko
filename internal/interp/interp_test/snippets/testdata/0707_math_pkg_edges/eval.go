package main

import (
	"fmt"
	"math"
)

func run() string {
	result := ""

	result += fmt.Sprintf("sqrt-1:%v;", math.IsNaN(math.Sqrt(-1)))
	result += fmt.Sprintf("sqrt0:%v;", math.Sqrt(0))
	result += fmt.Sprintf("sqrt4:%v;", math.Sqrt(4))

	result += fmt.Sprintf("log0:%v;", math.IsInf(math.Log(0), -1))
	result += fmt.Sprintf("log-1:%v;", math.IsNaN(math.Log(-1)))

	result += fmt.Sprintf("expHuge:%v;", math.IsInf(math.Exp(1000), 1))

	result += fmt.Sprintf("pow0_0:%v;", math.Pow(0, 0))
	result += fmt.Sprintf("powNeg:%v;", math.IsNaN(math.Pow(-1, 0.5)))

	result += fmt.Sprintf("modPos:%v;", math.Mod(10.5, 3))
	result += fmt.Sprintf("modNeg:%v;", math.Mod(-10.5, 3))

	result += fmt.Sprintf("absMin:%v;", math.Abs(math.MinInt64*1.0+1))

	result += fmt.Sprintf("ceil:%v;floor:%v;round:%v;", math.Ceil(1.2), math.Floor(1.7), math.Round(2.5))

	result += fmt.Sprintf("maxFloat64:%v;smallestSubnormal:%v;",
		math.Inf(1) > math.MaxFloat64,
		math.SmallestNonzeroFloat64 > 0)

	result += fmt.Sprintf("atan2_0_0:%v;atan2_1_0:%v",
		math.Atan2(0, 0), math.Atan2(1, 0))

	return result
}
