// Package rot provides functions for rotating lowercase letters by a given number of positions.
package rot

func rot(r rune, n int) rune {
	if r < 'a' || r > 'z' {
		panic("rot: input must be a lowercase letter")
	}
	if n < 0 {
		n += 26
	}
	if n < 0 || n >= 26 {
		panic("rot: n must be in the range [-26, 25]")
	}
	return 'a' + (r-'a'+rune(n))%26
}
