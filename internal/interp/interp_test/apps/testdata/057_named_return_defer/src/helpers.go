package main

func adjustResult(r *int, delta int) {
	*r += delta
}

func wrapErrorMessage(msg string, source string) string {
	return source + ": " + msg
}
