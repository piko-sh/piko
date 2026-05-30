package main

import (
	"fmt"
	"strings"
)

type Tag struct {
	Name string
}

func (t *Tag) Scan(state fmt.ScanState, verb rune) error {
	tok, err := state.Token(true, func(r rune) bool { return r != ',' && r != ' ' })
	if err != nil {
		return err
	}
	t.Name = strings.ToUpper(string(tok))
	return nil
}

func run() string {
	reader := strings.NewReader("alpha beta gamma")
	var t1, t2, t3 Tag
	n, _ := fmt.Fscan(reader, &t1, &t2, &t3)
	return fmt.Sprintf("n=%d,t1=%s,t2=%s,t3=%s", n, t1.Name, t2.Name, t3.Name)
}
