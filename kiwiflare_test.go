package firebird

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
)

const _TEST_HOST = "kiwifarms.net"
const _TEST_ONION = "kiwifarmsaaf4t2h7gc3dfc5ojhmqruw2nit3uejrpiagrxeuxiyxcyd.onion"

type errBadZeroCheck struct {
	Diff uint32
	Hash []byte
}

func (e *errBadZeroCheck) Error() string {
	return fmt.Sprintf("Zero check failed. Diff: %d, Hash: %+v\n", e.Diff, e.Hash)
}

func TestSubmit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hc := http.Client{}

	log.Println("Fetching new challenge...")
	c, err := NewChallenge(hc, _TEST_HOST)
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
	d, h := uint32(17), []byte{0, 0, 64, 128, 42}
	if !checkZeros(d, h) {
		t.Error(errBadZeroCheck{d, h})
	}

	// This should fail (i.e. be false).
	d, h = uint32(3), []byte{33, 130, 222, 88}
	if checkZeros(d, h) {
		t.Error(errBadZeroCheck{d, h})
	}
}
