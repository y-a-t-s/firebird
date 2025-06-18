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
		rem    = diff % 8         // Remainder after dividing diff (given in bits) to bytes.
		nbytes = (diff - rem) / 8 // Amount of 0x0 bytes we can divide difficulty bits into.

		mask uint8 // Mask to check remaining bits.
	)

	lh := uint32(len(hash))
	// Check bounds for the loops found below.
	if lh < nbytes || (rem > 0 && lh < nbytes+1) {
		return false
	}

	// First, we count the number of leading 0x0 bytes.
	for i := range nbytes {
		if b := hash[i]; b != 0x0 {
			return false
		}
	}
	// If we don't have any more bits to check, return.
	if rem == 0 {
		return true
	}

	// Create bitmask by setting a bit to 1 for each remaining bit to check.
	// The mask is built from right-to-left and then shifted to the LHS of the octet.
	for range rem {
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

			sha.Write(fmt.Append(nil, c.Salt, nonce))

			sol := Solution{
				Hash:  sha.Sum(nil),
				Nonce: nonce,
			}
			// Ensure we don't hang if out channel is full on ctx close.
			select {
			case <-ctx.Done():
				return
			case out <- sol:
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
		// A reasonable hardware-based job limiter.
		// Use 1 worker per thread at most.
		threads := runtime.NumCPU()
		for i := 0; i < threads; i++ {
			go func() {
				hf := genHashes(ctx, c)
				// Loop until answer has been found.
				// Should break when hash worker terminates.
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
