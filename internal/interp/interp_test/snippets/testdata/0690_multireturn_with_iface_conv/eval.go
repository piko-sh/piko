package main

import "fmt"

type opaqueError struct {
	Code int
}

func (o *opaqueError) Error() string {
	return fmt.Sprintf("opaque/%d", o.Code)
}

func returnConcreteAsError() (int, error) {
	return 42, &opaqueError{Code: 7}
}

func returnConcreteAsAny() (int, any) {
	return 1, opaqueError{Code: 99}
}

func returnNilError() (int, error) {
	return 0, nil
}

func returnSwapped() (any, int) {
	return "swapped", 5
}

func run() string {
	result := ""

	v1, e1 := returnConcreteAsError()
	result += fmt.Sprintf("conc:%d/%v;", v1, e1)

	v2, a := returnConcreteAsAny()
	result += fmt.Sprintf("any:%d/%v;", v2, a)

	v3, e3 := returnNilError()
	result += fmt.Sprintf("nilErr:%d/%v;", v3, e3)

	if e3 == nil {
		result += "nilErr_isNil;"
	} else {
		result += "nilErr_notNil;"
	}

	swapped, n := returnSwapped()
	result += fmt.Sprintf("swap:%v/%d", swapped, n)

	return result
}
