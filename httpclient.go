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

type httpClient struct {
	http.Client
	domain url.URL
}

// For Tor or other proxy stuff.
type proxyDialer func(ctx context.Context, network string, addr string) (net.Conn, error)

func newHttpClient(proxy proxyDialer, host string) (httpClient, error) {
	u, err := url.Parse(host)
	if err != nil {
		panic(err)
	}

	return httpClient{
		http.Client{
			Transport: &http.Transport{
				DialContext:        proxy, // May be nil. If so, uses default.
				DisableCompression: false,
				IdleConnTimeout:    time.Minute * 3,
				MaxIdleConns:       4,
			},
		},
		*u,
	}, nil
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

func (p *httpClient) getChallengePage() (*html.Node, error) {
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
