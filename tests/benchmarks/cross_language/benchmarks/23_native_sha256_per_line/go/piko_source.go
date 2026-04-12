package main

import (
	"crypto/sha256"
	"os"
	"time"
)

const digestBytes = 32

func Run() string {
	corpus := loadCorpus()
	return doLineHashing(corpus)
}

func RunInner(k int) (string, int64) {
	corpus := loadCorpus()
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doLineHashing(corpus)
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func loadCorpus() []byte {
	candidates := []string{
		"testdata/corpus.txt",
		"benchmarks/23_native_sha256_per_line/testdata/corpus.txt",
		"tests/benchmarks/cross_language/benchmarks/23_native_sha256_per_line/testdata/corpus.txt",
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data
		}
	}
	panic("corpus not found at any candidate path")
}

func doLineHashing(corpus []byte) string {
	accumulator := [digestBytes]byte{}
	lineStart := 0
	for index := 0; index < len(corpus); index++ {
		if corpus[index] != '\n' {
			continue
		}
		digest := sha256.Sum256(corpus[lineStart:index])
		for digestIndex := 0; digestIndex < digestBytes; digestIndex++ {
			accumulator[digestIndex] ^= digest[digestIndex]
		}
		lineStart = index + 1
	}
	if lineStart < len(corpus) {
		digest := sha256.Sum256(corpus[lineStart:])
		for digestIndex := 0; digestIndex < digestBytes; digestIndex++ {
			accumulator[digestIndex] ^= digest[digestIndex]
		}
	}
	return bytesToHex(accumulator[:])
}

func bytesToHex(input []byte) string {
	hexChars := [16]byte{
		'0', '1', '2', '3', '4', '5', '6', '7',
		'8', '9', 'a', 'b', 'c', 'd', 'e', 'f',
	}
	out := make([]byte, len(input)*2)
	for index := 0; index < len(input); index++ {
		out[index*2] = hexChars[(input[index]>>4)&0x0F]
		out[index*2+1] = hexChars[input[index]&0x0F]
	}
	return string(out)
}
