package firebird

import (
	"context"
	"crypto/sha256"
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

// Given difficulty is measured in number of leading 0 bits.
func checkZeros(diff uint32, hash []byte) bool {
	rem := diff % 8
	nbytes := (diff - rem) / 8

	switch lh := uint32(len(hash)); {
	case lh < nbytes, rem > 0 && lh < nbytes+1:
		return false
	}

	for i := uint32(0); i < nbytes; i++ {
		b := hash[i]
		if b != 0x0 {
			return false
		}
	}
	if rem == 0 {
		return true
	}

	// Construct mask to check remaining bits.
	mask := uint8(0)
	for i := uint32(0); i < rem; i++ {
		mask <<= 1
		mask += 1
	}
	// Shift 1s we just added to the LHS of the octet.
	mask <<= 8 - rem

	if hash[nbytes]&mask == 0x0 {
		return true
	}

	return false
}

// Solve Challenge c. Returns Solution that can be submitted.
func Solve(ctx context.Context, c Challenge) (Solution, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dm", c.Patience))
	if err != nil {
		// Fallback to common value.
		duration = time.Minute * 3
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	threads := runtime.NumCPU()

	hashes := make(chan Solution, threads)

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

	// Loop until answer has been found.
	for {
		select {
		case <-ctx.Done():
			return Solution{}, ctx.Err()
		case h := <-hashes:
			if checkZeros(c.Diff, h.Hash) {
				h.Salt = c.Salt
				return h, nil
			}
		}
	}
}
