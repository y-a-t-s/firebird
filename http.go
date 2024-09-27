package firebird

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var ErrParseFailed = errors.New("Failed to parse challenge from HTML data tags.")

func parseHost(addr string) (*url.URL, error) {
	// Guess https as protocol if one wasn't provided and hope it parses.
	if !strings.Contains(addr, "://") {
		addr = "https://" + addr
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	u.Path = ""
	u.RawPath = ""

	return u, nil
}

func parseTags(r io.Reader, c *Challenge) error {
	parseAttr := func(v string) (uint32, error) {
		tmp, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}

		return uint32(tmp), nil
	}

	z := html.NewTokenizer(r)
	for i := z.Next(); i != html.ErrorToken; i = z.Next() {
		tk := z.Token()
		if tk.DataAtom == atom.Html {
			for _, a := range tk.Attr {
				switch a.Key {
				case "data-sssg-challenge":
					c.Salt = a.Val
				case "data-sssg-difficulty":
					diff, err := parseAttr(a.Val)
					if err != nil {
						return err
					}
					c.Diff = diff
				case "data-sssg-patience":
					pat, err := parseAttr(a.Val)
					if err != nil {
						return err
					}
					c.Patience = pat
				}
			}
		}
	}

	if c.Salt == "" {
		return ErrParseFailed
	}

	return nil
}

func NewChallenge(hc http.Client, host string) (c Challenge, err error) {
	u, err := parseHost(host)
	if err != nil {
		return
	}
	c.host = u

	hostRE := regexp.MustCompile(`^kiwifarms`)
	// Update host url in case we get redirected across domains.
	hc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		rh := req.URL.Host
		if rh != u.Host && hostRE.MatchString(rh) {
			u.Host = rh
		}

		return nil
	}

	resp, err := hc.Get(u.String())
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Check for 203 status
	if resp.StatusCode != 203 {
		err = errors.New("No redirect to challenge page.")
		return
	}

	// Kept separate from the return because of the defer.
	err = parseTags(resp.Body, &c)
	if err != nil {
		return
	}

	return
}

func Submit(hc http.Client, s Solution) (string, error) {
	pu := fmt.Sprintf("%s://%s/.sssg/api/answer", s.host.Scheme, s.host.Hostname())
	resp, err := hc.PostForm(pu, url.Values{
		"a": []string{s.Salt},
		"b": []string{fmt.Sprint(s.Nonce)},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	type auth struct {
		Auth   string
		Domain string
	}

	a := auth{}
	err = json.Unmarshal(b, &a)
	if err != nil {
		return "", err
	}

	return a.Auth, nil
}
