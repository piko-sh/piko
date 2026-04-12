package main

import (
	"bytes"
	"fmt"
)

type Logger struct {
	*bytes.Buffer
	Prefix string
}

func (l *Logger) Log(msg string) {
	l.Buffer.WriteString(l.Prefix)
	l.Buffer.WriteString(": ")
	l.Buffer.WriteString(msg)
	l.Buffer.WriteString("\n")
}

func run() string {
	logger := &Logger{Buffer: &bytes.Buffer{}, Prefix: "INFO"}
	logger.Log("startup")
	logger.Log("ready")
	out := logger.String()
	return fmt.Sprintf("len=%d;content=%q", logger.Len(), out)
}
