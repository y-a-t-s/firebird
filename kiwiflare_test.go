package firebird

import (
	"log"
	"net/http"
	"testing"
)

func TestSubmit(t *testing.T) {
	hc := http.Client{}

	log.Println("Fetching new challenge...")
	c, err := NewChallenge(hc)
	if err != nil {
		t.Error(err)
	}
	log.Printf("Challenge: %s, Difficulty: %d, Patience: %d\n", c.Salt, c.Diff, c.Patience)
	s, err := Solve(c)
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
