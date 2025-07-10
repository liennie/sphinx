package main

import (
	"fmt"
	"math/big"
	"os"
	"sphinx/internal/rotate"
)

func main() {
	if len(os.Args) <= 2 {
		fmt.Println("Usage: decode <string> <key>")
		return
	}

	s := os.Args[1]

	n := &big.Int{}
	_, ok := n.SetString(os.Args[2], 10)
	if !ok || n.Cmp(&big.Int{}) < 0 {
		fmt.Println("Key must be a positive whole number.")
		return
	}

	s, n = rotate.Decode(s, n)
	fmt.Printf("%s %s\n", s, n.String())
}
