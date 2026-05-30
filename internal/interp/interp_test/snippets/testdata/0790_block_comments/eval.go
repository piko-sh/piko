package main

/* file-level block comment */

import "fmt"

/*
multi-line block comment
spanning several lines
describing the run function
*/
func run() string {
	x := 1 /* inline */ + 2
	y := /* leading */ 10 - /* middle */ 3 /* trailing */
	z := 5 * 2                             /* trailing after expr */
	w := /* before-call */ fmt.Sprintf("z=%d", z)
	return fmt.Sprintf("x=%d,y=%d,z=%d,w=%s", x, y, z, w)
}
