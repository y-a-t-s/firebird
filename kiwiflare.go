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
	"time"
)

type Challenge struct {
	Salt     string // Challenge salt from server.
	Diff     uint32 // Difficulty level.
	Patience uint32 // Time limit for answer in minutes.
}

type Solution struct {
	Salt  string
	Hash  []byte
	Nonce uint32
}

type worker struct {
	hasher hash.Hash // Sha256 hasher obj.
	nonce  uint32    // Brute-forced nonce appended to salt.
}

func Solve(c Challenge) (Solution, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dm", c.Patience))
	if err != nil {
		// Fallback to common value.
		duration = time.Minute * 3
	}

	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	threads := runtime.NumCPU()

	hashes := make(chan Solution, threads/2)

	// Brute force nonces until a valid solution is found.
	runWorker := func(w *worker) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			w.hasher.Write([]byte(fmt.Sprintf("%s%d", c.Salt, w.nonce)))
			hashes <- Solution{
				Hash:  w.hasher.Sum(nil),
				Nonce: w.nonce,
			}
			// Reset input for next iteration.
			w.hasher.Reset()
			w.nonce++
		}
	}

	for i := 0; i < threads; i++ {
		go runWorker(&worker{sha256.New(), rand.Uint32()})
	}

	// Initializes to all 0s. Used below for prefix checking.
	// Given difficulty is measured in number of leading 0 bits. Here, we want # of bytes.
	zeros := make([]byte, c.Diff/8)
	// TODO: Compare at bit level for technical correctness.

	// Loop until answer has been found.
	for {
		select {
		// Return nil result if timeout has expired.
		case <-ctx.Done():
			return Solution{}, errors.New("KiwiFlare challenge timed out.")
		case h := <-hashes:
			if bytes.HasPrefix(h.Hash, zeros) {
				h.Salt = c.Salt
				return h, nil
			}
		}
	}
}
