package main

func run() string {
	result := ""

	var u8 uint8 = 255
	u8++
	result += intToStr(int64(u8)) + ","

	var i8 int8 = 127
	i8++
	result += intToStr(int64(i8)) + ","

	var u16 uint16 = 65535
	u16++
	result += intToStr(int64(u16)) + ","

	var i16 int16 = 32767
	i16++
	result += intToStr(int64(i16)) + ","

	var u32 uint32 = 0xFFFFFFFF
	u32++
	result += intToStr(int64(u32)) + ","

	var i32 int32 = 0x7FFFFFFF
	i32++
	result += intToStr(int64(i32)) + ","

	var u8mul uint8 = 200
	u8mul *= 3
	result += intToStr(int64(u8mul)) + ","

	var i8sub int8 = -128
	i8sub -= 1
	result += intToStr(int64(i8sub)) + ","

	var u16shift uint16 = 0xFFFF
	u16shift <<= 1
	result += intToStr(int64(u16shift)) + ","

	var u8and uint8 = 0xFF
	u8and &= 0x0F
	result += intToStr(int64(u8and))

	return result
}

func intToStr(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
