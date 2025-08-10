package rotate

import (
	"math/big"
	"strings"
)

func Decode(s string, n *big.Int) (string, *big.Int) {
	n = big.NewInt(0).Set(n) // Clone n

	new := &strings.Builder{}
	new.Grow(len(s))

	rot := big.NewInt(0)
	m := big.NewInt(26)

	for _, r := range s {
		rot.Mod(n, m)

		if 'a' <= r && r <= 'z' {
			r = 'a' + (r-'a'+rune(rot.Int64()))%26
			n.Div(n, m)
		} else if 'A' <= r && r <= 'Z' {
			r = 'A' + (r-'A'+rune(rot.Int64()))%26
			n.Div(n, m)
		}

		new.WriteRune(r)
	}
	return new.String(), n
}
