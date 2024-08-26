package firebird

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func parseTags(r io.Reader) (Challenge, error) {
	parseAttr := func(v string) (uint32, error) {
		tmp, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}

		return uint32(tmp), nil
	}

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
		return c, errors.New("Failed to parse challenge from data tags.")
	}

	return c, nil
}

func NewChallenge(hc http.Client, host string) (c Challenge, err error) {
	resp, err := hc.Get(host)
	if err != nil {
		return c, err
	}
	defer resp.Body.Close()

	// Check for 203 status
	if resp.StatusCode != 203 {
		return c, errors.New("No redirect to challenge page.")
	}

	c, err = parseTags(resp.Body)
	return c, err
}

func Submit(hc http.Client, host string, s Solution) (string, error) {
	resp, err := hc.PostForm(host+"/.sssg/api/answer", url.Values{
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
