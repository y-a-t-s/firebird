package firebird

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/proxy"
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

func submit(ctx context.Context, hc http.Client, host string) error {
	connType := "clearnet"
	if strings.HasSuffix(host, ".onion") {
		connType = "tor"
	}

	log.Printf("Fetching new %s challenge...", connType)
	c, err := NewChallenge(hc, host)
	if err != nil {
		return err
	}
	log.Printf("Challenge: %s, Difficulty: %d, Patience: %d\n", c.Salt, c.Diff, c.Patience)

	s, err := Solve(ctx, c)
	if err != nil {
		return err
	}
	log.Printf("Solution hash: %x, nonce: %d\n", s.Hash, s.Nonce)

	a, err := Submit(hc, s)
	if err != nil {
		return err
	}
	log.Printf("Response: %s\n\n", a)

	return nil
}

func newProxyTransport() *http.Transport {
	pcd := proxy.FromEnvironment().(proxy.ContextDialer)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = pcd.DialContext

	return tr
}

func TestSubmit(t *testing.T) {
	ctx := t.Context()

	hc := http.Client{}

	err := submit(ctx, hc, _TEST_HOST)
	if err != nil {
		t.Error(err)
	}

	hc.Transport = newProxyTransport()

	var dnsErr *net.DNSError
	err = submit(ctx, hc, _TEST_ONION)
	if err != nil {
		if errors.As(err, &dnsErr) {
			log.Println("Unable to resolve .onion domain. Make sure ALL_PROXY is set and tor is running.")
			log.Println("Skipping...")
		} else {
			t.Error(err)
		}
	}
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
