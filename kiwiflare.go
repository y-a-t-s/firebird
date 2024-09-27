package firebird

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"runtime"
	"time"

	"github.com/minio/sha256-simd"
)

type Challenge struct {
	Salt     string // Challenge salt from server.
	Diff     uint32 // Difficulty level.
	Patience uint32 // Time limit for answer in minutes.
	host     *url.URL
}

type Solution struct {
	Salt  string
	Hash  []byte
	Nonce uint32
	host  *url.URL
}

// Given difficulty is measured in number of leading 0 bits.
func checkZeros(diff uint32, hash []byte) bool {
	var (
		rem    = diff % 8
		nbytes = (diff - rem) / 8

		i    uint32 // Loop counter. Defined up here to save space.
		mask uint8  // Mask to check remaining bits.
	)

	if lh := uint32(len(hash)); lh < nbytes || (rem > 0 && lh < nbytes+1) {
		return false
	}

	for i = 0; i < nbytes; i++ {
		if b := hash[i]; b != 0x0 {
			return false
		}
	}
	if rem == 0 {
		return true
	}

	for i = 0; i < rem; i++ {
		mask <<= 1
		mask += 1
	}
	// Shift 1s we just added to the LHS of the octet.
	mask <<= 8 - rem

	return hash[nbytes]&mask == 0x0
}

// Brute force nonces until a valid solution is found.
func genHashes(ctx context.Context, c Challenge) <-chan Solution {
	var (
		out   = make(chan Solution, 1)
		sha   = sha256.New()
		nonce = rand.Uint32()
	)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			sha.Write([]byte(fmt.Sprintf("%s%d", c.Salt, nonce)))

			out <- Solution{
				Hash:  sha.Sum(nil),
				Nonce: nonce,
			}

			// Reset hasher input for next iteration.
			sha.Reset()
			nonce++
		}
	}()

	return out
}

// Solve Challenge c. Returns Solution that can be submitted.
func Solve(ctx context.Context, c Challenge) (Solution, error) {
	duration, err := time.ParseDuration(fmt.Sprintf("%dm", c.Patience))
	if err != nil {
		// Fallback to common value.
		duration = 3 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	sol := make(chan Solution, 1)

	go func() {
		threads := runtime.NumCPU()
		for i := 0; i < threads; i++ {
			go func() {
				hf := genHashes(ctx, c)
				// Loop until answer has been found.
				for h := range hf {
					if checkZeros(c.Diff, h.Hash) {
						h.Salt = c.Salt
						// Set host url to submit this to.
						h.host = c.host
						sol <- h
					}
				}
			}()
		}
	}()

	select {
	case <-ctx.Done():
		return Solution{}, ctx.Err()
	case s := <-sol:
		return s, nil
	}
}
