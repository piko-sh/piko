package main

func sizeLabel(s size) string {
	switch s {
	case small:
		return "small"
	case medium:
		return "medium"
	case large:
		return "large"
	}
	return "unknown"
}
