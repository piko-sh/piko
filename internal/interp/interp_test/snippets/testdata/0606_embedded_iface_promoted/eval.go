package main

type Reader interface {
	Read() string
}

type Writer interface {
	Write(s string)
}

type ReadWriter interface {
	Reader
	Writer
}

type buf struct {
	data string
}

func (b *buf) Read() string   { return b.data }
func (b *buf) Write(s string) { b.data = s }

func run() string {
	var rw ReadWriter = &buf{}
	rw.Write("hello")
	return rw.Read()
}
