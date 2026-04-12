package main

type Event interface{ kind() string }

type NumberEvent struct{ N int }

func (NumberEvent) kind() string { return "number" }

type TextEvent struct{ S string }

func (TextEvent) kind() string { return "text" }
