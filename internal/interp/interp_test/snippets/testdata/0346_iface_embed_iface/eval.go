package main

type Reader interface {
	Read() string
}

type Writer interface {
	Write(s string)
}

type ReaderWriter interface {
	Reader
	Writer
}

type Buf struct {
	data string
}

func (b *Buf) Read() string   { return b.data }
func (b *Buf) Write(s string) { b.data = s }

func run() string {
	var rw ReaderWriter = &Buf{}
	rw.Write("hello")
	return rw.Read()
}
