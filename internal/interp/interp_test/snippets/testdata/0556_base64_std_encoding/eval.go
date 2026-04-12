package main

import "encoding/base64"

func run() string {
	enc := base64.StdEncoding.EncodeToString([]byte("hello"))
	dec, _ := base64.StdEncoding.DecodeString(enc)
	return enc + "|" + string(dec)
}
