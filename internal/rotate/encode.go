package rotate

import (
	"fmt"
	"math/big"
	"strings"
)

func norm(r rune) rune {
	// Return lowercase as is
	if 'a' <= r && r <= 'z' {
		return r
	}
	// Return uppercase as lowercase
	if 'A' <= r && r <= 'Z' {
		return r - 'A' + 'a'
	}
	// Drop everything else
	return -1
}

func Encode(s, t string, n *big.Int) (*big.Int, error) {
	n = big.NewInt(0).Set(n) // Clone n

	sr := []rune(strings.Map(norm, s))
	tr := []rune(strings.Map(norm, t))
	if len(sr) != len(tr) {
		return nil, fmt.Errorf("string and target must have the same number of letters")
	}

	m := big.NewInt(26)

	for i := len(sr) - 1; i >= 0; i-- {
		rot := sr[i] - tr[i]
		for rot < 0 {
			rot += 26
		}

		n.Mul(n, m)
		n.Add(n, big.NewInt(int64(rot)))
	}
	return n, nil
}
