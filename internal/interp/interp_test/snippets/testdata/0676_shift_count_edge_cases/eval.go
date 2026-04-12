package main

import "fmt"

func run() string {
	result := ""

	var u8 uint8 = 1
	result += fmt.Sprintf("u8<<8=%d;", u8<<8)
	result += fmt.Sprintf("u8<<15=%d;", u8<<15)

	var u32 uint32 = 0xFFFFFFFF
	result += fmt.Sprintf("u32<<32=%d;", u32<<32)

	var i8 int8 = -8
	result += fmt.Sprintf("i8>>1=%d;", i8>>1)
	result += fmt.Sprintf("i8>>8=%d;", i8>>8)

	var i32 int32 = -1
	result += fmt.Sprintf("i32>>10=%d;", i32>>10)
	result += fmt.Sprintf("i32>>32=%d;", i32>>32)

	var u32hi uint32 = 0x80000000
	result += fmt.Sprintf("u32hi>>31=%d;", u32hi>>31)
	result += fmt.Sprintf("u32hi>>32=%d;", u32hi>>32)

	var shiftCount uint = 65
	var i64 int64 = 1
	result += fmt.Sprintf("i64<<%d=%d;", shiftCount, i64<<shiftCount)

	result += fmt.Sprintf("u8>>9=%d", u8>>9)

	return result
}
