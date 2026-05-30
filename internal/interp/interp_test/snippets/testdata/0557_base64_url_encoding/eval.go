package main

import "encoding/base64"

func run() string {
	enc := base64.URLEncoding.EncodeToString([]byte("a?b/c"))
	return enc
}
