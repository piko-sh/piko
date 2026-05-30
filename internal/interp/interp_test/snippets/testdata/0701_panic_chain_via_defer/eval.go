package main

import "fmt"

func recoverOnly() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	panic("original")
}

func panicReplaceInDefer() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			_ = r
			panic("replaced")
		}
	}()
	panic("original")
}

func deferPanicNoOriginal() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("%v", r)
		}
	}()
	defer func() {
		panic("from-defer")
	}()
	return ""
}

func deferOrderRecover() (recovered string) {
	defer func() {
		if r := recover(); r != nil {
			recovered = fmt.Sprintf("outer:%v", r)
		}
	}()
	defer func() {
		_ = recover()
	}()
	defer func() {
		panic("inner")
	}()
	return ""
}

func run() string {
	return "recoverOnly:" + recoverOnly() +
		";replace:" + panicReplaceInDefer() +
		";deferNo:" + deferPanicNoOriginal() +
		";order:" + deferOrderRecover()
}
