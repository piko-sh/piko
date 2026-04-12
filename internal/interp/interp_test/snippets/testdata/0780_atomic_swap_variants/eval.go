package main

import (
	"fmt"
	"sync/atomic"
)

func run() string {
	var i32 int32 = 100
	prev32 := atomic.SwapInt32(&i32, 200)

	var i64 int64 = 1000
	prev64 := atomic.SwapInt64(&i64, 2000)

	var u32 uint32 = 50
	prevU32 := atomic.SwapUint32(&u32, 75)

	var u64 uint64 = 9999
	prevU64 := atomic.SwapUint64(&u64, 8888)

	return fmt.Sprintf("i32=%d->%d;i64=%d->%d;u32=%d->%d;u64=%d->%d",
		prev32, i32, prev64, i64, prevU32, u32, prevU64, u64)
}
