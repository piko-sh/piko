package main

type Stringer interface {
	String() string
}

type Holder struct{}

func (h *Holder) String() string {
	if h == nil {
		return "nil-receiver"
	}
	return "real"
}

func returnsTypedNilHolder() *Holder {
	return nil
}

func returnsUntypedNilInterface() Stringer {
	return nil
}

func returnsTypedNilThroughInterface() Stringer {
	var h *Holder
	return h
}

func returnsTypedNilFromHelper() Stringer {
	return returnsTypedNilHolder()
}

func boolToStr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func run() string {
	result := ""

	a := returnsUntypedNilInterface()
	result += "a==nil:" + boolToStr(a == nil) + ";"

	b := returnsTypedNilThroughInterface()
	result += "b==nil:" + boolToStr(b == nil) + ";"

	c := returnsTypedNilFromHelper()
	result += "c==nil:" + boolToStr(c == nil) + ";"

	result += "c.String:" + c.String() + ";"

	var d Stringer
	result += "d==nil:" + boolToStr(d == nil)

	return result
}
