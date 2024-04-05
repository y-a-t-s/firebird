package firebird

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"hash"
	"math/rand"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/net/html"
)

// KiwiFlare instance. A variant of Hashcash.
type kiwiFlare struct {
	*kfParams
	hashes  chan kfSolution
	workers []kfWorker
}

type kfParams struct {
	challenge string // Challenge salt from server.
	diff      uint32 // Difficulty level.
	patience  uint32 // Time limit for answer in minutes.
}

type kfSolution struct {
	challenge []byte
	hash      []byte
	nonce     uint32
}

type kfWorker struct {
	hasher hash.Hash // Sha256 hasher obj.
	nonce  uint32    // Brute-forced nonce appended to salt.
}

func (s kfSolution) QueryStr() string {
	return fmt.Sprintf("a=%s&b=%d", s.challenge, s.nonce)
}

func (s kfSolution) Solution() ([]byte, uint32, []byte) {
	return s.challenge, s.nonce, s.hash
}

func initKF(hc httpClient) (pow, error) {
	page, err := hc.getChallengePage()
	if err != nil {
		return kiwiFlare{}, err
	}
	root, err := getRootNode(page)
	if err != nil {
		return kiwiFlare{}, err
	}

	kf := kiwiFlare{
		&kfParams{},
		make(chan kfSolution),
		make([]kfWorker, 1),
	}
	err = kf.getParams(root)
	if err != nil {
		return kiwiFlare{}, err
	}

	return kf, nil
}

func (kf kiwiFlare) getParams(root *html.Node) error {
	if root == nil {
		return errors.New("Challenge page reference is nil.")
	}

	parseAttr := func(av string) (uint32, error) {
		tmp, err := strconv.Atoi(av)
		if err != nil {
			return 0, err
		}
		return uint32(tmp), nil
	}

	attrs := root.Attr
	for _, a := range attrs {
		switch a.Key {
		case "data-sssg-challenge":
			kf.challenge = a.Val
		case "data-sssg-difficulty":
			diff, err := parseAttr(a.Val)
			if err != nil {
				return err
			}
			kf.diff = diff
		case "data-sssg-patience":
			patience, err := parseAttr(a.Val)
			if err != nil {
				return err
			}
			kf.patience = patience
		}
	}

	return nil
}

func (kf *kiwiFlare) generate(ctx context.Context) error {
	if len(kf.workers) == 0 {
		errors.New("No initialized thread workers found.")
	}

	// Brute force nonces until hashes channel receives valid answer.
	runWorker := func(w *kfWorker) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			w.hasher.Write([]byte(fmt.Sprintf("%s%d", kf.challenge, w.nonce)))
			kf.hashes <- kfSolution{
				hash:  w.hasher.Sum(nil),
				nonce: w.nonce,
			}
			// Reset input for next iteration.
			w.hasher.Reset()
			w.nonce++
		}
	}

	for i := range kf.workers {
		go runWorker(&kf.workers[i])
	}

	return nil
}

func (kf kiwiFlare) solve() (Solution, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dm", kf.patience))
	if err != nil {
		// Fallback to common value.
		duration = time.Minute * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	kf.workers = make([]kfWorker, runtime.NumCPU())
	for i := range kf.workers {
		kf.workers[i] = kfWorker{sha256.New(), rand.Uint32()}
	}
	go kf.generate(ctx)

	// Initializes to all 0s. Used below for prefix checking.
	// Given difficulty is measured in number of leading 0 bits. Here, we want # of bytes.
	zeros := make([]byte, kf.diff/8)
	// TODO: Compare at bit level for technical correctness.

	// Loop until answer has been found.
	for {
		select {
		// Return nil result if timeout has expired.
		case <-ctx.Done():
			return kfSolution{}, errors.New("KiwiFlare challenge timed out.")
		case h := <-kf.hashes:
			if bytes.HasPrefix(h.hash, zeros) {
				h.challenge = []byte(kf.challenge)
				cancel()
				return h, nil
			}
		}
	}
}
