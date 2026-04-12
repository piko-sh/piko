package main

import (
	"fmt"
	"runtime"
)

func run() string {
	return fmt.Sprintf("ncpu_positive=%t;goos_nonempty=%t;goarch_nonempty=%t;version_starts_go=%t",
		runtime.NumCPU() > 0,
		runtime.GOOS != "",
		runtime.GOARCH != "",
		len(runtime.Version()) >= 2 && runtime.Version()[:2] == "go")
}
