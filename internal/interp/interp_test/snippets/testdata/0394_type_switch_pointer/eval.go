package main

type A struct{ X int }
type B struct{ Y string }

func tag(v any) string {
	switch t := v.(type) {
	case *A:
		return "A:" + string(rune('0'+t.X))
	case *B:
		return "B:" + t.Y
	default:
		return "other"
	}
}

func run() string {
	return tag(&A{X: 5}) + "," + tag(&B{Y: "k"}) + "," + tag(7)
}
