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

// Various errors.
var (
	ErrNoRedirect  = errors.New("No redirect to challenge page.")
	ErrParseFailed = errors.New("Failed to parse challenge from HTML data tags.")
)

type auth struct {
	Auth   string
	Domain string
}

func parseHost(addr string) (*url.URL, error) {
	// Guess https as protocol if one wasn't provided and hope it parses.
	if !strings.Contains(addr, "://") {
		addr = "https://" + addr
	}

	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func parseAttr(v string) (uint32, error) {
	tmp, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}

	return uint32(tmp), nil
}

func parseTags(r io.Reader) (Challenge, error) {
	c := Challenge{}

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
						return Challenge{}, ErrParseFailed
					}
					c.Diff = diff
				case "data-sssg-patience":
					pat, err := parseAttr(a.Val)
					if err != nil {
						return Challenge{}, ErrParseFailed
					}
					c.Patience = pat
				}
			}
		}
	}

	if c.Salt == "" {
		return Challenge{}, ErrParseFailed
	}

	return c, nil
}

func NewChallenge(hc http.Client, host string) (Challenge, error) {
	u, err := parseHost(host)
	if err != nil {
		return Challenge{}, err
	}

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
		return Challenge{}, err
	}
	defer resp.Body.Close()

	// Check for 203 status
	if resp.StatusCode != 203 {
		return Challenge{}, ErrNoRedirect
	}

	// Kept separate from the return because of the defer.
	c, err := parseTags(resp.Body)
	if err != nil {
		return Challenge{}, err
	}
	c.host = u

	return c, nil
}

func postSolution(hc http.Client, s Solution) (*http.Response, error) {
	// Ensure the POST url parses properly before passing the string.
	u, err := url.Parse(fmt.Sprintf("%s://%s/.sssg/api/answer", s.host.Scheme, s.host.Hostname()))
	if err != nil {
		return nil, err
	}

	return hc.PostForm(u.String(), url.Values{
		"a": []string{s.Salt},
		"b": []string{fmt.Sprint(s.Nonce)},
	})
}

func parseAuthToken(r io.Reader) (auth, error) {
	var a auth

	jd := json.NewDecoder(r)
	err := jd.Decode(&a)
	if err != nil {
		return auth{}, err
	}

	return a, nil
}

func Submit(hc http.Client, s Solution) (string, error) {
	resp, err := postSolution(hc, s)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	a, err := parseAuthToken(resp.Body)
	if err != nil {
		return "", err
	}

	return a.Auth, nil
}
