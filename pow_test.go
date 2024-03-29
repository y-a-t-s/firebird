package main

import (
	"log"
	"testing"
)

func TestSolve(t *testing.T) {
	res, err := Solve(nil, "https://kiwifarms.st")
	if err != nil {
		t.Error(err)
	}

	challenge, nonce, hash := res.Result()
	log.Printf("Challenge: %s, Hash: %x, Nonce: %d", challenge, hash, nonce)
}
