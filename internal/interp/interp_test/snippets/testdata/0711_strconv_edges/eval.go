package main

import (
	"fmt"
	"strconv"
)

func run() string {
	result := ""

	parseCases := []string{"42", "+42", "-42", "0", "-0", "  10", "", "abc", "9223372036854775807", "9223372036854775808"}
	for _, raw := range parseCases {
		v, err := strconv.Atoi(raw)
		if err != nil {
			result += fmt.Sprintf("%q=ERR;", raw)
		} else {
			result += fmt.Sprintf("%q=%d;", raw, v)
		}
	}

	binNum, _ := strconv.ParseInt("1011", 2, 64)
	result += fmt.Sprintf("bin1011=%d;", binNum)

	hexNum, _ := strconv.ParseInt("ff", 16, 32)
	result += fmt.Sprintf("hexff=%d;", hexNum)

	octNum, _ := strconv.ParseInt("777", 8, 32)
	result += fmt.Sprintf("oct777=%d;", octNum)

	parseUintCases := []string{"42", "-1", "+0"}
	for _, raw := range parseUintCases {
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			result += fmt.Sprintf("u%q=ERR;", raw)
		} else {
			result += fmt.Sprintf("u%q=%d;", raw, v)
		}
	}

	floatCases := []string{"3.14", "0.1", "1e10", "1e1000", "nan", "inf", "-inf", "abc"}
	for _, raw := range floatCases {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			result += fmt.Sprintf("f%q=ERR;", raw)
		} else {
			result += fmt.Sprintf("f%q=%v;", raw, v)
		}
	}

	result += fmt.Sprintf("itoa=%s;", strconv.Itoa(-42))
	result += fmt.Sprintf("formatInt=%s;", strconv.FormatInt(255, 16))
	result += "quote=" + strconv.Quote("hello\nworld")

	return result
}
