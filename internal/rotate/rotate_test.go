package rotate

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"
)

func randStr(l int) string {
	s := &strings.Builder{}
	s.Grow(l)
	for range l {
		s.WriteRune('a' + rand.Int32N(26))
	}
	return s.String()
}

func TestRand(t *testing.T) {
	for range 100 {
		str := randStr(20)
		target := randStr(20)
		seed := big.NewInt(rand.Int64())

		t.Run(fmt.Sprintf("%s/%s/%s", str, target, seed), func(t *testing.T) {
			key, err := Encode(str, target, seed)
			if err != nil {
				t.Fatal(err)
			}

			dec, rem := Decode(target, key)
			if have, want := dec, str; have != want {
				t.Fatalf("Decoded %s != %s", have, want)
			}
			if have, want := rem, seed; have.Cmp(seed) != 0 {
				t.Fatalf("Remainder %s != %s", have, want)
			}
		})
	}
}

func TestPure(t *testing.T) {
	const c = 1234567890
	n := big.NewInt(c)

	_, err := Encode("foo", "bar", n)
	if err != nil {
		t.Fatal(err)
	}

	if n.Cmp(big.NewInt(c)) != 0 {
		t.Fatalf("Encode side effect: N changed from %d to %s", c, n)
	}

	Decode("foo", n)

	if n.Cmp(big.NewInt(c)) != 0 {
		t.Fatalf("Decode side effect: N changed from %d to %s", c, n)
	}
}
