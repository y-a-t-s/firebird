package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/html"
)

type proxy func(ctx context.Context, network string, addr string) (net.Conn, error)

type pow struct {
	http.Client
	domain url.URL
}

type result interface {
	Result() ([]byte, uint32)
}

type Solver interface {
	Solve() (result, error)
}

func newPow(p proxy, host string) (pow, error) {
	u, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	return pow{
		http.Client{
			Transport: &http.Transport{
				DialContext:        p, // May be nil. If so, uses default.
				DisableCompression: false,
				IdleConnTimeout:    time.Minute * 3,
				MaxIdleConns:       4,
			},
		},
		*u,
	}, nil
}

func Solve(p proxy, host string) (result, error) {
	pw, err := newPow(p, host)
	if err != nil {
		panic(err)
	}

	page, err := pw.getChallengePage()
	if err != nil {
		panic(err)
	}
	root, err := getRootNode(page)
	if err != nil {
		panic(err)
	}

	// TODO: Tor haproxy shit.

	params, err := getKFParams(root)
	if err != nil {
		panic(err)
	}

	kf := kiwiFlare{
		params,
		make([]kfWorker, 1),
		make(chan kfResult, 32),
	}
	res, err := kf.Solve()
	if err != nil {
		panic(err)
	}

	return res, nil
}

func getRootNode(n *html.Node) (*html.Node, error) {
	if n == nil {
		panic("Failed to find <html> tag in document.")
	}

	if n.Type == html.ElementNode && n.Data == "html" {
		return n, nil
	}

	return getRootNode(n.NextSibling)
}

func (p *pow) getChallengePage() (*html.Node, error) {
	resp, err := p.Get(fmt.Sprintf("https://%s", p.domain.Host))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	ht, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	if ht.Type == html.DocumentNode {
		rn, err := getRootNode(ht.FirstChild)
		if err != nil {
			panic(err)
		}
		ht = rn
	}

	return ht, nil
}
