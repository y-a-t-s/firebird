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

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func splitProtocol(addr string) (proto string, host string, err error) {
	// FindStringSubmatch is used to capture the groups.
	// Index 0 is the full matching string with all groups.
	// The rest are numbered by the order of the opening parens.
	// Here, we want the last 2 groups (indexes 1 and 2, requiring length 3).
	tmp := regexp.MustCompile(`^([\w-]+://)?([^/]+)`).FindStringSubmatch(addr)
	// At the very least, we need the hostname part (index 2).
	if len(tmp) < 3 || tmp[2] == "" {
		err = errors.New("Failed to parse address: " + addr)
		return
	}

	proto = tmp[1]
	host = tmp[2]
	return
}

func parseTags(r io.Reader) (c Challenge, err error) {
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
						return c, err
					}
					c.Diff = diff
				case "data-sssg-patience":
					pat, err := parseAttr(a.Val)
					if err != nil {
						return c, err
					}
					c.Patience = pat
				}
			}
		}
	}

	if c.Salt == "" {
		err = errors.New("Failed to parse challenge from data tags.")
		return
	}

	return
}

func NewChallenge(hc http.Client, host string) (Challenge, error) {
	_, host, err := splitProtocol(host)
	if err != nil {
		return Challenge{}, err
	}

	resp, err := hc.Get("https://" + host)
	if err != nil {
		return Challenge{}, err
	}
	defer resp.Body.Close()

	// Check for 203 status
	if resp.StatusCode != 203 {
		return Challenge{}, errors.New("No redirect to challenge page.")
	}

	c, err := parseTags(resp.Body)
	if err != nil {
		return Challenge{}, err
	}

	return c, nil
}

func Submit(hc http.Client, host string, s Solution) (string, error) {
	_, host, err := splitProtocol(host)
	if err != nil {
		return "", err
	}

	resp, err := hc.PostForm("https://"+host+"/.sssg/api/answer", url.Values{
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
