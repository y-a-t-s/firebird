package firebird

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
)

func TestSubmit(t *testing.T) {
	const HOST = "kiwifarms.net"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := http.Client{}

	log.Println("Fetching new challenge...")
	c, err := NewChallenge(hc, HOST)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Challenge: %s, Difficulty: %d, Patience: %d\n", c.Salt, c.Diff, c.Patience)

	s, err := Solve(ctx, c)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Solution hash: %x, nonce: %d\n", s.Hash, s.Nonce)

	a, err := Submit(hc, s)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Response: %s\n", a)
}

func TestCheckZeros(t *testing.T) {
	failErr := func(diff uint32, hash []byte) string {
		return fmt.Sprintf("Zero check failed. Diff: %d, Hash: %+v", diff, hash)
	}
	var (
		d uint32
		h []byte
	)

	d, h = 17, []byte{0, 0, 64, 128, 42}
	if !checkZeros(d, h) {
		t.Error(failErr(d, h))
	}

	// This should fail (i.e. be false).
	d, h = 3, []byte{33, 130, 222, 88}
	if checkZeros(d, h) {
		t.Error(failErr(d, h))
	}
}
