package main

import "fmt"

func captureAtDeferTime() string {
	result := ""
	for i := 0; i < 3; i++ {
		defer fmt.Fprintf(&runWriter, "early:%d\n", i)
		_ = result
	}
	return runOutput()
}

func captureAtCallTime() string {
	i := 0
	for ; i < 3; i++ {
		defer func() {
			fmt.Fprintf(&runWriter, "late:%d\n", i)
		}()
	}
	return runOutput()
}

func mixedCapture() string {
	x := 10
	defer func(snapshot int) {
		fmt.Fprintf(&runWriter, "mixed_snap=%d_x=%d\n", snapshot, x)
	}(x)
	x = 99
	return runOutput()
}

var runWriter stringWriter

type stringWriter struct {
	data []byte
}

func (w *stringWriter) Write(p []byte) (int, error) {
	w.data = append(w.data, p...)
	return len(p), nil
}

func runOutput() string {
	return ""
}

func reset() string {
	out := string(runWriter.data)
	runWriter.data = nil
	return out
}

func run() string {
	_ = captureAtDeferTime()
	earlyDeferred := reset()

	_ = captureAtCallTime()
	lateDeferred := reset()

	_ = mixedCapture()
	mixedDeferred := reset()

	return "early=" + earlyDeferred + "late=" + lateDeferred + "mixed=" + mixedDeferred
}
