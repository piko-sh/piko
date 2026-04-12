package main

import (
	"database/sql/driver"
	"fmt"
)

type Status int

const (
	StatusUnknown Status = iota
	StatusActive
	StatusArchived
)

func (s Status) Value() (driver.Value, error) {
	switch s {
	case StatusActive:
		return "active", nil
	case StatusArchived:
		return "archived", nil
	default:
		return "unknown", nil
	}
}

func (s *Status) Scan(src any) error {
	switch v := src.(type) {
	case string:
		switch v {
		case "active":
			*s = StatusActive
		case "archived":
			*s = StatusArchived
		default:
			*s = StatusUnknown
		}
		return nil
	case nil:
		*s = StatusUnknown
		return nil
	}
	return fmt.Errorf("unsupported type")
}

func run() string {
	s := StatusActive
	var valuer driver.Valuer = s
	v, _ := valuer.Value()

	var decoded Status
	_ = decoded.Scan("archived")

	return fmt.Sprintf("value=%v;decoded=%d", v, decoded)
}
