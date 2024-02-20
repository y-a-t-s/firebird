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

	hash, nonce, salt := res.Result()
	log.Printf("Challenge: %s, Hash: %x, Nonce: %d", salt, hash, nonce)
}
