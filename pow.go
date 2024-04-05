package firebird

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type pow interface {
	getParams(root *html.Node) error
	solve() (Solution, error)
}

// PoW challenge solution.
type Solution interface {
	QueryStr() string
	Solution() ([]byte, uint32, []byte)
}

func Solve(proxy proxyDialer, host string) (Solution, error) {
	hc, err := newHttpClient(proxy, host)
	if err != nil {
		return nil, err
	}

	// Supplied host may begin with protocol shit.
	// Isolate the hostname and reassemble to parse to URL.
	tmp := regexp.MustCompile(`(https?://)?([\w.]+)/?`).FindStringSubmatch(host)
	if len(tmp) < 3 {
		return nil, errors.New("Failed to parse host string.")
	}
	host = tmp[2]

	u, err := url.Parse(fmt.Sprintf("https://%s", host))
	if err != nil {
		return nil, err
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
			return nil, err
		}
	}

	res, err := p.solve()
	if err != nil {
		return nil, err
	}

	return res, nil
}
