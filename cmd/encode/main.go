package main

import (
	"fmt"
	"math/big"
	"os"
	"sphinx/internal/rotate"
)

func main() {
	if len(os.Args) <= 3 {
		fmt.Println("Usage: decode <string> <target> <seed>")
		return
	}

	s := os.Args[1]
	t := os.Args[2]

	n := &big.Int{}
	_, ok := n.SetString(os.Args[3], 10)
	if !ok || n.Cmp(&big.Int{}) < 0 {
		fmt.Println("Seed must be a positive whole number.")
		return
	}

	n, err := rotate.Encode(s, t, n)
	if err != nil {
		fmt.Println("Encode error:")
		fmt.Println(err)
		return
	}
	fmt.Println(n.String())
}
