package main

import (
	"fmt"
	"io"
)

type SeekableBuffer struct {
	data []byte
	pos  int64
}

func (s *SeekableBuffer) Read(p []byte) (int, error) {
	if s.pos >= int64(len(s.data)) {
		return 0, io.EOF
	}
	n := copy(p, s.data[s.pos:])
	s.pos += int64(n)
	return n, nil
}

func (s *SeekableBuffer) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = s.pos + offset
	case io.SeekEnd:
		newPos = int64(len(s.data)) + offset
	}
	if newPos < 0 {
		return 0, fmt.Errorf("negative")
	}
	s.pos = newPos
	return newPos, nil
}

func run() string {
	sb := &SeekableBuffer{data: []byte("hello world")}
	var seeker io.Seeker = sb

	pos1, _ := seeker.Seek(6, io.SeekStart)
	buf := make([]byte, 5)
	n, _ := sb.Read(buf)
	pos2, _ := seeker.Seek(0, io.SeekStart)

	return fmt.Sprintf("pos1=%d,read=%s(%d),pos2=%d", pos1, string(buf[:n]), n, pos2)
}
