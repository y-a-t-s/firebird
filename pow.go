package main

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type pow interface {
	getParams(root *html.Node) error
	solve() (result, error)
}

type result interface {
	Result() ([]byte, uint32, []byte)
}

func Solve(proxy proxyDialer, host string) (result, error) {
	hc, err := newHttpClient(proxy, host)
	if err != nil {
		panic(err)
	}

	// Supplied host may begin with protocol shit.
	// Isolate the hostname and reassemble to parse to URL.
	tmp := regexp.MustCompile(`(https?://)?([\w.]+)/?`).FindStringSubmatch(host)
	if len(tmp) < 3 {
		panic("Failed to parse host string.")
	}
	hn := tmp[2]

	u, err := url.Parse(fmt.Sprintf("https://%s", hn))
	if err != nil {
		panic(err)
	}
	// TODO: Fix .net redirects.
	hc.domain = *u

	tmp = strings.Split(u.Host, ".")
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

	res, err := p.solve()
	if err != nil {
		panic(err)
	}

	return res, nil
}
