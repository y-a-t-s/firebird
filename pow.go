package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type proxyDialer func(ctx context.Context, network string, addr string) (net.Conn, error)

type pow interface {
	getParams(root *html.Node) error
	Solve() (result, error)
}

type httpClient struct {
	http.Client
	domain url.URL
}

type result interface {
	Result() ([]byte, uint32, []byte)
}

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

func Solve(proxy proxyDialer, host string) (result, error) {
	hc, err := newHttpClient(proxy, host)
	if err != nil {
		panic(err)
	}

	// Supplied host may begin with protocol shit.
	// Isolate the hostname and reassemble to parse to URL.
	re := regexp.MustCompile(`(https?://)?([\w.]+)/?`)
	hostname := re.FindStringSubmatch(host)[2]
	u, err := url.Parse(fmt.Sprintf("https://%s", hostname))
	if err != nil {
		panic(err)
	}

	tmp := strings.Split(u.Host, ".")
	tld := tmp[len(tmp)-1]

	var p pow
	switch tld {
	case "onion":
		// TODO: haproxyShit()
	default:
		p, err = initKF(hc)
		if err != nil {
			panic(err)
		}
	}

	res, err := p.Solve()
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
