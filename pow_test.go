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

	hash, nonce := res.Result()
	log.Printf("Hash: %x, Nonce: %d", hash, nonce)
}
