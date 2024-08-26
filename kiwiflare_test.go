package firebird

import (
	"log"
	"net/http"
	"testing"
)

func TestSubmit(t *testing.T) {
	const HOST = "kiwifarms.st"

	hc := http.Client{}

	log.Println("Fetching new challenge...")
	c, err := NewChallenge(hc, HOST)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Challenge: %s, Difficulty: %d, Patience: %d\n", c.Salt, c.Diff, c.Patience)
	s, err := Solve(c)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Solution hash: %x, nonce: %d\n", s.Hash, s.Nonce)

	a, err := Submit(hc, HOST, s)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Response: %s\n", a)
}
